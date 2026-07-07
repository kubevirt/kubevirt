package envtest_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
)

var _ = Describe("Domain XML", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.WithFakeLibvirt())
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	It("should produce domain XML via the converter when a VMI reaches Scheduled", func() {
		By("creating a VM with RunStrategyAlways")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
			),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the domain XML to be captured")
		var domainName string
		Eventually(func() *framework.FakeDomain {
			vmi, err := f.VirtClient().VirtualMachineInstance("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return nil
			}
			domainName = vmi.Namespace + "_" + vmi.Name
			return f.FakeLibvirt().LookupDomain(domainName)
		}, 30*time.Second, 100*time.Millisecond).ShouldNot(BeNil())

		By("asserting the domain XML contains expected elements")
		domain := f.FakeLibvirt().LookupDomain(domainName)
		Expect(domain.XML).To(ContainSubstring("<memory"),
			"domain XML should contain a memory element from the converter")
		Expect(domain.XML).To(ContainSubstring("<devices>"),
			"domain XML should contain a devices element")
	})
})
