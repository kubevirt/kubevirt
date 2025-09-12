package disruptionbudget

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/util/pdbs"
)

const deleteNotifFail = "Failed to process delete notification"

const (
	// FailedDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget failed.
	FailedDeletePodDisruptionBudgetReason = "FailedDelete"
	// SuccessfulDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget succeeded.
	SuccessfulDeletePodDisruptionBudgetReason = "SuccessfulDelete"
)

type DisruptionBudgetController struct {
	clientset                       kubernetes.Interface
	Queue                           workqueue.TypedRateLimitingInterface[string]
	vmiStore                        cache.Store
	pdbIndexer                      cache.Indexer
	recorder                        record.EventRecorder
	podDisruptionBudgetExpectations *controller.UIDTrackingControllerExpectations
	hasSynced                       func() bool
}

func NewDisruptionBudgetController(
	vmiInformer cache.SharedIndexInformer,
	pdbInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubernetes.Interface,
) (*DisruptionBudgetController, error) {

	c := &DisruptionBudgetController{
		Queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-disruption-budget"},
		),
		vmiStore:                        vmiInformer.GetStore(),
		pdbIndexer:                      pdbInformer.GetIndexer(),
		recorder:                        recorder,
		clientset:                       clientset,
		podDisruptionBudgetExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.hasSynced = func() bool {
		return vmiInformer.HasSynced() && pdbInformer.HasSynced() && podInformer.HasSynced() && migrationInformer.HasSynced()
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = pdbInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPodDisruptionBudget,
		DeleteFunc: c.deletePodDisruptionBudget,
		UpdateFunc: c.updatePodDisruptionBudget,
	})
	if err != nil {
		return nil, err
	}

	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updatePod,
	})
	if err != nil {
		return nil, err
	}

	_, err = migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateMigration,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *DisruptionBudgetController) updateMigration(_, curr interface{}) {
	vmim := curr.(*virtv1.VirtualMachineInstanceMigration)

	if vmim.DeletionTimestamp != nil {
		return
	}

	vmi := &virtv1.VirtualMachineInstance{
		ObjectMeta: v1.ObjectMeta{
			Namespace: vmim.GetNamespace(),
			Name:      vmim.Spec.VMIName,
		},
	}
	c.enqueueVirtualMachine(vmi)
}

func (c *DisruptionBudgetController) updatePod(_, curr interface{}) {
	pod := curr.(*corev1.Pod)

	if pod.DeletionTimestamp != nil {
		return
	}

	controllerRef := v1.GetControllerOf(pod)
	if controllerRef == nil {
		return
	}
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	c.enqueueVirtualMachine(vmi)
}

func (c *DisruptionBudgetController) addVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *DisruptionBudgetController) deleteVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *DisruptionBudgetController) updateVirtualMachineInstance(_, curr interface{}) {
	c.enqueueVMI(curr)
}

func (c *DisruptionBudgetController) enqueueVMI(obj interface{}) {
	logger := log.Log
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a pdb in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pdb
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(deleteNotifFail)
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pdb %#v", obj)).Error(deleteNotifFail)
			return
		}
	}
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		logger.Object(vmi).Reason(err).Error("Failed to extract key from vmi.")
	}
	c.Queue.Add(key)
}

// When a pdb is created, enqueue the vmi that manages it and update its pdbExpectations.
func (c *DisruptionBudgetController) addPodDisruptionBudget(obj interface{}) {
	pdb := obj.(*policyv1.PodDisruptionBudget)

	if pdb.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pdb shows up in a state that
		// is already pending deletion. Prevent the pdb from being a creation observation.
		c.deletePodDisruptionBudget(pdb)
		return
	}

	controllerRef := v1.GetControllerOf(pdb)
	vmi := c.resolveControllerRef(pdb.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	vmiKey, err := controller.KeyFunc(vmi)
	if err != nil {
		return
	}
	log.Log.V(4).Object(pdb).Infof("PodDisruptionBudget created")
	c.podDisruptionBudgetExpectations.CreationObserved(vmiKey)
	c.enqueueVirtualMachine(vmi)
}

// When a pdb is updated, figure out what vmi/s manage it and wake them
// up. If the labels of the pdb have changed we need to awaken both the old
// and new vmi. old and cur must be *v1.PodDisruptionBudget types.
func (c *DisruptionBudgetController) updatePodDisruptionBudget(old, cur interface{}) {
	curPodDisruptionBudget := cur.(*policyv1.PodDisruptionBudget)
	oldPodDisruptionBudget := old.(*policyv1.PodDisruptionBudget)
	if curPodDisruptionBudget.ResourceVersion == oldPodDisruptionBudget.ResourceVersion {
		// Periodic resync will send update events for all known pdbs.
		// Two different versions of the same pdb will always have different RVs.
		return
	}

	if curPodDisruptionBudget.DeletionTimestamp != nil {
		labelChanged := !equality.Semantic.DeepEqual(curPodDisruptionBudget.Labels, oldPodDisruptionBudget.Labels)
		// having a pdb marked for deletion is enough to count as a deletion expectation
		c.deletePodDisruptionBudget(curPodDisruptionBudget)
		if labelChanged {
			// we don't need to check the oldPodDisruptionBudget.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePodDisruptionBudget(oldPodDisruptionBudget)
		}
		return
	}

	curControllerRef := v1.GetControllerOf(curPodDisruptionBudget)
	oldControllerRef := v1.GetControllerOf(oldPodDisruptionBudget)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vmi := c.resolveControllerRef(oldPodDisruptionBudget.Namespace, oldControllerRef); vmi != nil {
			c.enqueueVirtualMachine(vmi)
		}
	}

	vmi := c.resolveControllerRef(curPodDisruptionBudget.Namespace, curControllerRef)
	if vmi == nil {
		return
	}
	log.Log.V(4).Object(curPodDisruptionBudget).Infof("PodDisruptionBudget updated")
	c.enqueueVirtualMachine(vmi)
	return
}

