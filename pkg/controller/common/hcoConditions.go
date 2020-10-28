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
	existingCondition, ok := hc[newCondition.Type]

	if !ok {
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

func (hc HcoConditions) Empty() bool {
	return len(hc) == 0
}
