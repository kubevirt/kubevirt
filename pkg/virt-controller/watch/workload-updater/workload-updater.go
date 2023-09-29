package workloadupdater

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus"
	k8sv1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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
	"kubevirt.io/kubevirt/pkg/util/status"
)

const (
	// FailedCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration failed.
	FailedCreateVirtualMachineInstanceMigrationReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineInstanceMigrationReason is added in an event if creating a VirtualMachineInstanceMigration succeeded.
	SuccessfulCreateVirtualMachineInstanceMigrationReason = "SuccessfulCreate"
	// FailedEvictVirtualMachineInstanceReason is added in an event if a deletion of a VMI fails
	FailedEvictVirtualMachineInstanceReason = "FailedEvict"
	// SuccessfulEvictVirtualMachineInstanceReason is added in an event if a deletion of a VMI Succeeds
	SuccessfulEvictVirtualMachineInstanceReason = "SuccessfulEvict"
)

var (
	outdatedVirtualMachineInstanceWorkloads = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubevirt_vmi_number_of_outdated",
			Help: "Indication for the total number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment.",
		},
	)
)

// time to wait before re-enqueing when outdated VMIs are still detected
const periodicReEnqueueIntervalSeconds = 30

// ensures we don't execute more than once every 5 seconds
const defaultThrottleInterval = 5 * time.Second

const defaultBatchDeletionIntervalSeconds = 60
const defaultBatchDeletionCount = 10

func init() {
	prometheus.MustRegister(outdatedVirtualMachineInstanceWorkloads)
}

type WorkloadUpdateController struct {
	clientset             kubecli.KubevirtClient
	queue                 workqueue.RateLimitingInterface
	vmiInformer           cache.SharedIndexInformer
	podInformer           cache.SharedIndexInformer
	migrationInformer     cache.SharedIndexInformer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	kubeVirtInformer      cache.SharedIndexInformer
	clusterConfig         *virtconfig.ClusterConfig
	statusUpdater         *status.KVStatusUpdater
	launcherImage         string

	lastDeletionBatch time.Time
}

type updateData struct {
	allOutdatedVMIs        []*virtv1.VirtualMachineInstance
	migratableOutdatedVMIs []*virtv1.VirtualMachineInstance
	evictOutdatedVMIs      []*virtv1.VirtualMachineInstance

	numActiveMigrations int
}

func NewWorkloadUpdateController(
	launcherImage string,
	vmiInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	kubeVirtInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
) (*WorkloadUpdateController, error) {

	rl := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(defaultThrottleInterval, 300*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Every(defaultThrottleInterval), 1)},
	)

	c := &WorkloadUpdateController{
		queue:                 workqueue.NewNamedRateLimitingQueue(rl, "virt-controller-workload-update"),
		vmiInformer:           vmiInformer,
		podInformer:           podInformer,
		migrationInformer:     migrationInformer,
		kubeVirtInformer:      kubeVirtInformer,
		recorder:              recorder,
		clientset:             clientset,
		statusUpdater:         status.NewKubeVirtStatusUpdater(clientset),
		launcherImage:         launcherImage,
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		clusterConfig:         clusterConfig,
	}

	_, err := c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateVmi,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addKubeVirt,
		DeleteFunc: c.deleteKubeVirt,
		UpdateFunc: c.updateKubeVirt,
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

	return c, nil
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

	c.queue.AddAfter(key, defaultThrottleInterval)
}

func (c *WorkloadUpdateController) deleteMigration(_ interface{}) {
	key, err := c.getKubeVirtKey()
	if key == "" || err != nil {
		return
	}

	c.queue.AddAfter(key, defaultThrottleInterval)
}

func (c *WorkloadUpdateController) updateMigration(_, _ interface{}) {
	key, err := c.getKubeVirtKey()
	if key == "" || err != nil {
		return
	}

	c.queue.AddAfter(key, defaultThrottleInterval)
}

func (c *WorkloadUpdateController) updateVmi(_, obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)
	if !ok {
		return
	}

	key, err := c.getKubeVirtKey()
	if key == "" || err != nil {
		return
	}

	if vmi.IsFinal() {
		return
	}

	if !isHotplugInProgress(vmi) || migrationutils.IsMigrating(vmi) {
		return
	}

	c.queue.AddAfter(key, defaultThrottleInterval)
}

