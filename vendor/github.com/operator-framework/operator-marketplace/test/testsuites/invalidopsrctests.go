package testsuites

import (
	"testing"
	"strings"

	operator "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// endpointType represents the endpoint we currently support
	endpointType string = "appregistry"
	// marketplaceRegistryNamespace is the e2e registry namespace in quay
	marketplaceRegistryNamespace string = "marketplace_e2e"
)

// InvalidOpSrc tests OperatorSources created with invalid values
// to make sure the expected failure state is reached
func InvalidOpSrc(t *testing.T) {
	t.Run("invalid-endpoint", testOpSrcWithInvalidEndpoint)
	t.Run("invalid-url", testOpSrcWithInvalidURL)
	t.Run("nonexistent-registry-namespace", testOpSrcWithNonexistentRegistryNamespace)
}

// Create OperatorSource with invalid endpoint
// Expected result: OperatorSource stuck in downloading state
func testOpSrcWithInvalidEndpoint(t *testing.T) {
	opSrcName := "invalid-endpoint-opsrc"
	// invalidEndpoint is the invalid endpoint for the OperatorSource
	invalidEndpoint := "https://not-quay.io/cnr"

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Errorf("Could not get namespace: %v", err)
	}

	invalidURLOperatorSource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opSrcName,
			Namespace: namespace,
		},
		Spec: operator.OperatorSourceSpec{
			Type:              endpointType,
			Endpoint:          invalidEndpoint,
			RegistryNamespace: marketplaceRegistryNamespace,
		},
	}
	err = helpers.CreateRuntimeObject(client, ctx, invalidURLOperatorSource)
	if err != nil {
		t.Errorf("Could not create OperatorSource: %v", err)
	}

	// Check that OperatorSource is in "Downloading" state with appropriate message
	resultOperatorSource := &operator.OperatorSource{}
	expectedPhase := "Downloading"
	err = wait.Poll(helpers.RetryInterval, helpers.Timeout, func() (bool, error) {
		err = helpers.WaitForResult(client, resultOperatorSource, namespace, opSrcName)
		if err != nil {
			return false, err
		}
		if resultOperatorSource.Status.CurrentPhase.Name == expectedPhase &&
			strings.Contains(resultOperatorSource.Status.CurrentPhase.Message, "no such host") {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("OperatorSource never reached expected phase/message, expected %v: %v", expectedPhase, err)
	}
}

// Create OperatorSource with invalid URL
// Expected result: OperatorSource reaches failed state
func testOpSrcWithInvalidURL(t *testing.T) {
	opSrcName := "invalid-url-opsrc"
	// invalidURL is an invalid URI
	invalidURL := "not-a-url"

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Errorf("Could not get namespace: %v", err)
	}

	invalidURLOperatorSource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opSrcName,
			Namespace: namespace,
		},
		Spec: operator.OperatorSourceSpec{
			Type:              endpointType,
			Endpoint:          invalidURL,
			RegistryNamespace: marketplaceRegistryNamespace,
		},
	}
	err = helpers.CreateRuntimeObject(client, ctx, invalidURLOperatorSource)
	if err != nil {
		t.Errorf("Could not create OperatorSource: %v", err)
	}

	// Check that OperatorSource reaches "Failed" state eventually
	resultOperatorSource := &operator.OperatorSource{}
	expectedPhase := "Failed"
	err = wait.Poll(helpers.RetryInterval, helpers.Timeout, func() (bool, error) {
		err = helpers.WaitForResult(client, resultOperatorSource, namespace, opSrcName)
		if err != nil {
			return false, err
		}
		if resultOperatorSource.Status.CurrentPhase.Name == expectedPhase &&
			strings.Contains(resultOperatorSource.Status.CurrentPhase.Message, "Invalid operator source endpoint") {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("OperatorSource never reached expected phase/message, expected %v: %v", expectedPhase, err)
	}
}

// Create OperatorSource with valid URL but non-existent registry namespace
// Expected result: OperatorSource reaches failed state
func testOpSrcWithNonexistentRegistryNamespace(t *testing.T) {
	opSrcName := "nonexistent-namespace-opsrc"
	// validURL is a valid endpoint for the OperatorSource
	validURL := "https://quay.io/cnr"

	// nonexistentRegistryNamespace is a namespace that does not exist
	// on the app registry
	nonexistentRegistryNamespace := "not-existent-namespace"

	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Errorf("Could not get namespace: %v", err)
	}

	nonexistentRegistryNamespaceOperatorSource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opSrcName,
			Namespace: namespace,
		},
		Spec: operator.OperatorSourceSpec{
			Type:              endpointType,
			Endpoint:          validURL,
			RegistryNamespace: nonexistentRegistryNamespace,
		},
	}
	err = helpers.CreateRuntimeObject(client, ctx, nonexistentRegistryNamespaceOperatorSource)
	if err != nil {
		t.Errorf("Could not create OperatorSource: %v", err)
	}

	// Check that OperatorSource reaches "Failed" state eventually
	resultOperatorSource := &operator.OperatorSource{}
	expectedPhase := "Failed"
	err = wait.Poll(helpers.RetryInterval, helpers.Timeout, func() (bool, error) {
		err = helpers.WaitForResult(client, resultOperatorSource, namespace, opSrcName)
		if err != nil {
			return false, err
		}
		if resultOperatorSource.Status.CurrentPhase.Name == expectedPhase &&
			strings.Contains(resultOperatorSource.Status.CurrentPhase.Message, "The operator source endpoint returned an empty manifest list") {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("OperatorSource never reached expected phase/message, expected %v: %v", expectedPhase, err)
	}
}
