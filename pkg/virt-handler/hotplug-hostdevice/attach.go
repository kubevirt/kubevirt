package hotplug_hostdevice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

const (
	procRootTmpl   = "/proc/%d/root"
	devBusUsbTmpl  = "/dev/bus/usb/%03d/%03d"
	podsDevBusTmpl = "/pods/%s/volumes/kubernetes.io~empty-dir/dev-bus-usb/%03d/%03d"
)

var (
	// We use the same socket for container PID detection as for disk hotplug, because they are in the same container.
	socketPath = func(podUID types.UID) string {
		return fmt.Sprintf("/pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks/hp.sock", string(podUID))
	}

	sourceUsbPath = func(pid int, usbAddr v1.USBAddress) (*safepath.Path, error) {
		root := fmt.Sprintf(procRootTmpl, pid)
		usb := fmt.Sprintf(devBusUsbTmpl, usbAddr.Bus, usbAddr.DeviceNumber)
		return safepath.JoinAndResolveWithRelativeRoot(root, usb)
	}

	unsafeTargetUsbPath = func(podUID types.UID, usbAddr v1.USBAddress) string {
		return fmt.Sprintf(podsDevBusTmpl, string(podUID), usbAddr.Bus, usbAddr.DeviceNumber)
	}

	targetUsbPath = func(podUID types.UID, usbAddr v1.USBAddress) (*safepath.Path, error) {
		path := fmt.Sprintf(podsDevBusTmpl, string(podUID), usbAddr.Bus, usbAddr.DeviceNumber)
		return safepath.JoinAndResolveWithRelativeRoot("/", path)
	}

	statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
		info, err := safepath.StatAtNoFollow(fileName)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeCharDevice == 0 {
			return info, fmt.Errorf("%v is not a character device", fileName)
		}
		return info, nil
	}

	mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
		return safepath.MknodAtNoFollow(basePath, deviceName, blockDevicePermissions|syscall.S_IFCHR, dev)
	}

	isCharacterDevice = func(path *safepath.Path) (bool, error) {
		return isolation.IsCharacterDevice(path)
	}
)

