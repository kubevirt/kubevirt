package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/trace"

	appsv1 "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/pointer"

	virtv1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

// Controller is the main Controller struct.
type Controller struct {
	clientset       kubecli.KubevirtClient
	queue           workqueue.TypedRateLimitingInterface[string]
	vmIndexer       cache.Indexer
	vmiStore        cache.Store
	poolIndexer     cache.Indexer
	revisionIndexer cache.Indexer
	recorder        record.EventRecorder
	expectations    *controller.UIDTrackingControllerExpectations
	burstReplicas   uint
	hasSynced       func() bool
}

const (
	FailedUpdateVirtualMachineReason     = "FailedUpdate"
	SuccessfulUpdateVirtualMachineReason = "SuccessfulUpdate"

	defaultAddDelay   = 1 * time.Second
	defaultRetryDelay = 3 * time.Second
)

const (
	FailedScaleOutReason        = "FailedScaleOut"
	FailedScaleInReason         = "FailedScaleIn"
	FailedUpdateReason          = "FailedUpdate"
	FailedRevisionPruningReason = "FailedRevisionPruning"

	SuccessfulPausedPoolReason = "SuccessfulPaused"
	SuccessfulResumePoolReason = "SuccessfulResume"
)

var virtControllerPoolWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

// NewController creates a new instance of the PoolController struct.
func NewController(clientset kubecli.KubevirtClient,
	vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	poolInformer cache.SharedIndexInformer,
	revisionInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	burstReplicas uint) (*Controller, error) {
	c := &Controller{
		clientset: clientset,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-pool"},
		),
		poolIndexer:     poolInformer.GetIndexer(),
		vmiStore:        vmiInformer.GetStore(),
		vmIndexer:       vmInformer.GetIndexer(),
		revisionIndexer: revisionInformer.GetIndexer(),
		recorder:        recorder,
		expectations:    controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		burstReplicas:   burstReplicas,
	}

	c.hasSynced = func() bool {
		return poolInformer.HasSynced() && vmInformer.HasSynced() && vmiInformer.HasSynced() && revisionInformer.HasSynced()
	}

	_, err := poolInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPool,
		DeleteFunc: c.deletePool,
		UpdateFunc: c.updatePool,
	})
	if err != nil {
		return nil, err
	}

	_, err = vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMHandler,
		DeleteFunc: c.deleteVMHandler,
		UpdateFunc: c.updateVMHandler,
	})
	if err != nil {
		return nil, err
	}

	_, err = revisionInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addRevisionHandler,
		UpdateFunc: c.updateRevisionHandler,
	})
	if err != nil {
		return nil, err
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMIHandler,
		UpdateFunc: c.updateVMIHandler,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Controller) resolveVMIControllerRef(namespace string, controllerRef *metav1.OwnerReference) *virtv1.VirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return nil
	}
	vm, exists, err := c.vmIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if vm.(*virtv1.VirtualMachine).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return vm.(*virtv1.VirtualMachine)
}

func (c *Controller) addVMIHandler(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	if vmi.DeletionTimestamp != nil {
		return
	}

	vmiControllerRef := metav1.GetControllerOf(vmi)
	if vmiControllerRef == nil {
		return
	}

	log.Log.Object(vmi).V(4).Info("Looking for VirtualMachineInstance Ref")
	vm := c.resolveVMIControllerRef(vmi.Namespace, vmiControllerRef)
	if vm == nil {
		// VMI is not controlled by a VM
		return
	}

	vmControllerRef := metav1.GetControllerOf(vm)
	if vmControllerRef == nil {
		return
	}

	pool := c.resolveControllerRef(vm.Namespace, vmControllerRef)
	if pool == nil {
		// VM is not controlled by a pool
		return
	}

	vmRevisionName, vmOk := vm.Spec.Template.ObjectMeta.Labels[virtv1.VirtualMachinePoolRevisionName]
	vmiRevisionName, vmiOk := vmi.Labels[virtv1.VirtualMachinePoolRevisionName]
	if vmOk && vmiOk && vmRevisionName == vmiRevisionName {
		// nothing to do here, VMI is up-to-date with VM's Template
		return
	}

	// enqueue the Pool due to a VMI detected that isn't up to date
	c.enqueuePool(pool)

}

func (c *Controller) updateVMIHandler(old, cur interface{}) {
	c.addVMIHandler(cur)
}

// When a revision is created, enqueue the pool that manages it and update its expectations.
func (c *Controller) addRevisionHandler(obj interface{}) {
	cr := obj.(*appsv1.ControllerRevision)

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(cr); controllerRef != nil {
		pool := c.resolveControllerRef(cr.Namespace, controllerRef)
		if pool == nil {
			return
		}
		poolKey, err := controller.KeyFunc(pool)
		if err != nil {
			return
		}
		c.expectations.CreationObserved(poolKey)
		c.enqueuePool(pool)
		return
	}
}

func (c *Controller) updateRevisionHandler(old, cur interface{}) {
	cr := cur.(*appsv1.ControllerRevision)

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(cr); controllerRef != nil {
		pool := c.resolveControllerRef(cr.Namespace, controllerRef)
		if pool == nil {
			return
		}

		c.enqueuePool(pool)
		return
	}
}

// When a vm is created, enqueue the pool that manages it and update its expectations.
func (c *Controller) addVMHandler(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)

	if vm.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vm shows up in a state that
		// is already pending deletion. Prevent the vm from being a creation observation.
		c.deleteVMHandler(vm)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(vm); controllerRef != nil {
		pool := c.resolveControllerRef(vm.Namespace, controllerRef)
		if pool == nil {
			return
		}
		poolKey, err := controller.KeyFunc(pool)
		if err != nil {
			return
		}
		log.Log.V(4).Object(vm).Infof("VirtualMachine created")
		c.expectations.CreationObserved(poolKey)
		c.enqueuePool(pool)
		return
	}
}

