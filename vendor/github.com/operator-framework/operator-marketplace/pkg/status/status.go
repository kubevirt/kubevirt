package status

import (
	"fmt"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	operatorhelpers "github.com/openshift/library-go/pkg/operator/v1helpers"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// clusterOperatorName is the name of the cluster operator
	clusterOperatorName = "marketplace"

	// minSyncsBeforeReporting is the minimum number of syncs we wish to see
	// before reporting that the operator is available
	minSyncsBeforeReporting = 4

	// successRatio is the ratio of successful syncs / total syncs that we
	// want to see in order to report that the marketplace operator is healthy.
	// This value is low right now because the failed syncs come from invalid CRs.
	// As the status reporting evolves we can tweek this ratio to be a better
	// representation of the operator's health.
	successRatio = 0.3

	// syncsBeforeTruncate is used to prevent the totalSyncs and
	// failedSyncs values from overflowing in a long running operator.
	// Once totalSyncs reaches the maxSyncsBeforeTruncate value, totalSyncs
	// and failedSyncs will be recalculated with the following
	// equation: updatedValue = currentValue % syncTruncateValue.
	syncsBeforeTruncate = 10000
	syncTruncateValue   = 100

	// coStatusReportInterval is the interval at which the cluster operator status is updated
	coStatusReportInterval = 20 * time.Second
)

// status will be a singleton
var instance *status
var once sync.Once

type status struct {
	configClient    *configclient.ConfigV1Client
	coAPINotPresent bool
	namespace       string
	clusterOperator *configv1.ClusterOperator
	version         string
	syncRatio       SyncRatio
	// syncCh is used to report sync events
	syncCh chan error
	// stopCh is used to signal that threads should stop reporting ClusterOperator status
	stopCh <-chan struct{}
	// monitorDoneCh is used to signal that threads are done reporting ClusterOperator status
	monitorDoneCh chan struct{}
}

// SendSyncMessage is used to send messages to the syncCh. If the channel is
// busy, the sync will be dropped to prevent the controller from stalling.
func SendSyncMessage(err error) {
	// If the coAPI is not available do not attempt to send messages to the
	// sync channel
	if instance == nil || instance.coAPINotPresent {
		return
	}
	// A missing sync status is better than stalling the controller
	select {
	case instance.syncCh <- err:
		log.Debugf("[status] Sent message to the sync channel")
		break
	default:
		log.Debugf("[status] Sync channel is busy, not reporting sync")
	}
}

// StartReporting ensures that the cluster supports reporting ClusterOperator status
// and returns a channel that reports if it is actively reporting.
func StartReporting(cfg *rest.Config, mgr manager.Manager, namespace string, version string, stopCh <-chan struct{}) <-chan struct{} {
	// ensure instance is only created once.
	once.Do(func() {
		instance = new(cfg, mgr, namespace, version, stopCh)
		// exit if cluster operator api is not present
		if instance.coAPINotPresent {
			return
		}

		// start consuming messages on the sync channel
		go instance.syncChannelReceiver()

		// start reporting ClusterOperator status
		go instance.monitorClusterStatus()
	})

	return instance.monitorDoneCh
}

// New returns an initialized status
func new(cfg *rest.Config, mgr manager.Manager, namespace string, version string, stopCh <-chan struct{}) *status {
	// Check if the ClusterOperator API is present on the cluster. If the API
	// is not present on the cluster set the status.coAPINotPresent to true so
	// status is not reported
	k8sInterface, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("[status] " + err.Error())
	}
	err = discovery.ServerSupportsVersion(k8sInterface.Discovery(), schema.GroupVersion{
		Group:   "config.openshift.io",
		Version: "v1",
	})

	monitorDoneCh := make(chan struct{})
	coAPINotPresent := false
	if err != nil {
		log.Warningf("[status] ClusterOperator API not present: %v", err)
		coAPINotPresent = true
		// If the co is not present, close the monitorDoneCh channel to prevent
		// recievers from stalling
		close(monitorDoneCh)
	}

	// Client for handling reporting of operator status
	configClient, err := configclient.NewForConfig(cfg)
	if err != nil {
		log.Fatal("[status] " + err.Error())
	}

	syncRatio, err := NewSyncRatio(successRatio, syncsBeforeTruncate, syncTruncateValue)
	if err != nil {
		log.Fatal("[status] " + err.Error())
	}

	return &status{
		configClient:    configClient,
		coAPINotPresent: coAPINotPresent,
		namespace:       namespace,
		version:         version,
		syncRatio:       syncRatio,
		// Add a buffer to prevent dropping syncs
		syncCh:        make(chan error, 25),
		stopCh:        stopCh,
		monitorDoneCh: monitorDoneCh,
	}
}

