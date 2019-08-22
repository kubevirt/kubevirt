package subscription

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/reconciler"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubestate"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorlister"
)

type syncerConfig struct {
	logger                    *logrus.Logger
	clock                     utilclock.Clock
	client                    versioned.Interface
	lister                    operatorlister.OperatorLister
	subscriptionInformer      cache.SharedIndexInformer
	catalogInformer           cache.SharedIndexInformer
	installPlanInformer       cache.SharedIndexInformer
	subscriptionQueue         workqueue.RateLimitingInterface
	reconcilers               kubestate.ReconcilerChain
	registryReconcilerFactory reconciler.RegistryReconcilerFactory
	globalCatalogNamespace    string
}

// SyncerOption is a configuration option for a subscription syncer.
type SyncerOption func(*syncerConfig)

func defaultSyncerConfig() *syncerConfig {
	return &syncerConfig{
		logger:      logrus.New(),
		clock:       utilclock.RealClock{},
		reconcilers: kubestate.ReconcilerChain{},
	}
}

func (s *syncerConfig) apply(options []SyncerOption) {
	for _, option := range options {
		option(s)
	}
}

// WithLogger sets a syncer's logger.
func WithLogger(logger *logrus.Logger) SyncerOption {
	return func(config *syncerConfig) {
		config.logger = logger
	}
}

// WithClock sets a syncer's clock.
func WithClock(clock utilclock.Clock) SyncerOption {
	return func(config *syncerConfig) {
		config.clock = clock
	}
}

// WithClient sets a syncer's OLM client.
func WithClient(client versioned.Interface) SyncerOption {
	return func(config *syncerConfig) {
		config.client = client
	}
}

// WithSubscriptionInformer sets the informer a syncer will extract its subscription indexer from.
func WithSubscriptionInformer(subscriptionInformer cache.SharedIndexInformer) SyncerOption {
	return func(config *syncerConfig) {
		config.subscriptionInformer = subscriptionInformer
	}
}

// WithCatalogInformer sets a CatalogSource informer to act as an event source for dependent Subscriptions.
func WithCatalogInformer(catalogInformer cache.SharedIndexInformer) SyncerOption {
	return func(config *syncerConfig) {
		config.catalogInformer = catalogInformer
	}
}

// WithInstallPlanInformer sets an InstallPlan informer to act as an event source for dependent Subscriptions.
func WithInstallPlanInformer(installPlanInformer cache.SharedIndexInformer) SyncerOption {
	return func(config *syncerConfig) {
		config.installPlanInformer = installPlanInformer
	}
}

// WithOperatorLister sets a syncer's operator lister.
func WithOperatorLister(lister operatorlister.OperatorLister) SyncerOption {
	return func(config *syncerConfig) {
		config.lister = lister
	}
}

// WithSubscriptionQueue sets a syncer's subscription queue.
func WithSubscriptionQueue(subscriptionQueue workqueue.RateLimitingInterface) SyncerOption {
	return func(config *syncerConfig) {
		config.subscriptionQueue = subscriptionQueue
	}
}

// WithAppendedReconcilers adds the given reconcilers to the end of a syncer's reconciler chain, to be
// invoked after its default reconcilers have been called.
func WithAppendedReconcilers(reconcilers ...kubestate.Reconciler) SyncerOption {
	return func(config *syncerConfig) {
		// Add non-nil reconcilers to the chain
		for _, rec := range reconcilers {
			if rec != nil {
				config.reconcilers = append(config.reconcilers, rec)
			}
		}
	}
}

// WithRegistryReconcilerFactory sets a syncer's registry reconciler factory.
func WithRegistryReconcilerFactory(r reconciler.RegistryReconcilerFactory) SyncerOption {
	return func(config *syncerConfig) {
		config.registryReconcilerFactory = r
	}
}

// WithGlobalCatalogNamespace sets a syncer's global catalog namespace.
func WithGlobalCatalogNamespace(namespace string) SyncerOption {
	return func(config *syncerConfig) {
		config.globalCatalogNamespace = namespace
	}
}

func newInvalidConfigError(msg string) error {
	return errors.Errorf("invalid subscription syncer config: %s", msg)
}

func (s *syncerConfig) validate() (err error) {
	switch {
	case s.logger == nil:
		err = newInvalidConfigError("nil logger")
	case s.clock == nil:
		err = newInvalidConfigError("nil clock")
	case s.client == nil:
		err = newInvalidConfigError("nil client")
	case s.lister == nil:
		err = newInvalidConfigError("nil lister")
	case s.subscriptionInformer == nil:
		err = newInvalidConfigError("nil subscription informer")
	case s.catalogInformer == nil:
		err = newInvalidConfigError("nil catalog informer")
	case s.installPlanInformer == nil:
		err = newInvalidConfigError("nil installplan informer")
	case s.subscriptionQueue == nil:
		err = newInvalidConfigError("nil subscription queue")
	case len(s.reconcilers) == 0:
		err = newInvalidConfigError("no reconcilers")
	case s.registryReconcilerFactory == nil:
		err = newInvalidConfigError("nil reconciler factory")
	case s.globalCatalogNamespace == metav1.NamespaceAll:
		err = newInvalidConfigError("global catalog namespace cannot be namespace all")
	}

	return
}
