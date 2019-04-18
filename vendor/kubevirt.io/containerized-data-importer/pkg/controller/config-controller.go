package controller

import (
	"fmt"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	routeinformers "github.com/openshift/client-go/route/informers/externalversions/route/v1"
	routelisters "github.com/openshift/client-go/route/listers/route/v1"
	"github.com/pkg/errors"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	extensioninformers "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	extensionlisters "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdiclientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions/core/v1alpha1"
	listers "kubevirt.io/containerized-data-importer/pkg/client/listers/core/v1alpha1"
)

// ConfigController members
type ConfigController struct {
	client                                         kubernetes.Interface
	cdiClientSet                                   cdiclientset.Interface
	queue                                          workqueue.RateLimitingInterface
	ingressInformer, routeInformer, configInformer cache.SharedIndexInformer
	ingressLister                                  extensionlisters.IngressLister
	routeLister                                    routelisters.RouteLister
	configLister                                   listers.CDIConfigLister
	ingressesSynced                                cache.InformerSynced
	routesSynced                                   cache.InformerSynced
	configsSynced                                  cache.InformerSynced
	pullPolicy                                     string // Options: IfNotPresent, Always, or Never
	verbose                                        string // verbose levels: 1, 2, ...
	uploadProxyServiceName                         string
	configName                                     string
}

//NewConfigController creates a new ConfigController
func NewConfigController(client kubernetes.Interface,
	cdiClientSet cdiclientset.Interface,
	ingressInformer extensioninformers.IngressInformer,
	routeInformer routeinformers.RouteInformer,
	configInformer informers.CDIConfigInformer,
	uploadProxyServiceName string,
	configName string,
	pullPolicy string,
	verbose string) *ConfigController {
	c := &ConfigController{
		client:                 client,
		cdiClientSet:           cdiClientSet,
		queue:                  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		ingressInformer:        ingressInformer.Informer(),
		routeInformer:          routeInformer.Informer(),
		configInformer:         configInformer.Informer(),
		ingressLister:          ingressInformer.Lister(),
		routeLister:            routeInformer.Lister(),
		configLister:           configInformer.Lister(),
		ingressesSynced:        ingressInformer.Informer().HasSynced,
		routesSynced:           routeInformer.Informer().HasSynced,
		configsSynced:          configInformer.Informer().HasSynced,
		uploadProxyServiceName: uploadProxyServiceName,
		configName:             configName,
		pullPolicy:             pullPolicy,
		verbose:                verbose,
	}

	// Bind the ingress SharedIndexInformer to the ingress queue
	c.ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObjAdd,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*extensionsv1beta1.Ingress)
			oldDepl := old.(*extensionsv1beta1.Ingress)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known ingresses.
				return
			}
			c.handleObjUpdate(new)
		},

		DeleteFunc: c.handleObjDelete,
	})

	// Bind the route SharedIndexInformer to the route queue
	c.routeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObjAdd,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*routev1.Route)
			oldDepl := old.(*routev1.Route)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known routes.
				return
			}
			c.handleObjUpdate(new)
		},

		DeleteFunc: c.handleObjDelete,
	})

	// Bind the CDIconfig SharedIndexInformer to the CDIconfig queue
	c.configInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObjAdd,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*cdiv1.CDIConfig)
			oldDepl := old.(*cdiv1.CDIConfig)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known CDIconfigs.
				return
			}
			c.handleObjUpdate(new)
		},

		DeleteFunc: c.handleObjDelete,
	})

	return c
}

func (c *ConfigController) handleObjAdd(obj interface{}) {
	c.handleObject(obj, "add")
}
func (c *ConfigController) handleObjUpdate(obj interface{}) {
	c.handleObject(obj, "update")
}
func (c *ConfigController) handleObjDelete(obj interface{}) {
	c.handleObject(obj, "delete")
}

func (c *ConfigController) handleObject(obj interface{}, verb string) {

	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(errors.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(errors.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(3).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(3).Infof("Processing object: %s", object.GetName())

	var config *cdiv1.CDIConfig
	config, err := c.configLister.Get(c.configName)
	if err != nil {
		runtime.HandleError(errors.Errorf("error getting CDI config: %s", err))
		return
	}

	if ing, ok := obj.(*extensionsv1beta1.Ingress); ok {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName == c.uploadProxyServiceName {
					c.enqueueCDIConfig(config)
					return
				}
			}
		}
	}
	if route, ok := obj.(*routev1.Route); ok {
		if route.Spec.To.Name == c.uploadProxyServiceName {
			c.enqueueCDIConfig(config)
			return
		}
	}
	if conf, ok := obj.(*cdiv1.CDIConfig); ok {
		if conf.Name == config.Name {
			c.enqueueCDIConfig(conf)
			return
		}
	}

	return
}

