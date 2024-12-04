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

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

const dispatcher = "virtiofs-dispatcher"

var virtiofsKubeletVolumePath = filepath.Join("volumes/kubernetes.io~empty-dir", virtiofs.ExtraVolName)

type VirtiofsManager struct {
	mountBaseDir string
}

func NewVirtiofsManager(mountBaseDir string) *VirtiofsManager {
	return &VirtiofsManager{mountBaseDir: mountBaseDir}
}

func (m *VirtiofsManager) virtiofsPlaceholderSocketFromHost(podUID, volumeName string) string {
	return filepath.Join(m.mountBaseDir, podUID, virtiofsKubeletVolumePath, virtiofs.VirtiofsPlaceholderSocketName(volumeName))
}

var execCommand = exec.Command
var getPid = isolation.GetPid

func (m *VirtiofsManager) StartVirtiofsDispatcher(vmi *v1.VirtualMachineInstance) error {
	vols := virtiofs.GetFilesystemPersistentVolumes(vmi)
	pid := -1
	var err error
	for _, v := range vols {
		for podUID, _ := range vmi.Status.ActivePods {
			socket := m.virtiofsPlaceholderSocketFromHost(string(podUID), v.Name)
			pid, err = getPid(socket)
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed getting the virtiofs placeholder socket %s", socket)
				return err
			}
		}
		if pid == -1 {
			return fmt.Errorf("pid not found")
		}

		// Detect if the virtiofs socket was already existing, meaning virtiofs has already been launched
		// or there was an error to detect it.
		path := virtiofs.VirtioFSSocketPath(v.Name)
		if exists, err := diskutils.FileExists(path); exists || err != nil {
			return err
		}

		cmd := execCommand(dispatcher, "--pid", strconv.Itoa(pid),
			"--socket-path", path,
			"--shared-dir", virtiofs.VirtioFSMountPoint(&v))
		log.DefaultLogger().Object(vmi).Infof("Dispatcher: %s", cmd.String())
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("virtiofs-dispatcher failed output:%s error:%v", stdoutStderr, err)
		}
		log.DefaultLogger().Object(vmi).Infof("Launch virtiofs dispatcher: %s", stdoutStderr)
	}

	return nil
}
