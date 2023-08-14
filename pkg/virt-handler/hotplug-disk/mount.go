package hotplug_volume

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"github.com/opencontainers/runc/libcontainer/configs"

	"github.com/opencontainers/runc/libcontainer/devices"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

const (
	unableFindHotplugMountedDir            = "unable to find hotplug mounted directories for vmi without uid"
	failedToCreateCgroupManagerErrTemplate = "could not create cgroup manager. err: %v"
)

var (
	nodeIsolationResult = func() isolation.IsolationResult {
		return isolation.NodeIsolationResult()
	}
	deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
		return safepath.JoinAndResolveWithRelativeRoot("/proc/1/root", fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks", string(podUID)))
	}

	socketPath = func(podUID types.UID) string {
		return fmt.Sprintf("pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks/hp.sock", string(podUID))
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
			entries, err := os.ReadDir(safePath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				if info.Mode()&os.ModeDevice == 0 {
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

	isMounted = func(path *safepath.Path) (bool, error) {
		return isolation.IsMounted(path)
	}

	isBlockDevice = func(path *safepath.Path) (bool, error) {
		return isolation.IsBlockDevice(path)
	}

	isolationDetector = func(path string) isolation.PodIsolationDetector {
		return isolation.NewSocketBasedIsolationDetector(path)
	}

	getCgroupManager = func(vmi *v1.VirtualMachineInstance) (cgroup.Manager, error) {
		return cgroup.NewManagerFromVM(vmi)
	}

	parentPathForMount = func(
		parent isolation.IsolationResult,
		child isolation.IsolationResult,
		findmntInfo FindmntInfo,
	) (*safepath.Path, error) {
		return isolation.ParentPathForMount(parent, child, findmntInfo.Source, findmntInfo.Target)
	}
)

type volumeMounter struct {
	mountStateDir      string
	mountRecords       map[types.UID]*vmiMountTargetRecord
	mountRecordsLock   sync.Mutex
	skipSafetyCheck    bool
	hotplugDiskManager hotplugdisk.HotplugDiskManagerInterface
	ownershipManager   diskutils.OwnershipManagerInterface
}

// VolumeMounter is the interface used to mount and unmount volumes to/from a running virtlauncher pod.
type VolumeMounter interface {
	// Mount any new volumes defined in the VMI
	Mount(vmi *v1.VirtualMachineInstance) error
	MountFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) error
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
func NewVolumeMounter(mountStateDir string, kubeletPodsDir string) VolumeMounter {
	return &volumeMounter{
		mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
		mountStateDir:      mountStateDir,
		hotplugDiskManager: hotplugdisk.NewHotplugDiskManager(kubeletPodsDir),
		ownershipManager:   diskutils.DefaultOwnershipManager,
	}
}

func (m *volumeMounter) deleteMountTargetRecord(vmi *v1.VirtualMachineInstance) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf(unableFindHotplugMountedDir)
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
		return nil, fmt.Errorf(unableFindHotplugMountedDir)
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
		bytes, err := os.ReadFile(recordFile)
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
		return fmt.Errorf(unableFindHotplugMountedDir)
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

	err = os.WriteFile(recordFile, bytes, 0600)
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

func (m *volumeMounter) mountHotplugVolume(vmi *v1.VirtualMachineInstance, volumeName string, sourceUID types.UID, record *vmiMountTargetRecord, mountDirectory bool) error {
	logger := log.DefaultLogger()
	logger.V(4).Infof("Hotplug check volume name: %s", volumeName)
	if sourceUID != types.UID("") {
		if m.isBlockVolume(&vmi.Status, volumeName) {
			logger.V(4).Infof("Mounting block volume: %s", volumeName)
			if err := m.mountBlockHotplugVolume(vmi, volumeName, sourceUID, record); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to mount block hotplug volume %s: %v", volumeName, err)
				}
			}
		} else {
			logger.V(4).Infof("Mounting file system volume: %s", volumeName)
			if err := m.mountFileSystemHotplugVolume(vmi, volumeName, sourceUID, record, mountDirectory); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to mount filesystem hotplug volume %s: %v", volumeName, err)
				}
			}
		}
	}
	return nil
}

