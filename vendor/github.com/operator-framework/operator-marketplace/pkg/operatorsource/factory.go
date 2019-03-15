package operatorsource

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// PhaseReconcilerFactory is an interface that wraps the GetPhaseReconciler
// method.
type PhaseReconcilerFactory interface {
	// GetPhaseReconciler returns an appropriate phase.Reconciler based on the
	// current phase of an OperatorSource object.
	// The following chain shows how an OperatorSource object progresses through
	// a series of transitions from the initial phase to complete reconciled state.
	//
	//  Initial --> Validating --> Downloading --> Configuring --> Succeeded
	//     ^
	//     |
	//  Purging
	//
	// logger is the prepared contextual logger that is to be used for logging.
	// opsrc represents the given OperatorSource object
	//
	// On error, the object is transitioned into "Failed" phase.
	GetPhaseReconciler(logger *log.Entry, opsrc *marketplace.OperatorSource) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	registryClientFactory appregistry.ClientFactory
	datastore             datastore.Writer
	client                client.Client
	refresher             PackageRefreshNotificationSender
}

func (s *phaseReconcilerFactory) GetPhaseReconciler(logger *log.Entry, opsrc *marketplace.OperatorSource) (Reconciler, error) {
	currentPhase := opsrc.GetCurrentPhaseName()

	// If the object has a deletion timestamp, it means it has been marked for
	// deletion. Return a deleted reconciler to remove that opsrc data from
	// the datastore, and remove the finalizer so the garbage collector can
	// clean it up.
	if !opsrc.ObjectMeta.DeletionTimestamp.IsZero() {
		return NewDeletedReconciler(logger, s.datastore, s.client), nil
	}

	switch currentPhase {
	case phase.Initial:
		return NewInitialReconciler(logger, s.datastore), nil

	case phase.OperatorSourceValidating:
		return NewValidatingReconciler(logger, s.datastore), nil

	case phase.OperatorSourceDownloading:
		return NewDownloadingReconciler(logger, s.registryClientFactory, s.datastore, s.client, s.refresher), nil

	case phase.Configuring:
		return NewConfiguringReconciler(logger, s.datastore, s.client), nil

	case phase.OperatorSourcePurging:
		return NewPurgingReconciler(logger, s.datastore, s.client), nil

	case phase.Succeeded:
		return NewSucceededReconciler(logger), nil

	case phase.Failed:
		return NewFailedReconciler(logger), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for OperatorSource type [phase=%s]", currentPhase)
	}
}
