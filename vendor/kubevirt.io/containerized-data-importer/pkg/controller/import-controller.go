package controller

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	// AnnSource provide a const for our PVC import source annotation
	AnnSource = AnnAPIGroup + "/storage.import.source"
	// AnnEndpoint provides a const for our PVC endpoint annotation
	AnnEndpoint = AnnAPIGroup + "/storage.import.endpoint"
	// AnnSecret provides a const for our PVC secretName annotation
	AnnSecret = AnnAPIGroup + "/storage.import.secretName"
	// AnnCertConfigMap is the name of a configmap containing tls certs
	AnnCertConfigMap = AnnAPIGroup + "/storage.import.certConfigMap"
	// AnnContentType provides a const for the PVC content-type
	AnnContentType = AnnAPIGroup + "/storage.contentType"
	// AnnImportPod provides a const for our PVC importPodName annotation
	AnnImportPod = AnnAPIGroup + "/storage.import.importPodName"
	// AnnRequiresScratch provides a const for our PVC requires scratch annotation
	AnnRequiresScratch = AnnAPIGroup + "/storage.import.requiresScratch"

	//LabelImportPvc is a pod label used to find the import pod that was created by the relevant PVC
	LabelImportPvc = AnnAPIGroup + "/storage.import.importPvcName"
	//AnnDefaultStorageClass is the annotation indicating that a storage class is the default one.
	AnnDefaultStorageClass = "storageclass.kubernetes.io/is-default-class"
)

// ImportController represents a CDI Import Controller
type ImportController struct {
	cdiClient clientset.Interface
	Controller
}

type importPodEnvVar struct {
	ep, secretName, source, contentType, imageSize, certConfigMap string
	insecureTLS                                                   bool
}

// NewImportController sets up an Import Controller, and returns a pointer to
// the newly created Import Controller
func NewImportController(client kubernetes.Interface,
	cdiClientSet clientset.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	image string,
	pullPolicy string,
	verbose string) *ImportController {
	c := &ImportController{
		cdiClient:  cdiClientSet,
		Controller: *NewController(client, pvcInformer, podInformer, image, pullPolicy, verbose),
	}
	return c
}

func (ic *ImportController) findScratchPvcFromCache(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	return ic.pvcLister.PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name + "-scratch")
}

func (ic *ImportController) findImportPodFromCache(pvc *v1.PersistentVolumeClaim) (*v1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{LabelImportPvc: pvc.Name}})
	if err != nil {
		return nil, err
	}

	podList, err := ic.podLister.Pods(pvc.Namespace).List(selector)
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

func (ic *ImportController) requiresScratchSpace(pvc *v1.PersistentVolumeClaim) bool {
	scratchRequired := false
	contentType := getContentType(pvc)
	// All archive requires scratch space.
	if contentType == "archive" {
		scratchRequired = true
	} else {
		switch getSource(pvc) {
		case SourceGlance:
			scratchRequired = true
		case SourceRegistry:
			scratchRequired = true
		}
	}
	value, ok := pvc.Annotations[AnnRequiresScratch]
	if ok {
		boolVal, _ := strconv.ParseBool(value)
		scratchRequired = scratchRequired || boolVal
	}
	klog.V(3).Infof("import pvc %s scratch requirement is %v", pvc.Name, scratchRequired)
	return scratchRequired
}

// Create the importer pod based the pvc. The endpoint and optional secret are available to
// the importer pod as env vars. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple importer pods for the same pvc.
func (ic *ImportController) processPvcItem(pvc *v1.PersistentVolumeClaim) error {
	// find import Pod
	pod, err := ic.findImportPodFromCache(pvc)
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
	needsSync := ic.podExpectations.SatisfiedExpectations(pvcKey)

	// make sure not to reprocess a PVC that has already completed successfully,
	// even if the pod no longer exists
	previousPhase, exists := pvc.ObjectMeta.Annotations[AnnPodPhase]
	if exists && (previousPhase == string(v1.PodSucceeded)) {
		needsSync = false
	}

	if pod == nil && needsSync {
		return ic.createImporterPod(pvc, pvcKey)
	}

	// update pvc with importer pod name and optional cdi label
	anno := map[string]string{}
	if pod != nil {
		scratchExitCode := false
		if pod.Status.ContainerStatuses != nil && pod.Status.ContainerStatuses[0].LastTerminationState.Terminated != nil &&
			pod.Status.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode > 0 {
			klog.V(3).Infof("Pod %s termination code: %d\n", pod.Name, pod.Status.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode)
			if pod.Status.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode == common.ScratchSpaceNeededExitCode {
				klog.V(3).Infof("Pod %s requires scratch space, terminating pod, and restarting with scratch space\n", pod.Name)
				scratchExitCode = true
				anno[AnnRequiresScratch] = "true"
			}
		}
		anno[AnnImportPod] = string(pod.Name)
		if !scratchExitCode {
			anno[AnnPodPhase] = string(pod.Status.Phase)
			//this is for a case where the import container is failing and the restartPolicy is OnFailure. In such case
			//the pod phase is "Running" although the container state is Waiting. When the container recovers, its state
			//changes back to "Running". If the pod failed because it didn't have scratch space, don't mark the import failed.
			if pod.Status.Phase != v1.PodPending && pod.Status.ContainerStatuses != nil && pod.Status.ContainerStatuses[0].State.Waiting != nil {
				anno[AnnPodPhase] = string(v1.PodFailed)
			}
		}

		if pod.Status.Phase == v1.PodSucceeded || scratchExitCode {
			dReq := podDeleteRequest{
				namespace: pod.Namespace,
				podName:   pod.Name,
				podLister: ic.Controller.podLister,
				k8sClient: ic.Controller.clientset,
			}
			// just use defer here so we make sure our pvc updates get written prior to actual deletion
			defer deletePod(dReq)
		}
		if pod.Status.Phase == v1.PodPending && ic.requiresScratchSpace(pvc) {
			err = ic.createScratchPvcForPod(pvc, pod)
			if err != nil {
				return err
			}
		}
	}

	var lab map[string]string
	if !checkIfLabelExists(pvc, common.CDILabelKey, common.CDILabelValue) {
		lab = map[string]string{common.CDILabelKey: common.CDILabelValue}
	}

	pvc, err = updatePVC(ic.clientset, pvc, anno, lab)
	if err != nil {
		return errors.WithMessage(err, "could not update pvc %q annotation and/or label")
	}
	return nil
}

