package libvm

import (
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libvmi"
)

type Option func(vm *virtv1.VirtualMachine)

func New(opts ...Option) *virtv1.VirtualMachine {
	vm := baseVm(randName())
	for _, f := range opts {
		f(vm)
	}
	return vm
}

func randName() string {
	const randomPostfixLen = 5
	return "testvm" + "-" + rand.String(randomPostfixLen)
}

func baseVm(name string) *virtv1.VirtualMachine {
	vm := virtv1.NewVMReferenceFromNameWithNS("", name)
	vm.Spec = virtv1.VirtualMachineSpec{}
	vm.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: virtv1.GroupVersion.String(),
		Kind:       "VirtualMachine",
	}
	return vm
}

func WithVMITemplateSpec(vmiOpts ...libvmi.Option) Option {
	vmi := libvmi.New(vmiOpts...)
	return func(vm *virtv1.VirtualMachine) {
		vm.Spec.Template = &virtv1.VirtualMachineInstanceTemplateSpec{
			ObjectMeta: vmi.ObjectMeta,
			Spec:       vmi.Spec,
		}
	}
}

func WithNamespace(namespace string) Option {
	return func(vm *virtv1.VirtualMachine) {
		vm.Namespace = namespace
	}
}
