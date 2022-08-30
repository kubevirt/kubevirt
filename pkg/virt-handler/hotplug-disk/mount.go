package hotplug_volume

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"kubevirt.io/kubevirt/pkg/util"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/cgroups/fscommon"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/prometheus/procfs"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

var (
	nodeIsolationResult = func() isolation.IsolationResult {
		return isolation.NodeIsolationResult()
	}
	deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
		return safepath.JoinAndResolveWithRelativeRoot("/proc/1/root", "/var/lib/kubelet/pods/", string(podUID))
	}

	sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
		return safepath.JoinAndResolveWithRelativeRoot("/proc/1/root", fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes", string(podUID)))
	}

	socketPath = func(podUID types.UID) string {
		return fmt.Sprintf("pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks/hp.sock", string(podUID))
	}

	cgroupsBasePath = func() (*safepath.Path, error) {
		return safepath.JoinAndResolveWithRelativeRoot("/proc/1/root", cgroup.ControllerPath("devices"))
	}

	statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
		info, err := safepath.StatAtNoFollow(fileName)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeDevice == 0 {
			return info, fmt.Errorf("%v is not a block device", fileName)
		}
		return info, nil
	}

	statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
		// we don't know the device name, we only know that it is the only
		// device in a specific directory, let's look it up
		var devName string
		err := fileName.ExecuteNoFollow(func(safePath string) error {
			entries, err := ioutil.ReadDir(safePath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if entry.Mode()&os.ModeDevice == 0 {
					// not a device
					continue
				}
				devName = entry.Name()
				return nil
			}
			return fmt.Errorf("no device in %v", fileName)
		})
		if err != nil {
			return nil, err
		}
		devPath, err := safepath.JoinNoFollow(fileName, devName)
		if err != nil {
			return nil, err
		}
		return statDevice(devPath)
	}

	mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
		return safepath.MknodAtNoFollow(basePath, deviceName, blockDevicePermissions|syscall.S_IFBLK, dev)
	}

	mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
		return virt_chroot.MountChroot(sourcePath, targetPath, false).CombinedOutput()
	}

	unmountCommand = func(diskPath *safepath.Path) ([]byte, error) {
		return virt_chroot.UmountChroot(diskPath).CombinedOutput()
	}

	isMounted = func(mountPoint *safepath.Path) (bool, error) {
		return isolation.NodeIsolationResult().IsMounted(mountPoint)
	}

	isBlockDevice = func(path *safepath.Path) (bool, error) {
		return isolation.NodeIsolationResult().IsBlockDevice(path)
	}

	isolationDetector = func(path string) isolation.PodIsolationDetector {
		return isolation.NewSocketBasedIsolationDetector(path, cgroup.NewParser())
	}

	procMounts = func(pid int) ([]*procfs.MountInfo, error) {
		return procfs.GetProcMounts(pid)
	}
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type volumeMounter struct {
	podIsolationDetector isolation.PodIsolationDetector
	mountStateDir        string
	mountRecords         map[types.UID]*vmiMountTargetRecord
	mountRecordsLock     sync.Mutex
	skipSafetyCheck      bool
}

// VolumeMounter is the interface used to mount and unmount volumes to/from a running virtlauncher pod.
type VolumeMounter interface {
	// Mount any new volumes defined in the VMI
	Mount(vmi *v1.VirtualMachineInstance) error
	// Unmount any volumes no longer defined in the VMI
	Unmount(vmi *v1.VirtualMachineInstance) error
	//UnmountAll cleans up all hotplug volumes
	UnmountAll(vmi *v1.VirtualMachineInstance) error
	//IsMounted returns if the volume is mounted or not.
	IsMounted(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID) (bool, error)
}

type vmiMountTargetEntry struct {
	TargetFile string `json:"targetFile"`
}

type vmiMountTargetRecord struct {
	MountTargetEntries []vmiMountTargetEntry `json:"mountTargetEntries"`
	UsesSafePaths      bool                  `json:"usesSafePaths"`
}

// NewVolumeMounter creates a new VolumeMounter
func NewVolumeMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string) VolumeMounter {
	return &volumeMounter{
		podIsolationDetector: isoDetector,
		mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
		mountStateDir:        mountStateDir,
	}
}

