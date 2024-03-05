package selinux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"

	"kubevirt.io/client-go/log"
)

const procOnePrefix = "/proc/1/root"

type execFunc = func(binary string, args ...string) ([]byte, error)
type copyPolicy = func(policyName string, dir string) (err error)

func defaultExecFunc(binary string, args ...string) ([]byte, error) {
	// #nosec No risk for attacket injection. args get specific selinux exec parameters
	return exec.Command(binary, args...).CombinedOutput()
}

var POLICY_FILES = []string{"virt_launcher"}

type SELinuxImpl struct {
	Paths          []string
	execFunc       execFunc
	copyPolicyFunc copyPolicy
	procOnePrefix  string
	mode           string
}

func NewSELinux() (SELinux, bool, error) {
	paths := []string{
		"/sbin/",
		"/usr/sbin/",
		"/bin/",
		"/usr/bin/",
	}
	selinux := &SELinuxImpl{
		Paths:          paths,
		execFunc:       defaultExecFunc,
		procOnePrefix:  procOnePrefix,
		copyPolicyFunc: defaultCopyPolicyFunc,
	}
	present, mode, err := selinux.IsPresent()
	selinux.mode = mode
	return selinux, present, err
}

func (se *SELinuxImpl) IsPresent() (present bool, mode string, err error) {
	out, err := se.selinux("getenforce")
	if err != nil {
		return false, string(out), err
	}
	outStr := strings.TrimSpace(string(out))
	if outStr == "disabled" {
		return false, outStr, nil
	}
	return true, outStr, nil
}

func lookupPath(binary string, prefix string, paths []string) (string, bool, error) {
	for _, path := range paths {
		fullPath := filepath.Join(prefix, path, binary)
		_, err := os.Stat(fullPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", false, err
		} else {
			return filepath.Join(path, binary), true, nil
		}
	}
	return "", false, nil
}

func (se *SELinuxImpl) semodule(args ...string) (out []byte, err error) {
	path, exists, err := lookupPath("semodule", se.procOnePrefix, se.Paths)
	if err != nil {
		return []byte{}, err
	} else if !exists {
		// on some environments some selinux related binaries are missing, e.g. when the cluster runs in containers (kind).
		// In such a case, inform the admin, but continue.
		if se.IsPermissive() {
			log.DefaultLogger().Warning("Permissive mode, ignoring missing 'semodule' binary. SELinux policies will not be installed.")
			return []byte{}, nil
		}
		return []byte{}, fmt.Errorf("could not find 'semodule' binary")
	}

	argsArray := []string{"--mount", virt_chroot.GetChrootMountNamespace(), "exec", "--", path}
	for _, arg := range args {
		argsArray = append(argsArray, arg)
	}

	out, err = se.execFunc(virt_chroot.GetChrootBinaryPath(), argsArray...)
	if err != nil && se.IsPermissive() {
		log.DefaultLogger().Warningf("Permissive mode, ignoring 'semodule' failure: out: %q, error: %v", string(out), err)
		return []byte{}, nil
	}

	return out, err
}

func (se *SELinuxImpl) IsPermissive() bool {
	return se.mode == "permissive"
}

func (se *SELinuxImpl) Mode() string {
	return se.mode
}

func (se *SELinuxImpl) selinux(args ...string) (out []byte, err error) {
	argsArray := []string{"--mount", virt_chroot.GetChrootMountNamespace(), "selinux"}
	for _, arg := range args {
		argsArray = append(argsArray, arg)
	}

	return se.execFunc(virt_chroot.GetChrootBinaryPath(), argsArray...)
}

type SELinux interface {
	Mode() string
	IsPermissive() bool
}

func RelabelFiles(newLabel string, continueOnError bool, files ...*safepath.Path) error {
	relabelArgs := []string{"selinux", "relabel", newLabel}
	for _, file := range files {
		cmd := exec.Command("virt-chroot", append(relabelArgs, "--root", unsafepath.UnsafeRoot(file.Raw()), unsafepath.UnsafeRelative(file.Raw()))...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			err := fmt.Errorf("error relabeling file %s with label %s. Reason: %v", file, newLabel, err)
			if !continueOnError {
				return err
			} else {
				log.DefaultLogger().Reason(err).Errorf("Relabeling a file faild, continuing since selinux is permissive.")
			}
		}
	}
	return nil
}

func GetVirtLauncherContext(vmi *v1.VirtualMachineInstance) (string, error) {
	detector := isolation.NewSocketBasedIsolationDetector(util.VirtShareDir)
	isolationRes, err := detector.Detect(vmi)
	if err != nil {
		return "", err
	}
	virtLauncherRoot, err := isolationRes.MountRoot()
	if err != nil {
		return "", err
	}
	context, err := safepath.GetxattrNoFollow(virtLauncherRoot, "security.selinux")
	if err != nil {
		return "", err
	}

	return string(context), nil
}
