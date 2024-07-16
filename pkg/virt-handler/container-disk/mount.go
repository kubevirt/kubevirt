package container_disk

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	"kubevirt.io/kubevirt/pkg/safepath"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"kubevirt.io/client-go/log"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

const (
	failedCheckMountPointFmt = "failed to check mount point for containerDisk %v: %v"
	failedUnmountFmt         = "failed to unmount containerDisk %v: %v : %v"
)

var (
	ErrChecksumMissing   = errors.New("missing checksum")
	ErrChecksumMismatch  = errors.New("checksum mismatch")
	ErrDiskContainerGone = errors.New("disk container is gone")
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector       isolation.PodIsolationDetector
	checkpointManager          checkpoint.CheckpointManager
	mountRecords               map[types.UID]*vmiMountTargetRecord
	mountRecordsLock           sync.Mutex
	suppressWarningTimeout     time.Duration
	socketPathGetter           containerdisk.SocketPathGetter
	kernelBootSocketPathGetter containerdisk.KernelBootSocketPathGetter
	clusterConfig              *virtconfig.ClusterConfig
	nodeIsolationResult        isolation.IsolationResult
}

type Mounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error)
	Unmount(vmi *v1.VirtualMachineInstance) error
	ComputeChecksums(vmi *v1.VirtualMachineInstance) (*DiskChecksums, error)
}

type vmiMountTargetEntry struct {
	TargetFile string `json:"targetFile"`
	SocketFile string `json:"socketFile"`
}

type vmiMountTargetRecord struct {
	MountTargetEntries []vmiMountTargetEntry `json:"mountTargetEntries"`
	UsesSafePaths      bool                  `json:"usesSafePaths"`
}

type DiskChecksums struct {
	KernelBootChecksum     KernelBootChecksum
	ContainerDiskChecksums map[string]uint32
}

func NewMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string, clusterConfig *virtconfig.ClusterConfig) Mounter {
	return &mounter{
		mountRecords:               make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector:       isoDetector,
		checkpointManager:          checkpoint.NewSimpleCheckpointManager(mountStateDir),
		suppressWarningTimeout:     1 * time.Minute,
		socketPathGetter:           containerdisk.NewSocketPathGetter(""),
		kernelBootSocketPathGetter: containerdisk.NewKernelBootSocketPathGetter(""),
		clusterConfig:              clusterConfig,
		nodeIsolationResult:        isolation.NodeIsolationResult(),
	}
}

func (m *mounter) deleteMountTargetRecord(vmi *v1.VirtualMachineInstance) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find container disk mounted directories for vmi without uid")
	}

	record := vmiMountTargetRecord{}
	err := m.checkpointManager.Get(string(vmi.UID), &record)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to get a checkpoint %s, %w", vmi.UID, err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		for _, target := range record.MountTargetEntries {
			os.Remove(target.TargetFile)
			os.Remove(target.SocketFile)
		}

		if err := m.checkpointManager.Delete(string(vmi.UID)); err != nil {
			return fmt.Errorf("failed to delete checkpoint %s, %w", vmi.UID, err)
		}
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	delete(m.mountRecords, vmi.UID)

	return nil
}

func (m *mounter) getMountTargetRecord(vmi *v1.VirtualMachineInstance) (*vmiMountTargetRecord, error) {
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
	record := vmiMountTargetRecord{}
	err := m.checkpointManager.Get(string(vmi.UID), &record)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to get checkpoint %s, %w", vmi.UID, err)
	}

	if err == nil {
		// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
		// After a one-time convert and persist, old records are safe too.
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

func (m *mounter) addMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	return m.setAddMountTargetRecordHelper(vmi, record, true)
}

func (m *mounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	return m.setAddMountTargetRecordHelper(vmi, record, false)
}

func (m *mounter) setAddMountTargetRecordHelper(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord, addPreviousRules bool) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to set container disk mounted directories for vmi without uid")
	}
	// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
	// After a one-time convert and persist, old records are safe too.
	record.UsesSafePaths = true

	err := m.checkpointManager.Get(string(vmi.UID), &vmiMountTargetRecord{})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to get checkpoint %s, %w", vmi.UID, err)
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()

	existingRecord, ok := m.mountRecords[vmi.UID]
	if ok && !errors.Is(err, os.ErrNotExist) && equality.Semantic.DeepEqual(existingRecord, record) {
		// already done
		return nil
	}

	if addPreviousRules && existingRecord != nil && len(existingRecord.MountTargetEntries) > 0 {
		record.MountTargetEntries = append(record.MountTargetEntries, existingRecord.MountTargetEntries...)
	}

	if err := m.checkpointManager.Store(string(vmi.UID), record); err != nil {
		return fmt.Errorf("failed to checkpoint %s, %w", vmi.UID, err)
	}

	m.mountRecords[vmi.UID] = record

	return nil
}

