package helpers

import (
	"context"
	"testing"
	"time"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-sdk/pkg/test"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// RetryInterval defines the frequency at which we check for updates against the
	// k8s api when waiting for a specific condition to be true.
	RetryInterval = time.Second * 5

	// Timeout defines the amount of time we should spend querying the k8s api
	// when waiting for a specific condition to be true.
	Timeout = time.Minute * 5

	// TestOperatorSourceName is the name of an OperatorSource that is returned by
	// the CreateOperatorSource function.
	TestOperatorSourceName string = "test-operators"

	// TestOperatorSourceLabelKey is a label key added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelKey string = "opsrc-group"

	// TestOperatorSourceLabelValue is a label value added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelValue string = "Community"
)

// WaitForResult polls the cluster for a particular resource name and namespace.
// If the request fails because of an IsNotFound error it retries until the specified timeout.
// If it succeeds it sets the result runtime.Object to the requested object.
func WaitForResult(t *testing.T, result runtime.Object, namespace, name string) error {
	// Get global framework variables
	f := test.Global

	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
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
func WaitForSuccessfulDeployment(t *testing.T, deployment apps.Deployment) error {
	// Get global framework variables
	f := test.Global

	// If deployment is already ready, lets just return.
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return nil
	}

	namespacedName := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
	result := &apps.Deployment{}
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
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

// CreateRuntimeObject creates a runtime object using the test framework.
func CreateRuntimeObject(f *test.Framework, ctx *test.TestCtx, obj runtime.Object) error {
	return f.Client.Create(
		context.TODO(),
		obj,
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       time.Second * 30,
			RetryInterval: time.Second * 1,
		})
}

// CreateOperatorSourceDefinition returns an OperatorSource definition that can be turned into
// a runtime object for tests that rely on an OperatorSource
func CreateOperatorSourceDefinition(namespace string) *marketplace.OperatorSource {
	return &marketplace.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: marketplace.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestOperatorSourceName,
			Namespace: namespace,
			Labels: map[string]string{
				TestOperatorSourceLabelKey: TestOperatorSourceLabelValue,
			},
		},
		Spec: marketplace.OperatorSourceSpec{
			Type:              "appregistry",
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "marketplace_e2e",
		},
	}
}
