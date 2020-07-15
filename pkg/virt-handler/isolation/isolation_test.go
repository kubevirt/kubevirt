package isolation

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Isolation", func() {

	Context("With an existing socket", func() {

		var socket net.Listener
		var tmpDir string
		var podsDir string
		var podUID string
		var finished chan struct{} = nil

		podUID = "pid-uid-1234"
		vm := v1.NewMinimalVMIWithNS("default", "testvm")
		vm.UID = "1234"
		vm.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): "myhost",
			},
		}
		vm.Status.NodeName = "myhost"

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "kubevirt")
			podsDir, err = ioutil.TempDir("", "pods")

			cmdclient.SetLegacyBaseDir(tmpDir)
			cmdclient.SetPodsBaseDir(tmpDir)

			os.MkdirAll(tmpDir+"/sockets/", os.ModePerm)
			socketFile := cmdclient.SocketFilePathOnHost(podUID)
			os.MkdirAll(filepath.Dir(socketFile), os.ModePerm)
			socket, err = net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			finished = make(chan struct{})
			go func() {
				for {
					conn, err := socket.Accept()
					if err != nil {
						close(finished)
						// closes when socket listener is closed
						return
					}
					conn.Close()
				}
			}()

		})

		It("Should detect the PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Pid()).To(Equal(os.Getpid()))
		})

		It("Should not detect any slice if there is no matching controller", func() {
			_, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"not_existing_slice"}).Detect(vm)
			Expect(err).To(HaveOccurred())
		})

		It("Should detect the 'devices' controller slice of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Slice()).To(HavePrefix("/"))
		})

		It("Should detect the PID namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PIDNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/pid", os.Getpid())))
		})

		It("Should detect the Mount root of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.MountRoot()).To(Equal(fmt.Sprintf("/proc/%d/root", os.Getpid())))
		})

		It("Should detect the Network namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector(tmpDir).Whitelist([]string{"devices"}).Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.NetNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/net", os.Getpid())))
		})

		AfterEach(func() {
			socket.Close()
			os.RemoveAll(tmpDir)
			os.RemoveAll(podsDir)
			if finished != nil {
				<-finished
			}
		})
	})
})

var _ = Describe("getMemlockSize", func() {
	vm := v1.NewMinimalVMIWithNS("default", "testvm")

	It("Should return correct number of bytes for memlock limit", func() {
		bytes_, err := getMemlockSize(vm)
		Expect(err).ToNot(HaveOccurred())
		// 1Gb (static part for vfio VMs) + 256Mb (estimated overhead) + 8 Mb (VM)
		Expect(int(bytes_)).To(Equal(1264389000))
	})
})
