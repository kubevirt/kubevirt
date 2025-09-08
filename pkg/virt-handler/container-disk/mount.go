package container_disk

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

const (
	failedCheckMountPointFmt = "failed to check mount point for containerDisk %v: %v"
	failedUnmountFmt         = "failed to unmount containerDisk %v: %v : %v"
)

var (
	ErrWaitingForDisks   = errors.New("waiting for containerdisks")
	ErrDiskContainerGone = errors.New("disk container is gone")
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector       isolation.PodIsolationDetector
	checkpointManager          checkpoint.CheckpointManager
	mountRecords               map[types.UID]*vmiMountTargetRecord
	mountRecordsLock           sync.Mutex
	suppressWarningTimeout     time.Duration
	needsBindMountFunc         needsBindMountFunc
	socketPathGetter           containerdisk.SocketPathGetter
	kernelBootSocketPathGetter containerdisk.KernelBootSocketPathGetter
	clusterConfig              *virtconfig.ClusterConfig
	nodeIsolationResult        isolation.IsolationResult
}

type Mounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) error
	Unmount(vmi *v1.VirtualMachineInstance) error
	// ComputeChecksums method, along with the code added in this commit, can be removed after the 1.7 release.
	// By then, we can be sure that during upgrades older versions of virt-handler no longer expect the checksum
	// in the VMI status.
	// Therefore, it will no longer be necessary to include this information in the VMI status.
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

type kernelArtifacts struct {
	kernel *safepath.Path
	initrd *safepath.Path
}

type DiskChecksums struct {
	KernelBootChecksum     KernelBootChecksum
	ContainerDiskChecksums map[string]uint32
}

type KernelBootChecksum struct {
	Initrd *uint32
	Kernel *uint32
}

func NewMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string, clusterConfig *virtconfig.ClusterConfig) Mounter {
	return &mounter{
		mountRecords:               make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector:       isoDetector,
		checkpointManager:          checkpoint.NewSimpleCheckpointManager(mountStateDir),
		suppressWarningTimeout:     1 * time.Minute,
		needsBindMountFunc:         newNeedsBindMountFunc(""),
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
func (m *mounter) MountAndVerify(vmi *v1.VirtualMachineInstance) error {
	if m.clusterConfig.ImageVolumeEnabled() {
		bindMountNeeded, err := m.needsBindMountFunc(vmi)
		if err != nil {
			return fmt.Errorf("fail to detect if bind mount needed for vmi: %s in namespace: %v. err: %v", vmi.Name, vmi.Namespace, err)
		}
		if !bindMountNeeded {
			return nil
		}
	}

	record := vmiMountTargetRecord{}
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			// If diskName is a symlink it will fail if the target exists.
			if err := safepath.TouchAtNoFollow(diskTargetDir, diskName, os.ModePerm); err != nil {
				if !os.IsExist(err) {
					return fmt.Errorf("failed to create mount point target: %v", err)
				}
			}
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return err
			}

			sock, err := m.socketPathGetter(vmi, i)
			if err != nil {
				return err
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
			return err
		}
	}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskTargetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
			if err != nil {
				return err
			}
			diskName := containerdisk.GetDiskTargetName(i)
			targetFile, err := safepath.JoinNoFollow(diskTargetDir, diskName)
			if err != nil {
				return err
			}

			if isMounted, err := isolation.IsMounted(targetFile); err != nil {
				return fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {

				sourceFile, err := m.getContainerDiskPath(vmi, &volume, i)
				if err != nil {
					return fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}

				log.DefaultLogger().Object(vmi).Infof("Bind mounting container disk at %s to %s", sourceFile, targetFile)
				out, err := virt_chroot.MountChroot(sourceFile, targetFile, true).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}
		}
	}
	err := m.mountKernelArtifacts(vmi, true)
	if err != nil {
		return fmt.Errorf("error mounting kernel artifacts: %v", err)
	}

	return nil
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
			// #nosec No risk for attacker injection. Parameters are predefined strings
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
	if m.clusterConfig.ImageVolumeEnabled() {
		bindMountNeeded, err := m.needsBindMountFunc(vmi)
		if err != nil {
			return false, fmt.Errorf("fail to detect if bind mount needed for vmi: %s in namespace: %v. err: %v", vmi.Name, vmi.Namespace, err)
		}
		if !bindMountNeeded {
			return true, nil
		}
	}
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			sock, err := m.socketPathGetter(vmi, i)
			if err == nil {
				_, err = m.podIsolationDetector.DetectForSocket(vmi, sock)
			}

			if err != nil {
				log.DefaultLogger().Object(vmi).Reason(err).Infof("containerdisk %s not yet ready", volume.Name)
				if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
					return false, fmt.Errorf("containerdisk %s still not ready after one minute", volume.Name)
				}
				return false, nil
			}

		}
	}

	if util.HasKernelBootContainerImage(vmi) {
		sock, err := m.kernelBootSocketPathGetter(vmi)
		if err == nil {
			_, err = m.podIsolationDetector.DetectForSocket(vmi, sock)
		}
		if err != nil {
			log.DefaultLogger().Object(vmi).Reason(err).Info("kernelboot container not yet ready")
			if time.Now().After(notInitializedSince.Add(m.suppressWarningTimeout)) {
				return false, fmt.Errorf("kernelboot container still not ready after one minute")
			}
			return false, nil
		}
	}

	log.DefaultLogger().Object(vmi).V(4).Info("all containerdisks are ready")
	return true, nil
}

