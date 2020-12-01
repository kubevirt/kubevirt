package workloadupdater

import (
	"fmt"
	"math"
	"math/rand"
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

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	// FailedCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration failed.
	FailedCreateVirtualMachineInstanceMigrationReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration succeeded.
	SuccessfulCreateVirtualMachineInstanceMigrationReason = "SuccessfulCreate"
)

// time to wait before re-enqueing when max migration count is encountered
const reEnqueueIntervalSeconds = 10

// ensures we don't execute more than once every 5 seconds
const throttleIntervalSeconds = 5

type WorkloadUpdateController struct {
	clientset             kubecli.KubevirtClient
	queue                 workqueue.RateLimitingInterface
	vmiInformer           cache.SharedIndexInformer
	migrationInformer     cache.SharedIndexInformer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	kubeVirtInformer      cache.SharedIndexInformer
	clusterConfig         *virtconfig.ClusterConfig
	launcherImage         string

	// loop can become quite chatty during the update process. This optimization
	// throttles how quickly the loop can fire since each loop execution is acting at
	// a cluster wide level.
	reconcileThrottleMap map[string]time.Time
	mapLock              sync.Mutex
}

type updateData struct {
	allOutdatedVMIs        []*virtv1.VirtualMachineInstance
	migratableOutdatedVMIs []*virtv1.VirtualMachineInstance

	numActiveMigrations int
}

func NewWorkloadUpdateController(
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	kubeVirtInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
	launcherImage string,
) *WorkloadUpdateController {

	c := &WorkloadUpdateController{
		queue:                 workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:           vmiInformer,
		migrationInformer:     migrationInformer,
		kubeVirtInformer:      kubeVirtInformer,
		recorder:              recorder,
		clientset:             clientset,
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		clusterConfig:         clusterConfig,
		launcherImage:         launcherImage,

		reconcileThrottleMap: make(map[string]time.Time),
	}

	c.kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addKubeVirt,
		DeleteFunc: c.deleteKubeVirt,
		UpdateFunc: c.updateKubeVirt,
	})

	c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addMigration,
		DeleteFunc: c.deleteMigration,
		UpdateFunc: c.updateMigration,
	})

	return c
}

func (c *WorkloadUpdateController) getKubeVirtKey() (string, error) {
	kvs := c.kubeVirtInformer.GetStore().List()
	if len(kvs) > 1 {
		log.Log.Errorf("More than one KubeVirt custom resource detected: %v", len(kvs))
		return "", fmt.Errorf("more than one KubeVirt custom resource detected: %v", len(kvs))
	}

	if len(kvs) == 1 {
		kv := kvs[0].(*virtv1.KubeVirt)
		return controller.KeyFunc(kv)
	}
	return "", nil
}

func (c *WorkloadUpdateController) addMigration(obj interface{}) {
	migration, ok := obj.(*virtv1.VirtualMachineInstanceMigration)
	if !ok {
		return
	}

	key, err := c.getKubeVirtKey()
	if err != nil {
		return
	}

	if migration.Annotations != nil {
		// only observe the migration expectation if our controller created it
		_, ok = migration.Annotations[virtv1.WorkloadUpdateMigrationAnnotation]
		if ok {
			c.migrationExpectations.CreationObserved(key)
		}
	}

	c.queue.Add(key)
}

func (c *WorkloadUpdateController) deleteMigration(obj interface{}) {
	key, err := c.getKubeVirtKey()
	if err != nil {
		return
	}

	c.queue.Add(key)
}

func (c *WorkloadUpdateController) updateMigration(old, curr interface{}) {
	key, err := c.getKubeVirtKey()
	if err != nil {
		return
	}

	c.queue.Add(key)
}

func (c *WorkloadUpdateController) addKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *WorkloadUpdateController) deleteKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *WorkloadUpdateController) updateKubeVirt(old, curr interface{}) {
	c.enqueueKubeVirt(curr)
}

func (c *WorkloadUpdateController) enqueueKubeVirt(obj interface{}) {
	logger := log.Log
	kv := obj.(*virtv1.KubeVirt)
	key, err := controller.KeyFunc(kv)
	if err != nil {
		logger.Object(kv).Reason(err).Error("Failed to extract key from KubeVirt.")
	}
	c.queue.Add(key)
}

// Run runs the passed in NodeController.
func (c *WorkloadUpdateController) Run(stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting workload update controller.")

	// This is hardcoded because there's no reason to make thread count
	// configurable. The queue keys off the KubeVirt install object, and
	// there can only be a single one of these in a cluster at a time.
	threadiness := 1

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.migrationInformer.HasSynced, c.vmiInformer.HasSynced, c.kubeVirtInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping workload update controller.")
}

func (c *WorkloadUpdateController) runWorker() {
	for c.Execute() {
	}
}