// When a vm is updated, figure out what pool/s manage it and wake them
// up. If the labels of the vm have changed we need to awaken both the old
// and new pool. old and cur must be *metav1.VirtualMachine types.
func (c *Controller) updateVMHandler(old, cur interface{}) {
	curVM := cur.(*virtv1.VirtualMachine)
	oldVM := old.(*virtv1.VirtualMachine)
	if curVM.ResourceVersion == oldVM.ResourceVersion {
		return
	}

	labelChanged := !equality.Semantic.DeepEqual(curVM.Labels, oldVM.Labels)
	if curVM.DeletionTimestamp != nil {
		c.deleteVMHandler(curVM)
		if labelChanged {
			c.deleteVMHandler(oldVM)
		}
		return
	}

	curControllerRef := metav1.GetControllerOf(curVM)
	oldControllerRef := metav1.GetControllerOf(oldVM)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if pool := c.resolveControllerRef(oldVM.Namespace, oldControllerRef); pool != nil {
			c.enqueuePool(pool)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		pool := c.resolveControllerRef(curVM.Namespace, curControllerRef)
		if pool == nil {
			return
		}
		log.Log.V(4).Object(curVM).Infof("VirtualMachine updated")
		c.enqueuePool(pool)
		return
	}
}

// When a vm is deleted, enqueue the pool that manages the vm and update its expectations.
// obj could be an *metav1.VirtualMachine, or a DeletionFinalStateUnknown marker item.
func (c *Controller) deleteVMHandler(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)

	// When a delete is dropped, the relist will notice a vm in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vm
	// changed labels the new Pool will not be woken up till the periodic resync.
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

	controllerRef := metav1.GetControllerOf(vm)
	if controllerRef == nil {
		return
	}
	pool := c.resolveControllerRef(vm.Namespace, controllerRef)
	if pool == nil {
		return
	}
	poolKey, err := controller.KeyFunc(pool)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(poolKey, controller.VirtualMachineKey(vm))
	c.enqueuePool(pool)
}

func (c *Controller) addPool(obj interface{}) {
	c.enqueuePool(obj)
}

func (c *Controller) deletePool(obj interface{}) {
	c.enqueuePool(obj)
}

func (c *Controller) updatePool(_, curr interface{}) {
	c.enqueuePool(curr)
}

func (c *Controller) enqueuePool(obj interface{}) {
	logger := log.Log
	pool := obj.(*poolv1.VirtualMachinePool)
	key, err := controller.KeyFunc(pool)
	if err != nil {
		logger.Object(pool).Reason(err).Error("Failed to extract key from pool.")
		return
	}

	// Delay prevents pool from being reconciled too often
	c.queue.AddAfter(key, defaultAddDelay)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *Controller) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *poolv1.VirtualMachinePool {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != poolv1.VirtualMachinePoolKind {
		return nil
	}
	pool, exists, err := c.poolIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if pool.(*poolv1.VirtualMachinePool).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return pool.(*poolv1.VirtualMachinePool)
}

// listControllerFromNamespace takes a namespace and returns all Pools from the Pool cache which run in this namespace
func (c *Controller) listControllerFromNamespace(namespace string) ([]*poolv1.VirtualMachinePool, error) {
	objs, err := c.poolIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	pools := []*poolv1.VirtualMachinePool{}
	for _, obj := range objs {
		pool := obj.(*poolv1.VirtualMachinePool)
		pools = append(pools, pool)
	}
	return pools, nil
}

// getMatchingController returns the first Pool which matches the labels of the VirtualMachine from the listener cache.
// If there are no matching controllers, a NotFound error is returned.
func (c *Controller) getMatchingControllers(vm *virtv1.VirtualMachine) (pools []*poolv1.VirtualMachinePool) {
	logger := log.Log
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	for _, pool := range controllers {
		selector, err := metav1.LabelSelectorAsSelector(pool.Spec.Selector)
		if err != nil {
			logger.Object(pool).Reason(err).Error("Failed to parse label selector from pool.")
			continue
		}

		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			pools = append(pools, pool)
		}

	}
	return pools
}

// Run runs the passed in PoolController.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting pool controller.")

	// Wait for cache sync before we start the pool controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping pool controller.")
}

func (c *Controller) runWorker() {
	for c.Execute() {
	}
}

func (c *Controller) listVMsFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
	objs, err := c.vmIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []*virtv1.VirtualMachine{}
	for _, obj := range objs {
		vms = append(vms, obj.(*virtv1.VirtualMachine))
	}
	return vms, nil
}

func (c *Controller) calcDiff(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) int {
	wantedReplicas := int32(1)
	if pool.Spec.Replicas != nil {
		wantedReplicas = *pool.Spec.Replicas
	}

	return len(vms) - int(wantedReplicas)
}

func filterRunningVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	filtered := []*virtv1.VirtualMachine{}
	for _, vm := range vms {
		if vm.DeletionTimestamp == nil {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

func filterDeletingVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	filtered := []*virtv1.VirtualMachine{}
	for _, vm := range vms {
		if vm.DeletionTimestamp != nil {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// filterReadyVMs takes a list of VMs and returns all VMs which are in ready state.
func (c *Controller) filterReadyVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filterVMs(vms, func(vm *virtv1.VirtualMachine) bool {
		return controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, virtv1.VirtualMachineConditionType(k8score.PodReady), k8score.ConditionTrue)
	})
}

func (c *Controller) filterNotReadyVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filterVMs(vms, func(vm *virtv1.VirtualMachine) bool {
		return !controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, virtv1.VirtualMachineConditionType(k8score.PodReady), k8score.ConditionTrue)
	})
}

