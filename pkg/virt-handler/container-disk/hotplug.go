package container_disk

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	"kubevirt.io/kubevirt/pkg/safepath"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"kubevirt.io/client-go/log"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

type HotplugMounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time, sourceUID types.UID) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error)
	MoundAndVerifyFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (map[string]*containerdisk.DiskInfo, error)
	IsMounted(vmi *v1.VirtualMachineInstance, volumeName string) (bool, error)
	Umount(vmi *v1.VirtualMachineInstance) error
	UmountAll(vmi *v1.VirtualMachineInstance) error
	ComputeChecksums(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (*DiskChecksums, error)
}

type hotplugMounter struct {
	podIsolationDetector   isolation.PodIsolationDetector
	mountStateDir          string
	mountRecords           map[types.UID]*vmiMountTargetRecord
	mountRecordsLock       sync.Mutex
	suppressWarningTimeout time.Duration
	clusterConfig          *virtconfig.ClusterConfig
	nodeIsolationResult    isolation.IsolationResult

	hotplugPathGetter containerdisk.HotplugSocketPathGetter
	hotplugManager    hotplugdisk.HotplugDiskManagerInterface
}

func (m *hotplugMounter) IsMounted(vmi *v1.VirtualMachineInstance, volumeName string) (bool, error) {
	virtLauncherUID := m.findVirtlauncherUID(vmi)
	if virtLauncherUID == "" {
		return false, nil
	}
	target, err := m.hotplugManager.GetFileSystemDiskTargetPathFromHostView(virtLauncherUID, volumeName, false)
	if err != nil {
		return false, err
	}
	return isolation.IsMounted(target)
}

func NewHotplugMounter(isoDetector isolation.PodIsolationDetector,
	mountStateDir string,
	clusterConfig *virtconfig.ClusterConfig,
	hotplugManager hotplugdisk.HotplugDiskManagerInterface,
) HotplugMounter {
	return &hotplugMounter{
		mountRecords:           make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector:   isoDetector,
		mountStateDir:          mountStateDir,
		suppressWarningTimeout: 1 * time.Minute,
		clusterConfig:          clusterConfig,
		nodeIsolationResult:    isolation.NodeIsolationResult(),

		hotplugPathGetter: containerdisk.NewHotplugSocketPathGetter(""),
		hotplugManager:    hotplugManager,
	}
}

func (m *hotplugMounter) deleteMountTargetRecord(vmi *v1.VirtualMachineInstance) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find container disk mounted directories for vmi without uid")
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
			os.Remove(target.SocketFile)
		}

		os.Remove(recordFile)
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	delete(m.mountRecords, vmi.UID)

	return nil
}

