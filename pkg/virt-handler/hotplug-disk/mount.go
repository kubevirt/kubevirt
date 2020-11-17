package hotplug_volume

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

var (
	deviceBasePath = func(podUID types.UID) string {
		return fmt.Sprintf("/proc/1/root/var/lib/kubelet/pods/%s/volumeDevices", string(podUID))
	}

	sourcePodBasePath = func(podUID types.UID) string {
		return fmt.Sprintf("/proc/1/root/var/lib/kubelet/pods/%s/volumes", string(podUID))
	}

	cgroupsBasePath = func() string {
		return "/proc/1/root/sys/fs/cgroup/devices/"
	}

	statCommand = func(fileName string) ([]byte, error) {
		return exec.Command("/usr/bin/stat", fileName, "-L", "-c%t,%T,%a,%F").CombinedOutput()
	}

	mknodCommand = func(deviceName string, major, minor int, blockDevicePermissions string) ([]byte, error) {
		return exec.Command("/usr/bin/mknod", "--mode", fmt.Sprintf("0%s", blockDevicePermissions), deviceName, "b", strconv.Itoa(major), strconv.Itoa(minor)).CombinedOutput()
	}

	mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
		return exec.Command("/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "mount", "-o", "bind", strings.TrimPrefix(sourcePath, isolation.NodeIsolationResult().MountRoot()), targetPath).CombinedOutput()
	}

	unmountCommand = func(diskPath string) ([]byte, error) {
		return exec.Command("/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "umount", diskPath).CombinedOutput()
	}

	isMounted = func(path string) (bool, error) {
		return isolation.NodeIsolationResult().IsMounted(path)
	}

	isBlockDevice = func(path string) (bool, error) {
		return isolation.NodeIsolationResult().IsBlockDevice(path)
	}
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type volumeMounter struct {
	podIsolationDetector isolation.PodIsolationDetector
	mountStateDir        string
	mountRecords         map[types.UID]*vmiMountTargetRecord
	mountRecordsLock     sync.Mutex
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

	err = os.MkdirAll(filepath.Dir(recordFile), 0755)
	if err != nil {
		return err
	}

	log.DefaultLogger().V(1).Infof("Writing the following to the record file: %s", bytes)
	err = ioutil.WriteFile(recordFile, bytes, 0644)
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

	return nil
}

// isBlockVolume checks if the volumeDevices directory exists in the pod path, we assume there is a single volume associated with
// each pod, we use this knowledge to determine if we have a block volume or not.
func (m *volumeMounter) isBlockVolume(sourceUID types.UID) bool {
	// Check if the volumeDevices directory exists in the attachment pod, if so, its a block device, otherwise its file system.
	if sourceUID != types.UID("") {
		devicePath := deviceBasePath(sourceUID)
		res, err := isBlockDevice(devicePath)
		if err != nil {
			log.Log.V(4).Infof("%s is not a block device %v", devicePath, err)
			return false
		}
		return res
	}
	return false
}

func (m *volumeMounter) mountBlockHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	logger := log.DefaultLogger()
	logger.V(4).Infof("Hotplug block volume: %s", volume)

	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetPath, err := hotplugdisk.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		return err
	}

	sourceMajor, sourceMinor, permissions, err := m.getSourceMajorMinor(vmi, sourceUID)
	if err != nil {
		return err
	}
	deviceName := filepath.Join(targetPath, volume)

	if isBlockExists, _ := isBlockDevice(deviceName); !isBlockExists {
		if err := m.writePathToMountRecord(deviceName, vmi, record); err != nil {
			return err
		}
		computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		// allow block devices
		if err := m.allowBlockMajorMinor(sourceMajor, sourceMinor, computeCGroupPath); err != nil {
			return err
		}
		if _, err = m.createBlockDeviceFile(deviceName, sourceMajor, sourceMinor, permissions); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (m *volumeMounter) getSourceMajorMinor(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (int, int, string, error) {
	result := make([]int, 2)
	perms := ""
	if sourceUID != types.UID("") {
		basepath := deviceBasePath(sourceUID)
		err := filepath.Walk(basepath, func(filePath string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && m.isBlockFile(filePath) {
				result[0], result[1], perms, err = m.getBlockFileMajorMinor(filePath)
				// Err != nil means not a block device or unable to determine major/minor, try next file
				if err == nil {
					// Successfully located
					return io.EOF
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

func (m *volumeMounter) getBlockFileMajorMinor(fileName string) (int, int, string, error) {
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
		result[i], err = strconv.Atoi(split[i])
		if err != nil {
			return -1, -1, "", err
		}
	}
	return result[0], result[1], split[2], nil
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

func (m *volumeMounter) removeBlockMajorMinor(major, minor int, path string) error {
	denyPath := filepath.Join(path, "devices.deny")
	return m.updateBlockMajorMinor(major, minor, denyPath)
}

func (m *volumeMounter) allowBlockMajorMinor(major, minor int, path string) error {
	// example: echo 'b 252:16 rwm' > /sys/fs/cgroup/devices/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod1f94197e_915d_43cf_ba30_6d6beaac7a61.slice/docker-a4ae23f2aa0794cf9c59e2f579fd809c44fda85d1ae305a9cda066f129a6267f.scope/devices.allow
	allowPath := filepath.Join(path, "devices.allow")
	return m.updateBlockMajorMinor(major, minor, allowPath)
}

func (m *volumeMounter) updateBlockMajorMinor(major, minor int, fileName string) error {
	permission := fmt.Sprintf("b %d:%d rwm", major, minor)
	file, err := os.OpenFile(fileName, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(permission)
	if err != nil {
		return err
	}
	return nil
}

func (m *volumeMounter) createBlockDeviceFile(deviceName string, major, minor int, blockDevicePermissions string) (string, error) {
	exists, err := diskutils.FileExists(deviceName)
	if err != nil {
		return "", err
	}
	if !exists {
		_, err := mknodCommand(deviceName, major, minor, blockDevicePermissions)
		if err != nil {
			return "", err
		}
	}
	return deviceName, nil
}

func (m *volumeMounter) mountFileSystemHotplugVolume(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID, record *vmiMountTargetRecord) error {
	sourcePath, err := m.getSourcePodFilePath(sourceUID)
	if err != nil {
		return err
	}

	virtlauncherUID := m.findVirtlauncherUID(vmi)
	if virtlauncherUID == "" {
		// This is not the node the pod is running on.
		return nil
	}
	targetPath, err := hotplugdisk.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume)
	if err != nil {
		return err
	}

	if isMounted, err := isMounted(targetPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetPath, err)
	} else if !isMounted {
		if err := m.writePathToMountRecord(targetPath, vmi, record); err != nil {
			return err
		}
		if out, err := mountCommand(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to bindmount hotplug-disk %v: %v : %v", volume, string(out), err)
		}
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

func (m *volumeMounter) getSourcePodFilePath(sourceUID types.UID) (string, error) {
	diskPath := ""
	if sourceUID != types.UID("") {
		basepath := sourcePodBasePath(sourceUID)
		err := filepath.Walk(basepath, func(filePath string, info os.FileInfo, err error) error {
			if path.Base(filePath) == "disk.img" {
				// Found disk image
				diskPath = path.Dir(filePath)
				return io.EOF
			}
			return nil
		})
		if err != nil && err != io.EOF {
			return diskPath, err
		}
	}
	if diskPath == "" {
		// Did not find the disk image file, return error
		return diskPath, fmt.Errorf("Unable to find source disk image path for pod %s", sourceUID)
	}
	return diskPath, nil
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

		currentHotplugPaths := make(map[string]types.UID, 0)
		virtlauncherUID := m.findVirtlauncherUID(vmi)

		basePath, err := hotplugdisk.GetHotplugTargetPodPathOnHost(virtlauncherUID)
		if err != nil {
			return err
		}
		for _, volumeStatus := range vmi.Status.VolumeStatus {
			if volumeStatus.HotplugVolume == nil {
				continue
			}
			if m.isBlockVolume(volumeStatus.HotplugVolume.AttachPodUID) {
				path := filepath.Join(basePath, volumeStatus.Name)
				currentHotplugPaths[path] = virtlauncherUID
			} else {
				path, err := hotplugdisk.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volumeStatus.Name)
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
					logger.Infof("Unable to remove block device at path %s", diskPath)
					// Don't return error, try next.
				}
			} else {
				if err := m.unmountFileSystemHotplugVolumes(diskPath); err != nil {
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
		return false, err
	}
	if m.isBlockVolume(sourceUID) {
		deviceName := filepath.Join(targetPath, volume)
		isBlockExists, _ := isBlockDevice(deviceName)
		return isBlockExists, nil
	}
	return isMounted(filepath.Join(targetPath, volume))
}
