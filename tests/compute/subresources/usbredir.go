package subresources

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmops"
)

// Only checks for the default which is non configured usbredir.
// The functest for configured usbredir is under tests/virtctl/usbredir.go
var _ = Describe(compute.SIG("usbredir support", func() {

	const enoughMemForSafeBiosEmulation = "32Mi"
	var (
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		vmi = libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
	})

	It("should fail to connect to VMI's usbredir socket", func() {
		usbredirVMI, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).USBRedir(vmi.ObjectMeta.Name)
		Expect(err).To(HaveOccurred())
		Expect(usbredirVMI).To(BeNil())
	})
}))
