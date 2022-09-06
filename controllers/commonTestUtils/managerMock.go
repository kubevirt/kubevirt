package commonTestUtils

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type ManagerMock struct {
	runnables []manager.Runnable

	// cluster holds a variety of methods to interact with a cluster. Required.
	cluster cluster.Cluster

	// controllerOptions are the global controller options.
	controllerOptions v1alpha1.ControllerConfigurationSpec

	// Logger is the logger that should be used by this manager.
	// If none is set, it defaults to log.Log global logger.
	logger logr.Logger

	// elected is closed when this manager becomes the leader of a group of
	// managers, either because it won a leader election or because no leader
	// election was configured.
	elected chan struct{}

	webhookServer *webhook.Server
}

// Add sets dependencies on i, and adds it to the list of Runnables to start.
func (mm *ManagerMock) Add(r manager.Runnable) error {
	mm.runnables = append(mm.runnables, r)
	return nil
}

// Deprecated: use the equivalent Options field to set a field. This method will be removed in v0.10.
func (mm ManagerMock) SetFields(i interface{}) error {
	return nil
}

// AddMetricsExtraHandler adds extra handler served on path to the http server that serves metrics.
func (mm ManagerMock) AddMetricsExtraHandler(path string, handler http.Handler) error {
	return nil
}

// AddHealthzCheck allows you to add Healthz checker.
func (mm ManagerMock) AddHealthzCheck(name string, check healthz.Checker) error {
	return nil
}

// AddReadyzCheck allows you to add Readyz checker.
func (mm ManagerMock) AddReadyzCheck(name string, check healthz.Checker) error {
	return nil
}

func (mm ManagerMock) GetConfig() *rest.Config {
	return mm.cluster.GetConfig()
}

func (mm ManagerMock) GetClient() client.Client {
	return mm.cluster.GetClient()
}

func (mm ManagerMock) GetScheme() *runtime.Scheme {
	return mm.cluster.GetScheme()
}

func (mm ManagerMock) GetFieldIndexer() client.FieldIndexer {
	return mm.cluster.GetFieldIndexer()
}

func (mm ManagerMock) GetCache() cache.Cache {
	return mm.cluster.GetCache()
}

func (mm ManagerMock) GetEventRecorderFor(name string) record.EventRecorder {
	return mm.cluster.GetEventRecorderFor(name)
}

func (mm ManagerMock) GetRESTMapper() meta.RESTMapper {
	return mm.cluster.GetRESTMapper()
}

func (mm ManagerMock) GetAPIReader() client.Reader {
	return mm.cluster.GetAPIReader()
}

func (mm ManagerMock) GetWebhookServer() *webhook.Server {
	return mm.webhookServer
}

func (mm ManagerMock) GetLogger() logr.Logger {
	return mm.logger
}

func (mm ManagerMock) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return mm.controllerOptions
}

func (mm ManagerMock) Start(ctx context.Context) (err error) {
	return nil
}

func (mm ManagerMock) Elected() <-chan struct{} {
	return mm.elected
}

func (mm ManagerMock) GetRunnables() []manager.Runnable {
	return mm.runnables
}

// NewManagerMock returns a new mocked Manager for unit test which involves Controller Managers
func NewManagerMock(config *rest.Config, options manager.Options, client client.Client, logger logr.Logger) (manager.Manager, error) {

	cluster, err := NewClusterMock(config, client, logger)
	if err != nil {
		return nil, err
	}

	runnables := make([]manager.Runnable, 0)

	return &ManagerMock{
		cluster:           cluster,
		runnables:         runnables,
		controllerOptions: options.Controller,
		logger:            logger,
		elected:           make(chan struct{}),
		webhookServer:     options.WebhookServer,
	}, nil
}
