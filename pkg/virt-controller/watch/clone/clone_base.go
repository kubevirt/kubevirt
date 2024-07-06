package clone

import (
	"fmt"
	"time"

	k8scorev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/snapshot"
	"kubevirt.io/kubevirt/pkg/util/status"
)

type Event string

const (
	defaultVerbosityLevel = 2
	unknownTypeErrFmt     = "clone controller expected object of type %s but found object of unknown type"

	SnapshotCreated       Event = "SnapshotCreated"
	SnapshotReady         Event = "SnapshotReady"
	RestoreCreated        Event = "RestoreCreated"
	RestoreCreationFailed Event = "RestoreCreationFailed"
	RestoreReady          Event = "RestoreReady"
	TargetVMCreated       Event = "TargetVMCreated"
	PVCBound              Event = "PVCBound"

	SnapshotDeleted    Event = "SnapshotDeleted"
	SourceDoesNotExist Event = "SourceDoesNotExist"
)

type VMCloneController struct {
	client                  kubecli.KubevirtClient
	vmCloneInformer         cache.SharedIndexInformer
	snapshotInformer        cache.SharedIndexInformer
	restoreInformer         cache.SharedIndexInformer
	vmInformer              cache.SharedIndexInformer
	snapshotContentInformer cache.SharedIndexInformer
	pvcInformer             cache.SharedIndexInformer
	recorder                record.EventRecorder

	vmCloneQueue       workqueue.RateLimitingInterface
	vmStatusUpdater    *status.VMStatusUpdater
	cloneStatusUpdater *status.CloneStatusUpdater
}

func NewVmCloneController(client kubecli.KubevirtClient, vmCloneInformer, snapshotInformer, restoreInformer, vmInformer, snapshotContentInformer, pvcInformer cache.SharedIndexInformer, recorder record.EventRecorder) (*VMCloneController, error) {
	ctrl := VMCloneController{
		client:                  client,
		vmCloneInformer:         vmCloneInformer,
		snapshotInformer:        snapshotInformer,
		restoreInformer:         restoreInformer,
		vmInformer:              vmInformer,
		snapshotContentInformer: snapshotContentInformer,
		pvcInformer:             pvcInformer,
		recorder:                recorder,
		vmCloneQueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-vmclone"),
		vmStatusUpdater:         status.NewVMStatusUpdater(client),
		cloneStatusUpdater:      status.NewCloneStatusUpdater(client),
	}

	_, err := ctrl.vmCloneInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMClone,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMClone(newObj) },
			DeleteFunc: ctrl.handleVMClone,
		},
	)

	if err != nil {
		return nil, err
	}

	_, err = ctrl.snapshotInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleSnapshot(newObj) },
			DeleteFunc: ctrl.handleSnapshot,
		},
	)

	if err != nil {
		return nil, err
	}

	_, err = ctrl.restoreInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleRestore,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleRestore(newObj) },
			DeleteFunc: ctrl.handleRestore,
		},
	)

	if err != nil {
		return nil, err
	}

	_, err = ctrl.pvcInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
		},
	)

	if err != nil {
		return nil, err
	}

	_, err = ctrl.vmInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: ctrl.handleDeleteVM,
		},
	)

	if err != nil {
		return nil, err
	}
	return &ctrl, nil
}

func (ctrl *VMCloneController) handleVMClone(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	vmClone, ok := obj.(*clonev1alpha1.VirtualMachineClone)
	if !ok {
		log.Log.Errorf(unknownTypeErrFmt, clone.ResourceVMCloneSingular)
		return
	}

	objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmClone)
	if err != nil {
		log.Log.Errorf("vm clone controller failed to get key from object: %v, %v", err, vmClone)
		return
	}

	log.Log.V(defaultVerbosityLevel).Infof("enqueued %q for sync", objName)
	ctrl.vmCloneQueue.Add(objName)
}

func (ctrl *VMCloneController) handleSnapshot(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
	if !ok {
		log.Log.Errorf(unknownTypeErrFmt, "virtualmachinesnapshot")
		return
	}

	if ownedByClone, key := isOwnedByClone(snapshot); ownedByClone {
		ctrl.vmCloneQueue.AddRateLimited(key)
	}

	snapshotKey, err := cache.MetaNamespaceKeyFunc(snapshot)
	if err != nil {
		log.Log.Object(snapshot).Reason(err).Error("cannot get snapshot key")
		return
	}

	snapshotSourceKeys, err := ctrl.vmCloneInformer.GetIndexer().IndexKeys("snapshotSource", snapshotKey)
	if err != nil {
		log.Log.Object(snapshot).Reason(err).Error("cannot get clone snapshotSourceKeys from snapshotSource indexer")
		return
	}

	snapshotWaitingKeys, err := ctrl.vmCloneInformer.GetIndexer().IndexKeys(string(clonev1alpha1.SnapshotInProgress), snapshotKey)
	if err != nil {
		log.Log.Object(snapshot).Reason(err).Error("cannot get clone snapshotWaitingKeys from " + string(clonev1alpha1.SnapshotInProgress) + " indexer")
		return
	}

	for _, key := range append(snapshotSourceKeys, snapshotWaitingKeys...) {
		ctrl.vmCloneQueue.AddRateLimited(key)
	}
}

