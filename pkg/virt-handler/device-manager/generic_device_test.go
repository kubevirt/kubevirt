package device_manager

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Generic Device", func() {
	var workDir string
	var err error
	var dpi *GenericDevicePlugin
	var stop chan struct{}
	var devicePath string

	BeforeEach(func() {
		workDir, err = ioutil.TempDir("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		devicePath = path.Join(workDir, "foo")
		fileObj, err := os.Create(devicePath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()

		dpi = NewGenericDevicePlugin("foo", devicePath, 1, "rw", true)
		dpi.socketPath = filepath.Join(workDir, "test.sock")
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.done = make(chan struct{})
		dpi.deviceRoot = "/"
		stop = make(chan struct{})
		dpi.stop = stop

	})

	AfterEach(func() {
		close(stop)
		os.RemoveAll(workDir)
	})

	It("Should allocate a new device upon request", func() {
		previousCount := dpi.counter
		dpi.addNewGenericDevice()

		Expect(dpi.counter).To(Equal(previousCount + 1))
		Expect(len(dpi.devs)).To(Equal(dpi.counter))
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 2*time.Second, 500*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).To(BeNil())
	})

	It("Should monitor health of device node", func() {

		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		go dpi.healthCheck()
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

		time.Sleep(1 * time.Second)
		By("Removing a (fake) device node")
		os.Remove(devicePath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(func() string {
			return <-dpi.health
		}, 5*time.Second).Should(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		fileObj, err := os.Create(devicePath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()

		By("waiting for healthcheck to send Healthy message")
		Eventually(func() string {
			return <-dpi.health
		}, 5*time.Second).Should(Equal(pluginapi.Healthy))
	})
})