// MountKernelArtifacts mounts artifacts defined by KernelBootName in VMI.
// This function is assumed to run after MountAndVerify.
func (m *mounter) mountKernelArtifacts(vmi *v1.VirtualMachineInstance, verify bool) error {
	const kernelBootName = containerdisk.KernelBootName

	log.Log.Object(vmi).Infof("mounting kernel artifacts")

	if !util.HasKernelBootContainerImage(vmi) {
		log.Log.Object(vmi).Infof("kernel boot not defined - nothing to mount")
		return nil
	}

	kb := vmi.Spec.Domain.Firmware.KernelBoot.Container

	targetDir, err := containerdisk.GetDiskTargetDirFromHostView(vmi)
	if err != nil {
		return fmt.Errorf("failed to get disk target dir: %v", err)
	}
	if err := safepath.MkdirAtNoFollow(targetDir, containerdisk.KernelBootName, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	targetDir, err = safepath.JoinNoFollow(targetDir, containerdisk.KernelBootName)
	if err != nil {
		return err
	}
	if err := safepath.ChpermAtNoFollow(targetDir, 0, 0, 0755); err != nil {
		return err
	}

	socketFilePath, err := m.kernelBootSocketPathGetter(vmi)
	if err != nil {
		return fmt.Errorf("failed to find socket path for kernel artifacts: %v", err)
	}

	record := vmiMountTargetRecord{
		MountTargetEntries: []vmiMountTargetEntry{{
			TargetFile: unsafepath.UnsafeAbsolute(targetDir.Raw()),
			SocketFile: socketFilePath,
		}},
	}

	err = m.addMountTargetRecord(vmi, &record)
	if err != nil {
		return err
	}

	var targetInitrdPath *safepath.Path
	var targetKernelPath *safepath.Path

	if kb.InitrdPath != "" {
		if err := safepath.TouchAtNoFollow(targetDir, filepath.Base(kb.InitrdPath), 0655); err != nil && !os.IsExist(err) {
			return err
		}

		targetInitrdPath, err = safepath.JoinNoFollow(targetDir, filepath.Base(kb.InitrdPath))
		if err != nil {
			return err
		}
	}

	if kb.KernelPath != "" {
		if err := safepath.TouchAtNoFollow(targetDir, filepath.Base(kb.KernelPath), 0655); err != nil && !os.IsExist(err) {
			return err
		}

		targetKernelPath, err = safepath.JoinNoFollow(targetDir, filepath.Base(kb.KernelPath))
		if err != nil {
			return err
		}
	}

	areKernelArtifactsMounted := func(artifactsDir *safepath.Path, artifactFiles ...*safepath.Path) (bool, error) {
		if _, err = safepath.StatAtNoFollow(artifactsDir); errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		for _, mountPoint := range artifactFiles {
			if mountPoint != nil {
				isMounted, err := isolation.IsMounted(mountPoint)
				if !isMounted || err != nil {
					return isMounted, err
				}
			}
		}
		return true, nil
	}

	if isMounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetDir, err)
	} else if !isMounted {
		log.Log.Object(vmi).Infof("kernel artifacts are not mounted - mounting...")

		kernelArtifacts, err := m.getKernelArtifactPaths(vmi)
		if err != nil {
			return err
		}

		if kernelArtifacts.kernel != nil {
			out, err := virt_chroot.MountChroot(kernelArtifacts.kernel, targetKernelPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}
		}

		if kernelArtifacts.initrd != nil {
			out, err := virt_chroot.MountChroot(kernelArtifacts.initrd, targetInitrdPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}
		}

	}

	if verify {
		mounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath)
		if err != nil {
			return fmt.Errorf("failed to check if kernel artifacts are mounted. error: %v", err)
		} else if !mounted {
			return fmt.Errorf("kernel artifacts verification failed")
		}
	}

	return nil
}