func (m *volumeMounter) deleteMountTargetRecord(vmi *v1.VirtualMachineInstance) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find hotplug mounted directories for vmi without uid")
	}

	recordFile := filepath.Join(m.mountStateDir, string(vmi.UID))
	exists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return err
	}

	if exists {
		record, err := m.getMountTargetRecord(vmi)
		if err != nil {
			return err
		}

		for _, target := range record.MountTargetEntries {
			os.Remove(target.TargetFile)
		}

		os.Remove(recordFile)
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	delete(m.mountRecords, vmi.UID)

	return nil
}

func (m *volumeMounter) getMountTargetRecord(vmi *v1.VirtualMachineInstance) (*vmiMountTargetRecord, error) {
	var ok bool
	var existingRecord *vmiMountTargetRecord

	if string(vmi.UID) == "" {
		return nil, fmt.Errorf("unable to find hotplug mounted directories for vmi without uid")
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	existingRecord, ok = m.mountRecords[vmi.UID]

	// first check memory cache
	if ok {
		return existingRecord, nil
	}

	// if not there, see if record is on disk, this can happen if virt-handler restarts
	recordFile := filepath.Join(m.mountStateDir, string(vmi.UID))

	exists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return nil, err
	}

	if exists {
		record := vmiMountTargetRecord{}
		bytes, err := ioutil.ReadFile(recordFile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bytes, &record)
		if err != nil {
			return nil, err
		}

		// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
		// After a one-time convert and persist, old records are safe too.
		if !record.UsesSafePaths {
			for i, path := range record.MountTargetEntries {
				record.UsesSafePaths = true
				safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", path.TargetFile)
				if err != nil {
					return nil, fmt.Errorf("failed converting legacy path to safepath: %v", err)
				}
				record.MountTargetEntries[i].TargetFile = unsafepath.UnsafeAbsolute(safePath.Raw())
			}
		}

		m.mountRecords[vmi.UID] = &record
		return &record, nil
	}

	// not found
	return &vmiMountTargetRecord{UsesSafePaths: true}, nil
}

func (m *volumeMounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find hotplug mounted directories for vmi without uid")
	}

	// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
	// After a one-time convert and persist, old records are safe too.
	record.UsesSafePaths = true

	recordFile := filepath.Join(m.mountStateDir, string(vmi.UID))

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()

	bytes, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(recordFile), 0750)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(recordFile, bytes, 0600)
	if err != nil {
		return err
	}
	m.mountRecords[vmi.UID] = record
	return nil
}

func (m *volumeMounter) writePathToMountRecord(path string, vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	record.MountTargetEntries = append(record.MountTargetEntries, vmiMountTargetEntry{
		TargetFile: path,
	})
	if err := m.setMountTargetRecord(vmi, record); err != nil {
		return err
	}
	return nil
}

