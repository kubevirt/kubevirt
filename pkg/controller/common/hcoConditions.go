package common

import (
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
)

var (
	HcoConditionTypes = []conditionsv1.ConditionType{
		hcov1beta1.ConditionReconcileComplete,
		conditionsv1.ConditionAvailable,
		conditionsv1.ConditionProgressing,
		conditionsv1.ConditionDegraded,
		conditionsv1.ConditionUpgradeable,
	}
)

type HcoConditions map[conditionsv1.ConditionType]conditionsv1.Condition

func NewHcoConditions() HcoConditions {
	return HcoConditions{}
}

func (hc HcoConditions) SetStatusCondition(newCondition conditionsv1.Condition) {
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

func (hc HcoConditions) SetStatusConditionIfUnset(newCondition conditionsv1.Condition) {
	if !hc.HasCondition(newCondition.Type) {
		hc.SetStatusCondition(newCondition)
	}
}

func (hc HcoConditions) IsEmpty() bool {
	return len(hc) == 0
}

func (hc HcoConditions) HasCondition(conditionType conditionsv1.ConditionType) bool {
	_, exists := hc[conditionType]

	return exists
}
