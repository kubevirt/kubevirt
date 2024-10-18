package handler

import (
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/find"
)

type VMControllerHandler struct {
	find.SpecFinder
}

func NewVMControllerHandler(
	preferenceStore, clusterPreferenceStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient,
) *VMControllerHandler {
	return &VMControllerHandler{
		SpecFinder: *find.NewSpecFinder(preferenceStore, clusterPreferenceStore, revisionStore, virtClient),
	}
}

type MockVMControllerHandler struct {
	FindFunc func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

func (m *MockVMControllerHandler) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	return m.FindFunc(vm)
}

func NewMockVMControllerHandler() *MockVMControllerHandler {
	return &MockVMControllerHandler{
		FindFunc: func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
	}
}
