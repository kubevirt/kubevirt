package virthandler

import (
	kubeapi "k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/libvirt"
	"time"
)

type Controller struct {
	indexer       cache.Indexer
	queue         workqueue.RateLimitingInterface
	controller    *cache.Controller
	domainManager libvirt.DomainManager
	recorder      record.EventRecorder
	restclient    rest.RESTClient
}

func NewVMController(listWatcher cache.ListerWatcher, domainManager libvirt.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient) (cache.Indexer, *Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, controller := cache.NewIndexerInformer(listWatcher, &v1.VM{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	return indexer, &Controller{
		controller:    controller,
		indexer:       indexer,
		queue:         queue,
		domainManager: domainManager,
		recorder:      recorder,
		restclient:    restClient,
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer kubecli.NewPanicCatcher()
	defer c.queue.ShutDown()
	logging.DefaultLogger().Info().Msg("Starting VM controller")

	go c.controller.Run(stopCh)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping VM controller")
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		logging.DefaultLogger().V(3).Info().Msg("Exiting")
		return false
	}
	defer c.queue.Done(key)
	// Fetch the latest Vm state from cache
	obj, exists, err := c.indexer.GetByKey(key.(string))
	logging.DefaultLogger().V(3).Info().Msgf("Object %s", obj)

	if err != nil {
		// TODO do something more smart here
		c.queue.Forget(key)
		return true
	}

	// Retrieve the VM
	var vm *v1.VM
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			// TODO do something more smart here
			c.queue.Forget(key)
			return true
		}
		vm = libvirt.NewVMReferenceFromName(name)
	} else {
		vm = obj.(*v1.VM)
	}
	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Processing VM update")

	// Process the VM
	if !exists {
		// Since the VM was not in the cache, we delete it
		err = c.domainManager.KillVM(vm)
	} else {
		// Synchronize the VM state
		err = c.domainManager.SyncVM(vm)

		// Update VM status to running
		if err == nil && vm.Status.Phase != v1.Running {
			obj, err = kubeapi.Scheme.Copy(vm)
			if err == nil {
				vm = obj.(*v1.VM)
				vm.Status.Phase = v1.Running
				err = c.restclient.Put().Resource("vms").Body(vm).
					Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
			}
		}
	}

	if err != nil {
		// Something went wrong, reenqueue the item with a delay
		logging.DefaultLogger().V(3).Info().Object(vm).Msgf("Synchronizing the VM failed with: %s", err)
		c.recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
		c.queue.AddRateLimited(key)
		return true
	}

	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Synchronizing the VM succeeded")
	c.queue.Forget(key)
	return true
}
