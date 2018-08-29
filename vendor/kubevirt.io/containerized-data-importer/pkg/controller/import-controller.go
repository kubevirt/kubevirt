package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	. "kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	// pvc annotations
	AnnEndpoint  = "cdi.kubevirt.io/storage.import.endpoint"
	AnnSecret    = "cdi.kubevirt.io/storage.import.secretName"
	AnnImportPod = "cdi.kubevirt.io/storage.import.importPodName"
	// importer pod annotations
	AnnCreatedBy = "cdi.kubevirt.io/storage.createdByController"
	AnnPodPhase  = "cdi.kubevirt.io/storage.import.pod.phase"
)

type ImportController struct {
	clientset                kubernetes.Interface
	pvcQueue, podQueue       workqueue.RateLimitingInterface
	pvcInformer, podInformer cache.SharedIndexInformer
	importerImage            string
	pullPolicy               string // Options: IfNotPresent, Always, or Never
	verbose                  string // verbose levels: 1, 2, ...
}

func NewImportController(client kubernetes.Interface, pvcInformer, podInformer cache.SharedIndexInformer, importerImage string, pullPolicy string, verbose string) *ImportController {
	c := &ImportController{
		clientset:     client,
		pvcQueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pvcInformer:   pvcInformer,
		podInformer:   podInformer,
		importerImage: importerImage,
		pullPolicy:    pullPolicy,
		verbose:       verbose,
	}

	// Bind the pvc SharedIndexInformer to the pvc queue
	c.pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.pvcQueue.AddRateLimited(key)
			}
		},
		// this is triggered by an update or it will also be
		// be triggered periodically even if no changes were made.
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.pvcQueue.AddRateLimited(key)
			}
		},
	})

	// Bind the pod SharedIndexInformer to the pod queue
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.podQueue.AddRateLimited(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				c.podQueue.AddRateLimited(key)
			}
		},
	})

	return c
}

func (c *ImportController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer func() {
		c.pvcQueue.ShutDown()
		c.podQueue.ShutDown()
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
		go wait.Until(c.runPodWorkers, time.Second, stopCh)
	}
	<-stopCh
	return nil
}

func (c *ImportController) runPodWorkers() {
	for c.ProcessNextPodItem() {
	}
}

func (c *ImportController) runPVCWorkers() {
	for c.ProcessNextPvcItem() {
	}
}

// ProcessNextPodItem gets the next pod key from the queue and verifies that it was created by the
// controller. If not the key is discarded; otherwise, the pod object is passed to processPodItem.
// Note: pods are already filtered by label "app=containerized-data-importer".
func (c *ImportController) ProcessNextPodItem() bool {
	key, shutdown := c.podQueue.Get()
	if shutdown {
		return false
	}
	defer c.podQueue.Done(key)

	pod, err := c.podFromKey(key)
	if err != nil {
		c.forgetKey(key, fmt.Sprintf("ProcessNextPodItem: unable to get pod from key %v: %v", key, err))
		return true
	}
	if !metav1.HasAnnotation(pod.ObjectMeta, AnnCreatedBy) {
		c.forgetKey(key, fmt.Sprintf("ProcessNextPodItem: pod %q does not have annotation %q", key, AnnCreatedBy))
		return true
	}
	glog.V(Vdebug).Infof("ProcessNextPodItem: next pod to process: %s\n", key)
	err = c.processPodItem(pod)
	if err != nil { // processPodItem errors may not have been logged so log here
		glog.Errorf("error processing pod %q: %v", key, err)
		return true
	}
	return c.forgetKey(key, fmt.Sprintf("ProcessNextPodItem: processing pod %q completed", key))
}

// processPodItem verifies that the passed in pod is genuine and, if so, annotates the Phase
// of the pod in the PVC to indicate the status of the import process.
func (c *ImportController) processPodItem(pod *v1.Pod) error {
	// verify that this pod has the expected pvc name
	var pvcKey string
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == DataVolName {
			pvcKey = fmt.Sprintf("%s/%s", pod.Namespace, vol.PersistentVolumeClaim.ClaimName)
			glog.V(Vadmin).Infof("pod \"%s/%s\" has volume matching claim %q\n", pod.Namespace, pod.Name, pvcKey)
			break
		}
	}
	if len(pvcKey) == 0 {
		// If this block is ever reached, something has gone very wrong.  The pod should ALWAYS be created with the volume.
		// A missing volume would most likely indicate a pod that has been created manually, but also incorrectly defined.
		return errors.Errorf("Pod does not contain volume %q", DataVolName)
	}

	glog.V(Vdebug).Infof("processPodItem: getting pvc for key %q", pvcKey)
	pvc, err := c.pvcFromKey(pvcKey)
	if err != nil {
		return errors.WithMessage(err, "could not retrieve pvc from cache")
	}
	// see if pvc's importer pod phase anno needs to be added/updated
	phase := string(pod.Status.Phase)
	if !checkIfAnnoExists(pvc, AnnPodPhase, phase) {
		pvc, err = setPVCAnnotation(c.clientset, pvc, AnnPodPhase, phase)
		if err != nil {
			glog.V(Vdebug).Infof("processPodItem: pod phase %q annotated in pvc %q", pod.Status.Phase, pvcKey)
			return errors.WithMessage(err, fmt.Sprintf("could not set annotation \"%s: %s\" on pvc %q", AnnPodPhase, phase, pvc.Name))
		}
	}
	return nil
}

// Select only pvcs with the importer endpoint annotation and that are not being processed.
// We forget the key unless `processPvcItem` returns an error in which case the key can be
// retried.
func (c *ImportController) ProcessNextPvcItem() bool {
	key, shutdown := c.pvcQueue.Get()
	if shutdown {
		return false
	}
	defer c.pvcQueue.Done(key)

	pvc, err := c.pvcFromKey(key)
	if err != nil || pvc == nil {
		return c.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: error converting key %v to pvc: %v", key, err))
	}
	// filter pvc and decide if the importer pod should be created
	if continue_processing, _, _ := checkPVC(c.clientset, pvc, false); !continue_processing {
		return c.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: skipping pvc %q\n", key))
	}
	glog.V(Vdebug).Infof("ProcessNextPvcItem: next pvc to process: %s\n", key)
	err = c.processPvcItem(pvc)
	if err != nil { // processPvcItem errors may not have been logged so log here
		glog.Errorf("error processing pvc %q: %v", key, err)
		return true
	}
	return c.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: processing pvc %q completed", key))
}

// Create the importer pod based the pvc. The endpoint and optional secret are available to
// the importer pod as env vars. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple importer pods for the same pvc.
func (c *ImportController) processPvcItem(pvc *v1.PersistentVolumeClaim) error {
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

	// check our existing pvc one more time to ensure we should be working on it
	// and to help mitigate any race conditions. This time we get the latest pvc.
	doCreate, pvc, err := checkPVC(c.clientset, pvc, true)
	if err != nil { // maybe an intermittent api error
		return err
	}
	if !doCreate { // don't create importer pod but not an error
		return nil // forget key; logging already done
	}

	// all checks passed, let's create the importer pod!
	pod, err := CreateImporterPod(c.clientset, c.importerImage, c.verbose, c.pullPolicy, ep, secretName, pvc)
	if err != nil {
		return err
	}
	// update pvc with importer pod name and optional cdi label
	anno := map[string]string{AnnImportPod: pod.Name}
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
	c.pvcQueue.Forget(key)
	return true
}