func (m *hotplugMounter) getMountTargetRecord(vmi *v1.VirtualMachineInstance) (*vmiMountTargetRecord, error) {
	var ok bool
	var existingRecord *vmiMountTargetRecord

	if string(vmi.UID) == "" {
		return nil, fmt.Errorf("unable to find container disk mounted directories for vmi without uid")
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	existingRecord, ok = m.mountRecords[vmi.UID]

	// first check memory cache
	if ok {
		return existingRecord, nil
	}

	// if not there, see if record is on disk, this can happen if virt-handler restarts
	recordFile := filepath.Join(m.mountStateDir, filepath.Clean(string(vmi.UID)))

	exists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return nil, err
	}

	if exists {
		record := vmiMountTargetRecord{}
		// #nosec No risk for path injection. Using static base and cleaned filename
		bytes, err := os.ReadFile(recordFile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bytes, &record)
		if err != nil {
			return nil, err
		}

		if !record.UsesSafePaths {
			record.UsesSafePaths = true
			for i, entry := range record.MountTargetEntries {
				safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", entry.TargetFile)
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
	return nil, nil
}

func (m *hotplugMounter) addMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	return m.setAddMountTargetRecordHelper(vmi, record, true)
}

func (m *hotplugMounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	return m.setAddMountTargetRecordHelper(vmi, record, false)
}

func (m *hotplugMounter) setAddMountTargetRecordHelper(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord, addPreviousRules bool) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to set container disk mounted directories for vmi without uid")
	}

	record.UsesSafePaths = true

	recordFile := filepath.Join(m.mountStateDir, string(vmi.UID))
	fileExists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return err
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()

	existingRecord, ok := m.mountRecords[vmi.UID]
	if ok && fileExists && equality.Semantic.DeepEqual(existingRecord, record) {
		// already done
		return nil
	}

	if addPreviousRules && existingRecord != nil && len(existingRecord.MountTargetEntries) > 0 {
		record.MountTargetEntries = append(record.MountTargetEntries, existingRecord.MountTargetEntries...)
	}

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

func (m *hotplugMounter) MoundAndVerifyFromPod(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (map[string]*containerdisk.DiskInfo, error) {
	return m.mountAndVerify(vmi, sourceUID)
}

func (m *hotplugMounter) MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error) {
	return m.mountAndVerify(vmi, "")
}

func (m *hotplugMounter) mountAndVerify(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (map[string]*containerdisk.DiskInfo, error) {
	virtLauncherUID := m.findVirtlauncherUID(vmi)
	if virtLauncherUID == "" {
		return nil, nil
	}

	record := vmiMountTargetRecord{}
	disksInfo := map[string]*containerdisk.DiskInfo{}

	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.Hotpluggable {
			entry, err := m.newMountTargetEntry(vmi, virtLauncherUID, sourceUID, volume.Name)
			if err != nil {
				return nil, err
			}
			record.MountTargetEntries = append(record.MountTargetEntries, entry)
		}
	}

	if len(record.MountTargetEntries) > 0 {
		err := m.setMountTargetRecord(vmi, &record)
		if err != nil {
			return nil, err
		}
	}

	vmiRes, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return nil, fmt.Errorf("failed to detect VMI pod: %v", err)
	}

	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.Hotpluggable {
			target, err := m.hotplugManager.GetFileSystemDiskTargetPathFromHostView(virtLauncherUID, volume.Name, false)

			if isMounted, err := isolation.IsMounted(target); err != nil {
				return nil, fmt.Errorf("failed to determine if %s is already mounted: %v", target, err)
			} else if !isMounted {

				sourceFile, err := m.getContainerDiskPath(vmi, &volume, volume.Name, sourceUID)
				if err != nil {
					return nil, fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}

				log.DefaultLogger().Object(vmi).Infof("Bind mounting container disk at %s to %s", sourceFile, target)
				opts := []string{
					"bind", "ro", "uid=107", "gid=107",
				}
				err = virt_chroot.MountChrootWithOptions(sourceFile, target, opts...)
				if err != nil {
					return nil, fmt.Errorf("failed to bindmount containerDisk %v. err: %w", volume.Name, err)
				}
			}

			imageInfo, err := isolation.GetImageInfo(
				containerdisk.GetHotplugContainerDiskTargetPathFromLauncherView(volume.Name),
				vmiRes,
				m.clusterConfig.GetDiskVerification(),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get image info: %v", err)
			}
			if err := containerdisk.VerifyImage(imageInfo); err != nil {
				return nil, fmt.Errorf("invalid image in containerDisk %v: %v", volume.Name, err)
			}
			disksInfo[volume.Name] = imageInfo
		}
	}

	return disksInfo, nil
}

func (m *hotplugMounter) newMountTargetEntry(
	vmi *v1.VirtualMachineInstance,
	virtLauncherUID, sourceUID types.UID,
	volumeName string,
) (vmiMountTargetEntry, error) {
	targetFile, err := m.getTarget(virtLauncherUID, volumeName)
	if err != nil {
		return vmiMountTargetEntry{}, err
	}

	sock, err := m.hotplugPathGetter(vmi, volumeName, sourceUID)
	if err != nil {
		return vmiMountTargetEntry{}, err
	}
	return vmiMountTargetEntry{
		TargetFile: targetFile,
		SocketFile: sock,
	}, nil
}

func (m *hotplugMounter) getTarget(virtLauncherUID types.UID, volumeName string) (string, error) {
	target, err := m.hotplugManager.GetFileSystemDiskTargetPathFromHostView(virtLauncherUID, volumeName, true)
	if err != nil {
		return "", err
	}
	return unsafepath.UnsafeAbsolute(target.Raw()), nil
}

func (m *hotplugMounter) getMountedVolumesInWorld(vmi *v1.VirtualMachineInstance, virtLauncherUID types.UID) ([]vmiMountTargetEntry, error) {
	if vmi == nil || virtLauncherUID == "" {
		return nil, nil
	}
	path, err := m.hotplugManager.GetHotplugTargetPodPathOnHost(virtLauncherUID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	rawPath := unsafepath.UnsafeAbsolute(path.Raw())
	entries, err := os.ReadDir(rawPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var volumes []string
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".img") {
			name := strings.TrimSuffix(entry.Name(), ".img")
			volumes = append(volumes, name)
		}
	}
	var mountedVolumes []vmiMountTargetEntry
	for _, v := range volumes {
		mounted, err := m.IsMounted(vmi, v)
		if err != nil {
			return nil, err
		}
		if mounted {
			target, err := m.getTarget(virtLauncherUID, v)
			if err != nil {
				return nil, err
			}
			mountedVolumes = append(mountedVolumes, vmiMountTargetEntry{
				TargetFile: target,
			})
		}
	}
	return mountedVolumes, nil
}

