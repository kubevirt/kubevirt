package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewHandlerWithParams returns a new Handler.
func NewHandlerWithParams(client client.Client, datastore datastore.Writer, factory PhaseReconcilerFactory, transitioner phase.Transitioner, newCacheReconciler NewOutOfSyncCacheReconcilerFunc) Handler {
	return &operatorsourcehandler{
		client:             client,
		datastore:          datastore,
		factory:            factory,
		transitioner:       transitioner,
		newCacheReconciler: newCacheReconciler,
	}
}

func NewHandler(mgr manager.Manager, client client.Client) Handler {
	return &operatorsourcehandler{
		client:    client,
		datastore: datastore.Cache,
		factory: &phaseReconcilerFactory{
			registryClientFactory: appregistry.NewClientFactory(),
			datastore:             datastore.Cache,
			client:                client,
			refresher:             cscRefresher,
		},
		transitioner:       phase.NewTransitioner(),
		newCacheReconciler: NewOutOfSyncCacheReconciler,
	}
}

// Handler is the interface that wraps the Handle method
//
// Handle handles a new event associated with OperatorSource type.
//
// ctx represents the parent context.
// event encapsulates the event fired by operator sdk.
type Handler interface {
	Handle(ctx context.Context, operatorSource *marketplace.OperatorSource) error
}

// operatorsourcehandler implements the Handler interface
type operatorsourcehandler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client             client.Client
	datastore          datastore.Writer
	factory            PhaseReconcilerFactory
	transitioner       phase.Transitioner
	newCacheReconciler NewOutOfSyncCacheReconcilerFunc
}

func (h *operatorsourcehandler) Handle(ctx context.Context, in *marketplace.OperatorSource) error {
	logger := log.WithFields(log.Fields{
		"type":      in.TypeMeta.Kind,
		"namespace": in.GetNamespace(),
		"name":      in.GetName(),
	})

	outOfSyncCacheReconciler := h.newCacheReconciler(logger, h.datastore, h.client)
	out, status, err := outOfSyncCacheReconciler.Reconcile(ctx, in)
	if err != nil {
		return err
	}

	// The out of sync cache reconciler has handled the event. So we should
	// transition into the new phase and return.
	if status != nil {
		return h.transition(ctx, logger, out, status, err)
	}

	// The out of sync cache reconciler has not handled the event, so let's use
	// the regular phase reconciler to handle this event.
	phaseReconciler, err := h.factory.GetPhaseReconciler(logger, in)
	if err != nil {
		return err
	}

	out, status, err = phaseReconciler.Reconcile(ctx, in)
	if out == nil {
		// If the reconciler didn't return an object, that means it must have been deleted.
		// In that case, we should just return without attempting to modify it.
		return err
	}

	return h.transition(ctx, logger, out, status, err)
}

func (h *operatorsourcehandler) transition(ctx context.Context, logger *log.Entry, opsrc *marketplace.OperatorSource, nextPhase *marketplace.Phase, reconciliationErr error) error {
	// If reconciliation threw an error, we can't quit just yet. We need to
	// figure out whether the OperatorSource object needs to be updated.

	if !h.transitioner.TransitionInto(&opsrc.Status.CurrentPhase, nextPhase) {
		// OperatorSource object has not changed, no need to update. We are done.
		return reconciliationErr
	}

	// OperatorSource object has been changed. At this point, reconciliation has
	// either completed successfully or failed.
	// In either case, we need to update the modified OperatorSource object.
	if updateErr := h.client.Update(ctx, opsrc); updateErr != nil {
		if reconciliationErr == nil {
			// No reconciliation err, but update of object has failed!
			return updateErr
		}

		// Presence of both Reconciliation error and object update error.
		logger.Errorf("Failed to update object - %v", updateErr)

		// TODO: find a way to chain the update error?
		return reconciliationErr
	}

	return reconciliationErr
}
