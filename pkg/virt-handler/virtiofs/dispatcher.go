// SPDX-License-Identifier: Apache-2.0

package virtiofs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

const dispatcher = "virtiofs-dispatcher"

var virtiofsKubeletVolumePath = filepath.Join("volumes/kubernetes.io~empty-dir", virtiofs.PlaceholderSocketVolumeName)

type execCommandFn func(string, ...string) *exec.Cmd
type getPeerPidFn func(string) (int, error)

type VirtiofsManager struct {
	mountBaseDir string
	execCommand  execCommandFn
	getPeerPid   getPeerPidFn
}

func NewVirtiofsManager(mountBaseDir string) *VirtiofsManager {
	return newManager(mountBaseDir, exec.Command, isolation.GetPeerPid)
}

func newManager(mountBaseDir string, execCommand execCommandFn, getPeerPid getPeerPidFn) *VirtiofsManager {
	return &VirtiofsManager{
		mountBaseDir: mountBaseDir,
		execCommand:  execCommand,
		getPeerPid:   getPeerPid,
	}
}

func (m *VirtiofsManager) virtiofsPlaceholderSocketFromHost(podUID, volumeName string) string {
	return filepath.Join(m.mountBaseDir, podUID, virtiofsKubeletVolumePath, virtiofs.PlaceholderSocketName(volumeName))
}

func (m *VirtiofsManager) StartVirtiofsDispatcher(vmi *v1.VirtualMachineInstance) error {
	vols := virtiofs.GetFilesystemPersistentVolumes(vmi)
	pid := -1
	var err error
	for _, v := range vols {
		for podUID := range vmi.Status.ActivePods {
			socket := m.virtiofsPlaceholderSocketFromHost(string(podUID), v.Name)
			pid, err = m.getPeerPid(socket)
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return fmt.Errorf("failed getting the virtiofs placeholder socket %s error: %v", socket, err)
			}
		}
		if pid == -1 {
			return fmt.Errorf("pid not found")
		}

		path := virtiofs.VirtioFSSocketPath(v.Name)
		cmd := m.execCommand(dispatcher, "--pid", strconv.Itoa(pid),
			"--socket-path", path,
			"--shared-dir", virtiofs.FSMountPoint(&v))
		log.Log.Object(vmi).Infof("Dispatcher: %s", cmd.String())
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("virtiofs-dispatcher failed output:%s error: %v", stdoutStderr, err)
		}
		log.Log.Object(vmi).Infof("Launch virtiofs dispatcher: %s", stdoutStderr)
	}

	return nil
}
