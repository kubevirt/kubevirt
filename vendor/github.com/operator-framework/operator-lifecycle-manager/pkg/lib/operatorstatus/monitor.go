package operatorstatus

import (
	"fmt"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/discovery"
)

const (
	// Wait time before we probe next while checking whether cluster
	// operator API is available.
	defaultProbeInterval = 1 * time.Minute

	// Default size of the notification channel.
	defaultNotificationChannelSize = 64
)

// NewMonitor returns a new instance of Monitor that can be used to continuously
// update a clusteroperator resource and an instance of Sender that can be used
// to send update notifications to it.
//
// The name of the clusteroperator resource to monitor is specified in name.
func NewMonitor(name string, log *logrus.Logger, discovery discovery.DiscoveryInterface, configClient configv1client.ConfigV1Interface) (Monitor, Sender) {
	logger := log.WithField("monitor", "clusteroperator")
	names := split(name)

	logger.Infof("monitoring the following components %s", names)

	monitor := &monitor{
		logger:         logger,
		writer:         NewWriter(discovery, configClient),
		notificationCh: make(chan NotificationFunc, defaultNotificationChannelSize),
		names:          names,
	}

	return monitor, monitor
}

// Monitor is an interface that wraps the Run method.
//
// Run runs forever, it reads from an underlying notification channel and
// updates an clusteroperator resource accordingly.
// If the specified stop channel is closed the loop must terminate gracefully.
type Monitor interface {
	Run(stopCh <-chan struct{})
}

// MutatorFunc accepts an existing status object and appropriately mutates it
// to reflect the observed states.
type MutatorFunc func(existing *configv1.ClusterOperatorStatus) (new *configv1.ClusterOperatorStatus)

// Mutate is a wrapper for MutatorFunc
func (m MutatorFunc) Mutate(existing *configv1.ClusterOperatorStatus) (new *configv1.ClusterOperatorStatus) {
	return m(existing)
}

// NotificationFunc wraps a notification event. it returns the name of the
// cluster operator object associated and a mutator function that will set the
// new status for the cluster operator object.
type NotificationFunc func() (name string, mutator MutatorFunc)

// Get is a wrapper for NotificationFunc.
func (n NotificationFunc) Get() (name string, mutator MutatorFunc) {
	return n()
}

// Sender is an interface that wraps the Send method.
//
// Send can be used to send notification(s) to the underlying monitor. Send is a
// non-blocking operation.
// If the underlying monitor is not ready to receive the notification will be lost.
// If the notification context specified is nil then it is ignored.
type Sender interface {
	Send(NotificationFunc)
}

type monitor struct {
	notificationCh chan NotificationFunc
	writer         *Writer
	logger         *logrus.Entry
	names          []string
}

func (m *monitor) Send(notification NotificationFunc) {
	if notification == nil {
		return
	}

	select {
	case m.notificationCh <- notification:
	default:
		m.logger.Warn("monitor not ready to receive")
	}
}

func (m *monitor) Run(stopCh <-chan struct{}) {
	m.logger.Info("starting clusteroperator monitor loop")
	defer func() {
		m.logger.Info("exiting from clusteroperator monitor loop")
	}()

	// First, we need to ensure that cluster operator API is available.
	// We will keep probing until it is available.
	for {
		exists, err := m.writer.IsAPIAvailable()
		if err != nil {
			m.logger.Infof("ClusterOperator api not present, skipping update (%v)", err)
		}

		if exists {
			m.logger.Info("ClusterOperator api is present")
			break
		}

		// Wait before next probe, or quit if parent has asked to do so.
		select {
		case <-time.After(defaultProbeInterval):
		case <-stopCh:
			return
		}
	}

	// If we are here, cluster operator is available.
	// We are expecting CSV notification which may or may not arrive.
	// Given this, let's write an initial ClusterOperator object with our expectation.
	m.logger.Infof("initializing clusteroperator resource(s) for %s", m.names)

	for _, name := range m.names {
		if err := m.init(name); err != nil {
			m.logger.Errorf("initialization error - %v", err)
			break
		}
	}

	for {
		select {
		case notification := <-m.notificationCh:
			if notification != nil {
				name, mutator := notification.Get()
				if err := m.update(name, mutator); err != nil {
					m.logger.Errorf("status update error - %v", err)
				}
			}

		case <-stopCh:
			return
		}
	}
}

func (m *monitor) update(name string, mutator MutatorFunc) error {
	if mutator == nil {
		return fmt.Errorf("no status mutator specified name=%s", name)
	}

	existing, err := m.writer.EnsureExists(name)
	if err != nil {
		return fmt.Errorf("failed to ensure initial clusteroperator name=%s - %v", name, err)
	}

	existingStatus := existing.Status.DeepCopy()
	newStatus := mutator.Mutate(existingStatus)
	if err := m.writer.UpdateStatus(existing, newStatus); err != nil {
		return fmt.Errorf("failed to update clusteroperator status name=%s - %v", name, err)
	}

	return nil
}

func (m *monitor) init(name string) error {
	existing, err := m.writer.EnsureExists(name)
	if err != nil {
		return fmt.Errorf("failed to ensure name=%s - %v", name, err)
	}

	if len(existing.Status.Conditions) > 0 {
		return nil
	}

	// No condition(s) in existing status, let's add conditions that reflect our expectation.
	newStatus := Waiting(&clock.RealClock{}, name)
	if err := m.writer.UpdateStatus(existing, newStatus); err != nil {
		return fmt.Errorf("failed to update status name=%s - %v", name, err)
	}

	return nil
}

// Waiting returns an initialized ClusterOperatorStatus object that
// is suited for creation if the given object does not exist already. The
// initialized object has the expected status for cluster operator resource
// before we have seen any corresponding CSV.
func Waiting(clock clock.Clock, name string) *configv1.ClusterOperatorStatus {
	builder := &Builder{
		clock: clock,
	}

	status := builder.WithDegraded(configv1.ConditionFalse).
		WithAvailable(configv1.ConditionFalse, "").
		WithProgressing(configv1.ConditionTrue, fmt.Sprintf("waiting for events - source=%s", name)).
		GetStatus()

	return status
}

func split(n string) []string {
	names := make([]string, 0)

	values := strings.Split(n, ",")
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			names = append(names, v)
		}
	}

	return names
}
