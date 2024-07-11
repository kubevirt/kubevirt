package backendstorage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	utils "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

const (
	qemuUserID  = 107
	qemuGroupID = 107
	tssUserID   = 102
	tssGroupID  = 102

	defaultSelinuxContext = "system_u:object_r:container_file_t:s0"
)

var configureNonRootOwnerAndSelinuxContext = func(vmi *v1.VirtualMachineInstance, f *safepath.Path) error {
	if utils.IsNonRootVMI(vmi) {
		if err := os.Chown(unsafepath.UnsafeAbsolute(f.Raw()), qemuUserID, qemuGroupID); err != nil {
			return err
		}
	} else {
		// In the root VM case, the swtpm process is started with user "tss" which
		// is a non-root user. Therefore, we give this user the access of the swtpm
		// directories by changing the owner and group to "tss".
		p := unsafepath.UnsafeRelative(f.Raw())
		if strings.Contains(p, "swtpm") {
			if err := os.Chown(unsafepath.UnsafeAbsolute(f.Raw()), tssUserID, tssGroupID); err != nil {
				return err
			}
		}
	}
	_, present, err := selinux.NewSELinux()
	if err != nil {
		return err
	}
	if !present {
		return nil
	}
	return selinux.RelabelFiles(defaultSelinuxContext, false, f)
}

// usingBlockStorage checks if the VM is using block backend storage.
func usingBlockStorage(vmi *v1.VirtualMachineInstance) (bool, error) {
	for _, v := range vmi.Status.VolumeStatus {
		if v.Name != "vm-state" {
			continue
		}
		if v.PersistentVolumeClaimInfo == nil || v.PersistentVolumeClaimInfo.VolumeMode == nil {
			return false, fmt.Errorf("block storage volume name is not available")
		}
		if *v.PersistentVolumeClaimInfo.VolumeMode == k8sv1.PersistentVolumeBlock {
			return true, nil
		}
	}
	return false, nil
}

// prepareVMStateDirectories creates the VM state directory at
// "/var/lib/libvirt/vm-state". Besides, for each persistent device, it creates
// the corresponding subdirectories such as "/var/lib/libvirt/vm-state/nvram".
// Finally, It links the persistent device directory read by libvirt to the
// newly created subdirectory. For example:
// "/var/lib/libvirt/qemu/nvram -> /var/lib/libvirt/vm-state/nvram".
func prepareVMStateDirectories(vmi *v1.VirtualMachineInstance, vmStateDir *safepath.Path) error {
	if err := configureNonRootOwnerAndSelinuxContext(vmi, vmStateDir); err != nil {
		return err
	}

	rootPath := unsafepath.UnsafeRoot(vmStateDir.Raw())
	dirs := make(map[string]string)
	if backendstorage.HasPersistentEFI(&vmi.Spec) {
		dirs[services.PathForNVram(vmi)] = "nvram"
	}
	if backendstorage.HasPersistentTPMDevice(&vmi.Spec) {
		dirs[services.PathForSwtpm(vmi)] = "swtpm"
		dirs[services.PathForSwtpmLocalca(vmi)] = "swtpm-localca"
	}
	for srcRel, subDir := range dirs {
		srcAbs := path.Join(rootPath, srcRel)
		dstRel := path.Join(unsafepath.UnsafeRelative(vmStateDir.Raw()), subDir)
		dstAbs := path.Join(unsafepath.UnsafeAbsolute(vmStateDir.Raw()), subDir)
		createSymlink := true
		if f, err := os.Lstat(srcAbs); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		} else if err == nil { // if the file already exists
			if f.Mode()&os.ModeDir != 0 {
				if err := os.Remove(srcAbs); err != nil {
					return err
				}
			} else if f.Mode()&os.ModeSymlink != 0 {
				t, err := os.Readlink(srcAbs)
				if err != nil {
					return err
				}
				if t != dstRel {
					return fmt.Errorf("symlink %s exists but pointed to an unexpected target %s", srcAbs, t)
				}
				createSymlink = false
			} else {
				return fmt.Errorf("unknown file type: %s, %d", srcAbs, f.Mode())
			}
		}

		if err := os.MkdirAll(path.Dir(srcAbs), os.ModePerm); err != nil {
			return err
		}
		if err := os.MkdirAll(dstAbs, os.ModePerm); err != nil {
			return err
		}
		dst, err := safepath.JoinAndResolveWithRelativeRoot(rootPath, dstRel)
		if err != nil {
			return err
		}
		if configureNonRootOwnerAndSelinuxContext(vmi, dst); err != nil {
			return err
		}

		if createSymlink {
			log.DefaultLogger().V(2).Infof("Creating the symlink %s pointing to %s", srcAbs, dstRel)
			if err := os.Symlink(dstRel, srcAbs); err != nil {
				return err
			}
		}
	}
	return nil
}

// configureSwtpmDirsOwnership changes the owner and the group of the swtpm
// directories to "tss" for a root VM in the filesystem backend storage case.
// The block backend storage case is handled separately when the symlinks are
// created, as the directories are different.
//
// In the root VM case, the swtpm process is started with user "tss" which
// is a non-root user. Therefore, we give this user the access of the swtpm
// directories by changing the owner and group to "tss".
func configureSwtpmDirsOwnership(vmi *v1.VirtualMachineInstance, root *safepath.Path) error {
	if !backendstorage.HasPersistentTPMDevice(&vmi.Spec) {
		return nil
	}
	var uid, gid int
	if utils.IsNonRootVMI(vmi) {
		uid, gid = qemuUserID, qemuGroupID
	} else {
		uid, gid = tssUserID, tssGroupID
	}
	swtpm, err := root.AppendAndResolveWithRelativeRoot(services.PathForSwtpm(vmi))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !errors.Is(err, os.ErrNotExist) {
		if err := os.Chown(unsafepath.UnsafeAbsolute(swtpm.Raw()), uid, gid); err != nil {
			return err
		}
	}
	swtpmLocalca, err := root.AppendAndResolveWithRelativeRoot(services.PathForSwtpmLocalca(vmi))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !errors.Is(err, os.ErrNotExist) {
		if err := os.Chown(unsafepath.UnsafeAbsolute(swtpmLocalca.Raw()), uid, gid); err != nil {
			return err
		}
	}
	return nil
}
