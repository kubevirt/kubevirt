//nolint:lll,gomnd
package instancetype

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype memory overcommit", Serial, decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("should apply memory overcommit instancetype to VMI even with cluster overcommit set", func() {
		kv := libkubevirt.GetCurrentKv(virtClient)

		config := kv.Spec.Configuration
		config.DeveloperConfiguration.MemoryOvercommit = 200
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)

		// Use an Alpine VMI so we have enough memory in the eventual instance type and launched VMI to get past validation checks
		vmi := libvmifact.NewAlpine()

		instancetype := builder.NewInstancetypeFromVMI(vmi)
		instancetype.Spec.Memory.OvercommitPercent = 15

		instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
			Create(context.Background(), instancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Remove any requested resources from the VMI before generating the VM
		vm := libvmi.NewVirtualMachine(vmi,
			libvmi.WithInstancetype(instancetype.Name),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))
		expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
		Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
		memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
		Expect(memRequest.Value()).To(Equal(expectedOverhead))
	})

	It("should apply memory overcommit instancetype to VMI", func() {
		// Use an Alpine VMI so we have enough memory in the eventual instance type and launched VMI to get past validation checks
		vmi := libvmifact.NewAlpine()
		instancetype := builder.NewInstancetypeFromVMI(vmi)
		instancetype.Spec.Memory.OvercommitPercent = 15

		instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
			Create(context.Background(), instancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm := libvmi.NewVirtualMachine(vmi,
			libvmi.WithInstancetype(instancetype.Name),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))

		expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
		Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
		memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
		Expect(memRequest.Value()).To(Equal(expectedOverhead))
	})
})
