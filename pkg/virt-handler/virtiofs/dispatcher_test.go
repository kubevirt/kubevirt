package virtiofs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

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
		execCommand = f.fakeExecDispatcherCommand
		dir := GinkgoT().TempDir()
		vfsdManager = NewVirtiofsManager(dir)
		path := filepath.Join(dir, podUID, "volumes/kubernetes.io~empty-dir", virtiofs.ExtraVolName)
		Expect(os.MkdirAll(path, 0664)).ToNot(HaveOccurred())
		vmi = libvmi.New(
			libvmi.WithFilesystemPVC(volName),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithActivePod(types.UID(podUID), "node01"),
			)),
		)
	})
	AfterEach(func() {
		execCommand = exec.Command
		getPid = isolation.GetPid
	})

	Context("StartVirtiofsDispatcher", func() {
		It("should succeed", func() {
			f.pid = pid
			getPid = func(socket string) (int, error) {
				return pid, nil
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).ToNot(HaveOccurred())
		})

		It("should fail if no pid for any active pods is found", func() {
			getPid = func(socket string) (int, error) {
				return -1, os.ErrNotExist
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).Should(MatchError("pid not found"))
		})

		It("should fail for any other error in getting the pid of the socket", func() {
			errStr := "one error"
			getPid = func(socket string) (int, error) {
				return -1, errors.New(errStr)
			}
			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).Should(MatchError(errStr))
		})

		It("should succeeded if virtiofs has already been started", func() {
			virtiofs.VirtioFSContainersMountBaseDir = GinkgoT().TempDir()
			path := virtiofs.VirtioFSSocketPath(volName)
			Expect(os.WriteFile(path, []byte{}, 0755)).ToNot(HaveOccurred())
			getPid = func(socket string) (int, error) {
				return pid, nil
			}

			Expect(vfsdManager.StartVirtiofsDispatcher(vmi)).ToNot(HaveOccurred())
		})
	})
})

type fakeExecCommandExecutor struct {
	pid int
}

func (f *fakeExecCommandExecutor) fakeExecDispatcherCommand(command string, args ...string) *exec.Cmd {
	for i, arg := range args {
		if arg == "--pid" {
			Expect(args[i+1]).To(Equal(strconv.Itoa(f.pid)))
		}
	}
	return exec.Command("true")
}
