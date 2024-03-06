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

func createExt3IfNotExist(mountNamespace string, deviceFile *safepath.Path) error {
	sb, err := blockfs.Probe(unsafepath.UnsafeAbsolute(deviceFile.Raw()))
	if err != nil {
		return err
	}
	if sb != nil {
		return nil
	}
	log.DefaultLogger().V(2).Infof("Creating ext3 filesystem on block device: %s", deviceFile)
	b, err := virt_chroot.ExecWithMountNamespace(mountNamespace, "/sbin/mkfs.ext3", "-O", "^has_journal", unsafepath.UnsafeRelative(deviceFile.Raw())).CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to run mkfs: %s, error: %w", b, err)
	}
	return nil
}

func mountRelativeIfNotMounted(mountNamespace string, source, target *safepath.Path) error {
	m, err := isExt3Mounted(target)
	if err != nil {
		return err
	}
	if m {
		return nil
	}
	log.DefaultLogger().V(2).Infof("Mounting block device: %s", source)
	b, err := virt_chroot.MountWithMountNamespaceAndRawPath(mountNamespace, unsafepath.UnsafeRelative(source.Raw()), unsafepath.UnsafeRelative(target.Raw()), "ext3", "sync").CombinedOutput()
	if err != nil {
		log.DefaultLogger().Errorf("failed to run virt-chroot mount: %s", b)
		return err
	}
	return nil
}

func isExt3Mounted(dir *safepath.Path) (bool, error) {
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
