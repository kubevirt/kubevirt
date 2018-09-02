package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	. "kubevirt.io/containerized-data-importer/pkg/common"
	"time"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	// pvc annotations
	AnnCloneRequest = "k8s.io/CloneRequest"
	AnnCloneOf      = "k8s.io/CloneOf"
	AnnCloningPods  = "cdi.kubevirt.io/storage.clone.cloningPods"
	// importer pod annotations
	AnnCloningCreatedBy = "cdi.kubevirt.io/storage.cloningCreatedByController"
)

type CloneController struct {
	clientset                kubernetes.Interface
	pvcQueue, podQueue       workqueue.RateLimitingInterface
	pvcInformer, podInformer cache.SharedIndexInformer
	cloneImage               string
	pullPolicy               string // Options: IfNotPresent, Always, or Never
	verbose                  string // verbose levels: 1, 2, ...
}

func NewCloneController(client kubernetes.Interface, pvcInformer, podInformer cache.SharedIndexInformer, cloneImage string, pullPolicy string, verbose string) *CloneController {
	c := &CloneController{
		clientset:   client,
		pvcQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		pvcInformer: pvcInformer,
		podInformer: podInformer,
		cloneImage:  cloneImage,
		pullPolicy:  pullPolicy,
		verbose:     verbose,
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

func (c *CloneController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer func() {
		c.pvcQueue.ShutDown()
		c.podQueue.ShutDown()
	}()
	glog.V(Vadmin).Infoln("Starting clone controller Run loop")
	if threadiness < 1 {
		return errors.Errorf("expected >0 threads, got %d", threadiness)
	}

	if !cache.WaitForCacheSync(stopCh, c.pvcInformer.HasSynced) {
		return errors.New("Timeout waiting for pvc cache sync")
	}
	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		return errors.New("Timeout waiting for pod cache sync")
	}
	glog.V(Vdebug).Infoln("CloneController cache has synced")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runPVCWorkers, time.Second, stopCh)
		go wait.Until(c.runPodWorkers, time.Second, stopCh)
	}
	<-stopCh
	return nil
}

func (c *CloneController) runPodWorkers() {
	for c.ProcessNextPodItem() {
	}
}

func (c *CloneController) runPVCWorkers() {
	for c.ProcessNextPvcItem() {
	}
}

// ProcessNextPodItem gets the next pod key from the queue and verifies that it was created by the
// controller. If not the key is discarded; otherwise, the pod object is passed to processPodItem.
// Note: pods are already filtered by label "host-assisted-cloning".
func (c *CloneController) ProcessNextPodItem() bool {
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
	if !metav1.HasAnnotation(pod.ObjectMeta, AnnCloningCreatedBy) {
		c.forgetKey(key, fmt.Sprintf("ProcessNextPodItem: pod %q does not have annotation %q", key, AnnCloningCreatedBy))
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
// of the pod in the PVC to indicate the status of the cloning process.
func (c *CloneController) processPodItem(pod *v1.Pod) error {
	// verify that this pod has the expected pvc name
	var pvcKey string
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == ImagePathName {
			pvcKey = fmt.Sprintf("%s/%s", pod.Namespace, vol.PersistentVolumeClaim.ClaimName)
			glog.V(Vadmin).Infof("pod \"%s/%s\" has volume matching claim %q\n", pod.Namespace, pod.Name, pvcKey)
			break
		}
	}
	if len(pvcKey) == 0 {
		// If this block is ever reached, something has gone very wrong.  The pod should ALWAYS be created with the volume.
		// A missing volume would most likely indicate a pod that has been created manually, but also incorrectly defined.
		return errors.Errorf("Pod does not contain volume %q", ImagePathName)
	}

	glog.V(Vdebug).Infof("processPodItem: getting pvc for key %q", pvcKey)
	pvc, err := c.pvcFromKey(pvcKey)
	if err != nil {
		return errors.WithMessage(err, "could not retrieve pvc from cache")
	}
	// see if pvc's pod phase anno needs to be added/updated. The update is done only on the target PVC
	phase := string(pod.Status.Phase)
	_, exists := pvc.ObjectMeta.Annotations[AnnCloneRequest]
	if !checkIfAnnoExists(pvc, AnnPodPhase, phase) && exists {
		pvc, err = setPVCAnnotation(c.clientset, pvc, AnnPodPhase, phase)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("could not set annotation \"%s: %s\" on pvc %q", AnnPodPhase, phase, pvc.Name))

		}
		glog.V(Vdebug).Infof("processPodItem: pod phase %q annotated in pvc %q", pod.Status.Phase, pvcKey)
		if phase == "Succeeded" {
			pvc, err = setPVCAnnotation(c.clientset, pvc, AnnCloneOf, "true")
			if err != nil {
				return errors.WithMessage(err, fmt.Sprintf("could not set annotation \"%s: %s\" on pvc %q", AnnCloneOf, "true", pvc.Name))
			}
			glog.V(Vdebug).Infof("processPodItem: CloneOf annotatated in pvc %q", pvcKey)
		}
	}
	return nil
}

