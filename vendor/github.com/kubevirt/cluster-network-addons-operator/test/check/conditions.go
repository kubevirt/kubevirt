package check

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
)

type ConditionType conditionsv1.ConditionType

type ConditionStatus corev1.ConditionStatus

const (
	ConditionAvailable   = ConditionType(conditionsv1.ConditionAvailable)
	ConditionProgressing = ConditionType(conditionsv1.ConditionProgressing)
	ConditionDegraded    = ConditionType(conditionsv1.ConditionDegraded)

	ConditionTrue  = ConditionStatus(corev1.ConditionTrue)
	ConditionFalse = ConditionStatus(corev1.ConditionFalse)
)
