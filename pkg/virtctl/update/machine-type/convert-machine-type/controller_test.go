package convertmachinetype_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	. "kubevirt.io/kubevirt/pkg/virtctl/update/machine-type/convert-machine-type"
)

const (
	machineTypeGlob        = "*rhel8.*"
	machineTypeNeedsUpdate = "pc-q35-rhel8.2.0"
	machineTypeNoUpdate    = "pc-q35-rhel9.0.0"
)

var _ = Describe("JobController", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var kubeClient *fake.Clientset
	var vmInformer cache.SharedIndexInformer
	var vmiInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var controller *JobController
	var stop chan struct{}
	var err error

	shouldExpectVMICreation := func(vmi *virtv1.VirtualMachineInstance) {
		kubeClient.Fake.PrependReactor("create", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(create.GetObject().(*virtv1.VirtualMachineInstance).Name).To(Equal(vmi.Name))

			return true, create.GetObject(), nil
		})
	}

	shouldExpectVMCreation := func(vm *virtv1.VirtualMachine) {
		kubeClient.Fake.PrependReactor("create", "virtualmachine", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(create.GetObject().(*virtv1.VirtualMachineInstance).Name).To(Equal(vm.Name))

			return true, create.GetObject(), nil
		})
	}

	shouldExpectUpdateVMStatusFalse := func(vm *virtv1.VirtualMachine) {
		patchData := `[{ "op": "replace", "path": "/status/machineTypeRestartRequired", "value": false }]`

		vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &v1.PatchOptions{}).Times(1)

		kubeClient.Fake.PrependReactor("patch", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patch.GetPatch()).To(Equal([]byte(patchData)))
			Expect(patch.GetPatchType()).To(Equal(types.JSONPatchType))
			Expect(vm.Status.MachineTypeRestartRequired).To(BeFalse())
			return true, vm, nil
		})
	}

	shouldExpectUpdateVMStatusTrue := func(vm *virtv1.VirtualMachine) {
		patchData := `[{ "op": "add", "path": "/status/machineTypeRestartRequired", "value": true }]`
		vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &v1.PatchOptions{}).Times(1)

		kubeClient.Fake.PrependReactor("patch", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patch.GetPatch()).To(Equal([]byte(patchData)))
			Expect(patch.GetPatchType()).To(Equal(types.JSONPatchType))
			Expect(vm.Status.MachineTypeRestartRequired).To(BeTrue())
			return true, vm, nil
		})
	}

	shouldExpectPatchMachineType := func(vm *virtv1.VirtualMachine) {
		patchData := `[{"op": "remove", "path": "/spec/template/spec/domain/machine"}]`

		vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &v1.PatchOptions{}).Times(1)
		kubeClient.Fake.PrependReactor("patch", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patch.GetPatch()).To(Equal([]byte(patchData)))
			Expect(patch.GetPatchType()).To(Equal(types.JSONPatchType))
			Expect(vm.Spec.Template.Spec.Domain.Machine).To(BeNil())
			return true, vm, nil
		})
	}

	addVm := func(vm *virtv1.VirtualMachine) {
		err = vmInformer.GetStore().Add(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		key, err := cache.MetaNamespaceKeyFunc(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		mockQueue.Add(key)
	}

	addVmi := func(vmi *virtv1.VirtualMachineInstance) {
		err = vmiInformer.GetStore().Add(vmi)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	stopVM := func(vm *virtv1.VirtualMachine) {
		vm.Spec.Running = pointer.Bool(false)
		err = vmInformer.GetStore().Update(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		key, err := cache.MetaNamespaceKeyFunc(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		mockQueue.Add(key)
	}

	Describe("JobController", func() {
		var vm *virtv1.VirtualMachine
		var vmi *virtv1.VirtualMachineInstance

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			kubeClient = fake.NewSimpleClientset()
			stop = make(chan struct{})

			vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
			vmiInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})

			controller, err = NewJobController(vmInformer, vmiInformer, virtClient)
			Expect(err).ToNot(HaveOccurred())

			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue

			go controller.VmInformer.Run(stop)
			go controller.VmiInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, controller.VmInformer.HasSynced)).To(BeTrue())

			virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()

			MachineTypeGlob = machineTypeGlob
			Testing = true
		})

		AfterEach(func() {
			Testing = false
		})

		Describe("with VM keys in queue", func() {
			Context("with no envs set", func() {
				It("Should update machine type of all VMs with specified machine type", func() {
					vm = newVM(machineTypeNeedsUpdate, v1.NamespaceDefault, false, false)
					vmNoUpdate := newVM(machineTypeNoUpdate, v1.NamespaceDefault, false, false)
					addVm(vm)
					addVm(vmNoUpdate)

					shouldExpectVMCreation(vm)
					shouldExpectVMCreation(vmNoUpdate)
					shouldExpectPatchMachineType(vm)

					controller.Execute()
				})

				It("Should update VM Status MachineTypeRestartRequired label for updated running VMs", func() {
					vm = newVM(machineTypeNeedsUpdate, v1.NamespaceDefault, true, false)
					vmi = newVMIWithMachineType(machineTypeNeedsUpdate, vm.Name)
					addVm(vm)
					addVmi(vmi)

					shouldExpectVMCreation(vm)
					shouldExpectPatchMachineType(vm)
					shouldExpectUpdateVMStatusTrue(vm)

					controller.Execute()
				})
			})

			Context("when restart-now env is set", func() {
				It("Should immediately restart any running VMs that have been updated", func() {
					RestartNow = true
					vm = newVM(machineTypeNeedsUpdate, v1.NamespaceDefault, true, false)
					vmi = newVMIWithMachineType(machineTypeNeedsUpdate, vm.Name)
					addVm(vm)
					addVmi(vmi)

					shouldExpectVMCreation(vm)
					shouldExpectPatchMachineType(vm)
					vmInterface.EXPECT().Restart(context.Background(), vm.Name, &virtv1.RestartOptions{}).Times(1)

					controller.Execute()
					RestartNow = false
				})
			})
		})

		Describe("when VMI has been deleted", func() {
			Context("if VM machine type has been updated", func() {
				It("should set MachineTypeRestartRequired to false in VM Status", func() {
					vm = newVM("q35", v1.NamespaceDefault, true, true)
					vmi = newVMIWithMachineType(machineTypeNoUpdate, vm.Name)
					addVmi(vmi)
					addVm(vm)

					shouldExpectVMCreation(vm)
					shouldExpectVMICreation(vmi)
					shouldExpectUpdateVMStatusFalse(vm)

					stopVM(vm)
					controller.Execute()
				})
			})
		})
	})
})

