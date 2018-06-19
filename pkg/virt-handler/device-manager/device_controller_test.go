package device_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Device Controller", func() {
	var workDir string
	var err error
	var host string
	var deviceController *DeviceController
	var stop chan struct{}

	Context("Basic Tests", func() {
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

			go func() {
				time.Sleep(1 * time.Second)

				fileObj, err := os.Create(devicePath)
				Expect(err).ToNot(HaveOccurred())
				fileObj.Close()
			}()

			go func() {
				time.Sleep(5 * time.Second)
				close(timeout)
			}()

			err := deviceController.waitForPath(devicePath, timeout)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Device Controller")
}
