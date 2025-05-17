// SPDX-License-Identifier: Apache-2.0

package virtiofs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

var _ = Describe("Virtiofs dispatcher", func() {
	const (
		podUID  = "123456789"
		volName = "vol"
		pid     = 123
	)
	var (
		vfsdManager *VirtiofsManager
		f           fakeExecCommandExecutor
		vmi         *v1.VirtualMachineInstance
	)
	BeforeEach(func() {
		dir := GinkgoT().TempDir()
		vfsdManager = NewVirtiofsManager(dir)
		vfsdManager.execCommand = f.fakeExecDispatcherCommand
		path := filepath.Join(dir, podUID, "volumes/kubernetes.io~empty-dir", virtiofs.PlaceholderSocketVolumeName)
		Expect(os.MkdirAll(path, 0664)).To(Succeed())
		vmi = libvmi.New(
			libvmi.WithFilesystemPVC(volName),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithActivePod(podUID, "node01"),
			)),
		)
	})
	AfterEach(func() {
		vfsdManager.execCommand = exec.Command
		vfsdManager.getPeerPid = isolation.GetPeerPid
	})

	Context("StartVirtiofsDispatcher", func() {
		It("should succeed", func() {
			f.pid = pid
			vfsdManager.getPeerPid = func(socket string) (int, error) {
				return pid, nil
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).ToNot(HaveOccurred())
		})

		It("should fail if no pid for any active pods is found", func() {
			vfsdManager.getPeerPid = func(socket string) (int, error) {
				return -1, os.ErrNotExist
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).Should(MatchError("pid not found"))
		})

		It("should fail for any other error in getting the pid of the socket", func() {
			vfsdManager.getPeerPid = func(socket string) (int, error) {
				return -1, errors.New("error")
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).Should(HaveOccurred())
		})

		It("should succeeded if virtiofs has already been started", func() {
			virtiofs.VirtioFSContainersMountBaseDir = GinkgoT().TempDir()
			path := virtiofs.VirtioFSSocketPath(volName)
			Expect(os.WriteFile(path, []byte{}, 0755)).ToNot(HaveOccurred())
			vfsdManager.getPeerPid = func(socket string) (int, error) {
				return pid, nil
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).ToNot(HaveOccurred())
		})
	})
})

type fakeExecCommandExecutor struct {
	pid int
}

func (f *fakeExecCommandExecutor) fakeExecDispatcherCommand(_ string, args ...string) *exec.Cmd {
	for i, arg := range args {
		if arg == "--pid" {
			Expect(args[i+1]).To(Equal(strconv.Itoa(f.pid)))
		}
	}
	return exec.Command("true")
}
