package status

import (
	configv1 "github.com/openshift/api/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
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