func (m *hotplugMounter) mergeMountEntries(r1, r2 []vmiMountTargetEntry) []vmiMountTargetEntry {
	targetSocket := make(map[string]string)

	sortTargetSocket := func(r []vmiMountTargetEntry) {
		for _, entry := range r {
			socket := targetSocket[entry.TargetFile]
			if socket == "" {
				targetSocket[entry.TargetFile] = entry.SocketFile
			}
		}
	}
	sortTargetSocket(r1)
	sortTargetSocket(r2)

	newRecords := make([]vmiMountTargetEntry, len(targetSocket))
	count := 0
	for t, s := range targetSocket {
		newRecords[count] = vmiMountTargetEntry{
			TargetFile: t,
			SocketFile: s,
		}
		count++
	}

	return newRecords
}

func (m *hotplugMounter) Umount(vmi *v1.VirtualMachineInstance) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	worldEntries, err := m.getMountedVolumesInWorld(vmi, m.findVirtlauncherUID(vmi))
	if err != nil {
		return fmt.Errorf("failed to get world entries: %w", err)
	}
	var recordMountTargetEntries []vmiMountTargetEntry
	if record != nil {
		recordMountTargetEntries = record.MountTargetEntries
	}

	mountEntries := m.mergeMountEntries(recordMountTargetEntries, worldEntries)

	if len(mountEntries) == 0 {
		log.DefaultLogger().Object(vmi).Infof("No container disk mount entries found to unmount")
		return nil
	}

	entriesTargetForDelete := make(map[string]struct{})

	for _, r := range mountEntries {
		name := extractNameFromTarget(r.TargetFile)

		needUmount := true
		for _, v := range vmi.Status.VolumeStatus {
			if v.Name == name {
				needUmount = false
			}
		}
		if needUmount {
			file, err := safepath.NewFileNoFollow(r.TargetFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					entriesTargetForDelete[r.TargetFile] = struct{}{}
					continue
				}
				return fmt.Errorf(failedCheckMountPointFmt, r.TargetFile, err)
			}
			_ = file.Close()
			mounted, err := m.IsMounted(vmi, name)
			if err != nil {
				return fmt.Errorf(failedCheckMountPointFmt, r.TargetFile, err)
			}
			if !mounted {
				if err = safepath.UnlinkAtNoFollow(file.Path()); err != nil {
					return fmt.Errorf("failed to delete file %s: %w", file.Path(), err)
				}
				entriesTargetForDelete[r.TargetFile] = struct{}{}
				continue
			}
			// #nosec No risk for attacket injection. Parameters are predefined strings
			out, err := virt_chroot.UmountChroot(file.Path()).CombinedOutput()
			if err != nil {
				return fmt.Errorf(failedUnmountFmt, file, string(out), err)
			}
			if err = safepath.UnlinkAtNoFollow(file.Path()); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", file.Path(), err)
			}
			entriesTargetForDelete[r.TargetFile] = struct{}{}
		}
	}
	newEntries := make([]vmiMountTargetEntry, 0, len(recordMountTargetEntries)-len(entriesTargetForDelete))
	for _, entry := range recordMountTargetEntries {
		if _, found := entriesTargetForDelete[entry.TargetFile]; found {
			continue
		}
		newEntries = append(newEntries, entry)
	}
	record.MountTargetEntries = newEntries
	return m.setMountTargetRecord(vmi, record)
}

func extractNameFromSocket(socketFile string) (string, error) {
	base := filepath.Base(socketFile)
	if strings.HasPrefix(base, "hotplug-container-disk-") && strings.HasSuffix(base, ".sock") {
		name := strings.TrimPrefix(base, "hotplug-container-disk-")
		name = strings.TrimSuffix(name, ".sock")
		return name, nil
	}
	return "", fmt.Errorf("name not found in path")
}

func extractNameFromTarget(targetFile string) string {
	filename := filepath.Base(targetFile)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name
}