type HostDeviceAttacher interface {
	Attach(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error
	AttachFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID, cgroupManager cgroup.Manager) error
	Detach(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error
	DetachAll(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error
	IsAttached(vmi *v1.VirtualMachineInstance, device string, sourceUID types.UID) (bool, error)
}

var _ HostDeviceAttacher = &hostDeviceAttacher{}

type hostDeviceAttacher struct {
	ownershipManager  diskutils.OwnershipManagerInterface
	isolationDetector isolation.PodIsolationDetector
	recordManager     *recordManager
}

func NewHostDeviceAttacher(stateDir string) HostDeviceAttacher {
	return newHostDeviceAttacher(stateDir)
}

func newHostDeviceAttacher(stateDir string) HostDeviceAttacher {
	return &hostDeviceAttacher{
		ownershipManager:  diskutils.DefaultOwnershipManager,
		isolationDetector: isolation.NewSocketBasedIsolationDetector(""),
		recordManager:     newRecordManager(stateDir),
	}
}

func (a hostDeviceAttacher) Attach(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error {
	return a.attachFromPod(vmi, "", cgroupManager)
}

func (a hostDeviceAttacher) AttachFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID, cgroupManager cgroup.Manager) error {
	return a.attachFromPod(vmi, sourceUID, cgroupManager)
}

func (a hostDeviceAttacher) Detach(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error {
	if vmi.UID == "" {
		return nil
	}

	logger := log.DefaultLogger()

	currRecord, exists, err := a.recordManager.Get(vmi.UID)
	if err != nil {
		return fmt.Errorf("failed to get record: %w", err)
	} else if !exists || len(currRecord.Entries) == 0 {
		// no entries to unmount
		return nil
	}

	currentHotplugPaths := make(map[string]struct{})
	detachCandidates := make(map[string]*safepath.Path)
	virtlauncherUID := a.findVirtlauncherUID(vmi)

	// Ideally, we would be looking at the actual host device in the domain but:
	// 1. The domain is built from the VMI spec
	// 2. The domain syncs before detach is called
	// 3. Detach will not get called if VMI sync fails
	// we should be good

	if vmi.Status.DeviceStatus != nil {

		specHostDevicesSet := make(map[string]struct{})
		for _, hostDevice := range vmi.Spec.Domain.Devices.HostDevices {
			specHostDevicesSet[hostDevice.Name] = struct{}{}
		}

		for _, hostDeviceStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
			if hostDeviceStatus.Hotplug == nil {
				// skip non hotpluggable host devices
				continue
			}

			if hostDeviceStatus.DeviceResourceClaimStatus == nil {
				logger.V(3).Infof("Device %s is not claimed, skipping", hostDeviceStatus.Name)
				continue
			}

			if hostDeviceStatus.DeviceResourceClaimStatus.Attributes == nil {
				logger.V(3).Infof("Device %s has no attributes, skipping", hostDeviceStatus.Name)
				continue
			}

			info := deviceInfo{
				Name: hostDeviceStatus.Name,
				Attr: *hostDeviceStatus.DeviceResourceClaimStatus.Attributes,
			}

			if !isUSB(info.Attr) {
				logger.V(4).Infof("Device %s is not a USB device, skipping (supported only usb devices)", info.Name)
				return nil
			}
			usbAddr := *info.Attr.USBAddress

			targetUSBPath, err := targetUsbPath(virtlauncherUID, usbAddr)
			if err != nil {
				if os.IsNotExist(err) {
					// already detached or never attached
					continue
				}
				return fmt.Errorf("failed to get target usb path: %w", err)
			}

			if _, ok := specHostDevicesSet[hostDeviceStatus.Name]; ok {
				logger.V(3).Infof("Device %s exists in spec, skipping for detaching", hostDeviceStatus.Name)
				currentHotplugPaths[unsafepath.UnsafeAbsolute(targetUSBPath.Raw())] = struct{}{}
			} else {
				logger.V(3).Infof("Device %s does not exist in spec, should be detaching", hostDeviceStatus.Name)
				detachCandidates[unsafepath.UnsafeAbsolute(targetUSBPath.Raw())] = targetUSBPath
			}
		}
	}

	newRecord := record{}

	for _, entry := range currRecord.Entries {
		fd, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", entry.TargetFile, err)
		}
		_ = fd.Close()
		targetPath := fd.Path()

		unsafeTargetPath := unsafepath.UnsafeAbsolute(targetPath.Raw())

		_, shouldBeAttached := currentHotplugPaths[unsafeTargetPath]
		if !shouldBeAttached {
			isCharacterDevice, err := isCharacterDevice(targetPath)
			if err != nil {
				return err
			}
			if isCharacterDevice {
				if err := a.detachHostDevice(targetPath, cgroupManager); err != nil {
					return fmt.Errorf("failed to detach host device %s: %w", targetPath, err)
				}
			}

			logger.V(3).Infof("Unplugged host device path %s", targetPath)
			delete(detachCandidates, unsafeTargetPath)
		} else {
			newRecord.Add(recordEntry{TargetFile: unsafeTargetPath})
		}
	}

	if len(newRecord.Entries) > 0 {
		err = a.recordManager.Store(vmi.UID, newRecord)
	} else {
		err = a.recordManager.Delete(vmi.UID)
	}
	if err != nil {
		return fmt.Errorf("failed to update attach record: %w", err)
	}

	// handle detach candidates which are not in the record
	for _, targetPath := range detachCandidates {
		isCharacterDevice, err := isCharacterDevice(targetPath)
		if err != nil {
			return err
		}
		if isCharacterDevice {
			if err := a.detachHostDevice(targetPath, cgroupManager); err != nil {
				return fmt.Errorf("failed to detach host device %s: %w", targetPath, err)
			}
		}

		logger.V(3).Infof("Unplugged host device path %s", targetPath)
	}

	return nil
}

func (a hostDeviceAttacher) detachHostDevice(hostDevicePath *safepath.Path, cgroupManager cgroup.Manager) error {
	dev, _, err := a.getSourceMajorMinor(hostDevicePath)
	if err != nil {
		return fmt.Errorf("detach failed to get device major/minor: %w", err)
	}

	err = a.removeCharacterDeviceMajorMinor(dev, cgroupManager)
	if err != nil {
		return fmt.Errorf("failed to remove character device access: %w", err)
	}

	err = safepath.UnlinkAtNoFollow(hostDevicePath)
	if err != nil {
		return fmt.Errorf("detach failed to delete device: %w", err)
	}

	return nil
}

func (a hostDeviceAttacher) DetachAll(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error {
	if vmi.UID == "" {
		return nil
	}
	logger := log.DefaultLogger().Object(vmi)
	logger.Info("Detaching all hotplugged host devices")

	currRecord, exists, err := a.recordManager.Get(vmi.UID)
	if err != nil {
		return fmt.Errorf("failed to get record: %w", err)
	} else if !exists || len(currRecord.Entries) == 0 {
		// no entries to unmount
		return nil
	}

	for _, entry := range currRecord.Entries {
		targetFile, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			if os.IsNotExist(err) {
				logger.V(3).Infof("Device %s is not attached, skipping", entry.TargetFile)
				continue
			}
			logger.Warningf("Unable to detach device %s: %v", entry.TargetFile, err)
			continue
		}
		_ = targetFile.Close()
		targetPath := targetFile.Path()

		if isCharacterDevice, err := isCharacterDevice(targetPath); err != nil {
			logger.Warningf("Unable to determine if device %s is a character device: %v", entry.TargetFile, err)
			continue
		} else if isCharacterDevice {
			if err := a.detachHostDevice(targetPath, cgroupManager); err != nil {
				logger.Warningf("Unable to detach host device %s: %v", entry.TargetFile, err)
				continue
			}
		}
	}

	return a.recordManager.Delete(vmi.UID)
}

func (a hostDeviceAttacher) IsAttached(vmi *v1.VirtualMachineInstance, device string, sourceUID types.UID) (bool, error) {
	virtlauncherUID := a.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return false, fmt.Errorf("Unable to determine virt-launcher UID")
	}

	if vmi.Status.DeviceStatus == nil {
		return false, nil
	}

	var hostDeviceStatus v1.DeviceStatusInfo

	for _, deviceStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		if deviceStatus.Name == device {
			hostDeviceStatus = deviceStatus
			break
		}
	}

	if hostDeviceStatus.Hotplug == nil {
		// skip non hotpluggable host devices
		return false, nil
	}

	if hostDeviceStatus.DeviceResourceClaimStatus == nil || hostDeviceStatus.DeviceResourceClaimStatus.Attributes == nil {
		return false, nil
	}

	info := deviceInfo{
		Name: hostDeviceStatus.Name,
		Attr: *hostDeviceStatus.DeviceResourceClaimStatus.Attributes,
	}

	// support only usb devices for now
	if !isUSB(info.Attr) {
		return false, nil
	}
	usbAddr := *info.Attr.USBAddress

	targetUSBPath, err := targetUsbPath(virtlauncherUID, usbAddr)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get target usb path: %w", err)
	}

	return isCharacterDevice(targetUSBPath)
}