func (ctrl *VMCloneController) handleRestore(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	restore, ok := obj.(*snapshotv1.VirtualMachineRestore)
	if !ok {
		log.Log.Errorf(unknownTypeErrFmt, "virtualmachinerestore")
		return
	}

	if ownedByClone, key := isOwnedByClone(restore); ownedByClone {
		ctrl.vmCloneQueue.AddRateLimited(key)
	}

	restoreKey, err := cache.MetaNamespaceKeyFunc(restore)
	if err != nil {
		log.Log.Object(restore).Reason(err).Error("cannot get snapshot key")
		return
	}

	restoreWaitingKeys, err := ctrl.vmCloneInformer.GetIndexer().IndexKeys(string(clonev1alpha1.RestoreInProgress), restoreKey)
	if err != nil {
		log.Log.Object(restore).Reason(err).Error("cannot get clone restoreWaitingKeys from " + string(clonev1alpha1.RestoreInProgress) + " indexer")
		return
	}

	for _, key := range restoreWaitingKeys {
		ctrl.vmCloneQueue.AddRateLimited(key)
	}
}

func (ctrl *VMCloneController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	pvc, ok := obj.(*k8scorev1.PersistentVolumeClaim)
	if !ok {
		log.Log.Errorf(unknownTypeErrFmt, "persistentvolumeclaim")
		return
	}

	var (
		restoreName string
		exists      bool
	)

	if restoreName, exists = pvc.Annotations[snapshot.RestoreNameAnnotation]; !exists {
		return
	}

	if pvc.Status.Phase != k8scorev1.ClaimBound {
		return
	}

	restoreKey := getKey(restoreName, pvc.Namespace)

	succeededWaitingKeys, err := ctrl.vmCloneInformer.GetIndexer().IndexKeys(string(clonev1alpha1.Succeeded), restoreKey)
	if err != nil {
		log.Log.Object(pvc).Reason(err).Error("cannot get clone succeededWaitingKeys from " + string(clonev1alpha1.Succeeded) + " indexer")
		return
	}

	for _, key := range succeededWaitingKeys {
		ctrl.vmCloneQueue.AddRateLimited(key)
	}
}

func (ctrl *VMCloneController) handleDeleteVM(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)
	// When a delete is dropped, the relist will notice a vm in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vm, ok = tombstone.Obj.(*virtv1.VirtualMachine)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vm %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	vmCloneList, err := ctrl.listVmCloneMatchingVM(vm.Namespace, vm.Name)
	if err != nil {
		log.Log.Errorf("error retrieving vm clone list: %v", err.Error())
		return
	}
	log.Log.V(4).Object(vm).Infof("vm clone lis: %v", vmCloneList)
	for _, vmClone := range vmCloneList {
		log.Log.V(4).Object(vm).Infof("vm deleted for vm clone %s", vmClone.Name)
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmClone)
		if err != nil {
			log.Log.Errorf("vm clone controller failed to get key from object: %v, %v", err, vmClone)
			return
		}
		ctrl.vmCloneQueue.AddRateLimited(objName)
	}

	return
}

func (ctrl *VMCloneController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmCloneQueue.ShutDown()

	log.Log.Info("Starting clone controller")
	defer log.Log.Info("Shutting down clone controller")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.vmCloneInformer.HasSynced,
		ctrl.snapshotInformer.HasSynced,
		ctrl.restoreInformer.HasSynced,
		ctrl.vmInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.runWorker, time.Second, stopCh)
	}

	<-stopCh
	return nil
}

func (ctrl *VMCloneController) Execute() bool {
	key, quit := ctrl.vmCloneQueue.Get()
	if quit {
		return false
	}
	defer ctrl.vmCloneQueue.Done(key)

	err := ctrl.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing clone %v", key)
		ctrl.vmCloneQueue.AddRateLimited(key)
	} else {
		log.Log.V(defaultVerbosityLevel).Infof("processed clone %v", key)
		ctrl.vmCloneQueue.Forget(key)
	}
	return true
}

func (ctrl *VMCloneController) runWorker() {
	for ctrl.Execute() {
	}
}

// takes a namespace and returns all vm clone with the specified target vm name
func (ctrl *VMCloneController) listVmCloneMatchingVM(namespace, name string) ([]*clonev1alpha1.VirtualMachineClone, error) {
	return ctrl.filterVmClone(namespace, func(vmClone *clonev1alpha1.VirtualMachineClone) bool {
		return vmClone.Spec.Target.Name == name
	})
}

func (ctrl *VMCloneController) filterVmClone(namespace string, filter func(*clonev1alpha1.VirtualMachineClone) bool) ([]*clonev1alpha1.VirtualMachineClone, error) {
	objs, err := ctrl.vmCloneInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	var vmClones []*clonev1alpha1.VirtualMachineClone
	for _, obj := range objs {
		vmClone := obj.(*clonev1alpha1.VirtualMachineClone)

		if filter(vmClone) {
			vmClones = append(vmClones, vmClone)
		}
	}
	return vmClones, nil
}