func newVM(machineType, namespace string, running, restartRequired bool) *virtv1.VirtualMachine {
	testVM := &virtv1.VirtualMachine{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("test-vm-%s", machineType),
			Namespace: namespace,
		},
		Spec: virtv1.VirtualMachineSpec{
			Running: pointer.Bool(running),
			Template: &virtv1.VirtualMachineInstanceTemplateSpec{
				Spec: virtv1.VirtualMachineInstanceSpec{
					Domain: virtv1.DomainSpec{},
				},
			},
		},
		Status: virtv1.VirtualMachineStatus{
			MachineTypeRestartRequired: restartRequired,
		},
	}

	if machineType != "q35" {
		testVM.Spec.Template.Spec.Domain.Machine = &virtv1.Machine{Type: machineType}
	}

	return testVM
}

func newVMIWithMachineType(machineType string, name string) *virtv1.VirtualMachineInstance {
	statusMachineType := machineType
	if machineType == "q35" {
		statusMachineType = "pc-q35-rhel9.2.0"
	}

	testvmi := &virtv1.VirtualMachineInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: v1.NamespaceDefault,
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				Machine: &virtv1.Machine{
					Type: machineType,
				},
			},
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			Machine: &virtv1.Machine{
				Type: statusMachineType,
			},
			Phase: virtv1.Running,
		},
	}

	return testvmi
}
