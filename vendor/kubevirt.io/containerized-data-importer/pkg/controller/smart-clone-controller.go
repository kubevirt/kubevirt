package controller

import (
	"fmt"
	"time"

	"reflect"

	csisnapshotv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"k8s.io/klog"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	cdischeme "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned/scheme"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions/core/v1alpha1"
	listers "kubevirt.io/containerized-data-importer/pkg/client/listers/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/common"
	csiclientset "kubevirt.io/containerized-data-importer/pkg/snapshot-client/clientset/versioned"
	snapshotsinformers "kubevirt.io/containerized-data-importer/pkg/snapshot-client/informers/externalversions/volumesnapshot/v1alpha1"
	snapshotslisters "kubevirt.io/containerized-data-importer/pkg/snapshot-client/listers/volumesnapshot/v1alpha1"
)

const (
	//AnnSmartCloneRequest sets our expected annotation for a CloneRequest
	AnnSmartCloneRequest = "k8s.io/SmartCloneRequest"
)

// SmartCloneController represents the CDI SmartClone Controller
type SmartCloneController struct {
	clientset    kubernetes.Interface
	cdiClientSet clientset.Interface
	csiClientSet csiclientset.Interface

	snapshotInformer  cache.SharedIndexInformer
	snapshotsLister   snapshotslisters.VolumeSnapshotLister
	pvcLister         corelisters.PersistentVolumeClaimLister
	dataVolumeLister  listers.DataVolumeLister
	dataVolumesSynced cache.InformerSynced
	snapshotsSynced   cache.InformerSynced

	queue    workqueue.RateLimitingInterface
	recorder record.EventRecorder
}

// NewSmartCloneController sets up a Smart Clone Controller, and returns a pointer to
// to the newly created Controller
func NewSmartCloneController(client kubernetes.Interface,
	cdiClientSet clientset.Interface,
	csiClientSet csiclientset.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	snapshotInformer snapshotsinformers.VolumeSnapshotInformer,
	dataVolumeInformer informers.DataVolumeInformer) *SmartCloneController {

	// Create event broadcaster
	// Add smart-clone-controller types to the default Kubernetes Scheme so Events can be
	// logged for smart-clone-controller types.
	cdischeme.AddToScheme(scheme.Scheme)
	klog.V(3).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	c := &SmartCloneController{
		clientset:         client,
		cdiClientSet:      cdiClientSet,
		csiClientSet:      csiClientSet,
		pvcLister:         pvcInformer.Lister(),
		snapshotsLister:   snapshotInformer.Lister(),
		snapshotInformer:  snapshotInformer.Informer(),
		dataVolumeLister:  dataVolumeInformer.Lister(),
		dataVolumesSynced: dataVolumeInformer.Informer().HasSynced,
		snapshotsSynced:   snapshotInformer.Informer().HasSynced,
		recorder:          recorder,
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	// Set up an event handler for when VolumeSnapshot resources change
	snapshotInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueItem,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueItem(new)
		},
		DeleteFunc: c.enqueueItem,
	})

	// Set up an event handler for when PVC resources change
	// handleObject function ensures we filter PVCs not created by this controller
	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueItem,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueItem(new)
		},
		DeleteFunc: c.enqueueItem,
	})

	return c
}

