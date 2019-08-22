package olm

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilclock "k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/rest"

	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/internalversion"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/resolver"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/labeler"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

type OperatorOption func(*operatorConfig)

type operatorConfig struct {
	resyncPeriod      time.Duration
	operatorNamespace string
	watchedNamespaces []string
	clock             utilclock.Clock
	logger            *logrus.Logger
	operatorClient    operatorclient.ClientInterface
	externalClient    versioned.Interface
	internalClient    internalversion.Interface
	strategyResolver  install.StrategyResolverInterface
	apiReconciler     resolver.APIIntersectionReconciler
	apiLabeler        labeler.Labeler
	restConfig        *rest.Config
	configClient      configv1client.Interface
}

func (o *operatorConfig) apply(options []OperatorOption) {
	for _, option := range options {
		option(o)
	}
}

func newInvalidConfigError(name, msg string) error {
	return errors.Errorf("%s config invalid: %s", name, msg)
}

func (o *operatorConfig) validate() (err error) {
	// TODO: Add better config validation
	switch {
	case o.resyncPeriod < 0:
		err = newInvalidConfigError("resync period", "must be >= 0")
	case o.operatorNamespace == metav1.NamespaceAll:
		err = newInvalidConfigError("operator namespace", "must be a single namespace")
	case len(o.watchedNamespaces) == 0:
		err = newInvalidConfigError("watched namespaces", "must watch at least one namespace")
	case o.clock == nil:
		err = newInvalidConfigError("clock", "must not be nil")
	case o.logger == nil:
		err = newInvalidConfigError("logger", "must not be nil")
	case o.operatorClient == nil:
		err = newInvalidConfigError("operator client", "must not be nil")
	case o.externalClient == nil:
		err = newInvalidConfigError("external client", "must not be nil")
	// case o.internalClient == nil:
	// err = newInvalidConfigError("internal client", "must not be nil")
	case o.strategyResolver == nil:
		err = newInvalidConfigError("strategy resolver", "must not be nil")
	case o.apiReconciler == nil:
		err = newInvalidConfigError("api reconciler", "must not be nil")
	case o.apiLabeler == nil:
		err = newInvalidConfigError("api labeler", "must not be nil")
	case o.restConfig == nil:
		err = newInvalidConfigError("rest config", "must not be nil")
	}

	return
}

func defaultOperatorConfig() *operatorConfig {
	return &operatorConfig{
		resyncPeriod:      30 * time.Second,
		operatorNamespace: "default",
		watchedNamespaces: []string{metav1.NamespaceAll},
		clock:             utilclock.RealClock{},
		logger:            logrus.New(),
		strategyResolver:  &install.StrategyResolver{},
		apiReconciler:     resolver.APIIntersectionReconcileFunc(resolver.ReconcileAPIIntersection),
		apiLabeler:        labeler.Func(resolver.LabelSetsFor),
	}
}

func WithResyncPeriod(period time.Duration) OperatorOption {
	return func(config *operatorConfig) {
		config.resyncPeriod = period
	}
}

func WithOperatorNamespace(namespace string) OperatorOption {
	return func(config *operatorConfig) {
		config.operatorNamespace = namespace
	}
}

func WithWatchedNamespaces(namespaces ...string) OperatorOption {
	return func(config *operatorConfig) {
		config.watchedNamespaces = namespaces
	}
}

func WithLogger(logger *logrus.Logger) OperatorOption {
	return func(config *operatorConfig) {
		config.logger = logger
	}
}

func WithClock(clock utilclock.Clock) OperatorOption {
	return func(config *operatorConfig) {
		config.clock = clock
	}
}

func WithOperatorClient(operatorClient operatorclient.ClientInterface) OperatorOption {
	return func(config *operatorConfig) {
		config.operatorClient = operatorClient
	}
}

func WithExternalClient(externalClient versioned.Interface) OperatorOption {
	return func(config *operatorConfig) {
		config.externalClient = externalClient
	}
}

func WithInternalClient(internalClient internalversion.Interface) OperatorOption {
	return func(config *operatorConfig) {
		config.internalClient = internalClient
	}
}

func WithStrategyResolver(strategyResolver install.StrategyResolverInterface) OperatorOption {
	return func(config *operatorConfig) {
		config.strategyResolver = strategyResolver
	}
}

func WithAPIReconciler(apiReconciler resolver.APIIntersectionReconciler) OperatorOption {
	return func(config *operatorConfig) {
		config.apiReconciler = apiReconciler
	}
}

func WithAPILabeler(apiLabeler labeler.Labeler) OperatorOption {
	return func(config *operatorConfig) {
		config.apiLabeler = apiLabeler
	}
}

func WithRestConfig(restConfig *rest.Config) OperatorOption {
	return func(config *operatorConfig) {
		config.restConfig = restConfig
	}
}

func WithConfigClient(configClient configv1client.Interface) OperatorOption {
	return func(config *operatorConfig) {
		config.configClient = configClient
	}
}
