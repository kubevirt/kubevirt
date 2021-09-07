package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func BeRunning() types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: "Running",
	}
}

func HaveSucceeded() types.GomegaMatcher {
	return phaseMatcher{
		expectedPhase: "Succeeded",
	}
}

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
	if helper.IsNil(actual) {
		return false, nil
	}
	if helper.IsSlice(actual) {
		return helper.MatchElementsInSlice(actual, p.match)
	}
	return p.match(actual)
}

func (p phaseMatcher) match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
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
	if helper.IsNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if helper.IsSlice(actual) {
		return fmt.Sprintf("expected phases to be in %v but got %v", expectedPhase, collectPhasesForPrinting(actual))
	}

	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("expected phase is '%v' but got '%v'", expectedPhase, phase)
}

func (p phaseMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if helper.IsNil(actual) {
		return fmt.Sprintf("object does not exist")
	}
	expectedPhase := getExpectedPhase(p.expectedPhase)
	if helper.IsSlice(actual) {
		return fmt.Sprintf("expected phases to not be in %v but got %v", expectedPhase, collectPhasesForPrinting(actual))
	}
	phase, err := getCurrentPhase(actual)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("expected phase '%v' to not match '%v'", expectedPhase, phase)
}

func getCurrentPhase(actual interface{}) (string, error) {
	actual = helper.ToPointer(actual)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return "", err
	}
	str, _, err := unstructured.NestedString(obj, "status", "phase")
	return str, err
}

func collectPhasesForPrinting(actual interface{}) (phases []string) {
	helper.IterateOverSlice(actual, func(value interface{}) bool {
		if helper.IsNil(value) {
			phases = append(phases, "nil")
			return true
		}
		phase, err := getCurrentPhase(value)
		if err != nil {
			phase = err.Error()
		}
		phases = append(phases, phase)
		return true
	})
	return
}

func getExpectedPhase(actual interface{}) string {
	return reflect.ValueOf(actual).String()
}