func (c *WorkloadUpdateController) addKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *WorkloadUpdateController) deleteKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *WorkloadUpdateController) updateKubeVirt(_, curr interface{}) {
	c.enqueueKubeVirt(curr)
}

func (c *WorkloadUpdateController) enqueueKubeVirt(obj interface{}) {
	logger := log.Log
	kv, ok := obj.(*virtv1.KubeVirt)
	if !ok {
		return
	}
	key, err := controller.KeyFunc(kv)
	if err != nil {
		logger.Object(kv).Reason(err).Error("Failed to extract key from KubeVirt.")
		return
	}
	c.queue.AddAfter(key, defaultThrottleInterval)
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
	cache.WaitForCacheSync(stopCh, c.migrationInformer.HasSynced, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.kubeVirtInformer.HasSynced)

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
	if vmi.Status.LauncherContainerImageVersion == "" {
		return false
	} else if vmi.Status.LauncherContainerImageVersion != c.launcherImage {
		return true
	}

	return false
}

func isHotplugInProgress(vmi *virtv1.VirtualMachineInstance) bool {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	return condManager.HasCondition(vmi, virtv1.VirtualMachineInstanceVCPUChange) ||
		condManager.HasCondition(vmi, virtv1.VirtualMachineInstanceMemoryChange)
}

