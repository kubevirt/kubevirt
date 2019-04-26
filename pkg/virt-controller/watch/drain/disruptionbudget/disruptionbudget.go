package disruptionbudget

import (
	"fmt"
	"reflect"
	"time"

	v12 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	// FailedCreatePodDisruptionBudgetReason is added in an event if creating a PodDisruptionBudget failed.
	FailedCreatePodDisruptionBudgetReason = "FailedCreate"
	// SuccessfulCreatePodDisruptionBudgetReason is added in an event if creating a PodDisruptionBudget succeeded.
	SuccessfulCreatePodDisruptionBudgetReason = "SuccessfulCreate"
	// FailedDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget failed.
	FailedDeletePodDisruptionBudgetReason = "FailedDelete"
	// SuccessfulDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget succeeded.
	SuccessfulDeletePodDisruptionBudgetReason = "SuccessfulDelete"
)

type DisruptionBudgetController struct {
	clientset                       kubecli.KubevirtClient
	Queue                           workqueue.RateLimitingInterface
	vmiInformer                     cache.SharedIndexInformer
	pdbInformer                     cache.SharedIndexInformer
	recorder                        record.EventRecorder
	podDisruptionBudgetExpectations *controller.UIDTrackingControllerExpectations
}

func NewDisruptionBudgetController(
	vmiInformer cache.SharedIndexInformer,
	pdbInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
) *DisruptionBudgetController {

	c := &DisruptionBudgetController{
		Queue:                           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:                     vmiInformer,
		pdbInformer:                     pdbInformer,
		recorder:                        recorder,
		clientset:                       clientset,
		podDisruptionBudgetExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})

	c.pdbInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPodDisruptionBudget,
		DeleteFunc: c.deletePodDisruptionBudget,
		UpdateFunc: c.updatePodDisruptionBudget,
	})

	return c
}

func (c *DisruptionBudgetController) addVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *DisruptionBudgetController) deleteVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *DisruptionBudgetController) updateVirtualMachineInstance(old, curr interface{}) {
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
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pdb %#v", obj)).Error("Failed to process delete notification")
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
	pdb := obj.(*v1beta1.PodDisruptionBudget)

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
	curPodDisruptionBudget := cur.(*v1beta1.PodDisruptionBudget)
	oldPodDisruptionBudget := old.(*v1beta1.PodDisruptionBudget)
	if curPodDisruptionBudget.ResourceVersion == oldPodDisruptionBudget.ResourceVersion {
		// Periodic resync will send update events for all known pdbs.
		// Two different versions of the same pdb will always have different RVs.
		return
	}

	if curPodDisruptionBudget.DeletionTimestamp != nil {
		labelChanged := !reflect.DeepEqual(curPodDisruptionBudget.Labels, oldPodDisruptionBudget.Labels)
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
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
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
	pdb, ok := obj.(*v1beta1.PodDisruptionBudget)

	// When a delete is dropped, the relist will notice a pdb in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pdb
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		pdb, ok = tombstone.Obj.(*v1beta1.PodDisruptionBudget)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pdb %#v", obj)).Error("Failed to process delete notification")
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
	vmi, exists, err := c.vmiInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	return vmi.(*virtv1.VirtualMachineInstance)
}

// Run runs the passed in NodeController.
func (c *DisruptionBudgetController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting disruption budget controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.pdbInformer.HasSynced, c.vmiInformer.HasSynced)

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
	err := c.execute(key.(string))

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
	obj, exists, err := c.vmiInformer.GetStore().GetByKey(key)

	if err != nil {
		return err
	}

	var vmi *virtv1.VirtualMachineInstance
	// Once all finalizers are removed the vmi gets deleted and we can clean all expectations
	if exists {
		vmi = obj.(*virtv1.VirtualMachineInstance)
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Could not extract namespace and name from the controlller key.")
		// If the situation does not change there is no benefit in retrying
		return nil
	}

	// Only consider pdbs which belong to this vmi
	pdb, err := c.pdbForVMI(namespace, name)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to fetch pod disruption budgets for namespace from cache.")
		// If the situation does not change there is no benefit in retrying
		return nil
	}

	return c.sync(key, vmi, pdb)

}

