package isolation

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Isolation", func() {

	Context("With an existing socket", func() {

		var socket net.Listener
		var tmpDir string
		vm := v1.NewMinimalVMIWithNS("default", "testvm")

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "kubevirt")
			os.MkdirAll(tmpDir+"/sockets/", os.ModePerm)
			socket, err = net.Listen("unix", cmdclient.SocketFromUID(
				tmpDir,
				string(vm.UID)),
			)
			Expect(err).ToNot(HaveOccurred())
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
		})
	})
})
