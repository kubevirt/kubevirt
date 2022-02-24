package tests_test

import (
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
)

var _ = Describe("[Serial][sig-compute]HostDevices", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("with ephemeral disk", func() {
		It("Should successfully passthrough an emulated PCI device", func() {
			deviceName := "example.org/soundcard"
			deviceIDs := "8086:2668"
			kv := util.GetCurrentKv(virtClient)

			By("Adding the emulated sound card to the permitted host devices")
			config := kv.Spec.Configuration
			config.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{virtconfig.HostDevicesGate},
				DiskVerification: &v1.DiskVerification{
					MemoryLimit: resource.NewScaledQuantity(2, resource.Giga),
				},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{
				PciHostDevices: []v1.PciHostDevice{
					{
						PCIVendorSelector: deviceIDs,
						ResourceName:      deviceName,
					},
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Creating a Fedora VMI with the sound card as a host device")
			randomVMI := tests.NewRandomFedoraVMIWithGuestAgent()
			hostDevs := []v1.HostDevice{
				{
					Name:       "sound",
					DeviceName: deviceName,
				},
			}
			randomVMI.Spec.Domain.Devices.HostDevices = hostDevs
			vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(randomVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Making sure the sound card is present inside the VMI")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "grep -c " + strings.Replace(deviceIDs, ":", "", 1) + " /proc/bus/pci/devices\n"},
				&expect.BExp{R: console.RetValue("1")},
			}, 15)).To(Succeed(), "Device not found")
		})
	})
})
