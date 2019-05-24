package testsuites

import (
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
)

// DeleteOpSrc tests that the correct cleanup occurs when an OpSrc is deleted
func DeleteOpSrc(t *testing.T) {
	t.Run("delete-operator-source", testDeleteOpSrc)
}

// testDeleteOpSrc ensures that after deleting an OperatorSource that the
// objects created as a result are deleted
func testDeleteOpSrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	testOperatorSource := helpers.CreateOperatorSourceDefinition(namespace)

	// Create the OperatorSource with no cleanup options.
	err = helpers.CreateRuntimeObjectNoCleanup(client, testOperatorSource)
	require.NoError(t, err, "Could not create operator source.")

	// Check for the datastore CatalogSourceConfig and its child resources.
	err = helpers.CheckCscSuccessfulCreation(test.Global.Client, testOperatorSource.Name, namespace, namespace)
	require.NoError(t, err, "Could not ensure that CatalogSourceConfig and its child resources were deleted")

	// Now let's delete the OperatorSource
	err = helpers.DeleteRuntimeObject(client, testOperatorSource)
	require.NoError(t, err, "OperatorSource could not be deleted successfully. Client returned error.")

	// Now let's wait until the OperatorSource is successfully deleted and the
	// child resources are removed.
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, testOperatorSource.Name, namespace, targetNamespace)
	require.NoError(t, err, "Could not ensure that CatalogSourceConfig and its child resources were deleted.")
}
