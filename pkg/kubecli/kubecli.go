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

func Get() (*kubernetes.Clientset, error) {

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

func GetRESTClient() (*rest.RESTClient, error) {

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

func NewInformer(
	lw cache.ListerWatcher,
	objType runtime.Object,
	resyncPeriod time.Duration,
	h ResourceEventHandler,
) (cache.Indexer, *cache.Controller) {
	clientState := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
	fifo := cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, clientState)

	cfg := &cache.Config{
		Queue:            fifo,
		ListerWatcher:    lw,
		ObjectType:       objType,
		FullResyncPeriod: resyncPeriod,
		RetryOnError:     true,

		Process: func(obj interface{}) error {
			// from oldest to newest

			for _, d := range obj.(cache.Deltas) {
				switch d.Type {
				case cache.Sync, cache.Added, cache.Updated:
					if old, exists, err := clientState.Get(d.Object); err == nil && exists {
						if err := clientState.Update(d.Object); err != nil {
							return err
						}
						err = h.OnUpdate(old, d.Object)
						if err != nil {
							// TODO real backoff strategy
							// TODO solve this by using workqueues as soon as they hit client-go
							time.Sleep(1 * time.Second)
							return handleErr(err)
						}
					} else {
						if err := clientState.Add(d.Object); err != nil {
							return err
						}
						err = h.OnAdd(d.Object)
						if err != nil {
							// TODO real backoff strategy
							// TODO solve this by using workqueues as soon as they hit client-go
							time.Sleep(1 * time.Second)
							return handleErr(err)
						}
					}
				case cache.Deleted:
					err := h.OnDelete(d.Object)
					if err != nil {
						// TODO real backoff strategy
						// TODO solve this by using workqueues as soon as they hit client-go
						time.Sleep(1 * time.Second)
						return handleErr(err)
					}
					if err := clientState.Delete(d.Object); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return clientState, cache.New(cfg)
}

/*
Helper to translate between requeue errors like they are used in queues and the controller error handling.
This allows  to use the controlle rerror handling if it is enable and not trigger the queues reenqueue logic.
*/
func handleErr(err error) error {
	if e, ok := err.(cache.ErrRequeue); ok == true {
		return e.Err
	}
	return nil
}

type ResourceEventHandler interface {
	OnAdd(obj interface{}) error
	OnUpdate(oldObj, newObj interface{}) error
	OnDelete(obj interface{}) error
}

func NewPanicCatcher() func() {
	return func() {
		if r := recover(); r != nil {
			logging.DefaultLogger().Critical().Log("stacktrace", debug.Stack(), "msg", r)
		}
	}
}

type ResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{}) error
	UpdateFunc func(oldObj, newObj interface{}) error
	DeleteFunc func(obj interface{}) error
}

func (r ResourceEventHandlerFuncs) OnAdd(obj interface{}) error {
	return r.AddFunc(obj)
}

func (r ResourceEventHandlerFuncs) OnUpdate(oldObj, newObj interface{}) error {
	return r.UpdateFunc(oldObj, newObj)
}

func (r ResourceEventHandlerFuncs) OnDelete(obj interface{}) error {
	return r.DeleteFunc(obj)
}
