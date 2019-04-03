package phase

import (
	"errors"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
)

// The following list is the set of phases a Marketplace object can be in while
// it is going through its reconciliation process.
const (
	// This phase applies to when an object has been created and the Phase
	// attribute is empty.
	Initial = ""

	// In this phase, for OperatorSource objects we ensure that a corresponding
	// CatalogSourceConfig object is created. For CatalogSourceConfig objects,
	// we ensure that an Operator-Registry pod is created and is associated with a
	// CatalogSource.
	Configuring = "Configuring"

	// This phase indicates that the object has been successfully reconciled.
	Succeeded = "Succeeded"

	// This phase indicates that reconciliation of the object has failed.
	Failed = "Failed"
)

// The following list is the set of OperatorSource specific phases
const (
	// In this phase we validate the OperatorSource object.
	OperatorSourceValidating = "Validating"

	// In this phase, we connect to the specified registry, download available
	// manifest(s) and save them to the underlying datastore.
	OperatorSourceDownloading = "Downloading"

	// In this phase, the given OperatorSource object is purged. All resource(s)
	// created as a result of reconciliation are removed in this phase.
	//
	// The following scenarios should trigger this phase:
	//   a. An admin changes the spec of the given OperatorSource object. This
	//      warrants a purge and reconciliation to start anew.
	//   b. When marketplace operator restarts it needs to rebuild the cache for
	//      all existing OperatorSource object(s).
	OperatorSourcePurging = "Purging"
)

var (
	// Default descriptive message associated with each phase.
	phaseMessages = map[string]string{
		OperatorSourceValidating:  "Scheduled for validation",
		OperatorSourceDownloading: "Scheduled for download of operator manifest(s)",
		OperatorSourcePurging:     "Scheduled for purging",
		Configuring:               "Scheduled for configuration",
		Succeeded:                 "The object has been successfully reconciled",
		Failed:                    "Reconciliation has failed",

		// This message is set by Marketplace operator when an OperatorSource
		// object has been purged and scheduled for reconciliation from the
		// Initial phase. Please note, when an admin creates an OperatorSource
		// object the message is empty.
		Initial: "Out of sync, scheduled for phased reconciliation",
	}
	// ErrWrongReconcilerInvoked is thrown when a wrong reconciler is invoked.
	ErrWrongReconcilerInvoked = errors.New("Wrong phase reconciler invoked for the given object")
)

// GetMessage returns the default message associated with a
// particular phase.
func GetMessage(phaseName string) string {
	return phaseMessages[phaseName]
}

// GetNext returns a Phase object with the given phase name and default message
func GetNext(name string) *marketplace.Phase {
	return marketplace.NewPhase(name, GetMessage(name))
}

// GetNextWithMessage returns a Phase object with the given phase name and default message
func GetNextWithMessage(name string, message string) *marketplace.Phase {
	return marketplace.NewPhase(name, message)
}