func filterVMs(vms []*virtv1.VirtualMachine, f func(vmi *virtv1.VirtualMachine) bool) []*virtv1.VirtualMachine {
	filtered := []*virtv1.VirtualMachine{}
	for _, vm := range vms {
		if f(vm) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

func resolveBasePolicy(scaleInStrategy *poolv1.VirtualMachinePoolScaleInStrategy) poolv1.VirtualMachinePoolBasePolicy {
	if scaleInStrategy == nil ||
		scaleInStrategy.Proactive == nil ||
		scaleInStrategy.Proactive.SelectionPolicy == nil ||
		scaleInStrategy.Proactive.SelectionPolicy.BasePolicy == nil {
		return poolv1.VirtualMachinePoolBasePolicyRandom
	}
	return *scaleInStrategy.Proactive.SelectionPolicy.BasePolicy
}

func sortVMsForDownscale(vms []*virtv1.VirtualMachine, basePolicy poolv1.VirtualMachinePoolBasePolicy) {
	switch basePolicy {
	case poolv1.VirtualMachinePoolBasePolicyDescendingOrder:
		sortVMsByOrdinalDescending(vms)
	default:
		sortVMsRandom(vms)
	}
}

func sortVMsByOrdinalDescending(vms []*virtv1.VirtualMachine) {
	sort.Slice(vms, func(i, j int) bool {
		ordinalI, errI := indexFromName(vms[i].Name)
		ordinalJ, errJ := indexFromName(vms[j].Name)

		if errI != nil {
			ordinalI = 0
		}
		if errJ != nil {
			ordinalJ = 0
		}
		return ordinalI > ordinalJ
	})
}

func sortVMsRandom(vms []*virtv1.VirtualMachine) {
	rand.Shuffle(len(vms), func(i, j int) {
		vms[i], vms[j] = vms[j], vms[i]
	})
}

func (c *Controller) scaleIn(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine, count int) error {

	poolKey, err := controller.KeyFunc(pool)
	if err != nil {
		return err
	}

	elgibleVMs := filterRunningVMs(vms)

	// make sure we count already deleting VMs here during scale in.
	count = count - (len(vms) - len(elgibleVMs))

	if len(elgibleVMs) == 0 || count == 0 {
		return nil
	} else if count > len(elgibleVMs) {
		count = len(elgibleVMs)
	}

	basePolicy := resolveBasePolicy(pool.Spec.ScaleInStrategy)
	sortVMsForDownscale(elgibleVMs, basePolicy)

	log.Log.Object(pool).Infof("Removing %d VMs from pool", count)

	var wg sync.WaitGroup

	deleteList := elgibleVMs[0:count]
	c.expectations.ExpectDeletions(poolKey, controller.VirtualMachineKeys(deleteList))
	wg.Add(len(deleteList))
	errChan := make(chan error, len(deleteList))
	for i := 0; i < len(deleteList); i++ {
		go func(idx int) {
			defer wg.Done()
			vm := deleteList[idx]

			if err := c.removeFinalizer(vm); err != nil {
				log.Log.Object(pool).Reason(err).Errorf("Failed to remove finalizer from virtual machine %s/%s", vm.Namespace, vm.Name)
				errChan <- err
				return
			}

			if err := c.clientset.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{PropagationPolicy: pointer.P(metav1.DeletePropagationForeground)}); err != nil {
				c.expectations.DeletionObserved(poolKey, controller.VirtualMachineKey(vm))
				c.recorder.Eventf(pool, k8score.EventTypeWarning, common.FailedDeleteVirtualMachineReason, "Error deleting virtual machine %s/%s: %v", vm.Namespace, vm.Name, err)
				errChan <- err
				return
			}

			c.recorder.Eventf(pool, k8score.EventTypeNormal, common.SuccessfulDeleteVirtualMachineReason, "Deleted VM %s/%s with uid %v from pool", vm.Namespace, vm.Name, vm.ObjectMeta.UID)
			log.Log.Object(pool).Infof("Deleted vm %s/%s from pool", vm.Namespace, vm.Name)
		}(i)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred. We log the rest
		return err
	default:
	}

	return nil
}

func generateVMName(index int, baseName string) string {
	return fmt.Sprintf("%s-%d", baseName, index)
}

func calculateNewVMNames(count int, baseName string, namespace string, vmStore cache.Store) []string {
	var newNames []string

	// generate `count` new unused VM names
	curIndex := 0
	for n := 0; n < count; n++ {
		// find next unused index starting where we left off last
		i := curIndex
		for {
			name := generateVMName(i, baseName)
			vmKey := controller.NamespacedKey(namespace, name)
			_, exists, _ := vmStore.GetByKey(vmKey)
			if !exists {
				newNames = append(newNames, name)
				curIndex = i + 1
				break
			}

			i++
		}
	}

	return newNames
}

func poolOwnerRef(pool *poolv1.VirtualMachinePool) metav1.OwnerReference {
	t := pointer.P(true)
	gvk := schema.GroupVersionKind{Group: poolv1.SchemeGroupVersion.Group, Version: poolv1.SchemeGroupVersion.Version, Kind: poolv1.VirtualMachinePoolKind}
	return metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               pool.ObjectMeta.Name,
		UID:                pool.ObjectMeta.UID,
		Controller:         t,
		BlockOwnerDeletion: t,
	}
}

func indexFromName(name string) (int, error) {
	slice := strings.Split(name, "-")
	return strconv.Atoi(slice[len(slice)-1])
}

func indexVMSpec(poolSpec *poolv1.VirtualMachinePoolSpec, idx int) *virtv1.VirtualMachineSpec {
	spec := poolSpec.VirtualMachineTemplate.Spec.DeepCopy()

	dvNameMap := map[string]string{}
	for i := range spec.DataVolumeTemplates {

		indexName := fmt.Sprintf("%s-%d", spec.DataVolumeTemplates[i].Name, idx)
		dvNameMap[spec.DataVolumeTemplates[i].Name] = indexName

		spec.DataVolumeTemplates[i].Name = indexName
	}

	appendIndexToConfigMapRefs := false
	appendIndexToSecretRefs := false
	if poolSpec.NameGeneration != nil {
		if poolSpec.NameGeneration.AppendIndexToConfigMapRefs != nil {
			appendIndexToConfigMapRefs = *poolSpec.NameGeneration.AppendIndexToConfigMapRefs
		}
		if poolSpec.NameGeneration.AppendIndexToSecretRefs != nil {
			appendIndexToSecretRefs = *poolSpec.NameGeneration.AppendIndexToSecretRefs
		}
	}

	for i, volume := range spec.Template.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			indexName, ok := dvNameMap[volume.VolumeSource.PersistentVolumeClaim.ClaimName]
			if ok {
				spec.Template.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = indexName
			}
		} else if volume.VolumeSource.DataVolume != nil {
			indexName, ok := dvNameMap[volume.VolumeSource.DataVolume.Name]
			if ok {
				spec.Template.Spec.Volumes[i].DataVolume.Name = indexName
			}
		} else if volume.VolumeSource.ConfigMap != nil && appendIndexToConfigMapRefs {
			volume.VolumeSource.ConfigMap.Name += "-" + strconv.Itoa(idx)
		} else if volume.VolumeSource.Secret != nil && appendIndexToSecretRefs {
			volume.VolumeSource.Secret.SecretName += "-" + strconv.Itoa(idx)
		}
	}

	return spec
}

func injectPoolRevisionLabelsIntoVM(vm *virtv1.VirtualMachine, revisionName string) *virtv1.VirtualMachine {

	if vm.Labels == nil {
		vm.Labels = map[string]string{}
	}
	if vm.Spec.Template.ObjectMeta.Labels == nil {
		vm.Spec.Template.ObjectMeta.Labels = map[string]string{}
	}

	vm.Labels[virtv1.VirtualMachinePoolRevisionName] = revisionName
	vm.Spec.Template.ObjectMeta.Labels[virtv1.VirtualMachinePoolRevisionName] = revisionName

	return vm
}

