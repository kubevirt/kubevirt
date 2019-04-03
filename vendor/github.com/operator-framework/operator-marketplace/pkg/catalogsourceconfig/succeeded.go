package catalogsourceconfig

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
)

// NewSucceededReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Succeeded" phase.
func NewSucceededReconciler(log *logrus.Entry) Reconciler {
	return &succeededReconciler{
		log: log,
	}
}

// succeededReconciler is an implementation of Reconciler interface that
// reconciles an CatalogSourceConfig object in the "Succeeded" phase.
type succeededReconciler struct {
	log *logrus.Entry
}

// Reconcile reconciles an CatalogSourceConfig object that is in "Succeeded"
// phase. Since this phase indicates that the object has been successfully
// reconciled, no further action is taken.
//
// Given that nil is returned here, it implies that no phase transition is
// expected.
func (r *succeededReconciler) Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Succeeded {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	// No change is being made, so return the CatalogSourceConfig object as is.
	out = in

	r.log.Info("No action taken, the object has already been reconciled")

	return out, nil, nil
}