func (m *volumeMounter) Mount(vmi *v1.VirtualMachineInstance) error {
	logger := log.DefaultLogger()
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume == nil {
			// Skip non hotplug volumes
			continue
		}
		logger.V(4).Infof("Hotplug check volume name: %s", volumeStatus.Name)
		sourceUID := volumeStatus.HotplugVolume.AttachPodUID
		if sourceUID != types.UID("") {
			if m.isBlockVolume(sourceUID) {
				logger.V(4).Infof("Mounting block volume: %s", volumeStatus.Name)
				if err := m.mountBlockHotplugVolume(vmi, volumeStatus.Name, sourceUID, record); err != nil {
					return err
				}
			} else {
				logger.V(4).Infof("Mounting file system volume: %s", volumeStatus.Name)
				if err := m.mountFileSystemHotplugVolume(vmi, volumeStatus.Name, sourceUID, record); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// isBlockVolume checks if the volumeDevices directory exists in the pod path, we assume there is a single volume associated with
// each pod, we use this knowledge to determine if we have a block volume or not.
func (m *volumeMounter) isBlockVolume(sourceUID types.UID) bool {
	// Check if the volumeDevices directory exists in the attachment pod, if so, its a block device, otherwise its file system.
	if sourceUID != types.UID("") {
		devicePath, err := deviceBasePath(sourceUID)
		if err != nil {
			log.Log.V(4).Error(err.Error())
			return false
		}
		devicePath, err = devicePath.AppendAndResolveWithRelativeRoot("volumeDevices")
		if err != nil {
			log.Log.V(4).Infof("%s pod does not contain a block device %v", sourceUID, err)
			return false
		}
		info, err := safepath.StatAtNoFollow(devicePath)
		if err != nil {
			log.Log.V(4).Error(err.Error())
			return false
		}
		return info.IsDir()
	}
	return false
}

func (m *volumeMounter) mountBlockHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetPath, err := hotplugdisk.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		return err
	}

	if _, err := safepath.JoinNoFollow(targetPath, volume); os.IsNotExist(err) {
		computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		dev, permissions, err := m.getSourceMajorMinor(sourceUID)
		if err != nil {
			return err
		}

		if err := m.writePathToMountRecord(filepath.Join(unsafepath.UnsafeAbsolute(targetPath.Raw()), volume), vmi, record); err != nil {
			return err
		}

		if err := m.allowBlockMajorMinor(dev, computeCGroupPath); err != nil {
			return err
		}

		if err := m.createBlockDeviceFile(targetPath, volume, dev, permissions); err != nil && !os.IsExist(err) {
			return err
		}
		log.DefaultLogger().V(1).Infof("successfully created block device %v", volume)
	} else if err != nil {
		return err
	}

	devicePath, err := safepath.JoinNoFollow(targetPath, volume)
	if err != nil {
		return err
	}
	if isBlockExists, err := isBlockDevice(devicePath); err != nil {
		return err
	} else if !isBlockExists {
		return fmt.Errorf("target device %v exists but it is not a block device", devicePath)
	}

	volumeNotReady := !m.volumeStatusReady(volume, vmi)

	if volumeNotReady {
		computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		dev, _, err := m.getSourceMajorMinor(sourceUID)
		if err != nil {
			return err
		}
		// allow block devices
		if err := m.allowBlockMajorMinor(dev, computeCGroupPath); err != nil {
			return err
		}
	}

	return nil
}

func (m *volumeMounter) volumeStatusReady(volumeName string, vmi *v1.VirtualMachineInstance) bool {
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName && volumeStatus.HotplugVolume != nil {
			if volumeStatus.Phase != v1.VolumeReady {
				log.DefaultLogger().Infof("Volume %s is not ready, but the target block device exists", volumeName)
			}
			return volumeStatus.Phase == v1.VolumeReady
		}
	}
	// This should never happen, it should always find the volume status in the VMI.
	return true
}

func (m *volumeMounter) getSourceMajorMinor(sourceUID types.UID) (uint64, os.FileMode, error) {
	var result uint64
	var perms os.FileMode
	if sourceUID != types.UID("") {
		basepath, err := deviceBasePath(sourceUID)
		if err != nil {
			return 0, 0, err
		}
		basepath, err = basepath.AppendAndResolveWithRelativeRoot("volumes")
		if err != nil {
			return 0, 0, err
		}
		err = basepath.ExecuteNoFollow(func(safePath string) error {
			return filepath.Walk(safePath, func(filePath string, info os.FileInfo, err error) error {
				if info != nil && !info.IsDir() {
					// Walk doesn't follow symlinks which is good because I need to massage symlinks
					linkInfo, err := os.Lstat(filePath)
					if err != nil {
						return err
					}
					path := filePath
					if linkInfo.Mode()&os.ModeSymlink != 0 {
						// Its a symlink, follow it
						link, err := os.Readlink(filePath)
						if err != nil {
							return err
						}
						if !strings.HasPrefix(link, util.HostRootMount) {
							path = filepath.Join(util.HostRootMount, link)
						} else {
							path = link
						}
					}
					sPath, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
					if err != nil {
						return err
					}
					result, perms, err = m.getBlockFileMajorMinor(sPath, statDevice)
					// Err != nil means not a block device or unable to determine major/minor, try next file
					if err == nil {
						// Successfully located
						return io.EOF
					}
					return nil
				}
				return nil
			})
		})
		if err != nil && err != io.EOF {
			return 0, 0, err
		}
	}
	if perms == 0 {
		return 0, 0, fmt.Errorf("Unable to find block device")
	}
	return result, perms, nil
}

func (m *volumeMounter) getBlockFileMajorMinor(devicePath *safepath.Path, getter func(fileName *safepath.Path) (os.FileInfo, error)) (uint64, os.FileMode, error) {
	fileInfo, err := getter(devicePath)
	if err != nil {
		return 0, 0, err
	}
	info := fileInfo.Sys().(*syscall.Stat_t)
	return info.Rdev, fileInfo.Mode(), nil
}

