package selinux

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
)

type execFunc = func(binary string, args ...string) ([]byte, error)

func defaultExecFunc(binary string, args ...string) ([]byte, error) {
	return exec.Command(binary, args...).CombinedOutput()
}

var POLICY_FILES = []string{"base_container", "virt_launcher"}

type SELinuxImpl struct {
	Paths         []string
	execFunc      execFunc
	procOnePrefix string
}

func NewSELinux() (SELinux, bool, error) {
	paths := []string{
		"/sbin/",
		"/usr/sbin/",
		"/bin/",
		"/usr/bin/",
	}
	selinux := &SELinuxImpl{
		Paths:         paths,
		execFunc:      defaultExecFunc,
		procOnePrefix: util.HostRootMount,
	}
	present, err := selinux.IsPresent()
	return selinux, present, err
}

func (se *SELinuxImpl) IsPresent() (present bool, err error) {
	_, exists, err := lookupPath("getenforce", se.procOnePrefix, se.Paths)
	if !exists {
		return exists, err
	}
	out, err := se.execute("getenforce", se.Paths)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), "Disabled") {
		return false, nil
	}
	return true, nil
}

func (se *SELinuxImpl) IsPermissive() (present bool, err error) {
	out, err := se.execute("getenforce", se.Paths)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), "Permissive") {
		return true, nil
	}
	return false, nil
}

func lookupPath(binary string, prefix string, paths []string) (string, bool, error) {
	for _, path := range paths {
		fullPath := filepath.Join(prefix, path, binary)
		_, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return "", false, err
		} else {
			return filepath.Join(path, binary), true, nil
		}
	}
	return "", false, nil
}

func (se *SELinuxImpl) execute(binary string, paths []string, args ...string) (out []byte, err error) {
	path, exists, err := lookupPath(binary, se.procOnePrefix, paths)
	if err != nil {
		return []byte{}, err
	} else if !exists {
		return []byte{}, fmt.Errorf("could not find binary %v", binary)
	}

	argsArray := []string{"--mount", "/proc/1/ns/mnt", "exec", "--", path}
	for _, arg := range args {
		argsArray = append(argsArray, arg)
	}

	return se.execFunc("/usr/bin/chroot", argsArray...)
}

func copyPolicy(policyName string, dir string) (err error) {
	sourceFile := "/" + policyName + ".cil"

	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read a policy file %v: %v ", sourceFile, err)
	}

	destinationFile := dir + "/" + sourceFile
	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to create a policy file %v: %v ", destinationFile, err)
	}
	return nil
}

// Label sets selinux label on the directory
func (se *SELinuxImpl) Label(label string, dir string) error {
	dir = strings.TrimRight(dir, "/") + "(/.*)?"
	if out, err := se.execute("semanage", se.Paths, "fcontext", "-a", "-t", label, dir); err != nil {
		if perm, _ := se.IsPermissive(); perm {
			log.Log.Warningf("Permissive mode, ignoring 'semanage' failure: out: %q, error: %v", string(out), err)
			return nil
		}
		return fmt.Errorf("failed to set label for directory %v: out: %q, error: %v", dir, string(out), err)
	}
	return nil
}

// IsLabeled verifies if the directory already labeled
func (se *SELinuxImpl) IsLabeled(dir string) (bool, error) {
	dir = strings.TrimRight(dir, "/") + "(/.*)?"
	out, err := se.execute("semanage", se.Paths, "fcontext", "-l")
	if err != nil {
		if perm, _ := se.IsPermissive(); perm {
			log.Log.Warningf("Permissive mode, ignoring 'semanage' failure: out: %q, error: %v", string(out), err)
			return false, nil
		}
		return false, fmt.Errorf("failed to list labels: out: %q, error: %v", string(out), err)
	}
	if strings.Contains(string(out), dir) {
		return true, nil
	}
	return false, nil
}

// Restore restores selinux labels on the directory
func (se *SELinuxImpl) Restore(dir string) error {
	dir = strings.TrimRight(dir, "/") + "/"
	if out, err := se.execute("restorecon", se.Paths, "-r", "-v", dir); err != nil {
		if perm, _ := se.IsPermissive(); perm {
			log.Log.Warningf("Permissive mode, ignoring 'restorecon' failure: out: %q, error: %v", string(out), err)
			return nil
		}
		return fmt.Errorf("failed to set selinux permissions: out: %q, error: %v", string(out), err)
	}
	return nil
}

func (se *SELinuxImpl) InstallPolicy(dir string) (err error) {
	for _, policyName := range POLICY_FILES {
		fileDest := dir + "/" + policyName + ".cil"
		err := copyPolicy(policyName, dir)
		if err != nil {
			return fmt.Errorf("failed to copy policy %v - err: % v", fileDest, err)
		}
		out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/semodule", "-i", fileDest).CombinedOutput()
		if err != nil {
			if perm, _ := se.IsPermissive(); perm {
				log.Log.Warningf("Permissive mode, ignoring 'semodule' failure: out: %q, error: %v", string(out), err)
				return nil
			}
			return fmt.Errorf("failed to install policy %v - err: % v", fileDest, err)
		}
	}
	return nil
}

type SELinux interface {
	Label(dir string, label string) (err error)
	IsLabeled(dir string) (labeled bool, err error)
	Restore(dir string) (err error)
	InstallPolicy(dir string) (err error)
}
