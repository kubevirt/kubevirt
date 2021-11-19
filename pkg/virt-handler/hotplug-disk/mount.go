package hotplug_volume

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/cgroups/fscommon"
	"github.com/opencontainers/runc/libcontainer/configs"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

var (
	deviceBasePath = func(podUID types.UID) string {
		return fmt.Sprintf("/proc/1/root/var/lib/kubelet/pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks", string(podUID))
	}

	sourcePodBasePath = func(podUID types.UID) string {
		return fmt.Sprintf("/proc/1/root/var/lib/kubelet/pods/%s/volumes", string(podUID))
	}

	socketPath = func(podUID types.UID) string {
		return fmt.Sprintf("pods/%s/volumes/kubernetes.io~empty-dir/hotplug-disks/hp.sock", string(podUID))
	}

	cgroupsBasePath = func() string {
		return filepath.Join("/proc/1/root", cgroup.ControllerPath("devices"))
	}

	statCommand = func(fileName string) ([]byte, error) {
		return exec.Command("/usr/bin/stat", fileName, "-L", "-c%t,%T,%a,%F").CombinedOutput()
	}

	mknodCommand = func(deviceName string, major, minor int64, blockDevicePermissions string) ([]byte, error) {
		return exec.Command("/usr/bin/mknod", "--mode", fmt.Sprintf("0%s", blockDevicePermissions), deviceName, "b", strconv.FormatInt(major, 10), strconv.FormatInt(minor, 10)).CombinedOutput()
	}

	mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
		return virt_chroot.MountChroot(strings.TrimPrefix(sourcePath, isolation.NodeIsolationResult().MountRoot()), targetPath, false).CombinedOutput()
	}

	unmountCommand = func(diskPath string) ([]byte, error) {
		return virt_chroot.UmountChroot(diskPath).CombinedOutput()
	}

	isMounted = func(path string) (bool, error) {
		return isolation.NodeIsolationResult().IsMounted(path)
	}

	isBlockDevice = func(path string) (bool, error) {
		return isolation.NodeIsolationResult().IsBlockDevice(path)
	}

	isolationDetector = func(path string) isolation.PodIsolationDetector {
		return isolation.NewSocketBasedIsolationDetector(path, cgroup.NewParser())
	}
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type volumeMounter struct {
	podIsolationDetector isolation.PodIsolationDetector
	mountStateDir        string
	mountRecords         map[types.UID]*vmiMountTargetRecord
	mountRecordsLock     sync.Mutex
	skipSafetyCheck      bool
	hotplugDiskManager   hotplugdisk.HotplugDiskManagerInterface
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
}

// NewVolumeMounter creates a new VolumeMounter
func NewVolumeMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string) VolumeMounter {
	return &volumeMounter{
		podIsolationDetector: isoDetector,
		mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
		mountStateDir:        mountStateDir,
		hotplugDiskManager:   hotplugdisk.NewHotplugDiskManager(),
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

		m.mountRecords[vmi.UID] = &record
		return &record, nil
	}

	// not found
	return &vmiMountTargetRecord{}, nil
}

func (m *volumeMounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find hotplug mounted directories for vmi without uid")
	}

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
	if path != "" {
		record.MountTargetEntries = append(record.MountTargetEntries, vmiMountTargetEntry{
			TargetFile: path,
		})
	}
	if err := m.setMountTargetRecord(vmi, record); err != nil {
		return err
	}
	return nil
}

func (m *volumeMounter) mountHotplugVolume(vmi *v1.VirtualMachineInstance, volumeName string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	logger := log.DefaultLogger()
	logger.V(4).Infof("Hotplug check volume name: %s", volumeName)
	if sourceUID != types.UID("") {
		if m.isBlockVolume(&vmi.Status, volumeName) {
			logger.V(4).Infof("Mounting block volume: %s", volumeName)
			if err := m.mountBlockHotplugVolume(vmi, volumeName, sourceUID, record); err != nil {
				return err
			}
		} else {
			logger.V(4).Infof("Mounting file system volume: %s", volumeName)
			if err := m.mountFileSystemHotplugVolume(vmi, volumeName, sourceUID, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *volumeMounter) Mount(vmi *v1.VirtualMachineInstance) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume == nil {
			// Skip non hotplug volumes
			continue
		}
		sourceUID := volumeStatus.HotplugVolume.AttachPodUID
		if err := m.mountHotplugVolume(vmi, volumeStatus.Name, sourceUID, record); err != nil {
			return err
		}
	}
	return nil
}

func (m *volumeMounter) MountFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume == nil {
			// Skip non hotplug volumes
			continue
		}
		if err := m.mountHotplugVolume(vmi, volumeStatus.Name, sourceUID, record); err != nil {
			return err
		}
	}
	return nil
}