// getTargetCgroupPath returns the container cgroup path of the compute container in the pod.
func (m *volumeMounter) getTargetCgroupPath(vmi *v1.VirtualMachineInstance) (*safepath.Path, error) {
	basePath, err := cgroupsBasePath()
	if err != nil {
		return nil, err
	}
	isoRes, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return nil, err
	}

	virtlauncherCgroupPath, err := safepath.JoinNoFollow(basePath, isoRes.Slice())
	if err != nil {
		return nil, fmt.Errorf("failed to determine custom image path %s: %v", virtlauncherCgroupPath, err)
	}
	fileInfo, err := safepath.StatAtNoFollow(virtlauncherCgroupPath)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("detected path %s, but it is not a directory", virtlauncherCgroupPath)
	}
	return virtlauncherCgroupPath, nil
}

func (m *volumeMounter) removeBlockMajorMinor(dev uint64, path *safepath.Path) error {
	return m.updateBlockMajorMinor(dev, path, false)
}

func (m *volumeMounter) allowBlockMajorMinor(dev uint64, path *safepath.Path) error {
	return m.updateBlockMajorMinor(dev, path, true)
}

func (m *volumeMounter) updateBlockMajorMinor(dev uint64, path *safepath.Path, allow bool) error {
	deviceRule := &configs.DeviceRule{
		Type:        configs.BlockDevice,
		Major:       int64(unix.Major(dev)),
		Minor:       int64(unix.Minor(dev)),
		Permissions: "rwm",
		Allow:       allow,
	}
	if err := m.updateDevicesList(path, deviceRule); err != nil {
		return err
	}
	return nil
}

func (m *volumeMounter) loadEmulator(path *safepath.Path) (*devices.Emulator, error) {
	list, err := fscommon.ReadFile(unsafepath.UnsafeAbsolute(path.Raw()), "devices.list")
	if err != nil {
		return nil, err
	}
	return devices.EmulatorFromList(bytes.NewBufferString(list))
}

func (m *volumeMounter) updateDevicesList(path *safepath.Path, rule *configs.DeviceRule) error {
	// Create the target emulator for comparison later.
	target, err := m.loadEmulator(path)
	if err != nil {
		return err
	}
	target.Apply(*rule)

	file := "devices.deny"
	if rule.Allow {
		file = "devices.allow"
	}
	if err := fscommon.WriteFile(unsafepath.UnsafeAbsolute(path.Raw()), file, rule.CgroupString()); err != nil {
		return err
	}

	// Final safety check -- ensure that the resulting state is what was
	// requested. This is only really correct for white-lists, but for
	// black-lists we can at least check that the cgroup is in the right mode.
	currentAfter, err := m.loadEmulator(path)
	if err != nil {
		return err
	}
	if !m.skipSafetyCheck {
		if !target.IsBlacklist() && !reflect.DeepEqual(currentAfter, target) {
			return errors.New("resulting devices cgroup doesn't precisely match target")
		} else if target.IsBlacklist() != currentAfter.IsBlacklist() {
			return errors.New("resulting devices cgroup doesn't match target mode")
		}
	}
	return nil
}

func (m *volumeMounter) createBlockDeviceFile(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
	if _, err := safepath.JoinNoFollow(basePath, deviceName); os.IsNotExist(err) {
		return mknodCommand(basePath, deviceName, dev, blockDevicePermissions)
	} else {
		return err
	}
}

func (m *volumeMounter) mountFileSystemHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	sourcePath, err := m.getSourcePodFilePath(sourceUID, vmi, volume)
	if err != nil {
		log.DefaultLogger().Infof("Error finding source path: %v", err)
		return nil
	}

	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetPath, err := hotplugdisk.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume, true)
	if err != nil {
		return err
	}

	if isMounted, err := isMounted(targetPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetPath, err)
	} else if !isMounted {
		if err := m.writePathToMountRecord(unsafepath.UnsafeAbsolute(targetPath.Raw()), vmi, record); err != nil {
			return err
		}
		if out, err := mountCommand(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to bindmount hotplug-disk %v to %v: %v : %v", sourcePath, targetPath, string(out), err)
		}
		log.DefaultLogger().V(1).Infof("successfully mounted %v", volume)
	} else {
		return nil
	}

	return nil
}

