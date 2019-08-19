package components

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
)

// NewMachineRemediationOperator retruns new MachineRemediationOperator object
func NewMachineRemediationOperator(name string, namespace string, imageRepository string, pullPolicy corev1.PullPolicy, operatorVersion string) *mrv1.MachineRemediationOperator {
	return &mrv1.MachineRemediationOperator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineremediation.kubevirt.io/v1alpha1",
			Kind:       "MachineRemediationOperator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				mrv1.SchemeGroupVersion.Group:              "",
				mrv1.SchemeGroupVersion.Group + "/version": operatorVersion,
			},
		},
		Spec: mrv1.MachineRemediationOperatorSpec{
			ImagePullPolicy: pullPolicy,
			ImageRegistry:   imageRepository,
		},
	}
}
