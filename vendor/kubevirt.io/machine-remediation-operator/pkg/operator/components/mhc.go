package components

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
)

// NewMastersMachineHealthCheck retruns new MachineHealthCheck object for master nodes
func NewMastersMachineHealthCheck(name string, namespace string, operatorVersion string) *mrv1.MachineHealthCheck {
	rebootStrategy := mrv1.RemediationStrategyTypeReboot
	return &mrv1.MachineHealthCheck{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineremediation.kubevirt.io/v1alpha1",
			Kind:       "MachineHealthCheck",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				mrv1.SchemeGroupVersion.Group:              "",
				mrv1.SchemeGroupVersion.Group + "/version": operatorVersion,
			},
		},
		Spec: mrv1.MachineHealthCheckSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					consts.MasterRoleLabel: "",
				},
			},
			RemediationStrategy: &rebootStrategy,
		},
	}
}
