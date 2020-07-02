package util

import (
	"context"
	"fmt"

	"errors"
	"os"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"sigs.k8s.io/controller-runtime/pkg/client"

	csvv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetOperatorNamespaceFromEnv() (string, error) {
	if namespace, ok := os.LookupEnv(OperatorNamespaceEnv); ok {
		return namespace, nil
	}

	return "", fmt.Errorf("%s unset or empty in environment", OperatorNamespaceEnv)
}

func GetPod(c client.Client, logger logr.Logger) (*corev1.Pod, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		logger.Error(err, "Failed to get HCO namespace")
		return nil, err
	}
	pod, err := k8sutil.GetPod(context.TODO(), c, operatorNs)
	if err != nil {
		logger.Error(err, "Failed to get HCO pod")
		return nil, err
	}

	return pod, nil
}

func GetCSVfromPod(pod *corev1.Pod, c client.Client, logger logr.Logger) (*csvv1alpha1.ClusterServiceVersion, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	rs_reference := metav1.GetControllerOf(pod)
	if rs_reference == nil || rs_reference.Kind != "ReplicaSet" {
		err = errors.New("Failed getting HCO replicaSet reference")
		logger.Error(err, "Failed getting HCO replicaSet reference")
		return nil, err
	}
	rs := &appsv1.ReplicaSet{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      rs_reference.Name,
	}, rs)
	if err != nil {
		logger.Error(err, "Failed to get HCO ReplicaSet")
		return nil, err
	}

	d_reference := metav1.GetControllerOf(rs)
	if d_reference == nil || d_reference.Kind != "Deployment" {
		err = errors.New("Failed getting HCO deployment reference")
		logger.Error(err, "Failed getting HCO deployment reference")
		return nil, err
	}
	d := &appsv1.Deployment{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      d_reference.Name,
	}, d)
	if err != nil {
		logger.Error(err, "Failed to get HCO Deployment")
		return nil, err
	}

	var csv_reference *metav1.OwnerReference
	for _, owner := range d.GetOwnerReferences() {
		if owner.Kind == "ClusterServiceVersion" {
			csv_reference = &owner
		}
	}
	if csv_reference == nil {
		err = errors.New("Failed getting HCO CSV reference")
		logger.Error(err, "Failed getting HCO CSV reference")
		return nil, err
	}
	csv := &csvv1alpha1.ClusterServiceVersion{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      csv_reference.Name,
	}, csv)
	if err != nil {
		logger.Error(err, "Failed to get HCO CSV")
		return nil, err
	}

	return csv, nil
}

func NewKubeVirtPriorityClass(crname string) *schedulingv1.PriorityClass {
	labels := map[string]string{
		AppLabel: crname,
	}
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-cluster-critical",
			Labels: labels,
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}
