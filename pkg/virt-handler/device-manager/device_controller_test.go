package device_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakePlugin struct {
	Starts     int32
	devicePath string
	deviceName string
	Error      error
}

func (fp *FakePlugin) Start(stop chan struct{}) (err error) {
	atomic.AddInt32(&fp.Starts, 1)
	return fp.Error
}

func (fp *FakePlugin) GetDevicePath() string {
	return fp.devicePath
}

func (fp *FakePlugin) GetDeviceName() string {
	return fp.deviceName
}

func NewFakePlugin(name string, path string) *FakePlugin {
	return &FakePlugin{
		deviceName: name,
		devicePath: path,
	}
}

var _ = Describe("Device Controller", func() {
	var workDir string
	var err error
	var host string
	var stop chan struct{}

	BeforeEach(func() {
		workDir, err = ioutil.TempDir("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		stop = make(chan struct{})
	})

	AfterEach(func() {
		close(stop)
		os.RemoveAll(workDir)
	})

	Context("Basic Tests", func() {
		It("Should indicate if node has device", func() {
			deviceController := NewDeviceController(host, 10)
			devicePath := path.Join(workDir, "fake-device")
			res := deviceController.nodeHasDevice(devicePath)
			Expect(res).To(BeFalse())

			fileObj, err := os.Create(devicePath)
			Expect(err).ToNot(HaveOccurred())
			fileObj.Close()

			res = deviceController.nodeHasDevice(devicePath)
			Expect(res).To(BeTrue())
		})
	})

	Context("Multiple Plugins", func() {
		var devicePath1 string
		var devicePath2 string

		var plugin1 *FakePlugin
		var plugin2 *FakePlugin

		BeforeEach(func() {
			// only create the second device.
			devicePath1 = path.Join(workDir, "fake-device1")
			devicePath2 = path.Join(workDir, "fake-device2")
			os.Create(devicePath2)

			plugin1 = NewFakePlugin("fake-device1", devicePath1)
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
		})

		It("should restart the device plugin immeidiately without delays", func() {
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
			deviceController := NewDeviceController(host, 10)
			deviceController.devicePlugins = []GenericDevice{plugin2}
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 10 * time.Second}
			go deviceController.Run(stop)
			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically(">=", 3))
		})

		It("should restart the device plugin with delays if it returns errors", func() {
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
			plugin2.Error = fmt.Errorf("failing")
			deviceController := NewDeviceController(host, 10)
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 300 * time.Millisecond}
			deviceController.devicePlugins = []GenericDevice{plugin2}
			go deviceController.Run(stop)
			Consistently(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically("<", 3))

		})

		It("Should not block on other plugins", func() {
			deviceController := NewDeviceController(host, 10)
			deviceController.devicePlugins = []GenericDevice{plugin1, plugin2}
			go deviceController.Run(stop)

			Expect(deviceController.nodeHasDevice(devicePath1)).To(BeFalse())
			Expect(deviceController.nodeHasDevice(devicePath2)).To(BeTrue())

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin1.Starts))
			}).Should(BeNumerically(">=", 1))

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}).Should(BeNumerically(">=", 1))
		})
	})
})
