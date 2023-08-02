package convertmachinetype_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	. "kubevirt.io/kubevirt/pkg/virtctl/update/machine-type/convert-machine-type"
)

const (
	machineTypeGlob        = "*rhel8.*"
	machineTypeNeedsUpdate = "pc-q35-rhel8.2.0"
	machineTypeNoUpdate    = "pc-q35-rhel9.0.0"
	restartRequiredLabel   = "restart-vm-required"
)

var _ = Describe("Update Machine Type", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInformer cache.SharedIndexInformer
	var vmiInformer cache.SharedIndexInformer
	var controller *JobController
	var err error

	shouldExpectRestartRequiredLabel := func(vm *k6tv1.VirtualMachine) {
		patchData := `[{ "op": "add", "path": "/status/machineTypeRestartRequired", "value": true }]`

		vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &v1.PatchOptions{}).Times(1)
	}

	shouldExpectPatchMachineType := func(vm *k6tv1.VirtualMachine) {
		patchData := `[{"op": "remove", "path": "/spec/template/spec/domain/machine"}]`

		vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &v1.PatchOptions{}).Times(1)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

		vmInformer, _ = testutils.NewFakeInformerFor(&k6tv1.VirtualMachine{})
		vmiInformer, _ = testutils.NewFakeInformerFor(&k6tv1.VirtualMachineInstance{})
		controller, err = NewJobController(vmInformer, vmiInformer, virtClient)
		Expect(err).ToNot(HaveOccurred())

		virtClient.EXPECT().VirtualMachine(v1.NamespaceDefault).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(v1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachine(v1.NamespaceAll).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(v1.NamespaceAll).Return(vmiInterface).AnyTimes()

		MachineTypeGlob = machineTypeGlob
	})

	Context("For VM with machine type matching machine type glob", func() {
		It("should remove machine type from VM spec", func() {
			vm := newVMWithMachineType(machineTypeNeedsUpdate, false)

			shouldExpectPatchMachineType(vm)

			err := controller.UpdateMachineType(vm, false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should set MachineTypeRestartRequired to true in VM Status", func() {
			vm := newVMWithMachineType(machineTypeNeedsUpdate, true)

			shouldExpectPatchMachineType(vm)
			shouldExpectRestartRequiredLabel(vm)

			err := controller.UpdateMachineType(vm, true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should restart VM when VM is running and restartNow=true", func() {
			vm := newVMWithMachineType(machineTypeNeedsUpdate, true)

			shouldExpectPatchMachineType(vm)
			RestartNow = true

			vmInterface.EXPECT().Restart(context.Background(), vm.Name, &k6tv1.RestartOptions{}).Times(1)

			err := controller.UpdateMachineType(vm, true)
			Expect(err).ToNot(HaveOccurred())

			RestartNow = false
		})
	})
})

func newVMWithMachineType(machineType string, running bool) *k6tv1.VirtualMachine {
	vmName := "test-vm-" + machineType
	testVM := &k6tv1.VirtualMachine{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: vmName,
			Namespace:    v1.NamespaceDefault,
		},
		Spec: k6tv1.VirtualMachineSpec{
			Running: &running,
			Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
				Spec: k6tv1.VirtualMachineInstanceSpec{
					Domain: k6tv1.DomainSpec{
						Machine: &k6tv1.Machine{
							Type: machineType,
						},
					},
				},
			},
		},
	}
	testVM.Labels = map[string]string{}
	return testVM
}
