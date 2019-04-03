package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
)

// Reconciler is the interface that wraps the Reconcile method.
//
// Reconcile reconciles a particular phase of a given OperatorSource object
// and returns the next desired phase.
//
// in represents the original OperatorSource object received from the sdk
// and before reconciliation has started.
//
// out represents the OperatorSource object after reconciliation has completed
// and could be different from the original. The OperatorSource object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given OperatorSource
// object. If nil is returned, it implies that no phase transition is expected.
//
// On error, the next desired phase should be set appropriately.
//
// The OperatorSource object returned via out reflects the most recent version
// and hence should be used by the caller.
// The caller is responsible for calling update on the OperatorSource object
// returned so that the updated object is persistent.
// Reconcile must not call update.
//
// Reconcile operation is expected to be idempotent so that in the event of
// multiple invocations there would be no adverse or side effect.
type Reconciler interface {
	Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error)
}
