/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virthandler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	// The node-labeller init container probes QEMU once and leaves the
	// libvirt cache here. virt-handler already mounts this host path.
	qemuCapsSourceDir = "/var/lib/kubevirt-node-labeller/qemu-capabilities"
	// libvirt looks up the cache here inside the virt-launcher mount namespace.
	qemuCapsTargetDir = "/var/run/kubevirt-private/libvirt/qemu/cache/capabilities"
)

// Directories created on the way to the cache file. Each one has to carry
// the pod's SELinux category or the launcher cannot traverse to the file.
var qemuCapsLabelPaths = []string{
	"/var/run/kubevirt-private/libvirt",
	"/var/run/kubevirt-private/libvirt/qemu",
	"/var/run/kubevirt-private/libvirt/qemu/cache",
	qemuCapsTargetDir,
}

var (
	relabelPath  = defaultRelabelPath
	readProcAttr = func(pid int) ([]byte, error) {
		return os.ReadFile(fmt.Sprintf("/proc/%d/attr/current", pid))
	}
)

// copyQEMUCapabilities gives this launcher its own copy of the node-labeller
// QEMU capability cache. The copy is labeled with that pod's category.
// Launchers do not mount the host directory: one pod must not be able to
// change the capabilities the next VM trusts, and the host file carries
// virt-handler's category, which the launcher cannot read.
// A missing cache is not a failure. Libvirt probes QEMU itself in that case.
func (c *BaseController) copyQEMUCapabilities(vmi *v1.VirtualMachineInstance) {
	iso, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).V(3).Info("qemu capability cache skipped, launcher is not detectable")
		return
	}
	label, err := containerFileLabel(iso.Pid())
	if err != nil {
		log.Log.Object(vmi).Reason(err).Info("qemu capability cache skipped, SELinux category is unknown")
		return
	}
	copied, err := copyQEMUCapabilityFiles(qemuCapsSourceDir, fmt.Sprintf("/proc/%d/root", iso.Pid()), label)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Info("qemu capability cache copy failed")
		return
	}
	if copied > 0 {
		log.Log.Object(vmi).Infof("copied %d qemu capability cache files into the launcher", copied)
	}
}

func containerFileLabel(pid int) (string, error) {
	raw, err := readProcAttr(pid)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	trimmed := strings.Trim(string(raw), "\x00\n ")
	label := containerFileLabelFromAttr(trimmed)
	if label == "" && trimmed != "" {
		return "", fmt.Errorf("unrecognized SELinux context %q", trimmed)
	}
	return label, nil
}

// containerFileLabelFromAttr turns the process domain in /proc/pid/attr/current
// into the file label for objects that process may read.
// "system_u:system_r:container_t:s0:c1,c2" becomes
// "system_u:object_r:container_file_t:s0:c1,c2".
func containerFileLabelFromAttr(raw string) string {
	raw = strings.Trim(raw, "\x00\n ")
	fields := strings.Split(raw, ":")
	if len(fields) < 4 || fields[0] == "" || fields[2] == "" {
		return ""
	}
	return "system_u:object_r:container_file_t:" + strings.Join(fields[3:], ":")
}

// launcherPath joins a container mount root with an absolute path inside it.
// The second path is trimmed so Join cannot treat it as a new root.
func launcherPath(mountRoot, absPath string) string {
	return filepath.Join(mountRoot, strings.TrimPrefix(absPath, "/"))
}

func copyQEMUCapabilityFiles(sourceDir, mountRoot, label string) (int, error) {
	names, err := qemuCapsXMLFiles(sourceDir)
	if err != nil || len(names) == 0 {
		return 0, err
	}
	dstDir := launcherPath(mountRoot, qemuCapsTargetDir)
	missing := make([]string, 0, len(names))
	for _, name := range names {
		_, err := os.Stat(filepath.Join(dstDir, name))
		if err == nil {
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
		missing = append(missing, name)
	}
	if len(missing) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return 0, err
	}
	if label != "" {
		for _, rel := range qemuCapsLabelPaths {
			path := launcherPath(mountRoot, rel)
			if err := relabelPath(path, label); err != nil {
				return 0, fmt.Errorf("relabel %s: %w", rel, err)
			}
		}
	}
	copied := 0
	for _, name := range missing {
		dst := filepath.Join(dstDir, name)
		if err := copyFileExclusive(filepath.Join(sourceDir, name), dst); err != nil {
			return copied, err
		}
		if label != "" {
			if err := relabelPath(dst, label); err != nil {
				_ = os.Remove(dst)
				return copied, fmt.Errorf("relabel %s: %w", name, err)
			}
		}
		copied++
	}
	return copied, nil
}

func qemuCapsXMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".xml" {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func copyFileExclusive(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

func defaultRelabelPath(path, label string) error {
	return exec.Command("chcon", label, path).Run()
}
