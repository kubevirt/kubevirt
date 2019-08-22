package operatorstatus

import (
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/util/clock"
)

const (
	// versionName reflects the name of the version CVO expects in Status.
	versionName = "operator"
)

// newCSVStatusReporter returns a new instance of CSVStatusReporter
func newCSVStatusReporter(releaseVersion string) *csvStatusReporter {
	return &csvStatusReporter{
		clock:          &clock.RealClock{},
		releaseVersion: releaseVersion,
	}
}

// csvStatusReporter provides the logic for initialzing ClusterOperator and
// ClusterOperatorStatus types.
type csvStatusReporter struct {
	clock          clock.Clock
	releaseVersion string
}

// NewNotification prepares a new notification event to be sent to the monitor.
func (r *csvStatusReporter) NewNotification(context *csvEventContext) NotificationFunc {
	return func() (name string, mutator MutatorFunc) {
		name = context.Name
		mutator = func(existing *configv1.ClusterOperatorStatus) (new *configv1.ClusterOperatorStatus) {
			new = r.GetNewStatus(existing, context)
			return
		}

		return
	}
}

// GetNewStatus returns the expected new status based on the notification context.
// We cover the following scenarios:
// a. Fresh install of an operator (v1), no previous version is installed.
//   1. Working toward v1
//   2. v1 successfully installed
//   3. v1 deploy failed
//   4. v1 has been removed, post successful install.
//
// b. Newer version of the operator (v2) is being installed (v1 is already installed)
//   1. Working toward v2. (v1 is being replaced, it waits until v2 successfully
//	    is successfully installed) Is v1 available while v2 is being installed?
//   2. When v1 is uninstalled, we remove the old version from status.
//   3. When v3 is installed successfully, we add the new version (v2) to status.
func (r *csvStatusReporter) GetNewStatus(existing *configv1.ClusterOperatorStatus, context *csvEventContext) (status *configv1.ClusterOperatorStatus) {
	builder := &Builder{
		clock:  r.clock,
		status: existing,
	}

	defer func() {
		status = builder.GetStatus()
	}()

	// We don't monitor whether the CSV backed operator is in degraded status.
	builder.WithDegraded(configv1.ConditionFalse)

	// A CSV has been deleted.
	if context.CurrentDeleted {
		csv := context.Current
		gvk := csv.GetObjectKind().GroupVersionKind()

		builder.WithoutVersion(csv.GetName(), csv.Spec.Version.String()).
			WithoutRelatedObject(gvk.Group, gvk.Kind, csv.GetNamespace(), csv.GetName())

		if context.WorkingToward == nil {
			builder.WithProgressing(configv1.ConditionFalse, fmt.Sprintf("Uninstalled version %s", csv.Spec.Version)).
				WithAvailable(configv1.ConditionFalse, "")

			return
		}
	}

	// It's either a fresh install or an upgrade.
	csv := context.GetActiveCSV()
	name := csv.GetName()
	version := csv.Spec.Version
	phase := csv.Status.Phase

	gvk := csv.GetObjectKind().GroupVersionKind()
	builder.WithRelatedObject("", "namespaces", "", csv.GetNamespace()).
		WithRelatedObject(gvk.Group, gvk.Kind, csv.GetNamespace(), csv.GetName())

	switch phase {
	case v1alpha1.CSVPhaseSucceeded:
		builder.WithAvailable(configv1.ConditionTrue, "")
	default:
		builder.WithAvailable(configv1.ConditionFalse, "")
	}

	switch phase {
	case v1alpha1.CSVPhaseSucceeded:
		builder.WithProgressing(configv1.ConditionFalse, fmt.Sprintf("Deployed version %s", version))
	case v1alpha1.CSVPhaseFailed:
		builder.WithProgressing(configv1.ConditionFalse, fmt.Sprintf("Failed to deploy %s", version))
	default:
		builder.WithProgressing(configv1.ConditionTrue, fmt.Sprintf("Working toward %s", version))
	}

	if phase == v1alpha1.CSVPhaseSucceeded {
		builder.WithVersion(versionName, r.releaseVersion)
		builder.WithVersion(name, version.String())
	}

	return
}
