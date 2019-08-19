package utils

import (
	"context"

	"github.com/ghodss/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/utils/conditions"
)

// CreateMachineHealthCheck will create MachineHealthCheck CR with the relevant selector
func CreateMachineHealthCheck(name string, labels map[string]string) error {
	c, err := LoadClient()
	if err != nil {
		return err
	}

	mhc := &mrv1.MachineHealthCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NamespaceOpenShiftMachineAPI,
		},
		Spec: mrv1.MachineHealthCheckSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}
	return c.Create(context.TODO(), mhc)
}

// DeleteMachineHealthCheck deletes machine health check by name
func DeleteMachineHealthCheck(healthcheckName string) error {
	c, err := LoadClient()
	if err != nil {
		return err
	}

	key := types.NamespacedName{
		Name:      healthcheckName,
		Namespace: NamespaceOpenShiftMachineAPI,
	}
	healthcheck := &mrv1.MachineHealthCheck{}
	if err := c.Get(context.TODO(), key, healthcheck); err != nil {
		return err
	}
	return c.Delete(context.TODO(), healthcheck)
}

// CreateUnhealthyConditionsConfigMap creates node-unhealthy-conditions configmap with relevant conditions
func CreateUnhealthyConditionsConfigMap(unhealthyConditions *conditions.UnhealthyConditions) error {
	c, err := LoadClient()
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NamespaceOpenShiftMachineAPI,
			Name:      mrv1.ConfigMapNodeUnhealthyConditions,
		},
	}

	conditionsData, err := yaml.Marshal(unhealthyConditions)
	if err != nil {
		return err
	}

	cm.Data = map[string]string{"conditions": string(conditionsData)}
	return c.Create(context.TODO(), cm)
}

// DeleteUnhealthyConditionsConfigMap deletes node-unhealthy-conditions configmap
func DeleteUnhealthyConditionsConfigMap() error {
	c, err := LoadClient()
	if err != nil {
		return err
	}

	key := types.NamespacedName{
		Name:      mrv1.ConfigMapNodeUnhealthyConditions,
		Namespace: NamespaceOpenShiftMachineAPI,
	}
	cm := &corev1.ConfigMap{}
	if err := c.Get(context.TODO(), key, cm); err != nil {
		return err
	}

	return c.Delete(context.TODO(), cm)
}

// StopKubelet creates pod in the node PID namespace that stops kubelet process
func StopKubelet(nodeName string) error {
	client, err := LoadClient()
	if err != nil {
		return err
	}

	_true := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeletKillerPodName + rand.String(5),
			Namespace: NamespaceOpenShiftMachineAPI,
			Labels: map[string]string{
				KubeletKillerPodName: "",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    KubeletKillerPodName,
					Image:   "busybox",
					Command: []string{"pkill", "-STOP", "hyperkube"},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &_true,
					},
				},
			},
			NodeName: nodeName,
			HostPID:  true,
		},
	}
	return client.Create(context.TODO(), pod)
}

// DeleteKubeletKillerPods deletes kubelet killer pod
func DeleteKubeletKillerPods() error {
	c, err := LoadClient()
	if err != nil {
		return err
	}
	podList := &corev1.PodList{}
	if err := c.List(
		context.TODO(),
		podList,
		client.InNamespace(NamespaceOpenShiftMachineAPI),
		client.MatchingLabels(map[string]string{KubeletKillerPodName: ""}),
	); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		if err := c.Delete(context.TODO(), &pod); err != nil {
			return err
		}
	}
	return nil
}
