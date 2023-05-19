package device_manager

import (
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Socket device", func() {
	var workDir string
	var err error
	var dpi *SocketDevicePlugin
	var stop chan struct{}
	var sockPath string

	BeforeEach(func() {
		workDir, err = os.MkdirTemp("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		sockPath = path.Join(workDir, "fake-test.sock")
		fileObj, err := os.Create(sockPath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()
		dpi = NewSocketDevicePlugin("test", workDir, "fake-test.sock", 1)
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.pluginSocketPath = filepath.Join(workDir, "kubevirt-test.sock")
		dpi.done = make(chan struct{})
		stop = make(chan struct{})
		dpi.stop = stop
	})

	AfterEach(func() {
		close(stop)
		os.RemoveAll(workDir)
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.pluginSocketPath, os.O_RDONLY|os.O_CREATE, 0666)
		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 2*time.Second, 500*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.pluginSocketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})

	It("Should monitor health of device node", func() {
		os.OpenFile(dpi.pluginSocketPath, os.O_RDONLY|os.O_CREATE, 0666)

		go dpi.healthCheck()
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

		By("Removing a (fake) device node")
		os.Remove(sockPath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		fileObj, err := os.Create(sockPath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()

		By("waiting for healthcheck to send Healthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Healthy))
	})
})
