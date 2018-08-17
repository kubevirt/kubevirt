package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	. "kubevirt.io/containerized-data-importer/pkg/common"
	expectations "kubevirt.io/containerized-data-importer/pkg/expectations"
)

const (
	// pvc annotations
	AnnEndpoint  = "cdi.kubevirt.io/storage.import.endpoint"
	AnnSecret    = "cdi.kubevirt.io/storage.import.secretName"
	AnnImportPod = "cdi.kubevirt.io/storage.import.importPodName"
	// importer pod annotations
	AnnCreatedBy   = "cdi.kubevirt.io/storage.createdByController"
	AnnPodPhase    = "cdi.kubevirt.io/storage.import.pod.phase"
	LabelImportPvc = "cdi.kubevirt.io/storage.import.importPvcName"
)

type ImportController struct {
	clientset                kubernetes.Interface
	queue                    workqueue.RateLimitingInterface
	pvcInformer, podInformer cache.SharedIndexInformer
	pvcLister                corelisters.PersistentVolumeClaimLister
	podLister                corelisters.PodLister
	pvcsSynced               cache.InformerSynced
	podsSynced               cache.InformerSynced
	importerImage            string
	pullPolicy               string // Options: IfNotPresent, Always, or Never
	verbose                  string // verbose levels: 1, 2, ...
	podExpectations          *expectations.UIDTrackingControllerExpectations
}

func NewImportController(client kubernetes.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	importerImage string,
	pullPolicy string,
	verbose string) *ImportController {
	c := &ImportController{
		clientset:       client,
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pvcInformer:     pvcInformer.Informer(),
		podInformer:     podInformer.Informer(),
		pvcLister:       pvcInformer.Lister(),
		podLister:       podInformer.Lister(),
		pvcsSynced:      pvcInformer.Informer().HasSynced,
		podsSynced:      podInformer.Informer().HasSynced,
		importerImage:   importerImage,
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

func (c *ImportController) handlePodAdd(obj interface{}) {
	c.handlePodObject(obj, "add")
}
func (c *ImportController) handlePodUpdate(obj interface{}) {
	c.handlePodObject(obj, "update")
}
func (c *ImportController) handlePodDelete(obj interface{}) {
	c.handlePodObject(obj, "delete")
}

func (c *ImportController) expectPodCreate(pvcKey string) {
	c.podExpectations.ExpectCreations(pvcKey, 1)
}
func (c *ImportController) observePodCreate(pvcKey string) {
	c.podExpectations.CreationObserved(pvcKey)
}

func (c *ImportController) handlePodObject(obj interface{}, verb string) {
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
		glog.V(Vdebug).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(Vdebug).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		_, createdByUs := object.GetAnnotations()[AnnCreatedBy]

		if ownerRef.Kind != "PersistentVolumeClaim" {
			return
		} else if !createdByUs {
			return
		}

		pvc, err := c.pvcLister.PersistentVolumeClaims(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(Vdebug).Infof("ignoring orphaned object '%s' of pvc '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		if verb == "add" {
			pvcKey, err := cache.MetaNamespaceKeyFunc(pvc)
			if err != nil {
				runtime.HandleError(err)
				return
			}

			c.observePodCreate(pvcKey)
		}
		c.enqueuePVC(pvc)
		return
	}
}

func (c *ImportController) enqueuePVC(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *ImportController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer func() {
		c.queue.ShutDown()
	}()
	glog.V(Vadmin).Infoln("Starting cdi controller Run loop")
	if threadiness < 1 {
		return errors.Errorf("expected >0 threads, got %d", threadiness)
	}

	if !cache.WaitForCacheSync(stopCh, c.pvcInformer.HasSynced) {
		return errors.New("Timeout waiting for pvc cache sync")
	}
	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		return errors.New("Timeout waiting for pod cache sync")
	}
	glog.V(Vdebug).Infoln("ImportController cache has synced")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runPVCWorkers, time.Second, stopCh)
	}
	<-stopCh
	return nil
}

func (c *ImportController) runPVCWorkers() {
	for c.ProcessNextPvcItem() {
	}
}

