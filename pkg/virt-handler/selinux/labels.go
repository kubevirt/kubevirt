package selinux

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const procOnePrefix = "/proc/1/root"

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
		procOnePrefix: procOnePrefix,
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

func (se *SELinuxImpl) Label(label string, dir string) (err error) {
	dir = strings.TrimRight(dir, "/") + "(/.*)?"
	out, err := se.execute("semanage", se.Paths, "fcontext", "-a", "-t", label, dir)
	if err != nil {
		return fmt.Errorf("failed to set label for directory %v: %v ", dir, string(out))
	}
	return nil
}

func (se *SELinuxImpl) IsLabeled(dir string) (labeled bool, err error) {
	dir = strings.TrimRight(dir, "/") + "(/.*)?"
	out, err := se.execute("semanage", se.Paths, "fcontext", "-l")
	if err != nil {
		return false, fmt.Errorf("failed to list labels: %v ", string(out))
	}
	if strings.Contains(string(out), dir) {
		return true, nil
	}
	return false, nil
}

func (se *SELinuxImpl) Restore(dir string) (err error) {
	dir = strings.TrimRight(dir, "/") + "/"
	out, err := se.execute("restorecon", se.Paths, "-r", "-v", dir)
	if err != nil {
		return fmt.Errorf("failed to set selinux permissions: %v ", string(out))
	}
	return nil
}

func (*SELinuxImpl) InstallPolicy(dir string) (err error) {
	for _, policyName := range POLICY_FILES {
		fileDest := dir + "/" + policyName + ".cil"
		modules, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/semodule", "-l").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to retrive a list of installed modules, err: % v", err)
		}
		if !strings.Contains(string(modules), policyName) {
			err := copyPolicy(policyName, dir)
			if err != nil {
				return fmt.Errorf("failed to copy policy %v - err: % v", fileDest, err)
			}
			_, err = exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "exec", "--", "/usr/sbin/semodule", "-i", fileDest).CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to install policy %v - err: % v", fileDest, err)
			}
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