func getRevisionName(pool *poolv1.VirtualMachinePool) string {
	return fmt.Sprintf("%s-%d", pool.Name, pool.Generation)
}

func (c *Controller) ensureControllerRevision(pool *poolv1.VirtualMachinePool) (string, error) {
	poolKey, err := controller.KeyFunc(pool)
	if err != nil {
		return "", err
	}
	revisionName := getRevisionName(pool)

	_, alreadyExists, err := c.getControllerRevision(pool.Namespace, revisionName)
	if err != nil {
		return "", err
	} else if alreadyExists {
		// already created
		return revisionName, nil
	}

	bytes, err := json.Marshal(&pool.Spec)
	if err != nil {
		return "", err
	}

	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            revisionName,
			Namespace:       pool.Namespace,
			OwnerReferences: []metav1.OwnerReference{poolOwnerRef(pool)},
		},
		Data:     runtime.RawExtension{Raw: bytes},
		Revision: pool.ObjectMeta.Generation,
	}

	c.expectations.RaiseExpectations(poolKey, 1, 0)
	_, err = c.clientset.AppsV1().ControllerRevisions(pool.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
	if err != nil {
		c.expectations.CreationObserved(poolKey)
		return "", err
	}

	return cr.Name, nil
}

func (c *Controller) getControllerRevision(namespace, name string) (*poolv1.VirtualMachinePoolSpec, bool, error) {

	key := controller.NamespacedKey(namespace, name)

	storeObj, exists, err := c.revisionIndexer.GetByKey(key)
	if !exists || err != nil {
		return nil, false, err
	}

	cr, ok := storeObj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, false, fmt.Errorf("unexpected resource %+v", storeObj)
	}

	spec := &poolv1.VirtualMachinePoolSpec{}

	err = json.Unmarshal(cr.Data.Raw, spec)
	if err != nil {
		return nil, false, err
	}
	return spec, true, nil

}

func (c *Controller) scaleOut(pool *poolv1.VirtualMachinePool, count int) error {

	var wg sync.WaitGroup

	newNames := calculateNewVMNames(count, pool.Name, pool.Namespace, c.vmIndexer)

	revisionName, err := c.ensureControllerRevision(pool)
	if err != nil {
		return err
	}

	log.Log.Object(pool).Infof("Adding %d VMs to pool", len(newNames))
	poolKey, err := controller.KeyFunc(pool)
	if err != nil {
		return err
	}

	// We have to create VMs
	c.expectations.RaiseExpectations(poolKey, len(newNames), 0)
	wg.Add(len(newNames))
	errChan := make(chan error, len(newNames))

	for _, name := range newNames {
		go func(name string) {
			defer wg.Done()

			index, err := indexFromName(name)
			if err != nil {
				errChan <- err
				return
			}

			vm := virtv1.NewVMReferenceFromNameWithNS(pool.Namespace, name)

			vm.Labels = maps.Clone(pool.Spec.VirtualMachineTemplate.ObjectMeta.Labels)
			vm.Annotations = maps.Clone(pool.Spec.VirtualMachineTemplate.ObjectMeta.Annotations)
			vm.Spec = *indexVMSpec(&pool.Spec, index)
			vm = injectPoolRevisionLabelsIntoVM(vm, revisionName)
			controller.AddFinalizer(vm, poolv1.VirtualMachinePoolControllerFinalizer)

			vm.ObjectMeta.OwnerReferences = []metav1.OwnerReference{poolOwnerRef(pool)}

			vm, err = c.clientset.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})

			if err != nil {
				c.expectations.CreationObserved(poolKey)
				log.Log.Object(pool).Reason(err).Errorf("Failed to add vm %s/%s to pool", pool.Namespace, name)
				errChan <- err
				return
			}
			c.recorder.Eventf(pool, k8score.EventTypeNormal, common.SuccessfulCreateVirtualMachineReason, "Created VM %s/%s", vm.Namespace, vm.ObjectMeta.Name)
			log.Log.Object(pool).Infof("Adding vm %s/%s to pool", pool.Namespace, name)
		}(name)
	}
	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred. We log the rest
		c.recorder.Eventf(pool, k8score.EventTypeWarning, common.FailedCreateVirtualMachineReason, "Error creating VM: %v", err)
		return err
	default:
	}

	return nil
}

func (c *Controller) scale(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) (common.SyncError, bool) {
	diff := c.calcDiff(pool, vms)
	if diff == 0 {
		if err := c.opportunisticScaleIn(vms); err != nil {
			return common.NewSyncError(fmt.Errorf("error during opportunistic scale in: %v", err), FailedScaleInReason), false
		}

		return nil, true
	}

	maxDiff := int(math.Min(math.Abs(float64(diff)), float64(c.burstReplicas)))
	if diff < 0 {
		err := c.scaleOut(pool, maxDiff)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("error during scale out: %v", err), FailedScaleOutReason), false
		}
	} else {
		err := c.scaleIn(pool, vms, maxDiff)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("error during scale in: %v", err), FailedScaleInReason), false
		}
	}

	return nil, false
}

func isVMIReady(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.DeletionTimestamp != nil || vmi.Status.Phase != virtv1.Running {
		return false
	}
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceReady, k8score.ConditionTrue)
}

func (c *Controller) getUnavailableVMICount(vms []*virtv1.VirtualMachine) (int, error) {
	unavailableCount := 0
	for _, vm := range vms {
		obj, exists, err := c.vmiStore.GetByKey(controller.NamespacedKey(vm.Namespace, vm.Name))
		if err != nil {
			return 0, err
		}
		if !exists {
			unavailableCount++
			continue
		}
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if !isVMIReady(vmi) {
			unavailableCount++
		}
	}
	return unavailableCount, nil
}

