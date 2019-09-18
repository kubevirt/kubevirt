package components

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
)

// NewMastersMachineDisruptionBudget retruns new MachineDisruptionBudget object for master nodes
func NewMastersMachineDisruptionBudget(name string, namespace string, operatorVersion string) *mrv1.MachineDisruptionBudget {
	return &mrv1.MachineDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineremediation.kubevirt.io/v1alpha1",
			Kind:       "MachineDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				mrv1.SchemeGroupVersion.Group:              "",
				mrv1.SchemeGroupVersion.Group + "/version": operatorVersion,
			},
		},
		Spec: mrv1.MachineDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					consts.MasterRoleLabel: "",
				},
			},
			MinAvailable: pointer.Int32Ptr(1),
		},
	}
}
