package kubecli

import (
	"flag"
	"github.com/go-kit/kit/log/levels"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/pkg/watch"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/tools/clientcmd"
	"kubevirt.io/core/pkg/api/v1"
	"runtime/debug"
	"time"
)

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
}

func Get() (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
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

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
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
	listFunc := func(options api.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, api.ParameterCodec).
			FieldsSelectorParam(fieldSelector).
			LabelsSelectorParam(labelSelector).
			Do().
			Get()
	}
	watchFunc := func(options api.ListOptions) (watch.Interface, error) {
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
		RetryOnError:     false,

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
							return err
						}
					} else {
						if err := clientState.Add(d.Object); err != nil {
							return err
						}
						err = h.OnAdd(d.Object)
						if err != nil {
							return err
						}
					}
				case cache.Deleted:
					if err := clientState.Delete(d.Object); err != nil {
						return err
					}
					err := h.OnDelete(d.Object)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return clientState, cache.New(cfg)
}

type ResourceEventHandler interface {
	OnAdd(obj interface{}) error
	OnUpdate(oldObj, newObj interface{}) error
	OnDelete(obj interface{}) error
}

func NewPanicCatcher(logger levels.Levels) func() {
	return func() {
		if r := recover(); r != nil {
			logger.Crit().Log("stacktrace", debug.Stack(), "msg", r)
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
