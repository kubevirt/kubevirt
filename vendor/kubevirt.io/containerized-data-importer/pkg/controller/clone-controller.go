package controller

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	storageV1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	//AnnCloneRequest sets our expected annotation for a CloneRequest
	AnnCloneRequest = "k8s.io/CloneRequest"
	//AnnCloneOf is used to indicate that cloning was complete
	AnnCloneOf = "k8s.io/CloneOf"
	//CloneUniqueID is used as a special label to be used when we search for the pod
	CloneUniqueID = "cdi.kubevirt.io/storage.clone.cloneUniqeId"
	//AnnTargetPodNamespace is being used as a pod label to find the related target PVC
	AnnTargetPodNamespace = "cdi.kubevirt.io/storage.clone.targetPod.namespace"
)

// CloneController represents the CDI Clone Controller
type CloneController struct {
	Controller
}

// NewCloneController sets up a Clone Controller, and returns a pointer to
// to the newly created Controller
func NewCloneController(client kubernetes.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	image string,
	pullPolicy string,
	verbose string) *CloneController {
	c := &CloneController{
		Controller: *NewController(client, pvcInformer, podInformer, image, pullPolicy, verbose),
	}
	return c
}

func (cc *CloneController) findClonePodsFromCache(pvc *v1.PersistentVolumeClaim) (*v1.Pod, *v1.Pod, error) {
	var sourcePod, targetPod *v1.Pod
	annCloneRequest := pvc.GetAnnotations()[AnnCloneRequest]
	if annCloneRequest != "" {
		sourcePvcNamespace, _ := ParseSourcePvcAnnotation(annCloneRequest, "/")
		if sourcePvcNamespace == "" {
			return nil, nil, errors.Errorf("Bad CloneRequest Annotation")
		}
		//find the source pod
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{CloneUniqueID: pvc.Name + "-source-pod"}})
		if err != nil {
			return nil, nil, err
		}
		podList, err := cc.podLister.Pods(sourcePvcNamespace).List(selector)
		if err != nil {
			return nil, nil, err
		}
		if len(podList) == 0 {
			return nil, nil, nil
		} else if len(podList) > 1 {
			return nil, nil, errors.Errorf("multiple source pods found for clone PVC %s/%s", pvc.Namespace, pvc.Name)
		}
		sourcePod = podList[0]
		//find target pod
		selector, err = metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{CloneUniqueID: pvc.Name + "-target-pod"}})
		if err != nil {
			return nil, nil, err
		}
		podList, err = cc.podLister.Pods(pvc.Namespace).List(selector)
		if err != nil {
			return nil, nil, err
		}
		if len(podList) == 0 {
			return nil, nil, nil
		} else if len(podList) > 1 {
			return nil, nil, errors.Errorf("multiple target pods found for clone PVC %s/%s", pvc.Namespace, pvc.Name)
		}
		targetPod = podList[0]
	}
	return sourcePod, targetPod, nil
}

// Create the cloning source and target pods based the pvc. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple pods for the same pvc.
func (cc *CloneController) processPvcItem(pvc *v1.PersistentVolumeClaim) error {
	anno := map[string]string{}

	// find cloning source and target Pods
	sourcePod, targetPod, err := cc.findClonePodsFromCache(pvc)
	if err != nil {
		return err
	}
	pvcKey, err := cache.MetaNamespaceKeyFunc(pvc)
	if err != nil {
		return err
	}

	// Pods must be controlled by this PVC
	if sourcePod != nil && !metav1.IsControlledBy(sourcePod, pvc) {
		return errors.Errorf("found pod %s/%s not owned by pvc %s/%s", sourcePod.Namespace, sourcePod.Name, pvc.Namespace, pvc.Name)
	}
	if targetPod != nil && !metav1.IsControlledBy(sourcePod, pvc) {
		return errors.Errorf("found pod %s/%s not owned by pvc %s/%s", targetPod.Namespace, targetPod.Name, pvc.Namespace, pvc.Name)
	}

	// expectations prevent us from creating multiple pods. An expectation forces
	// us to observe a pod's creation in the cache.
	needsSync := cc.podExpectations.SatisfiedExpectations(pvcKey)

	// make sure not to reprocess a PVC that has already completed successfully,
	// even if the pod no longer exists
	phase, exists := pvc.ObjectMeta.Annotations[AnnPodPhase]
	if exists && (phase == string(v1.PodSucceeded)) {
		needsSync = false
	}

	if needsSync && (sourcePod == nil || targetPod == nil) {
		err := cc.initializeExpectations(pvcKey)
		if err != nil {
			return err
		}
		//create random string to be used for pod labeling and hostpath name
		if sourcePod == nil {
			cr, err := getCloneRequestPVC(pvc)
			if err != nil {
				return err
			}
			// all checks passed, let's create the cloner pods!
			cc.raisePodCreate(pvcKey)
			//create the source pod
			sourcePod, err = CreateCloneSourcePod(cc.clientset, cc.image, cc.pullPolicy, cr, pvc)
			if err != nil {
				cc.observePodCreate(pvcKey)
				return err
			}
		}
		if targetPod == nil {
			cc.raisePodCreate(pvcKey)
			//create the target pod
			targetPod, err = CreateCloneTargetPod(cc.clientset, cc.image, cc.pullPolicy, pvc, sourcePod.ObjectMeta.Namespace)
			if err != nil {
				cc.observePodCreate(pvcKey)
				return err
			}
		}
		return nil
	}

	// update pvc with cloner pod name and optional cdi label
	//we update the target PVC according to the target pod. Only the target pods indicates the real status of the cloning.
	anno[AnnPodPhase] = string(targetPod.Status.Phase)
	//add the following annotation only if the pod pahse is succeeded, meaning job is completed
	if phase == string(v1.PodSucceeded) {
		anno[AnnCloneOf] = "true"
		defer cc.deleteClonePods(sourcePod.Namespace, sourcePod.Name, targetPod.Name)
	}
	var lab map[string]string
	if !checkIfLabelExists(pvc, common.CDILabelKey, common.CDILabelValue) {
		lab = map[string]string{common.CDILabelKey: common.CDILabelValue}
	}
	pvc, err = updatePVC(cc.clientset, pvc, anno, lab)
	if err != nil {
		return errors.WithMessage(err, "could not update pvc %q annotation and/or label")
	}
	return nil
}

