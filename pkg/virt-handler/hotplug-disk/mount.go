package hotplug_volume

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"github.com/opencontainers/runc/libcontainer/configs"

	"github.com/opencontainers/runc/libcontainer/devices"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
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

	//cgroupsBasePath = func() string {
	//	return filepath.Join("/proc/1/root", cgroup.ControllerPath("devices"))
	//}

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
		return isolation.NewSocketBasedIsolationDetector(path)
	}
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type volumeMounter struct {
	//podIsolationDetector isolation.PodIsolationDetector // ihol3 remove
	cgroupManager      cgroup.Manager
	mountStateDir      string
	mountRecords       map[types.UID]*vmiMountTargetRecord
	mountRecordsLock   sync.Mutex
	skipSafetyCheck    bool
	hotplugDiskManager hotplugdisk.HotplugDiskManagerInterface
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
func NewVolumeMounter( /*isoDetector isolation.PodIsolationDetector,*/ mountStateDir string) VolumeMounter {
	return &volumeMounter{
		//podIsolationDetector: isoDetector, // ihol3 remove
		mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
		mountStateDir:      mountStateDir,
		hotplugDiskManager: hotplugdisk.NewHotplugDiskManager(),
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
		if m.isBlockVolume(sourceUID, volumeName) {
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
func (m *volumeMounter) isBlockVolume(sourceUID types.UID, volumeName string) bool {
	// Check if the volumeDevices directory exists in the attachment pod, if so, its a block device, otherwise its file system.
	if sourceUID != types.UID("") {
		devicePath := filepath.Join(deviceBasePath(sourceUID), volumeName)
		info, err := os.Stat(devicePath)
		if err != nil {
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
	targetPath, err := m.hotplugDiskManager.GetHotplugTargetPodPathOnHost(virtlauncherUID)
	if err != nil {
		return err
	}

	deviceName := filepath.Join(targetPath, volume)
	isolationRes, err := detectVMIsolation(vmi)
	if err != nil {
		return err
	}

	pid := isolationRes.Pid()

	isMigrationInProgress := vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.Completed

	cgroupsManager, err := getCgroupsManager(vmi, sourceUID)
	if err != nil {
		return fmt.Errorf("could not create cgroup manager. err: %v", err)
	}

	if isBlockExists, _ := isBlockDevice(deviceName); !isBlockExists {
		log.Log.Infof("hotplug [mountBlockHotplugVolume]: isBlockExists, _ := isBlockDevice(deviceName); !isBlockExists")
		//computeCGroupPath, err := m.getTargetCgroupPath(vmi)
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
		log.Log.Infof("hotplug [mountBlockHotplugVolume]: FINISHED writePathToMountRecord. err: %v", err)
		// allow block devices
		if err := m.allowBlockMajorMinor(sourceMajor, sourceMinor, cgroupsManager, pid); err != nil {
			return err
		}
		log.Log.Infof("hotplug [mountBlockHotplugVolume]: FINISHED allowBlockMajorMinor. err: %v", err)

		if _, err = m.createBlockDeviceFile(deviceName, sourceMajor, sourceMinor, permissions); err != nil {
			return err
		}
		log.Log.Infof("hotplug [mountBlockHotplugVolume]: FINISHED createBlockDeviceFile. err: %v", err)
	} else if isBlockExists && (!m.volumeStatusReady(volume, vmi) || isMigrationInProgress) {

		log.Log.Infof("hotplug [mountBlockHotplugVolume]: isBlockExists && !m.volumeStatusReady(volume, vmi)")
		// Block device exists already, but the volume is not ready yet, ensure that the device is allowed.
		//computeCGroupPath, err := m.getTargetCgroupPath(vmi)
		if err != nil {
			return err
		}
		sourceMajor, sourceMinor, _, err := m.getSourceMajorMinor(sourceUID, volume)
		if err != nil {
			return err
		}
		if err := m.allowBlockMajorMinor(sourceMajor, sourceMinor, cgroupsManager, pid); err != nil {
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

//// getTargetCgroupPath returns the container cgroup path of the compute container in the pod.
//func (m *volumeMounter) getTargetCgroupPath(vmi *v1.VirtualMachineInstance) (string, error) {
//	basePath := cgroupsBasePath()
//	isoRes, err := m.podIsolationDetector.Detect(vmi)
//	if err != nil {
//		return "", err
//	}
//
//	virtlauncherCgroupPath := filepath.Join(basePath, )//isoRes.Slice())
//	fileInfo, err := os.Stat(virtlauncherCgroupPath)
//	if err != nil {
//		return "", err
//	}
//	if !fileInfo.IsDir() {
//		return "", fmt.Errorf("detected path %s, but it is not a directory", virtlauncherCgroupPath)
//	}
//	return virtlauncherCgroupPath, nil
//}

func (m *volumeMounter) removeBlockMajorMinor(major, minor int64, manager cgroup.Manager) error {
	//idx := strings.Index(path, "/sys/fs/cgroup/")
	//chrootPath := path[:idx] // ihol3 rename me
	//newPath := path[idx:]

	//return cgroup.RunWithChroot(cgroup.HostRootPath, func() error {
	return m.updateBlockMajorMinor(major, minor, false, manager, -1)
	//})
}

// DELETE ME!!!!!!! ihol3
func logRootFiles(name string, path string) {
	const filePattern = " (name: %s, is dir? %v) "
	filesStr := ""
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Log.Infof("hotplug [%s]. ERR READING FILES: %v", name, err)
	}

	for _, f := range files {
		filesStr += fmt.Sprintf(filePattern, f.Name(), f.IsDir())
	}

	log.Log.Infof("hotplug [%s]. ls on %s: [%s]", name, path, filesStr)
}

func (m *volumeMounter) allowBlockMajorMinor(major, minor int64, manager cgroup.Manager, pid int) error {
	//const cgroupBase = "/sys/fs/cgroup/devices/"
	//cgroupBase, err := manager.GetBasePathToHostSubsystem("devices")
	//if err != nil {
	//	// ihol3 maybe expand error
	//	return err
	//}

	//idx := strings.Index(path, cgroupBase)
	//var newPath string
	//
	//if idx > -1 {
	//	newPath = path[idx:]
	//} else {
	//	//newPath := filepath.Join("/sys/fs/cgroup/", path)
	//	//_, err := os.Stat(newPath)
	//	//return m.updateBlockMajorMinor(major, minor, "", true, manager)
	//	newPath = path
	//}
	log.Log.Infof("hotplug [allowBlockMajorMinor]. PATHS=%s", manager.GetPaths())
	//logRootFiles("allowBlockMajorMinor", "/")
	//logRootFiles("allowBlockMajorMinor", cgroup.HostRootPath)

	//if pid > -1 {
	//	err = manager.Apply(pid)
	//	log.Log.Infof("hotplug [updateBlockMajorMinor]: APPLIED. err: %v", err)
	//	log.Log.Infof("hotplug [allowBlockMajorMinor]. PATHS (after applying)=%s", manager.GetPaths())
	//}

	log.Log.Infof("hotplug [allowBlockMajorMinor]. CHROOTING TO: %s", cgroup.HostRootPath)

	//return cgroup.RunWithChroot(cgroup.HostRootPath, func() error {
	return m.updateBlockMajorMinor(major, minor, true, manager, pid)
	//})

	//return m.updateBlockMajorMinor(major, minor, true, manager, pid)
}

func (m *volumeMounter) updateBlockMajorMinor(major, minor int64, allow bool, manager cgroup.Manager, pid int) error {
	var err error
	deviceRule := &devices.Rule{
		Type:        devices.BlockDevice,
		Major:       major,
		Minor:       minor,
		Permissions: "rwm",
		Allow:       allow,
	}
	log.Log.Infof("hotplug [updateBlockMajorMinor]: major == %v, minor == %v", major, minor)

	//err := manager.Set(&configs.Resources{
	//	Devices: []*devices.Rule{deviceRule},
	//})
	devicesPath, ok := manager.GetPaths()[""]
	//devicesPath = filepath.Join("/sys/fs/cgroup/", devicesPath)
	log.Log.Infof("hotplug [updateBlockMajorMinor]: devicesPath() == %s, ok == %v", devicesPath, ok)

	if _, err := os.Stat(devicesPath); os.IsNotExist(err) {
		log.Log.Infof("hotplug [updateBlockMajorMinor]: devicesPath does NOT exist!!!!!!!!!!!!!!!!")
	}

	logRootFiles("updateBlockMajorMinor", devicesPath)

	log.Log.Infof("hotplug [updateBlockMajorMinor]: RULE -> %+v", deviceRule)
	//err = set_del(devicesPath, &configs.Resources{
	//	Devices: []*devices.Rule{deviceRule},
	//})

	const permissions = "rwm"
	const toAllow = true

	defaultRules := []*devices.Rule{
		deviceRule,
		{ // /dev/ptmx (PTY master multiplex)
			Type:        devices.CharDevice,
			Major:       5,
			Minor:       2,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/null (Null device)
			Type:        devices.CharDevice,
			Major:       1,
			Minor:       3,
			Permissions: permissions,
			Allow:       toAllow,
		},
		{ // /dev/pts/... (PTY slaves)
			Type:        devices.CharDevice,
			Major:       136,
			Minor:       -1,
			Permissions: permissions,
			Allow:       toAllow,
		},
	}

	err = manager.Set(&configs.Resources{
		Devices: defaultRules,
	})
	logRootFiles("updateBlockMajorMinor", "/")

	log.Log.Infof("setting rule. err: %v", err)

	return err

	//var manager cgroups.Manager
	//var err error
	//var config *configs.Cgroup
	//var dirPath string
	//var rootless bool
	//dirPath = path

	//if !cgroups.IsCgroup2UnifiedMode() {
	//	// ihol3
	//	// key is cgroup. how do I get it?
	//	//cgroups.cgrou
	//	//manager = fs.NewManager(config, map[string]string{"devices": dirPath}, rootless)
	//	//deviceManager := manager.(*fs.DevicesGroup)
	//
	//	//m.podIsolationDetector.De
	//
	//	if err := m.updateDevicesList(path, deviceRule); err != nil {
	//		return err
	//	} else {
	//		return nil
	//	}
	//} else {
	//	//manager, err = fs2.NewManager(config, dirPath, rootless)
	//	resourceConfig := &configs.Resources{
	//		Devices:     []*devices.Rule{deviceRule},
	//		SkipDevices: false,
	//	}
	//	if err := m.setDevices(path, resourceConfig); err != nil {
	//		return err
	//	} else {
	//		return nil
	//	}
	//}
	//path_cgroups := manager.Path()
	//
	//if err != nil {
	//	return err
	//}
	//
	//err = manager.Set(&configs.Resources{
	//	Devices: []*devices.Rule{deviceRule},
	//})
	//if err != nil {
	//	return err
	//}

	//return nil
}

//func (m *volumeMounter) loadEmulator(path string) (*cgroupdevices.Emulator, error) {
//	list, err := fscommon.ReadFile(path, "devices.list")
//	if err != nil {
//		return nil, err
//	}
//	return cgroupdevices.EmulatorFromList(bytes.NewBufferString(list))
//}

// ihol3 delete this and others that aren't in use here
//func (m *volumeMounter) updateDevicesList(path string, rule *configs.DeviceRule) error {
//	// Create the target emulator for comparison later.
//	target, err := m.loadEmulator(path)
//	if err != nil {
//		return err
//	}
//	target.Apply(*rule)
//
//	file := "devices.deny"
//	if rule.Allow {
//		file = "devices.allow"
//	}
//	if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
//		return err
//	}
//
//	// Final safety check -- ensure that the resulting state is what was
//	// requested. This is only really correct for white-lists, but for
//	// black-lists we can at least check that the cgroup is in the right mode.
//	currentAfter, err := m.loadEmulator(path)
//	if err != nil {
//		return err
//	}
//	if !m.skipSafetyCheck {
//		if !target.IsBlacklist() && !reflect.DeepEqual(currentAfter, target) {
//			return errors.New("resulting devices cgroup doesn't precisely match target")
//		} else if target.IsBlacklist() != currentAfter.IsBlacklist() {
//			return errors.New("resulting devices cgroup doesn't match target mode")
//		}
//	}
//	return nil
//}

func (m *volumeMounter) createBlockDeviceFile(deviceName string, major, minor int64, blockDevicePermissions string) (string, error) {
	log.Log.Infof("hotplug [createBlockDeviceFile]: deviceName == %s", deviceName)
	exists, err := diskutils.FileExists(deviceName)
	log.Log.Infof("hotplug [createBlockDeviceFile]: exists == %v, err: %v", exists, err)
	if err != nil {
		return "", err
	}
	if !exists {
		out, err := mknodCommand(deviceName, major, minor, blockDevicePermissions)
		log.Log.Infof("hotplug [createBlockDeviceFile]: MKNOD! err: %v, out: %v", err, out)
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
	targetDisk, err := m.hotplugDiskManager.GetFileSystemDiskTargetPathFromHostView(virtlauncherUID, volume, true)
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
			if m.isBlockVolume(volumeStatus.HotplugVolume.AttachPodUID, volumeStatus.Name) {
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
	cgroupsManager, err := getCgroupsManager(vmi, "")
	if err != nil {
		return fmt.Errorf("could not create cgroup manager. err: %v", err)
	}

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
	//path, err := m.getTargetCgroupPath(vmi)
	//if err != nil {
	//	return err
	//}
	if err := m.removeBlockMajorMinor(major, minor, cgroupsManager); err != nil {
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
	if m.isBlockVolume(sourceUID, volume) {
		deviceName := filepath.Join(targetPath, volume)
		isBlockExists, _ := isBlockDevice(deviceName)
		return isBlockExists, nil
	}
	return isMounted(filepath.Join(targetPath, fmt.Sprintf("%s.img", volume)))
}

//func (m *volumeMounter) setDevices(dirPath string, r *configs.Resources) error {
//	if r.SkipDevices {
//		return nil
//	}
//	// XXX: This is currently a white-list (but all callers pass a blacklist of
//	//      devices). This is bad for a whole variety of reasons, but will need
//	//      to be fixed with co-ordinated effort with downstreams.
//	insts, license, err := devicefilter.DeviceFilter(r.Devices)
//	if err != nil {
//		return err
//	}
//	dirFD, err := unix.Open(dirPath, unix.O_DIRECTORY|unix.O_RDONLY, 0600)
//	if err != nil {
//		return fmt.Errorf("cannot get dir FD for %s", dirPath)
//	}
//	defer unix.Close(dirFD)
//	// XXX: This code is currently incorrect when it comes to updating an
//	//      existing cgroup with new rules (new rulesets are just appended to
//	//      the program list because this uses BPF_F_ALLOW_MULTI). If we didn't
//	//      use BPF_F_ALLOW_MULTI we could actually atomically swap the
//	//      programs.
//	//
//	//      The real issue is that BPF_F_ALLOW_MULTI makes it hard to have a
//	//      race-free blacklist because it acts as a whitelist by default, and
//	//      having a deny-everything program cannot be overridden by other
//	//      programs. You could temporarily insert a deny-everything program
//	//      but that would result in spurrious failures during updates.
//	if _, err := ebpf.LoadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
//		if !canSkipEBPFError(r) {
//			return err
//		}
//	}
//	return nil
//}
//
//// This is similar to the logic applied in crun for handling errors from bpf(2)
//// <https://github.com/containers/crun/blob/0.17/src/libcrun/cgroup.c#L2438-L2470>.
//func canSkipEBPFError(r *configs.Resources) bool {
//	// If we're running in a user namespace we can ignore eBPF rules because we
//	// usually cannot use bpf(2), as well as rootless containers usually don't
//	// have the necessary privileges to mknod(2) device inodes or access
//	// host-level instances (though ideally we would be blocking device access
//	// for rootless containers anyway).
//	if userns.RunningInUserNS() {
//		return true
//	}
//
//	// We cannot ignore an eBPF load error if any rule if is a block rule or it
//	// doesn't permit all access modes.
//	//
//	// NOTE: This will sometimes trigger in cases where access modes are split
//	//       between different rules but to handle this correctly would require
//	//       using ".../libcontainer/cgroup/devices".Emulator.
//	for _, dev := range r.Devices {
//		if !dev.Allow || !isRWM(dev.Permissions) {
//			return false
//		}
//	}
//	return true
//}
//
//func isRWM(perms devices.Permissions) bool {
//	var r, w, m bool
//	for _, perm := range perms {
//		switch perm {
//		case 'r':
//			r = true
//		case 'w':
//			w = true
//		case 'm':
//			m = true
//		}
//	}
//	return r && w && m
//}

// ihol3 consider to delete
func getCgroupsManager(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (manager cgroup.Manager, err error) {
	manager, err = cgroup.NewManagerFromVM(vmi)
	log.Log.Object(vmi).Infof("creating new manager. err: %v", err)

	//if sourceUID == "" {
	//	manager, err = cgroup.NewManagerFromVM(vmi)
	//	log.Log.Object(vmi).Infof("creating new manager. err: %v", err)
	//} else {
	//	socketPath := socketPath(sourceUID) // ihol3 refactor to getSocketPath
	//	manager, err = cgroup.NewManagerFromVMAndSocket(vmi, socketPath)
	//	log.Log.Object(vmi).Infof("creating new manager. socket: \"%s\", err: %v", socketPath, err)
	//}

	return manager, err
}

// --------------- DELETE THOSE:

//func set_del(path string, r *configs.Resources) error {
//	log.Log.Infof("hotplug [set_del]: userns.RunningInUserNS() || r.SkipDevices == %v", userns.RunningInUserNS() || r.SkipDevices)
//	if userns.RunningInUserNS() || r.SkipDevices {
//		return nil
//	}
//
//	// Generate two emulators, one for the current state of the cgroup and one
//	// for the requested state by the user.
//	current, err := loadEmulator(path)
//	if err != nil {
//		return err
//	}
//	target, err := buildEmulator(r.Devices)
//	if err != nil {
//		return err
//	}
//
//	log.Log.Infof("hotplug [set_del]: current == %v, target == %v", current, target)
//	rule := *r.Devices[0]
//	//current.Apply(rule)
//	//log.Log.Infof("hotplug [set_del]: APPLYING rule == %v. target == %v", rule, target)
//	file := "devices.deny"
//	if rule.Allow {
//		file = "devices.allow"
//	}
//	content, err := fscommon.ReadFile(path, "devices.list")
//	log.Log.Infof("hotplug [set_del]: ReadFile - err: %v, Content: %s", err, content)
//
//	//if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
//	//	return err
//	//}
//	//log.Log.Infof("hotplug [set_del]: WriteFile - ERR: %v", err)
//	//log.Log.Infof("hotplug [set_del]: WriteFile - Rule: %s", rule.CgroupString())
//
//	//Compute the minimal set of transition rules needed to achieve the
//	//requested state.
//	transitionRules, err := transition(target, current)
//	if err != nil {
//		return err
//	}
//
//	log.Log.Infof("hotplug [set_del]: len(transitionRules) == %v", len(transitionRules))
//	log.Log.Infof("hotplug [set_del]: transitionRules == %v", transitionRules)
//	for _, rule := range transitionRules {
//		file := "devices.deny"
//		if rule.Allow {
//			file = "devices.allow"
//		}
//		log.Log.Infof("hotplug [set_del]: path == %v, file == %v, rule.CgroupString() == %v, ", path, file, rule.CgroupString())
//		if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
//			return err
//		}
//	}
//
//	if len(transitionRules) == 0 {
//		log.Log.Infof("hotplug [set_del]: len(transitionRules) == 0")
//
//		if err := fscommon.WriteFile(path, file, rule.CgroupString()); err != nil {
//			return err
//		}
//		log.Log.Infof("hotplug [set_del]: WriteFile - ERR: %v", err)
//		log.Log.Infof("hotplug [set_del]: WriteFile - Rule: %s", rule.CgroupString())
//	}
//
//	content, err = fscommon.ReadFile(path, "devices.list")
//	log.Log.Infof("hotplug [set_del]: ReadFile - err: %v, Content: %s", err, content)
//
//	// Final safety check -- ensure that the resulting state is what was
//	// requested. This is only really correct for white-lists, but for
//	// black-lists we can at least check that the cgroup is in the right mode.
//	//
//	// This safety-check is skipped for the unit tests because we cannot
//	// currently mock devices.list correctly.
//	//currentAfter, err := loadEmulator(path)
//	//if err != nil {
//	//	return err
//	//}
//	//if !target.IsBlacklist() && !reflect.DeepEqual(currentAfter, target) {
//	//	return errors.New("resulting devices cgroup doesn't precisely match target")
//	//} else if target.IsBlacklist() != currentAfter.IsBlacklist() {
//	//	return errors.New("resulting devices cgroup doesn't match target mode")
//	//}
//	return nil
//}

//func loadEmulator(path string) (*cgroupdevices.Emulator, error) {
//	list, err := fscommon.ReadFile(path, "devices.list")
//	if err != nil {
//		return nil, err
//	}
//	return cgroupdevices.EmulatorFromList(bytes.NewBufferString(list))
//}
//
//func buildEmulator(rules []*devices.Rule) (*cgroupdevices.Emulator, error) {
//	// This defaults to a white-list -- which is what we want!
//	emu := &cgroupdevices.Emulator{}
//	for _, rule := range rules {
//		if err := emu.Apply(*rule); err != nil {
//			return nil, err
//		}
//	}
//	return emu, nil
//}

//func transition(target *cgroupdevices.Emulator, source *cgroupdevices.Emulator) ([]*devices.Rule, error) {
//	var transitionRules []*devices.Rule
//	oldRules := source.Rules
//
//	// If the default policy doesn't match, we need to include a "disruptive"
//	// rule (either allow-all or deny-all) in order to switch the cgroup to the
//	// correct default policy.
//	//
//	// However, due to a limitation in "devices.list" we cannot be sure what
//	// deny rules are in place in a black-list cgroup. Thus if the source is a
//	// black-list we also have to include a disruptive rule.
//	if source.IsBlacklist() || source.DefaultAllow != target.DefaultAllow {
//		transitionRules = append(transitionRules, &devices.Rule{
//			Type:        'a',
//			Major:       -1,
//			Minor:       -1,
//			Permissions: devices.Permissions("rwm"),
//			Allow:       target.DefaultAllow,
//		})
//		// The old rules are only relevant if we aren't starting out with a
//		// disruptive rule.
//		oldRules = nil
//	}
//
//	// NOTE: We traverse through the rules in a sorted order so we always write
//	//       the same set of rules (this is to aid testing).
//
//	// First, we create inverse rules for any old rules not in the new set.
//	// This includes partial-inverse rules for specific permissions. This is a
//	// no-op if we added a disruptive rule, since oldRules will be empty.
//	log.Log.Infof("hotplug [transition]: oldRules == %v, ", oldRules.OrderedEntries())
//	log.Log.Infof("hotplug [transition]: target.Rules == %v, ", target.Rules.OrderedEntries())
//
//	for _, rule := range oldRules.OrderedEntries() {
//		meta, oldPerms := rule.Meta, rule.Perms
//		newPerms := target.Rules[meta]
//		droppedPerms := oldPerms.Difference(newPerms)
//		log.Log.Infof("hotplug [transition]: oldPerms == %v, ", oldPerms)
//		log.Log.Infof("hotplug [transition]: oldPerms == %v, ", newPerms)
//		log.Log.Infof("hotplug [transition]: difference(droppedPerms) == %v, ", droppedPerms)
//		if !droppedPerms.IsEmpty() {
//			transitionRules = append(transitionRules, &devices.Rule{
//				Type:        meta.Node,
//				Major:       meta.Major,
//				Minor:       meta.Minor,
//				Permissions: droppedPerms,
//				Allow:       target.DefaultAllow,
//			})
//		}
//	}
//
//	// Add any additional rules which weren't in the old set. We happen to
//	// filter out rules which are present in both sets, though this isn't
//	// strictly necessary.
//	for _, rule := range target.Rules.OrderedEntries() {
//		meta, newPerms := rule.Meta, rule.Perms
//		oldPerms := oldRules[meta]
//		gainedPerms := newPerms.Difference(oldPerms)
//		log.Log.Infof("hotplug [transition]: oldPerms == %v, ", oldPerms)
//		log.Log.Infof("hotplug [transition]: oldPerms == %v, ", newPerms)
//		log.Log.Infof("hotplug [transition]: difference(gainedPerms) == %v, ", gainedPerms)
//		if !gainedPerms.IsEmpty() {
//			transitionRules = append(transitionRules, &devices.Rule{
//				Type:        meta.Node,
//				Major:       meta.Major,
//				Minor:       meta.Minor,
//				Permissions: gainedPerms,
//				Allow:       !target.DefaultAllow,
//			})
//		}
//	}
//	return transitionRules, nil
//}

func detectVMIsolation(vm *v1.VirtualMachineInstance) (isolationRes isolation.IsolationResult, err error) {
	const detectionErrFormat = "cannot detect vm \"%s\", err: %v"
	detector := isolation.NewSocketBasedIsolationDetector(util.VirtShareDir)

	isolationRes, err = detector.Detect(vm)

	if err != nil {
		return nil, fmt.Errorf(detectionErrFormat, vm.Name, err)
	}

	return isolationRes, nil
}
