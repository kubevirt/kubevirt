package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
)

var (
	HcoConditionTypes = []string{
		hcov1beta1.ConditionReconcileComplete,
		hcov1beta1.ConditionAvailable,
		hcov1beta1.ConditionProgressing,
		hcov1beta1.ConditionDegraded,
		hcov1beta1.ConditionUpgradeable,
	}
)

type HcoConditions map[string]metav1.Condition

func NewHcoConditions() HcoConditions {
	return HcoConditions{}
}

func (hc HcoConditions) SetStatusCondition(newCondition metav1.Condition) {
	existingCondition, exists := hc[newCondition.Type]

	if !exists {
		hc[newCondition.Type] = newCondition
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	hc[newCondition.Type] = existingCondition
}

func (hc HcoConditions) SetStatusConditionIfUnset(newCondition metav1.Condition) {
	if !hc.HasCondition(newCondition.Type) {
		hc.SetStatusCondition(newCondition)
	}
}

func (hc HcoConditions) IsEmpty() bool {
	return len(hc) == 0
}

func (hc HcoConditions) HasCondition(conditionType string) bool {
	_, exists := hc[conditionType]

	return exists
}
