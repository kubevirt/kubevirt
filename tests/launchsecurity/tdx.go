package launchsecurity

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[sig-compute]Intel TDX", decorators.TDX, decorators.SigCompute, func() {
	// I could use embedded data but it's not worth the effort at the moment
	const cloudInitUserData = "#cloud-config\n" +
		"user: fedora\n" +
		"password: fedora\n" +
		"chpasswd: { expire: False }\n" +
		"ssh_pwauth: true\n"

	newTDXFedora := func() *v1.VirtualMachineInstance {
		// Configure libvirt to use TDX
		// Use fedora image with support for TDX
		// I had some issues with just using NewFedora()
		tdxOptions := []libvmi.Option{
			libvmi.WithTDX(true),
			libvmi.WithMemoryRequest("2Gi"),
			libvmi.WithContainerDisk("rootdisk1", "quay.io/containerdisks/fedora"),
			libvmi.WithUefi(false),
			// Set the credentials for LoginToFedora()
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(cloudInitUserData)),
		}

		vmi := libvmi.New(tdxOptions...)

		vmi.Spec.Domain.CPU = &v1.CPU{
			Model:   "host-passthrough",
			Cores:   2,
			Sockets: 1,
			Threads: 1,
		}

		vmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(true)
		vmi.Spec.Domain.Devices.DisableHotplug = true

		return vmi
	}

	FContext("lifecycle", func() {

		var (
			virtClient kubecli.KubevirtClient
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()
		})

		It("should verify TDX guest execution with proper logging", func() {
			By("Checking if we have a valid virt client")
			Expect(virtClient).ToNot(BeNil())

			By("Creating VMI definition")

			vmi := newTDXFedora()

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

			// TODO: login is sucessful, but LoginToFedora is failing
			// ignore the result of the login for the moment
			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying that TDX is enabled in the guest")

			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sudo dmesg | grep --color=never tdx\n"},
				&expect.BExp{R: "tdx: Guest detected"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 30)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
