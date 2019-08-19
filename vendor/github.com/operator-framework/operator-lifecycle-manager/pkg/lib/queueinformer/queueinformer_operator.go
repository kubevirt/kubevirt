package queueinformer

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

// An Operator is a collection of QueueInformers
// OpClient is used to establish the connection to kubernetes
type Operator struct {
	queueInformers []*QueueInformer
	queueIndexers  []*QueueIndexer
	informers      []cache.SharedIndexInformer
	OpClient       operatorclient.ClientInterface
	Log            *logrus.Logger
	syncCh         chan error
}

// NewOperator creates a new Operator configured to manage the cluster defined in kubeconfig.
func NewOperator(kubeconfig string, logger *logrus.Logger, queueInformers ...*QueueInformer) (*Operator, error) {
	opClient := operatorclient.NewClientFromConfig(kubeconfig, logger)
	if queueInformers == nil {
		queueInformers = []*QueueInformer{}
	}
	operator := &Operator{
		OpClient:       opClient,
		queueInformers: queueInformers,
		Log:            logger,
	}
	return operator, nil
}

func NewOperatorFromClient(opClient operatorclient.ClientInterface, logger *logrus.Logger, queueInformers ...*QueueInformer) (*Operator, error) {
	if queueInformers == nil {
		queueInformers = []*QueueInformer{}
	}
	operator := &Operator{
		OpClient:       opClient,
		queueInformers: queueInformers,
		Log:            logger,
	}
	return operator, nil
}

// RegisterQueueInformer adds a QueueInformer to this operator
func (o *Operator) RegisterQueueInformer(queueInformer *QueueInformer) {
	if o.queueInformers == nil {
		o.queueInformers = []*QueueInformer{}
	}
	o.queueInformers = append(o.queueInformers, queueInformer)
}

// RegisterInformer adds an Informer to this operator
func (o *Operator) RegisterInformer(informer cache.SharedIndexInformer) {
	if o.informers == nil {
		o.informers = []cache.SharedIndexInformer{}
	}
	o.informers = append(o.informers, informer)
}

// RegisterQueueIndexer adds a QueueIndexer to this operator
func (o *Operator) RegisterQueueIndexer(indexer *QueueIndexer) {
	if o.queueIndexers == nil {
		o.queueIndexers = []*QueueIndexer{}
	}
	o.queueIndexers = append(o.queueIndexers, indexer)
}

// Run starts the operator's control loops
func (o *Operator) Run(stopc <-chan struct{}) (ready, done chan struct{}, atLevel chan error) {
	ready = make(chan struct{})
	atLevel = make(chan error, 25)
	done = make(chan struct{})

	o.syncCh = atLevel

	go func() {
		defer func() {
			close(ready)
			close(atLevel)
			close(done)
		}()

		for _, queueInformer := range o.queueInformers {
			defer queueInformer.queue.ShutDown()
		}

		errChan := make(chan error)
		go func() {
			v, err := o.OpClient.KubernetesInterface().Discovery().ServerVersion()
			if err != nil {
				errChan <- errors.Wrap(err, "communicating with server failed")
				return
			}
			o.Log.Infof("connection established. cluster-version: %v", v)
			errChan <- nil
		}()

		var hasSyncedCheckFns []cache.InformerSynced
		for _, queueInformer := range o.queueInformers {
			hasSyncedCheckFns = append(hasSyncedCheckFns, queueInformer.informer.HasSynced)
		}
		for _, informer := range o.informers {
			hasSyncedCheckFns = append(hasSyncedCheckFns, informer.HasSynced)
		}

		select {
		case err := <-errChan:
			if err != nil {
				o.Log.Infof("operator not ready: %s", err.Error())
				return
			}
			o.Log.Info("operator ready")
		case <-stopc:
			return
		}

		o.Log.Info("starting informers...")
		for _, queueInformer := range o.queueInformers {
			go queueInformer.informer.Run(stopc)
		}

		for _, informer := range o.informers {
			go informer.Run(stopc)
		}

		o.Log.Info("waiting for caches to sync...")
		if ok := cache.WaitForCacheSync(stopc, hasSyncedCheckFns...); !ok {
			o.Log.Info("failed to wait for caches to sync")
			return
		}

		o.Log.Info("starting workers...")
		for _, queueInformer := range o.queueInformers {
			go o.worker(queueInformer)
			go o.worker(queueInformer)
		}

		for _, queueIndexer := range o.queueIndexers {
			go o.indexerWorker(queueIndexer)
			go o.indexerWorker(queueIndexer)
		}
		ready <- struct{}{}
		<-stopc
	}()

	return
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (o *Operator) worker(loop *QueueInformer) {
	for o.processNextWorkItem(loop) {
	}
}

func (o *Operator) processNextWorkItem(loop *QueueInformer) bool {
	queue := loop.queue
	key, quit := queue.Get()

	if quit {
		return false
	}
	defer queue.Done(key)

	// requeue five times on error
	err := o.sync(loop, key.(string))
	if err != nil && queue.NumRequeues(key.(string)) < 5 {
		o.Log.Infof("retrying %s", key)
		utilruntime.HandleError(errors.Wrap(err, fmt.Sprintf("Sync %q failed", key)))
		queue.AddRateLimited(key)
		return true
	}
	queue.Forget(key)

	select {
	case o.syncCh <- err:
	default:
	}

	if err := loop.HandleMetrics(); err != nil {
		o.Log.Error(err)
	}
	return true
}

func (o *Operator) sync(loop *QueueInformer, key string) error {
	logger := o.Log.WithField("queue", loop.name).WithField("key", key)
	obj, exists, err := loop.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// For now, we ignore the case where an object used to exist but no longer does
		logger.Info("couldn't get from queue")
		logger.Debugf("have keys: %v", loop.informer.GetIndexer().ListKeys())
		return nil
	}
	return loop.syncHandler(obj)
}

// This provides the same function as above, but for queues that are not auto-fed by informers.
// indexerWorker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (o *Operator) indexerWorker(loop *QueueIndexer) {
	for o.processNextIndexerWorkItem(loop) {
	}
}

func (o *Operator) processNextIndexerWorkItem(loop *QueueIndexer) bool {
	queue := loop.queue
	key, quit := queue.Get()

	if quit {
		return false
	}
	defer queue.Done(key)

	// requeue five times on error
	if err := o.syncIndexer(loop, key.(string)); err != nil && queue.NumRequeues(key.(string)) < 5 {
		o.Log.Infof("retrying %s", key)
		utilruntime.HandleError(errors.Wrap(err, fmt.Sprintf("Sync %q failed", key)))
		queue.AddRateLimited(key)
		return true
	}
	queue.Forget(key)
	if err := loop.HandleMetrics(); err != nil {
		o.Log.Error(err)
	}
	return true
}

func (o *Operator) syncIndexer(loop *QueueIndexer, key string) error {
	logger := o.Log.WithField("queue", loop.name).WithField("key", key)
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	indexer, ok := loop.indexers[namespace]
	if !ok {
		if indexer, ok = loop.indexers[v1.NamespaceAll]; !ok {
			return fmt.Errorf("no indexer found for %s, have %v", namespace, loop.indexers)
		}
	}
	obj, exists, err := indexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// For now, we ignore the case where an object used to exist but no longer does
		logger.Info("couldn't get from queue")
		logger.Debugf("have keys: %v", indexer.ListKeys())
		return nil
	}
	return loop.syncHandler(obj)
}
