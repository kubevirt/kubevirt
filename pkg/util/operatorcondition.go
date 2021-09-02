package util

import (
	"context"

	operatorframeworkv2 "github.com/operator-framework/api/pkg/operators/v2"
	"github.com/operator-framework/operator-lib/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Condition - We just need the Set method in our code.
type Condition interface {
	Set(ctx context.Context, status metav1.ConditionStatus, reason, message string) error
}

// OperatorCondition wraps operator-lib's Condition to make it not crash,
// when running locally or in Kubernetes without OLM.
type OperatorCondition struct {
	cond conditions.Condition
}

const (
	UpgradeableInitReason  = "Initializing"
	UpgradeableInitMessage = "The hyperConverged cluster operator is starting up"

	UpgradeableUpgradingReason  = "AlreadyPerformingUpgrade"
	UpgradeableUpgradingMessage = "upgrading the hyperConverged cluster operator to version "

	UpgradeableAllowReason  = "Upgradeable"
	UpgradeableAllowMessage = ""
)

func NewOperatorCondition(clusterInfo ClusterInfo, c client.Client, condType string) (*OperatorCondition, error) {
	oc := &OperatorCondition{}
	if clusterInfo.IsRunningLocally() {
		// Don't try to update OperatorCondition when we don't run in cluster,
		// because operator-lib can't discover the namespace we are running in.
		return oc, nil
	}
	if !clusterInfo.IsManagedByOLM() {
		// We are not managed by OLM -> no OperatorCondition
		return oc, nil
	}

	cond, err := conditions.NewCondition(c, operatorframeworkv2.ConditionType(condType))
	if err != nil {
		return nil, err
	}
	oc.cond = cond
	return oc, nil
}

func (oc *OperatorCondition) Set(ctx context.Context, status metav1.ConditionStatus, reason, message string) error {
	if oc == nil || oc.cond == nil {
		// no op
		return nil
	}

	return oc.cond.Set(ctx, status, conditions.WithReason(reason), conditions.WithMessage(message))
}
