package tests_test

import (
	"context"
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	failedDeleteVMI = "Failed to delete VMI"
)

var _ = Describe("[Serial][sig-compute]HostDevices", Serial, decorators.SigCompute, func() {
	var (
		virtClient kubecli.KubevirtClient
		config     v1.KubeVirtConfiguration
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := util.GetCurrentKv(virtClient)
		config = kv.Spec.Configuration
	})

	AfterEach(func() {
		kv := util.GetCurrentKv(virtClient)
		// Reinitialized the DeveloperConfiguration to avoid to influence the next test
		config = kv.Spec.Configuration
		config.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		config.PermittedHostDevices = &v1.PermittedHostDevices{}
		tests.UpdateKubeVirtConfigValueAndWait(config)
	})

	Context("with ephemeral disk", func() {
		DescribeTable("with emulated PCI devices", func(deviceIDs []string) {
			deviceName := "example.org/soundcard"

			By("Adding the emulated sound card to the permitted host devices")
			config.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{virtconfig.HostDevicesGate},
				DiskVerification: &v1.DiskVerification{
					MemoryLimit: resource.NewScaledQuantity(2, resource.Giga),
				},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
			var hostDevs []v1.HostDevice
			for i, id := range deviceIDs {
				config.PermittedHostDevices.PciHostDevices = append(config.PermittedHostDevices.PciHostDevices, v1.PciHostDevice{
					PCIVendorSelector: id,
					ResourceName:      deviceName,
				})
				hostDevs = append(hostDevs, v1.HostDevice{
					Name:       fmt.Sprintf("sound%d", i),
					DeviceName: deviceName,
				})
			}
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Creating a Fedora VMI with the sound card as a host device")
			randomVMI := tests.NewRandomFedoraVMI()
			randomVMI.Spec.Domain.Devices.HostDevices = hostDevs
			vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), randomVMI)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Making sure the sound card is present inside the VMI")
			for _, id := range deviceIDs {
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c " + strings.Replace(id, ":", "", 1) + " /proc/bus/pci/devices\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 15)).To(Succeed(), "Device not found")
			}
			// Make sure to delete the VMI before ending the test otherwise a device could still be taken
			err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
		},
			Entry("Should successfully passthrough an emulated PCI device", []string{"8086:2668"}),
			Entry("Should successfully passthrough 2 emulated PCI devices", []string{"8086:2668", "8086:2415"}),
		)
	})
})