// Select only pvcs with the 'CloneRequest' annotation and that are not being processed.
// We forget the key unless `processPvcItem` returns an error in which case the key can be
// retried.
func (c *CloneController) ProcessNextPvcItem() bool {
	key, shutdown := c.pvcQueue.Get()
	if shutdown {
		return false
	}
	defer c.pvcQueue.Done(key)

	pvc, err := c.pvcFromKey(key)
	if err != nil || pvc == nil {
		return c.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: error converting key %v to pvc: %v", key, err))
	}
	// filter pvc and decide if the source and target pods should be created
	if continue_processing, _, _ := checkClonePVC(c.clientset, pvc, false); !continue_processing {
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

// Create the cloning source and target pods based the pvc. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple pods for the same pvc.
func (c *CloneController) processPvcItem(pvc *v1.PersistentVolumeClaim) error {
	cr, err := getCloneRequestPVC(pvc)
	if err != nil {
		return err
	}

	// check our existing pvc one more time to ensure we should be working on it
	// and to help mitigate any race conditions. This time we get the latest pvc.
	doCreate, pvc, err := checkClonePVC(c.clientset, pvc, true)
	if err != nil { // maybe an intermittent api error
		return err
	}
	if !doCreate { // don't create pods but not an error
		return nil // forget key; logging already done
	}

	// update pvc with 'AnnCloningPods' annotation to indicate cloning is in process
	anno := map[string]string{AnnCloningPods: "exist"}
	var lab map[string]string
	pvc, err = updatePVC(c.clientset, pvc, anno, lab)
	if err != nil {
		return errors.WithMessage(err, "could not update pvc %q annotation and/or label")
	}

	//create random string to be used for pod labeling and hostpath name
	generatedLabelStr := util.RandAlphaNum(GENERATED_CLONING_LABEL_LEN)
	//create the source pod
	pod, err := CreateCloneSourcePod(c.clientset, c.cloneImage, c.verbose, c.pullPolicy, cr, pvc, generatedLabelStr)
	if err != nil {
		//TODO: remove annotation AnnCloningPods from pvc as pod failed to run
		return err
	}

	//create the target pod
	_, err = CreateCloneTargetPod(c.clientset, c.cloneImage, c.verbose, c.pullPolicy, pvc, generatedLabelStr, pod.ObjectMeta.Namespace)
	if err != nil {
		//TODO: remove annotation AnnCloningPods from pvc as pod failed to run
		return err
	}

	return nil
}

// forget the passed-in key for this event and optionally log a message.
func (c *CloneController) forgetKey(key interface{}, msg string) bool {
	if len(msg) > 0 {
		glog.V(Vdebug).Info(msg)
	}
	c.pvcQueue.Forget(key)
	return true
}
