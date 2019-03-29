package evacuation

import (
	"fmt"
	"math"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type EvacuationController struct {
	clientset             kubecli.KubevirtClient
	Queue                 workqueue.RateLimitingInterface
	vmiInformer           cache.SharedIndexInformer
	migrationInformer     cache.SharedIndexInformer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	nodeInformer          cache.SharedIndexInformer
}

func NewEvacuationController(
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	nodeInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
) *EvacuationController {

	c := &EvacuationController{
		Queue:                 workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:           vmiInformer,
		migrationInformer:     migrationInformer,
		nodeInformer:          nodeInformer,
		recorder:              recorder,
		clientset:             clientset,
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})

	c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addMigration,
		DeleteFunc: c.deleteMigration,
		UpdateFunc: c.updateMigration,
	})

	c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
	})

	return c
}

func (c *EvacuationController) addNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *EvacuationController) deleteNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *EvacuationController) updateNode(old, curr interface{}) {
	c.enqueueNode(curr)
}

func (c *EvacuationController) enqueueNode(obj interface{}) {
	logger := log.Log
	node := obj.(*k8sv1.Node)
	key, err := controller.KeyFunc(node)
	if err != nil {
		logger.Object(node).Reason(err).Error("Failed to extract key from node.")
	}
	c.Queue.Add(key)
}

func (c *EvacuationController) addVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *EvacuationController) deleteVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *EvacuationController) updateVirtualMachineInstance(old, curr interface{}) {
	c.enqueueVMI(curr)
}

func (c *EvacuationController) enqueueVMI(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a migration in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the migration
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a migration %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	node := c.nodeFromVMI(vmi)
	if node != "" {
		c.Queue.Add(node)
	}
}

func (c *EvacuationController) nodeFromVMI(obj interface{}) string {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a migration in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the migration
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return ""
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a migration %#v", obj)).Error("Failed to process delete notification")
			return ""
		}
	}
	return vmi.Status.NodeName
}

func (c *EvacuationController) addMigration(obj interface{}) {
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)
	o, exists, err := c.vmiInformer.GetStore().GetByKey(migration.Namespace + "/" + migration.Spec.VMIName)
	if err != nil {
		return
	}
	if exists {
		node := c.nodeFromVMI(o)
		if node != "" {
			c.migrationExpectations.CreationObserved(node)
			c.Queue.Add(node)
		}
	}
}

func (c *EvacuationController) deleteMigration(obj interface{}) {
	c.enqueueMigration(obj)
}

func (c *EvacuationController) updateMigration(old, curr interface{}) {
	c.enqueueMigration(curr)
}

func (c *EvacuationController) enqueueMigration(obj interface{}) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)

	// When a delete is dropped, the relist will notice a migration in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the migration
	// changed labels the new migration will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		migration, ok = tombstone.Obj.(*virtv1.VirtualMachineInstanceMigration)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a migration %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	o, exists, err := c.vmiInformer.GetStore().GetByKey(migration.Namespace + "/" + migration.Spec.VMIName)
	if err != nil {
		return
	}
	if exists {
		c.enqueueVMI(o)
	}
}

