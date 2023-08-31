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

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	deleteNotifFail       = "Failed to process delete notification"
	getObjectErrFmt       = "couldn't get object from tombstone %+v"
	objectNotMigrationFmt = "tombstone contained object that is not a migration %#v"
)

const (
	// FailedCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration failed.
	FailedCreateVirtualMachineInstanceMigrationReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration succeeded.
	SuccessfulCreateVirtualMachineInstanceMigrationReason = "SuccessfulCreate"
)

type EvacuationController struct {
	clientset             kubecli.KubevirtClient
	Queue                 workqueue.RateLimitingInterface
	vmiInformer           cache.SharedIndexInformer
	vmiPodInformer        cache.SharedIndexInformer
	migrationInformer     cache.SharedIndexInformer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	nodeInformer          cache.SharedIndexInformer
	clusterConfig         *virtconfig.ClusterConfig
}

func NewEvacuationController(
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	nodeInformer cache.SharedIndexInformer,
	vmiPodInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
) (*EvacuationController, error) {

	c := &EvacuationController{
		Queue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-evacuation"),
		vmiInformer:           vmiInformer,
		migrationInformer:     migrationInformer,
		nodeInformer:          nodeInformer,
		vmiPodInformer:        vmiPodInformer,
		recorder:              recorder,
		clientset:             clientset,
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		clusterConfig:         clusterConfig,
	}

	_, err := c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})

	if err != nil {
		return nil, err
	}

	_, err = c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addMigration,
		DeleteFunc: c.deleteMigration,
		UpdateFunc: c.updateMigration,
	})

	if err != nil {
		return nil, err
	}
	_, err = c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		DeleteFunc: c.deleteNode,
		UpdateFunc: c.updateNode,
	})

	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *EvacuationController) addNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *EvacuationController) deleteNode(obj interface{}) {
	c.enqueueNode(obj)
}

func (c *EvacuationController) updateNode(_, curr interface{}) {
	c.enqueueNode(curr)
}

func (c *EvacuationController) enqueueNode(obj interface{}) {
	logger := log.Log
	node := obj.(*k8sv1.Node)
	key, err := controller.KeyFunc(node)
	if err != nil {
		logger.Object(node).Reason(err).Error("Failed to extract key from node.")
		return
	}
	c.Queue.Add(key)
}

func (c *EvacuationController) addVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *EvacuationController) deleteVirtualMachineInstance(obj interface{}) {
	c.enqueueVMI(obj)
}

func (c *EvacuationController) updateVirtualMachineInstance(_, curr interface{}) {
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
			log.Log.Reason(fmt.Errorf(getObjectErrFmt, obj)).Error(deleteNotifFail)
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf(objectNotMigrationFmt, obj)).Error(deleteNotifFail)
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
			log.Log.Reason(fmt.Errorf(getObjectErrFmt, obj)).Error(deleteNotifFail)
			return ""
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf(objectNotMigrationFmt, obj)).Error(deleteNotifFail)
			return ""
		}
	}
	return vmi.Status.NodeName
}

func (c *EvacuationController) addMigration(obj interface{}) {
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)

	node := ""

	// only observe the migration expectation if our controller created it
	key, ok := migration.Annotations[virtv1.EvacuationMigrationAnnotation]
	if ok {
		c.migrationExpectations.CreationObserved(key)
		node = key
	} else {
		o, exists, err := c.vmiInformer.GetStore().GetByKey(migration.Namespace + "/" + migration.Spec.VMIName)
		if err != nil {
			return
		}
		if exists {
			node = c.nodeFromVMI(o)
		}
	}

	if node != "" {
		c.Queue.Add(node)
	}
}

func (c *EvacuationController) deleteMigration(obj interface{}) {
	c.enqueueMigration(obj)
}

