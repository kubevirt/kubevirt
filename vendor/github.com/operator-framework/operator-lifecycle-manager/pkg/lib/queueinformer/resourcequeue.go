package queueinformer

import (
	"fmt"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

// ResourceQueueSet is a set of workqueues that is assumed to be keyed by namespace
type ResourceQueueSet struct {
	queueSet map[string]workqueue.RateLimitingInterface
	mutex    sync.RWMutex
}

// NewResourceQueueSet returns a new queue set with the given queue map
func NewResourceQueueSet(queueSet map[string]workqueue.RateLimitingInterface) *ResourceQueueSet {
	return &ResourceQueueSet{queueSet: queueSet}
}

// NewEmptyResourceQueueSet returns a new queue set with an empty but initialized queue map
func NewEmptyResourceQueueSet() *ResourceQueueSet {
	return &ResourceQueueSet{queueSet: make(map[string]workqueue.RateLimitingInterface)}
}

// Set sets the queue at the given key
func (r *ResourceQueueSet) Set(key string, queue workqueue.RateLimitingInterface) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.queueSet[key] = queue
}

// Requeue requeues the resource in the set with the given name and namespace
func (r *ResourceQueueSet) Requeue(name, namespace string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// We can build the key directly, will need to change if queue uses different key scheme
	key := fmt.Sprintf("%s/%s", namespace, name)

	if queue, ok := r.queueSet[metav1.NamespaceAll]; len(r.queueSet) == 1 && ok {
		queue.Add(key)
		return nil
	}

	if queue, ok := r.queueSet[namespace]; ok {
		queue.Add(key)
		return nil
	}

	return fmt.Errorf("couldn't find queue for resource")
}

// RequeueByKey adds the given key to the resource queue that should contain it
func (r *ResourceQueueSet) RequeueByKey(key string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if queue, ok := r.queueSet[metav1.NamespaceAll]; len(r.queueSet) == 1 && ok {
		queue.Add(key)
		return nil
	}

	parts := strings.Split(key, "/")
	if len(parts) != 2 {
		return fmt.Errorf("non-namespaced key %s cannot be used with namespaced queues", key)
	}

	if queue, ok := r.queueSet[parts[0]]; ok {
		queue.Add(key)
		return nil
	}

	return fmt.Errorf("couldn't find queue for resource")
}

// Remove removes the resource in the set with the given name and namespace
func (r *ResourceQueueSet) Remove(name, namespace string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// We can build the key directly, will need to change if queue uses different key scheme
	key := fmt.Sprintf("%s/%s", namespace, name)

	if queue, ok := r.queueSet[metav1.NamespaceAll]; len(r.queueSet) == 1 && ok {
		queue.Forget(key)
		return nil
	}

	if queue, ok := r.queueSet[namespace]; ok {
		queue.Forget(key)
		return nil
	}

	return fmt.Errorf("couldn't find queue for resource")
}