func (m *hotplugMounter) UmountAll(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID == "" {
		return nil
	}

	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	worldEntries, err := m.getMountedVolumesInWorld(vmi, m.findVirtlauncherUID(vmi))
	if err != nil {
		return fmt.Errorf("failed to get world entries: %w", err)
	}
	var recordMountTargetEntries []vmiMountTargetEntry
	if record != nil {
		recordMountTargetEntries = record.MountTargetEntries
	}

	mountEntries := m.mergeMountEntries(recordMountTargetEntries, worldEntries)

	if len(mountEntries) == 0 {
		log.DefaultLogger().Object(vmi).Infof("No container disk mount entries found to unmount")
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("Found container disk mount entries")

	for _, entry := range mountEntries {
		log.DefaultLogger().Object(vmi).Infof("Looking to see if containerdisk is mounted at path %s", entry.TargetFile)
		file, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf(failedCheckMountPointFmt, entry.TargetFile, err)
		}
		_ = file.Close()
		if mounted, err := isolation.IsMounted(file.Path()); err != nil {
			return fmt.Errorf(failedCheckMountPointFmt, file, err)
		} else if mounted {
			log.DefaultLogger().Object(vmi).Infof("unmounting container disk at path %s", file)
			// #nosec No risk for attacket injection. Parameters are predefined strings
			out, err := virt_chroot.UmountChroot(file.Path()).CombinedOutput()
			if err != nil {
				return fmt.Errorf(failedUnmountFmt, file, string(out), err)
			}
			if err = safepath.UnlinkAtNoFollow(file.Path()); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", file.Path(), err)
			}
		} else {
			if err = safepath.UnlinkAtNoFollow(file.Path()); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", file.Path(), err)
			}
		}
	}
	err = m.deleteMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	return nil
}

func (m *hotplugMounter) ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time, sourceUID types.UID) (bool, error) {
	for _, volume := range vmi.Spec.Volumes {
		if containerdisk.IsHotplugContainerDisk(&volume) {
			_, err := m.hotplugPathGetter(vmi, volume.Name, sourceUID)
			if err != nil {
				log.DefaultLogger().Object(vmi).Reason(err).Infof("hotplug containerdisk %s not yet ready", volume.Name)
				if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
					return false, fmt.Errorf("hotplug containerdisk %s still not ready after one minute", volume.Name)
				}
				return false, nil
			}
		}
	}

	log.DefaultLogger().Object(vmi).V(4).Info("all hotplug containerdisks are ready")
	return true, nil
}

func (m *hotplugMounter) getContainerDiskPath(vmi *v1.VirtualMachineInstance, volume *v1.Volume, volumeName string, sourceUID types.UID) (*safepath.Path, error) {
	sock, err := m.hotplugPathGetter(vmi, volumeName, sourceUID)
	if err != nil {
		return nil, ErrDiskContainerGone
	}

	res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
	if err != nil {
		return nil, fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
	}

	mountPoint, err := isolation.ParentPathForRootMount(m.nodeIsolationResult, res)
	if err != nil {
		return nil, fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
	}

	return containerdisk.GetImage(mountPoint, volume.ContainerDisk.Path)
}

func (m *hotplugMounter) ComputeChecksums(vmi *v1.VirtualMachineInstance, sourceUID types.UID) (*DiskChecksums, error) {

	diskChecksums := &DiskChecksums{
		ContainerDiskChecksums: map[string]uint32{},
	}

	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk == nil || !volume.VolumeSource.ContainerDisk.Hotpluggable {
			continue
		}

		path, err := m.getContainerDiskPath(vmi, &volume, volume.Name, sourceUID)
		if err != nil {
			return nil, err
		}

		checksum, err := getDigest(path)
		if err != nil {
			return nil, err
		}

		diskChecksums.ContainerDiskChecksums[volume.Name] = checksum
	}

	return diskChecksums, nil
}

func (m *hotplugMounter) findVirtlauncherUID(vmi *v1.VirtualMachineInstance) (uid types.UID) {
	cnt := 0
	for podUID := range vmi.Status.ActivePods {
		_, err := m.hotplugManager.GetHotplugTargetPodPathOnHost(podUID)
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

func GetMigrationAttachmentPodUID(vmi *v1.VirtualMachineInstance) (types.UID, bool) {
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != types.UID("") {
		return attachmentPodUID, true
	}
	return types.UID(""), false
}

func VerifyHotplugChecksums(mounter HotplugMounter, vmi *v1.VirtualMachineInstance, sourceUID types.UID) error {
	diskChecksums, err := mounter.ComputeChecksums(vmi, sourceUID)
	if err != nil {
		return fmt.Errorf("failed to compute hotplug container disk checksums: %s", err)
	}

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.ContainerDiskVolume == nil || volumeStatus.HotplugVolume == nil {
			continue
		}

		expectedChecksum := volumeStatus.ContainerDiskVolume.Checksum
		computedChecksum := diskChecksums.ContainerDiskChecksums[volumeStatus.Name]
		if err := compareChecksums(expectedChecksum, computedChecksum); err != nil {
			return fmt.Errorf("checksum error for hotplug volume %s: %w", volumeStatus.Name, err)
		}
	}
	return nil
}
