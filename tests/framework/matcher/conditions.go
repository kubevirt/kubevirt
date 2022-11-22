package matcher

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func HaveConditionMissingOrFalse(conditionType interface{}) types.GomegaMatcher {
	return conditionMatcher{
		expectedType:          conditionType,
		expectedStatus:        k8sv1.ConditionFalse,
		conditionCanBeMissing: true,
	}
}

func HaveConditionTrue(conditionType interface{}) types.GomegaMatcher {
	return conditionMatcher{
		expectedType:          conditionType,
		expectedStatus:        k8sv1.ConditionTrue,
		conditionCanBeMissing: false,
	}
}

func HaveConditionFalse(conditionType interface{}) types.GomegaMatcher {
	return conditionMatcher{
		expectedType:          conditionType,
		expectedStatus:        k8sv1.ConditionFalse,
		conditionCanBeMissing: false,
	}
}

type conditionMatcher struct {
	expectedType          interface{}
	expectedStatus        k8sv1.ConditionStatus
	conditionCanBeMissing bool
}

func (c conditionMatcher) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}

	if helper.IsSlice(actual) {
		// Not implemented
		return false, nil
	}

	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return false, err
	}

	if _, exist := u.Object["status"]; !exist {
		return false, fmt.Errorf("object doesn't contain status")
	}

	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if !found {
		if c.conditionCanBeMissing {
			return true, nil
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}

	conditionStatus, err := c.getStatusForExpectedConditionType(conditions)
	if err != nil {
		return false, err
	}

	if conditionStatus == "" {
		if c.conditionCanBeMissing {
			return true, nil
		}
		return false, nil
	}
	return conditionStatus == string(c.expectedStatus), nil
}

func (c conditionMatcher) FailureMessage(actual interface{}) string {
	if helper.IsSlice(actual) {
		// Not implemented
		return "slices are not implemented"
	}

	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}

	if _, exist := u.Object["status"]; !exist {
		return "object doesn't contain status"
	}

	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if !found {
		if c.conditionCanBeMissing {
			return ""
		}
		return format.Message(actual, "to contain conditions")

	}
	if err != nil {
		return err.Error()
	}

	conditionStatus, err := c.getStatusForExpectedConditionType(conditions)
	if err != nil {
		return err.Error()
	}

	if conditionStatus == "" {
		if c.conditionCanBeMissing {
			return ""
		}
		return format.Message(conditions, fmt.Sprintf("expected condition of type '%s' was not found", c.expectedType))
	}
	return format.Message(conditions, fmt.Sprintf("to find condition of type '%v' and status '%s' but got '%s'", c.expectedType, c.expectedStatus, conditionStatus))
}

func (c conditionMatcher) NegatedFailureMessage(actual interface{}) string {
	if helper.IsSlice(actual) {
		// Not implemented
		return "slices are not implemented"
	}

	u, err := helper.ToUnstructured(actual)
	if err != nil {
		return err.Error()
	}

	if _, exist := u.Object["status"]; !exist {
		return "object doesn't contain status"
	}

	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if !found {
		if c.conditionCanBeMissing {
			return format.Message(conditions, fmt.Sprintf("to find condition of type '%s'", c.expectedType))
		}
		return "object doesn't contain conditions"
	}
	if err != nil {
		return err.Error()
	}

	conditionStatus, err := c.getStatusForExpectedConditionType(conditions)
	if err != nil {
		return err.Error()
	}

	if conditionStatus == "" {
		if c.conditionCanBeMissing {
			return format.Message(conditions, fmt.Sprintf("to find condition of type '%s'", c.expectedType))
		}
		return format.Message(conditions, fmt.Sprintf("expected condition of type '%s' was not found", c.expectedType))
	}
	return format.Message(conditions, fmt.Sprintf("to not find condition of type '%v' with status '%s'", c.expectedType, c.expectedStatus))
}

func (c conditionMatcher) getExpectedType() string {
	return reflect.ValueOf(c.expectedType).String()
}

func (c conditionMatcher) getStatusForExpectedConditionType(conditions []interface{}) (status string, err error) {
	for _, condition := range conditions {
		condition, err := helper.ToUnstructured(condition)
		if err != nil {
			return "", err
		}

		foundType, found, err := unstructured.NestedString(condition.Object, "type")
		if !found {
			return "", fmt.Errorf("conditions don't contain type")
		}
		if err != nil {
			return "", err
		}

		if foundType == c.getExpectedType() {
			value, found, err := unstructured.NestedString(condition.Object, "status")
			if !found {
				return "", fmt.Errorf("conditions don't contain status")
			}
			if err != nil {
				return "", err
			}

			return value, nil
		}
	}

	return "", nil
}
