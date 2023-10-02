package backendstorage

import (
	"errors"
	"fmt"
	"os"
	"path"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/safepath"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type Mounter interface {
	Mount(vmi *v1.VirtualMachineInstance) error
}

type mounter struct {
	podIsolationDetector isolation.PodIsolationDetector
}

// NewVolumeMounter returns a Mounter for VM backend storage.
func NewVolumeMounter(d isolation.PodIsolationDetector) Mounter {
	return &mounter{
		podIsolationDetector: d,
	}
}

// Mount handles the storage used to persist the device states.
//
// For block storage, it creates a filesystem on the block backend storage of
// the VM and mounts it to the target directory of the virt-launcher Pod.
//
// For filesystem storage, it properly configures the owner of the swtpm
// directories.
func (m *mounter) Mount(vmi *v1.VirtualMachineInstance) error {
	if !backendstorage.IsBackendStorageNeededForVMI(&vmi.Spec) {
		return nil
	}
	b, err := usingBlockStorage(vmi)
	if err != nil {
		return err
	}

	isolationResult, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect VMI pod: %w", err)
	}
	mountRoot, err := isolationResult.MountRoot()
	if err != nil {
		return fmt.Errorf("failed to get mount root for Pod: %w", err)
	}

	if b {
		return m.MountBlockDevice(vmi, mountRoot, isolationResult.MountNamespace())
	}
	return configureSwtpmDirsOwnership(vmi, mountRoot)
}
func (m *mounter) MountBlockDevice(vmi *v1.VirtualMachineInstance, mountRoot *safepath.Path, mountNamespace string) error {
	// Locate the block device file in the virt-launcher's namespace, e.g.,
	// "/proc/123456/root/dev/vm-state".
	deviceFile, err := mountRoot.AppendAndResolveWithRelativeRoot(backendstorage.BlockVolumeDevicePath)
	if err != nil {
		return fmt.Errorf("failed to locate the block device file: %w", err)
	}

	// Create the filesystem.
	if err := createExt4IfNotExist(mountNamespace, deviceFile); err != nil {
		return fmt.Errorf("failed to create ext4 filesystem on backend storage: %w", err)
	}

	// Crate the VM state directory and mount the filesystem. For example, it
	// will create "/proc/123456/root/var/lib/libvirt/vm-state", and mount
	// "/proc/123456/root/dev/vm-state" to
	// "/proc/123456/root/var/lib/libvirt/vm-state".
	vmStateDirPath := path.Join(unsafepath.UnsafeAbsolute(mountRoot.Raw()), backendstorage.PodVMStatePath)
	vmStateDirFile, err := os.Stat(vmStateDirPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(vmStateDirPath, os.ModePerm)
	} else if !vmStateDirFile.IsDir() {
		return fmt.Errorf("file %s is not a directory", vmStateDirPath)
	}
	vmStateDir, err := mountRoot.AppendAndResolveWithRelativeRoot(backendstorage.PodVMStatePath)
	if err != nil {
		return err
	}
	if err := mountRelativeIfNotMounted(mountNamespace, deviceFile, vmStateDir); err != nil {
		return err
	}

	// Prepare the directories for the persisted devices and create the symlinks.
	if err := prepareVMStateDirectories(vmi, vmStateDir); err != nil {
		return fmt.Errorf("failed to prepare VM state dirs: %w", err)
	}

	return nil
}
