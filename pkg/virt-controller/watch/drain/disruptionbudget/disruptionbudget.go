package disruptionbudget

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/util/pdbs"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const deleteNotifFail = "Failed to process delete notification"

const (
	// FailedCreatePodDisruptionBudgetReason is added in an event if creating a PodDisruptionBudget failed.
	FailedCreatePodDisruptionBudgetReason = "FailedCreate"
	// SuccessfulCreatePodDisruptionBudgetReason is added in an event if creating a PodDisruptionBudget succeeded.
	SuccessfulCreatePodDisruptionBudgetReason = "SuccessfulCreate"
	// FailedDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget failed.
	FailedDeletePodDisruptionBudgetReason = "FailedDelete"
	// SuccessfulDeletePodDisruptionBudgetReason is added in an event if deleting a PodDisruptionBudget succeeded.
	SuccessfulDeletePodDisruptionBudgetReason = "SuccessfulDelete"
	// FailedUpdatePodDisruptionBudgetReason is added in an event if updating a PodDisruptionBudget failed.
	FailedUpdatePodDisruptionBudgetReason = "FailedUpdate"
	// SuccessfulUpdatePodDisruptionBudgetReason is added in an event if updating a PodDisruptionBudget succeeded.
	SuccessfulUpdatePodDisruptionBudgetReason = "SuccessfulUpdate"
)

type DisruptionBudgetController struct {
	clientset                       kubecli.KubevirtClient
	clusterConfig                   *virtconfig.ClusterConfig
	Queue                           workqueue.RateLimitingInterface
	vmiInformer                     cache.SharedIndexInformer
	pdbInformer                     cache.SharedIndexInformer
	podInformer                     cache.SharedIndexInformer
	migrationInformer               cache.SharedIndexInformer
	recorder                        record.EventRecorder
	podDisruptionBudgetExpectations *controller.UIDTrackingControllerExpectations
}

func NewDisruptionBudgetController(
	vmiInformer cache.SharedIndexInformer,
	pdbInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
) (*DisruptionBudgetController, error) {

	c := &DisruptionBudgetController{
		Queue:                           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-disruption-budget"),
		vmiInformer:                     vmiInformer,
		pdbInformer:                     pdbInformer,
		podInformer:                     podInformer,
		migrationInformer:               migrationInformer,
		recorder:                        recorder,
		clientset:                       clientset,
		clusterConfig:                   clusterConfig,
		podDisruptionBudgetExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	_, err := c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.pdbInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPodDisruptionBudget,
		DeleteFunc: c.deletePodDisruptionBudget,
		UpdateFunc: c.updatePodDisruptionBudget,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updatePod,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	cache.WaitForCacheSync(stopCh, c.pdbInformer.HasSynced, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.migrationInformer.HasSynced)

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
	obj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(key)

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
		vmi = virtv1.NewVMIReferenceFromNameWithNS(namespace, name)
	}

	// Only consider pdbs which belong to this vmi
	pdbs, err := pdbs.PDBsForVMI(vmi, c.pdbInformer)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to fetch pod disruption budgets for namespace from cache.")
		// If the situation does not change there is no benefit in retrying
		return nil
	}

	if len(pdbs) == 0 {
		return c.sync(key, vmiExists, vmi, nil)
	}

	for i := range pdbs {
		if syncErr := c.sync(key, vmiExists, vmi, pdbs[i]); syncErr != nil {
			err = syncErr
		}
	}
	return err
}

func (c *DisruptionBudgetController) isMigrationComplete(vmi *virtv1.VirtualMachineInstance, migrationName string) (bool, error) {
	objs, err := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return false, err
	}

	var migration *virtv1.VirtualMachineInstanceMigration
	for _, obj := range objs {
		vmim := obj.(*virtv1.VirtualMachineInstanceMigration)
		if vmim.GetName() == migrationName {
			migration = vmim
			break
		}
	}

	if migration == nil {
		// if no migration is found we consider it as completed
		return true, nil
	} else if !migration.IsFinal() {
		return false, nil
	}

	runningPods := controller.VMIActivePodsCount(vmi, c.podInformer)
	return runningPods == 1, nil
}

func (c *DisruptionBudgetController) isVMIMCompletedForPDB(pdb *policyv1.PodDisruptionBudget, vmi *virtv1.VirtualMachineInstance) (bool, error) {
	migrationName := pdb.ObjectMeta.Labels[virtv1.MigrationNameLabel]
	if migrationName == "" {
		return false, nil
	}

	return c.isMigrationComplete(vmi, migrationName)
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

func (c *DisruptionBudgetController) shrinkPDB(vmi *virtv1.VirtualMachineInstance, pdb *policyv1.PodDisruptionBudget) error {
	if pdb != nil && pdb.DeletionTimestamp == nil && pdb.Spec.MinAvailable.IntValue() != 1 {
		patchOps := []byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/minAvailable", "value": 1 }, { "op": "remove", "path": "/metadata/labels/%s" }]`,
			patch.EscapeJSONPointer(virtv1.MigrationNameLabel)))

		_, err := c.clientset.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Patch(context.Background(), pdb.Name, types.JSONPatchType, patchOps, v1.PatchOptions{})
		if err != nil {
			c.recorder.Eventf(vmi, corev1.EventTypeWarning, FailedUpdatePodDisruptionBudgetReason, "Error updating the PodDisruptionBudget %s: %v", pdb.Name, err)
			return err
		}
		c.recorder.Eventf(vmi, corev1.EventTypeNormal, SuccessfulUpdatePodDisruptionBudgetReason, "shrank PodDisruptionBudget %s", pdb.Name)
	}
	return nil
}

