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
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
	var fakeConfigMap *virtconfig.ClusterConfig

	BeforeEach(func() {
		permittedDevices := `{"pciHostDevices":[{"pciVendorSelector":"DEAD:BEEF","resourceName":"fake-device2","externalResourceProvider":"true"},{"pciVendorSelector":"DEAD:BEEG","resourceName":"fake-device1","externalResourceProvider":"true"}]}`
		fakeConfigMap, _, _, _ = testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{
			Data: map[string]string{virtconfig.PermittedHostDevicesKey: permittedDevices},
		})
		workDir, err = ioutil.TempDir("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		stop = make(chan struct{})
	})

	AfterEach(func() {
		os.RemoveAll(workDir)
	})

	Context("Basic Tests", func() {
		It("Should indicate if node has device", func() {
			deviceController := NewDeviceController(host, 10, fakeConfigMap)
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

		It("should restart the device plugin immediately without delays", func() {
			deviceController := NewDeviceController(host, 10, fakeConfigMap)
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 10 * time.Second}
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop,
			}
			go deviceController.Run(stop)
			Eventually(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically(">=", 3))
			Expect(deviceController.Initialized()).To(BeTrue())
		})

		It("should restart the device plugin with delays if it returns errors", func() {
			plugin2 = NewFakePlugin("fake-device2", devicePath2)
			plugin2.Error = fmt.Errorf("failing")
			deviceController := NewDeviceController(host, 10, fakeConfigMap)
			deviceController.backoff = []time.Duration{10 * time.Millisecond, 300 * time.Millisecond}
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop,
			}
			go deviceController.Run(stop)
			Consistently(func() int {
				return int(atomic.LoadInt32(&plugin2.Starts))
			}, 500*time.Millisecond).Should(BeNumerically("<", 3))

		})

		It("Should not block on other plugins", func() {
			deviceController := NewDeviceController(host, 10, fakeConfigMap)
			// New device controllers include the permanent device plugins, we don't want those
			deviceController.devicePlugins = make(map[string]ControlledDevice)
			deviceController.devicePlugins[deviceName1] = ControlledDevice{
				devicePlugin: plugin1,
				stopChan:     stop,
			}
			deviceController.devicePlugins[deviceName2] = ControlledDevice{
				devicePlugin: plugin2,
				stopChan:     stop,
			}
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
