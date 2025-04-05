package vm

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("FirmwareController", func() {
	var (
		controller     *FirmwareController
		recorder       *record.FakeRecorder
		mockQueue      *testutils.MockWorkQueue[string]
		virtClient     *kubecli.MockKubevirtClient
		virtfakeClient *kubevirtfake.Clientset
		vmInformer     cache.SharedIndexInformer
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtfakeClient = kubevirtfake.NewSimpleClientset()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(virtfakeClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		virtfakeClient.PrependReactor("patch", "virtualmachines", PatchReactor(Handle, virtfakeClient.Tracker(), ModifyVM))
		vmInformer, _ = testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
		recorder = record.NewFakeRecorder(100)
		controller, _ = NewFirmwareController(vmInformer, virtClient, recorder)

		mockQueue = testutils.NewMockWorkQueue(controller.queue)
		controller.queue = mockQueue
	})

	addVirtualMachine := func(vm *v1.VirtualMachine) {
		key, err := virtcontroller.KeyFunc(vm)
		Expect(err).To(Not(HaveOccurred()))
		controller.queue.Add(key)
		Expect(controller.vmIndexer.Add(vm)).To(Succeed())
	}

	sanityExecute := func(vm *v1.VirtualMachine) {
		controllertesting.SanityExecute(controller, []cache.Store{controller.vmIndexer}, Default)
	}

	It("should patch VM if firmware UUID is missing", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)))
		_, err := virtfakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		addVirtualMachine(vm)
		sanityExecute(vm)

		updatedVM, err := virtfakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
		Expect(updatedVM.Spec.Template.Spec.Domain.Firmware.UUID).To(Equal(CalculateLegacyUUID(vm.Name)))
	})

	It("should not modify firmware UUID if already set", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithFirmwareUUID("existing-uuid")))
		_, err := virtfakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		addVirtualMachine(vm)

		sanityExecute(vm)

		updatedVM, err := virtfakeClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVM.Spec.Template.Spec.Domain.Firmware.UUID).To(Equal(vm.Spec.Template.Spec.Domain.Firmware.UUID))
	})

})
