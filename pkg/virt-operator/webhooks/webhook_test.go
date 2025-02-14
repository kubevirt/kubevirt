package webhooks

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Webhook", func() {
	var admitter *KubeVirtDeletionAdmitter
	var vmirsInterface *kubecli.MockReplicaSetInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var kvInterface *kubecli.MockKubeVirtInterface
	var kv *v1.KubeVirt

	BeforeEach(func() {
		kv = &v1.KubeVirt{}
		kv.Status.Phase = v1.KubeVirtPhaseDeployed
		ctrl := gomock.NewController(GinkgoT())
		kubeCli := kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		kvInterface.EXPECT().Get(context.Background(), "kubevirt", gomock.Any()).Return(kv, nil).AnyTimes()
		kvInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.KubeVirtList{}, nil).AnyTimes()
		admitter = &KubeVirtDeletionAdmitter{kubeCli}
		kubeCli.EXPECT().KubeVirt("test").Return(kvInterface)

		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmirsInterface = kubecli.NewMockReplicaSetInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubeCli.EXPECT().VirtualMachineInstance(k8sv1.NamespaceAll).Return(vmiInterface).AnyTimes()
		kubeCli.EXPECT().ReplicaSet(k8sv1.NamespaceAll).Return(vmirsInterface).AnyTimes()
		kubeCli.EXPECT().VirtualMachine(k8sv1.NamespaceAll).Return(vmInterface).AnyTimes()
	})

	Context("if uninstall strategy is BlockUninstallIfWorkloadExists", func() {
		BeforeEach(func() {
			kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
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
		kv.Spec.UninstallStrategy = ""
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion if the strategy is set to RemoveWorkloads", func() {
		kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion of namespaces, where it gets an admission request without a resource name", func() {
		kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: ""}})
		Expect(response.Allowed).To(BeTrue())
	})

	DescribeTable("should not check for workloads if kubevirt phase is", func(phase v1.KubeVirtPhase) {
		kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
		kv.Status.Phase = phase
		response := admitter.Admit(context.Background(), &admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	},
		Entry("unset", v1.KubeVirtPhase("")),
		Entry("deploying", v1.KubeVirtPhaseDeploying),
		Entry("deleting", v1.KubeVirtPhaseDeleting),
		Entry("deleted", v1.KubeVirtPhaseDeleted),
	)
})
