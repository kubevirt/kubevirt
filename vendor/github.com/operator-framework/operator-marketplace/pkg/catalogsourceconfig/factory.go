package catalogsourceconfig

import (
	"fmt"

	"github.com/sirupsen/logrus"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PhaseReconcilerFactory is the interface that wraps GetPhaseReconciler method.
//
// GetPhaseReconciler returns an appropriate phase.Reconciler based on the
// current phase of the csc object.
// The following chain shows how a CatalogSourceConfig object progresses through
// a series of transitions from the initial phase to complete reconciled state.
//
//  Initial --> Configuring --> Succeeded
//
// log is the prepared contextual logger that is to be used for logging.
// event represents the event fired by sdk, it is used to return the appropriate
// phase.Reconciler.
//
//  On error, the object is transitioned into "Failed" phase.
type PhaseReconcilerFactory interface {
	GetPhaseReconciler(log *logrus.Entry, csc *marketplace.CatalogSourceConfig) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	reader datastore.Reader
	client client.Client
	cache  Cache
}

func (f *phaseReconcilerFactory) GetPhaseReconciler(log *logrus.Entry, csc *marketplace.CatalogSourceConfig) (Reconciler, error) {
	// If the object has a deletion timestamp, it means it has been marked for
	// deletion. Return a deleted reconciler to remove that csc data from
	// the cache, and remove the finalizer so the garbage collector can
	// clean it up.
	if !csc.ObjectMeta.DeletionTimestamp.IsZero() {
		return NewDeletedReconciler(log, f.cache, f.client), nil
	}

	// Check if the cache is stale. A stale cache entry is an indicator that the
	// object has changed and requires updating. Only do this when the object has
	// already been created (not in the initial phase) and when it is not already
	// attempting to configure (which will populate the cache).
	if csc.Status.CurrentPhase.Name != phase.Initial && csc.Status.CurrentPhase.Name != phase.Configuring {
		pkgStale, targetStale := f.cache.IsEntryStale(csc)
		if pkgStale || targetStale {
			return NewUpdateReconciler(log, f.client, f.cache, targetStale), nil
		}
	}

	currentPhase := csc.Status.CurrentPhase.Name
	switch currentPhase {
	case phase.Initial:
		return NewInitialReconciler(log), nil

	case phase.Configuring:
		return NewConfiguringReconciler(log, f.reader, f.client, f.cache), nil

	case phase.Succeeded:
		return NewSucceededReconciler(log), nil

	case phase.Failed:
		return NewFailedReconciler(log), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for CatalogSourceConfig type [phase=%s]", currentPhase)
	}
}
