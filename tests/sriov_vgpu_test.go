package tests_test

import (
	"context"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]SR-IOV vGPU", Serial, decorators.VGPU, decorators.SigCompute, func() {
	const deviceName = "nvidia.com/L40S_SRIOV"
	var virtClient kubecli.KubevirtClient
	var originalConfig v1.KubeVirtConfiguration
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalConfig = *libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()

		config := *originalConfig.DeepCopy()
		if config.PermittedHostDevices == nil {
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
		}
		configured := false
		for _, device := range config.PermittedHostDevices.PciHostDevices {
			if device.ResourceName == deviceName && device.PCIVendorSelector == "10DE:26B9" {
				configured = true
			}
		}
		if !configured {
			config.PermittedHostDevices.PciHostDevices = append(config.PermittedHostDevices.PciHostDevices, v1.PciHostDevice{
				PCIVendorSelector: "10DE:26B9",
				ResourceName:      deviceName,
			})
		}
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)
	})

	AfterEach(func() {
		if vmi != nil {
			Expect(virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())
			Expect(libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180*time.Second)).To(Succeed())
		}
		kvconfig.UpdateKubeVirtConfigValueAndWait(originalConfig)
	})

	It("Should successfully passthrough an SR-IOV vGPU", func() {
		By("Creating a Fedora VMI with a synthetic SR-IOV VF")
		candidate := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithNodeSelector("vgpu_sriov_capable", "true"))
		candidate.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu1", DeviceName: deviceName}}
		createdVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(candidate)).Create(context.Background(), candidate, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = createdVMI
		libwait.WaitForSuccessfulVMIStart(vmi)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		By("Checking that the VF is visible in the guest")
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "grep -c 10de26b9 /proc/bus/pci/devices\n"},
			&expect.BExp{R: console.RetValue("1")},
		}, 250)).To(Succeed(), "SR-IOV VF not found")
	})
})
