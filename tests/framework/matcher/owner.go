package matcher

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"

	"github.com/onsi/gomega/types"
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
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return false, nil
	}
	if len(u.GetOwnerReferences()) > 0 {
		return true, nil
	}
	return false, nil
}

func (o ownedMatcher) FailureMessage(actual interface{}) (message string) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to not be empty, but got '%v'", u)
}

func (o ownedMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("Expected owner references to be empty, but got '%v'", u)
}
