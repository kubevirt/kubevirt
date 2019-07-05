package check

import (
	corev1 "k8s.io/api/core/v1"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

type ConditionType opv1alpha1.NetworkAddonsConditionType

type ConditionStatus corev1.ConditionStatus

const (
	ConditionAvailable   = ConditionType(opv1alpha1.NetworkAddonsConditionAvailable)
	ConditionProgressing = ConditionType(opv1alpha1.NetworkAddonsConditionProgressing)
	ConditionFailing     = ConditionType(opv1alpha1.NetworkAddonsConditionFailing)

	ConditionTrue  = ConditionStatus(corev1.ConditionTrue)
	ConditionFalse = ConditionStatus(corev1.ConditionFalse)
)