func (c *Controller) handleUnhealthyVMIs(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) error {
	notReadyVMs := c.filterNotReadyVMs(vms)

	for _, vm := range notReadyVMs {
		obj, exists, err := c.vmiStore.GetByKey(controller.NamespacedKey(vm.Namespace, vm.Name))
		if err != nil {
			return err
		}
		if exists {
			vmi := obj.(*virtv1.VirtualMachineInstance)
			if vmi.DeletionTimestamp != nil {
				continue
			}
			updateType, err := c.isOutdatedVMI(vm, vmi)
			if err != nil {
				return err
			}
			if err := c.handleResourceUpdate(pool, vm, vmi, updateType); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) opportunisticUpdate(pool *poolv1.VirtualMachinePool, vmOutdatedList []*virtv1.VirtualMachine) error {
	var wg sync.WaitGroup
	if len(vmOutdatedList) == 0 {
		return nil
	}

	revisionName, err := c.ensureControllerRevision(pool)
	if err != nil {
		return err
	}

	wg.Add(len(vmOutdatedList))
	errChan := make(chan error, len(vmOutdatedList))
	for i := 0; i < len(vmOutdatedList); i++ {
		go func(idx int) {
			defer wg.Done()
			vm := vmOutdatedList[idx]

			index, err := indexFromName(vm.Name)
			if err != nil {
				errChan <- err
				return
			}

			vmCopy := vm.DeepCopy()

			vmCopy.Labels = maps.Clone(pool.Spec.VirtualMachineTemplate.ObjectMeta.Labels)
			vmCopy.Annotations = maps.Clone(pool.Spec.VirtualMachineTemplate.ObjectMeta.Annotations)
			vmCopy.Spec = *indexVMSpec(&pool.Spec, index)
			vmCopy = injectPoolRevisionLabelsIntoVM(vmCopy, revisionName)

			_, err = c.clientset.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy, metav1.UpdateOptions{})
			if err != nil {
				c.recorder.Eventf(pool, k8score.EventTypeWarning, FailedUpdateVirtualMachineReason, "Error updating virtual machine %s/%s: %v", vm.Name, vm.Namespace, err)
				log.Log.Object(pool).Reason(err).Errorf("Error encountered during update of vm %s/%s in pool", vmCopy.Namespace, vmCopy.Name)
				errChan <- err
				return
			}
			log.Log.Object(pool).Infof("Updated vm %s/%s in pool", vmCopy.Namespace, vmCopy.Name)
			c.recorder.Eventf(pool, k8score.EventTypeNormal, SuccessfulUpdateVirtualMachineReason, "Updated VM %s/%s", vm.Namespace, vm.Name)
		}(i)
	}
	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred. We log the rest
		return err
	default:
	}

	return nil
}

func calculateMaxUnavailableInt(pool *poolv1.VirtualMachinePool) (int, error) {
	maxUnavailable := intstr.FromString("100%")
	if pool.Spec.MaxUnavailable != nil {
		maxUnavailable = *pool.Spec.MaxUnavailable
	}

	totalReplicas := int32(1)
	if pool.Spec.Replicas != nil {
		totalReplicas = *pool.Spec.Replicas
	}

	maxUnavailableInt := 0
	if maxUnavailable.Type == intstr.String {
		percentage, err := strconv.ParseInt(strings.TrimSuffix(maxUnavailable.StrVal, "%"), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid maxUnavailable percentage: %v", err)
		}
		maxUnavailableInt = int(totalReplicas) * int(percentage) / 100
	} else {
		maxUnavailableInt = int(maxUnavailable.IntVal)
	}

	if maxUnavailableInt < 1 {
		maxUnavailableInt = 1
	}

	return maxUnavailableInt, nil
}

func (c *Controller) proactiveUpdate(pool *poolv1.VirtualMachinePool, vmUpdatedList []*virtv1.VirtualMachine) error {
	// Handle unhealthy VMIs first to rollover any changes to the VMI spec in case last update failed
	if err := c.handleUnhealthyVMIs(pool, vmUpdatedList); err != nil {
		return err
	}

	maxUnavailableInt, err := calculateMaxUnavailableInt(pool)
	if err != nil {
		return err
	}
	unavailableCount, err := c.getUnavailableVMICount(vmUpdatedList)
	if err != nil {
		return err
	}

	maxUpdatable := maxUnavailableInt - unavailableCount
	for i := range vmUpdatedList {
		if maxUpdatable <= 0 {
			log.Log.V(4).Infof("Delaying proactive update for pool %s/%s - max unavailable (%d) reached", pool.Namespace, pool.Name, maxUnavailableInt)
			key, err := controller.KeyFunc(pool)
			if err != nil {
				return err
			}
			c.queue.AddAfter(key, defaultRetryDelay)
			return nil
		}
		vm := vmUpdatedList[i]
		obj, exists, err := c.vmiStore.GetByKey(controller.NamespacedKey(vm.Namespace, vm.Name))
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		vmi := obj.(*virtv1.VirtualMachineInstance)

		updateType, err := c.isOutdatedVMI(vm, vmi)
		if err != nil {
			return err
		}
		if updateType == proactiveUpdateTypeNone {
			continue
		}

		if err := c.handleResourceUpdate(pool, vm, vmi, updateType); err != nil {
			return err
		}
		maxUpdatable--
	}
	return nil
}

type proactiveUpdateType string

const (
	// VMI spec has changed within vmi pool and requires restart
	proactiveUpdateTypeRestart proactiveUpdateType = "restart"
	// VMI spec is identify in current vmi pool, just needs revision label updated
	proactiveUpdateTypePatchRevisionLabel proactiveUpdateType = "label-patch"
	// VMI does not need an update
	proactiveUpdateTypeNone proactiveUpdateType = "no-update"
	// VM needs to be deleted due to data volume changes
	proactiveUpdateTypeVMDelete proactiveUpdateType = "vm-delete"
)

