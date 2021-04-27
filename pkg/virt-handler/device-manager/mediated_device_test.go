package device_manager

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	fakeMdevNameSelector      = "FAKE 123"
	fakeIntelMdevNameSelector = "i915-GVTg_V5_4"
	fakeMdevResourceName      = "example.org/fake123"
	fakeMdevUUID              = "53764d0e-85a0-42b4-af5c-2046b460b1dc"
	fakeIntelMdevUUID         = "54444d0e-85a0-42b4-af5c-2046b4bbb1aa"
)

var _ = Describe("Mediated Device", func() {
	var mockPCI *MockDeviceHandler
	var fakePermittedHostDevicesConfig string
	var fakePermittedHostDevices v1.PermittedHostDevices
	var ctrl *gomock.Controller
	var fakeSupportedTypesPath string
	resourceNameToTypeName := func(rawName string) string {
		typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
		typeNameStr = strings.TrimSpace(typeNameStr)
		return typeNameStr
	}
	BeforeEach(func() {
		By("creating a temporary fake mdev directory tree")
		// create base mdev dir instead of /sys/bus/mdev/devices
		fakeMdevBasePath, err := ioutil.TempDir("/tmp", "mdevs")
		Expect(err).ToNot(HaveOccurred())
		// create an alternative mdev_supported_types dir instead of /sys/class/mdev_bus/[pciAddress]/
		fakeSupportedTypesPath, err = ioutil.TempDir("/tmp", "mdev_supported_types")
		Expect(err).ToNot(HaveOccurred())
		// create a fake path to nvidia mdev type
		fakeNvidiaTypePath := filepath.Join(fakeSupportedTypesPath, "nvidia-222")
		err = os.MkdirAll(fakeNvidiaTypePath, 0700)
		Expect(err).ToNot(HaveOccurred())
		// create a fake path to Intel mdev type
		fakeIntelTypePath := filepath.Join(fakeSupportedTypesPath, fakeIntelMdevNameSelector)
		err = os.MkdirAll(fakeIntelTypePath, 0700)
		Expect(err).ToNot(HaveOccurred())
		mdevBasePath = fakeMdevBasePath
		// create mdev directories and symlinks
		for _, uuid := range []string{fakeMdevUUID, fakeIntelMdevUUID} {
			mdevTypePath := filepath.Join(fakeMdevBasePath, uuid+"real")
			err = os.MkdirAll(mdevTypePath, 0700)
			Expect(err).ToNot(HaveOccurred())
			err = os.Symlink(filepath.Join(fakeMdevBasePath, uuid+"real"), filepath.Join(fakeMdevBasePath, uuid))
			Expect(err).ToNot(HaveOccurred())
		}
		// link nvidia type directory
		err = os.Symlink(fakeNvidiaTypePath, filepath.Join(fakeMdevBasePath, fakeMdevUUID+"real", "mdev_type"))
		Expect(err).ToNot(HaveOccurred())
		err = os.Symlink(fakeIntelTypePath, filepath.Join(fakeMdevBasePath, fakeIntelMdevUUID+"real", "mdev_type"))
		Expect(err).ToNot(HaveOccurred())
		// create a name file in the nvidia type directory
		mdevName, err := os.Create(filepath.Join(fakeNvidiaTypePath, "name"))
		Expect(err).ToNot(HaveOccurred())
		mdevNameWriter := bufio.NewWriter(mdevName)
		n, err := mdevNameWriter.WriteString(fakeMdevNameSelector + "\n")
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(len(fakeMdevNameSelector) + 1))
		mdevNameWriter.Flush()

	})

	AfterEach(func() {
		os.RemoveAll(mdevBasePath)
		os.RemoveAll(fakeSupportedTypesPath)
	})
	table.DescribeTable("should get correct file type name", func(namePathExist bool) {
		if namePathExist {
			mdevName, err := getMdevTypeName(fakeMdevUUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(mdevName).To(Equal(resourceNameToTypeName(fakeMdevNameSelector)))
		} else {
			mdevName, err := getMdevTypeName(fakeIntelMdevUUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(mdevName).To(Equal(resourceNameToTypeName(fakeIntelMdevNameSelector)))
		}
	},
		table.Entry("Nvidia name file exist", true),
		table.Entry("Intel name file doesn't exist", false),
	)
	Context("discover devices", func() {
		BeforeEach(func() {
			By("mocking PCI and MDEV functions to simulate an mdev an its parent PCI device")
			ctrl = gomock.NewController(GinkgoT())
			mockPCI = NewMockDeviceHandler(ctrl)
			Handler = mockPCI
			// Force pre-defined returned values and ensure the function only get called exacly once each on 0000:00:00.0
			mockPCI.EXPECT().GetMdevParentPCIAddr(fakeMdevUUID).Return(fakeAddress, nil).Times(1)
			mockPCI.EXPECT().GetDeviceIOMMUGroup(mdevBasePath, fakeMdevUUID).Return(fakeIommuGroup, nil).Times(1)
			mockPCI.EXPECT().GetDeviceNumaNode(pciBasePath, fakeAddress).Return(fakeNumaNode).Times(1)

			By("creating a list of fake device using the yaml decoder")
			fakePermittedHostDevicesConfig = `
    mediatedDevices:
    - mdevNameSelector: "` + fakeMdevNameSelector + `"
      resourceName: "` + fakeMdevResourceName + `"
    `
			err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(fakePermittedHostDevicesConfig), 1024).Decode(&fakePermittedHostDevices)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(fakePermittedHostDevices.MediatedDevices)).To(Equal(1))
			Expect(fakePermittedHostDevices.MediatedDevices[0].MDEVNameSelector).To(Equal(fakeMdevNameSelector))
			Expect(fakePermittedHostDevices.MediatedDevices[0].ResourceName).To(Equal(fakeMdevResourceName))
		})
		AfterEach(func() {
			ctrl.Finish()
		})
		It("Should parse the permitted devices and find 1 matching mediated device", func() {
			supportedMdevsMap := make(map[string]string)
			for _, supportedMdev := range fakePermittedHostDevices.MediatedDevices {
				// do not add a device plugin for this resource if it's being provided via an external device plugin
				if !supportedMdev.ExternalResourceProvider {
					selector := removeSelectorSpaces(supportedMdev.MDEVNameSelector)
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

		It("Should validate DPI devices", func() {
			iommuToMDEVMap := make(map[string]string)
			supportedMdevsMap := make(map[string]string)
			for _, supportedMdev := range fakePermittedHostDevices.MediatedDevices {
				// do not add a device plugin for this resource if it's being provided via an external device plugin
				if !supportedMdev.ExternalResourceProvider {
					selector := removeSelectorSpaces(supportedMdev.MDEVNameSelector)
					supportedMdevsMap[selector] = supportedMdev.ResourceName
				}
			}
			// discoverPermittedHostMediatedDevices() will walk real mdev devices wherever the tests are running
			mDevices := discoverPermittedHostMediatedDevices(supportedMdevsMap)
			selector := removeSelectorSpaces(fakeMdevNameSelector)
			devs := constructDPIdevicesFromMdev(mDevices[selector], iommuToMDEVMap)
			Expect(devs[0].ID).To(Equal(fakeIommuGroup))
			Expect(devs[0].Topology.Nodes[0].ID).To(Equal(int64(fakeNumaNode)))
		})

		It("Should update the device list according to the configmap", func() {
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
			fakeClusterConfig, _, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)

			By("creating an empty device controller")
			deviceController := NewDeviceController("master", 10, "rw", fakeClusterConfig)
			deviceController.devicePlugins = make(map[string]ControlledDevice)

			By("adding a host device to the cluster config")
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{virtconfig.HostDevicesGate}
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{
				MediatedDevices: []v1.MediatedHostDevice{
					{
						MDEVNameSelector: fakeMdevNameSelector,
						ResourceName:     fakeMdevResourceName,
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
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
			kvConfig.Spec.Configuration.PermittedHostDevices = &v1.PermittedHostDevices{}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
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
})
