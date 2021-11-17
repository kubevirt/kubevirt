package webhooks

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Webhook", func() {

	var ctrl *gomock.Controller
	var admitter *KubeVirtDeletionAdmitter

	var kubeCli *kubecli.MockKubevirtClient
	var vmirsInterface *kubecli.MockReplicaSetInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var kvInterface *kubecli.MockKubeVirtInterface
	var kv *k6tv1.KubeVirt

	BeforeEach(func() {
		kv = &k6tv1.KubeVirt{}
		kv.Status.Phase = k6tv1.KubeVirtPhaseDeployed
		ctrl = gomock.NewController(GinkgoT())
		kubeCli = kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		kvInterface.EXPECT().Get("kubevirt", gomock.Any()).Return(kv, nil).AnyTimes()
		kvInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.KubeVirtList{}, nil).AnyTimes()
		admitter = &KubeVirtDeletionAdmitter{kubeCli}
		kubeCli.EXPECT().KubeVirt("test").Return(kvInterface)

		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmirsInterface = kubecli.NewMockReplicaSetInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("if uninstall strategy is BlockUninstallIfWorkloadExists", func() {
		BeforeEach(func() {
			kubeCli.EXPECT().VirtualMachineInstance(v1.NamespaceAll).Return(vmiInterface).AnyTimes()
			kubeCli.EXPECT().ReplicaSet(v1.NamespaceAll).Return(vmirsInterface).AnyTimes()
			kubeCli.EXPECT().VirtualMachine(v1.NamespaceAll).Return(vmInterface).AnyTimes()

			kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
		})

		It("should allow the deletion if no workload exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{}, nil)

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeTrue())
		})

		It("should deny the deletion if a VMI exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{Items: []k6tv1.VirtualMachineInstance{{}}}, nil)

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VM exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{Items: []k6tv1.VirtualMachine{{}}}, nil)

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VMIRS exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{Items: []k6tv1.VirtualMachineInstanceReplicaSet{{}}}, nil)

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIs fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMs fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIRS fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})
	})

	It("should allow the deletion if the strategy is empty", func() {
		kv.Spec.UninstallStrategy = ""
		response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion if the strategy is set to RemoveWorkloads", func() {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion of namespaces, where it gets an admission request without a resource name", func() {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: ""}})
		Expect(response.Allowed).To(BeTrue())
	})

	table.DescribeTable("should not check for workloads if kubevirt phase is", func(phase k6tv1.KubeVirtPhase) {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
		kv.Status.Phase = phase
		response := admitter.Admit(&admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	},
		table.Entry("unset", k6tv1.KubeVirtPhase("")),
		table.Entry("deploying", k6tv1.KubeVirtPhaseDeploying),
		table.Entry("deleting", k6tv1.KubeVirtPhaseDeleting),
		table.Entry("deleted", k6tv1.KubeVirtPhaseDeleted),
	)
})
