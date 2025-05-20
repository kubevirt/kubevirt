package workloadupdater

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"

	k8sv1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	volumemig "kubevirt.io/kubevirt/pkg/virt-controller/watch/volume-migration"
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
	// SuccessfulChangeAbortionReason is added in an event if a deletion of a
	// migration succeeds
	SuccessfulChangeAbortionReason = "SuccessfulChangeAbortion"
	// FailedChangeAbortionReason is added in an event if a deletion of a
	// migration succeeds
	FailedChangeAbortionReason = "FailedChangeAbortion"
)

// time to wait before re-enqueing when outdated VMIs are still detected
const periodicReEnqueueIntervalSeconds = 30

// ensures we don't execute more than once every 5 seconds
const defaultThrottleInterval = 5 * time.Second

const defaultBatchDeletionIntervalSeconds = 60
const defaultBatchDeletionCount = 10

type WorkloadUpdateController struct {
	clientset             kubecli.KubevirtClient
	queue                 workqueue.TypedRateLimitingInterface[string]
	vmiStore              cache.Store
	podIndexer            cache.Indexer
	migrationIndexer      cache.Indexer
	recorder              record.EventRecorder
	migrationExpectations *controller.UIDTrackingControllerExpectations
	kubeVirtStore         cache.Store
	clusterConfig         *virtconfig.ClusterConfig
	launcherImage         string

	lastDeletionBatch time.Time

	hasSynced func() bool
}

type updateData struct {
	allOutdatedVMIs        []*virtv1.VirtualMachineInstance
	migratableOutdatedVMIs []*virtv1.VirtualMachineInstance
	evictOutdatedVMIs      []*virtv1.VirtualMachineInstance
	abortChangeVMIs        []*virtv1.VirtualMachineInstance

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

	rl := workqueue.NewTypedMaxOfRateLimiter[string](
		workqueue.NewTypedItemExponentialFailureRateLimiter[string](defaultThrottleInterval, 300*time.Second),
		&workqueue.TypedBucketRateLimiter[string]{Limiter: rate.NewLimiter(rate.Every(defaultThrottleInterval), 1)},
	)

	c := &WorkloadUpdateController{
		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			rl,
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-workload-update"},
		),
		vmiStore:              vmiInformer.GetStore(),
		podIndexer:            podInformer.GetIndexer(),
		migrationIndexer:      migrationInformer.GetIndexer(),
		kubeVirtStore:         kubeVirtInformer.GetStore(),
		recorder:              recorder,
		clientset:             clientset,
		launcherImage:         launcherImage,
		migrationExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		clusterConfig:         clusterConfig,
		hasSynced: func() bool {
			return migrationInformer.HasSynced() && vmiInformer.HasSynced() && podInformer.HasSynced() && kubeVirtInformer.HasSynced()
		},
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateVmi,
	})
	if err != nil {
		return nil, err
	}

	_, err = kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addKubeVirt,
		DeleteFunc: c.deleteKubeVirt,
		UpdateFunc: c.updateKubeVirt,
	})
	if err != nil {
		return nil, err
	}

	_, err = migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	kvs := c.kubeVirtStore.List()
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

	if !(isHotplugInProgress(vmi) || isVolumesUpdateInProgress(vmi) || isNodePlacementInProgress(vmi)) ||
		migrationutils.IsMigrating(vmi) {
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
	cache.WaitForCacheSync(stopCh, c.hasSynced)

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
	err := c.execute(key)

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
		condManager.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceMemoryChange, k8sv1.ConditionTrue) ||
		condManager.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceMigrationRequired, k8sv1.ConditionTrue)
}

func isVolumesUpdateInProgress(vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstanceVolumesChange, k8sv1.ConditionTrue)
}

func isNodePlacementInProgress(vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstanceNodePlacementNotMatched, k8sv1.ConditionTrue)
}

func (c *WorkloadUpdateController) doesRequireMigration(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.IsFinal() || migrationutils.IsMigrating(vmi) {
		return false
	}
	if metav1.HasAnnotation(vmi.ObjectMeta, v1.WorkloadUpdateMigrationAbortionAnnotation) {
		return false
	}
	if isHotplugInProgress(vmi) {
		return true
	}
	if isVolumesUpdateInProgress(vmi) {
		return true
	}
	if isNodePlacementInProgress(vmi) {
		return true
	}

	return false
}

func (c *WorkloadUpdateController) shouldAbortMigration(vmi *virtv1.VirtualMachineInstance) bool {
	numMig := len(migrationutils.ListWorkloadUpdateMigrations(c.migrationIndexer, vmi.Name, vmi.Namespace))
	if metav1.HasAnnotation(vmi.ObjectMeta, virtv1.WorkloadUpdateMigrationAbortionAnnotation) {
		return numMig > 0
	}
	if isHotplugInProgress(vmi) {
		return false
	}
	if isVolumesUpdateInProgress(vmi) {
		return false
	}
	if isNodePlacementInProgress(vmi) {
		return false
	}
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp != nil {
		return false
	}
	return numMig > 0
}

