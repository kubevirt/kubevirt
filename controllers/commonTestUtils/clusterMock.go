package commonTestUtils

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type clusterMock struct {
	// config is the rest.config used to talk to the apiserver.  Required.
	config *rest.Config

	// scheme is the scheme injected into Controllers, EventHandlers, Sources and Predicates.  Defaults
	// to scheme.scheme.
	scheme *runtime.Scheme

	cache cache.Cache

	// client is the client injected into Controllers (and EventHandlers, Sources and Predicates).
	client client.Client

	// apiReader is the reader that will make requests to the api server and not the cache.
	apiReader client.Reader

	// fieldIndexes knows how to add field indexes over the Cache used by this controller,
	// which can later be consumed via field selectors from the injected client.
	fieldIndexes client.FieldIndexer

	// mapper is used to map resources to kind, and map kind and version.
	mapper meta.RESTMapper

	// Logger is the logger that should be used by this manager.
	// If none is set, it defaults to log.Log global logger.
	logger logr.Logger
}

func (cm *clusterMock) SetFields(i interface{}) error {
	return nil
}

func (cm *clusterMock) GetConfig() *rest.Config {
	return cm.config
}

func (cm *clusterMock) GetClient() client.Client {
	return cm.client
}

func (cm *clusterMock) GetScheme() *runtime.Scheme {
	return cm.scheme
}

func (cm *clusterMock) GetFieldIndexer() client.FieldIndexer {
	return cm.fieldIndexes
}

func (cm *clusterMock) GetCache() cache.Cache {
	return cm.cache
}

func (cm *clusterMock) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}

func (cm *clusterMock) GetRESTMapper() meta.RESTMapper {
	return cm.mapper
}

func (cm *clusterMock) GetAPIReader() client.Reader {
	return cm.apiReader
}

func (cm *clusterMock) GetLogger() logr.Logger {
	return cm.logger
}

func (cm *clusterMock) Start(ctx context.Context) error {
	return nil
}

// NewClusterMock returns a new mocked Cluster for creating Controllers.
func NewClusterMock(config *rest.Config, client client.Client, logger logr.Logger) (cluster.Cluster, error) {

	options := cluster.Options{}

	return &clusterMock{
		config:       config,
		scheme:       options.Scheme,
		cache:        nil,
		fieldIndexes: nil,
		client:       client,
		apiReader:    client,
		mapper:       nil,
		logger:       logger,
	}, nil
}
