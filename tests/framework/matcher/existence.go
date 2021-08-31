package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func Exist() types.GomegaMatcher {
	return existMatcher{}
}

func BePresent() types.GomegaMatcher {
	return existMatcher{}
}

func BeGone() types.GomegaMatcher {
	return goneMatcher{}
}

type existMatcher struct {
}

func (e existMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}
	return true, nil
}

func (e existMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to still exist, but it is gone: %v", actual)
}

func (e existMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to be gone, but it still exists: %v", actual)
}

type goneMatcher struct {
}

func (g goneMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return true, nil
	}
	if helper.IsSlice(actual) && reflect.ValueOf(actual).Len() == 0 {
		return true, nil
	}
	return false, nil
}

func (g goneMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to be gone, but it still exists: %v", actual)
}

func (g goneMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected object to still exist, but it is gone: %v", actual)
}