func (c *EvacuationController) updateMigration(_, curr interface{}) {
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
			log.Log.Reason(fmt.Errorf(getObjectErrFmt, obj)).Error(deleteNotifFail)
			return
		}
		migration, ok = tombstone.Obj.(*virtv1.VirtualMachineInstanceMigration)
		if !ok {
			log.Log.Reason(fmt.Errorf(objectNotMigrationFmt, obj)).Error(deleteNotifFail)
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
		return
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

	vmis, err := c.listVMIsOnNode(node.Name)
	if err != nil {
		return fmt.Errorf("failed to list VMIs on node: %v", err)
	}

	migrations := migrationutils.ListUnfinishedMigrations(c.migrationInformer)

	return c.sync(node, vmis, migrations)
}

func getMarkedForEvictionVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	var evictionCandidates []*virtv1.VirtualMachineInstance
	for _, vmi := range vmis {
		if vmi.IsMarkedForEviction() && !hasMigratedOnEviction(vmi) && !migrationutils.IsMigrating(vmi) {
			evictionCandidates = append(evictionCandidates, vmi)
		}
	}
	return evictionCandidates
}

func GenerateNewMigration(vmiName string, key string) *virtv1.VirtualMachineInstanceMigration {

	annotations := map[string]string{
		virtv1.EvacuationMigrationAnnotation: key,
	}
	return &virtv1.VirtualMachineInstanceMigration{
		ObjectMeta: v1.ObjectMeta{
			Annotations:  annotations,
			GenerateName: "kubevirt-evacuation-",
		},
		Spec: virtv1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
}

func (c *EvacuationController) sync(node *k8sv1.Node, vmisOnNode []*virtv1.VirtualMachineInstance, activeMigrations []*virtv1.VirtualMachineInstanceMigration) error {
	// If the node has no drain taint, we have nothing to do
	taintKey := *c.clusterConfig.GetMigrationConfiguration().NodeDrainTaintKey
	taint := &k8sv1.Taint{
		Key:    taintKey,
		Effect: k8sv1.TaintEffectNoSchedule,
	}

	vmisToMigrate := vmisToMigrate(node, vmisOnNode, taint)
	if len(vmisToMigrate) == 0 {
		return nil
	}

	migrationCandidates, nonMigrateable := c.filterRunningNonMigratingVMIs(vmisToMigrate, activeMigrations)
	if len(migrationCandidates) == 0 && len(nonMigrateable) == 0 {
		return nil
	}

	runningMigrations := migrationutils.FilterRunningMigrations(activeMigrations)
	activeMigrationsFromThisSourceNode := c.numOfVMIMForThisSourceNode(vmisOnNode, runningMigrations)
	maxParallelMigrationsPerOutboundNode :=
		int(*c.clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode)
	maxParallelMigrations := int(*c.clusterConfig.GetMigrationConfiguration().ParallelMigrationsPerCluster)
	freeSpotsPerCluster := maxParallelMigrations - len(runningMigrations)
	freeSpotsPerThisSourceNode := maxParallelMigrationsPerOutboundNode - activeMigrationsFromThisSourceNode
	freeSpots := int(math.Min(float64(freeSpotsPerCluster), float64(freeSpotsPerThisSourceNode)))
	if freeSpots <= 0 {
		c.Queue.AddAfter(node.Name, 5*time.Second)
		return nil
	}

	diff := int(math.Min(float64(freeSpots), float64(len(migrationCandidates))))
	remaining := freeSpots - diff
	remainingForNonMigrateableDiff := int(math.Min(float64(remaining), float64(len(nonMigrateable))))

	if remainingForNonMigrateableDiff > 0 {
		// for all non-migrating VMIs which would get e spot emit a warning
		for _, vmi := range nonMigrateable[0:remainingForNonMigrateableDiff] {
			c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, FailedCreateVirtualMachineInstanceMigrationReason, "VirtualMachineInstance is not migrateable")
		}

	}

	if diff == 0 {
		if remainingForNonMigrateableDiff > 0 {
			// Let's ensure that some warnings will stay in the event log and periodically update
			// In theory the warnings could disappear after one hour if nothing else updates
			c.Queue.AddAfter(node.Name, 1*time.Minute)
		}
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
	for _, vmi := range selectedCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			createdMigration, err := c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Create(GenerateNewMigration(vmi.Name, node.Name), &v1.CreateOptions{})
			if err != nil {
				c.migrationExpectations.CreationObserved(node.Name)
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreateVirtualMachineInstanceMigrationReason, "Error creating a Migration: %v", err)
				errChan <- err
				return
			} else {
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreateVirtualMachineInstanceMigrationReason, "Created Migration %s", createdMigration.Name)
			}
		}(vmi)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
	}
	return nil
}

