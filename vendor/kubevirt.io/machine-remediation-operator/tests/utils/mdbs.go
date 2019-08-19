package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
)

// NewMachineDisruptionBudget returns new MachineDisruptionObject with specified parameters
func NewMachineDisruptionBudget(name string, machineLabels map[string]string, minAvailable *int32, maxUnavailable *int32) *mrv1.MachineDisruptionBudget {
	selector := &metav1.LabelSelector{MatchLabels: machineLabels}
	return &mrv1.MachineDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NamespaceOpenShiftMachineAPI,
		},
		Spec: mrv1.MachineDisruptionBudgetSpec{
			MinAvailable:   minAvailable,
			MaxUnavailable: maxUnavailable,
			Selector:       selector,
		},
	}
}
