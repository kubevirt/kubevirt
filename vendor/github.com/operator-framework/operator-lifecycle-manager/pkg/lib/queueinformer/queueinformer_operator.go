package queueinformer

import (
	"context"
	"fmt"
	"sync"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubestate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/cache"
)

// Operator describes a Reconciler that manages a set of QueueInformers.
type Operator interface {
	// Ready returns a channel that is closed when the Operator is ready to run.
	Ready() <-chan struct{}

	// Done returns a channel that is closed when the Operator is done running.
	Done() <-chan struct{}

	// AtLevel returns a channel that emits errors when the Operator is not at level.
	AtLevel() <-chan error

	// Started returns true if RunInformers() has been called, false otherwise.
	Started() bool

	// HasSynced returns true if the Operator's Informers have synced, false otherwise.
	HasSynced() bool

	// RegisterQueueInformer registers the given QueueInformer with the Operator.
	// This method returns an error if the Operator has already been started.
	RegisterQueueInformer(queueInformer *QueueInformer) error

	// RegisterInformer registers an informer with the Operator.
	// This method returns an error if the Operator has already been started.
	RegisterInformer(cache.SharedIndexInformer) error

	// RunInformers starts the Operator's underlying Informers.
	RunInformers(ctx context.Context)

	// Run starts the Operator and its underlying Informers.
	Run(ctx context.Context)
}

type operator struct {
	discovery        discovery.DiscoveryInterface
	queueInformers   []*QueueInformer
	informers        []cache.SharedIndexInformer
	hasSynced        cache.InformerSynced
	mu               sync.RWMutex
	numWorkers       int
	runInformersOnce sync.Once
	reconcileOnce    sync.Once
	logger           *logrus.Logger
	ready            chan struct{}
	done             chan struct{}
	atLevel          chan error
	syncCh           chan error
	started          bool
}

func (o *operator) Ready() <-chan struct{} {
	return o.ready
}

func (o *operator) Done() <-chan struct{} {
	return o.done
}

func (o *operator) AtLevel() <-chan error {
	return o.atLevel
}

func (o *operator) HasSynced() bool {
	return o.hasSynced()
}

func (o *operator) Started() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.started
}

func (o *operator) RegisterQueueInformer(queueInformer *QueueInformer) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	err := errors.New("failed to register queue informer")
	if queueInformer == nil {
		return errors.Wrap(err, "nil queue informer")
	}

	if o.started {
		return errors.Wrap(err, "operator already started")
	}

	o.queueInformers = append(o.queueInformers, queueInformer)

	// Some QueueInformers do not have informers associated with them.
	// Only add to the list of informers when one exists.
	if informer := queueInformer.informer; informer != nil {
		o.registerInformer(informer)
	}

	return nil
}

func (o *operator) RegisterInformer(informer cache.SharedIndexInformer) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	err := errors.New("failed to register informer")
	if informer == nil {
		return errors.Wrap(err, "nil informer")
	}

	if o.started {
		return errors.Wrap(err, "operator already started")
	}

	o.registerInformer(informer)

	return nil
}

func (o *operator) registerInformer(informer cache.SharedIndexInformer) {
	o.informers = append(o.informers, informer)
	o.addHasSynced(informer.HasSynced)
}

func (o *operator) addHasSynced(hasSynced cache.InformerSynced) {
	if o.hasSynced == nil {
		o.hasSynced = hasSynced
		return
	}

	prev := o.hasSynced
	o.hasSynced = func() bool {
		return prev() && hasSynced()
	}
}

func (o *operator) RunInformers(ctx context.Context) {
	o.runInformersOnce.Do(func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		for _, informer := range o.informers {
			go informer.Run(ctx.Done())
		}

		o.started = true
		o.logger.Infof("informers started")
	})
}

// Run starts the operator's control loops.
func (o *operator) Run(ctx context.Context) {
	o.reconcileOnce.Do(func() {
		go o.run(ctx)
	})
}

