package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func BeInPhase(phase interface{}) types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: phase,
	}
}

func HavePhase(phase interface{}) types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: phase,
	}
}

type phaseMatcher struct {
	expectedPhase interface{}
}

func (p phaseMatcher) Match(actual interface{}) (success bool, err error) {
	if isNil(actual) {
		return false, nil
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return false, err
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if phase == expectedPhase {
		return true, nil
	}
	return false, nil
}

func (p phaseMatcher) FailureMessage(actual interface{}) (message string) {
	if isNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	return fmt.Sprintf("expected phase is %v but got %v", expectedPhase, phase)
}

func (p phaseMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if isNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	return fmt.Sprintf("expected phase %v to not match %v", expectedPhase, phase)
}

func getCurrentPhase(actual interface{}) (string, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return "", err
	}
	str, _, err := unstructured.NestedString(obj, "status", "phase")
	return str, err
}

func getExpectedPhase(actual interface{}) string {
	return reflect.ValueOf(actual).String()
}

func isNil(actual interface{}) bool {
	return actual == nil || reflect.ValueOf(actual).IsNil()
}
