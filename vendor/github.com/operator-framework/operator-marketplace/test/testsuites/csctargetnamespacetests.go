package testsuites

import (
	"fmt"
	"testing"

	operator "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CscTargetNamespace is a test suit that confirms that the targetNamespace field within
// a CatalogSourceConfig is handled correctly.
func CscTargetNamespace(t *testing.T) {
	t.Run("non-existing-target-namespace", testCscWithNonExistingTargetNamespace)
}

// testCscWithNonExistingTargetNamespace creates context and calls the
// cscWithNonExistingTargetNamespace test case.
func testCscWithNonExistingTargetNamespace(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	f := test.Global

	// Run tests.
	if err := cscWithNonExistingTargetNamespace(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// cscWithNonExistingTargetNamespace is a test case that creates a CatalogSourceConfig
// with a targetNamespace that doesn't exist and ensures that the status is updated to reflect the
// nonexistant namespace. The test then creates the targeted namespace and ensures that the
// CatalogSourceConfig is properly reconciled.
func cscWithNonExistingTargetNamespace(t *testing.T, f *test.Framework, ctx *test.TestCtx) error {
	// Get test namespace.
	namespace, err := test.NewTestCtx(t).GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)
	}

	// nonExistingTargetNamespaceCscName is the name of the CatalogSourceConfig that points
	// to a non-existing targetNamespace.
	nonExistingTargetNamespaceCscName := "non-existing-namespace-operators"

	// targetNamespace is the non-existing target namespace.
	targetNamespace := "non-existing-namespace"

	// Create a new CatalogSourceConfig with a non-existing targetNamespace.
	nonExistingTargetNamespaceCsc := &operator.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		}, ObjectMeta: metav1.ObjectMeta{
			Name:      nonExistingTargetNamespaceCscName,
			Namespace: namespace,
		},
		Spec: operator.CatalogSourceConfigSpec{
			TargetNamespace: targetNamespace,
			Packages:        "descheduler",
		}}
	err = helpers.CreateRuntimeObject(f, ctx, nonExistingTargetNamespaceCsc)
	if err != nil {
		return err
	}

	// Check that we created the CatalogSourceConfig with a non-existing targetNamespace.
	resultCatalogSourceConfig := &operator.CatalogSourceConfig{}
	err = helpers.WaitForResult(t, resultCatalogSourceConfig, namespace, nonExistingTargetNamespaceCscName)
	if err != nil {
		return err
	}

	// Check if the CatalogSourceConfig phase and message are the expected values.
	expectedPhase := "Configuring"
	expectedMessage := fmt.Sprintf("namespaces \"%s\" not found", targetNamespace)
	// Check that the CatalogSourceConfig with an non-existing targetNamespace eventually reaches the
	// configuring phase with the expected message.
	err = wait.Poll(helpers.RetryInterval, helpers.Timeout, func() (bool, error) {
		// CatalogSourceConfig should exist so no wait.
		err = helpers.WaitForResult(t, resultCatalogSourceConfig, namespace, nonExistingTargetNamespaceCscName)
		if err != nil {
			return false, err
		}
		if resultCatalogSourceConfig.Status.CurrentPhase.Name == expectedPhase &&
			resultCatalogSourceConfig.Status.CurrentPhase.Message == expectedMessage {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("CatalogSourceConfig never reached expected phase/message, expected %v/%v", expectedPhase, expectedMessage)
	}

	// Create a namespace based on the targetNamespace string.
	targetNamespaceRuntimeObject := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNamespace}}
	err = helpers.CreateRuntimeObject(f, ctx, targetNamespaceRuntimeObject)
	if err != nil {
		return err
	}

	// Now that the targetNamespace has been created, periodically check that the CatalogSourceConfig
	// has reached the Succeeded phase.
	expectedPhase = "Succeeded"
	err = wait.Poll(helpers.RetryInterval, helpers.Timeout, func() (bool, error) {
		// CatalogSourceConfig should exist so no wait.
		err = helpers.WaitForResult(t, resultCatalogSourceConfig, namespace, nonExistingTargetNamespaceCscName)
		if err != nil {
			return false, err
		}
		if resultCatalogSourceConfig.Status.CurrentPhase.Name == expectedPhase {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("CatalogSourceConfig never reached expected phase/message, expected %v", expectedPhase)
	}

	return nil
}
