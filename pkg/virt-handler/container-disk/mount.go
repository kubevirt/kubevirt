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

var (
	ErrChecksumMissing   = errors.New("missing checksum")
	ErrChecksumMismatch  = errors.New("checksum mismatch")
	ErrDiskContainerGone = errors.New("disk container is gone")
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type mounter struct {
	podIsolationDetector   isolation.PodIsolationDetector
	mountStateDir          string
	mountRecords           map[types.UID]*vmiMountTargetRecord
	mountRecordsLock       sync.Mutex
	suppressWarningTimeout time.Duration
	clusterConfig          *virtconfig.ClusterConfig
	nodeIsolationResult    isolation.IsolationResult
}

type Mounter interface {
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
		mountRecords:           make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector:   isoDetector,
		mountStateDir:          mountStateDir,
		suppressWarningTimeout: 1 * time.Minute,
		clusterConfig:          clusterConfig,
		nodeIsolationResult:    isolation.NodeIsolationResult(),
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