func (c *Controller) isOutdatedVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (proactiveUpdateType, error) {
	// This function compares the pool revision (pool spec at a specific point in time) synced
	// to the VM vs the one used to create the VMI. By comparing the pool spec revisions between
	// the VM and VMI we can determine if the VM has mutated in a way that should result
	// in the VMI being updated. If the VMITemplate in these two pool revisions are not identical,
	// the VMI needs to be updated via forced restart when proactive updates are in use.
	//
	// Rules for determining if a VMI is out of date or not
	//
	// 1. If the VM revision name doesn't exist, it's going to get set by the reconcile loop.
	//    The (opportunist update) logic handles ensuring the VM revision name will get set again
	//    on a future reconcile loop.
	// 2. If the VMI revision name doesn't exist, the VMI has to be proactively restarted
	//    because we have no history of what revision was used to originate the VMI. The
	//    VM is an offline config we're comparing to, but the VMI is the active config.
	// 3. Compare the VMI template in the pool revision associated with the VM to the one
	//    associated with the VMI. If they are identical in name or DeepEquals, then no
	//    proactive restart is required.
	// 4. If the expected VMI template specs from the revisions are not identical in name, but
	//    are identical in DeepEquals, patch the VMI with the new revision name used on the vm.
	// 5. If only the DataVolumeTemplates differ, the VM needs to be deleted to ensure proper data volume handling.

	vmRevisionName, exists := vm.Labels[virtv1.VirtualMachinePoolRevisionName]
	if !exists {
		// If we can't detect the VM revision then consider the outdated
		// status as not being required. The VM revision will get set again
		// by this controller on a future reconcile loop
		return proactiveUpdateTypeNone, nil
	}

	vmiRevisionName, exists := vmi.Labels[virtv1.VirtualMachinePoolRevisionName]
	if !exists {
		// If the VMI doesn't have the revision label, then it is outdated
		log.Log.Infof("Marking vmi %s/%s for update due to missing revision label", vm.Namespace, vm.Name)
		return proactiveUpdateTypeRestart, nil
	}

	if vmRevisionName == vmiRevisionName {
		// no update required because revisions match
		return proactiveUpdateTypeNone, nil
	}

	// Get the pool revision used to create the VM
	poolSpecRevisionForVM, exists, err := c.getControllerRevision(vm.Namespace, vmRevisionName)
	if err != nil {
		return proactiveUpdateTypeNone, err
	} else if !exists {
		// if the revision associated with the pool can't be found, then
		// no update is required at this time. The revision will eventually
		// get created in a future reconcile loop and we'll be able to process the VMI.
		return proactiveUpdateTypeNone, nil
	}
	expectedVMITemplate := poolSpecRevisionForVM.VirtualMachineTemplate.Spec.Template
	expectedDataVolumeTemplates := poolSpecRevisionForVM.VirtualMachineTemplate.Spec.DataVolumeTemplates

	// Get the pool revision used to create the VMI
	poolSpecRevisionForVMI, exists, err := c.getControllerRevision(vm.Namespace, vmiRevisionName)
	if err != nil {
		return proactiveUpdateTypeRestart, err
	} else if !exists {
		// if the VMI does not have an associated revision, then we have to force
		// an update
		log.Log.Infof("Marking vmi %s/%s for update due to missing revision", vm.Namespace, vm.Name)
		return proactiveUpdateTypeRestart, nil
	}
	currentVMITemplate := poolSpecRevisionForVMI.VirtualMachineTemplate.Spec.Template
	currentDataVolumeTemplates := poolSpecRevisionForVMI.VirtualMachineTemplate.Spec.DataVolumeTemplates

	// If DataVolumeTemplates differ, we need to delete the VM to ensure proper data volume handling
	if !equality.Semantic.DeepEqual(currentDataVolumeTemplates, expectedDataVolumeTemplates) {
		log.Log.Infof("Marking vm %s/%s for deletion due to data volume changes", vm.Namespace, vm.Name)
		return proactiveUpdateTypeVMDelete, nil
	}

	// If the VMI templates differ between the revision used to create
	// the VM and the revision used to create the VMI, then the VMI
	// must be updated.
	if !equality.Semantic.DeepEqual(currentVMITemplate, expectedVMITemplate) {
		log.Log.Infof("Marking vmi %s/%s for update due out of sync spec", vm.Namespace, vm.Name)
		return proactiveUpdateTypeRestart, nil
	}

	// If we get here, the vmi templates are identical, but the revision
	// names are different, so patch the VMI with a new revision name.
	return proactiveUpdateTypePatchRevisionLabel, nil
}

func (c *Controller) isOutdatedVM(pool *poolv1.VirtualMachinePool, vm *virtv1.VirtualMachine) (bool, error) {

	if vm.Labels == nil {
		log.Log.Object(pool).Infof("Marking vm %s/%s for update due to missing labels ", vm.Namespace, vm.Name)
		return true, nil
	}

	revisionName, exists := vm.Labels[virtv1.VirtualMachinePoolRevisionName]
	if !exists {
		log.Log.Object(pool).Infof("Marking vm %s/%s for update due to missing revision labels ", vm.Namespace, vm.Name)
		return true, nil
	}

	oldPoolSpec, exists, err := c.getControllerRevision(pool.Namespace, revisionName)
	if err != nil {
		return true, err
	} else if !exists {
		log.Log.Object(pool).Infof("Marking vm %s/%s for update due to missing revision", vm.Namespace, vm.Name)
		return true, nil
	}

	if !equality.Semantic.DeepEqual(oldPoolSpec.VirtualMachineTemplate, pool.Spec.VirtualMachineTemplate) {
		log.Log.Object(pool).Infof("Marking vm %s/%s for update due out of date spec", vm.Namespace, vm.Name)
		return true, nil
	}

	return false, nil

}

func (c *Controller) pruneUnusedRevisions(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) common.SyncError {

	keys, err := c.revisionIndexer.IndexKeys("vmpool", string(pool.UID))
	if err != nil {
		return common.NewSyncError(fmt.Errorf("error while pruning vmpool revisions: %v", err), FailedRevisionPruningReason)
	}

	deletionMap := make(map[string]interface{})
	for _, key := range keys {
		strs := strings.Split(key, "/")
		if len(strs) != 2 {
			continue
		}
		deletionMap[strs[1]] = nil
	}

	for _, vm := range vms {

		// Check to see what revision is used by the VM, and remove
		// that from the revision prune list
		revisionName, exists := vm.Labels[virtv1.VirtualMachinePoolRevisionName]
		if exists {
			// remove from deletionMap since we found a VM that references this revision
			delete(deletionMap, revisionName)
		}

		// Check to see what revision is used by the VMI, and remove
		// that from the revision prune list
		vmiKey := controller.NamespacedKey(vm.Namespace, vm.Name)
		obj, exists, _ := c.vmiStore.GetByKey(vmiKey)
		if exists {
			vmi := obj.(*virtv1.VirtualMachineInstance)
			revisionName, exists = vmi.Labels[virtv1.VirtualMachinePoolRevisionName]
			if exists {
				// remove from deletionMap since we found a VMI that references this revision
				delete(deletionMap, revisionName)
			}
		}
	}

	for revisionName := range deletionMap {
		err := c.clientset.AppsV1().ControllerRevisions(pool.Namespace).Delete(context.Background(), revisionName, metav1.DeleteOptions{})
		if err != nil {
			return common.NewSyncError(fmt.Errorf("error while pruning vmpool revisions: %v", err), FailedRevisionPruningReason)
		}
	}

	return nil
}