// setFailing reports that operator has failed along with the error message
func (s *status) setFailing(message string) error {
	return s.setStatus(configv1.OperatorFailing, message)
}

// setAvailable reports that the operator is available to process events
func (s *status) setAvailable(message string) error {
	return s.setStatus(configv1.OperatorAvailable, message)
}

// setProgressing reports that the operator is being deployed
func (s *status) setProgressing(message string) error {
	return s.setStatus(configv1.OperatorProgressing, message)
}

// ensureClusterOperator ensures that a ClusterOperator CR is present on the
// cluster
func (s *status) ensureClusterOperator() error {
	var err error
	s.clusterOperator, err = s.configClient.ClusterOperators().Get(clusterOperatorName, v1.GetOptions{})

	if err == nil {
		log.Debug("[status] Found existing ClusterOperator")
		return nil
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("Error %v getting ClusterOperator", err)
	}

	s.clusterOperator, err = s.configClient.ClusterOperators().Create(&configv1.ClusterOperator{
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterOperatorName,
			Namespace: s.namespace,
		},
	})
	if err != nil {
		return fmt.Errorf("Error %v creating ClusterOperator", err)
	}
	log.Info("[status] Created ClusterOperator")
	return nil
}

// setStatus handles setting all the required fields for the given
// ClusterStatusConditionType
func (s *status) setStatus(condition configv1.ClusterStatusConditionType, message string) error {
	if s.coAPINotPresent {
		return nil
	}
	err := s.ensureClusterOperator()
	if err != nil {
		return err
	}
	previousStatus := s.clusterOperator.Status.DeepCopy()
	updatedCondition := s.setStatusCondition(condition, message)
	err = s.updateStatus(previousStatus, updatedCondition, message)
	if err != nil {
		return err
	}
	return nil
}

// setOperandVersion sets the operator version in the ClusterOperator Status
// Per instructions from the CVO team, setOperandVersion should only be called
// when the operator becomes available
func (s *status) setOperandVersion() {
	// Report the operator version
	operatorVersion := configv1.OperandVersion{
		Name:    "operator",
		Version: s.version,
	}
	operatorhelpers.SetOperandVersion(&s.clusterOperator.Status.Versions, operatorVersion)
}

// setStatusCondition sets the operator StatusCondition in the ClusterOperator Status
func (s *status) setStatusCondition(condition configv1.ClusterStatusConditionType, message string) string {
	updatedCondition := string(condition)

	availableStatus := configv1.ConditionFalse
	failingStatus := configv1.ConditionFalse
	progressingStatus := configv1.ConditionFalse
	availableMessage := ""
	failingMessage := ""
	progressingMessage := ""

	switch condition {
	case configv1.OperatorAvailable:
		availableStatus = configv1.ConditionTrue
		availableMessage = message
		// Only update the version when the operator becomes available
		s.setOperandVersion()

	case configv1.OperatorFailing:
		failingStatus = configv1.ConditionTrue
		failingMessage = message

	case configv1.OperatorProgressing:
		progressingStatus = configv1.ConditionTrue
		progressingMessage = message
	}

	time := v1.Now()
	// https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#conditions
	// implies that all three StatusConditionTypes needs to be set with only
	// the relevant ClusterStatusConditionType's Status set to ConditionTrue
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorProgressing,
		Status:             progressingStatus,
		Message:            progressingMessage,
		LastTransitionTime: time,
	})
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorAvailable,
		Status:             availableStatus,
		Message:            availableMessage,
		LastTransitionTime: time,
	})
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorFailing,
		Status:             failingStatus,
		Message:            failingMessage,
		LastTransitionTime: time,
	})
	return updatedCondition
}

