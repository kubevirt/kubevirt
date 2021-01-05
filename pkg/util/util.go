package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"

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

// ForceRunModeEnv indicates if the operator should be forced to run in either local
// or cluster mode (currently only used for local mode)
var ForceRunModeEnv = "OSDK_FORCE_RUN_MODE"

type RunModeType string

const (
	LocalRunMode   RunModeType = "local"
	ClusterRunMode RunModeType = "cluster"

	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which is the namespace where the watch activity happens.
	// this value is empty if the operator is running with clusterScope.
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"

	// PodNameEnvVar is the constant for env variable POD_NAME
	// which is the name of the current pod.
	PodNameEnvVar = "POD_NAME"
)

// ErrNoNamespace indicates that a namespace could not be found for the current
// environment
var ErrNoNamespace = fmt.Errorf("namespace not found for current environment")

// ErrRunLocal indicates that the operator is set to run in local mode (this error
// is returned by functions that only work on operators running in cluster mode)
var ErrRunLocal = fmt.Errorf("operator run mode forced to local")

func GetOperatorNamespaceFromEnv() (string, error) {
	if namespace, ok := os.LookupEnv(OperatorNamespaceEnv); ok {
		return namespace, nil
	}

	return "", fmt.Errorf("%s unset or empty in environment", OperatorNamespaceEnv)
}

func isRunModeLocal() bool {
	return os.Getenv(ForceRunModeEnv) == string(LocalRunMode)
}

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace(logger logr.Logger) (string, error) {
	if isRunModeLocal() {
		return "", ErrRunLocal
	}
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	logger.Info("Found namespace", "Namespace", ns)
	return ns, nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

func GetPod(ctx context.Context, c client.Reader, logger logr.Logger, ci ClusterInfo) (*corev1.Pod, error) {
	operatorNs, err := GetOperatorNamespace(logger)
	if err != nil {
		logger.Error(err, "Failed to get HCO namespace")
		return nil, err
	}

	// This is taken from k8sutil.GetPod. This method only receives client. But the client is not always ready. We'll
	// use --- instead
	if ci.IsRunningLocally() {
		return nil, nil
	}
	podName := os.Getenv(PodNameEnvVar)
	if podName == "" {
		return nil, fmt.Errorf("required env %s not set, please configure downward API", PodNameEnvVar)
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
	operatorNs, err := GetOperatorNamespace(logger)
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
// with wait=true it will wait, (util ctx timeout, please set it!) for the resource to be effectively deleted
func ComponentResourceRemoval(ctx context.Context, c client.Client, obj interface{}, hcoName string, logger logr.Logger, dryRun bool, wait bool) error {
	resource, err := toUnstructured(obj)
	if err != nil {
		logger.Error(err, "Failed to convert object to Unstructured")
		return err
	}

	logger.Info("Removing resource", "name", resource.GetName(), "namespace", resource.GetNamespace(), "GVK", resource.GetObjectKind().GroupVersionKind(), "dryRun", dryRun)

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
	if wait {
		foreground := metav1.DeletePropagationForeground
		opts.PropagationPolicy = &foreground
	}

	dErr := c.Delete(ctx, resource, opts)
	if dErr != nil {
		if apierrors.IsNotFound(dErr) {
			// to be idempotent if called on a object that was
			// already marked for deletion in a previous reconciliation loop
			return nil
		}
		// failure
		return dErr
	}
	if !wait || dryRun {
		return nil
	}

	for {
		err = c.Get(ctx, types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}, resource)
		if apierrors.IsNotFound(err) {
			// success!
			return nil
		}
		select {
		case <-ctx.Done():
			// failed to delete in time
			return errors.New(fmt.Sprintf("timed out waiting for %q - %q to be deleted", resource.GetObjectKind(), resource.GetName()))
		case <-time.After(100 * time.Millisecond):
			// do nothing, try again
		}
	}
	return errors.New(fmt.Sprintf("timed out waiting for %q - %q to be deleted", resource.GetObjectKind(), resource.GetName()))
}

// EnsureDeleted calls ComponentResourceRemoval if the runtime object exists
// with wait=true it will wait, (util ctx timeout, please set it!) for the resource to be effectively deleted
func EnsureDeleted(ctx context.Context, c client.Client, obj runtime.Object, hcoName string, logger logr.Logger, dryRun bool, wait bool) error {
	err := GetRuntimeObject(ctx, c, obj, logger)

	if err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			logger.Info("Resource doesn't exist, there is nothing to remove", "Kind", obj.GetObjectKind())
			return nil
		}

		logger.Error(err, "failed to get object from kubernetes", "Kind", obj.GetObjectKind())
		return err
	}

	return ComponentResourceRemoval(ctx, c, obj, hcoName, logger, dryRun, wait)
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

func ContainsString(s []string, word string) bool {
	for _, w := range s {
		if w == word {
			return true
		}
	}
	return false
}