func (c *Controller) update(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) (common.SyncError, bool) {
	// List of VMs that need to be updated
	vmOutdatedList := []*virtv1.VirtualMachine{}
	// List of VMs that are up-to-date that need to be checked to see if VMI is up-to-date
	vmUpdatedList := []*virtv1.VirtualMachine{}

	for _, vm := range vms {
		outdated, err := c.isOutdatedVM(pool, vm)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("error while detected outdated VMs: %v", err), FailedUpdateReason), false
		}

		if outdated {
			vmOutdatedList = append(vmOutdatedList, vm)
		} else {
			vmUpdatedList = append(vmUpdatedList, vm)
		}
	}

	err := c.opportunisticUpdate(pool, vmOutdatedList)
	if err != nil {
		return common.NewSyncError(fmt.Errorf("error during VM update: %v", err), FailedUpdateReason), false
	}

	err = c.proactiveUpdate(pool, vmUpdatedList)
	if err != nil {
		return common.NewSyncError(fmt.Errorf("error during VMI update: %v", err), FailedUpdateReason), false
	}

	vmUpdateStable := false
	if len(vmOutdatedList) == 0 {
		vmUpdateStable = true
	}

	return nil, vmUpdateStable
}

// Execute runs commands from the controller queue, if there is
// an error it requeues the command. Returns false if the queue
// is empty.
func (c *Controller) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	virtControllerPoolWorkQueueTracer.StartTrace(key, "virt-controller VMPool workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer virtControllerPoolWorkQueueTracer.StopTrace(key)

	err := c.execute(key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing pool %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed pool %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *Controller) updateStatus(origPool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine, syncErr common.SyncError) error {

	key, err := controller.KeyFunc(origPool)
	if err != nil {
		return err
	}
	defer virtControllerPoolWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VMPool Name", Value: origPool.Name})

	pool := origPool.DeepCopy()

	labelSelector, err := metav1.LabelSelectorAsSelector(pool.Spec.Selector)
	if err != nil {
		return err
	}

	pool.Status.LabelSelector = labelSelector.String()

	cm := controller.NewVirtualMachinePoolConditionManager()

	if syncErr != nil && !cm.HasCondition(pool, poolv1.VirtualMachinePoolReplicaFailure) {
		cm.UpdateCondition(pool,
			&poolv1.VirtualMachinePoolCondition{
				Type:               poolv1.VirtualMachinePoolReplicaFailure,
				Reason:             syncErr.Reason(),
				Message:            syncErr.Error(),
				LastTransitionTime: metav1.Now(),
				Status:             k8score.ConditionTrue,
			})
		c.recorder.Eventf(pool, k8score.EventTypeWarning, syncErr.Reason(), syncErr.Error())
	} else if syncErr == nil && cm.HasCondition(pool, poolv1.VirtualMachinePoolReplicaFailure) {
		cm.RemoveCondition(pool, poolv1.VirtualMachinePoolReplicaFailure)
	}

	if pool.Spec.Paused && !cm.HasCondition(pool, poolv1.VirtualMachinePoolReplicaPaused) {
		cm.UpdateCondition(pool,
			&poolv1.VirtualMachinePoolCondition{
				Type:               poolv1.VirtualMachinePoolReplicaPaused,
				Reason:             SuccessfulPausedPoolReason,
				Message:            "Pool controller is paused",
				LastTransitionTime: metav1.Now(),
				Status:             k8score.ConditionTrue,
			})

		c.recorder.Eventf(pool, k8score.EventTypeNormal, SuccessfulPausedPoolReason, "Pool is paused")
	} else if !pool.Spec.Paused && cm.HasCondition(pool, poolv1.VirtualMachinePoolReplicaPaused) {
		cm.RemoveCondition(pool, poolv1.VirtualMachinePoolReplicaPaused)
		c.recorder.Eventf(pool, k8score.EventTypeNormal, SuccessfulResumePoolReason, "Pool is unpaused")
	}

	pool.Status.Replicas = int32(len(vms))
	pool.Status.ReadyReplicas = int32(len(c.filterReadyVMs(vms)))

	if !equality.Semantic.DeepEqual(pool.Status, origPool.Status) || pool.Status.Replicas != pool.Status.ReadyReplicas {
		_, err := c.clientset.VirtualMachinePool(pool.Namespace).UpdateStatus(context.Background(), pool, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil

}

func (c *Controller) execute(key string) error {
	logger := log.DefaultLogger()

	var syncErr common.SyncError

	obj, poolExists, err := c.poolIndexer.GetByKey(key)
	if err != nil {
		return err
	}

	var pool *poolv1.VirtualMachinePool
	if poolExists {
		pool = obj.(*poolv1.VirtualMachinePool)
		logger = logger.Object(pool)
	} else {
		c.expectations.DeleteExpectations(key)
		return nil
	}

	selector, err := metav1.LabelSelectorAsSelector(pool.Spec.Selector)
	if err != nil {
		logger.Reason(err).Error("Invalid selector on pool, will not re-enqueue.")
		return nil
	}
	if !selector.Matches(labels.Set(pool.Spec.VirtualMachineTemplate.ObjectMeta.Labels)) {
		logger.Reason(err).Error("Selector does not match template labels, will not re-enqueue.")
		return nil
	}

	vms, err := c.listVMsFromNamespace(pool.ObjectMeta.Namespace)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch vms for namespace from cache.")
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := c.clientset.VirtualMachinePool(pool.ObjectMeta.Namespace).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != pool.ObjectMeta.UID {
			return nil, fmt.Errorf("original Pool %v/%v is gone: got uid %v, wanted %v", pool.Namespace, pool.Name, fresh.UID, pool.UID)
		}

		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(controller.RealVirtualMachineControl{Clientset: c.clientset}, pool, selector, virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind, canAdoptFunc)
	vms, err = cm.ReleaseDetachedVirtualMachines(vms)
	if err != nil {
		return err
	}

	if pool.DeletionTimestamp == nil {
		if err := c.addPoolFinalizer(pool); err != nil {
			return err
		}
	}

	needsSync := c.expectations.SatisfiedExpectations(key)
	if needsSync && !pool.Spec.Paused && pool.DeletionTimestamp == nil {
		scaleIsStable := false
		updateIsStable := false

		syncErr, scaleIsStable = c.scale(pool, vms)
		if syncErr != nil {
			logger.Reason(err).Error("Scaling the pool failed.")
		}

		needsSync = c.expectations.SatisfiedExpectations(key)
		if needsSync && scaleIsStable && syncErr == nil {
			// Handle updates after scale operations are satisfied.
			syncErr, updateIsStable = c.update(pool, vms)
		}

		needsSync = c.expectations.SatisfiedExpectations(key)
		if needsSync && syncErr == nil && scaleIsStable && updateIsStable {
			// handle pruning revisions after scale and update operations are satisfied
			syncErr = c.pruneUnusedRevisions(pool, vms)
		}
		virtControllerPoolWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VMPool Name", Value: pool.Name})
	} else if pool.DeletionTimestamp != nil {
		if err := c.handlePoolDeletion(pool, vms); err != nil {
			return err
		}

		syncErr = c.pruneUnusedRevisions(pool, vms)
	}

	err = c.updateStatus(pool, vms, syncErr)
	if err != nil {
		return err
	}

	return syncErr
}

func (c *Controller) handleResourceUpdate(pool *poolv1.VirtualMachinePool, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, updateType proactiveUpdateType) error {
	var err error

	switch updateType {
	case proactiveUpdateTypeRestart:
		err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
		log.Log.Object(pool).Infof("Proactive update of VM %s/%s by deleting outdated VMI", vm.Namespace, vm.Name)
	case proactiveUpdateTypeVMDelete:
		err = c.clientset.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{PropagationPolicy: pointer.P(metav1.DeletePropagationForeground)})
		log.Log.Object(pool).Infof("Proactive update of VM %s/%s by deleting VM due to data volume changes", vm.Namespace, vm.Name)
	case proactiveUpdateTypePatchRevisionLabel:
		patchSet := patch.New()
		vmiLabels := maps.Clone(vmi.Labels)
		if vmiLabels == nil {
			vmiLabels = make(map[string]string)
		}
		revisionName, exists := vm.Labels[virtv1.VirtualMachinePoolRevisionName]
		if !exists {
			return nil
		}
		vmiLabels[virtv1.VirtualMachinePoolRevisionName] = revisionName

		if vmi.Labels == nil {
			patchSet.AddOption(patch.WithAdd("/metadata/labels", vmiLabels))
		} else {
			patchSet.AddOption(
				patch.WithTest("/metadata/labels", vmi.Labels),
				patch.WithReplace("/metadata/labels", vmiLabels),
			)
		}

		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return fmt.Errorf("failed to marshal patch: %v", err)
		}

		_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("patching of vmi labels with new pool revision name: %v", err)
		}
		log.Log.Object(pool).Infof("Proactive update of VM %s/%s in pool via label patch", vm.Namespace, vm.Name)
	}

	if err != nil {
		c.recorder.Eventf(pool, k8score.EventTypeWarning, FailedUpdateVirtualMachineReason, "Error updating resource %s/%s: %v", vm.Namespace, vm.Name, err)
		return err
	}

	c.recorder.Eventf(pool, k8score.EventTypeNormal, common.SuccessfulDeleteVirtualMachineReason, "Successfully updated resource %s/%s", vm.Namespace, vm.Name)
	return nil
}

