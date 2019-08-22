package operatorstatus

import (
	"fmt"
	"os"
	"strconv"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/csv"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	// SelectorKey is the key of the label we use to identify the
	// corresponding ClusterServiceVersion object related to the cluster operator.
	// If we want to update a cluster operator named "package-server" then the
	// corresponding ClusterServiceVersion must have the following label
	//
	// "olm.clusteroperator.name": "package-server"
	//
	SelectorKey = "olm.clusteroperator.name"
)

// NewCSVWatchNotificationHandler returns a new instance of csv.WatchNotification
// This can be used to get notification of every CSV reconciliation request.
func NewCSVWatchNotificationHandler(log *logrus.Logger, csvSet csv.SetGenerator, finder csv.ReplaceFinder, sender Sender) *handler {
	logger := log.WithField("monitor", "clusteroperator")
	releaseVersion := os.Getenv("RELEASE_VERSION")

	return &handler{
		csvSet:   csvSet,
		finder:   finder,
		sender:   sender,
		reporter: newCSVStatusReporter(releaseVersion),
		logger:   logger,
	}
}

// csvEventContext contains all necessary information related to a notification.
type csvEventContext struct {
	// Name of the clusteroperator resource associated with this CSV.
	Name string

	// Current is the CSV for which we have received notification.
	// If there is an upgrade going on, Current is set to the latest version of
	// the CSV that is replacing the older version.
	// For a chain like this, (v1) -> v2 -> v3 -> (v4)
	// Current will be set to the CSV linked to v4.
	WorkingToward *v1alpha1.ClusterServiceVersion

	// Current is the CSV for which we have received notification.
	Current *v1alpha1.ClusterServiceVersion

	// CurrentDeleted indicates that the Current CSV has been deleted
	CurrentDeleted bool
}

func (c *csvEventContext) GetActiveCSV() *v1alpha1.ClusterServiceVersion {
	if c.WorkingToward != nil {
		return c.WorkingToward
	}

	return c.Current
}

func (c *csvEventContext) String() string {
	replaces := "<nil>"
	if c.WorkingToward != nil {
		replaces = c.WorkingToward.GetName()
	}

	return fmt.Sprintf("name=%s csv=%s deleted=%s replaces=%s", c.Name, c.Current.GetName(), strconv.FormatBool(c.CurrentDeleted), replaces)
}

type handler struct {
	csvSet   csv.SetGenerator
	finder   csv.ReplaceFinder
	sender   Sender
	reporter *csvStatusReporter
	logger   *logrus.Entry
}

// OnAddOrUpdate is invoked when a CSV has been added or edited. We tap into
// this notification and do the following:
//
// a. Make sure this is the CSV related to the cluster operator resource we are
//    tracking. Otherwise, do nothing.
// b. If this is the right CSV then send it to the monitor.
func (h *handler) OnAddOrUpdate(in *v1alpha1.ClusterServiceVersion) {
	h.onNotification(in, false)
}

// OnDelete is invoked when a CSV has been deleted. We tap into
// this notification and do the following:
//
// a. Make sure this is the CSV related to the cluster operator resource we are
//    tracking. Otherwise, do nothing.
// b. If this is the right CSV then send it to the monitor.
func (h *handler) OnDelete(in *v1alpha1.ClusterServiceVersion) {
	h.onNotification(in, true)
}

func (h *handler) onNotification(current *v1alpha1.ClusterServiceVersion, deleted bool) {
	name, matched := h.isMatchingCSV(current)
	if !matched {
		return
	}

	workingToward := h.getLatestInReplacementChain(current)
	context := &csvEventContext{
		Name:           name,
		Current:        current,
		CurrentDeleted: deleted,
		WorkingToward:  workingToward,
	}

	if err := ownerutil.InferGroupVersionKind(current); err != nil {
		h.logger.Errorf("could not set GroupVersionKind - csv=%s", current.GetName())
	}

	if workingToward != nil {
		if err := ownerutil.InferGroupVersionKind(workingToward); err != nil {
			h.logger.Errorf("could not set GroupVersionKind - csv=%s", workingToward.GetName())
		}
	}

	h.logger.Debugf("found a matching CSV %s, sending notification", context)

	notification := h.reporter.NewNotification(context)
	h.sender.Send(notification)
}

func (h *handler) getLatestInReplacementChain(in *v1alpha1.ClusterServiceVersion) (final *v1alpha1.ClusterServiceVersion) {
	requirement, _ := labels.NewRequirement(SelectorKey, selection.Exists, []string{})
	selector := labels.NewSelector().Add(*requirement)
	related := h.csvSet.WithNamespaceAndLabels(in.GetNamespace(), v1alpha1.CSVPhaseAny, selector)

	return h.finder.GetFinalCSVInReplacing(in, related)
}

func (h *handler) isMatchingCSV(in *v1alpha1.ClusterServiceVersion) (name string, matched bool) {
	// If it is a "copy" CSV we ignore it.
	if in.IsCopied() {
		return
	}

	// Does it have the right label?
	labels := in.GetLabels()
	if labels == nil {
		return
	}

	name, _ = labels[SelectorKey]
	if name == "" {
		return
	}

	matched = true
	return
}
