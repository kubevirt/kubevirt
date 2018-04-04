package controller

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

type VirtualMachineConditionManager struct {
}

func (d *VirtualMachineConditionManager) CheckFailure(vm *v1.VirtualMachine, syncErr error, reason string) (changed bool) {
	if syncErr != nil && !d.HasCondition(vm, v1.VirtualMachineSynchronized) {
		vm.Status.Conditions = append(vm.Status.Conditions, v1.VirtualMachineCondition{
			Type:               v1.VirtualMachineSynchronized,
			Reason:             reason,
			Message:            syncErr.Error(),
			LastTransitionTime: metav1.Now(),
			Status:             k8sv1.ConditionFalse,
		})
		return true
	} else if syncErr == nil && d.HasCondition(vm, v1.VirtualMachineSynchronized) {
		d.RemoveCondition(vm, v1.VirtualMachineSynchronized)
		return true
	}
	return false
}

func (d *VirtualMachineConditionManager) HasCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) bool {
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (d *VirtualMachineConditionManager) RemoveCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) {
	var conds []v1.VirtualMachineCondition
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vm.Status.Conditions = conds
}

func NewVirtualMachineConditionManager() *VirtualMachineConditionManager {
	return &VirtualMachineConditionManager{}
}
