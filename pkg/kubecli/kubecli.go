package kubecli

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"runtime/debug"
	"time"
)

var (
	kubeconfig string
	master     string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
}

func GetFromFlags(master string, kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	return kubernetes.NewForConfig(config)
}

func Get() (*kubernetes.Clientset, error) {
	return GetFromFlags(master, kubeconfig)
}

func GetRESTClient() (*rest.RESTClient, error) {
	return GetRESTClientFromFlags(master, kubeconfig)
}

func GetRESTClientFromFlags(master string, kubeconfig string) (*rest.RESTClient, error) {

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	return rest.RESTClientFor(config)
}

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func NewListWatchFromClient(c cache.Getter, resource string, namespace string, fieldSelector fields.Selector, labelSelector labels.Selector) *cache.ListWatch {
	listFunc := func(options kubev1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, api.ParameterCodec).
			FieldsSelectorParam(fieldSelector).
			LabelsSelectorParam(labelSelector).
			Do().
			Get()
	}
	watchFunc := func(options kubev1.ListOptions) (watch.Interface, error) {
		return c.Get().
			Prefix("watch").
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, api.ParameterCodec).
			FieldsSelectorParam(fieldSelector).
			LabelsSelectorParam(labelSelector).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func HandlePanic() {
	if r := recover(); r != nil {
		logging.DefaultLogger().Critical().Log("stacktrace", debug.Stack(), "msg", r)
	}
}

func NewResourceEventHandlerFuncsForQorkqueue(queue workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
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
	}
}

type Controller struct {
	indexer  cache.Store
	queue    workqueue.RateLimitingInterface
	informer cache.ControllerInterface
	dispatch ControllerDispatch
	done     chan struct{}
}

func NewController(lw cache.ListerWatcher, queue workqueue.RateLimitingInterface, objType runtime.Object, dispatch ControllerDispatch) (cache.Store, *Controller) {

	indexer, informer := cache.NewIndexerInformer(lw, objType, 0, NewResourceEventHandlerFuncsForQorkqueue(queue), cache.Indexers{})
	return NewControllerFromInformer(indexer, informer, queue, dispatch)
}

type ControllerDispatch interface {
	Execute( /*cache*/ cache.Store /*queue*/, workqueue.RateLimitingInterface /*key*/, interface{})
}

func NewControllerFromInformer(indexer cache.Store, informer cache.ControllerInterface, queue workqueue.RateLimitingInterface, dispatch ControllerDispatch) (cache.Store, *Controller) {
	c := &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		done:     make(chan struct{}),
		dispatch: dispatch,
	}
	return indexer, c
}

type ControllerFunc func(cache.Store, workqueue.RateLimitingInterface, interface{})

func (c *Controller) callControllerFn(s cache.Store, w workqueue.RateLimitingInterface) bool {
	key, quit := w.Get()
	if quit {
		close(c.done)
		return false
	} else {
		defer w.Done(key)
		c.dispatch.Execute(s, w, key)
		return true
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer HandlePanic()
	defer c.queue.ShutDown()
	logging.DefaultLogger().Info().Msg("Starting controller.")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
}

func (c *Controller) StartInformer(stopCh chan struct{}) {
	go c.informer.Run(stopCh)
}

func (c *Controller) WaitForSync(stopCh chan struct{}) {
	cache.WaitForCacheSync(stopCh, c.informer.HasSynced)
}

func (c *Controller) runWorker() {
	for c.callControllerFn(c.indexer, c.queue) {
	}
}

func (c *Controller) WaitUntilDone() {
	<-c.done
}

// Shut down the embedded queue. After the shutdown was issued, all items already in the queue will be processed but no
// new items will be accepted. It is possible to wait via #WaitUntilDone() until the last item was processed.
func (c *Controller) ShutDownQueue() {
	c.queue.ShutDown()
}
