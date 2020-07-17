package container_disk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
)

type mounter struct {
	podIsolationDetector isolation.PodIsolationDetector
	mountStateDir        string
	mountRecords         map[types.UID]*vmiMountTargetRecord
	mountRecordsLock     sync.Mutex
}

type Mounter interface {
	Mount(vmi *v1.VirtualMachineInstance, verify bool) error
	Unmount(vmi *v1.VirtualMachineInstance) error
}

type vmiMountTargetEntry struct {
	TargetFile string `json:"targetFile"`
	SocketFile string `json:"socketFile"`
}

type vmiMountTargetRecord struct {
	MountTargetEntries []vmiMountTargetEntry `json:"mountTargetEntries"`
}

func NewMounter(isoDetector isolation.PodIsolationDetector, mountStateDir string) Mounter {
	return &mounter{
		mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
		podIsolationDetector: isoDetector,
		mountStateDir:        mountStateDir,
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
	return nil, nil
}

func (m *mounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to set container disk mounted directories for vmi without uid")
	}

	recordFile := filepath.Join(m.mountStateDir, string(vmi.UID))
	fileExists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return err
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()

	existingRecord, ok := m.mountRecords[vmi.UID]
	if ok && fileExists && reflect.DeepEqual(existingRecord, record) {
		// already done
		return nil
	}

	bytes, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(recordFile), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(recordFile, bytes, 0644)
	if err != nil {
		return err
	}

	m.mountRecords[vmi.UID] = record

	return nil
}

// Mount takes a vmi and mounts all container disks of the VMI, so that they are visible for the qemu process.
// Additionally qcow2 images are validated if "verify" is true. The validation happens with rlimits set, to avoid DOS.
func (m *mounter) Mount(vmi *v1.VirtualMachineInstance, verify bool) error {
	record := vmiMountTargetRecord{}

	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			targetFile, err := containerdisk.GetDiskTargetPathFromHostView(vmi, i)
			if err != nil {
				return err
			}

			sock, err := containerdisk.GetSocketPathFromHostView(vmi, i)
			if err != nil {
				return err
			}

			record.MountTargetEntries = append(record.MountTargetEntries, vmiMountTargetEntry{
				TargetFile: targetFile,
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
			targetFile, err := containerdisk.GetDiskTargetPathFromHostView(vmi, i)
			if err != nil {
				return err
			}

			nodeRes := isolation.NodeIsolationResult()

			if isMounted, err := nodeRes.IsMounted(targetFile); err != nil {
				return fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {
				sock, err := containerdisk.GetSocketPathFromHostView(vmi, i)
				if err != nil {
					return err
				}

				res, err := m.podIsolationDetector.DetectForSocket(vmi, sock)
				if err != nil {
					return fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
				}
				mountInfo, err := res.MountInfoRoot()
				if err != nil {
					return fmt.Errorf("failed to detect root mount info of containerDisk  %v: %v", volume.Name, err)
				}
				nodeMountInfo, err := nodeRes.ParentMountInfoFor(mountInfo)
				if err != nil {
					return fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
				}
				sourceFile, err := containerdisk.GetImage(filepath.Join(nodeRes.MountRoot(), nodeMountInfo.Root, nodeMountInfo.MountPoint), volume.ContainerDisk.Path)
				if err != nil {
					return fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}
				f, err := os.Create(targetFile)
				if err != nil {
					return fmt.Errorf("failed to create mount point target %v: %v", targetFile, err)
				}
				f.Close()

				if err = os.Chmod(sourceFile, 0444); err != nil {
					return fmt.Errorf("failed to change permisions on %s", sourceFile)
				}

				out, err := exec.Command("/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "mount", "-o", "ro,bind", strings.TrimPrefix(sourceFile, nodeRes.MountRoot()), targetFile).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}
			if verify {
				res, err := m.podIsolationDetector.Detect(vmi)
				if err != nil {
					return fmt.Errorf("failed to detect VMI pod: %v", err)
				}
				imageInfo, err := isolation.GetImageInfo(containerdisk.GetDiskTargetPathFromLauncherView(i), res)
				if err != nil {
					return fmt.Errorf("failed to get image info: %v", err)
				}

				if err := containerdisk.VerifyImage(imageInfo); err != nil {
					return fmt.Errorf("invalid image in containerDisk %v: %v", volume.Name, err)
				}
			}
		}
	}
	return nil
}

// Legacy Unmount unmounts all container disks of a given VMI when the hold HostPath method was in use.
// This exists for backwards compatibility for VMIs running before a KubeVirt update occurs.
func (m *mounter) legacyUnmount(vmi *v1.VirtualMachineInstance) error {
	mountDir := containerdisk.GetLegacyVolumeMountDirOnHost(vmi)

	files, err := ioutil.ReadDir(mountDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to list container disk mounts: %v", err)
	}

	if vmi.UID != "" {
		for _, file := range files {
			path := filepath.Join(mountDir, file.Name())
			if strings.HasSuffix(path, ".sock") {
				continue
			}
			if mounted, err := isolation.NodeIsolationResult().IsMounted(path); err != nil {
				return fmt.Errorf("failed to check mount point for containerDisk %v: %v", path, err)
			} else if mounted {
				out, err := exec.Command("/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "umount", path).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to unmount containerDisk %v: %v : %v", path, string(out), err)
				}
			}
		}

		if err := os.RemoveAll(mountDir); err != nil {
			return fmt.Errorf("failed to remove containerDisk files: %v", err)
		}
	}
	return nil
}

// Unmount unmounts all container disks of a given VMI.
func (m *mounter) Unmount(vmi *v1.VirtualMachineInstance) error {
	if vmi.UID != "" {

		// this will catch unmounting a vmi's container disk when
		// an old VMI is left over after a KubeVirt update
		err := m.legacyUnmount(vmi)
		if err != nil {
			return err
		}

		record, err := m.getMountTargetRecord(vmi)
		if err != nil {
			return err
		} else if record == nil {
			// no entries to unmount
			return nil
		}

		for _, entry := range record.MountTargetEntries {
			path := entry.TargetFile
			if mounted, err := isolation.NodeIsolationResult().IsMounted(path); err != nil {
				return fmt.Errorf("failed to check mount point for containerDisk %v: %v", path, err)
			} else if mounted {
				out, err := exec.Command("/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "umount", path).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to unmount containerDisk %v: %v : %v", path, string(out), err)
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
