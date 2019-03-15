package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// NewSucceededReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Succeeded" phase.
func NewSucceededReconciler(logger *log.Entry) Reconciler {
	return &succeededReconciler{
		logger: logger,
	}
}

// succeededReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Succeeded" phase.
type succeededReconciler struct {
	logger *log.Entry
}

// Reconcile reconciles an OperatorSource object that is in "Succeeded" phase.
// Since this phase indicates that the object has been successfully reconciled,
// no further action is taken.
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
func (r *succeededReconciler) Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Succeeded {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	// No change is being made, so return the OperatorSource object that was specified as is.
	out = in

	r.logger.Info("No action taken, the object has already been reconciled")

	return out, nil, nil
}