func (c *WorkloadUpdateController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing workload updates for KubeVirt %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed workload updates for KubeVirt %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *WorkloadUpdateController) isOutdated(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.IsFinal() {
		return false
	}

	// if the launcher image isn't detected yet, that means
	// we don't know what the launcher image is yet.
	// This could be due to a migration, or the VMI is still
	// initializing. virt-controller will set it for us once
	// either the VMI is either running or done migrating.
	if vmi.Status.CurrentLauncherImage == "" {
		return false
	} else if vmi.Status.CurrentLauncherImage == c.launcherImage {
		return false
	}

	return true
}

func (c *WorkloadUpdateController) getUpdateData() *updateData {
	data := &updateData{}

	lookup := make(map[string]bool)

	migrations := migrationutils.ListUnfinishedMigrations(c.migrationInformer)

	for _, migration := range migrations {
		lookup[migration.Namespace+"/"+migration.Spec.VMIName] = true
	}

	data.numActiveMigrations = len(migrations)

	objs := c.vmiInformer.GetStore().List()
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if c.isOutdated(vmi) {
			data.allOutdatedVMIs = append(data.allOutdatedVMIs, vmi)

			// don't consider VMIs with migrations inflight as migratable for our dataset
			if migrationutils.IsMigrating(vmi) {
				continue
			} else if exists := lookup[vmi.Namespace+"/"+vmi.Name]; exists {
				continue
			}

			// make sure the vmi has the migration condition set to true
			// in order to consider it migratable
			for _, c := range vmi.Status.Conditions {
				if c.Type == virtv1.VirtualMachineInstanceIsMigratable && c.Status == k8sv1.ConditionTrue {
					data.migratableOutdatedVMIs = append(data.migratableOutdatedVMIs, vmi)
					break
				}
			}
		}
	}
	return data
}

func (c *WorkloadUpdateController) execute(key string) error {
	obj, exists, err := c.kubeVirtInformer.GetStore().GetByKey(key)

	if err != nil {
		return err
	} else if !exists {
		c.migrationExpectations.DeleteExpectations(key)

		c.mapLock.Lock()
		delete(c.reconcileThrottleMap, key)
		c.mapLock.Unlock()
		return nil
	}

	// don't process anything until expectations are satisfied
	// this ensures we don't do things like creating multiple
	// migrations for the same vmi
	if !c.migrationExpectations.SatisfiedExpectations(key) {
		return nil
	}

	now := time.Now()
	c.mapLock.Lock()
	ts, ok := c.reconcileThrottleMap[key]
	if !ok {
		c.reconcileThrottleMap[key] = now.Add(time.Duration(throttleIntervalSeconds) * time.Second)
	} else if now.Before(ts) {
		c.queue.AddAfter(key, time.Duration(throttleIntervalSeconds)*time.Second)
		return nil
	}
	c.mapLock.Unlock()

	kv := obj.(*virtv1.KubeVirt)

	// don't update workloads unless the infra is completely deployed and not updating
	if kv.Status.Phase != virtv1.KubeVirtPhaseDeployed {
		return nil
	} else if kv.Status.ObservedDeploymentID != kv.Status.TargetDeploymentID {
		return nil
	}

	data := c.getUpdateData()

	return c.sync(kv, data)
}

func (c *WorkloadUpdateController) sync(kv *virtv1.KubeVirt, data *updateData) error {

	// nothing to do
	if len(data.migratableOutdatedVMIs) == 0 {
		return nil
	}

	key, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	// This is a best effort attempt at not creating a bunch of pending migrations
	// in the event that we've hit the global max. This check isn't meant to prevent
	// overloading the cluster. The migration controller handles that. We're merely
	// optimizing here by not introducing new migration objects we know can't be processed
	// right now.
	maxParallelMigrations := int(*c.clusterConfig.GetMigrationConfiguration().ParallelMigrationsPerCluster)
	if data.numActiveMigrations >= maxParallelMigrations {
		c.queue.AddAfter(key, time.Duration(reEnqueueIntervalSeconds)*time.Second)
		return nil
	}

	maxNewMigrations := maxParallelMigrations - data.numActiveMigrations
	if maxNewMigrations == 0 {
		c.queue.AddAfter(key, 1*time.Minute)
		return nil
	}

	count := int(math.Min(float64(maxNewMigrations), float64(len(data.migratableOutdatedVMIs))))

	migrationCandidates := data.migratableOutdatedVMIs[0:count]
	rand.Shuffle(len(migrationCandidates), func(i, j int) {
		migrationCandidates[i], migrationCandidates[j] = migrationCandidates[j], migrationCandidates[i]
	})

	wg := &sync.WaitGroup{}
	wg.Add(len(migrationCandidates))

	errChan := make(chan error, count)

	c.migrationExpectations.ExpectCreations(key, count)
	annotations := map[string]string{
		virtv1.WorkloadUpdateMigrationAnnotation: "",
	}
	for _, vmi := range migrationCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			createdMigration, err := c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Create(&virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: v1.ObjectMeta{
					Annotations:  annotations,
					GenerateName: "kubevirt-workload-update-",
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
			})
			if err != nil {
				c.migrationExpectations.CreationObserved(key)
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
