package device_manager

import (
	"io/ioutil"
	"path"
	"os"
	"time"

	//"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	//"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	//"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("Device Controller", func() {
	//var virtClient *kubecli.MockKubevirtClient
	//var ctrl *gomock.Controller
	var workDir string
	var err error
	var host string
	var deviceController *DeviceController
	var stop chan struct{}

	Context("", func() {
		BeforeEach(func() {
			workDir, err = ioutil.TempDir("", "kubevirt-test")
			Expect(err).ToNot(HaveOccurred())

			//ctrl = gomock.NewController(GinkgoT())

			//virtClient = kubecli.NewMockKubevirtClient(ctrl)
			//virtClient.EXPECT().VM(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

			host = "master"
			stop = make(chan struct {})
		})

		AfterEach(func() {
			//ctrl.Finish()
			close(stop)
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
			stopChan := make(chan struct {})
			close(stopChan)
			err := deviceController.waitForPath(devicePath, stopChan)
			Expect(err).To(HaveOccurred())
		})

		It("Should wait for path to exist", func() {
			deviceController = NewDeviceController(host, 10)
			devicePath := path.Join(workDir, "fake-device")

			result := make(chan error)

			timeout := make(chan struct{})
			go func() {
				time.Sleep(1 * time.Second)
				close(timeout)
			}()

			go func() {
				err := deviceController.waitForPath(devicePath, timeout)
				result <- err
			}()

			fileObj, err := os.Create(devicePath)
			fileObj.Close()

			err = <-result
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
