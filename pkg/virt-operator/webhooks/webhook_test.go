package webhooks

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

var _ = Describe("Webhook", func() {
	var admitter *KubeVirtDeletionAdmitter
	var fakeClient *kubevirtfake.Clientset
	var vmirsInterface *kubecli.MockReplicaSetInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "test",
			},
			Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed},
		}
		fakeClient = kubevirtfake.NewSimpleClientset(kv)
		kubeCli := kubecli.NewMockKubevirtClient(ctrl)
		admitter = &KubeVirtDeletionAdmitter{kubeCli}
		kubeCli.
			EXPECT().
			KubeVirt("test").
			Return(fakeClient.KubevirtV1().KubeVirts("test")).
			AnyTimes()

		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmirsInterface = kubecli.NewMockReplicaSetInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubeCli.EXPECT().VirtualMachineInstance(k8sv1.NamespaceAll).Return(vmiInterface).AnyTimes()
		kubeCli.EXPECT().ReplicaSet(k8sv1.NamespaceAll).Return(vmirsInterface).AnyTimes()
		kubeCli.EXPECT().VirtualMachine(k8sv1.NamespaceAll).Return(vmInterface).AnyTimes()
	})

	Context("if uninstall strategy is BlockUninstallIfWorkloadExists", func() {
		BeforeEach(func() {
			setKV(fakeClient, v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist, v1.KubeVirtPhaseDeployed)
		})

		It("should allow the deletion if no workload exists", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceReplicaSetList{}, nil)

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeTrue())
		})

		It("should deny the deletion if a VMI exists", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{Items: []v1.VirtualMachineInstance{{}}}, nil)

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VM exists", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineList{Items: []v1.VirtualMachine{{}}}, nil)

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VMIRS exists", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceReplicaSetList{Items: []v1.VirtualMachineInstanceReplicaSet{{}}}, nil)

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIs fails", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMs fails", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIRS fails", func() {
			vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceReplicaSetList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})
	})

	It("should allow the deletion if the strategy is empty", func() {
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion if the strategy is set to RemoveWorkloads", func() {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyRemoveWorkloads, v1.KubeVirtPhaseDeployed)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion of namespaces, where it gets an admission request without a resource name", func() {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyRemoveWorkloads, v1.KubeVirtPhaseDeployed)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: ""}})
		Expect(response.Allowed).To(BeTrue())
	})

	DescribeTable("should not check for workloads if kubevirt phase is", func(phase v1.KubeVirtPhase) {
		setKV(fakeClient, v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist, phase)
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	},
		Entry("unset", v1.KubeVirtPhase("")),
		Entry("deploying", v1.KubeVirtPhaseDeploying),
		Entry("deleting", v1.KubeVirtPhaseDeleting),
		Entry("deleted", v1.KubeVirtPhaseDeleted),
	)
})

func setKV(fakeClient *kubevirtfake.Clientset, strategy v1.KubeVirtUninstallStrategy, phase v1.KubeVirtPhase) {
	patchBytes, err := patch.New(
		patch.WithReplace("/spec/uninstallStrategy", strategy),
		patch.WithReplace("/status/phase", phase),
	).GeneratePayload()
	Expect(err).NotTo(HaveOccurred())
	_, err = fakeClient.KubevirtV1().KubeVirts("test").Patch(context.TODO(), "kubevirt", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}
