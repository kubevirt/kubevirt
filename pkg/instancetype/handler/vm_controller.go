package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
)

type VMControllerHandler struct {
	find.SpecFinder
	apply.VMApplier
	apply.VMIApplier
	revision.Handler
	upgrade.Upgrader
}

func NewVMControllerHandler(
	instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, revisionStore cache.Store,
	virtClient kubecli.KubevirtClient,
) *VMControllerHandler {
	return &VMControllerHandler{
		SpecFinder: *find.NewSpecFinder(instancetypeStore, clusterInstancetypeStore, revisionStore, virtClient),
		VMApplier: *apply.NewVMApplier(
			find.NewSpecFinder(instancetypeStore, clusterInstancetypeStore, revisionStore, virtClient),
			preferenceFind.NewSpecFinder(preferenceStore, clusterPreferenceStore, revisionStore, virtClient),
		),
		VMIApplier: *apply.NewVMIApplier(),
		Handler:    *revision.New(instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, virtClient),
		Upgrader:   *upgrade.New(revisionStore, virtClient),
	}
}

type MockVMControllerHandler struct {
	ApplyToVMFunc  func(*virtv1.VirtualMachine) error
	ApplyToVMIFunc func(
		*k8sfield.Path,
		*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
		*metav1.ObjectMeta,
	) apply.Conflicts
	FindFunc    func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
	StoreFunc   func(*virtv1.VirtualMachine) error
	UpgradeFunc func(*virtv1.VirtualMachine) error
}

func (m *MockVMControllerHandler) ApplyToVM(vm *virtv1.VirtualMachine) error {
	return m.ApplyToVMFunc(vm)
}

func (m *MockVMControllerHandler) ApplyToVMI(
	path *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmSpec *virtv1.VirtualMachineInstanceSpec,
	objectMeta *metav1.ObjectMeta,
) apply.Conflicts {
	return m.ApplyToVMIFunc(path, instancetypeSpec, preferenceSpec, vmSpec, objectMeta)
}

func (m *MockVMControllerHandler) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	return m.FindFunc(vm)
}

func (m *MockVMControllerHandler) Store(vm *virtv1.VirtualMachine) error {
	return m.StoreFunc(vm)
}

func (m *MockVMControllerHandler) Upgrade(vm *virtv1.VirtualMachine) error {
	return m.UpgradeFunc(vm)
}

func NewMockVMControllerHandler() *MockVMControllerHandler {
	return &MockVMControllerHandler{
		ApplyToVMFunc: func(*virtv1.VirtualMachine) error {
			return nil
		},
		ApplyToVMIFunc: func(
			*k8sfield.Path,
			*v1beta1.VirtualMachineInstancetypeSpec,
			*v1beta1.VirtualMachinePreferenceSpec,
			*virtv1.VirtualMachineInstanceSpec,
			*metav1.ObjectMeta,
		) apply.Conflicts {
			return nil
		},
		FindFunc: func(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
			return nil, nil
		},
		StoreFunc: func(*virtv1.VirtualMachine) error {
			return nil
		},
		UpgradeFunc: func(*virtv1.VirtualMachine) error {
			return nil
		},
	}
}