func (m *volumeMounter) Mount(vmi *v1.VirtualMachineInstance) error {
	return m.mountFromPod(vmi, types.UID(""))
}

func (m *volumeMounter) MountFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) error {
	return m.mountFromPod(vmi, sourceUID)
}

func (m *volumeMounter) mountFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume == nil {
			// Skip non hotplug volumes
			continue
		}
		mountDirectory := false
		if volumeStatus.MemoryDumpVolume != nil {
			mountDirectory = true
		}
		if sourceUID == types.UID("") {
			sourceUID = volumeStatus.HotplugVolume.AttachPodUID
		}
		if err := m.mountHotplugVolume(vmi, volumeStatus.Name, sourceUID, record, mountDirectory); err != nil {
			return err
		}
	}
	return nil
}

func (m *volumeMounter) isDirectoryMounted(vmiStatus *v1.VirtualMachineInstanceStatus, volumeName string) bool {
	for _, status := range vmiStatus.VolumeStatus {
		if status.Name == volumeName {
			return status.MemoryDumpVolume != nil
		}
	}
	return false
}

// isBlockVolume checks if the volumeDevices directory exists in the pod path, we assume there is a single volume associated with
// each pod, we use this knowledge to determine if we have a block volume or not.
func (m *volumeMounter) isBlockVolume(vmiStatus *v1.VirtualMachineInstanceStatus, volumeName string) bool {
	// Check if the volumeDevices directory exists in the attachment pod, if so, its a block device, otherwise its file system.
	for _, status := range vmiStatus.VolumeStatus {
		if status.Name == volumeName {
			return status.PersistentVolumeClaimInfo != nil && storagetypes.IsPVCBlock(status.PersistentVolumeClaimInfo.VolumeMode)
		}
	}
	return false
}

func (m *volumeMounter) mountBlockHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetPath, err := m.hotplugDiskManager.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		return err
	}

	cgroupsManager, err := getCgroupManager(vmi)
	if err != nil {
		return fmt.Errorf(failedToCreateCgroupManagerErrTemplate, err)
	}

	if _, err := safepath.JoinNoFollow(targetPath, volume); errors.Is(err, os.ErrNotExist) {
		dev, permissions, err := m.getSourceMajorMinor(sourceUID, volume)
		if err != nil {
			return err
		}

		if err := m.writePathToMountRecord(filepath.Join(unsafepath.UnsafeAbsolute(targetPath.Raw()), volume), vmi, record); err != nil {
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

	isMigrationInProgress := vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.Completed
	volumeNotReady := !m.volumeStatusReady(volume, vmi)

	if isMigrationInProgress || volumeNotReady {
		dev, _, err := m.getSourceMajorMinor(sourceUID, volume)
		if err != nil {
			return err
		}
		// allow block devices
		if err := m.allowBlockMajorMinor(dev, cgroupsManager); err != nil {
			return err
		}
	}

	return m.ownershipManager.SetFileOwnership(devicePath)
}

func (m *volumeMounter) volumeStatusReady(volumeName string, vmi *v1.VirtualMachineInstance) bool {
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName && volumeStatus.HotplugVolume != nil {
			if volumeStatus.Phase != v1.VolumeReady {
				log.DefaultLogger().V(4).Infof("Volume %s is not ready, but the target block device exists", volumeName)
			}
			return volumeStatus.Phase == v1.VolumeReady
		}
	}
	// This should never happen, it should always find the volume status in the VMI.
	return true
}

func (m *volumeMounter) getSourceMajorMinor(sourceUID types.UID, volumeName string) (uint64, os.FileMode, error) {
	basePath, err := deviceBasePath(sourceUID)
	if err != nil {
		return 0, 0, err
	}
	devicePath, err := basePath.AppendAndResolveWithRelativeRoot(volumeName)
	if err != nil {
		return 0, 0, err
	}
	return m.getBlockFileMajorMinor(devicePath, statSourceDevice)
}