func (a hostDeviceAttacher) attachFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID, cgroupManager cgroup.Manager) error {
	if vmi.Status.DeviceStatus == nil {
		return nil
	}

	specHostDeviceSet := make(map[string]struct{})
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		specHostDeviceSet[hd.Name] = struct{}{}
	}

	var rules []*devices.Rule
	var attachErr error

	for _, hostDeviceStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		if _, ok := specHostDeviceSet[hostDeviceStatus.Name]; !ok {
			continue
		}

		if hostDeviceStatus.Hotplug == nil {
			// Skip non hotplug hostdevices
			continue
		}

		if sourceUID == "" {
			sourceUID = hostDeviceStatus.Hotplug.AttachPodUID
		}

		if hostDeviceStatus.DeviceResourceClaimStatus == nil {
			log.DefaultLogger().V(3).Infof("Device %s is not claimed, skipping", hostDeviceStatus.Name)
			continue
		}

		if hostDeviceStatus.DeviceResourceClaimStatus.Attributes == nil {
			log.DefaultLogger().V(3).Infof("Device %s has no attributes, skipping", hostDeviceStatus.Name)
			continue
		}

		info := deviceInfo{
			Name: hostDeviceStatus.Name,
			Attr: *hostDeviceStatus.DeviceResourceClaimStatus.Attributes,
		}

		rule, err := a.attachHostDevice(vmi, info, sourceUID)
		if err != nil {
			attachErr = errors.Join(attachErr, fmt.Errorf("failed to attach hostdevice %s: %v", hostDeviceStatus.Name, err))
			continue
		}
		if rule != nil {
			rules = append(rules, rule)
		}
	}

	if err := a.applyDeviceRules(rules, cgroupManager); err != nil {
		return fmt.Errorf("failed to apply device rules: %w", err)
	}

	return attachErr
}

