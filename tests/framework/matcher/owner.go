package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func BeOwned() types.GomegaMatcher {
	return ownedMatcher{}
}

func HaveOwners() types.GomegaMatcher {
	return ownedMatcher{}
}

type ownedMatcher struct {
}

func (o ownedMatcher) Match(actual interface{}) (success bool, err error) {
	u, err := toUnstructured(actual)
	if err != nil {
		return false, nil
	}
	if len(u.GetOwnerReferences()) > 0 {
		return true, nil
	}
	return false, nil
}

func (o ownedMatcher) FailureMessage(actual interface{}) (message string) {
	u, err := toUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to not be empty, but got '%v'", u)
}

func (o ownedMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	u, err := toUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to be empty, but got '%v'", u)
}

func toUnstructured(actual interface{}) (*unstructured.Unstructured, error) {
	if helper.IsNil(actual) {
		return nil, fmt.Errorf("object does not exist")
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