func (c *ImportController) syncPvc(key string) error {
	pvc, err := c.pvcFromKey(key)
	if err != nil {
		return err
	}
	if pvc == nil {
		return nil
	}
	// filter pvc and decide if the importer pod should be created
	if !checkPVC(pvc) {
		return nil
	}
	glog.V(Vdebug).Infof("ProcessNextPvcItem: next pvc to process: %s\n", key)
	return c.processPvcItem(pvc)
}

// Select only pvcs with the importer endpoint annotation and that are not being processed.
// We forget the key unless `processPvcItem` returns an error in which case the key can be
// retried.
func (c *ImportController) ProcessNextPvcItem() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncPvc(key.(string))
	if err != nil { // processPvcItem errors may not have been logged so log here
		glog.Errorf("error processing pvc %q: %v", key, err)
		return true
	}
	return c.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: processing pvc %q completed", key))
}

func (c *ImportController) findImportPodFromCache(pvc *v1.PersistentVolumeClaim) (*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{LabelImportPvc: pvc.Name}})
	if err != nil {
		return nil, err
	}

	podList, err := c.podLister.Pods(pvc.Namespace).List(selector)
	if err != nil {
		return nil, err
	}

	if len(podList) == 0 {
		return nil, nil
	} else if len(podList) > 1 {
		return nil, errors.Errorf("multiple pods found for import PVC %s/%s", pvc.Namespace, pvc.Name)
	}
	return podList[0], nil
}

// Create the importer pod based the pvc. The endpoint and optional secret are available to
// the importer pod as env vars. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple importer pods for the same pvc.
func (c *ImportController) processPvcItem(pvc *v1.PersistentVolumeClaim) error {

	// find import Pod
	pod, err := c.findImportPodFromCache(pvc)
	if err != nil {
		return err
	}
	pvcKey, err := cache.MetaNamespaceKeyFunc(pvc)
	if err != nil {
		return err
	}

	// Pod must be controlled by this PVC
	if pod != nil && !metav1.IsControlledBy(pod, pvc) {
		return errors.Errorf("found pod %s/%s not owned by pvc %s/%s", pod.Namespace, pod.Name, pvc.Namespace, pvc.Name)
	}

	// expectations prevent us from creating multiple pods. An expectation forces
	// us to observe a pod's creation in the cache.
	needsSync := c.podExpectations.SatisfiedExpectations(pvcKey)

	// make sure not to reprocess a PVC that has already completed successfully,
	// even if the pod no longer exists
	previousPhase, exists := pvc.ObjectMeta.Annotations[AnnPodPhase]
	if exists && (previousPhase == string(v1.PodSucceeded)) {
		needsSync = false
	}

	if pod == nil && needsSync {

		ep, err := getEndpoint(pvc)
		if err != nil {
			return err
		}

		secretName, err := getSecretName(c.clientset, pvc)
		if err != nil {
			return err
		}
		if secretName == "" {
			glog.V(Vadmin).Infof("no secret will be supplied to endpoint %q\n", ep)
		}

		// all checks passed, let's create the importer pod!
		c.expectPodCreate(pvcKey)
		pod, err = CreateImporterPod(c.clientset, c.importerImage, c.verbose, c.pullPolicy, ep, secretName, pvc)
		if err != nil {
			c.observePodCreate(pvcKey)
			return err
		}
		return nil
	}

	// update pvc with importer pod name and optional cdi label
	anno := map[string]string{}
	if pod != nil {
		anno[AnnImportPod] = string(pod.Name)
		anno[AnnPodPhase] = string(pod.Status.Phase)
	}

	var lab map[string]string
	if !checkIfLabelExists(pvc, CDI_LABEL_KEY, CDI_LABEL_VALUE) {
		lab = map[string]string{CDI_LABEL_KEY: CDI_LABEL_VALUE}
	}

	pvc, err = updatePVC(c.clientset, pvc, anno, lab)
	if err != nil {
		return errors.WithMessage(err, "could not update pvc %q annotation and/or label")
	}
	return nil
}

// forget the passed-in key for this event and optionally log a message.
func (c *ImportController) forgetKey(key interface{}, msg string) bool {
	if len(msg) > 0 {
		glog.V(Vdebug).Info(msg)
	}
	c.queue.Forget(key)
	return true
}
