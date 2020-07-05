package util

import (
	"context"
	"encoding/json"
	"fmt"

	"errors"
	"os"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"sigs.k8s.io/controller-runtime/pkg/client"

	csvv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// toUnstructured convers an arbitrary object (which MUST obey the
// k8s object conventions) to an Unstructured
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(b, u); err != nil {
		return nil, err
	}
	return u, nil
}

func ComponentResourceRemoval(obj interface{}, c client.Client, ctx context.Context, hcoName string, logger logr.Logger, dryRun bool) error {
	resource, err := toUnstructured(obj)
	if err != nil {
		logger.Error(err, "Failed to convert object to Unstructured")
		return err
	}

	err = c.Get(ctx, types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource doesn't exist, there is nothing to remove", "Kind", resource.GetObjectKind())
			return nil
		}
		return err
	}

	labels := resource.GetLabels()

	if app, labelExists := labels[AppLabel]; !labelExists || app != hcoName {
		logger.Info("Existing resource wasn't deployed by HCO, ignoring", "Kind", resource.GetObjectKind())
		return nil
	}

	opts := &client.DeleteOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	logger.Info("Removing resource", "Kind", resource.GetObjectKind(), "DryRun", dryRun)

	return c.Delete(ctx, resource, opts)
}

func EnsureDeleted(c client.Client, ctx context.Context, hcoName string, obj runtime.Object, logger logr.Logger, dryRun bool) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		logger.Error(err, "Failed to get object key", "Kind", obj.GetObjectKind())
		return err
	}

	err = c.Get(ctx, key, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource doesn't exist, there is nothing to remove", "Kind", obj.GetObjectKind())
			return nil
		}

		logger.Error(err, "Failed to get object from kubernetes", "Kind", obj.GetObjectKind())
		return err
	}

	return ComponentResourceRemoval(obj, c, ctx, hcoName, logger, dryRun)
}
