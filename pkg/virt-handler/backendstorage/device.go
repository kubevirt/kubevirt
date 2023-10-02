package backendstorage

import (
	"fmt"
	"io"
	"os"

	blockfs "github.com/siderolabs/go-blockdevice/blockdevice/filesystem"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

func createExt4IfNotExist(mountNamespace string, deviceFile *safepath.Path) error {
	sb, err := blockfs.Probe(unsafepath.UnsafeAbsolute(deviceFile.Raw()))
	if err != nil {
		return err
	}
	if sb != nil {
		if sb.Type() == "ext4" {
			log.DefaultLogger().V(3).Infof("Block device %s already has ext4 filesystem", deviceFile)
			return nil
		} else {
			return fmt.Errorf("block device %s has non-ext4 filesystem: %s", deviceFile, sb.Type())
		}
	}
	log.DefaultLogger().V(2).Infof("Creating ext4 filesystem on block device: %s", deviceFile)
	b, err := virt_chroot.ExecWithMountNamespace(mountNamespace, "/sbin/mkfs.ext4", "-O", "^has_journal", unsafepath.UnsafeRelative(deviceFile.Raw())).CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to run mkfs: %s, error: %w", b, err)
	}
	return nil
}

func mountRelativeIfNotMounted(mountNamespace string, source, target *safepath.Path) error {
	m, err := isExt4Mounted(target)
	if err != nil {
		return err
	}
	if m {
		return nil
	}
	log.DefaultLogger().V(2).Infof("Mounting block device: %s", source)
	b, err := virt_chroot.MountWithMountNamespaceAndRawPath(mountNamespace, unsafepath.UnsafeRelative(source.Raw()), unsafepath.UnsafeRelative(target.Raw()), "ext4", "sync").CombinedOutput()
	if err != nil {
		log.DefaultLogger().Errorf("failed to run virt-chroot mount: %s", b)
		return err
	}
	return nil
}

func isExt4Mounted(dir *safepath.Path) (bool, error) {
	f, err := os.Open(unsafepath.UnsafeAbsolute(dir.Raw()))
	if err != nil {
		return false, err
	}
	defer f.Close()
	names, err := f.Readdirnames(0)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == "lost+found" {
			return true, nil
		}
	}
	return false, nil
}
