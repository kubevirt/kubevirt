package lookup

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var createVirtualMachineInstance = func(name, nodeName string, phase virtv1.VirtualMachineInstancePhase) *virtv1.VirtualMachineInstance {
	return &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				virtv1.NodeNameLabel: nodeName,
			},
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			Phase: phase,
		},
	}
}

var _ = Describe("Lookup", func() {

	var virtClient *kubecli.MockKubevirtClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
	})

	It("should return all vmis on a node", func() {
		vmi1 := createVirtualMachineInstance("vmi1", "node01", virtv1.Running)
		vmi2 := createVirtualMachineInstance("vmi2", "node01", virtv1.Failed)
		vmis := []virtv1.VirtualMachineInstance{*vmi1, *vmi2}

		vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{
			Items: vmis,
		}, nil)

		returnedVMIs, err := VirtualMachinesOnNode(virtClient, "node01")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(returnedVMIs)).To(Equal(2))
	})

	It("should return active vmis on a node", func() {
		vmi1 := createVirtualMachineInstance("vmi1", "node01", virtv1.Running)
		vmi2 := createVirtualMachineInstance("vmi2", "node01", virtv1.Failed)
		vmis := []virtv1.VirtualMachineInstance{*vmi1, *vmi2}

		vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{
			Items: vmis,
		}, nil)

		returnedVMIs, err := ActiveVirtualMachinesOnNode(virtClient, "node01")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(returnedVMIs)).To(Equal(1))
		Expect(returnedVMIs[0].Status.Phase).To(Equal(virtv1.Running))
	})

	table.DescribeTable("should filter out nonactive vmis", func(phase virtv1.VirtualMachineInstancePhase) {
		vmi := createVirtualMachineInstance("vmi2", "node01", phase)

		vmiInterface.EXPECT().List(gomock.Any()).Return(&virtv1.VirtualMachineInstanceList{
			Items: []virtv1.VirtualMachineInstance{*vmi},
		}, nil)

		vmis, err := ActiveVirtualMachinesOnNode(virtClient, "node01")
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis).To(BeEmpty())
	},
		table.Entry("unprocessed state", virtv1.VmPhaseUnset),
		table.Entry("pending state", virtv1.Pending),
		table.Entry("scheduling state", virtv1.Scheduling),
		table.Entry("failed state", virtv1.Failed),
		table.Entry("succeeded state", virtv1.Succeeded),
	)
})