func (m *mounter) unmountKernelArtifacts(vmi *v1.VirtualMachineInstance) error {
	if !util.HasKernelBootContainerImage(vmi) {
		return nil
	}

	log.DefaultLogger().Object(vmi).Infof("unmounting kernel artifacts")

	kb := vmi.Spec.Domain.Firmware.KernelBoot.Container

	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return fmt.Errorf("failed to get mount target record: %v", err)
	} else if record == nil {
		log.DefaultLogger().Object(vmi).Warning("Cannot find kernel-boot entries to unmount")
		return nil
	}

	unmount := func(targetDir *safepath.Path, artifactPaths ...string) error {
		for _, artifactPath := range artifactPaths {
			if artifactPath == "" {
				continue
			}

			targetPath, err := safepath.JoinNoFollow(targetDir, filepath.Base(artifactPath))
			if err != nil {
				return fmt.Errorf(failedCheckMountPointFmt, targetPath, err)
			}
			if mounted, err := isolation.IsMounted(targetPath); err != nil {
				return fmt.Errorf(failedCheckMountPointFmt, targetPath, err)
			} else if mounted {
				log.DefaultLogger().Object(vmi).Infof("unmounting container disk at targetDir %s", targetPath)

				out, err := virt_chroot.UmountChroot(targetPath).CombinedOutput()
				if err != nil {
					return fmt.Errorf(failedUnmountFmt, targetPath, string(out), err)
				}
			}
		}
		return nil
	}

	for idx, entry := range record.MountTargetEntries {
		if !strings.Contains(entry.TargetFile, containerdisk.KernelBootName) {
			continue
		}
		targetDir, err := safepath.NewFileNoFollow(entry.TargetFile)
		if err != nil {
			return fmt.Errorf("failed to obtaining a reference to the target directory %q: %v", targetDir, err)
		}
		_ = targetDir.Close()
		log.DefaultLogger().Object(vmi).Infof("unmounting kernel artifacts in path: %v", targetDir)

		if err = unmount(targetDir.Path(), kb.InitrdPath, kb.KernelPath); err != nil {
			// Not returning here since even if unmount wasn't successful it's better to keep
			// cleaning the mounted files.
			log.Log.Object(vmi).Reason(err).Error("unable to unmount kernel artifacts")
		}

		removeSliceElement := func(s []vmiMountTargetEntry, idxToRemove int) []vmiMountTargetEntry {
			// removes slice element efficiently
			s[idxToRemove] = s[len(s)-1]
			return s[:len(s)-1]
		}

		record.MountTargetEntries = removeSliceElement(record.MountTargetEntries, idx)
		return nil
	}

	return fmt.Errorf("kernel artifacts record wasn't found")
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

func (m *mounter) getKernelArtifactPaths(vmi *v1.VirtualMachineInstance) (*kernelArtifacts, error) {
	sock, err := m.kernelBootSocketPathGetter(vmi)
	if err != nil {
		return nil, ErrDiskContainerGone
	}

	res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
	if err != nil {
		return nil, fmt.Errorf("failed to detect socket for kernelboot container: %v", err)
	}

	mountPoint, err := isolation.ParentPathForRootMount(m.nodeIsolationResult, res)
	if err != nil {
		return nil, fmt.Errorf("failed to detect root mount point of kernel/initrd container on the node: %v", err)
	}

	kernelContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container
	kernelArtifacts := &kernelArtifacts{}

	if kernelContainer.KernelPath != "" {
		kernelPath, err := containerdisk.GetImage(mountPoint, kernelContainer.KernelPath)
		if err != nil {
			return nil, err
		}
		kernelArtifacts.kernel = kernelPath
	}
	if kernelContainer.InitrdPath != "" {
		initrdPath, err := containerdisk.GetImage(mountPoint, kernelContainer.InitrdPath)
		if err != nil {
			return nil, err
		}
		kernelArtifacts.initrd = initrdPath
	}

	return kernelArtifacts, nil
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

	// kernel and initrd
	if util.HasKernelBootContainerImage(vmi) {
		kernelArtifacts, err := m.getKernelArtifactPaths(vmi)
		if err != nil {
			return nil, err
		}

		if kernelArtifacts.kernel != nil {
			checksum, err := getDigest(kernelArtifacts.kernel)
			if err != nil {
				return nil, err
			}

			diskChecksums.KernelBootChecksum.Kernel = &checksum
		}

		if kernelArtifacts.initrd != nil {
			checksum, err := getDigest(kernelArtifacts.initrd)
			if err != nil {
				return nil, err
			}

			diskChecksums.KernelBootChecksum.Initrd = &checksum
		}
	}

	return diskChecksums, nil
}

type needsBindMountFunc func(vmi *v1.VirtualMachineInstance) (bool, error)

func newNeedsBindMountFunc(baseDir string) needsBindMountFunc {
	return func(vmi *v1.VirtualMachineInstance) (bool, error) {
		for podUID := range vmi.Status.ActivePods {
			virtLauncherSocketPath := cmdclient.SocketDirectoryOnHost(string(podUID))
			launcherSocketExists, err := diskutils.FileExists(virtLauncherSocketPath)
			if err != nil {
				return false, err
			}
			basePath := fmt.Sprintf("%s/pods/%s/containers", baseDir, string(podUID))
			containerDiskPath := filepath.Join(basePath, "container-disk-binary")
			containerDiskInitContainerExists, err := diskutils.FileExists(containerDiskPath)
			if err != nil {
				return false, err
			}
			// we must check for launcherSocket to make sure this isn't an old launcher that is already completed
			if launcherSocketExists && containerDiskInitContainerExists {
				return true, nil
			}
		}
		return false, nil
	}
}