func (c *WorkloadUpdateController) getUpdateData(kv *virtv1.KubeVirt) *updateData {
	data := &updateData{}

	lookup := make(map[string]bool)

	migrations := migrationutils.ListUnfinishedMigrations(c.migrationIndexer)

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

	objs := c.vmiStore.List()
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		switch {
		case !vmi.IsRunning() || vmi.IsFinal() || vmi.DeletionTimestamp != nil:
			// only consider running VMIs that aren't being shutdown
			continue
		case c.shouldAbortMigration(vmi) && !c.isOutdated(vmi):
			data.abortChangeVMIs = append(data.abortChangeVMIs, vmi)
			continue
		case !c.isOutdated(vmi) && !c.doesRequireMigration(vmi):
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
		volMig := false
		errValid := volumemig.ValidateVolumesUpdateMigration(vmi, nil, vmi.Status.MigratedVolumes)
		if len(vmi.Status.MigratedVolumes) > 0 && errValid == nil {
			volMig = true
		}
		if automatedMigrationAllowed && (vmi.IsMigratable() || volMig) {
			data.migratableOutdatedVMIs = append(data.migratableOutdatedVMIs, vmi)
		} else if automatedShutdownAllowed {
			data.evictOutdatedVMIs = append(data.evictOutdatedVMIs, vmi)
		}
	}

	return data
}

func (c *WorkloadUpdateController) execute(key string) error {
	obj, exists, err := c.kubeVirtStore.GetByKey(key)

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

	metrics.SetOutdatedVirtualMachineInstanceWorkloads(len(data.allOutdatedVMIs))

	// update outdated workload count on kv
	if kv.Status.OutdatedVirtualMachineInstanceWorkloads == nil || *kv.Status.OutdatedVirtualMachineInstanceWorkloads != len(data.allOutdatedVMIs) {
		l := len(data.allOutdatedVMIs)
		kvCopy := kv.DeepCopy()
		kvCopy.Status.OutdatedVirtualMachineInstanceWorkloads = &l
		patchSet := patch.New()
		if kv.Status.OutdatedVirtualMachineInstanceWorkloads == nil {
			patchSet.AddOption(patch.WithAdd("/status/outdatedVirtualMachineInstanceWorkloads", kvCopy.Status.OutdatedVirtualMachineInstanceWorkloads))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/outdatedVirtualMachineInstanceWorkloads", kv.Status.OutdatedVirtualMachineInstanceWorkloads),
				patch.WithReplace("/status/outdatedVirtualMachineInstanceWorkloads", kvCopy.Status.OutdatedVirtualMachineInstanceWorkloads),
			)
		}
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return err
		}
		_, err = c.clientset.KubeVirt(kv.Namespace).PatchStatus(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch kubevirt obj status to update the outdatedVirtualMachineInstanceWorkloads valued: %v", err)
		}
	}

	// Rather than enqueing based on VMI activity, we keep periodically poping the loop
	// until all VMIs are updated. Watching all VMI activity is chatty for this controller
	// when we don't need to be that efficent in how quickly the updates are being processed.
	if len(data.evictOutdatedVMIs) != 0 || len(data.migratableOutdatedVMIs) != 0 || len(data.abortChangeVMIs) != 0 {
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

	wgLen := len(migrationCandidates) + len(evictionCandidates) + len(data.abortChangeVMIs)
	wg := &sync.WaitGroup{}
	wg.Add(wgLen)
	errChan := make(chan error, wgLen)

	c.migrationExpectations.ExpectCreations(key, migrateCount)
	for _, vmi := range migrationCandidates {
		go func(vmi *virtv1.VirtualMachineInstance) {
			var labels map[string]string
			if isVolumesUpdateInProgress(vmi) {
				labels = make(map[string]string)
				labels[virtv1.VolumesUpdateMigration] = vmi.Name
				if len(vmi.Name) > k8svalidation.DNS1035LabelMaxLength {
					// Labels are limited to 63 characters, fall back to UID, remain backwards compatible otherwise
					labels[virtv1.VolumesUpdateMigration] = string(vmi.UID)
				}
			}
			defer wg.Done()
			createdMigration, err := c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Create(context.Background(), &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						virtv1.WorkloadUpdateMigrationAnnotation: "",
					},
					Labels:       labels,
					GenerateName: "kubevirt-workload-update-",
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
			}, metav1.CreateOptions{})
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

			pod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
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

	for _, vmi := range data.abortChangeVMIs {
		go func(vmi *virtv1.VirtualMachineInstance) {
			defer wg.Done()
			migList := migrationutils.ListWorkloadUpdateMigrations(c.migrationIndexer, vmi.Name, vmi.Namespace)
			for _, mig := range migList {
				err = c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Delete(context.Background(), mig.Name, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					log.Log.Object(vmi).Reason(err).Errorf("Failed to delete the migration due to a migration abortion")
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, FailedChangeAbortionReason, "Failed to abort change for vmi: %s: %v", vmi.Name, err)
					errChan <- err
				} else if err == nil {
					log.Log.Infof("Delete migration %s due to an update change abortion", mig.Name)
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulChangeAbortionReason, "Aborted change for vmi: %s", vmi.Name)

				}
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