//ProcessNextItem ...
func (c *SmartCloneController) ProcessNextItem() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	ns, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		runtime.HandleError(errors.Errorf("invalid resource key: %s", key))
		return false
	}

	pvc, err := c.pvcLister.PersistentVolumeClaims(ns).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			pvc = nil
		} else {
			// Error getting PVC - return
			return false
		}
	}
	snapshot, err := c.snapshotsLister.VolumeSnapshots(ns).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			snapshot = nil
		} else {
			// Error getting Snapshot - return
			return false
		}
	}

	if pvc != nil {
		if pvc.Status.Phase != corev1.ClaimBound {
			// PVC isn't bound yet - return
			return false
		}
		if pvc.ObjectMeta.Annotations[AnnSmartCloneRequest] == "true" {
			snapshotName := pvc.Spec.DataSource.Name
			snapshotToDelete, err := c.snapshotsLister.VolumeSnapshots(ns).Get(snapshotName)
			if err != nil {
				// Error getting Snapshot - return
				return true
			}
			if snapshotToDelete != nil {
				klog.V(3).Infof("ProcessNextItem snapshotName: %s", snapshotName)
				err = c.csiClientSet.SnapshotV1alpha1().VolumeSnapshots(ns).Delete(snapshotName, &metav1.DeleteOptions{})
				if err != nil {
					klog.Errorf("error deleting snapshot for smart-clone %q: %v", key, err)
					return true
				}
				klog.V(3).Infof("Snapshot deleted: %s", snapshotName)

				dataVolume, err := c.dataVolumeLister.DataVolumes(snapshot.Namespace).Get(snapshot.Name)
				if err != nil {
					return true
				}

				// Update DV phase and emit PVC in progress event
				c.updateSmartCloneStatusPhase(cdiv1.Succeeded, dataVolume, pvc)
			}
		}
	} else if snapshot != nil {
		err := c.syncSnapshot(key.(string))
		if err != nil {
			klog.Errorf("error processing snapshot %q: %v", key, err)
			return true
		}
	}
	return c.forgetKey(key, fmt.Sprintf("ProcessNextItem: processing pvc/snapshot %q completed", key))
}

// forget the passed-in key for this event and optionally log a message.
func (c *SmartCloneController) forgetKey(key interface{}, msg string) bool {
	if len(msg) > 0 {
		klog.V(3).Info(msg)
	}
	c.queue.Forget(key)
	return true
}

func (c *SmartCloneController) syncSnapshot(key string) error {
	snapshot, exists, err := c.snapshotFromKey(key)
	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	_, ok := snapshot.Annotations[AnnSmartCloneRequest]
	if !ok {
		//ignoring snapshot, not created by DataVolume Controller
		return nil
	}

	snapshotReadyToUse := snapshot.Status.ReadyToUse
	klog.V(3).Infof("Snapshot \"%s/%s\" - ReadyToUse: %t", snapshot.Namespace, snapshot.Name, snapshotReadyToUse)
	if !snapshotReadyToUse {
		return nil
	}

	return c.processNextSnapshotItem(snapshot)
}

// The snapshot is ReadyToUse, then we can create the PVC and update DV status
func (c *SmartCloneController) processNextSnapshotItem(snapshot *csisnapshotv1.VolumeSnapshot) error {
	dataVolume, err := c.dataVolumeLister.DataVolumes(snapshot.Namespace).Get(snapshot.Name)
	if err != nil {
		return err
	}

	// Update DV phase and emit PVC in progress event
	c.updateSmartCloneStatusPhase(SmartClonePVCInProgress, dataVolume, nil)

	newPvc := newPvcFromSnapshot(snapshot, dataVolume)
	if newPvc == nil {
		klog.Errorf("error creating new pvc from snapshot object")
		return nil
	}

	_, err = c.clientset.CoreV1().PersistentVolumeClaims(snapshot.Namespace).Create(newPvc)
	if err != nil {
		return err
	}

	return nil
}

func (c *SmartCloneController) objFromKey(informer cache.SharedIndexInformer, key interface{}) (interface{}, bool, error) {
	keyString, ok := key.(string)
	if !ok {
		return nil, false, errors.New("keys is not of type string")
	}
	obj, ok, err := informer.GetIndexer().GetByKey(keyString)
	if err != nil {
		return nil, false, errors.Wrap(err, "error getting interface obj from store")
	}
	if !ok {
		return nil, false, nil
	}
	return obj, true, nil
}

// return a VolumeSnapshot pointer based on the passed-in work queue key.
func (c *SmartCloneController) snapshotFromKey(key interface{}) (*csisnapshotv1.VolumeSnapshot, bool, error) {
	obj, exists, err := c.objFromKey(c.snapshotInformer, key)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get pvc object from key")
	} else if !exists {
		return nil, false, nil
	}

	snapshot, ok := obj.(*csisnapshotv1.VolumeSnapshot)
	if !ok {
		return nil, false, errors.New("Object not of type *v1.PersistentVolumeClaim")
	}
	return snapshot, true, nil
}