func (m *volumeMounter) findVirtlauncherUID(vmi *v1.VirtualMachineInstance) types.UID {
	if len(vmi.Status.ActivePods) == 1 {
		for k := range vmi.Status.ActivePods {
			return k
		}
	}
	// Either no pods, or multiple pods, skip.
	return types.UID("")
}

func (m *volumeMounter) getSourcePodFilePath(sourceUID types.UID, vmi *v1.VirtualMachineInstance, volume string) (*safepath.Path, error) {
	diskPath := ""
	if sourceUID != types.UID("") {
		basepath, err := sourcePodBasePath(sourceUID)
		if err != nil {
			return nil, err
		}
		err = filepath.Walk(unsafepath.UnsafeAbsolute(basepath.Raw()), func(filePath string, info os.FileInfo, err error) error {
			if path.Base(filePath) == "disk.img" {
				// Found disk image
				diskPath = path.Dir(filePath)
				return io.EOF
			}
			return nil
		})
		if err != nil && err != io.EOF {
			return nil, err
		}
	}
	if diskPath == "" {
		// Unfortunately I cannot use this approach for all storage, for instance in ceph the mount.Root is / which is obviously
		// not the path we want to mount. So we stick with try sourcePodBasePath first, then if not found try mountinfo.
		iso := isolationDetector("/path")
		isoRes, err := iso.DetectForSocket(vmi, socketPath(sourceUID))
		if err != nil {
			return nil, err
		}
		mounts, err := procMounts(isoRes.Pid())
		if err != nil {
			return nil, err
		}
		for _, mount := range mounts {
			if mount.MountPoint == "/pvc" {
				mountRoot, err := nodeIsolationResult().MountRoot()
				if err != nil {
					return nil, err
				}
				return mountRoot.AppendAndResolveWithRelativeRoot(mount.Root)
			}
		}
	}
	if diskPath == "" {
		// Did not find the disk image file, return error
		return nil, fmt.Errorf("Unable to find source disk image path for pod %s", sourceUID)
	}
	path, err := safepath.JoinAndResolveWithRelativeRoot("/", diskPath)
	if err != nil {
		return nil, err
	}
	return path, nil
}

