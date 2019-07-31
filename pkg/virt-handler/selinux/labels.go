package selinux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type SELinuxImpl struct {
}

func NewSELinux() (SELinux, bool, error) {
	present, err := isPresent()
	return &SELinuxImpl{}, present, err
}

func isPresent() (present bool, err error) {
	_, err = os.Stat("/proc/1/root/usr/sbin/getenforce")
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/getenforce").CombinedOutput()
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), "Disabled") {
		return false, nil
	}
	return true, nil
}

func (*SELinuxImpl) Label(label string, dir string) (err error) {
	dir = strings.TrimRight(dir, "/") + "(/.*)?"
	out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/semanage", "fcontext", "-a", "-t", label, dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set label for directory %v: %v ", dir, string(out))
	}
	return nil
}

func (*SELinuxImpl) Restore(dir string) (err error) {
	dir = strings.TrimRight(dir, "/") + "/"
	out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/restorecon", "-r", "-v", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set selinux permissions: %v ", string(out))
	}
	return nil
}

type SELinux interface {
	Label(dir string, label string) (err error)
	Restore(dir string) (err error)
}
