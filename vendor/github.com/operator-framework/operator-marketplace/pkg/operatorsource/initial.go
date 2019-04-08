package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// NewInitialReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Initial" phase.
func NewInitialReconciler(logger *log.Entry, datastore datastore.Writer) Reconciler {
	return &initialReconciler{
		logger:    logger,
		datastore: datastore,
	}
}

// initialReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Initial" phase.
type initialReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
}

// Reconcile reconciles an OperatorSource object that is in "Initial" phase.
// This is the first phase in the reconciliation process.
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
// Upon success, it returns "Validating" as the next desired phase.
func (r *initialReconciler) Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Initial {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in.DeepCopy()

	// When an opsrc is created, make sure the opsrc finalizer is included
	// in the object meta.
	out.EnsureFinalizer()

	// Make underlying datastore aware of the OperatorSource object that is
	// being reconciled.
	r.datastore.AddOperatorSource(in)

	r.logger.Info("Scheduling for validation")

	nextPhase = phase.GetNext(phase.OperatorSourceValidating)
	return
}
