package matcher

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/tests/framework/matcher/helper"
)

func HaveReadyReplicasNumerically(comparator string, compareTo ...interface{}) types.GomegaMatcher {
	return HaveReadyReplicas(gomega.BeNumerically(comparator, compareTo...))
}

func HaveReadyReplicas(comparator types.GomegaMatcher) *readiness {
	return &readiness{
		comparator: comparator,
	}
}

type readiness struct {
	comparator types.GomegaMatcher
}

func (r *readiness) Match(actual interface{}) (success bool, err error) {
	if helper.IsNil(actual) {
		return false, nil
	}
	readyReplicas, err := getReadyReplicaCount(actual)
	if err != nil {
		return false, err
	}
	return r.comparator.Match(readyReplicas)
}

func (r *readiness) FailureMessage(actual interface{}) (message string) {
	if helper.IsNil(actual) {
		return "object must not be nil"
	}
	readyReplicas, err := getReadyReplicaCount(actual)
	if err != nil {
		return fmt.Sprintf("failed extracting an error count from the object: %v", err)
	}
	return r.comparator.FailureMessage(readyReplicas)
}

func (r *readiness) NegatedFailureMessage(actual interface{}) (message string) {
	if helper.IsNil(actual) {
		return "object must not be nil"
	}
	readyReplicas, err := getReadyReplicaCount(actual)
	if err != nil {
		return fmt.Sprintf("failed extracting an error count from the object: %v", err)
	}
	return r.comparator.NegatedFailureMessage(readyReplicas)
}

func getReadyReplicaCount(actual interface{}) (int64, error) {
	actual = helper.ToPointer(actual)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return 0, err
	}
	str, _, err := unstructured.NestedInt64(obj, "status", "readyReplicas")
	return str, err
}