func (c *DisruptionBudgetController) sync(key string, vmi *virtv1.VirtualMachineInstance, pdb *v1beta1.PodDisruptionBudget) error {
	delete := false
	create := false
	// delete if there is no VMI
	if vmi == nil && pdb != nil {
		delete = true
	} else if vmi != nil {
		wantsToMigrate := wantsToMigrateOnDrain(vmi)
		if vmi.DeletionTimestamp != nil && pdb != nil {
			// pdb can already be deleted, shutdown already in process
			delete = true
		} else if !wantsToMigrate && pdb != nil {
			// We don't want migrations on evictions, if there is a pdb, remove it
			delete = true
		} else if wantsToMigrate && pdb == nil {
			// No pdb and we want migrations on evictions
			create = true
		} else if wantsToMigrate && pdb != nil {
			if ownerRef := v1.GetControllerOf(pdb); ownerRef != nil && ownerRef.UID != vmi.UID {
				// The pdb is from an old vmi with a different uid, delete and later create the correct one
				// The VMI always has a minimum grace period, so normally this should not happen, therefore no optimizations
				delete = true
			}
		}
	}

	if delete && pdb != nil && pdb.DeletionTimestamp == nil {
		pdbKey, err := cache.MetaNamespaceKeyFunc(pdb)
		if err != nil {
			return err
		}
		c.podDisruptionBudgetExpectations.ExpectDeletions(key, []string{pdbKey})
		err = c.clientset.PolicyV1beta1().PodDisruptionBudgets(pdb.Namespace).Delete(pdb.Name, &v1.DeleteOptions{})
		if err != nil {
			c.podDisruptionBudgetExpectations.DeletionObserved(key, pdbKey)
			c.recorder.Eventf(vmi, v12.EventTypeWarning, FailedDeletePodDisruptionBudgetReason, "Error deleting the PodDisruptionBudget %s: %v", pdb.Name, err)
			return err
		}
		c.recorder.Eventf(vmi, v12.EventTypeNormal, SuccessfulDeletePodDisruptionBudgetReason, "Deleted PodDisruptionBudget %s", pdb.Name)
		return nil
	} else if create {
		two := intstr.FromInt(2)
		c.podDisruptionBudgetExpectations.ExpectCreations(key, 1)
		createdPDB, err := c.clientset.PolicyV1beta1().PodDisruptionBudgets(vmi.Namespace).Create(&v1beta1.PodDisruptionBudget{
			ObjectMeta: v1.ObjectMeta{
				OwnerReferences: []v1.OwnerReference{
					*v1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind),
				},
				GenerateName: "kubevirt-disruption-budget-",
			},
			Spec: v1beta1.PodDisruptionBudgetSpec{
				MinAvailable: &two,
				Selector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						virtv1.CreatedByLabel: string(vmi.UID),
					},
				},
			},
		})
		if err != nil {
			c.podDisruptionBudgetExpectations.CreationObserved(key)
			c.recorder.Eventf(vmi, v12.EventTypeWarning, FailedCreatePodDisruptionBudgetReason, "Error creating a PodDisruptionBudget: %v", err)
			return err
		}
		c.recorder.Eventf(vmi, v12.EventTypeNormal, SuccessfulCreatePodDisruptionBudgetReason, "Created PodDisruptionBudget %s", createdPDB.Name)
	}
	return nil
}

func (c *DisruptionBudgetController) pdbForVMI(namespace, name string) (*v1beta1.PodDisruptionBudget, error) {
	pbds, err := c.pdbInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	for _, pdb := range pbds {
		p := v1.GetControllerOf(pdb.(*v1beta1.PodDisruptionBudget))
		if p != nil && p.Kind == virtv1.VirtualMachineInstanceGroupVersionKind.Kind &&
			p.Name == name {
			return pdb.(*v1beta1.PodDisruptionBudget), nil
		}
	}
	return nil, nil
}

func wantsToMigrateOnDrain(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Spec.EvictionStrategy == nil {
		return false
	}
	if *vmi.Spec.EvictionStrategy == virtv1.EvictionStrategyLiveMigrate {
		return true
	}
	return false
}
