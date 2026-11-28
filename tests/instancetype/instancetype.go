//nolint:lll
package instancetype

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const timeout = 300 * time.Second

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences", decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("instance type with dedicatedCPUPlacement enabled", decorators.RequiresNodeWithCPUManager, func() {
		It("should be accepted and result in running VirtualMachineInstance", func() {
			vmi := libvmifact.NewGuestless()

			clusterInstancetype := builder.NewClusterInstancetypeFromVMI(vmi)
			clusterInstancetype.Spec.CPU.DedicatedCPUPlacement = pointer.P(true)
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithClusterInstancetype(clusterInstancetype.Name), libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that DedicatedCPUPlacement is not set in the VM but enabled in the VMI through the instance type
			Expect(vm.Spec.Template.Spec.Domain.CPU).To(BeNil())
			Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(BeTrue())
		})
	})
})