func (c *ConfigController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ConfigController) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)

		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.queue.Forget(obj)
			runtime.HandleError(errors.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			return errors.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.queue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil

	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *ConfigController) syncHandler(key string) error {
	updateConfig := false
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(errors.Errorf("invalid resource key: %s", key))
		return nil
	}

	config, err := c.configLister.Get(name)
	if err != nil {
		runtime.HandleError(errors.Errorf("CDIConfig '%s' in work queue no longer exists", key))
		return nil
	}

	var url string
	if config.Spec.UploadProxyURLOverride != nil {
		url = *config.Spec.UploadProxyURLOverride
	} else {

		routes, err := c.routeLister.List(labels.NewSelector())
		if err != nil {
			return err
		}
		for _, route := range routes {
			url = getURLFromRoute(route, c.uploadProxyServiceName)
			if url != "" {
				break
			}
		}
		if url == "" {
			ingresses, err := c.ingressLister.List(labels.NewSelector())
			if err != nil {
				return err
			}
			for _, ing := range ingresses {
				url = getURLFromIngress(ing, c.uploadProxyServiceName)
				if url != "" {
					break
				}
			}
		}
	}

	if (config.Status.UploadProxyURL != nil && url == *config.Status.UploadProxyURL) || (config.Status.UploadProxyURL == nil && url == "") {
		updateConfig = false
	} else {
		updateConfig = true
	}
	newConfig := config.DeepCopy()
	if updateConfig {
		// mutate newConfig
		if url == "" {
			newConfig.Status.UploadProxyURL = nil
		} else {
			newConfig.Status.UploadProxyURL = &url
		}
	}

	storageClass, err := c.scratchSpaceStorageClassStatus(config)

	if storageClass == config.Status.ScratchSpaceStorageClass {
		updateConfig = updateConfig || false
	} else {
		newConfig.Status.ScratchSpaceStorageClass = storageClass
		updateConfig = true
	}

	if updateConfig {
		err = updateCDIConfig(c.cdiClientSet, newConfig)
		if err != nil {
			return fmt.Errorf("Error updating CDI Config %s: %s", key, err)
		}
	}

	return nil
}

func (c *ConfigController) scratchSpaceStorageClassStatus(config *cdiv1.CDIConfig) (string, error) {
	storageClassList, err := c.client.StorageV1().StorageClasses().List(metav1.ListOptions{})
	if err != nil {
		klog.Warningf("Unable to find storage classes, %v\n", err)
	}
	// Check config for scratch space class
	if config.Spec.ScratchSpaceStorageClass != nil {
		for _, storageClass := range storageClassList.Items {
			if storageClass.Name == *config.Spec.ScratchSpaceStorageClass {
				return storageClass.Name, nil
			}
		}
	}
	// Check for default storage class.
	for _, storageClass := range storageClassList.Items {
		if defaultClassValue, ok := storageClass.Annotations[AnnDefaultStorageClass]; ok {
			if defaultClassValue == "true" {
				return storageClass.Name, nil
			}
		}
	}
	return "", nil
}

// Init is meant to be called synchroniously when the the controller is starting
func (c *ConfigController) Init() error {
	klog.V(3).Infoln("Creating CDI config if necessary")

	if err := EnsureCDIConfigExists(c.client, c.cdiClientSet, c.configName); err != nil {
		runtime.HandleError(err)
		return errors.Wrap(err, "Error creating CDI config")
	}

	return nil
}

// Run sets up ConfigController state and executes main event loop
func (c *ConfigController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer func() {
		c.queue.ShutDown()
	}()

	klog.V(3).Infoln("Starting config controller Run loop")
	if threadiness < 1 {
		return errors.Errorf("expected >0 threads, got %d", threadiness)
	}

	informersNeedingSync := []cache.InformerSynced{c.ingressesSynced}
	if isOpenshift := IsOpenshift(c.client); isOpenshift {
		informersNeedingSync = append(informersNeedingSync, c.routesSynced)
	}
	if ok := cache.WaitForCacheSync(stopCh, informersNeedingSync...); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	klog.V(3).Infoln("ConfigController cache has synced")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")
	return nil
}

func (c *ConfigController) enqueueCDIConfig(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}