func (c *WorkloadUpdateController) doesRequireMigration(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.IsFinal() || migrationutils.IsMigrating(vmi) {
		return false
	}

	if isHotplugInProgress(vmi) {
		return true
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
		} else if method == virtv1.WorkloadUpdateMethodEvict {
			automatedShutdownAllowed = true
		}
	}

	runningMigrations := migrationutils.FilterRunningMigrations(migrations)
	data.numActiveMigrations = len(runningMigrations)

	objs := c.vmiInformer.GetStore().List()
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if !vmi.IsRunning() || vmi.IsFinal() || vmi.DeletionTimestamp != nil {
			// only consider running VMIs that aren't being shutdown
			continue
		} else if !c.isOutdated(vmi) && !c.doesRequireMigration(vmi) {
			continue
		}

		data.allOutdatedVMIs = append(data.allOutdatedVMIs, vmi)

		// don't consider VMIs with migrations inflight as migratable for our dataset
		// while a migrating workload can still be counted towards
		// the outDatedVMIs list, we don't want to add it to any
		// of the lists that results in actions being performed on them
		if migrationutils.IsMigrating(vmi) {
			continue
		} else if exists := lookup[vmi.Namespace+"/"+vmi.Name]; exists {
			continue
		}

		if automatedMigrationAllowed && vmi.IsMigratable() {
			data.migratableOutdatedVMIs = append(data.migratableOutdatedVMIs, vmi)
		} else if automatedShutdownAllowed {
			data.evictOutdatedVMIs = append(data.evictOutdatedVMIs, vmi)
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
		return nil
	}

	// don't process anything until expectations are satisfied
	// this ensures we don't do things like creating multiple
	// migrations for the same vmi
	if !c.migrationExpectations.SatisfiedExpectations(key) {
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

	outdatedVirtualMachineInstanceWorkloads.Set(float64(len(data.allOutdatedVMIs)))

	// update outdated workload count on kv
	if kv.Status.OutdatedVirtualMachineInstanceWorkloads == nil || *kv.Status.OutdatedVirtualMachineInstanceWorkloads != len(data.allOutdatedVMIs) {
		l := len(data.allOutdatedVMIs)
		kvCopy := kv.DeepCopy()
		kvCopy.Status.OutdatedVirtualMachineInstanceWorkloads = &l

		oldJson, err := json.Marshal(kv.Status.OutdatedVirtualMachineInstanceWorkloads)
		if err != nil {
			return err
		}

		newJson, err := json.Marshal(kvCopy.Status.OutdatedVirtualMachineInstanceWorkloads)
		if err != nil {
			return err
		}

		patch := ""
		if kv.Status.OutdatedVirtualMachineInstanceWorkloads == nil {
			update := fmt.Sprintf(`{ "op": "add", "path": "/status/outdatedVirtualMachineInstanceWorkloads", "value": %s}`, string(newJson))
			patch = fmt.Sprintf("[%s]", update)
		} else {
			test := fmt.Sprintf(`{ "op": "test", "path": "/status/outdatedVirtualMachineInstanceWorkloads", "value": %s}`, string(oldJson))
			update := fmt.Sprintf(`{ "op": "replace", "path": "/status/outdatedVirtualMachineInstanceWorkloads", "value": %s}`, string(newJson))
			patch = fmt.Sprintf("[%s, %s]", test, update)
		}

		err = c.statusUpdater.PatchStatus(kv, types.JSONPatchType, []byte(patch))
		if err != nil {
			return fmt.Errorf("unable to patch kubevirt obj status to update the outdatedVirtualMachineInstanceWorkloads valued: %v", err)
		}
	}

	// Rather than enqueing based on VMI activity, we keep periodically poping the loop
	// until all VMIs are updated. Watching all VMI activity is chatty for this controller
	// when we don't need to be that efficent in how quickly the updates are being processed.
	if len(data.evictOutdatedVMIs) != 0 || len(data.migratableOutdatedVMIs) != 0 {
		c.queue.AddAfter(key, periodicReEnqueueIntervalSeconds)
	}

	// Randomizes list so we don't always re-attempt the same vmis in
	// the event that some are having difficulty being relocated
	rand.Shuffle(len(data.migratableOutdatedVMIs), func(i, j int) {
		data.migratableOutdatedVMIs[i], data.migratableOutdatedVMIs[j] = data.migratableOutdatedVMIs[j], data.migratableOutdatedVMIs[i]
	})

	batchDeletionInterval := time.Duration(defaultBatchDeletionIntervalSeconds) * time.Second
	batchDeletionCount := defaultBatchDeletionCount

	if kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize != nil {
		batchDeletionCount = *kv.Spec.WorkloadUpdateStrategy.BatchEvictionSize
	}

	if kv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval != nil {
		batchDeletionInterval = kv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval.Duration
	}

	now := time.Now()

	nextBatch := c.lastDeletionBatch.Add(batchDeletionInterval)
	if now.After(nextBatch) && len(data.evictOutdatedVMIs) > 0 {
		batchDeletionCount = int(math.Min(float64(batchDeletionCount), float64(len(data.evictOutdatedVMIs))))
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
	var migrationCandidates []*virtv1.VirtualMachineInstance
	if migrateCount > 0 {
		migrationCandidates = data.migratableOutdatedVMIs[0:migrateCount]
	}

	var evictionCandidates []*virtv1.VirtualMachineInstance
	if batchDeletionCount > 0 {
		evictionCandidates = data.evictOutdatedVMIs[0:batchDeletionCount]
	}

	wgLen := len(migrationCandidates) + len(evictionCandidates)
	wg := &sync.WaitGroup{}
	wg.Add(wgLen)
	errChan := make(chan error, wgLen)

	c.migrationExpectations.ExpectCreations(key, migrateCount)
	for _, vmi := range migrationCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			createdMigration, err := c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Create(&virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						virtv1.WorkloadUpdateMigrationAnnotation: "",
					},
					GenerateName: "kubevirt-workload-update-",
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
			}, &metav1.CreateOptions{})
			if err != nil {
				log.Log.Object(vmi).Reason(err).Errorf("Failed to migrate vmi as part of workload update")
				c.migrationExpectations.CreationObserved(key)
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreateVirtualMachineInstanceMigrationReason, "Error creating a Migration for automated workload update: %v", err)
				errChan <- err
				return
			} else {
				log.Log.Object(vmi).Infof("Initiated migration of vmi as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreateVirtualMachineInstanceMigrationReason, "Created Migration %s for automated workload update", createdMigration.Name)
			}
		}(vmi)
	}

	for _, vmi := range evictionCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()

			pod, err := controller.CurrentVMIPod(vmi, c.podInformer)
			if err != nil {

				log.Log.Object(vmi).Reason(err).Errorf("Failed to detect active pod for vmi during workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedEvictVirtualMachineInstanceReason, "Error detecting active pod for VMI during workload update: %v", err)
				errChan <- err
			}

			err = c.clientset.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(),
				&policy.Eviction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
					DeleteOptions: &metav1.DeleteOptions{},
				})

			if err != nil && !errors.IsNotFound(err) {
				log.Log.Object(vmi).Reason(err).Errorf("Failed to evict vmi as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedEvictVirtualMachineInstanceReason, "Error deleting VMI during automated workload update: %v", err)
				errChan <- err
			} else {
				log.Log.Object(vmi).Infof("Evicted vmi pod as part of workload update")
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulEvictVirtualMachineInstanceReason, "Initiated eviction of VMI as part of automated workload update: %v", err)
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