func (a hostDeviceAttacher) attachHostDevice(vmi *v1.VirtualMachineInstance, info deviceInfo, sourceUID types.UID) (*devices.Rule, error) {
	logger := log.DefaultLogger()
	logger.V(4).Infof("Hotplug check hostdevice name: %s", info.Name)
	if sourceUID == "" {
		return nil, nil
	}

	if !isUSB(info.Attr) {
		logger.V(4).Infof("Device %s is not a USB device, skipping (supported only usb devices)", info.Name)
		return nil, nil
	}
	usbAddr := *info.Attr.USBAddress
	logger.V(2).Infof("Attaching hostdevice %s to running VM", info.Name)

	virtlauncherUID := a.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil, nil
	}

	targetUSBPath, err := targetUsbPath(virtlauncherUID, usbAddr)
	if os.IsNotExist(err) {
		res, err := a.isolationDetector.DetectForSocket(vmi, socketPath(sourceUID))
		if err != nil {
			return nil, fmt.Errorf("failed to detect isolation for attachment pod: %w", err)
		}

		sourceUSBPath, err := sourceUsbPath(res.Pid(), *info.Attr.USBAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to get source usb path: %w", err)
		}

		dev, permission, err := a.getSourceMajorMinor(sourceUSBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get device major/minor: %w", err)
		}

		target := unsafeTargetUsbPath(virtlauncherUID, usbAddr)
		err = a.createCharacterDeviceFile(target, dev, permission)
		if err != nil {
			return nil, fmt.Errorf("failed to create character device file: %w", err)
		}

		logger.V(1).Infof("Successfully hot-plug hostdevice %s to VM", info.Name)

	} else if err != nil {
		return nil, fmt.Errorf("failed to get target usb path: %w", err)
	} else {
		logger.V(4).Info("HostDevice already attached to VM pod")
	}

	if targetUSBPath == nil {
		targetUSBPath, err = targetUsbPath(virtlauncherUID, usbAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to get target usb path: %w", err)
		}
	}

	if isCharacterExists, err := isCharacterDevice(targetUSBPath); err != nil {
		return nil, err
	} else if !isCharacterExists {
		return nil, fmt.Errorf("target device %v exists but it is not a character device", targetUSBPath)
	}

	// Add record to recordManager
	// If entry already exists, do nothing
	err = a.recordManager.StoreEntry(vmi.UID, unsafepath.UnsafeAbsolute(targetUSBPath.Raw()))
	if err != nil {
		return nil, fmt.Errorf("failed to store record: %w", err)
	}

	dev, _, err := a.getSourceMajorMinor(targetUSBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get device major/minor: %w", err)
	}

	err = a.ownershipManager.SetFileOwnership(targetUSBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to set file ownership: %w", err)
	}

	return buildDeviceRule(dev, true), nil
}

func (a hostDeviceAttacher) getSourceMajorMinor(usbPath *safepath.Path) (uint64, os.FileMode, error) {
	fileInfo, err := statDevice(usbPath)
	if err != nil {
		return 0, 0, err
	}
	info := fileInfo.Sys().(*syscall.Stat_t)
	return info.Rdev, fileInfo.Mode(), nil
}

func (a hostDeviceAttacher) createCharacterDeviceFile(usbPath string, dev uint64, blockDevicePermissions os.FileMode) error {
	dir := filepath.Dir(usbPath)

	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		} else {
			return err
		}
	}

	basePath, err := safepath.JoinAndResolveWithRelativeRoot("/", dir)
	if err != nil {
		return fmt.Errorf("failed to resolve base path %s: %w", dir, err)
	}
	deviceName := filepath.Base(usbPath)
	err = mknodCommand(basePath, deviceName, dev, blockDevicePermissions)
	if err != nil {
		return fmt.Errorf("failed to mknod device %s: %w", usbPath, err)
	}

	return nil
}

func (a hostDeviceAttacher) findVirtlauncherUID(vmi *v1.VirtualMachineInstance) (uid types.UID) {
	cnt := 0
	for podUID := range vmi.Status.ActivePods {
		// If an emptyDir is mounted, then the path exists and this pod is our virtlauncher pod
		path := filepath.Join("/pods", string(podUID), "/volumes/kubernetes.io~empty-dir/dev-bus-usb")
		_, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
		if err == nil {
			uid = podUID
			cnt++
		}
	}
	if cnt == 1 {
		return
	}
	// Either no pods, or multiple pods, skip.
	return ""
}

func (a hostDeviceAttacher) removeCharacterDeviceMajorMinor(dev uint64, cgroupManager cgroup.Manager) error {
	rule := buildDeviceRule(dev, false)
	return a.applyDeviceRules([]*devices.Rule{rule}, cgroupManager)
}

func buildDeviceRule(dev uint64, allow bool) *devices.Rule {
	return &devices.Rule{
		Type:        devices.CharDevice,
		Major:       int64(unix.Major(dev)),
		Minor:       int64(unix.Minor(dev)),
		Permissions: "rwm",
		Allow:       allow,
	}
}

func (a hostDeviceAttacher) applyDeviceRules(rules []*devices.Rule, cgroupManager cgroup.Manager) error {
	if len(rules) == 0 {
		return nil
	}

	if cgroupManager == nil {
		return fmt.Errorf("failed to apply device rules %+v: cgroup manager is nil", rules)
	}

	err := cgroupManager.Set(&configs.Resources{
		Devices: rules,
	})

	if err != nil {
		log.Log.Errorf("cgroup %s had failed to set device rules. error: %v. rule: %+v", cgroupManager.GetCgroupVersion(), err, rules)
	} else {
		log.Log.Infof("cgroup %s device rules is set successfully. rule: %+v", cgroupManager.GetCgroupVersion(), rules)
	}

	return err
}

type deviceInfo struct {
	Name string
	Attr v1.DeviceAttribute
}

func isUSB(attr v1.DeviceAttribute) bool {
	return attr.USBAddress != nil
}
