package testlib

import (
	"fmt"

	gomegatypes "github.com/onsi/gomega/types"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
)

// RepresentCondition - returns a GomegaMatcher useful for comparing conditions
func RepresentCondition(expected conditionsv1.Condition) gomegatypes.GomegaMatcher {
	return &representConditionMatcher{
		expected: expected,
	}
}

type representConditionMatcher struct {
	expected conditionsv1.Condition
}

// Match - compares two conditions
// two conditions are the same if they have the same type, status, reason, and message
func (matcher *representConditionMatcher) Match(actual interface{}) (success bool, err error) {
	actualCondition, ok := actual.(conditionsv1.Condition)
	if !ok {
		return false, fmt.Errorf("RepresentConditionMatcher expects a Condition")
	}

	if matcher.expected.Type != actualCondition.Type {
		return false, nil
	}
	if matcher.expected.Status != actualCondition.Status {
		return false, nil
	}
	if matcher.expected.Reason != actualCondition.Reason {
		return false, nil
	}
	if matcher.expected.Message != actualCondition.Message {
		return false, nil
	}
	return true, nil
}

func (matcher *representConditionMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto match the condition\n\t%#v", actual, matcher.expected)
}

func (matcher *representConditionMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match the condition\n\t%#v", actual, matcher.expected)
}
