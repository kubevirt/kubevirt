package device_manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	fakeName       = "example.org/deadbeef"
	fakeID         = "dead:beef"
	fakeDriver     = "vfio-pci"
	fakeIommuGroup = "0"
	fakeNumaNode   = 0
)

var _ = Describe("PCI Device", func() {
	var mockPCI *MockDeviceHandler
	var fakePermittedHostDevicesConfig string
	var fakePermittedHostDevices v1.PermittedHostDevices
	var ctrl *gomock.Controller
	var clientTest *fake.Clientset
	var fakeAddress string
	var pciAddresses []string

	BeforeEach(func() {
		clientTest = fake.NewSimpleClientset()
		By("making sure the environment has a PCI device")
		err := filepath.Walk(pciBasePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				pciAddresses = append(pciAddresses, info.Name())
			}
			return nil
		})
		if err != nil || len(pciAddresses) == 0 {
			Skip("No PCI devices found")
		}
		fakeAddress = pciAddresses[0]

		By("mocking PCI functions to simulate a vfio-pci device at " + fakeAddress)
		ctrl = gomock.NewController(GinkgoT())
		mockPCI = NewMockDeviceHandler(ctrl)
		Handler = mockPCI
		// Force pre-defined returned values and ensure the function only get called exacly once each for the address
		mockPCI.EXPECT().GetDeviceIOMMUGroup(pciBasePath, fakeAddress).Return(fakeIommuGroup, nil).Times(1)
		mockPCI.EXPECT().GetDeviceDriver(pciBasePath, fakeAddress).Return(fakeDriver, nil).Times(1)
		mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, fakeAddress).Return(fakeNumaNode).Times(1)
		mockPCI.EXPECT().GetDevicePCIID(pciBasePath, fakeAddress).Return(fakeID, nil).Times(1)

		By("creating a list of fake device using the yaml decoder")
		fakePermittedHostDevicesConfig = `
pciHostDevices:
- pciVendorSelector: "` + fakeID + `"
  resourceName: "` + fakeName + `"
`
		err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fakePermittedHostDevicesConfig), 1024).Decode(&fakePermittedHostDevices)
		Expect(err).ToNot(HaveOccurred())
		Expect(fakePermittedHostDevices.PciHostDevices).To(HaveLen(1))
		Expect(fakePermittedHostDevices.PciHostDevices[0].PCIVendorSelector).To(Equal(fakeID))
		Expect(fakePermittedHostDevices.PciHostDevices[0].ResourceName).To(Equal(fakeName))
	})

	It("Should parse the permitted devices and find 1 matching PCI device", func() {
		allowAnyDeviceCalls(*mockPCI)
		supportedPCIDeviceMap := make(map[string]string)
		for _, pciDev := range fakePermittedHostDevices.PciHostDevices {
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !pciDev.ExternalResourceProvider {
				supportedPCIDeviceMap[pciDev.PCIVendorSelector] = pciDev.ResourceName
			}
		}
		// discoverPermittedHostPCIDevices() will walk real PCI devices wherever the tests are running
		// It's assumed here that it will find a PCI device
		devices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
		Expect(devices).To(HaveLen(1), "only one PCI device is expected to be found")
		Expect(devices[fakeName]).To(HaveLen(1), "only one PCI device is expected to be found")
		Expect(devices[fakeName][0].pciID).To(Equal(fakeID))
		Expect(devices[fakeName][0].driver).To(Equal(fakeDriver))
		Expect(devices[fakeName][0].pciAddress).To(Equal(fakeAddress))
		Expect(devices[fakeName][0].iommuGroup).To(Equal(fakeIommuGroup))
		Expect(devices[fakeName][0].numaNode).To(Equal(fakeNumaNode))
	})

	It("Should validate DPI devices", func() {
		allowAnyDeviceCalls(*mockPCI)
		iommuToPCIMap := make(map[string]string)
		supportedPCIDeviceMap := make(map[string]string)
		for _, pciDev := range fakePermittedHostDevices.PciHostDevices {
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !pciDev.ExternalResourceProvider {
				supportedPCIDeviceMap[pciDev.PCIVendorSelector] = pciDev.ResourceName
			}
		}
		// discoverPermittedHostPCIDevices() will walk real PCI devices wherever the tests are running
		// It's assumed here that it will find a PCI device
		pciDevices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
		devs := constructDPIdevices(pciDevices[fakeName], iommuToPCIMap)
		Expect(devs[0].ID).To(Equal(strings.Join([]string{fakeIommuGroup, fakeAddress}, deviceIDSeparator)))
		Expect(devs[0].Topology.Nodes[0].ID).To(Equal(int64(fakeNumaNode)))
	})

	It("Should validate multiple devices in IOMMU group", func() {
		By("making sure the environment has at least 2 PCI devices")
		if len(pciAddresses) < 2 {
			Skip("the second PCI device was not found")
		}
		secondFakeAddress := pciAddresses[1]
		By("mocking PCI functions to simulate a vfio-pci device at " + secondFakeAddress)
		mockPCI.EXPECT().GetDeviceIOMMUGroup(pciBasePath, secondFakeAddress).Return(fakeIommuGroup, nil).Times(1)
		mockPCI.EXPECT().GetDeviceDriver(pciBasePath, secondFakeAddress).Return(fakeDriver, nil).Times(1)
		mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, secondFakeAddress).Return(fakeNumaNode).Times(1)
		mockPCI.EXPECT().GetDevicePCIID(pciBasePath, secondFakeAddress).Return(fakeID, nil).Times(1)
		allowAnyDeviceCalls(*mockPCI)

		iommuToPCIMap := make(map[string]string)
		supportedPCIDeviceMap := make(map[string]string)
		for _, pciDev := range fakePermittedHostDevices.PciHostDevices {
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !pciDev.ExternalResourceProvider {
				supportedPCIDeviceMap[pciDev.PCIVendorSelector] = pciDev.ResourceName
			}
		}

		pciDevices := discoverPermittedHostPCIDevices(supportedPCIDeviceMap)
		Expect(pciDevices[fakeName]).To(HaveLen(2), "two PCI device are expected to be found for: "+fakeName)
		devs := constructDPIdevices(pciDevices[fakeName], iommuToPCIMap)
		Expect(devs).To(HaveLen(2))
		Expect(iommuToPCIMap).To(HaveLen(2))
		devID := strings.Join([]string{fakeIommuGroup, fakeAddress}, deviceIDSeparator)
		secondDevID := strings.Join([]string{fakeIommuGroup, secondFakeAddress}, deviceIDSeparator)
		Expect(iommuToPCIMap).To(HaveKeyWithValue(devID, fakeAddress))
		Expect(iommuToPCIMap).To(HaveKeyWithValue(secondDevID, secondFakeAddress))

		devSpec := formatVFIODeviceSpecs(devID)
		Expect(devSpec).To(HaveLen(2))
		vfioDevice := filepath.Join(vfioDevicePath, fakeIommuGroup)
		Expect(devSpec).To(ContainElement(&v1beta1.DeviceSpec{
			HostPath:      vfioDevice,
			ContainerPath: vfioDevice,
			Permissions:   "mrw",
		}))
	})

	It("Should update the device list according to the configmap", func() {
		allowAnyDeviceCalls(*mockPCI)
		By("creating a cluster config")
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
		fakeClusterConfig, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

		By("creating an empty device controller")
		var noDevices []Device
		deviceController := NewDeviceController("master", 100, "rw", noDevices, fakeClusterConfig, clientTest.CoreV1())

		By("adding a host device to the cluster config")
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.HostDevicesGate}
		kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
			PciHostDevices: []v1.PciHostDevice{
				{
					PCIVendorSelector: fakeID,
					ResourceName:      fakeName,
				},
			},
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
		permittedDevices := fakeClusterConfig.GetPermittedHostDevices()
		Expect(permittedDevices).ToNot(BeNil(), "something went wrong while parsing the configmap(s)")
		Expect(permittedDevices.PciHostDevices).To(HaveLen(1), "the fake device was not found")

		By("ensuring a device plugin gets created for our fake device")
		enabledDevicePlugins, disabledDevicePlugins := deviceController.splitPermittedDevices(
			deviceController.updatePermittedHostDevicePlugins(),
		)
		Expect(enabledDevicePlugins).To(HaveLen(1), "a device plugin wasn't created for the fake device")
		Expect(disabledDevicePlugins).To(BeEmpty(), "no disabled device plugins are expected")
		Ω(enabledDevicePlugins).Should(HaveKey(fakeName))
		// Manually adding the enabled plugin, since the device controller is not actually running
		deviceController.startedPlugins[fakeName] = controlledDevice{devicePlugin: enabledDevicePlugins[fakeName]}

		By("deletting the device from the configmap")
		kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
		permittedDevices = fakeClusterConfig.GetPermittedHostDevices()
		Expect(permittedDevices).ToNot(BeNil(), "something went wrong while parsing the configmap(s)")
		Expect(permittedDevices.PciHostDevices).To(BeEmpty(), "the fake device was not deleted")

		By("ensuring the device plugin gets stopped")
		enabledDevicePlugins, disabledDevicePlugins = deviceController.splitPermittedDevices(
			deviceController.updatePermittedHostDevicePlugins(),
		)
		Expect(enabledDevicePlugins).To(BeEmpty(), "no enabled device plugins should be found")
		Expect(disabledDevicePlugins).To(HaveLen(1), "the fake device plugin did not get disabled")
		Ω(disabledDevicePlugins).Should(HaveKey(fakeName))
	})
})

// Allow the regular functions to be called for all the other devices, they're harmless.
// Just force the driver to NOT vfio-pci to ensure they all get ignored.
func allowAnyDeviceCalls(mockPCI MockDeviceHandler) {
	mockPCI.EXPECT().GetDeviceIOMMUGroup(pciBasePath, gomock.Any()).AnyTimes()
	mockPCI.EXPECT().GetDeviceDriver(pciBasePath, gomock.Any()).Return("definitely-not-vfio-pci", nil).AnyTimes()
	mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, gomock.Any()).AnyTimes()
	mockPCI.EXPECT().GetDevicePCIID(pciBasePath, gomock.Any()).AnyTimes()
}
