package workloadupdater

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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
	"kubevirt.io/kubevirt/pkg/util/status"
)

const (
	// FailedCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration failed.
	FailedCreateVirtualMachineInstanceMigrationReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration succeeded.
	SuccessfulCreateVirtualMachineInstanceMigrationReason = "SuccessfulCreate"
	// FailedDeleteVirtualMachineReason is added in an event if a deletion of a VMI fails
	FailedDeleteVirtualMachineInstanceReason = "FailedDelete"
	// SuccessfulDeleteVirtualMachineReason is added in an event if a deletion of a VMI fails
	SuccessfulDeleteVirtualMachineInstanceReason = "SuccessfulDelete"
)

var (
	outdatedVMIWorkloads = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "num_outdated_vmi_workloads",
			Help: "Indication for the number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment.",
		},
	)
)

// time to wait before re-enqueing when outdated VMIs are still detected
const periodicReEnqueueIntervalSeconds = 30

// ensures we don't execute more than once every 5 seconds
const defaultThrottleIntervalSeconds = 5

const defaultBatchDeletionIntervalSeconds = 60
const defaultBatchDeletionCount = 10

func init() {
	prometheus.MustRegister(outdatedVMIWorkloads)
}

type WorkloadUpdateController struct {
	clientset             kubecli.KubevirtClient
	queue                 workqueue.RateLimitingInterface
	vmiInformer           cache.SharedIndexInformer
	migrationInformer     cache.SharedIndexInformer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	kubeVirtInformer      cache.SharedIndexInformer
	clusterConfig         *virtconfig.ClusterConfig
	statusUpdater         *status.KVStatusUpdater

	throttleIntervalSeconds int

	// This lock protects cached data within this struct
	// that is dynamic. The lock is held for the duration
	// of the reconcile loop. The reconcile loop is already
	// single threaded and only a single KubeVirt object
	// may exist in the cluster at once. This lock is simply
	// protection in the event that those assumptions ever
	// change.
	cacheLock sync.Mutex

	// loop can become quite chatty during the update process. This optimization
	// throttles how quickly the loop can fire since each loop execution is acting at
	// a cluster wide level.
	reconcileThrottleMap map[string]time.Time

	lastDeletionBatch time.Time
}

type updateData struct {
	allOutdatedVMIs        []*virtv1.VirtualMachineInstance
	migratableOutdatedVMIs []*virtv1.VirtualMachineInstance
	shutdownOutdatedVMIs   []*virtv1.VirtualMachineInstance

	numActiveMigrations int
}

