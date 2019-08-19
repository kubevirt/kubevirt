package queueinformer

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/metrics"
)

// QueueIndexer ties an indexer to a queue in order to process events
// the syncHandler is called for all objects on the queue
// Unlike QueueInformer, nothing is automatically adding objects to the queue
type QueueIndexer struct {
	queue       workqueue.RateLimitingInterface
	indexers    map[string]cache.Indexer
	syncHandler SyncHandler
	name        string
	metrics.MetricsProvider
	log *logrus.Logger
}

// Enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (q *QueueIndexer) Enqueue(obj interface{}) {
	if obj == nil {
		return
	}

	key, ok := obj.(string)
	if !ok {
		key, ok = q.keyFunc(obj)
		if !ok {
			return
		}
	}

	q.queue.Add(key)
}

func (q *QueueIndexer) Add(key string) {
	q.queue.Add(key)
}

// keyFunc turns an object into a key for the queue. In the future will use a (name, namespace) struct as key
func (q *QueueIndexer) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		q.log.Infof("creating key failed: %s", err)
		return k, false
	}

	return k, true
}

func NewQueueIndexer(queue workqueue.RateLimitingInterface, indexers map[string]cache.Indexer, handler SyncHandler, name string, logger *logrus.Logger, provider metrics.MetricsProvider) *QueueIndexer {
	return &QueueIndexer{
		queue:           queue,
		indexers:        indexers,
		syncHandler:     handler,
		name:            name,
		log:             logger,
		MetricsProvider: provider,
	}
}