func (c *DisruptionBudgetController) createPDB(key string, vmi *virtv1.VirtualMachineInstance) error {
	minAvailable := intstr.FromInt(1)

	c.podDisruptionBudgetExpectations.ExpectCreations(key, 1)
	createdPDB, err := c.clientset.PolicyV1().PodDisruptionBudgets(vmi.Namespace).Create(context.Background(), &policyv1.PodDisruptionBudget{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind),
			},
			GenerateName: "kubevirt-disruption-budget-",
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					virtv1.CreatedByLabel: string(vmi.UID),
				},
			},
		},
	}, v1.CreateOptions{})
	if err != nil {
		c.podDisruptionBudgetExpectations.CreationObserved(key)
		c.recorder.Eventf(vmi, corev1.EventTypeWarning, FailedCreatePodDisruptionBudgetReason, "Error creating a PodDisruptionBudget: %v", err)
		return err
	}
	c.recorder.Eventf(vmi, corev1.EventTypeNormal, SuccessfulCreatePodDisruptionBudgetReason, "Created PodDisruptionBudget %s", createdPDB.Name)
	return nil
}

func isPDBFromOldVMI(vmi *virtv1.VirtualMachineInstance, pdb *policyv1.PodDisruptionBudget) bool {
	// The pdb might be from an old vmi with a different uid, delete and later create the correct one
	// The VMI always has a minimum grace period, so normally this should not happen, therefore no optimizations
	if pdb == nil {
		return false
	}
	ownerRef := v1.GetControllerOf(pdb)
	return ownerRef != nil && ownerRef.UID != vmi.UID
}

func (c *DisruptionBudgetController) sync(key string, vmiExists bool, vmi *virtv1.VirtualMachineInstance, pdb *policyv1.PodDisruptionBudget) error {
	needsEvictionProtection := c.vmiNeedsEvictionPDB(vmiExists, vmi)

	// check for deletions if pod exists
	if pdb != nil {
		if !vmiExists || vmi.DeletionTimestamp != nil {
			// being deleted
			log.Log.Infof("deleting pdb %s/%s due to VMI deletion", pdb.Namespace, pdb.Name)
			return c.deletePDB(key, pdb, vmi)
		} else if !needsEvictionProtection {
			// vmi isn't set to prevent eviction, so delete the pdb
			log.Log.Object(vmi).Infof("deleting pdb %s/%s due to not using evictionStrategy: LiveMigration|External", pdb.Namespace, pdb.Name)
			return c.deletePDB(key, pdb, vmi)
		} else if isPDBFromOldVMI(vmi, pdb) {
			// pdb for non existent vmi
			log.Log.Object(vmi).Infof("deleting pdb %s/%s due to VMI not existing anymore", pdb.Namespace, pdb.Name)
			return c.deletePDB(key, pdb, vmi)
		} else if pdbs.IsPDBFromOldMigrationController(pdb) {
			// pdb coming from an old migration controller
			log.Log.Object(vmi).Infof("deleting pdb %s/%s generated by an old migration controller", pdb.Namespace, pdb.Name)
			return c.deletePDB(key, pdb, vmi)
		} else {
			vmimCompleted, err := c.isVMIMCompletedForPDB(pdb, vmi)
			if err != nil {
				return err
			}
			if vmimCompleted {
				// pdb for completed migration
				log.Log.Object(vmi).Infof("shrinking pdb %s/%s due to migration completion", pdb.Namespace, pdb.Name)
				return c.shrinkPDB(vmi, pdb)
			}
		}
	} else if needsEvictionProtection {
		// pdb doesn't exist, create if vmi's eviction strategy means it is protected during drain.
		log.Log.Object(vmi).Infof("creating pdb for VMI %s/%s", vmi.Namespace, vmi.Name)
		return c.createPDB(key, vmi)
	}

	return nil
}

func (c *DisruptionBudgetController) vmiNeedsEvictionPDB(vmiExists bool, vmi *virtv1.VirtualMachineInstance) bool {
	if !vmiExists || vmi.DeletionTimestamp != nil {
		return false
	}

	evictionStrategy := migrations.VMIEvictionStrategy(c.clusterConfig, vmi)
	if evictionStrategy == nil {
		return false
	}

	switch *evictionStrategy {
	case virtv1.EvictionStrategyLiveMigrate, virtv1.EvictionStrategyExternal:
		return true
	case virtv1.EvictionStrategyLiveMigrateIfPossible:
		return vmi.IsMigratable()
	default:
		return false
	}
}
