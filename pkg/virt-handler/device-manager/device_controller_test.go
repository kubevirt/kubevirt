package device_manager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type FakePlugin struct {
	Starts     int32
	devicePath string
	deviceName string
	Error      error
}

func (fp *FakePlugin) Start(_ chan struct{}) (err error) {
	atomic.AddInt32(&fp.Starts, 1)
	return fp.Error
}

func (fp *FakePlugin) GetDevicePath() string {
	return fp.devicePath
}

func (fp *FakePlugin) GetDeviceName() string {
	return fp.deviceName
}

func (fp *FakePlugin) GetInitialized() bool {
	return true
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
	var stop1 chan struct{}
	var stop2 chan struct{}
	var fakeConfigMap *virtconfig.ClusterConfig
	var mockPCI *MockDeviceHandler
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockPCI = NewMockDeviceHandler(ctrl)
		mockPCI.EXPECT().GetDevicePCIID(gomock.Any(), gomock.Any()).Return("1234:5678", nil).AnyTimes()
		Handler = mockPCI

		fakeConfigMap, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			PermittedHostDevices: &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector:        "DEAD:BEE7",
						ResourceName:             "example.org/fake-device1",
						ExternalResourceProvider: true,
					},
					{
						PCIVendorSelector:        "DEAD:BEEF",
						ResourceName:             "example.org/fake-device2",
						ExternalResourceProvider: true,
					},
				},
			},
		})

		Expect(fakeConfigMap.GetPermittedHostDevices()).ToNot(BeNil())
		workDir, err = ioutil.TempDir("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		stop = make(chan struct{})
		stop1 = make(chan struct{})
		stop2 = make(chan struct{})
	})

	AfterEach(func() {
		defer os.RemoveAll(workDir)

		ctrl.Finish()
	})

	Context("Basic Tests", func() {
		It("Should indicate if node has device", func() {
			deviceController := NewDeviceController(host, 10, "rw", fakeConfigMap)
			devicePath := path.Join(workDir, "fake-device")
			res := deviceController.NodeHasDevice(devicePath)
			Expect(res).To(BeFalse())

			fileObj, err := os.Create(devicePath)
			Expect(err).ToNot(HaveOccurred())
			fileObj.Close()

			res = deviceController.NodeHasDevice(devicePath)
			Expect(res).To(BeTrue())
		})
	})

	Context("Multiple Plugins", func() {
		var deviceName1 string
		var deviceName2 string
		var devicePath1 string
		var devicePath2 string

		var plugin1 *FakePlugin
		var plugin2 *FakePlugin

		BeforeEach(func() {
			deviceName1 = "fake-device1"
			deviceName2 = "fake-device2"
			devicePath1 = path.Join(workDir, deviceName1)
			devicePath2 = path.Join(workDir, deviceName2)
			// only create the second device.
			os.Create(devicePath2)

			plugin1 = NewFakePlugin(deviceName1, devicePath1)
			plugin2 = NewFakePlugin(deviceName2, devicePath2)
		})

		It("should start the device plugin immediately without delays", func() {
			deviceController := NewDeviceController(host, 10, "rw", fakeConfigMap)
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 10 * time.Second}
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop2,
			}
			go deviceController.Run(stop)
			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically(">=", 1))
			Expect(deviceController.Initialized()).To(BeTrue())
		})

		It("should restart the device plugin with delays if it returns errors", func() {
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
			plugin2.Error = fmt.Errorf("failing")
			deviceController := NewDeviceController(host, 10, "rw", fakeConfigMap)
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 300 * time.Millisecond}
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop2,
			}
			go deviceController.Run(stop)
			Consistently(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically("<", 3))

		})

		It("Should not block on other plugins", func() {
			deviceController := NewDeviceController(host, 10, "rw", fakeConfigMap)
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName1] = ControlledDevice{
				devicePlugin: plugin1,
				stopChan:     stop1,
			}
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop2,
			}
			go deviceController.Run(stop)

			Expect(deviceController.NodeHasDevice(devicePath1)).To(BeFalse())
			Expect(deviceController.NodeHasDevice(devicePath2)).To(BeTrue())

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin1.Starts))
			}).Should(BeNumerically(">=", 1))

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}).Should(BeNumerically(">=", 1))
		})

		It("should remove all device plugins if permittedHostDevices is removed from the CR", func() {
			emptyConfigMap, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			Expect(emptyConfigMap.GetPermittedHostDevices()).To(BeNil())
			deviceController := NewDeviceController(host, 10, "rw", emptyConfigMap)
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName1] = ControlledDevice{
				devicePlugin: plugin1,
				stopChan:     stop1,
			}
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop2,
			}
			go deviceController.Run(stop)

			Eventually(func() bool {
				deviceController.devicePluginsMutex.Lock()
				defer deviceController.devicePluginsMutex.Unlock()
				_, exists1 := deviceController.devicePlugins[deviceName1]
				_, exists2 := deviceController.devicePlugins[deviceName2]
				return exists1 || exists2
			}).Should(BeFalse())
		})
	})
})