// Mount takes a vmi and mounts all container disks of the VMI, so that they are visible for the qemu process.
// Additionally qcow2 images are validated if "verify" is true. The validation happens with rlimits set, to avoid DOS.
func (m *mounter) MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error) {
	record := vmiMountTargetRecord{}
	disksInfo := map[string]*containerdisk.DiskInfo{}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return nil, err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			// If diskName is a symlink it will fail if the target exists.
			if err := safepath.TouchAtNoFollow(diskTargetDir, diskName, os.ModePerm); err != nil {
				if err != nil && !os.IsExist(err) {
					return nil, fmt.Errorf("failed to create mount point target: %v", err)
				}
			}
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return nil, err
			}

			sock, err := m.socketPathGetter(vmi, i)
			if err != nil {
				return nil, err
			}

			record.MountTargetEntries = append(record.MountTargetEntries, vmiMountTargetEntry{
				TargetFile: unsafepath.UnsafeAbsolute(targetFile.Raw()),
				SocketFile: sock,
			})
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

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return nil, err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return nil, err
			}

			if isMounted, err := isolation.IsMounted(targetFile); err != nil {
				return nil, fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {

				sourceFile, err := m.getContainerDiskPath(vmi, &volume, i)
				if err != nil {
					return nil, fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}

				log.DefaultLogger().Object(vmi).Infof("Bind mounting container disk at %s to %s", sourceFile, targetFile)
				out, err := virt_chroot.MountChroot(sourceFile, targetFile, true).CombinedOutput()
				if err != nil {
					return nil, fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}

			imageInfo, err := isolation.GetImageInfo(containerdisk.GetDiskTargetPathFromLauncherView(i), vmiRes, m.clusterConfig.GetDiskVerification())
			if err != nil {
				return nil, fmt.Errorf("failed to get image info: %v", err)
			}
			if err := containerdisk.VerifyImage(imageInfo); err != nil {
				return nil, fmt.Errorf("invalid image in containerDisk %v: %v", volume.Name, err)
			}
			disksInfo[volume.Name] = imageInfo
		}
	}
	err = m.mountKernelArtifacts(vmi, true)
	if err != nil {
		return nil, fmt.Errorf("error mounting kernel artifacts: %v", err)
	}

	return disksInfo, nil
}

// Unmount unmounts all container disks of a given VMI.
func (m *mounter) Unmount(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID == "" {
		return nil
	}

	err := m.unmountKernelArtifacts(vmi)
	if err != nil {
		return fmt.Errorf("error unmounting kernel artifacts: %v", err)
	}

	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	} else if record == nil {
		// no entries to unmount

		log.DefaultLogger().Object(vmi).Infof("No container disk mount entries found to unmount")
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("Found container disk mount entries")
	for _, entry := range record.MountTargetEntries {
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
		}
	}
	err = m.deleteMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	return nil
}

func (m *mounter) ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error) {
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			_, err := m.socketPathGetter(vmi, i)
			if err != nil {
				log.DefaultLogger().Object(vmi).Reason(err).Infof("containerdisk %s not yet ready", volume.Name)
				if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
					return false, fmt.Errorf("containerdisk %s still not ready after one minute", volume.Name)
				}
				return false, nil
			}
		}
	}

	if !m.kernelBootDisksReady(vmi) {
		if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
			return false, fmt.Errorf("kernelboot container still not ready after one minute")
		}
		return false, nil
	}

	log.DefaultLogger().Object(vmi).V(4).Info("all containerdisks are ready")
	return true, nil
}

func (m *mounter) getContainerDiskPath(vmi *v1.VirtualMachineInstance, volume *v1.Volume, volumeIndex int) (*safepath.Path, error) {
	sock, err := m.socketPathGetter(vmi, volumeIndex)
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

func getDigest(imageFile *safepath.Path) (uint32, error) {
	digest := crc32.NewIEEE()

	err := imageFile.ExecuteNoFollow(func(path string) (err error) {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		// 32 MiB chunks
		chunk := make([]byte, 1024*1024*32)

		_, err = io.CopyBuffer(digest, f, chunk)
		return err
	})

	return digest.Sum32(), err
}

func (m *mounter) ComputeChecksums(vmi *v1.VirtualMachineInstance) (*DiskChecksums, error) {

	diskChecksums := &DiskChecksums{
		ContainerDiskChecksums: map[string]uint32{},
	}

	// compute for containerdisks
	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk == nil {
			continue
		}

		path, err := m.getContainerDiskPath(vmi, &volume, i)
		if err != nil {
			return nil, err
		}

		checksum, err := getDigest(path)
		if err != nil {
			return nil, err
		}

		diskChecksums.ContainerDiskChecksums[volume.Name] = checksum
	}

	var err error
	// kernel and initrd
	diskChecksums.KernelBootChecksum.Kernel, diskChecksums.KernelBootChecksum.Initrd, err = m.kernelBootComputeChecksums(vmi)
	if err != nil {
		return nil, err
	}

	return diskChecksums, nil
}

func compareChecksums(expectedChecksum, computedChecksum uint32) error {
	if expectedChecksum == 0 {
		return ErrChecksumMissing
	}
	if expectedChecksum != computedChecksum {
		return ErrChecksumMismatch
	}
	// checksum ok
	return nil
}

func VerifyChecksums(mounter Mounter, vmi *v1.VirtualMachineInstance) error {
	diskChecksums, err := mounter.ComputeChecksums(vmi)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %s", err)
	}

	// verify containerdisks
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.ContainerDiskVolume == nil {
			continue
		}

		expectedChecksum := volumeStatus.ContainerDiskVolume.Checksum
		computedChecksum := diskChecksums.ContainerDiskChecksums[volumeStatus.Name]
		if err := compareChecksums(expectedChecksum, computedChecksum); err != nil {
			return fmt.Errorf("checksum error for volume %s: %w", volumeStatus.Name, err)
		}
	}

	// verify kernel and initrd
	if err := kernelBootVerifyChecksums(vmi, diskChecksums); err != nil {
		return err
	}

	return nil
}