func hasMigratedOnEviction(vmi *virtv1.VirtualMachineInstance) bool {
	return vmi.Status.NodeName != vmi.Status.EvacuationNodeName
}

func vmisToMigrate(node *k8sv1.Node, vmisOnNode []*virtv1.VirtualMachineInstance, taint *k8sv1.Taint) []*virtv1.VirtualMachineInstance {
	var vmisToMigrate []*virtv1.VirtualMachineInstance
	if nodeHasTaint(taint, node) {
		vmisToMigrate = vmisOnNode
	} else if evictedVMIs := getMarkedForEvictionVMIs(vmisOnNode); len(evictedVMIs) > 0 {
		vmisToMigrate = evictedVMIs
	}
	return vmisToMigrate
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

func (c *EvacuationController) filterRunningNonMigratingVMIs(vmis []*virtv1.VirtualMachineInstance, migrations []*virtv1.VirtualMachineInstanceMigration) (migrateable []*virtv1.VirtualMachineInstance, nonMigrateable []*virtv1.VirtualMachineInstance) {
	lookup := map[string]bool{}
	for _, migration := range migrations {
		lookup[migration.Namespace+"/"+migration.Spec.VMIName] = true
	}

	for _, vmi := range vmis {
		// vmi is shutting down
		if vmi.IsFinal() || vmi.DeletionTimestamp != nil {
			continue
		}

		// does not want to migrate
		if !migrationutils.VMIMigratableOnEviction(c.clusterConfig, vmi) {
			continue
		}
		// can't migrate
		if !controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceIsMigratable, k8sv1.ConditionTrue) {
			nonMigrateable = append(nonMigrateable, vmi)
			continue
		}

		hasMigration := lookup[vmi.Namespace+"/"+vmi.Name]
		// already migrating
		if hasMigration {
			continue
		}

		if controller.VMIActivePodsCount(vmi, c.vmiPodInformer) > 1 {
			// waiting on target/source pods from a previous migration to terminate
			//
			// We only want to create a migration when num pods == 1 or else we run the
			// risk of invalidating our pdb which prevents the VMI from being evicted
			continue
		}

		// no migration exists,
		// the vmi is running,
		// only one pod is currently active for vmi
		migrateable = append(migrateable, vmi)
	}
	return migrateable, nonMigrateable
}

// deprecated
// This node evacuation method is deprecated. Use node drain to trigger evictions instead.
func nodeHasTaint(taint *k8sv1.Taint, node *k8sv1.Node) bool {
	for _, t := range node.Spec.Taints {
		if t.MatchTaint(taint) {
			return true
		}
	}
	return false
}

func (c *EvacuationController) numOfVMIMForThisSourceNode(
	vmisOnNode []*virtv1.VirtualMachineInstance,
	activeMigrations []*virtv1.VirtualMachineInstanceMigration) (activeMigrationsFromThisSourceNode int) {

	vmiMap := make(map[string]bool)
	for _, vmi := range vmisOnNode {
		vmiMap[vmi.Name] = true
	}

	for _, vmim := range activeMigrations {
		if _, ok := vmiMap[vmim.Spec.VMIName]; ok {
			activeMigrationsFromThisSourceNode++
		}
	}

	return
}