func (m *volumeMounter) getBlockFileMajorMinor(devicePath *safepath.Path, getter func(fileName *safepath.Path) (os.FileInfo, error)) (uint64, os.FileMode, error) {
	fileInfo, err := getter(devicePath)
	if err != nil {
		return 0, 0, err
	}
	info := fileInfo.Sys().(*syscall.Stat_t)
	return info.Rdev, fileInfo.Mode(), nil
}

func (m *volumeMounter) removeBlockMajorMinor(dev uint64, manager cgroup.Manager) error {
	return m.updateBlockMajorMinor(dev, false, manager)
}

func (m *volumeMounter) allowBlockMajorMinor(dev uint64, manager cgroup.Manager) error {
	return m.updateBlockMajorMinor(dev, true, manager)
}

func (m *volumeMounter) updateBlockMajorMinor(dev uint64, allow bool, manager cgroup.Manager) error {

	var err error
	deviceRule := &devices.Rule{
		Type:        devices.BlockDevice,
		Major:       int64(unix.Major(dev)),
		Minor:       int64(unix.Minor(dev)),
		Permissions: "rwm",
		Allow:       allow,
	}

	err = manager.Set(&configs.Resources{
		Devices: []*devices.Rule{deviceRule},
	})

	if err != nil {
		log.Log.Infof("cgroup %s had failed to set device rule. error: %v. rule: %+v", manager.GetCgroupVersion(), err, *deviceRule)
	} else {
		log.Log.Infof("cgroup %s device rule is set successfully. rule: %+v", manager.GetCgroupVersion(), *deviceRule)
	}

	return err
}

func (m *volumeMounter) createBlockDeviceFile(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
	if _, err := safepath.JoinNoFollow(basePath, deviceName); errors.Is(err, os.ErrNotExist) {
		return mknodCommand(basePath, deviceName, dev, blockDevicePermissions)
	} else {
		return err
	}
}

func (m *volumeMounter) mountFileSystemHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord, mountDirectory bool) error {
	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	var target *safepath.Path
	var err error
	if mountDirectory {
		target, err = m.hotplugDiskManager.GetFileSystemDirectoryTargetPathFromHostView(virtlauncherUID, volume, true)
	} else {
		target, err = m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume, true)
	}
	if err != nil {
		return err
	}

	isMounted, err := isMounted(target)
	if err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", target, err)
	}
	if !isMounted {
		sourcePath, err := m.getSourcePodFilePath(sourceUID, vmi, volume)
		if err != nil {
			log.DefaultLogger().V(3).Infof("Error getting source path: %v", err)
			// We are eating the error to avoid spamming the log with errors, it might take a while for the volume
			// to get mounted on the node, and this will error until the volume is mounted.
			return nil
		}
		if err := m.writePathToMountRecord(unsafepath.UnsafeAbsolute(target.Raw()), vmi, record); err != nil {
			return err
		}
		if !mountDirectory {
			sourcePath, err = sourcePath.AppendAndResolveWithRelativeRoot("disk.img")
			if err != nil {
				return err
			}
		}
		if out, err := mountCommand(sourcePath, target); err != nil {
			return fmt.Errorf("failed to bindmount hotplug volume source from %v to %v: %v : %v", sourcePath, target, string(out), err)
		}
		log.DefaultLogger().V(1).Infof("successfully mounted %v", volume)
	}

	return m.ownershipManager.SetFileOwnership(target)
}

func (m *volumeMounter) findVirtlauncherUID(vmi *v1.VirtualMachineInstance) (uid types.UID) {
	cnt := 0
	for podUID := range vmi.Status.ActivePods {
		_, err := m.hotplugDiskManager.GetHotplugTargetPodPathOnHost(podUID)
		if err == nil {
			uid = podUID
			cnt++
		}
	}
	if cnt == 1 {
		return
	}
	// Either no pods, or multiple pods, skip.
	return types.UID("")
}

