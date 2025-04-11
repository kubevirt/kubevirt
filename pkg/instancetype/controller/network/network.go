package network

import (
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	preferencefind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
)

type finder interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type applier interface {
	Apply(*v1beta1.VirtualMachinePreferenceSpec, *virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstanceSpec
}

type netController struct {
	finder
	applier
}

func New(preferenceStore, clusterPreferenceStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *netController {
	return &netController{
		finder:  preferencefind.NewSpecFinder(preferenceStore, clusterPreferenceStore, revisionStore, virtClient),
		applier: &preferenceapply.InterfaceApplier{},
	}
}

func (n *netController) ApplyInterfacePreferencesToVMI(
	vm *virtv1.VirtualMachine,
	spec *virtv1.VirtualMachineInstanceSpec,
) *virtv1.VirtualMachineInstanceSpec {
	preferenceSpec, err := n.FindPreference(vm)
	// Allow the calling controller to be eventually consistent here by swallowing the err to find the preference
	if preferenceSpec == nil || err != nil {
		return spec
	}
	return n.Apply(preferenceSpec, spec)
}
