package vm_test

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Evacuate cancel command", func() {
	var (
		ctrl         *gomock.Controller
		vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		vmInterface  *kubecli.MockVirtualMachineInterface
		kubeClient   *k8sfake.Clientset
		virtClient   *kubecli.MockKubevirtClient
	)

	const (
		vmName  = "testvm"
		vmiName = "testvmi"
		node    = "node01"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance = virtClient
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig

		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		kubeClient = k8sfake.NewClientset(&corev1.Node{
			TypeMeta:   k8smetav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: k8smetav1.ObjectMeta{Name: node},
		})

		virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
	})

	It("should fail with missing arguments", func() {
		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("accepts 2 arg(s), received 0"))
	})

	It("should fail on unsupported kind", func() {
		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "pod", "my-pod")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(`unsupported resource type "pod"`))
	})

	It("should cancel evacuation for VM", func() {
		vmInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmName, &v1.EvacuateCancelOptions{}).
			Return(nil).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vm", vmName)
		err := cmd()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return error on VM evacuate cancel failure", func() {
		expectedErr := fmt.Errorf("failure on VM")
		vmInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmName, &v1.EvacuateCancelOptions{}).
			Return(expectedErr).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vm", vmName)
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(expectedErr))
	})

	It("should cancel evacuation for VMI", func() {
		vmiInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmiName, &v1.EvacuateCancelOptions{}).
			Return(nil).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "vmi", vmiName)
		err := cmd()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should cancel evacuation for VMIs on a node", func() {
		vmiList := &v1.VirtualMachineInstanceList{
			Items: []v1.VirtualMachineInstance{
				{
					TypeMeta:   k8smetav1.TypeMeta{Kind: "VirtualMachineInstance", APIVersion: "v1"},
					ObjectMeta: k8smetav1.ObjectMeta{Name: "vmi1", Namespace: "default"},
					Status:     v1.VirtualMachineInstanceStatus{EvacuationNodeName: node},
				},
				{
					TypeMeta:   k8smetav1.TypeMeta{Kind: "VirtualMachineInstance", APIVersion: "v1"},
					ObjectMeta: k8smetav1.ObjectMeta{Name: "vmi2", Namespace: "default"},
					Status:     v1.VirtualMachineInstanceStatus{EvacuationNodeName: "othernode"},
				},
			},
		}

		vmiInterface.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(vmiList, nil).
			Times(1)

		vmiInterface.EXPECT().
			EvacuateCancel(gomock.Any(), "vmi1", gomock.Any()).
			Return(nil).
			Times(1)

		cmd := testing.NewRepeatableVirtctlCommand("evacuate-cancel", "node", node)
		err := cmd()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should print dry-run message", func() {
		cmd := testing.NewRepeatableVirtctlCommandWithOut("evacuate-cancel", "vmi", vmiName, "--dry-run")
		vmiInterface.EXPECT().
			EvacuateCancel(gomock.Any(), vmiName, &v1.EvacuateCancelOptions{
				DryRun: []string{k8smetav1.DryRunAll},
			}).Return(nil)

		bytes, err := cmd()
		Expect(err).ToNot(HaveOccurred())
		Expect(string(bytes)).To(ContainSubstring("Dry Run execution"))
		Expect(string(bytes)).To(ContainSubstring(fmt.Sprintf("VMI %s/%s was canceled evacuation", k8smetav1.NamespaceDefault, vmiName)))
	})
})
