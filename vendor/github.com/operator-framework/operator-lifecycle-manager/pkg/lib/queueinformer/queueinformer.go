package queueinformer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubestate"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/metrics"
)

// KeyFunc returns a key for the given object and a bool which is true if the key was
// successfully generated and false otherwise.
type KeyFunc func(obj interface{}) (string, bool)

// QueueInformer ties an informer to a queue in order to process events from the informer
// the informer watches objects of interest and adds objects to the queue for processing
// the syncHandler is called for all objects on the queue
type QueueInformer struct {
	metrics.MetricsProvider

	logger   *logrus.Logger
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
	indexer  cache.Indexer
	keyFunc  KeyFunc
	syncer   kubestate.Syncer
}

// Sync invokes all registered sync handlers in the QueueInformer's chain
func (q *QueueInformer) Sync(ctx context.Context, event kubestate.ResourceEvent) error {
	return q.syncer.Sync(ctx, event)
}

// Enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (q *QueueInformer) Enqueue(event kubestate.ResourceEvent) {
	if event == nil {
		// Don't enqueue nil events
		return
	}

	resource := event.Resource()
	if event.Type() == kubestate.ResourceDeleted {
		// Get object from tombstone if possible
		if tombstone, ok := resource.(cache.DeletedFinalStateUnknown); ok {
			resource = tombstone
		}
	} else {
		// Extract key for add and update events
		if key, ok := q.key(resource); ok {
			resource = key
		}
	}

	// Create new resource event and add to queue
	e := kubestate.NewResourceEvent(event.Type(), resource)
	q.logger.WithField("event", e).Trace("enqueuing resource event")
	q.queue.Add(e)
}

// key turns an object into a key for the indexer.
func (q *QueueInformer) key(obj interface{}) (string, bool) {
	return q.keyFunc(obj)
}

// resourceHandlers provides the default implementation for responding to events
// these simply Log the event and add the object's key to the queue for later processing.
func (q *QueueInformer) resourceHandlers(ctx context.Context) *cache.ResourceEventHandlerFuncs {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			q.Enqueue(kubestate.NewResourceEvent(kubestate.ResourceUpdated, obj))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			q.Enqueue(kubestate.NewResourceEvent(kubestate.ResourceUpdated, newObj))
		},
		DeleteFunc: func(obj interface{}) {
			q.Enqueue(kubestate.NewResourceEvent(kubestate.ResourceDeleted, obj))
		},
	}
}

// metricHandlers provides the default implementation for handling metrics in response to events.
func (q *QueueInformer) metricHandlers() *cache.ResourceEventHandlerFuncs {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if err := q.HandleMetrics(); err != nil {
				q.logger.WithError(err).WithField("key", obj).Warn("error handling metrics on add event")
			}
		},
		DeleteFunc: func(obj interface{}) {
			if err := q.HandleMetrics(); err != nil {
				q.logger.WithError(err).WithField("key", obj).Warn("error handling metrics on delete event")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if err := q.HandleMetrics(); err != nil {
				q.logger.WithError(err).WithField("key", newObj).Warn("error handling metrics on update event")
			}
		},
	}
}

// NewQueueInformer returns a new QueueInformer configured with options.
func NewQueueInformer(ctx context.Context, options ...Option) (*QueueInformer, error) {
	// Get default config and apply given options
	config := defaultConfig()
	config.apply(options)
	config.complete()

	return newQueueInformerFromConfig(ctx, config)
}

func newQueueInformerFromConfig(ctx context.Context, config *queueInformerConfig) (*QueueInformer, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	// Extract config
	queueInformer := &QueueInformer{
		MetricsProvider: config.provider,
		logger:          config.logger,
		queue:           config.queue,
		indexer:         config.indexer,
		informer:        config.informer,
		keyFunc:         config.keyFunc,
		syncer:          config.syncer,
	}

	// Register event handlers for resource and metrics
	if queueInformer.informer != nil {
		queueInformer.informer.AddEventHandler(queueInformer.resourceHandlers(ctx))
		queueInformer.informer.AddEventHandler(queueInformer.metricHandlers())
	}

	return queueInformer, nil
}

// LegacySyncHandler is a deprecated signature for syncing resources.
type LegacySyncHandler func(obj interface{}) error

// ToSyncer returns the Syncer equivalent of the sync handler.
func (l LegacySyncHandler) ToSyncer() kubestate.Syncer {
	return l.ToSyncerWithDelete(nil)
}

// ToSyncerWithDelete returns the Syncer equivalent of the given sync handler and delete function.
func (l LegacySyncHandler) ToSyncerWithDelete(onDelete func(obj interface{})) kubestate.Syncer {
	var syncer kubestate.SyncFunc = func(ctx context.Context, event kubestate.ResourceEvent) error {
		logrus.New().WithField("event", fmt.Sprintf("%+v", event)).Trace("legacy syncer received event")
		switch event.Type() {
		case kubestate.ResourceDeleted:
			if onDelete != nil {
				onDelete(event.Resource())
			}
		case kubestate.ResourceAdded:
			// Added and updated are treated the same
			fallthrough
		case kubestate.ResourceUpdated:
			return l(event.Resource())
		default:
			return errors.Errorf("unexpected resource event type: %s", event.Type())
		}

		return nil
	}

	return syncer
}
