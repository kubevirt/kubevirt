package libvmruntime

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	_ "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

func StopVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	return patchVMWithRunStrategy(vm, v1.RunStrategyHalted)
}

func StartVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	return patchVMWithRunStrategy(vm, v1.RunStrategyAlways)
}

func patchVMWithRunStrategy(vm *v1.VirtualMachine, rs v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	ginkgo.By("Starting the VirtualMachineInstance")
	virtClient := kubevirt.Client()
	patchSet := patch.New()
	patchSet.AddOption(
		patch.WithAdd("/spec/runStrategy", rs),
	)
	payload, err := patchSet.GeneratePayload()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return vm
}
