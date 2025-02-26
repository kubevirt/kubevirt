package instancetype

import (
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/find"
	"kubevirt.io/kubevirt/pkg/instancetype/infer"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
)

type Methods interface {
	Upgrade(vm *virtv1.VirtualMachine) error
	FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error)
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
	InferDefaultInstancetype(vm *virtv1.VirtualMachine) error
	InferDefaultPreference(vm *virtv1.VirtualMachine) error
}

type InstancetypeMethods struct {
	InstancetypeStore        cache.Store
	ClusterInstancetypeStore cache.Store
	PreferenceStore          cache.Store
	ClusterPreferenceStore   cache.Store
	ControllerRevisionStore  cache.Store
	Clientset                kubecli.KubevirtClient
}

var _ Methods = &InstancetypeMethods{}

func (m *InstancetypeMethods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	return preferenceFind.NewSpecFinder(m.PreferenceStore, m.ClusterPreferenceStore, m.ControllerRevisionStore, m.Clientset).FindPreference(vm)
}

func (m *InstancetypeMethods) FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
	return find.NewSpecFinder(m.InstancetypeStore, m.ClusterInstancetypeStore, m.ControllerRevisionStore, m.Clientset).Find(vm)
}

func (m *InstancetypeMethods) InferDefaultInstancetype(vm *virtv1.VirtualMachine) error {
	return infer.New(m.Clientset).Instancetype(vm)
}

func (m *InstancetypeMethods) InferDefaultPreference(vm *virtv1.VirtualMachine) error {
	return infer.New(m.Clientset).Preference(vm)
}

func (m *InstancetypeMethods) Upgrade(vm *virtv1.VirtualMachine) error {
	return upgrade.New(m.ControllerRevisionStore, m.Clientset).Upgrade(vm)
}
