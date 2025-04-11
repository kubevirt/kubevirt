package network_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"kubevirt.io/client-go/kubecli"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/controller/network"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("network controller instance type and preference handler", func() {
	const (
		preferenceName = "preference"
	)

	type netHandler interface {
		ApplyInterfacePreferencesToVMI(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstanceSpec
	}

	var (
		virtClient *kubecli.MockKubevirtClient
		controller netHandler
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().VirtualMachinePreference(k8sv1.NamespaceDefault).Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		spec := v1beta1.VirtualMachinePreferenceSpec{
			Devices: &v1beta1.DevicePreferences{
				PreferredInterfaceModel: virtv1.VirtIO,
			},
		}

		preference := &v1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      preferenceName,
				Namespace: k8sv1.NamespaceDefault,
			},
			Spec: spec,
		}

		_, err := virtClient.VirtualMachinePreference(k8sv1.NamespaceDefault).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		clusterPreference := &v1beta1.VirtualMachineClusterPreference{
			ObjectMeta: metav1.ObjectMeta{
				Name: preferenceName,
			},
			Spec: spec,
		}
		_, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// We only care about asserting the specific logic within the handler so just pass a valid client without any stores for these tests
		controller = network.New(nil, nil, nil, virtClient)
	})

	DescribeTable("should apply interface preferences to provided spec", func(vmPreferenceOption func(*virtv1.VirtualMachine)) {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithNamespace(k8sv1.NamespaceDefault),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				libvmi.WithInterface(*virtv1.DefaultBridgeNetworkInterface()),
			),
			vmPreferenceOption,
		)
		// Ensure a model isn't set by DefaultBridgeNetworkInterface above
		Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces[0].Model).To(BeEmpty())

		spec := controller.ApplyInterfacePreferencesToVMI(vm, &vm.Spec.Template.Spec)
		Expect(spec.Domain.Devices.Interfaces[0].Model).To(Equal(virtv1.VirtIO))
	},
		Entry("using namespaced preference", libvmi.WithPreference(preferenceName)),
		Entry("using cluster preference", libvmi.WithClusterPreference(preferenceName)),
	)

	It("should return original spec when unable to find preference", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithNamespace(k8sv1.NamespaceDefault),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				libvmi.WithInterface(*virtv1.DefaultBridgeNetworkInterface()),
			),
			libvmi.WithPreference("unknown"),
		)
		spec := controller.ApplyInterfacePreferencesToVMI(vm, &vm.Spec.Template.Spec)
		Expect(*spec).To(Equal(vm.Spec.Template.Spec))
	})
})
