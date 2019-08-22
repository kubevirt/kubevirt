package controller

import (
	"time"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"kubevirt.io/containerized-data-importer/pkg/expectations"
)

const (
	// AnnAPIGroup is the APIGroup for CDI
	AnnAPIGroup = "cdi.kubevirt.io"
	// AnnCreatedBy is a pod annotation indicating if the pod was created by the PVC
	AnnCreatedBy = AnnAPIGroup + "/storage.createdByController"
	// AnnPodPhase is a PVC annotation indicating the related pod progress (phase)
	AnnPodPhase = AnnAPIGroup + "/storage.pod.phase"
	// AnnPodReady tells whether the pod is ready
	AnnPodReady = AnnAPIGroup + "/storage.pod.ready"
	// AnnOwnerRef is used when owner is in a different namespace
	AnnOwnerRef = AnnAPIGroup + "/storage.ownerRef"
)

//Controller is a struct that contains common information and functionality used by all CDI controllers.
type Controller struct {
	clientset                kubernetes.Interface
	queue                    workqueue.RateLimitingInterface
	pvcInformer, podInformer cache.SharedIndexInformer
	pvcLister                corelisters.PersistentVolumeClaimLister
	podLister                corelisters.PodLister
	pvcsSynced               cache.InformerSynced
	podsSynced               cache.InformerSynced
	image                    string
	pullPolicy               string // Options: IfNotPresent, Always, or Never
	verbose                  string // verbose levels: 1, 2, ...
	podExpectations          *expectations.UIDTrackingControllerExpectations
}

//NewController is called when we instantiate any CDI controller.
func NewController(client kubernetes.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	image string,
	pullPolicy string,
	verbose string) *Controller {
	c := &Controller{
		clientset:       client,
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pvcInformer:     pvcInformer.Informer(),
		podInformer:     podInformer.Informer(),
		pvcLister:       pvcInformer.Lister(),
		podLister:       podInformer.Lister(),
		pvcsSynced:      pvcInformer.Informer().HasSynced,
		podsSynced:      podInformer.Informer().HasSynced,
		image:           image,
		pullPolicy:      pullPolicy,
		verbose:         verbose,
		podExpectations: expectations.NewUIDTrackingControllerExpectations(expectations.NewControllerExpectations()),
	}

	// Bind the pvc SharedIndexInformer to the pvc queue
	c.pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueuePVC,
		UpdateFunc: func(old, new interface{}) {
			c.enqueuePVC(new)
		},
		DeleteFunc: c.enqueuePVC,
	})

	// Bind the pod SharedIndexInformer to the pod queue
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handlePodAdd,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*v1.Pod)
			oldDepl := old.(*v1.Pod)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known PVCs.
				// Two different versions of the same PVCs will always have different RVs.
				return
			}
			c.handlePodUpdate(new)
		},
		DeleteFunc: c.handlePodDelete,
	})

	return c
}

func (c *Controller) handlePodAdd(obj interface{}) {
	c.handlePodObject(obj, "add")
}
func (c *Controller) handlePodUpdate(obj interface{}) {
	c.handlePodObject(obj, "update")
}
func (c *Controller) handlePodDelete(obj interface{}) {
	c.handlePodObject(obj, "delete")
}

func (c *Controller) observePodCreate(pvcKey string) {
	c.podExpectations.CreationObserved(pvcKey)
}

func (c *Controller) handlePodObject(obj interface{}, verb string) {
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

	_, createdByUs := object.GetAnnotations()[AnnCreatedBy]
	if !createdByUs {
		klog.V(3).Infof("Ignoring pod %s/%s, as it's not created by us", object.GetNamespace(), object.GetName())
		return
	}

	klog.V(3).Infof("Processing object: %s/%s", object.GetNamespace(), object.GetName())

	var pvc *v1.PersistentVolumeClaim
	var err error

	if ownerRefObj := metav1.GetControllerOf(object); ownerRefObj != nil {
		if ownerRefObj.Kind == "PersistentVolumeClaim" {
			pvc, err = c.pvcLister.PersistentVolumeClaims(object.GetNamespace()).Get(ownerRefObj.Name)
			if err != nil {
				klog.V(3).Infof("ignoring orphaned object '%s' of pvc '%s'", object.GetSelfLink(), ownerRefObj.Name)
				return
			}
		}
	}

	if pvc == nil {
		ownerRefAnno, exists := object.GetAnnotations()[AnnOwnerRef]
		if ok {
			pvc, exists, err = c.pvcFromKey(ownerRefAnno)
			if err != nil {
				runtime.HandleError(errors.Wrapf(err, "error getting PVC %s", ownerRefAnno))
				return
			} else if !exists {
				runtime.HandleError(errors.Errorf("error getting PVC %s from ownerref", ownerRefAnno))
				return
			}
		}
	}

	if pvc == nil {
		klog.V(3).Infof("Object: %s/%s has unexpected owner and no ownerRef annotation", object.GetNamespace(), object.GetName())
		return
	}

	klog.V(3).Infof("Will queue PVC %s/%s in response to %s %s/%s", pvc.Namespace, pvc.Name, verb, object.GetNamespace(), object.GetName())

	if verb == "add" {
		pvcKey, err := cache.MetaNamespaceKeyFunc(pvc)
		if err != nil {
			runtime.HandleError(err)
			return
		}
		c.observePodCreate(pvcKey)
	}

	c.enqueuePVC(pvc)
}

func (c *Controller) enqueuePVC(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

//Run is being called from cdi controllers
func (c *Controller) run(threadiness int, stopCh <-chan struct{}, f func()) error { //*CloneContorler
	defer func() {
		c.queue.ShutDown()
	}()
	klog.V(3).Infoln("Starting cdi controller Run loop")
	if threadiness < 1 {
		return errors.Errorf("expected >0 threads, got %d", threadiness)
	}

	if !cache.WaitForCacheSync(stopCh, c.pvcInformer.HasSynced) {
		return errors.New("Timeout waiting for pvc cache sync")
	}
	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		return errors.New("Timeout waiting for pod cache sync")
	}

	klog.V(3).Infoln("Controller cache has synced")
	for i := 0; i < threadiness; i++ {
		go wait.Until(f, time.Second, stopCh)
	}
	<-stopCh
	return nil
}

// forget the passed-in key for this event and optionally log a message.
func (c *Controller) forgetKey(key interface{}, msg string) bool {
	if len(msg) > 0 {
		klog.V(3).Info(msg)
	}
	c.queue.Forget(key)
	return true
}

// return a pvc pointer based on the passed-in work queue key.
func (c *Controller) pvcFromKey(key string) (*v1.PersistentVolumeClaim, bool, error) {
	obj, exists, err := c.objFromKey(c.pvcInformer, key)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get pvc object from key")
	} else if !exists {
		return nil, false, nil
	}

	pvc, ok := obj.(*v1.PersistentVolumeClaim)
	if !ok {
		return nil, false, errors.New("Object not of type *v1.PersistentVolumeClaim")
	}
	return pvc, true, nil
}

func (c *Controller) objFromKey(informer cache.SharedIndexInformer, key string) (interface{}, bool, error) {
	obj, ok, err := informer.GetIndexer().GetByKey(key)
	if err != nil {
		return nil, false, errors.Wrap(err, "error getting interface obj from store")
	}

	if !ok {
		return nil, false, nil
	}
	return obj, true, nil
}