func NewWorkloadUpdateController(
	vmiInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	kubeVirtInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
) *WorkloadUpdateController {

	c := &WorkloadUpdateController{
		queue:                 workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:           vmiInformer,
		migrationInformer:     migrationInformer,
		kubeVirtInformer:      kubeVirtInformer,
		recorder:              recorder,
		clientset:             clientset,
		statusUpdater:         status.NewKubeVirtStatusUpdater(clientset),
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		clusterConfig:         clusterConfig,

		throttleIntervalSeconds: defaultThrottleIntervalSeconds,

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
	if key == "" || err != nil {
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
	if key == "" || err != nil {
		return
	}

	c.queue.Add(key)
}

func (c *WorkloadUpdateController) updateMigration(old, curr interface{}) {
	key, err := c.getKubeVirtKey()
	if key == "" || err != nil {
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
	if vmi.Labels == nil {
		return false
	}

	_, labelExists := vmi.Labels[virtv1.OutdatedLauncherImageLabel]
	if labelExists {
		return true
	}

	return false
}

func isMigratable(vmi *virtv1.VirtualMachineInstance) bool {
	for _, c := range vmi.Status.Conditions {
		if c.Type == virtv1.VirtualMachineInstanceIsMigratable && c.Status == k8sv1.ConditionTrue {
			return true
		}
	}

	return false
}

func (c *WorkloadUpdateController) getUpdateData(kv *virtv1.KubeVirt) *updateData {
	data := &updateData{}

	lookup := make(map[string]bool)

	migrations := migrationutils.ListUnfinishedMigrations(c.migrationInformer)

	for _, migration := range migrations {
		lookup[migration.Namespace+"/"+migration.Spec.VMIName] = true
	}

	automatedMigrationAllowed := false
	automatedShutdownAllowed := false

	for _, method := range kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods {
		if method == virtv1.WorkloadUpdateMethodLiveMigrate {
			automatedMigrationAllowed = true
		} else if method == virtv1.WorkloadUpdateMethodShutdown {
			automatedShutdownAllowed = true
		}
	}

	data.numActiveMigrations = len(migrations)

	objs := c.vmiInformer.GetStore().List()
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if !vmi.IsRunning() || vmi.IsFinal() || vmi.DeletionTimestamp != nil {
			// only consider running VMIs that aren't being shutdown
			continue
		} else if !c.isOutdated(vmi) {
			continue
		}

		// add label to outdated vmis for sorting
		data.allOutdatedVMIs = append(data.allOutdatedVMIs, vmi)

		if automatedMigrationAllowed && isMigratable(vmi) {
			// don't consider VMIs with migrations inflight as migratable for our dataset
			if migrationutils.IsMigrating(vmi) {
				continue
			} else if exists := lookup[vmi.Namespace+"/"+vmi.Name]; exists {
				continue
			}

			data.migratableOutdatedVMIs = append(data.migratableOutdatedVMIs, vmi)
		} else if automatedShutdownAllowed {
			data.shutdownOutdatedVMIs = append(data.shutdownOutdatedVMIs, vmi)
		}
	}

	return data
}

func (c *WorkloadUpdateController) execute(key string) error {
	obj, exists, err := c.kubeVirtInformer.GetStore().GetByKey(key)

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	if err != nil {
		return err
	} else if !exists {
		c.migrationExpectations.DeleteExpectations(key)

		delete(c.reconcileThrottleMap, key)
		return nil
	}

	// don't process anything until expectations are satisfied
	// this ensures we don't do things like creating multiple
	// migrations for the same vmi
	if !c.migrationExpectations.SatisfiedExpectations(key) {
		return nil
	}

	now := time.Now()

	ts, ok := c.reconcileThrottleMap[key]
	if !ok {
		c.reconcileThrottleMap[key] = now.Add(time.Duration(c.throttleIntervalSeconds) * time.Second)
	} else if now.Before(ts) {
		c.queue.AddAfter(key, time.Duration(c.throttleIntervalSeconds)*time.Second)
		return nil
	}

	kv := obj.(*virtv1.KubeVirt)

	// don't update workloads unless the infra is completely deployed and not updating
	if kv.Status.Phase != virtv1.KubeVirtPhaseDeployed {
		return nil
	} else if kv.Status.ObservedDeploymentID != kv.Status.TargetDeploymentID {
		return nil
	}

	return c.sync(kv)
}

func (c *WorkloadUpdateController) sync(kv *virtv1.KubeVirt) error {

	data := c.getUpdateData(kv)

	key, err := controller.KeyFunc(kv)
	if err != nil {
		return err
	}

	outdatedVMIWorkloads.Set(float64(len(data.allOutdatedVMIs)))

	// update outdated workload count on kv
	if kv.Status.OutdatedVMIWorkloads == nil || *kv.Status.OutdatedVMIWorkloads != len(data.allOutdatedVMIs) {
		l := len(data.allOutdatedVMIs)
		kvCopy := kv.DeepCopy()
		kvCopy.Status.OutdatedVMIWorkloads = &l

		oldJson, err := json.Marshal(kv.Status.OutdatedVMIWorkloads)
		if err != nil {
			return err
		}

		newJson, err := json.Marshal(kvCopy.Status.OutdatedVMIWorkloads)
		if err != nil {
			return err
		}

		patch := ""
		if kv.Status.OutdatedVMIWorkloads == nil {
			update := fmt.Sprintf(`{ "op": "add", "path": "/status/outdatedVMIWorkloads", "value": %s}`, string(newJson))
			patch = fmt.Sprintf("[%s]", update)
		} else {
			test := fmt.Sprintf(`{ "op": "test", "path": "/status/outdatedVMIWorkloads", "value": %s}`, string(oldJson))
			update := fmt.Sprintf(`{ "op": "replace", "path": "/status/outdatedVMIWorkloads", "value": %s}`, string(newJson))
			patch = fmt.Sprintf("[%s, %s]", test, update)
		}

		err = c.statusUpdater.PatchStatus(kv, types.JSONPatchType, []byte(patch))
		if err != nil {
			return fmt.Errorf("unable to patch kubevirt obj status to update the outdatedVMIWorkloads valued: %v", err)
		}
	}

	// Rather than enqueing based on VMI activity, we keep periodically poping the loop
	// until all VMIs are updated. Watching all VMI activity is chatty for this controller
	// when we don't need to be that efficent in how quickly the updates are being processed.
	if len(data.shutdownOutdatedVMIs) != 0 || len(data.migratableOutdatedVMIs) != 0 {
		c.queue.AddAfter(key, periodicReEnqueueIntervalSeconds)
	}

	// Randomizes list so we don't always re-attempt the same vmis in
	// the event that some are having difficulty being relocated
	rand.Shuffle(len(data.migratableOutdatedVMIs), func(i, j int) {
		data.migratableOutdatedVMIs[i], data.migratableOutdatedVMIs[j] = data.migratableOutdatedVMIs[j], data.migratableOutdatedVMIs[i]
	})

	batchDeletionInterval := time.Duration(defaultBatchDeletionIntervalSeconds) * time.Second
	batchDeletionCount := defaultBatchDeletionCount

	if kv.Spec.WorkloadUpdateStrategy.BatchShutdownCount != nil {
		batchDeletionCount = *kv.Spec.WorkloadUpdateStrategy.BatchShutdownCount
	}

	if kv.Spec.WorkloadUpdateStrategy.BatchShutdownInterval != nil {
		batchDeletionInterval = kv.Spec.WorkloadUpdateStrategy.BatchShutdownInterval.Duration
	}

	now := time.Now()

	nextBatch := c.lastDeletionBatch.Add(batchDeletionInterval)
	if now.After(nextBatch) && len(data.shutdownOutdatedVMIs) > 0 {
		batchDeletionCount = int(math.Min(float64(batchDeletionCount), float64(len(data.shutdownOutdatedVMIs))))
		c.lastDeletionBatch = now
	} else {
		batchDeletionCount = 0
	}

	// This is a best effort attempt at not creating a bunch of pending migrations
	// in the event that we've hit the global max. This check isn't meant to prevent
	// overloading the cluster. The migration controller handles that. We're merely
	// optimizing here by not introducing new migration objects we know can't be processed
	// right now.
	maxParallelMigrations := int(*c.clusterConfig.GetMigrationConfiguration().ParallelMigrationsPerCluster)

	maxNewMigrations := maxParallelMigrations - data.numActiveMigrations
	if maxNewMigrations < 0 {
		maxNewMigrations = 0
	}

	migrateCount := int(math.Min(float64(maxNewMigrations), float64(len(data.migratableOutdatedVMIs))))
	migrationCandidates := []*virtv1.VirtualMachineInstance{}
	if migrateCount > 0 {
		migrationCandidates = data.migratableOutdatedVMIs[0:migrateCount]
		log.Log.Infof("workload updated is migrating %d VMIs", migrateCount)
	}

	deletionCandidates := []*virtv1.VirtualMachineInstance{}
	if batchDeletionCount > 0 {
		deletionCandidates = data.shutdownOutdatedVMIs[0:batchDeletionCount]
		log.Log.Infof("workload updated is force shutting down %d VMIs", batchDeletionCount)
	}

	wgLen := len(migrationCandidates) + len(deletionCandidates)
	wg := &sync.WaitGroup{}
	wg.Add(wgLen)
	errChan := make(chan error, wgLen)

	c.migrationExpectations.ExpectCreations(key, migrateCount)
	for _, vmi := range migrationCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			createdMigration, err := c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Create(&virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						virtv1.WorkloadUpdateMigrationAnnotation: "",
					},
					GenerateName: "kubevirt-workload-update-",
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
			})
			if err != nil {
				log.Log.Object(vmi).Reason(err).Errorf("Failed to migrate vmi as part of workload update")
				c.migrationExpectations.CreationObserved(key)
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreateVirtualMachineInstanceMigrationReason, "Error creating a Migration for automated workload update: %v", err)
				errChan <- err
				return
			} else {
				log.Log.Object(vmi).Infof("Migrated vmi as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreateVirtualMachineInstanceMigrationReason, "Created Migration %s for automated workload update", createdMigration.Name)
			}
		}(vmi)
	}

	for _, vmi := range deletionCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			err := c.clientset.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &v1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				log.Log.Object(vmi).Reason(err).Errorf("Failed to delete vmi as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeleteVirtualMachineInstanceReason, "Error deleting VMI during automated workload update: %v", err)
				errChan <- err
			} else {
				log.Log.Object(vmi).Infof("Deleted vmi as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeleteVirtualMachineInstanceReason, "Initiated shutdown of VMI as part of automated workload update: %v", err)
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