// isBlockVolume checks if the volumeDevices directory exists in the pod path, we assume there is a single volume associated with
// each pod, we use this knowledge to determine if we have a block volume or not.
func (m *volumeMounter) isBlockVolume(vmiStatus *v1.VirtualMachineInstanceStatus, volumeName string) bool {
	// Check if the volumeDevices directory exists in the attachment pod, if so, its a block device, otherwise its file system.
	for _, status := range vmiStatus.VolumeStatus {
		if status.Name == volumeName {
			return status.PersistentVolumeClaimInfo != nil && status.PersistentVolumeClaimInfo.VolumeMode != nil && *status.PersistentVolumeClaimInfo.VolumeMode == k8sv1.PersistentVolumeBlock
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

	deviceName := filepath.Join(targetPath, volume)

	isMigrationInProgress := vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.Completed

	if isBlockExists, _ := isBlockDevice(deviceName); !isBlockExists {
		computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		sourceMajor, sourceMinor, permissions, err := m.getSourceMajorMinor(sourceUID, volume)
		if err != nil {
			return err
		}
		if err := m.writePathToMountRecord(deviceName, vmi, record); err != nil {
			return err
		}
		// allow block devices
		if err := m.allowBlockMajorMinor(sourceMajor, sourceMinor, computeCGroupPath); err != nil {
			return err
		}
		if _, err = m.createBlockDeviceFile(deviceName, sourceMajor, sourceMinor, permissions); err != nil {
			return err
		}
	} else if isBlockExists && (!m.volumeStatusReady(volume, vmi) || isMigrationInProgress) {
		// Block device exists already, but the volume is not ready yet, ensure that the device is allowed.
		computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		sourceMajor, sourceMinor, _, err := m.getSourceMajorMinor(sourceUID, volume)
		if err != nil {
			return err
		}
		if err := m.allowBlockMajorMinor(sourceMajor, sourceMinor, computeCGroupPath); err != nil {
			return err
		}
	}
	return nil
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

func (m *volumeMounter) getSourceMajorMinor(sourceUID types.UID, volumeName string) (int64, int64, string, error) {
	result := make([]int64, 2)
	perms := ""
	if sourceUID != types.UID("") {
		basepath := filepath.Join(deviceBasePath(sourceUID), volumeName)
		err := filepath.Walk(basepath, func(filePath string, info os.FileInfo, err error) error {
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
				if m.isBlockFile(path) {
					result[0], result[1], perms, err = m.getBlockFileMajorMinor(path)
					// Err != nil means not a block device or unable to determine major/minor, try next file
					if err == nil {
						// Successfully located
						return io.EOF
					}
				}
				return nil
			}
			return nil
		})
		if err != nil && err != io.EOF {
			return -1, -1, "", err
		}
	}
	if perms == "" {
		return -1, -1, "", fmt.Errorf("Unable to find block device")
	}
	return result[0], result[1], perms, nil
}

func (m *volumeMounter) isBlockFile(fileName string) bool {
	// Stat the file and see if there is no error
	out, err := statCommand(fileName)
	if err != nil {
		// Not a block device skip to next file
		return false
	}
	split := strings.Split(string(out), ",")
	// Verify I got 4 strings
	if len(split) != 4 {
		return false
	}
	return strings.TrimSpace(split[3]) == "block special file"
}

func (m *volumeMounter) getBlockFileMajorMinor(fileName string) (int64, int64, string, error) {
	result := make([]int, 2)
	// Stat the file and see if there is no error
	out, err := statCommand(fileName)
	if err != nil {
		// Not a block device skip to next file
		return -1, -1, "", err
	}
	split := strings.Split(string(out), ",")
	// Verify I got 4 strings
	if len(split) != 4 {
		return -1, -1, "", fmt.Errorf("Output invalid")
	}
	if strings.TrimSpace(split[3]) != "block special file" {
		return -1, -1, "", fmt.Errorf("Not a block device")
	}
	// Verify that both values are ints.
	for i := 0; i < 2; i++ {
		val, err := strconv.ParseInt(split[i], 16, 32)
		if err != nil {
			return -1, -1, "", err
		}
		result[i] = int(val)
	}
	return int64(result[0]), int64(result[1]), split[2], nil
}

// getTargetCgroupPath returns the container cgroup path of the compute container in the pod.
func (m *volumeMounter) getTargetCgroupPath(vmi *v1.VirtualMachineInstance) (string, error) {
	basePath := cgroupsBasePath()
	isoRes, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return "", err
	}

	virtlauncherCgroupPath := filepath.Join(basePath, isoRes.Slice())
	fileInfo, err := os.Stat(virtlauncherCgroupPath)
	if err != nil {
		return "", err
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("detected path %s, but it is not a directory", virtlauncherCgroupPath)
	}
	return virtlauncherCgroupPath, nil
}

func (m *volumeMounter) removeBlockMajorMinor(major, minor int64, path string) error {
	return m.updateBlockMajorMinor(major, minor, path, false)
}

func (m *volumeMounter) allowBlockMajorMinor(major, minor int64, path string) error {
	return m.updateBlockMajorMinor(major, minor, path, true)
}

func (m *volumeMounter) updateBlockMajorMinor(major, minor int64, path string, allow bool) error {
	deviceRule := &configs.DeviceRule{
		Type:        configs.BlockDevice,
		Major:       major,
		Minor:       minor,
		Permissions: "rwm",
		Allow:       allow,
	}
	if err := m.updateDevicesList(path, deviceRule); err != nil {
		return err
	}
	return nil
}

func (m *volumeMounter) loadEmulator(path string) (*devices.Emulator, error) {
	list, err := fscommon.ReadFile(path, "devices.list")
	if err != nil {
		return nil, err
	}
	return devices.EmulatorFromList(bytes.NewBufferString(list))
}

func (m *volumeMounter) updateDevicesList(path string, rule *configs.DeviceRule) error {
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
	if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
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
		// Reference to Blacklist is in external API
		if !target.IsBlacklist() && !reflect.DeepEqual(currentAfter, target) {
			return errors.New("resulting devices cgroup doesn't precisely match target")
		} else if target.IsBlacklist() != currentAfter.IsBlacklist() {
			return errors.New("resulting devices cgroup doesn't match target mode")
		}
	}
	return nil
}

