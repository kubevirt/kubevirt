package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/test"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	retryInterval        = time.Second * 5
	timeout              = time.Minute * 5
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 30
)

// WaitForResult polls the cluster for a particular resource name and namespace
// If the request fails because of an IsNotFound error it retries until the specified timeout
// If it succeeds it sets the result runtime.Object to the requested object
func WaitForResult(t *testing.T, f *test.Framework, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if errors.IsNotFound(err) {
				t.Logf("Waiting for creation of %s runtime object\n", name)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Runtime object %s has been created\n", name)
	return nil
}

// WaitForSuccessfulDeployment checks if a given deployment has readied all of
// its replicas. If it has not, it retries until the deployment is ready or it
// reaches the timeout.
func WaitForSuccessfulDeployment(t *testing.T, f *test.Framework, deployment apps.Deployment) error {
	// If deployment is already ready, lets just return.
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return nil
	}

	namespacedName := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
	result := &apps.Deployment{}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		if *deployment.Spec.Replicas == result.Status.ReadyReplicas {
			return true, nil
		}
		t.Logf("Waiting for deployment %s to have (%d/%d) replicas ready\n", deployment.Name, result.Status.ReadyReplicas,
			*deployment.Spec.Replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Deployment %s has been initialized successfully\n", deployment.Name)
	return nil
}

// createRuntimeObject creates a runtime object using the test framework
func createRuntimeObject(f *test.Framework, ctx *test.TestCtx, obj runtime.Object) error {
	return f.Client.Create(
		context.TODO(),
		obj,
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       cleanupTimeout,
			RetryInterval: cleanupRetryInterval,
		})
}