// When a pdb is deleted, enqueue the vmi that manages the pdb and update its pdbExpectations.
// obj could be an *v1.PodDisruptionBudget, or a DeletionFinalStateUnknown marker item.
func (c *DisruptionBudgetController) deletePodDisruptionBudget(obj interface{}) {
	pdb, ok := obj.(*policyv1.PodDisruptionBudget)

	// When a delete is dropped, the relist will notice a pdb in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pdb
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(deleteNotifFail)
			return
		}
		pdb, ok = tombstone.Obj.(*policyv1.PodDisruptionBudget)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pdb %#v", obj)).Error(deleteNotifFail)
			return
		}
	}

	controllerRef := v1.GetControllerOf(pdb)
	vmi := c.resolveControllerRef(pdb.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	vmiKey, err := controller.KeyFunc(vmi)
	if err != nil {
		return
	}
	key, err := controller.KeyFunc(pdb)
	if err != nil {
		return
	}
	c.podDisruptionBudgetExpectations.DeletionObserved(vmiKey, key)
	c.enqueueVirtualMachine(vmi)
}

func (c *DisruptionBudgetController) enqueueVirtualMachine(obj interface{}) {
	logger := log.Log
	vmi := obj.(*virtv1.VirtualMachineInstance)
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		logger.Object(vmi).Reason(err).Error("Failed to extract key from virtualmachineinstance.")
		return
	}
	c.Queue.Add(key)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *DisruptionBudgetController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineInstance {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it is nil or the wrong Kind.
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineInstanceGroupVersionKind.Kind {
		return nil
	}

	return &virtv1.VirtualMachineInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      controllerRef.Name,
			Namespace: namespace,
			UID:       controllerRef.UID,
		},
	}
}

// Run runs the passed in NodeController.
func (c *DisruptionBudgetController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting disruption budget controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping disruption budget controller.")
}

func (c *DisruptionBudgetController) runWorker() {
	for c.Execute() {
	}
}

func (c *DisruptionBudgetController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *DisruptionBudgetController) execute(key string) error {

	if !c.podDisruptionBudgetExpectations.SatisfiedExpectations(key) {
		return nil
	}

	// Fetch the latest Vm state from cache
	obj, vmiExists, err := c.vmiStore.GetByKey(key)

	if err != nil {
		return err
	}

	var vmi *virtv1.VirtualMachineInstance
	// Once all finalizers are removed the vmi gets deleted and we can clean all expectations
	if vmiExists {
		vmi = obj.(*virtv1.VirtualMachineInstance)
	} else {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			log.DefaultLogger().Reason(err).Error("Could not extract namespace and name from the controller key.")
			return err
		}
		vmi = libvmi.New(libvmi.WithName(name), libvmi.WithNamespace(namespace))
	}

	// Only consider pdbs which belong to this vmi
	pdbs, err := pdbs.PDBsForVMI(vmi, c.pdbIndexer)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to fetch pod disruption budgets for namespace from cache.")
		// If the situation does not change there is no benefit in retrying
		return nil
	}

	if len(pdbs) == 0 {
		return nil
	}

	for i := range pdbs {
		if syncErr := c.deletePDB(key, pdbs[i], vmi); syncErr != nil {
			err = syncErr
		}
	}
	return err
}

func (c *DisruptionBudgetController) deletePDB(key string, pdb *policyv1.PodDisruptionBudget, vmi *virtv1.VirtualMachineInstance) error {
	if pdb != nil && pdb.DeletionTimestamp == nil {
		pdbKey, err := cache.MetaNamespaceKeyFunc(pdb)
		if err != nil {
			return err
		}
		c.podDisruptionBudgetExpectations.ExpectDeletions(key, []string{pdbKey})
		err = c.clientset.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Delete(context.Background(), pdb.Name, v1.DeleteOptions{})
		if err != nil {
			c.podDisruptionBudgetExpectations.DeletionObserved(key, pdbKey)
			c.recorder.Eventf(vmi, corev1.EventTypeWarning, FailedDeletePodDisruptionBudgetReason, "Error deleting the PodDisruptionBudget %s: %v", pdb.Name, err)
			return err
		}
		c.recorder.Eventf(vmi, corev1.EventTypeNormal, SuccessfulDeletePodDisruptionBudgetReason, "Deleted PodDisruptionBudget %s", pdb.Name)
	}
	return nil
}
