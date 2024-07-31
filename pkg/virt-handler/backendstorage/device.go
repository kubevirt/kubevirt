package backendstorage

import (
	"fmt"
	"io"
	"os"
	"strings"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

func createExt3IfNotExist(mountNamespace string, deviceFile *safepath.Path) error {
	out, err := virt_chroot.ExecWithMountNamespace(mountNamespace, "/sbin/blkid", "-o", "export", unsafepath.UnsafeRelative(deviceFile.Raw())).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run blkid: %s, error: %w", out, err)
	}
	blkid := string(out)
	if strings.Contains(blkid, "TYPE=ext3") {
		return nil
	}
	if strings.Contains(blkid, "TYPE=") {
		return fmt.Errorf("partition contains non-ext3 filesystem: %s", blkid)
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
