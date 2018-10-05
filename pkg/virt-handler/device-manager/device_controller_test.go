package device_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakePlugin struct {
	started    chan struct{}
	devicePath string
	deviceName string
}

func (fp *FakePlugin) Start(stop chan struct{}) error {
	//fp.started <- struct{}{}
	close(fp.started)
	return nil
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
		started:    make(chan struct{}),
	}
}

var _ = Describe("Device Controller", func() {
	var workDir string
	var err error
	var host string
	var deviceController *DeviceController
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
			deviceController = NewDeviceController(host, 10)
			devicePath := path.Join(workDir, "fake-device")
			res := deviceController.nodeHasDevice(devicePath)
			Expect(res).To(BeFalse())

			fileObj, err := os.Create(devicePath)
			Expect(err).ToNot(HaveOccurred())
			fileObj.Close()

			res = deviceController.nodeHasDevice(devicePath)
			Expect(res).To(BeTrue())
		})

		It("Should stop waiting if channel is closed", func() {
			deviceController = NewDeviceController(host, 10)
			devicePath := path.Join(workDir, "fake-device")
			stopChan := make(chan struct{})
			close(stopChan)
			err := deviceController.waitForPath(devicePath, stopChan)
			Expect(err).To(HaveOccurred())
		})

		It("Should wait for path to exist", func() {
			deviceController = NewDeviceController(host, 10)
			devicePath := path.Join(workDir, "fake-device")

			By(fmt.Sprintf("Device Path: %s", devicePath))

			timeout := make(chan struct{})
			errChan := make(chan error)
			go func(devicePath string) {
				time.Sleep(1 * time.Second)

				fileObj, err := os.Create(devicePath)
				fileObj.Close()
				errChan <- err
			}(devicePath)

			go func(timeout chan struct{}) {
				time.Sleep(5 * time.Second)
				close(timeout)
			}(timeout)

			err := <-errChan
			Expect(err).ToNot(HaveOccurred())

			err = deviceController.waitForPath(devicePath, timeout)
			Expect(err).ToNot(HaveOccurred())
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

		It("Should not block on other plugins", func() {
			deviceController = NewDeviceController(host, 10)
			deviceController.devicePlugins = []GenericDevice{plugin1, plugin2}
			go deviceController.Run(stop)

			Expect(deviceController.nodeHasDevice(devicePath1)).To(BeFalse())
			Expect(deviceController.nodeHasDevice(devicePath2)).To(BeTrue())

			timeout := make(chan struct{})

			go func() {
				time.Sleep(1 * time.Second)
				close(timeout)
			}()

			started := false
			timedOut := false
			select {
			case <-plugin2.started:
				started = true
			case <-timeout:
				timedOut = true
			}

			Expect(started).To(BeTrue(), "device plugin never started")
			Expect(timedOut).To(BeFalse(), "device plugin should not have timed out")
		})

		It("Should block until ready", func() {
			deviceController = NewDeviceController(host, 10)
			deviceController.devicePlugins = []GenericDevice{plugin1, plugin2}
			go deviceController.Run(stop)

			timeout := make(chan struct{})

			go func() {
				time.Sleep(1 * time.Second)
				close(timeout)
			}()

			started := false
			timedOut := false
			select {
			case <-plugin1.started:
				started = true
			case <-timeout:
				timedOut = true
			}

			Expect(started).To(BeFalse(), "device plugin should not have started")
			Expect(timedOut).To(BeTrue(), "device plugin should have timed out")
		})
	})
})
