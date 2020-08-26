package device_manager

import (
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	fakeName       = "example.org/deadbeef"
	fakeID         = "DEAD:BEEF"
	fakeDriver     = "vfio-pci"
	fakeAddress    = "0000:00:00.0"
	fakeIommuGroup = "0"
	fakeNumaNode   = 0
)

var _ = Describe("PCI Device", func() {
	var mockPCI *MockDeviceHandler
	var fakePermittedHostDevicesConfig string
	var fakePermittedHostDevices v1.PermittedHostDevices
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockPCI = NewMockDeviceHandler(ctrl)
		Handler = mockPCI
		mockPCI.EXPECT().GetDeviceIOMMUGroup(pciBasePath, fakeAddress).Return(fakeIommuGroup, nil).Times(1)
		mockPCI.EXPECT().GetDeviceDriver(pciBasePath, fakeAddress).Return(fakeDriver, nil).Times(1)
		mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, fakeAddress).Return(fakeNumaNode).Times(1)
		mockPCI.EXPECT().GetDevicePCIID(pciBasePath, fakeAddress).Return(fakeID, nil).Times(1)
		mockPCI.EXPECT().GetDeviceIOMMUGroup(pciBasePath, gomock.Any()).AnyTimes()
		mockPCI.EXPECT().GetDeviceDriver(pciBasePath, gomock.Any()).AnyTimes()
		mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, gomock.Any()).AnyTimes()
		mockPCI.EXPECT().GetDevicePCIID(pciBasePath, gomock.Any()).AnyTimes()

		fakePermittedHostDevicesConfig = `
pciDevices:
- pciVendorSelector: "` + fakeID + `"
  resourceName: "` + fakeName + `"
`
		err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(fakePermittedHostDevicesConfig), 1024).Decode(&fakePermittedHostDevices)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fakePermittedHostDevices.PciHostDevices)).To(Equal(1))
		Expect(fakePermittedHostDevices.PciHostDevices[0].Selector).To(Equal(fakeID))
		Expect(fakePermittedHostDevices.PciHostDevices[0].ResourceName).To(Equal(fakeName))
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("Should parse the permitted devices and find 1 matching PCI device", func() {
		// TODO: skip the test if the current environment doesn't have a root PCI device?
		// i.e. if /sys/bus/pci/devices/0000:00:00.0 doesn't exist
		supportedPCIDeviceMap := make(map[string]string)
		Expect(len(fakePermittedHostDevices.PciHostDevices)).To(Equal(1))
		for _, pciDev := range fakePermittedHostDevices.PciHostDevices {
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !pciDev.ExternalResourceProvider {
				supportedPCIDeviceMap[pciDev.Selector] = pciDev.ResourceName
			}
		}
		// discoverPermittedHostPCIDevices() will walk real PCI devices wherever the tests are running
		// It's assumed here that it will find a PCI device at 0000:00:00.0
		devices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
		Expect(len(devices)).To(Equal(1))
		Expect(len(devices[fakeID])).To(Equal(1))
		Expect(devices[fakeID][0].pciID).To(Equal(fakeID))
		Expect(devices[fakeID][0].driver).To(Equal(fakeDriver))
		Expect(devices[fakeID][0].pciAddress).To(Equal(fakeAddress))
		Expect(devices[fakeID][0].iommuGroup).To(Equal(fakeIommuGroup))
		Expect(devices[fakeID][0].numaNode).To(Equal(fakeNumaNode))
	})

	It("Should update the device list according to the configmap", func() {
		//
		kvLabels := make(map[string]string)
		kvLabels["kubevirt.io"] = ""
		configMapData := make(map[string]string)
		configMapData[virtconfig.PermittedHostDevicesKey] = fakePermittedHostDevicesConfig
		fakeClusterConfig, fakeInformer, _, _, fakeHostDevInformer := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
		testutils.UpdateFakeClusterConfigByName(fakeHostDevInformer, &k8sv1.ConfigMap{
			Data: configMapData,
		}, testutils.HostDevicesConfigMapName)
		duh := fakeClusterConfig.GetPermittedHostDevices()
		Expect(duh).ToNot(BeNil())
		Expect(len(duh.PciHostDevices)).To(Equal(1))
		deviceController := NewDeviceController("master", 10, fakeClusterConfig, fakeInformer)
		enabledDevicePlugins, disabledDevicePlugins := deviceController.updatePermittedHostDevicePlugins()
		Expect(len(enabledDevicePlugins)).To(Equal(1))
		Expect(len(disabledDevicePlugins)).To(Equal(0))
		Ω(enabledDevicePlugins).Should(HaveKey(fakeName))
	})
})