func patchFinalizer(oldFinalizers, newFinalizers []string) ([]byte, error) {
	return patch.New(
		patch.WithTest("/metadata/finalizers", oldFinalizers),
		patch.WithReplace("/metadata/finalizers", newFinalizers)).
		GeneratePayload()
}

func removeFinalizerFromList(origFinalizers []string, finalizer string) []string {
	var filtered []string
	for _, f := range origFinalizers {
		if f != finalizer {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func (c *Controller) removeFinalizer(vm *virtv1.VirtualMachine) error {
	if !controller.HasFinalizer(vm, poolv1.VirtualMachinePoolControllerFinalizer) {
		return nil
	}

	newFinalizers := removeFinalizerFromList(vm.Finalizers, poolv1.VirtualMachinePoolControllerFinalizer)
	patch, err := patchFinalizer(vm.Finalizers, newFinalizers)
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Controller) opportunisticScaleIn(vms []*virtv1.VirtualMachine) error {
	// Find VMs that are marked for deletion
	// This happens when someone manually deletes a VM or when pool is scaled down and VMs are marked for deletion
	deletingVMs := filterDeletingVMs(vms)
	for _, vm := range deletingVMs {
		if err := c.removeFinalizer(vm); err != nil {
			return err
		}
		log.Log.Object(vm).Infof("Removed finalizer from VM %s/%s", vm.Namespace, vm.Name)
	}

	return nil
}

func (c *Controller) addPoolFinalizer(pool *poolv1.VirtualMachinePool) error {
	if controller.HasFinalizer(pool, poolv1.VirtualMachinePoolControllerFinalizer) {
		return nil
	}

	newFinalizers := make([]string, 0, len(pool.Finalizers))
	copy(newFinalizers, pool.Finalizers)
	newFinalizers = append(newFinalizers, poolv1.VirtualMachinePoolControllerFinalizer)

	patch, err := patchFinalizer(pool.Finalizers, newFinalizers)
	if err != nil {
		log.Log.Object(pool).Errorf("Failed to marshal patch: %v", err)
		return err
	}

	_, err = c.clientset.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Controller) removePoolFinalizer(pool *poolv1.VirtualMachinePool) error {
	if !controller.HasFinalizer(pool, poolv1.VirtualMachinePoolControllerFinalizer) {
		return nil
	}

	newFinalizers := removeFinalizerFromList(pool.Finalizers, poolv1.VirtualMachinePoolControllerFinalizer)
	patch, err := patchFinalizer(pool.Finalizers, newFinalizers)
	if err != nil {
		log.Log.Object(pool).Errorf("Failed to marshal patch: %v", err)
		return err
	}

	_, err = c.clientset.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Controller) cleanupVMs(vms []*virtv1.VirtualMachine) error {
	for _, vm := range vms {
		if err := c.removeFinalizer(vm); err != nil {
			log.Log.Object(vm).Errorf("Failed to remove finalizer: %v", err)
			return err
		}
	}

	return nil
}

func (c *Controller) handlePoolDeletion(pool *poolv1.VirtualMachinePool, vms []*virtv1.VirtualMachine) error {
	if err := c.cleanupVMs(vms); err != nil {
		return err
	}

	if err := c.removePoolFinalizer(pool); err != nil {
		return err
	}

	return nil
}