func (ic *ImportController) createImporterPod(pvc *v1.PersistentVolumeClaim, pvcKey string) error {
	var scratchPvcName *string
	var err error

	requiresScratch := ic.requiresScratchSpace(pvc)
	if requiresScratch {
		name := pvc.Name + "-scratch"
		scratchPvcName = &name
	}

	podEnvVar, err := createImportEnvVar(ic.clientset, pvc)
	if err != nil {
		return err
	}

	// all checks passed, let's create the importer pod!
	ic.expectPodCreate(pvcKey)
	pod, err := CreateImporterPod(ic.clientset, ic.image, ic.verbose, ic.pullPolicy, podEnvVar, pvc, scratchPvcName)
	if err != nil {
		ic.observePodCreate(pvcKey)
		return err
	}
	if requiresScratch {
		return ic.createScratchPvcForPod(pvc, pod)
	}
	return nil
}

func (ic *ImportController) createScratchPvcForPod(pvc *v1.PersistentVolumeClaim, pod *v1.Pod) error {
	scratchPvc, err := ic.findScratchPvcFromCache(pvc)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if scratchPvc == nil {
		storageClassName := GetScratchPvcStorageClass(ic.clientset, ic.cdiClient, pvc)
		// Scratch PVC doesn't exist yet, create it. Determine which storage class to use.
		scratchPvc, err = CreateScratchPersistentVolumeClaim(ic.clientset, pvc, pod, storageClassName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ic *ImportController) expectPodCreate(pvcKey string) {
	ic.podExpectations.ExpectCreations(pvcKey, 1)
}

//ProcessNextPvcItem ...
func (ic *ImportController) ProcessNextPvcItem() bool {
	key, shutdown := ic.queue.Get()
	if shutdown {
		return false
	}
	defer ic.queue.Done(key)

	err := ic.syncPvc(key.(string))
	if err != nil { // processPvcItem errors may not have been logged so log here
		klog.Errorf("error processing pvc %q: %v", key, err)
		return true
	}
	return ic.forgetKey(key, fmt.Sprintf("ProcessNextPvcItem: processing pvc %q completed", key))
}

func (ic *ImportController) syncPvc(key string) error {
	pvc, exists, err := ic.pvcFromKey(key)
	if err != nil {
		return err
	} else if !exists {
		ic.podExpectations.DeleteExpectations(key)
	}

	if pvc == nil {
		return nil
	}

	if pvc.DeletionTimestamp != nil {
		klog.V(3).Infof("detected PVC delete request for PVC '%s', cleaning up any associated PODS", pvc.Name)
		pod, err := ic.findImportPodFromCache(pvc)
		if err != nil {
			return err
		}
		if pod == nil {
			klog.V(3).Infof("unable to find POD associated with PVC: %s, already deleted maybe?", pvc.Name)
			return nil
		}
		dReq := podDeleteRequest{
			namespace: pod.Namespace,
			podName:   pod.Name,
			podLister: ic.Controller.podLister,
			k8sClient: ic.Controller.clientset,
		}
		err = deletePod(dReq)
		if err != nil && !k8serrors.IsNotFound(err) {
			klog.V(3).Infof("error encountered cleaning up associated PODS for PVC: %v", err)
			return err
		}
	}

	//check if AnnEndPoint or AnnSource annotation exists
	if !checkPVC(pvc, AnnEndpoint) && !checkPVC(pvc, AnnSource) {
		return nil
	}
	klog.V(3).Infof("ProcessNextPvcItem: next pvc to process: %s\n", key)
	return ic.processPvcItem(pvc)
}

//Run is being called from cdi-controller (cmd)
func (ic *ImportController) Run(threadiness int, stopCh <-chan struct{}) error {
	ic.Controller.run(threadiness, stopCh, ic)
	return nil
}

func (ic *ImportController) runPVCWorkers() {
	for ic.ProcessNextPvcItem() {
	}
}