func (m *volumeMounter) createBlockDeviceFile(deviceName string, major, minor int64, blockDevicePermissions string) (string, error) {
	exists, err := diskutils.FileExists(deviceName)
	if err != nil {
		return "", err
	}
	if !exists {
		out, err := mknodCommand(deviceName, major, minor, blockDevicePermissions)
		if err != nil {
			log.DefaultLogger().Errorf("Error creating block device file: %s, %v", out, err)
			return "", err
		}
	}
	return deviceName, nil
}

func (m *volumeMounter) mountFileSystemHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetDisk, err := m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume, false)
	if err != nil {
		return err
	}

	if isMounted, err := isMounted(targetDisk); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetDisk, err)
	} else if !isMounted {
		sourcePath, err := m.getSourcePodFilePath(sourceUID, vmi, volume)
		if err != nil {
			log.DefaultLogger().V(3).Infof("Error getting source path: %v", err)
			// We are eating the error to avoid spamming the log with errors, it might take a while for the volume
			// to get mounted on the node, and this will error until the volume is mounted.
			return nil
		}
		if err := m.writePathToMountRecord(targetDisk, vmi, record); err != nil {
			return err
		}
		targetDisk, err := m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume, true)
		if err != nil {
			return err
		}
		if out, err := mountCommand(filepath.Join(sourcePath, "disk.img"), targetDisk); err != nil {
			return fmt.Errorf("failed to bindmount hotplug-disk %v: %v : %v", volume, string(out), err)
		}
	} else {
		return nil
	}
	return nil
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