// Unmount unmounts all hotplug disk that are no longer part of the VMI
func (m *volumeMounter) Unmount(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID != "" {
		record, err := m.getMountTargetRecord(vmi)
		if err != nil {
			return err
		} else if record == nil {
			// no entries to unmount
			return nil
		}
		if len(record.MountTargetEntries) == 0 {
			return nil
		}

		currentHotplugPaths := make(map[string]types.UID, 0)
		virtlauncherUID := m.findVirtlauncherUID(vmi)

		basePath, err := hotplugdisk.GetHotplugTargetPodPathOnHost(virtlauncherUID)
		if err != nil {
			if os.IsNotExist(err) {
				// no mounts left, the base path does not even exist anymore
				if err := m.deleteMountTargetRecord(vmi); err != nil {
					return fmt.Errorf("failed to delete mount target records: %v", err)
				}
				return nil
			}
			return err
		}
		for _, volumeStatus := range vmi.Status.VolumeStatus {
			if volumeStatus.HotplugVolume == nil {
				continue
			}
			if m.isBlockVolume(volumeStatus.HotplugVolume.AttachPodUID) {
				path, err := safepath.JoinNoFollow(basePath, volumeStatus.Name)
				if err != nil {
					return err
				}
				currentHotplugPaths[unsafepath.UnsafeAbsolute(path.Raw())] = virtlauncherUID
			} else {
				path, err := hotplugdisk.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volumeStatus.Name, false)
				if os.IsNotExist(err) {
					// already unmounted or never mounted
					continue
				}
				if err != nil {
					return err
				}
				currentHotplugPaths[unsafepath.UnsafeAbsolute(path.Raw())] = virtlauncherUID
			}
		}
		newRecord := vmiMountTargetRecord{
			MountTargetEntries: make([]vmiMountTargetEntry, 0),
		}
		for _, entry := range record.MountTargetEntries {
			fd, err := safepath.NewFileNoFollow(entry.TargetFile)
			if err != nil {
				return err
			}
			fd.Close()
			diskPath := fd.Path()

			if _, ok := currentHotplugPaths[unsafepath.UnsafeAbsolute(diskPath.Raw())]; !ok {
				if blockDevice, err := isBlockDevice(diskPath); err != nil {
					return err
				} else if blockDevice {
					if err := m.unmountBlockHotplugVolumes(diskPath, vmi); err != nil {
						return err
					}
				} else if err := m.unmountFileSystemHotplugVolumes(diskPath); err != nil {
					return err
				}
			} else {
				newRecord.MountTargetEntries = append(newRecord.MountTargetEntries, vmiMountTargetEntry{
					TargetFile: unsafepath.UnsafeAbsolute(diskPath.Raw()),
				})
			}
		}
		if len(newRecord.MountTargetEntries) > 0 {
			err = m.setMountTargetRecord(vmi, &newRecord)
		} else {
			err = m.deleteMountTargetRecord(vmi)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *volumeMounter) unmountFileSystemHotplugVolumes(diskPath *safepath.Path) error {
	if mounted, err := isMounted(diskPath); err != nil {
		return fmt.Errorf("failed to check mount point for hotplug disk %v: %v", diskPath, err)
	} else if mounted {
		out, err := unmountCommand(diskPath)
		if err != nil {
			return fmt.Errorf("failed to unmount hotplug disk %v: %v : %v", diskPath, string(out), err)
		}
		err = safepath.UnlinkAtNoFollow(diskPath)
		if err != nil {
			return fmt.Errorf("failed to remove hotplug disk directory %v: %v : %v", diskPath, string(out), err)
		}

	}
	return nil
}

func (m *volumeMounter) unmountBlockHotplugVolumes(diskPath *safepath.Path, vmi *v1.VirtualMachineInstance) error {
	// Get major and minor so we can deny the container.
	dev, _, err := m.getBlockFileMajorMinor(diskPath, statDevice)
	if err != nil {
		return err
	}
	// Delete block device file
	if err := safepath.UnlinkAtNoFollow(diskPath); err != nil {
		return err
	}
	path, err := m.getTargetCgroupPath(vmi)
	if err != nil {
		return err
	}
	if err := m.removeBlockMajorMinor(dev, path); err != nil {
		return err
	}
	return nil
}

// UnmountAll unmounts all hotplug disks of a given VMI.
func (m *volumeMounter) UnmountAll(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID != "" {
		logger := log.DefaultLogger().Object(vmi)
		logger.Info("Cleaning up remaining hotplug volumes")
		record, err := m.getMountTargetRecord(vmi)
		if err != nil {
			return err
		} else if record == nil {
			// no entries to unmount
			logger.Info("No hotplug volumes found to unmount")
			return nil
		}

		for _, entry := range record.MountTargetEntries {
			diskPath, err := safepath.NewFileNoFollow(entry.TargetFile)
			if err != nil {
				if os.IsNotExist(err) {
					logger.Infof("Device %v is not mounted anymore, continuing.", entry.TargetFile)
					continue
				}
				logger.Infof("Unable to unmount volume at path %s: %v", entry.TargetFile, err)
				continue
			}
			diskPath.Close()
			if isBlock, err := isBlockDevice(diskPath.Path()); err != nil {
				logger.Infof("Unable to remove block device at path %s: %v", diskPath, err)
			} else if isBlock {
				if err := m.unmountBlockHotplugVolumes(diskPath.Path(), vmi); err != nil {
					logger.Infof("Unable to remove block device at path %s: %v", diskPath, err)
					// Don't return error, try next.
				}
			} else {
				if err := m.unmountFileSystemHotplugVolumes(diskPath.Path()); err != nil {
					logger.Infof("Unable to unmount volume at path %s", diskPath)
					// Don't return error, try next.
				}
			}
		}
		err = m.deleteMountTargetRecord(vmi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *volumeMounter) IsMounted(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID) (bool, error) {
	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return false, fmt.Errorf("Unable to determine virt-launcher UID")
	}
	targetPath, err := hotplugdisk.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if m.isBlockVolume(sourceUID) {
		deviceName, err := safepath.JoinNoFollow(targetPath, volume)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		isBlockExists, _ := isBlockDevice(deviceName)
		return isBlockExists, nil
	}
	path, err := safepath.JoinNoFollow(targetPath, volume)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return isMounted(path)
}
