package status

import (
	configv1 "github.com/openshift/api/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// compareArrayClusterOperatorStatusConditions takes two arrays of
// ClusterOperatorStatusCondition and returns true if the contents
// match.
func compareClusterOperatorStatusConditionArrays(a []configv1.ClusterOperatorStatusCondition, b []configv1.ClusterOperatorStatusCondition) bool {
	if len(a) != len(b) {
		return false
	}

	for _, aCondition := range a {
		bCondition := cohelpers.FindStatusCondition(b, aCondition.Type)
		if bCondition == nil {
			return false
		}
		isEqual := false
		if compareClusterOperatorStatusConditions(aCondition, *bCondition) {
			isEqual = true
		}
		if !isEqual {
			return false
		}
	}
	return true
}

// compareClusterOperatorStatusConditions takes two ClusterOperatorStatusCondition
// and returns true if the Name, Type, and Message match.
func compareClusterOperatorStatusConditions(a configv1.ClusterOperatorStatusCondition, b configv1.ClusterOperatorStatusCondition) bool {
	if a.Type == b.Type && a.Status == b.Status && a.Message == b.Message {
		return true
	}
	return false
}

func clusterStatusListBuilder() func(conditionType configv1.ClusterStatusConditionType, conditionStatus configv1.ConditionStatus, conditionMessage string) []configv1.ClusterOperatorStatusCondition {
	time := v1.Now()
	list := []configv1.ClusterOperatorStatusCondition{}
	return func(conditionType configv1.ClusterStatusConditionType, conditionStatus configv1.ConditionStatus, conditionMessage string) []configv1.ClusterOperatorStatusCondition {
		list = append(list, configv1.ClusterOperatorStatusCondition{
			Type:               conditionType,
			Status:             conditionStatus,
			Message:            conditionMessage,
			LastTransitionTime: time,
		})
		return list
	}
}