func (c *EvacuationController) enqueueVirtualMachine(obj interface{}) {
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
func (c *EvacuationController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineInstance {
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
func (c *EvacuationController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting evacuation controller.")

	// Wait for cache sync before we start the node controller
	cache.WaitForCacheSync(stopCh, c.migrationInformer.HasSynced, c.vmiInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping evacuation controller.")
}

func (c *EvacuationController) runWorker() {
	for c.Execute() {
	}
}

func (c *EvacuationController) Execute() bool {
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

func (c *EvacuationController) execute(key string) error {

	// Fetch the latest node state from cache
	obj, exists, err := c.nodeInformer.GetStore().GetByKey(key)

	if err != nil {
		return err
	}

	if !exists {
		c.migrationExpectations.DeleteExpectations(key)
		return nil
	}

	if !c.migrationExpectations.SatisfiedExpectations(key) {
		return nil
	}

	node := obj.(*k8sv1.Node)

	// If the node has not taints, we have nothing to do
	if !nodeHasTaint(node) {
		return nil
	}

	vmis, err := c.listVMIsOnNode(node.Name)
	if err != nil {
		// XXX
		return err
	}

	migrations, err := c.listRunningMigrations()

	if err != nil {
		// XXX
		return err
	}

	return c.sync(node, vmis, migrations)

}

func (c *EvacuationController) sync(node *k8sv1.Node, vmisOnNode []*virtv1.VirtualMachineInstance, activeMigrations []*virtv1.VirtualMachineInstanceMigration) error {
	if len(vmisOnNode) == 0 {
		// nothing to do
		return nil
	}

	now := time.Now()

	notMigratingVMIs := c.filterRunningNonMigratingVMIs(vmisOnNode, activeMigrations)
	notForeverToleratedVMIs := c.filterNotToleratedVMIs(now, notMigratingVMIs, node.Spec.Taints)

	// Ensure we enqueue VMIs again which are only temporary tolerated
	// and treat the others as migration candidates
	migrationCandidates := []*notolerate{}
	for _, candidate := range notForeverToleratedVMIs {
		if len(candidate.NotTolerated) > 0 {
			migrationCandidates = append(migrationCandidates, candidate)
		} else if candidate.FirstReque != nil {
			c.Queue.AddAfter(node.Name, candidate.FirstReque.Sub(now))
		}
	}

	if len(migrationCandidates) == 0 {
		// nothing to do
		return nil
	}

	if len(activeMigrations) >= 5 {
		// Don't create hundreds of pending migration objects.
		// This is just best-effort and is *not* intended to not overload the cluster
		// The migration controller needs to limit itself to a reasonable number
		// We have to re-enqueue since migrations from other controllers or workers` don't wake us up again
		c.Queue.AddAfter(node.Name, 5*time.Second)
		return nil
	}

	diff := int(math.Min(float64(5-len(activeMigrations)), float64(len(migrationCandidates))))

	if diff == 0 {
		// nothing to do
		return nil
	}

	// TODO: should the order be randomized?
	selectedCandidates := migrationCandidates[0:diff]
	log.DefaultLogger().Infof("node: %v, migrations: %v, candidates: %v, selected: %v", node.Name, len(activeMigrations), len(migrationCandidates), len(selectedCandidates))

	wg := &sync.WaitGroup{}
	wg.Add(diff)

	errChan := make(chan error, diff)

	c.migrationExpectations.ExpectCreations(node.Name, diff)
	for _, entry := range selectedCandidates {
		go func(entry *notolerate) {
			defer wg.Done()
			_, err := c.clientset.VirtualMachineInstanceMigration(entry.VMI.Namespace).Create(&virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "kubevirt-evacuation-",
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: entry.VMI.Name,
				},
			})
			if err != nil {
				c.migrationExpectations.CreationObserved(node.Name)
				errChan <- err
			}
		}(entry)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
	}
	return nil
}

func (c *EvacuationController) listVMIsOnNode(nodeName string) ([]*virtv1.VirtualMachineInstance, error) {
	objs, err := c.vmiInformer.GetIndexer().ByIndex("node", nodeName)
	if err != nil {
		return nil, err
	}
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, obj := range objs {
		vmis = append(vmis, obj.(*virtv1.VirtualMachineInstance))
	}
	return vmis, nil
}

func (c *EvacuationController) listRunningMigrations() ([]*virtv1.VirtualMachineInstanceMigration, error) {
	objs := c.migrationInformer.GetStore().List()
	migrations := []*virtv1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*virtv1.VirtualMachineInstanceMigration)
		if migration.Status.Phase != virtv1.MigrationFailed && migration.Status.Phase != virtv1.MigrationSucceeded {
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
}

func (c *EvacuationController) filterRunningNonMigratingVMIs(vmis []*virtv1.VirtualMachineInstance, migrations []*virtv1.VirtualMachineInstanceMigration) []*virtv1.VirtualMachineInstance {
	lookup := map[string]bool{}
	for _, migration := range migrations {
		lookup[migration.Namespace+"/"+migration.Spec.VMIName] = true
	}

	migrationCandidates := []*virtv1.VirtualMachineInstance{}

	for _, vmi := range vmis {

		if !controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceIsMigratable, k8sv1.ConditionTrue) {
			continue
		}
		if exists := lookup[vmi.Namespace+"/"+vmi.Name]; !exists &&
			!vmi.IsFinal() && vmi.DeletionTimestamp == nil {
			// no migration exists,
			// the vmi is running,
			migrationCandidates = append(migrationCandidates, vmi)
		}
	}
	return migrationCandidates
}

func (c *EvacuationController) filterNotToleratedVMIs(now time.Time, vmis []*virtv1.VirtualMachineInstance, taints []k8sv1.Taint) []*notolerate {

	migrationCandidates := []*notolerate{}

	for _, vmi := range vmis {
		if notTolerated, temporaryTolerated, firstRequeueTime := findNotToleratedTaints(now, vmi.Spec.EvictionPolicy, taints); notTolerated != nil || temporaryTolerated != nil {
			migrationCandidates = append(migrationCandidates, &notolerate{vmi, notTolerated, temporaryTolerated, firstRequeueTime})
		}
	}
	return migrationCandidates
}

func nodeHasTaint(node *k8sv1.Node) bool {
	return len(node.Spec.Taints) > 0
}

type notolerate struct {
	VMI                *virtv1.VirtualMachineInstance
	NotTolerated       []k8sv1.Taint
	TemporaryTolerated []k8sv1.Taint
	FirstReque         *time.Time
}

func findNotToleratedTaints(now time.Time, evictionStrategies *virtv1.EvictionPolicy, taints []k8sv1.Taint) (notTolerated []k8sv1.Taint, temporaryTolerated []k8sv1.Taint, firstRequeueTime *time.Time) {
	if evictionStrategies == nil {
		return
	}

	for _, taint := range taints {
		var tolerated *virtv1.TaintEvictionPolicy
		for _, toleration := range evictionStrategies.Taints {
			if toleration.ToleratesTaint(&taint) {
				tolerated = &toleration
				break
			}
		}
		if tolerated != nil {
			// we only care about VMIs with an eviction policy of LiveMigrate
			if tolerated.Strategy == nil || *tolerated.Strategy != virtv1.EvictionStrategyLiveMigrate {
				continue
			} else if tolerated.TolerationSeconds != nil && taint.TimeAdded != nil {
				toleratedUntil := now.Add(time.Duration(*tolerated.TolerationSeconds) * time.Second)
				if toleratedUntil.Before(taint.TimeAdded.Time) {
					notTolerated = append(notTolerated, taint)
				} else {
					if firstRequeueTime == nil {
						firstRequeueTime = &toleratedUntil
					} else if firstRequeueTime.After(toleratedUntil) {
						firstRequeueTime = &toleratedUntil
					}
					temporaryTolerated = append(temporaryTolerated, taint)
				}
			} else {
				notTolerated = append(notTolerated, taint)
			}
		}
	}
	return
}
