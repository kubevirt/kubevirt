/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	cdischeme "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned/scheme"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions/datavolumecontroller/v1alpha1"
	listers "kubevirt.io/containerized-data-importer/pkg/client/listers/datavolumecontroller/v1alpha1"
)

const controllerAgentName = "datavolume-controller"

const (
	SuccessSynced         = "Synced"
	ErrResourceExists     = "ErrResourceExists"
	MessageResourceExists = "Resource %q already exists and is not managed by DataVolume"
	MessageResourceSynced = "DataVolume synced successfully"
)

type DataVolumeController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// clientset is a clientset for our own API group
	cdiClientSet clientset.Interface

	pvcLister  corelisters.PersistentVolumeClaimLister
	pvcsSynced cache.InformerSynced

	dataVolumesLister listers.DataVolumeLister
	dataVolumesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

func NewDataVolumeController(
	kubeclientset kubernetes.Interface,
	cdiClientSet clientset.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	dataVolumeInformer informers.DataVolumeInformer) *DataVolumeController {

	// Create event broadcaster
	// Add datavolume-controller types to the default Kubernetes Scheme so Events can be
	// logged for datavolume-controller types.
	cdischeme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &DataVolumeController{
		kubeclientset:     kubeclientset,
		cdiClientSet:      cdiClientSet,
		pvcLister:         pvcInformer.Lister(),
		pvcsSynced:        pvcInformer.Informer().HasSynced,
		dataVolumesLister: dataVolumeInformer.Lister(),
		dataVolumesSynced: dataVolumeInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DataVolumes"),
		recorder:          recorder,
	}

	glog.Info("Setting up event handlers")

	// Set up an event handler for when DataVolume resources change
	dataVolumeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueDataVolume,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueDataVolume(new)
		},
	})
	// Set up an event handler for when PVC resources change
	// handleObject function ensures we filter PVCs not created by this controller
	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*corev1.PersistentVolumeClaim)
			oldDepl := old.(*corev1.PersistentVolumeClaim)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known PVCs.
				// Two different versions of the same PVCs will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *DataVolumeController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting DataVolume controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.pvcsSynced, c.dataVolumesSynced); !ok {
		return errors.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process DataVolume resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *DataVolumeController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *DataVolumeController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(errors.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// DataVolume resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return errors.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DataVolume resource
// with the current status of the resource.
func (c *DataVolumeController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(errors.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the DataVolume resource with this namespace/name
	dataVolume, err := c.dataVolumesLister.DataVolumes(namespace).Get(name)
	if err != nil {
		// The DataVolume resource may no longer exist, in which case we stop
		// processing.
		if k8serrors.IsNotFound(err) {
			runtime.HandleError(errors.Errorf("dataVolume '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Get the pvc with the name specified in DataVolume.spec
	pvc, err := c.pvcLister.PersistentVolumeClaims(dataVolume.Namespace).Get(dataVolume.Name)

	// If the resource doesn't exist, we'll create it
	if k8serrors.IsNotFound(err) {
		newPvc, err := newPersistentVolumeClaim(dataVolume)
		if err != nil {
			return err
		}
		pvc, err = c.kubeclientset.CoreV1().PersistentVolumeClaims(dataVolume.Namespace).Create(newPvc)
		if err != nil {
			return err
		}
	}

	// If the PVC is not controlled by this DataVolume resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(pvc, dataVolume) {
		msg := fmt.Sprintf(MessageResourceExists, pvc.Name)
		c.recorder.Event(dataVolume, corev1.EventTypeWarning, ErrResourceExists, msg)
		return errors.Errorf(msg)
	}

	// Finally, we update the status block of the DataVolume resource to reflect the
	// current state of the world
	err = c.updateDataVolumeStatus(dataVolume, pvc)
	if err != nil {
		return err
	}

	c.recorder.Event(dataVolume, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *DataVolumeController) updateDataVolumeStatus(dataVolume *cdiv1.DataVolume, pvc *corev1.PersistentVolumeClaim) error {
	dataVolumeCopy := dataVolume.DeepCopy()
	var err error

	switch pvc.Status.Phase {
	case corev1.ClaimPending:
		dataVolumeCopy.Status.Phase = cdiv1.Pending
	case corev1.ClaimBound:
		switch dataVolumeCopy.Status.Phase {
		case cdiv1.Pending:
			dataVolumeCopy.Status.Phase = cdiv1.PVCBound
		case cdiv1.Unknown:
			dataVolumeCopy.Status.Phase = cdiv1.PVCBound
		}

		_, ok := pvc.Annotations[AnnImportPod]
		if ok {
			dataVolumeCopy.Status.Phase = cdiv1.ImportScheduled
		}

		podPhase, ok := pvc.Annotations[AnnPodPhase]
		if ok {
			switch podPhase {
			case string(corev1.PodPending):
				dataVolumeCopy.Status.Phase = cdiv1.ImportScheduled
			case string(corev1.PodRunning):
				dataVolumeCopy.Status.Phase = cdiv1.ImportInProgress
			case string(corev1.PodFailed):
				dataVolumeCopy.Status.Phase = cdiv1.Failed
			case string(corev1.PodSucceeded):
				dataVolumeCopy.Status.Phase = cdiv1.Succeeded
			}
		}
	case corev1.ClaimLost:
		dataVolumeCopy.Status.Phase = cdiv1.Failed
	default:
		if pvc.Status.Phase != "" {
			dataVolumeCopy.Status.Phase = cdiv1.Unknown
		}
	}

	// Only update the object if something actually changed in the status.
	if !reflect.DeepEqual(dataVolume.Status, dataVolumeCopy.Status) {
		_, err = c.cdiClientSet.CdiV1alpha1().DataVolumes(dataVolume.Namespace).Update(dataVolumeCopy)
	}
	return err
}

// enqueueDataVolume takes a DataVolume resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DataVolume.
func (c *DataVolumeController) enqueueDataVolume(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the DataVolume resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that DataVolume resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *DataVolumeController) handleObject(obj interface{}) {
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
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a DataVolume, we should not do anything more
		// with it.
		if ownerRef.Kind != "DataVolume" {
			return
		}

		dataVolume, err := c.dataVolumesLister.DataVolumes(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of dataVolume '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueDataVolume(dataVolume)
		return
	}
}

// newPersistentVolumeClaim creates a new PVC the DataVolume resource.
// It also sets the appropriate OwnerReferences on the resource
// which allows handleObject to discover the DataVolume resource
// that 'owns' it.
func newPersistentVolumeClaim(dataVolume *cdiv1.DataVolume) (*corev1.PersistentVolumeClaim, error) {
	labels := map[string]string{
		"cdi-controller": dataVolume.Name,
		"app":            "containerized-data-importer",
	}

	if dataVolume.Spec.PVC == nil {
		// TODO remove this requirement and dynamically generate
		// PVC spec if not present on DataVolume
		return nil, errors.Errorf("datavolume.pvc field is required")
	}

	annotations := make(map[string]string)

	if dataVolume.Spec.Source.HTTP != nil {
		annotations[AnnEndpoint] = dataVolume.Spec.Source.HTTP.URL
	} else if dataVolume.Spec.Source.S3 != nil {
		annotations[AnnEndpoint] = dataVolume.Spec.Source.S3.URL
		if dataVolume.Spec.Source.S3.SecretRef != "" {
			annotations[AnnSecret] = dataVolume.Spec.Source.S3.SecretRef
		}
	} else {
		return nil, errors.Errorf("no source set for datavolume")
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dataVolume.Name,
			Namespace:   dataVolume.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dataVolume, schema.GroupVersionKind{
					Group:   cdiv1.SchemeGroupVersion.Group,
					Version: cdiv1.SchemeGroupVersion.Version,
					Kind:    "DataVolume",
				}),
			},
		},
		Spec: *dataVolume.Spec.PVC,
	}, nil
}