func (cc *CloneController) deleteClonePods(namespace, srcName, tgtName string) {
	srcReq := podDeleteRequest{
		namespace: namespace,
		podName:   srcName,
		podLister: cc.Controller.podLister,
		k8sClient: cc.Controller.clientset,
	}
	tgtReq := podDeleteRequest{
		namespace: namespace,
		podName:   tgtName,
		podLister: cc.Controller.podLister,
		k8sClient: cc.Controller.clientset,
	}
	deletePod(srcReq)
	deletePod(tgtReq)
}

func (c *Controller) initializeExpectations(pvcKey string) error {
	return c.podExpectations.SetExpectations(pvcKey, 0, 0)
}

func (c *Controller) raisePodCreate(pvcKey string) {
	c.podExpectations.RaiseExpectations(pvcKey, 1, 0)
}

// Select only pvcs with the 'CloneRequest' annotation and that are not being processed.
// We forget the key unless `processPvcItem` returns an error in which case the key can be
//ProcessNextPvcItem retried.

//ProcessNextPvcItem ...
func (cc *CloneController) ProcessNextPvcItem() bool {
	key, shutdown := cc.queue.Get()
	if shutdown {
		return false
	}
	defer cc.queue.Done(key)

	err := cc.syncPvc(key.(string))
	if err != nil { // processPvcItem errors may not have been logged so log here
		glog.Errorf("error processing pvc %q: %v", key, err)
		return true
	}
	return cc.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: processing pvc %q completed", key))
}

func (cc *CloneController) syncPvc(key string) error {
	pvc, exists, err := cc.pvcFromKey(key)
	if err != nil {
		return err
	} else if !exists {
		cc.podExpectations.DeleteExpectations(key)
	}

	if pvc == nil {
		return nil
	}
	//check if AnnoCloneRequest annotation exists
	if !checkPVC(pvc, AnnCloneRequest) {
		return nil
	}

	pvcPhase := pvc.Status.Phase
	glog.V(3).Infof("PVC phase for PVC \"%s/%s\" is %s", pvc.Namespace, pvc.Name, pvcPhase)
	if pvc.Spec.StorageClassName != nil {
		storageClassName := *pvc.Spec.StorageClassName
		glog.V(3).Infof("storageClassName used by PVC \"%s/%s\" is \"%s\"", pvc.Namespace, pvc.Name, storageClassName)
		storageclass, err := cc.clientset.StorageV1().StorageClasses().Get(storageClassName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		//Do not schedule the clone pods unless the target PVC is either Bound or Pending/WaitFirstConsumer.
		if !(pvcPhase == v1.ClaimBound || (pvcPhase == v1.ClaimPending && *storageclass.VolumeBindingMode == storageV1.VolumeBindingWaitForFirstConsumer)) {
			glog.V(3).Infof("PVC \"%s/%s\" is either not bound or is in pending phase and VolumeBindingMode is not VolumeBindingWaitForFirstConsumer."+
				" Ignoring this PVC.", pvc.Namespace, pvc.Name)
			glog.V(3).Infof("PVC phase is %s", pvcPhase)
			glog.V(3).Infof("VolumeBindingMode is %s", *storageclass.VolumeBindingMode)
			return nil
		}
	}

	//checking for CloneOf annotation indicating that the clone was already taken care of by the provisioner (smart clone).
	if metav1.HasAnnotation(pvc.ObjectMeta, AnnCloneOf) {
		glog.V(3).Infof("pvc annotation %q exists indicating cloning completed, skipping pvc \"%s/%s\"\n", AnnCloneOf, pvc.Namespace, pvc.Name)
		return nil
	}
	glog.V(3).Infof("ProcessNextPvcItem: next pvc to process: \"%s/%s\"\n", pvc.Namespace, pvc.Name)
	return cc.processPvcItem(pvc)
}

//Run is being called from cdi-controller (cmd)
func (cc *CloneController) Run(threadiness int, stopCh <-chan struct{}) error {
	cc.Controller.run(threadiness, stopCh, cc)
	return nil
}

func (cc *CloneController) runPVCWorkers() {
	for cc.ProcessNextPvcItem() {
	}
}