func (m *volumeMounter) getSourcePodFilePath(sourceUID types.UID, vmi *v1.VirtualMachineInstance, volume string) (*safepath.Path, error) {
	iso := isolationDetector("/path")
	isoRes, err := iso.DetectForSocket(vmi, socketPath(sourceUID))
	if err != nil {
		return nil, err
	}
	findmounts, err := LookupFindmntInfoByVolume(volume, isoRes.Pid())
	if err != nil {
		return nil, err
	}
	nodeIsoRes := nodeIsolationResult()
	mountRoot, err := nodeIsoRes.MountRoot()
	if err != nil {
		return nil, err
	}

	for _, findmnt := range findmounts {
		if filepath.Base(findmnt.Target) == volume {
			source := findmnt.GetSourcePath()
			path, err := parentPathForMount(nodeIsoRes, isoRes, findmnt)
			exists := !errors.Is(err, os.ErrNotExist)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}

			isBlock := false
			if exists {
				isBlock, _ = isBlockDevice(path)
			}

			if !exists || isBlock {
				// file not found, or block device, or directory check if we can find the mount.
				deviceFindMnt, err := LookupFindmntInfoByDevice(source)
				if err != nil {
					// Try the device found from the source
					deviceFindMnt, err = LookupFindmntInfoByDevice(findmnt.GetSourceDevice())
					if err != nil {
						return nil, err
					}
					// Check if the path was relative to the device.
					if !exists {
						return mountRoot.AppendAndResolveWithRelativeRoot(deviceFindMnt[0].Target, source)
					}
					return nil, err
				}
				return mountRoot.AppendAndResolveWithRelativeRoot(deviceFindMnt[0].Target)
			} else {
				return path, nil
			}
		}
	}
	// Did not find the disk image file, return error
	return nil, fmt.Errorf("unable to find source disk image path for pod %s", sourceUID)
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

		basePath, err := m.hotplugDiskManager.GetHotplugTargetPodPathOnHost(virtlauncherUID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
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
			var path *safepath.Path
			var err error
			if m.isBlockVolume(&vmi.Status, volumeStatus.Name) {
				path, err = safepath.JoinNoFollow(basePath, volumeStatus.Name)
				if errors.Is(err, os.ErrNotExist) {
					// already unmounted or never mounted
					continue
				}
			} else if m.isDirectoryMounted(&vmi.Status, volumeStatus.Name) {
				path, err = m.hotplugDiskManager.GetFileSystemDirectoryTargetPathFromHostView(virtlauncherUID, volumeStatus.Name, false)
				if errors.Is(err, os.ErrNotExist) {
					// already unmounted or never mounted
					continue
				}
			} else {
				path, err = m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volumeStatus.Name, false)
				if errors.Is(err, os.ErrNotExist) {
					// already unmounted or never mounted
					continue
				}
			}
			if err != nil {
				return err
			}
			currentHotplugPaths[unsafepath.UnsafeAbsolute(path.Raw())] = virtlauncherUID
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
	cgroupsManager, err := getCgroupManager(vmi)
	if err != nil {
		return fmt.Errorf(failedToCreateCgroupManagerErrTemplate, err)
	}

	// Get major and minor so we can deny the container.
	dev, _, err := m.getBlockFileMajorMinor(diskPath, statDevice)
	if err != nil {
		return err
	}
	// Delete block device file
	if err := safepath.UnlinkAtNoFollow(diskPath); err != nil {
		return err
	}
	if err := m.removeBlockMajorMinor(dev, cgroupsManager); err != nil {
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
				if errors.Is(err, os.ErrNotExist) {
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
					logger.Infof("Unable to unmount volume at path %s: %v", diskPath, err)
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
	targetPath, err := m.hotplugDiskManager.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if m.isBlockVolume(&vmi.Status, volume) {
		deviceName, err := safepath.JoinNoFollow(targetPath, volume)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return false, err
		}
		isBlockExists, _ := isBlockDevice(deviceName)
		return isBlockExists, nil
	}
	if m.isDirectoryMounted(&vmi.Status, volume) {
		path, err := safepath.JoinNoFollow(targetPath, volume)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return false, err
		}
		return isMounted(path)
	}
	path, err := safepath.JoinNoFollow(targetPath, fmt.Sprintf("%s.img", volume))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return isMounted(path)
}
