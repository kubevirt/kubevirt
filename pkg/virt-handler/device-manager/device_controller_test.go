package device_manager

import (
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type FakePlugin struct {
	Starts     int32
	devicePath string
	deviceName string
	Error      error
}

func (fp *FakePlugin) Start(_ <-chan struct{}) (err error) {
	atomic.AddInt32(&fp.Starts, 1)
	return fp.Error
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
	var maxDevices int
	var permissions string
	var stop chan struct{}
	var fakeConfigMap *virtconfig.ClusterConfig
	var mockPCI *MockDeviceHandler
	var ctrl *gomock.Controller
	var clientTest *fake.Clientset

	BeforeEach(func() {
		clientTest = fake.NewSimpleClientset()
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
		workDir, err = os.MkdirTemp("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		maxDevices = 100
		permissions = "rw"
		stop = make(chan struct{})
	})

	AfterEach(func() {
		defer os.RemoveAll(workDir)
	})

	Context("Basic Tests", func() {
		It("Should indicate if node has device", func() {
			var noDevices []Device
			deviceController := NewDeviceController(host, maxDevices, permissions, noDevices, fakeConfigMap, clientTest.CoreV1())
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
			initialDevices := []Device{plugin2}
			deviceController := NewDeviceController(host, maxDevices, permissions, initialDevices, fakeConfigMap, clientTest.CoreV1())
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 10 * time.Second}

			go deviceController.Run(stop)
			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 5*time.Second).Should(BeNumerically(">=", 1))
			Expect(deviceController.Initialized()).To(BeTrue())
		})

		It("should restart the device plugin with delays if it returns errors", func() {
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
			plugin2.Error = fmt.Errorf("failing")
			initialDevices := []Device{plugin2}

			deviceController := NewDeviceController(host, maxDevices, permissions, initialDevices, fakeConfigMap, clientTest.CoreV1())
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 300 * time.Millisecond}

			go deviceController.Run(stop)
			Consistently(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically("<", 3))

		})

		It("Should not block on other plugins", func() {
			initialDevices := []Device{plugin1, plugin2}
			deviceController := NewDeviceController(host, maxDevices, permissions, initialDevices, fakeConfigMap, clientTest.CoreV1())

			go deviceController.Run(stop)

			Expect(deviceController.NodeHasDevice(devicePath1)).To(BeFalse())
			Expect(deviceController.NodeHasDevice(devicePath2)).To(BeTrue())

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin1.Starts))
			}, 5*time.Second).Should(BeNumerically(">=", 1))

			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 5*time.Second).Should(BeNumerically(">=", 1))
		})

		It("should remove all device plugins if permittedHostDevices is removed from the CR", func() {
			emptyConfigMap, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
			Expect(emptyConfigMap.GetPermittedHostDevices()).To(BeNil())

			initialDevices := []Device{plugin1, plugin2}
			deviceController := NewDeviceController(host, maxDevices, permissions, initialDevices, emptyConfigMap, clientTest.CoreV1())

			go deviceController.Run(stop)

			Eventually(func() bool {
				deviceController.startedPluginsMutex.Lock()
				defer deviceController.startedPluginsMutex.Unlock()
				_, exists1 := deviceController.startedPlugins[deviceName1]
				_, exists2 := deviceController.startedPlugins[deviceName2]
				return exists1 || exists2
			}, 5*time.Second).Should(BeFalse())
		})
	})
})
