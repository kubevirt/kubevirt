package vm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	webhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Instance type and Preference VirtualMachine Admitter", func() {
	var virtClient *kubecli.MockKubevirtClient
	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fake.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			fake.NewSimpleClientset().KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstancetype(k8sv1.NamespaceDefault).Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineInstancetypes(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()

		virtClient.EXPECT().VirtualMachinePreference(k8sv1.NamespaceDefault).Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()
	})
	Context("Given a VirtualMachine", func() {
		DescribeTable("should admit when", func(vm *virtv1.VirtualMachine) {
			admitter := webhooks.NewAdmitter(virtClient)
			instancetypeSpec, preferenceSpec, causes := admitter.ApplyToVM(vm)
			Expect(instancetypeSpec).To(BeNil())
			Expect(preferenceSpec).To(BeNil())
			Expect(causes).To(BeNil())
		},
			Entry("VirtualMachineInstancetype is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithInstancetype("unknown"),
				),
			),
			Entry("VirtualMachineClusterInstancetype is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithClusterInstancetype("unknown"),
				),
			),
			Entry("VirtualMachinePreference is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithPreference("unknown"),
				),
			),
			Entry("VirtualMachineClusterPreference is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithClusterPreference("unknown"),
				),
			),
		)
	})
})
