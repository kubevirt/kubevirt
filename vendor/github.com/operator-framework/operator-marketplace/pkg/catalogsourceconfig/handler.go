package catalogsourceconfig

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/sirupsen/logrus"
)

func NewHandler(mgr manager.Manager, client client.Client) Handler {
	cache := NewCache()
	return &catalogsourceconfighandler{
		client: client,
		factory: &phaseReconcilerFactory{
			reader: datastore.Cache,
			client: client,
			cache:  cache,
		},
		transitioner: phase.NewTransitioner(),
		reader:       datastore.Cache,
		cache:        cache,
	}
}

// Handler is the interface that wraps the Handle method
type Handler interface {
	Handle(ctx context.Context, catalogSourceConfig *marketplace.CatalogSourceConfig) error
}

type catalogsourceconfighandler struct {
	client       client.Client
	factory      PhaseReconcilerFactory
	transitioner phase.Transitioner
	reader       datastore.Reader
	// cache gets updated everytime a CatalogSourceConfig is created or updated.
	// A cached item is removed when an update is detected.
	// Note: This is a temporary construct which will be removed when we move to
	// using the Operator Registry as the data store for CatalogSources
	cache Cache
}

// Handle handles a new event associated with the CatalogSourceConfig type.
func (h *catalogsourceconfighandler) Handle(ctx context.Context, in *marketplace.CatalogSourceConfig) error {

	log := getLoggerWithCatalogSourceConfigTypeFields(in)
	reconciler, err := h.factory.GetPhaseReconciler(log, in)
	if err != nil {
		return err
	}

	out, status, err := reconciler.Reconcile(ctx, in)
	if out == nil {
		// If the reconciler didn't return an object, that means it must have been deleted.
		// In that case, we should just return without attempting to modify it.
		return err
	}

	// If reconciliation threw an error, we can't quit just yet. We need to
	// figure out whether the CatalogSourceConfig object needs to be updated.
	if !h.transitioner.TransitionInto(&out.Status.CurrentPhase, status) {
		// CatalogSourceConfig object has not changed, no need to update. We are done.
		return err
	}

	// CatalogSourceConfig object has been changed. At this point,
	// reconciliation has either completed successfully or failed. In either
	// case, we need to update the modified CatalogSourceConfig object.
	if updateErr := h.client.Update(ctx, out); updateErr != nil {
		if err == nil {
			// No reconciliation err, but update of object has failed!
			return updateErr
		}

		// Presence of both Reconciliation error and object update error.
		log.Errorf("Failed to update object - %v", updateErr)

		// TODO: find a way to chain the update error?
		return err
	}

	return err
}

// getLoggerWithCatalogSourceConfigTypeFields returns a logger entry that can be
// used for consistent logging.
func getLoggerWithCatalogSourceConfigTypeFields(csc *marketplace.CatalogSourceConfig) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"type":            csc.TypeMeta.Kind,
		"targetNamespace": csc.Spec.TargetNamespace,
		"name":            csc.GetName(),
	})
}
