package device_manager

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

		dpi = NewGenericDevicePlugin("foo", devicePath, 1)
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

	It("Should monitor health of device node", func() {
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

func TestGenericDevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generic Device")
}
