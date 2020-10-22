package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"sigs.k8s.io/controller-runtime/pkg/client"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

func GetWebhookModeFromEnv() (bool, error) {
	if whmodestring, ok := os.LookupEnv(OperatorWebhookModeEnv); ok {
		if whmodestring == "true" {
			return true, nil
		} else if whmodestring == "false" {
			return false, nil
		} else {
			return false, fmt.Errorf("%s unexpected value in environment", OperatorWebhookModeEnv)
		}
	}

	return false, fmt.Errorf("%s unset or empty in environment", OperatorWebhookModeEnv)
}

func GetPod(ctx context.Context, c client.Reader, logger logr.Logger, ci ClusterInfo) (*corev1.Pod, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		logger.Error(err, "Failed to get HCO namespace")
		return nil, err
	}

	// This is taken from k8sutil.GetPod. This method only receives client. But the client is not always ready. We'll
	// use --- instead
	if ci.IsRunningLocally() {
		return nil, nil
	}
	podName := os.Getenv(k8sutil.PodNameEnvVar)
	if podName == "" {
		return nil, fmt.Errorf("required env %s not set, please configure downward API", k8sutil.PodNameEnvVar)
	}

	pod := &corev1.Pod{}
	key := client.ObjectKey{Namespace: operatorNs, Name: podName}
	err = c.Get(ctx, key, pod)
	if err != nil {
		logger.Error(err, "Failed to get Pod", "Pod.Namespace", operatorNs, "Pod.Name", podName)
		return nil, err
	}

	// .Get() clears the APIVersion and Kind,
	// so we need to set them before returning the object.
	pod.TypeMeta.APIVersion = "v1"
	pod.TypeMeta.Kind = "Pod"

	logger.Info("Found Pod", "Pod.Namespace", operatorNs, "Pod.Name", pod.Name)

	return pod, nil
}

func GetCSVfromPod(pod *corev1.Pod, c client.Reader, logger logr.Logger) (*csvv1alpha1.ClusterServiceVersion, error) {
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}
	rsReference := metav1.GetControllerOf(pod)
	if rsReference == nil || rsReference.Kind != "ReplicaSet" {
		err = errors.New("failed getting HCO replicaSet reference")
		logger.Error(err, "Failed getting HCO replicaSet reference")
		return nil, err
	}
	rs := &appsv1.ReplicaSet{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      rsReference.Name,
	}, rs)
	if err != nil {
		logger.Error(err, "Failed to get HCO ReplicaSet")
		return nil, err
	}

	dReference := metav1.GetControllerOf(rs)
	if dReference == nil || dReference.Kind != "Deployment" {
		err = errors.New("failed getting HCO deployment reference")
		logger.Error(err, "Failed getting HCO deployment reference")
		return nil, err
	}
	d := &appsv1.Deployment{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      dReference.Name,
	}, d)
	if err != nil {
		logger.Error(err, "Failed to get HCO Deployment")
		return nil, err
	}

	var csvReference *metav1.OwnerReference
	for _, owner := range d.GetOwnerReferences() {
		if owner.Kind == "ClusterServiceVersion" {
			csvReference = &owner
		}
	}
	if csvReference == nil {
		err = errors.New("failed getting HCO CSV reference")
		logger.Error(err, "Failed getting HCO CSV reference")
		return nil, err
	}
	csv := &csvv1alpha1.ClusterServiceVersion{}
	err = c.Get(context.TODO(), client.ObjectKey{
		Namespace: operatorNs,
		Name:      csvReference.Name,
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

// GetRuntimeObject will query the apiserver for the object
func GetRuntimeObject(ctx context.Context, c client.Client, obj runtime.Object, logger logr.Logger) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		logger.Error(err, "Failed to get object key", "Kind", obj.GetObjectKind())
		return err
	}

	return c.Get(ctx, key, obj)
}

// ComponentResourceRemoval removes the resource `obj` if it exists and belongs to the HCO
func ComponentResourceRemoval(ctx context.Context, c client.Client, obj interface{}, hcoName string, logger logr.Logger, dryRun bool) error {
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

	logger.Info("Removing resource", "GVK", resource.GetObjectKind().GroupVersionKind(), "DryRun", dryRun)

	return c.Delete(ctx, resource, opts)
}

// EnsureDeleted calls ComponentResourceRemoval if the runtime object exists
func EnsureDeleted(ctx context.Context, c client.Client, obj runtime.Object, hcoName string, logger logr.Logger, dryRun bool) error {
	err := GetRuntimeObject(ctx, c, obj, logger)

	if err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			logger.Info("Resource doesn't exist, there is nothing to remove", "Kind", obj.GetObjectKind())
			return nil
		}

		logger.Error(err, "failed to get object from kubernetes", "Kind", obj.GetObjectKind())
		return err
	}

	return ComponentResourceRemoval(ctx, c, obj, hcoName, logger, dryRun)
}

// EnsureCreated creates the runtime object if it does not exist
func EnsureCreated(ctx context.Context, c client.Client, obj runtime.Object, logger logr.Logger) error {
	err := GetRuntimeObject(ctx, c, obj, logger)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating object", "kind", obj.GetObjectKind())
			return c.Create(ctx, obj)
		}

		if meta.IsNoMatchError(err) {
			return err
		}

		logger.Error(err, "failed getting runtime object", "kind", obj.GetObjectKind())
		return err
	}

	return nil
}
