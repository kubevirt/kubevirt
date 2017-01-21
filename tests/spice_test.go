package tests_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
	"math/rand"
	"net"
	"strings"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)
	var vm *v1.VM

	BeforeEach(func() {
		vm = tests.NewRandomVMWithSpice()
		tests.MustCleanup()
	})

	Context("New VM with a spice connection given", func() {

		It("should return no connection details if VM does not exist", func(done Done) {
			result := restClient.Get().Resource("vms").SubResource("spice").Namespace(kubev1.NamespaceDefault).Name("something-random").Do()
			Expect(result.Error()).NotTo(BeNil())
			close(done)
		}, 3)

		It("should return connection details for running VMs in ini format", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			// TODO, that is going to be a common pattern, create some helpers for that
			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					return true
				}
				return false
			}).Watch()
			raw, err := restClient.Get().Resource("vms").SetHeader("Accept", "text/plain").SubResource("spice").Namespace(kubev1.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
			Expect(err).To(BeNil())
			spice := strings.Split(string(raw), "\n")

			Expect(spice[0]).To(Equal("[virt-viewer]"))
			Expect(spice[1]).To(Equal("type=spice"))
			Expect(spice[2]).To(ContainSubstring("host="))
			Expect(spice[3]).To(Equal("port=4000"))
			Expect(spice[4]).To(ContainSubstring("proxy="))
			close(done)
		}, 10)

		It("should return connection details for running VMs in json format", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			// TODO, that is going to be a common pattern, create some helpers for that
			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					return true
				}
				return false
			}).Watch()
			obj, err = restClient.Get().Resource("spices").Namespace(kubev1.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			spice := obj.(*v1.Spice).Info
			Expect(spice.Type).To(Equal("spice"))
			Expect(spice.Port).To(Equal(int32(4000)))
			close(done)
		}, 10)

		It("should allow accessing the spice device on the VM", func(done Done) {
			// Create the VM
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(err).To(BeNil())

			// Block until the VM is running
			// TODO, that is going to be a common pattern, create some helpers for that
			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					return true
				}
				return false
			}).Watch()
			raw, err := restClient.Get().Resource("vms").SetHeader("Accept", "text/plain").SubResource("spice").Namespace(kubev1.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Raw()
			Expect(err).To(BeNil())
			spice := strings.Split(string(raw), "\n")

			proxy := strings.TrimPrefix(spice[4], "proxy=http://")
			port := strings.TrimPrefix(spice[3], "port=")
			podIP := strings.TrimPrefix(spice[2], "host=")

			// Let's see if we can connect to the spice port through the proxy
			conn, err := net.Dial("tcp", proxy)
			Expect(err).To(BeNil())
			conn.Write([]byte("CONNECT " + podIP + ":" + port + " HTTP/1.1\r\n"))
			conn.Write([]byte("Host: " + podIP + ":" + port + "\r\n"))
			conn.Write([]byte("\r\n"))
			line, err := bufio.NewReader(conn).ReadString('\n')
			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(line)).To(Equal("HTTP/1.1 200 Connection established"))

			// Let's send a spice handshake
			conn.Write(newSpiceHandshake())

			// Let's parse the response
			var i int32
			x := make([]byte, 4, 4)
			io.ReadFull(conn, x)
			Expect(string(x)).To(Equal("REDQ")) // spice magic
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(2)), "Major version does not match.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(2)), "Minor version does not match.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(BeNumerically(">", 4), "Message not long enough.")
			binary.Read(conn, binary.LittleEndian, &i)
			Expect(i).To(Equal(int32(0)), "Message status is not OK.") // 0 is equal to OK

			close(done)
		}, 10)
	})

	AfterEach(func() {
		tests.MustCleanup()
	})
})

func newSpiceHandshake() []byte {
	var b []byte
	bb := bytes.NewBuffer(b)
	bb.Write([]byte("REDQ"))                                  // spice magic
	binary.Write(bb, binary.LittleEndian, uint32(2))          // protocol major version
	binary.Write(bb, binary.LittleEndian, uint32(2))          // protocol minor version
	binary.Write(bb, binary.LittleEndian, uint32(22))         // message size
	binary.Write(bb, binary.LittleEndian, uint32(rand.Int())) // session id
	binary.Write(bb, binary.LittleEndian, uint8(3))           // channel type
	binary.Write(bb, binary.LittleEndian, uint8(0))           // channel id
	binary.Write(bb, binary.LittleEndian, uint32(1))          // number of common capabilities
	binary.Write(bb, binary.LittleEndian, uint32(0))          // number of channel capabilities
	binary.Write(bb, binary.LittleEndian, uint32(18))         // capabilities offset
	binary.Write(bb, binary.LittleEndian, uint32(13))         // client common capabilities
	return bb.Bytes()
}
