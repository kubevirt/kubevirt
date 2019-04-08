package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// NewFailedReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Failed" phase.
func NewFailedReconciler(logger *log.Entry) Reconciler {
	return &failedReconciler{
		logger: logger,
	}
}

// failedReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Failed" phase.
type failedReconciler struct {
	logger *log.Entry
}

// Reconcile reconciles an OperatorSource object that is in "Failed" phase.
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
func (r *failedReconciler) Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Failed {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	r.logger.Info("No action taken, already in failed state")

	return out, nil, nil
}
