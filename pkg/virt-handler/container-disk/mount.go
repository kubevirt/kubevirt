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

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
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

const (
	failedCheckMountPointFmt = "failed to check mount point for containerDisk %v: %v"
	failedUnmountFmt         = "failed to unmount containerDisk %v: %v : %v"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector       isolation.PodIsolationDetector
	mountStateDir              string
	mountRecords               map[types.UID]*vmiMountTargetRecord
	mountRecordsLock           sync.Mutex
	suppressWarningTimeout     time.Duration
	socketPathGetter           containerdisk.SocketPathGetter
	kernelBootSocketPathGetter containerdisk.KernelBootSocketPathGetter
	clusterConfig              *virtconfig.ClusterConfig
}

type Mounter interface {
	ContainerDisksReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	MountAndVerify(vmi *v1.VirtualMachineInstance) (map[string]*containerdisk.DiskInfo, error)
	Unmount(vmi *v1.VirtualMachineInstance) error
}

type vmiMountTargetEntry struct {
	TargetFile string `json:"targetFile"`
	SocketFile string `json:"socketFile"`
}

type vmiMountTargetRecord struct {
	MountTargetEntries []vmiMountTargetEntry `json:"mountTargetEntries"`
	UsesSafePaths      bool                  `json:"usesSafePaths"`
}

func NewMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string, clusterConfig *virtconfig.ClusterConfig) Mounter {
	return &mounter{
		mountRecords:               make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector:       isoDetector,
		mountStateDir:              mountStateDir,
		suppressWarningTimeout:     1 * time.Minute,
		socketPathGetter:           containerdisk.NewSocketPathGetter(""),
		kernelBootSocketPathGetter: containerdisk.NewKernelBootSocketPathGetter(""),
		clusterConfig:              clusterConfig,
	}
}

func (m *mounter) deleteMountTargetRecord(vmi *v1.VirtualMachineInstance) error {
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

			nodeRes := isolation.NodeIsolationResult()

			if isMounted, err := isolation.IsMounted(targetFile); err != nil {
				return nil, fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {
				sock, err := m.socketPathGetter(vmi, i)
				if err != nil {
					return nil, err
				}

				res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
				if err != nil {
					return nil, fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
				}
				mountPoint, err := isolation.ParentPathForRootMount(nodeRes, res)
				if err != nil {
					return nil, fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
				}
				sourceFile, err := containerdisk.GetImage(mountPoint, volume.ContainerDisk.Path)
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

	nodeRes := isolation.NodeIsolationResult()

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

		mounted, err := nodeRes.AreMounted(artifactFiles...)
		return mounted, err
	}

	if isMounted, err := areKernelArtifactsMounted(targetDir, targetInitrdPath, targetKernelPath); err != nil {
		return fmt.Errorf("failed to determine if %s is already mounted: %v", targetDir, err)
	} else if !isMounted {
		log.Log.Object(vmi).Infof("kernel artifacts are not mounted - mounting...")

		res, err := m.podIsolationDetector.DetectForSocket(vmi, socketFilePath)
		if err != nil {
			return fmt.Errorf("failed to detect socket for containerDisk %v: %v", kernelBootName, err)
		}
		mountRootPath, err := isolation.ParentPathForRootMount(nodeRes, res)
		if err != nil {
			return fmt.Errorf("failed to detect root mount point of %v on the node: %v", kernelBootName, err)
		}

		mount := func(artifactPath string, targetPath *safepath.Path) error {

			sourcePath, err := containerdisk.GetImage(mountRootPath, artifactPath)
			if err != nil {
				return err
			}

			out, err := virt_chroot.MountChroot(sourcePath, targetPath, true).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to bindmount %v: %v : %v", kernelBootName, string(out), err)
			}

			return nil
		}

		if kb.InitrdPath != "" {
			if err = mount(kb.InitrdPath, targetInitrdPath); err != nil {
				return err
			}
		}

		if kb.KernelPath != "" {
			if err = mount(kb.KernelPath, targetKernelPath); err != nil {
				return err
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