func (o *operator) run(ctx context.Context) {
	defer func() {
		close(o.atLevel)
		close(o.done)
	}()

	for _, queueInformer := range o.queueInformers {
		defer queueInformer.queue.ShutDown()
	}

	errs := make(chan error)
	go func() {
		defer close(errs)
		v, err := o.discovery.ServerVersion()
		if err != nil {
			errs <- errors.Wrap(err, "communicating with server failed")
			return
		}
		o.logger.Infof("connection established. cluster-version: %v", v)
	}()

	select {
	case err := <-errs:
		if err != nil {
			o.logger.Infof("operator not ready: %s", err.Error())
			return
		}
		o.logger.Info("operator ready")
	case <-ctx.Done():
		return
	}

	o.logger.Info("starting informers...")
	o.RunInformers(ctx)

	o.logger.Info("waiting for caches to sync...")
	if ok := cache.WaitForCacheSync(ctx.Done(), o.hasSynced); !ok {
		o.logger.Info("failed to wait for caches to sync")
		return
	}

	o.logger.Info("starting workers...")
	for _, queueInformer := range o.queueInformers {
		for w := 0; w < o.numWorkers; w++ {
			go o.worker(ctx, queueInformer)
		}
	}

	close(o.ready)
	<-ctx.Done()
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (o *operator) worker(ctx context.Context, loop *QueueInformer) {
	for o.processNextWorkItem(ctx, loop) {
	}
}

func (o *operator) processNextWorkItem(ctx context.Context, loop *QueueInformer) bool {
	queue := loop.queue
	item, quit := queue.Get()

	if quit {
		return false
	}
	defer queue.Done(item)

	logger := o.logger.WithField("item", item)
	logger.WithField("queue-length", queue.Len()).Trace("popped queue")

	event, ok := item.(kubestate.ResourceEvent)
	if !ok || event.Type() != kubestate.ResourceDeleted {
		// Get the key
		key, keyable := loop.key(item)
		if !keyable {
			logger.WithField("item", item).Warn("could not form key")
			queue.Forget(item)
			return true
		}

		logger = logger.WithField("cache-key", key)

		// Get the current cached version of the resource
		resource, exists, err := loop.indexer.GetByKey(key)
		if err != nil {
			logger.WithError(err).Error("cache get failed")
			queue.Forget(item)
			return true
		}
		if !exists {
			logger.WithField("existing-cache-keys", loop.indexer.ListKeys()).Debug("cache get failed, key not in cache")
			queue.Forget(item)
			return true
		}

		if !ok {
			event = kubestate.NewResourceEvent(kubestate.ResourceUpdated, resource)
		} else {
			event = kubestate.NewResourceEvent(event.Type(), resource)
		}
	}

	// Sync and requeue on error (throw out failed deletion syncs)
	err := loop.Sync(ctx, event)
	if requeues := queue.NumRequeues(item); err != nil && requeues < 8 && event.Type() != kubestate.ResourceDeleted {
		logger.WithField("requeues", requeues).Trace("requeuing with rate limiting")
		utilruntime.HandleError(errors.Wrap(err, fmt.Sprintf("sync %q failed", item)))
		queue.AddRateLimited(item)
		return true
	}
	queue.Forget(item)

	select {
	case o.syncCh <- err:
	default:
	}

	return true
}

// NewOperator returns a new Operator configured to manage the cluster with the given discovery client.
func NewOperator(disc discovery.DiscoveryInterface, options ...OperatorOption) (Operator, error) {
	config := defaultOperatorConfig()
	config.discovery = disc
	config.apply(options)
	if err := config.validate(); err != nil {
		return nil, err
	}

	return newOperatorFromConfig(config)

}

func newOperatorFromConfig(config *operatorConfig) (Operator, error) {
	op := &operator{
		discovery:  config.discovery,
		numWorkers: config.numWorkers,
		logger:     config.logger,
		ready:      make(chan struct{}),
		done:       make(chan struct{}),
		atLevel:    make(chan error, 25),
	}
	op.syncCh = op.atLevel

	// Register QueueInformers and Informers
	for _, queueInformer := range op.queueInformers {
		if err := op.RegisterQueueInformer(queueInformer); err != nil {
			return nil, err
		}
	}
	for _, informer := range op.informers {
		if err := op.RegisterInformer(informer); err != nil {
			return nil, err
		}
	}

	return op, nil
}
