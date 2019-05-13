package testsuites

import (
	"fmt"
	"testing"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CscTargetNamespace is a test suit that confirms that targetNamespace values are handled appropriately.
func CscTargetNamespace(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	client := test.Global.Client

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}

	// Create a new CatalogSourceConfig with a non-existing targetNamespace.
	nonExistingTargetNamespaceCsc := &marketplace.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: marketplace.CatalogSourceConfigKind,
		}, ObjectMeta: metav1.ObjectMeta{
			Name:      cscName,
			Namespace: namespace,
		},
		Spec: marketplace.CatalogSourceConfigSpec{
			TargetNamespace: targetNamespace,
			Packages:        "camel-k-marketplace-e2e-tests",
		}}

	// Create the CatalogSourceConfig and if an error occurs do not run tests that
	// rely on the existence of the CatalogSourceConfig.
	// The CatalogSourceConfig is created with nil ctx and must be deleted manually before test suite exits.
	err = helpers.CreateRuntimeObject(client, nil, nonExistingTargetNamespaceCsc)
	if err != nil {
		t.Fatalf("Unable to create test CatalogSourceConfig: %v", err)
	}

	// Run tests.
	t.Run("configuring-state-when-target-namespace-dne", configuringStateWhenTargetNamespaceDoesNotExist)

	// Create a namespace based on the targetNamespace string.
	resultNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNamespace}}
	err = helpers.CreateRuntimeObject(client, ctx, resultNamespace)
	if err != nil {
		t.Fatalf("Unable to create test namespace: %v", err)
	}

	t.Run("succeeded-state-after-target-namespace-created", succeededStateAfterTargetNamespaceCreated)

	t.Run("child-resources-created", childResourcesCreated)

	// Delete the CatalogSourceConfig.
	err = helpers.DeleteRuntimeObject(client, nonExistingTargetNamespaceCsc)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("child-resources-deleted", childResourcesDeleted)
}

// configuringStateWhenTargetNamespaceDoesNotExist is a test case that creates a CatalogSourceConfig
// with a targetNamespace that doesn't exist and ensures that the status is updated to reflect the
// nonexistent namespace.
func configuringStateWhenTargetNamespaceDoesNotExist(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}

	// Check that the CatalogSourceConfig with an non-existing targetNamespace eventually reaches the
	// configuring phase with the expected message.
	expectedPhase := "Configuring"
	expectedMessage := fmt.Sprintf("namespaces \"%s\" not found", targetNamespace)
	err = helpers.WaitForExpectedPhaseAndMessage(test.Global.Client, cscName, namespace, expectedPhase, expectedMessage)
	if err != nil {
		t.Fatalf("CatalogSourceConfig never reached expected phase/message, expected %v/%v", expectedPhase, expectedMessage)
	}
}

// succeededStateAfterTargetNamespaceCreated is a test case that confirms that a csc that had a
// targetNamespace which did not exist eventually reaches a succeeded state once the targetNamespace is created.
func succeededStateAfterTargetNamespaceCreated(t *testing.T) {
	// Get test namespace.
	namespace, err := test.NewTestCtx(t).GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}

	// Now that the targetNamespace has been created, periodically check that the CatalogSourceConfig
	// has reached the Succeeded phase.
	expectedPhase := "Succeeded"
	err = helpers.WaitForExpectedPhaseAndMessage(test.Global.Client, cscName, namespace, expectedPhase, "")
	if err != nil {
		t.Fatalf("CatalogSourceConfig never reached expected phase, expected %v", expectedPhase)
	}
}

// childResourcesCreated checks that once a CatalogSourceConfig is created that all expected runtime
// objects are created as well.
func childResourcesCreated(t *testing.T) {
	// Get test namespace.
	namespace, err := test.NewTestCtx(t).GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}
	// Check that the CatalogSourceConfig and its child resources were created.
	err = helpers.CheckCscSuccessfulCreation(test.Global.Client, cscName, namespace, targetNamespace)
	if err != nil {
		t.Fatal(err)

	}
}

// childResourcesDeleted checks that once a CatalogSourceConfig is deleted that all expected runtime
// objects are deleted as well.
func childResourcesDeleted(t *testing.T) {
	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}

	// Check that the CatalogSourceConfig and its child resources were deleted.
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, cscName, namespace, targetNamespace)
	if err != nil {
		t.Fatalf("Could not ensure that CatalogSourceConfig and its child resources were deleted: %v", err)
	}
}