//Run is being called from cdi-controller (cmd)
func (c *SmartCloneController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.V(2).Info("Starting SmartCloneController controller")

	// Wait for the caches to be synced before starting workers
	klog.V(2).Info("Waiting for informer caches to sync")
	if !cache.WaitForCacheSync(stopCh, c.snapshotsSynced, c.dataVolumesSynced) {
		return errors.New("Timeout waiting for caches sync")
	}
	klog.V(2).Info("Starting worker")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runSnapshotWorkers, time.Second, stopCh)
	}

	klog.V(2).Info("Started workers")
	<-stopCh
	klog.V(2).Info("Shutting down workers")

	return nil
}

func (c *SmartCloneController) runSnapshotWorkers() {
	for c.ProcessNextItem() {
		// empty
	}
}

// enqueueItem takes a VolumeSnapshot or PVC resource and converts it into a namespace/name
// string which is then put onto the work queue.
func (c *SmartCloneController) enqueueItem(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *SmartCloneController) updateSmartCloneStatusPhase(phase cdiv1.DataVolumePhase, dataVolume *cdiv1.DataVolume, newPVC *corev1.PersistentVolumeClaim) error {
	var dataVolumeCopy = dataVolume.DeepCopy()
	var event DataVolumeEvent

	switch phase {
	case cdiv1.SnapshotForSmartCloneInProgress:
		dataVolumeCopy.Status.Phase = cdiv1.SnapshotForSmartCloneInProgress
		event.eventType = corev1.EventTypeNormal
		event.reason = SnapshotForSmartCloneInProgress
		event.message = fmt.Sprintf(MessageSmartCloneInProgress, dataVolumeCopy.Spec.Source.PVC.Namespace, dataVolumeCopy.Spec.Source.PVC.Name)
	case cdiv1.SmartClonePVCInProgress:
		dataVolumeCopy.Status.Phase = cdiv1.SmartClonePVCInProgress
		event.eventType = corev1.EventTypeNormal
		event.reason = SmartClonePVCInProgress
		event.message = fmt.Sprintf(MessageSmartClonePVCInProgress, dataVolumeCopy.Spec.Source.PVC.Namespace, dataVolumeCopy.Spec.Source.PVC.Name)
	case cdiv1.Succeeded:
		dataVolumeCopy.Status.Phase = cdiv1.Succeeded
		event.eventType = corev1.EventTypeNormal
		event.reason = CloneSucceeded
		event.message = fmt.Sprintf(MessageCloneSucceeded, dataVolumeCopy.Spec.Source.PVC.Namespace, dataVolumeCopy.Spec.Source.PVC.Name, newPVC.Namespace, newPVC.Name)
	}

	return c.emitEvent(dataVolume, dataVolumeCopy, &event)
}

func (c *SmartCloneController) emitEvent(dataVolume *cdiv1.DataVolume, dataVolumeCopy *cdiv1.DataVolume, event *DataVolumeEvent) error {
	// Only update the object if something actually changed in the status.
	if !reflect.DeepEqual(dataVolume.Status, dataVolumeCopy.Status) {
		_, err := c.cdiClientSet.CdiV1alpha1().DataVolumes(dataVolume.Namespace).Update(dataVolumeCopy)
		// Emit the event only when the status change happens, not every time
		if event.eventType != "" {
			c.recorder.Event(dataVolume, event.eventType, event.reason, event.message)
		}
		return err
	}
	return nil
}

func newPvcFromSnapshot(snapshot *csisnapshotv1.VolumeSnapshot, dataVolume *cdiv1.DataVolume) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"cdi-controller":         snapshot.Name,
		common.CDILabelKey:       common.CDILabelValue,
		common.CDIComponentLabel: common.SmartClonerCDILabel,
	}
	ownerRef := metav1.GetControllerOf(snapshot)
	if ownerRef == nil {
		return nil
	}
	annotations := make(map[string]string)
	annotations[AnnSmartCloneRequest] = "true"
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            snapshot.Name,
			Namespace:       snapshot.Namespace,
			Labels:          labels,
			Annotations:     annotations,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			DataSource: &corev1.TypedLocalObjectReference{
				Name:     snapshot.Name,
				Kind:     "VolumeSnapshot",
				APIGroup: &csisnapshotv1.SchemeGroupVersion.Group,
			},
			VolumeMode:       dataVolume.Spec.PVC.VolumeMode,
			AccessModes:      dataVolume.Spec.PVC.AccessModes,
			StorageClassName: dataVolume.Spec.PVC.StorageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: dataVolume.Spec.PVC.Resources.Requests,
			},
		},
	}
}
