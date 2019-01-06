package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	// AnnAPIGroup is the APIGroup for CDI
	AnnAPIGroup = "cdi.kubevirt.io"
	// AnnSource provide a const for our PVC import source annotation
	AnnSource = AnnAPIGroup + "/storage.import.source"
	// AnnEndpoint provides a const for our PVC endpoint annotation
	AnnEndpoint = AnnAPIGroup + "/storage.import.endpoint"
	// AnnSecret provides a const for our PVC secretName annotation
	AnnSecret = AnnAPIGroup + "/storage.import.secretName"
	// AnnContentType provides a const for the PVC content-type
	AnnContentType = AnnAPIGroup + "/storage.contentType"
	// AnnImportPod provides a const for our PVC importPodName annotation
	AnnImportPod = AnnAPIGroup + "/storage.import.importPodName"
	//LabelImportPvc is a pod label used to find the import pod that was created by the relevant PVC
	LabelImportPvc = AnnAPIGroup + "/storage.import.importPvcName"
)

// ImportController represents a CDI Import Controller
type ImportController struct {
	Controller
}

type importPodEnvVar struct {
	ep, secretName, source, contentType, imageSize string
}

// NewImportController sets up an Import Controller, and returns a pointer to
// the newly created Import Controller
func NewImportController(client kubernetes.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	image string,
	pullPolicy string,
	verbose string) *ImportController {
	c := &ImportController{
		Controller: *NewController(client, pvcInformer, podInformer, image, pullPolicy, verbose),
	}
	return c
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
		podEnvVar, err := createImportEnvVar(pvc, ic)
		if err != nil {
			return err
		}

		// all checks passed, let's create the importer pod!
		ic.expectPodCreate(pvcKey)
		pod, err = CreateImporterPod(ic.clientset, ic.image, ic.verbose, ic.pullPolicy, podEnvVar, pvc)
		if err != nil {
			ic.observePodCreate(pvcKey)
			return err
		}
		return nil
	}

	// update pvc with importer pod name and optional cdi label
	anno := map[string]string{}
	if pod != nil {
		anno[AnnImportPod] = string(pod.Name)
		anno[AnnPodPhase] = string(pod.Status.Phase)
		//this is for a case where the import container is failing and the restartPolicy is OnFailure. In such case
		//the pod phase is "Running" although the container state is Waiting. When the container recovers, its state
		//changes back to "Running".
		if pod.Status.ContainerStatuses != nil && pod.Status.ContainerStatuses[0].State.Waiting != nil {
			anno[AnnPodPhase] = string(v1.PodFailed)
		}

		if pod.Status.Phase == "Succeeded" {
			dReq := podDeleteRequest{
				namespace: pod.Namespace,
				podName:   pod.Name,
				podLister: ic.Controller.podLister,
				k8sClient: ic.Controller.clientset,
			}
			// just use defer here so we make sure our pvc updates get written prior to actual deletion
			defer deletePod(dReq)
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
		glog.Errorf("error processing pvc %q: %v", key, err)
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

	//check if AnnEndPoint or AnnSource annotation exists
	if !checkPVC(pvc, AnnEndpoint) && !checkPVC(pvc, AnnSource) {
		return nil
	}
	glog.V(3).Infof("ProcessNextPvcItem: next pvc to process: %s\n", key)
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