func (m *volumeMounter) getSourcePodFilePath(sourceUID types.UID, vmi *v1.VirtualMachineInstance, volume string) (string, error) {
	iso := isolationDetector("/path")
	isoRes, err := iso.DetectForSocket(vmi, socketPath(sourceUID))
	if err != nil {
		return "", err
	}
	findmounts, err := LookupFindmntInfoByVolume(volume, isoRes.Pid())
	if err != nil {
		return "", err
	}
	for _, findmnt := range findmounts {
		if filepath.Base(findmnt.Target) == volume {
			source := findmnt.GetSourcePath()
			isBlock, _ := isBlockDevice(filepath.Join(util.HostRootMount, source))
			if _, err := os.Stat(filepath.Join(util.HostRootMount, source)); os.IsNotExist(err) || isBlock {
				// file not found, or block device, or directory check if we can find the mount.
				deviceFindMnt, err := LookupFindmntInfoByDevice(source)
				if err != nil {
					// Try the device found from the source
					deviceFindMnt, err = LookupFindmntInfoByDevice(findmnt.GetSourceDevice())
					if err != nil {
						return "", err
					}
					// Check if the path was relative to the device.
					if _, err := os.Stat(filepath.Join(util.HostRootMount, source)); err != nil {
						return filepath.Join(deviceFindMnt[0].Target, source), nil
					}
					return "", err
				}
				return deviceFindMnt[0].Target, nil
			} else {
				return source, nil
			}
		}
	}
	// Did not find the disk image file, return error
	return "", fmt.Errorf("unable to find source disk image path for pod %s", sourceUID)
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
			return err
		}
		for _, volumeStatus := range vmi.Status.VolumeStatus {
			if volumeStatus.HotplugVolume == nil {
				continue
			}
			if m.isBlockVolume(&vmi.Status, volumeStatus.Name) {
				path := filepath.Join(basePath, volumeStatus.Name)
				currentHotplugPaths[path] = virtlauncherUID
			} else {
				path, err := m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volumeStatus.Name, false)
				if err != nil {
					return err
				}
				currentHotplugPaths[path] = virtlauncherUID
			}
		}
		newRecord := vmiMountTargetRecord{
			MountTargetEntries: make([]vmiMountTargetEntry, 0),
		}
		for _, entry := range record.MountTargetEntries {
			diskPath := entry.TargetFile
			if _, ok := currentHotplugPaths[diskPath]; !ok {
				if m.isBlockFile(diskPath) {
					if err := m.unmountBlockHotplugVolumes(diskPath, vmi); err != nil {
						return err
					}
				} else {
					if err := m.unmountFileSystemHotplugVolumes(diskPath); err != nil {
						return err
					}
				}
			} else {
				newRecord.MountTargetEntries = append(newRecord.MountTargetEntries, vmiMountTargetEntry{
					TargetFile: diskPath,
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

func (m *volumeMounter) unmountFileSystemHotplugVolumes(diskPath string) error {
	if mounted, err := isMounted(diskPath); err != nil {
		return fmt.Errorf("failed to check mount point for hotplug disk %v: %v", diskPath, err)
	} else if mounted {
		out, err := unmountCommand(diskPath)
		if err != nil {
			return fmt.Errorf("failed to unmount hotplug disk %v: %v : %v", diskPath, string(out), err)
		}
		err = os.Remove(diskPath)
		if err != nil {
			return fmt.Errorf("failed to remove hotplug disk directory %v: %v : %v", diskPath, string(out), err)
		}

	}
	return nil
}

func (m *volumeMounter) unmountBlockHotplugVolumes(diskPath string, vmi *v1.VirtualMachineInstance) error {
	// Get major and minor so we can deny the container.
	major, minor, _, err := m.getBlockFileMajorMinor(diskPath)
	if err != nil {
		return err
	}
	// Delete block device file
	err = os.Remove(diskPath)
	if err != nil {
		return err
	}
	path, err := m.getTargetCgroupPath(vmi)
	if err != nil {
		return err
	}
	if err := m.removeBlockMajorMinor(major, minor, path); err != nil {
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
			diskPath := entry.TargetFile
			if m.isBlockFile(diskPath) {
				if err := m.unmountBlockHotplugVolumes(diskPath, vmi); err != nil {
					logger.Infof("Unable to remove block device at path %s: %v", diskPath, err)
					// Don't return error, try next.
				}
			} else {
				if err := m.unmountFileSystemHotplugVolumes(diskPath); err != nil {
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
		return false, err
	}
	if m.isBlockVolume(&vmi.Status, volume) {
		deviceName := filepath.Join(targetPath, volume)
		isBlockExists, _ := isBlockDevice(deviceName)
		return isBlockExists, nil
	}
	return isMounted(filepath.Join(targetPath, fmt.Sprintf("%s.img", volume)))
}