// updateStatus makes the API call to update the ClusterOperator if the status has changed.
func (s *status) updateStatus(previousStatus *configv1.ClusterOperatorStatus, updatedCondition string, message string) error {
	var err error
	if compareClusterOperatorStatusConditionArrays(previousStatus.Conditions, s.clusterOperator.Status.Conditions) {
		log.Debugf("[status] Previous and current ClusterOperator Status are the same, the ClusterOperator Status will not be updated.")
	} else {
		log.Debugf("[status] Previous and current ClusterOperator Status are different, attempting to update the ClusterOperator Status.")

		// Check if the ClusterOperator version has changed and log the attempt to upgrade if it has
		previousVersion := operatorhelpers.FindOperandVersion(previousStatus.Versions, "operator")
		currentVersion := operatorhelpers.FindOperandVersion(s.clusterOperator.Status.Versions, "operator")
		if currentVersion != nil {
			if previousVersion == nil {
				log.Infof("[status] Attempting to set ClusterOperator to version %s", currentVersion.Version)
			} else if previousVersion.Version != currentVersion.Version {
				log.Infof("[status] Attempting to upgrade ClusterOperator version from %s to %s", previousVersion.Version, currentVersion.Version)
			}
		}

		_, err := s.configClient.ClusterOperators().UpdateStatus(s.clusterOperator)
		if err != nil {
			return fmt.Errorf("Error %v updating ClusterOperator", err)
		}
		// Log status change
		log.Infof(fmt.Sprintf("[status] Set ClusterOperator condition: %s message: %s", updatedCondition, message))
	}
	return err
}

// syncChannelReceiver will listen on the sync channel and update the status
// syncsRatio filed until the stopCh is closed.
func (s *status) syncChannelReceiver() {
	log.Info("[status] Starting sync consumer")
	for {
		select {
		case <-s.stopCh:
			return
		case err := <-s.syncCh:
			if err == nil {
				s.syncRatio.ReportSyncEvent()
			} else {
				s.syncRatio.ReportFailedSync()
			}
			failedSyncs, syncs := s.syncRatio.GetSyncs()
			log.Debugf("[status] Faild Syncs / Total Syncs : %d/%d", failedSyncs, syncs)
		}
	}
}

// monitorClusterStatus updates the cluster operator's status based on
// the number of successful syncs / total syncs
func (s *status) monitorClusterStatus() {
	// Signal to the main channel that we have stopped reporting status.
	defer func() {
		close(s.monitorDoneCh)
	}()
	for {
		select {
		case <-s.stopCh:
			// If the stopCh is closed, the operator will exit and CO should
			// be set to failing.
			s.setFailing("Operator exited")
			return
		// Attempt to update the cluster operator status whenever the seconds
		// number of seconds defined by coStatusReportInterval passes
		case <-time.After(coStatusReportInterval):
			var statusErr error
			// create the cluster operator in the porgressing state if it does not exist
			// or if it is the first report
			if s.clusterOperator == nil {
				statusErr = s.setProgressing(fmt.Sprintf("Progressing towards %s", s.version))
			} else {
				// wait until the operator has reconciled at least one sync
				_, syncEvents := s.syncRatio.GetSyncs()
				if syncEvents >= minSyncsBeforeReporting {
					// update the status with the appropriate state
					isSucceeding, ratio := s.syncRatio.IsSucceeding()
					if ratio != nil {
						if isSucceeding {
							statusErr = s.setAvailable(fmt.Sprintf("%s is available", s.version))
						} else {
							statusErr = s.setFailing(fmt.Sprintf("Current sync ratio (%g) does not meet the expected success ratio (%g)", *ratio, successRatio))
						}
					}
				} else {
					log.Debugf("[status] Waiting to observe %d additional sync(s)", minSyncsBeforeReporting-syncEvents)
				}
			}
			if statusErr != nil {
				log.Error("[status] " + statusErr.Error())
			}
		}
	}
}
