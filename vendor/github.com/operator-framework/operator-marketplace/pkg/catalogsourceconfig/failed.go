package catalogsourceconfig

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
)

// NewFailedReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Failed" phase.
func NewFailedReconciler(log *logrus.Entry) Reconciler {
	return &failedReconciler{
		log: log,
	}
}

// failedReconciler is an implementation of Reconciler interface that
// reconciles an CatalogSourceConfig object in "Failed" phase.
type failedReconciler struct {
	log *logrus.Entry
}

// Reconcile reconciles an CatalogSourceConfig object that is in the "Failed"
// phase.
//
// Given that nil is returned here, it implies that no phase transition is
// expected.
func (r *failedReconciler) Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Failed {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	r.log.Info("No action taken, already in failed state")

	return out, nil, nil
}
