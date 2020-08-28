package device_manager

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
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
	fakeMdevNameSelector = "FAKE 123"
	fakeMdevResourceName = "example.org/fake123"
	fakeMdevUUID         = "53764d0e-85a0-42b4-af5c-2046b460b1dc"
)

var _ = Describe("Mediated Device", func() {
	var mockPCI *MockDeviceHandler
	var fakePermittedHostDevicesConfig string
	var fakePermittedHostDevices v1.PermittedHostDevices
	var ctrl *gomock.Controller

	BeforeEach(func() {
		By("creating a temporary fake mdev directory tree")
		fakeMdevBasePath, err := ioutil.TempDir("/tmp", "mdevs")
		mdevBasePath = fakeMdevBasePath
		mdevTypePath := filepath.Join(fakeMdevBasePath, fakeMdevUUID+"real", "mdev_type")
		err = os.MkdirAll(mdevTypePath, 0700)
		Expect(err).ToNot(HaveOccurred())
		err = os.Symlink(filepath.Join(fakeMdevBasePath, fakeMdevUUID+"real"), filepath.Join(fakeMdevBasePath, fakeMdevUUID))
		Expect(err).ToNot(HaveOccurred())
		mdevName, err := os.Create(filepath.Join(mdevTypePath, "name"))
		Expect(err).ToNot(HaveOccurred())
		mdevNameWriter := bufio.NewWriter(mdevName)
		n, err := mdevNameWriter.WriteString(fakeMdevNameSelector + "\n")
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(len(fakeMdevNameSelector) + 1))
		mdevNameWriter.Flush()

		By("mocking PCI and MDEV functions to simulate an mdev an its parent PCI device")
		ctrl = gomock.NewController(GinkgoT())
		mockPCI = NewMockDeviceHandler(ctrl)
		Handler = mockPCI
		// Force pre-defined returned values and ensure the function only get called exacly once each on 0000:00:00.0
		mockPCI.EXPECT().GetMdevParentPCIAddr(fakeMdevUUID).Return(fakeAddress, nil).Times(1)
		mockPCI.EXPECT().GetDeviceIOMMUGroup(fakeMdevBasePath, fakeMdevUUID).Return(fakeIommuGroup, nil).Times(1)
		mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, fakeAddress).Return(fakeNumaNode).Times(1)

		By("creating a list of fake device using the yaml decoder")
		fakePermittedHostDevicesConfig = `
mdevs:
- mdevNameSelector: "` + fakeMdevNameSelector + `"
  resourceName: "` + fakeMdevResourceName + `"
`
		err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(fakePermittedHostDevicesConfig), 1024).Decode(&fakePermittedHostDevices)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(fakePermittedHostDevices.MediatedDevices)).To(Equal(1))
		Expect(fakePermittedHostDevices.MediatedDevices[0].Selector).To(Equal(fakeMdevNameSelector))
		Expect(fakePermittedHostDevices.MediatedDevices[0].ResourceName).To(Equal(fakeMdevResourceName))
	})

	AfterEach(func() {
		ctrl.Finish()
		os.RemoveAll(mdevBasePath)
	})

	It("Should parse the permitted devices and find 1 matching mediated device", func() {
		supportedMdevsMap := make(map[string]string)
		for _, supportedMdev := range fakePermittedHostDevices.MediatedDevices {
			// do not add a device plugin for this resource if it's being provided via an external device plugin
			if !supportedMdev.ExternalResourceProvider {
				selector := removeSelectorSpaces(supportedMdev.Selector)
				supportedMdevsMap[selector] = supportedMdev.ResourceName
			}
		}
		// discoverPermittedHostMediatedDevices() will walk real mdev devices wherever the tests are running
		devices := discoverPermittedHostMediatedDevices(supportedMdevsMap)
		Expect(len(devices)).To(Equal(1))
		selector := removeSelectorSpaces(fakeMdevNameSelector)
		Expect(len(devices[selector])).To(Equal(1))
		Expect(devices[selector][0].UUID).To(Equal(fakeMdevUUID))
		Expect(devices[selector][0].typeName).To(Equal(selector))
		Expect(devices[selector][0].parentPciAddress).To(Equal(fakeAddress))
		Expect(devices[selector][0].iommuGroup).To(Equal(fakeIommuGroup))
		Expect(devices[selector][0].numaNode).To(Equal(fakeNumaNode))
	})

	It("Should update the device list according to the configmap", func() {
		By("creating a cluster config")
		configMapData := make(map[string]string)
		configMapData[virtconfig.PermittedHostDevicesKey] = fakePermittedHostDevicesConfig
		fakeClusterConfig, fakeInformer, _, _, fakeHostDevInformer := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

		By("creating an empty device controller")
		deviceController := NewDeviceController("master", 10, fakeClusterConfig, fakeInformer)
		deviceController.devicePlugins = make(map[string]ControlledDevice)

		By("adding a host device to the cluster config")
		testutils.UpdateFakeClusterConfigByName(fakeHostDevInformer, &k8sv1.ConfigMap{
			Data: configMapData,
		}, testutils.HostDevicesConfigMapName)
		permittedDevices := fakeClusterConfig.GetPermittedHostDevices()
		Expect(permittedDevices).ToNot(BeNil(), "something went wrong while parsing the configmap(s)")
		Expect(len(permittedDevices.MediatedDevices)).To(Equal(1), "the fake device was not found")

		By("ensuring a device plugin gets created for our fake device")
		enabledDevicePlugins, disabledDevicePlugins := deviceController.updatePermittedHostDevicePlugins()
		Expect(len(enabledDevicePlugins)).To(Equal(1), "a device plugin wasn't created for the fake device")
		Expect(len(disabledDevicePlugins)).To(Equal(0))
		Ω(enabledDevicePlugins).Should(HaveKey(fakeMdevResourceName))
		// Manually adding the enabled plugin, since the device controller is not actually running
		deviceController.devicePlugins[fakeMdevResourceName] = enabledDevicePlugins[fakeMdevResourceName]

		By("deletting the device from the configmap")
		fakePermittedHostDevicesConfig = ""
		configMapData[virtconfig.PermittedHostDevicesKey] = fakePermittedHostDevicesConfig
		testutils.UpdateFakeClusterConfigByName(fakeHostDevInformer, &k8sv1.ConfigMap{
			Data: configMapData,
		}, testutils.HostDevicesConfigMapName)
		permittedDevices = fakeClusterConfig.GetPermittedHostDevices()
		Expect(permittedDevices).ToNot(BeNil(), "something went wrong while parsing the configmap(s)")
		Expect(len(permittedDevices.MediatedDevices)).To(Equal(0), "the fake device was not deleted")

		By("ensuring the device plugin gets stopped")
		enabledDevicePlugins, disabledDevicePlugins = deviceController.updatePermittedHostDevicePlugins()
		Expect(len(enabledDevicePlugins)).To(Equal(0))
		Expect(len(disabledDevicePlugins)).To(Equal(1), "the fake device plugin did not get disabled")
		Ω(disabledDevicePlugins).Should(HaveKey(fakeMdevResourceName))
	})
})
