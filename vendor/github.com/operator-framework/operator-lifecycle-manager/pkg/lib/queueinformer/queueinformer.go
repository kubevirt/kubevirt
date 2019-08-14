package queueinformer

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// SyncHandler is the function that reconciles the controlled object when seen
type SyncHandler func(obj interface{}) error

// QueueInformer ties an informer to a queue in order to process events from the informer
// the informer watches objects of interest and adds objects to the queue for processing
// the syncHandler is called for all objects on the queue
type QueueInformer struct {
	queue                     workqueue.RateLimitingInterface
	informer                  cache.SharedIndexInformer
	syncHandler               SyncHandler
	resourceEventHandlerFuncs *cache.ResourceEventHandlerFuncs
	name                      string
}

// enqueue adds a key to the queue. If obj is a key already it gets added directly.
// Otherwise, the key is extracted via keyFunc.
func (q *QueueInformer) enqueue(obj interface{}) {
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

// keyFunc turns an object into a key for the queue. In the future will use a (name, namespace) struct as key
func (q *QueueInformer) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Infof("creating key failed: %s", err)
		return k, false
	}

	return k, true
}

// defaultResourceEventhandlerFuncs provides the default implementation for responding to events
// these simply log the event and add the object's key to the queue for later processing
func (q *QueueInformer) defaultResourceEventHandlerFuncs() *cache.ResourceEventHandlerFuncs {
	return &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, ok := q.keyFunc(obj)
			if !ok {
				return
			}

			log.Infof("%s added", key)
			q.enqueue(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, ok := q.keyFunc(obj)
			if !ok {
				return
			}

			log.Infof("%s deleted", key)
			q.queue.Forget(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, ok := q.keyFunc(newObj)
			if !ok {
				return
			}

			log.Infof("%s updated", key)
			q.enqueue(key)
		},
	}
}

// New creates a set of new queueinformers given a name, a set of informers, and a sync handler to handle the objects
// that the operator is managing. Optionally, custom event handler funcs can be passed in (defaults will be provided)
func New(queue workqueue.RateLimitingInterface, informers []cache.SharedIndexInformer, handler SyncHandler, funcs *cache.ResourceEventHandlerFuncs, name string) []*QueueInformer {
	queueInformers := []*QueueInformer{}
	for _, informer := range informers {
		queueInformers = append(queueInformers, NewInformer(queue, informer, handler, funcs, name))
	}
	return queueInformers
}

// NewInformer creates a new queueinformer given a name, an informer, and a sync handler to handle the objects
// that the operator is managing. Optionally, custom event handler funcs can be passed in (defaults will be provided)
func NewInformer(queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, handler SyncHandler, funcs *cache.ResourceEventHandlerFuncs, name string) *QueueInformer {
	queueInformer := &QueueInformer{
		queue:       queue,
		informer:    informer,
		syncHandler: handler,
		name:        name,
	}
	if funcs == nil {
		queueInformer.resourceEventHandlerFuncs = queueInformer.defaultResourceEventHandlerFuncs()
	} else {
		queueInformer.resourceEventHandlerFuncs = funcs
	}
	queueInformer.informer.AddEventHandler(queueInformer.resourceEventHandlerFuncs)
	return queueInformer
}
