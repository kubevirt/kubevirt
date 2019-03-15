package catalogsourceconfig

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
)

// Reconciler is the interface that wraps the Reconcile method.
//
// Reconcile reconciles a particular phase of the CatalogSourceConfig object
// and returns the next desired phase.
//
// in represents the original CatalogSourceConfig object received from the sdk
// and before reconciliation has started.
//
// out represents the CatalogSourceConfig object after reconciliation has
// completed and could be different from the original. The CatalogSourceConfig
// object received (in) should be deep copied into (out) before changes are
// made.
//
// nextPhase represents the next desired phase for the given CatalogSourceConfig
// object. If nil is returned, it implies that no phase transition is expected.
//
// On error, the next desired phase should be set appropriately.
//
// The CatalogSourceConfig object returned via out reflects the most recent
// version and hence should be used by the caller. The caller is responsible for
// calling update on the CatalogSourceConfig object returned so that the updated
// object is persistent. Reconcile must not call update.
//
// Reconcile operation is expected to be idempotent so that in the event of
// multiple invocations there would be no adverse or side effect.
type Reconciler interface {
	Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error)
}
