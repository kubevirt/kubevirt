/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opencontainers/selinux/go-selinux"

	"kubevirt.io/api/migrations/v1alpha1"

	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/pdbs"
	"kubevirt.io/kubevirt/pkg/util/status"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/pkg/util/migrations"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const (
	failedToProcessDeleteNotificationErrMsg   = "Failed to process delete notification"
	successfulUpdatePodDisruptionBudgetReason = "SuccessfulUpdate"
	failedUpdatePodDisruptionBudgetReason     = "FailedUpdate"
	failedGetAttractionPodsFmt                = "failed to get attachment pods: %v"
)

// This is the timeout used when a target pod is stuck in
// a pending unschedulable state.
const defaultUnschedulablePendingTimeoutSeconds = int64(60 * 5)

// This is how many finalized migration objects left in
// the system before we begin garbage collecting the oldest
// migration objects
const defaultFinalizedMigrationGarbageCollectionBuffer = 5

// This is catch all timeout used when a target pod is stuck in
// a in the pending phase for any reason. The theory behind this timeout
// being longer than the unschedulable timeout is that we don't necessarily
// know all the reasons a pod will be stuck in pending for an extended
// period of time, so we want to make this timeout long enough that it doesn't
// cause the migration to fail when it could have reasonably succeeded.
const defaultCatchAllPendingTimeoutSeconds = int64(60 * 15)

var migrationBackoffError = errors.New(MigrationBackoffReason)

type MigrationController struct {
	templateService         services.TemplateService
	clientset               kubecli.KubevirtClient
	Queue                   workqueue.RateLimitingInterface
	vmiInformer             cache.SharedIndexInformer
	podInformer             cache.SharedIndexInformer
	migrationInformer       cache.SharedIndexInformer
	nodeInformer            cache.SharedIndexInformer
	pvcInformer             cache.SharedIndexInformer
	pdbInformer             cache.SharedIndexInformer
	migrationPolicyInformer cache.SharedIndexInformer
	resourceQuotaInformer   cache.SharedIndexInformer
	recorder                record.EventRecorder
	podExpectations         *controller.UIDTrackingControllerExpectations
	migrationStartLock      *sync.Mutex
	clusterConfig           *virtconfig.ClusterConfig
	statusUpdater           *status.MigrationStatusUpdater

	// the set of cancelled migrations before being handed off to virt-handler.
	// the map keys are migration keys
	handOffLock sync.Mutex
	handOffMap  map[string]struct{}

	unschedulablePendingTimeoutSeconds int64
	catchAllPendingTimeoutSeconds      int64
}

func NewMigrationController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	nodeInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	pdbInformer cache.SharedIndexInformer,
	migrationPolicyInformer cache.SharedIndexInformer,
	resourceQuotaInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
) (*MigrationController, error) {

	c := &MigrationController{
		templateService:         templateService,
		Queue:                   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-migration"),
		vmiInformer:             vmiInformer,
		podInformer:             podInformer,
		migrationInformer:       migrationInformer,
		nodeInformer:            nodeInformer,
		pvcInformer:             pvcInformer,
		pdbInformer:             pdbInformer,
		resourceQuotaInformer:   resourceQuotaInformer,
		migrationPolicyInformer: migrationPolicyInformer,
		recorder:                recorder,
		clientset:               clientset,
		podExpectations:         controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		migrationStartLock:      &sync.Mutex{},
		clusterConfig:           clusterConfig,
		statusUpdater:           status.NewMigrationStatusUpdater(clientset),
		handOffMap:              make(map[string]struct{}),

		unschedulablePendingTimeoutSeconds: defaultUnschedulablePendingTimeoutSeconds,
		catchAllPendingTimeoutSeconds:      defaultCatchAllPendingTimeoutSeconds,
	}

	_, err := c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMI,
		DeleteFunc: c.deleteVMI,
		UpdateFunc: c.updateVMI,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.deletePod,
		UpdateFunc: c.updatePod,
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

	_, err = c.pdbInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updatePDB,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.resourceQuotaInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateResourceQuota,
		DeleteFunc: c.deleteResourceQuota,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *MigrationController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting migration controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.migrationInformer.HasSynced, c.pdbInformer.HasSynced, c.resourceQuotaInformer.HasSynced)
	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping migration controller.")
}

func (c *MigrationController) runWorker() {
	for c.Execute() {
	}
}

func (c *MigrationController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing Migration %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed Migration %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func ensureSelectorLabelPresent(migration *virtv1.VirtualMachineInstanceMigration) {
	if migration.Labels == nil {
		migration.Labels = map[string]string{virtv1.MigrationSelectorLabel: migration.Spec.VMIName}
	} else if _, exist := migration.Labels[virtv1.MigrationSelectorLabel]; !exist {
		migration.Labels[virtv1.MigrationSelectorLabel] = migration.Spec.VMIName
	}
}

func (c *MigrationController) patchVMI(origVMI, newVMI *virtv1.VirtualMachineInstance) error {
	var ops []string

	if !equality.Semantic.DeepEqual(origVMI.Status.MigrationState, newVMI.Status.MigrationState) {
		newState, err := json.Marshal(newVMI.Status.MigrationState)
		if err != nil {
			return err
		}
		if origVMI.Status.MigrationState == nil {
			ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/status/migrationState", "value": %s }`, string(newState)))

		} else {
			oldState, err := json.Marshal(origVMI.Status.MigrationState)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/status/migrationState", "value": %s }`, string(oldState)))
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/status/migrationState", "value": %s }`, string(newState)))
		}
	}

	if !equality.Semantic.DeepEqual(origVMI.Labels, newVMI.Labels) {
		newLabels, err := json.Marshal(newVMI.Labels)
		if err != nil {
			return err
		}
		oldLabels, err := json.Marshal(origVMI.Labels)
		if err != nil {
			return err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s }`, string(oldLabels)))
		ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s }`, string(newLabels)))
	}

	if len(ops) > 0 {
		_, err := c.clientset.VirtualMachineInstance(origVMI.Namespace).Patch(context.Background(), origVMI.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MigrationController) execute(key string) error {
	var vmi *virtv1.VirtualMachineInstance
	var targetPods []*k8sv1.Pod

	// Fetch the latest state from cache
	obj, exists, err := c.migrationInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		c.podExpectations.DeleteExpectations(key)
		c.removeHandOffKey(key)
		return nil
	}
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)
	logger := log.Log.Object(migration)

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(migration) {
		migration := migration.DeepCopy()
		controller.SetLatestApiVersionAnnotation(migration)
		// Ensure the migration contains our selector label
		ensureSelectorLabelPresent(migration)
		_, err = c.clientset.VirtualMachineInstanceMigration(migration.Namespace).Update(migration)
		return err
	}

	vmiObj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", migration.Namespace, migration.Spec.VMIName))
	if err != nil {
		return err
	}

	if !vmiExists {
		var err error

		if migration.DeletionTimestamp == nil {
			logger.V(3).Infof("Deleting migration for deleted vmi %s/%s", migration.Namespace, migration.Spec.VMIName)
			err = c.clientset.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &v1.DeleteOptions{})
		}
		// nothing to process for a migration that's being deleted
		return err
	}

	vmi = vmiObj.(*virtv1.VirtualMachineInstance)
	targetPods, err = c.listMatchingTargetPods(migration, vmi)
	if err != nil {
		return err
	}

	needsSync := c.podExpectations.SatisfiedExpectations(key) && vmiExists

	logger.V(4).Infof("processing migration: needsSync %t, hasVMI %t, targetPod len %d", needsSync, vmiExists, len(targetPods))

	var syncErr error

	if needsSync {
		syncErr = c.sync(key, migration, vmi, targetPods)
	}

	err = c.updateStatus(migration, vmi, targetPods, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil {
		return syncErr
	}

	if migration.IsFinal() {
		err = c.garbageCollectFinalizedMigrations(vmi)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *MigrationController) canMigrateVMI(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) (bool, error) {

	if vmi.Status.MigrationState == nil {
		return true, nil
	} else if vmi.Status.MigrationState.MigrationUID == migration.UID {
		return true, nil
	} else if vmi.Status.MigrationState.MigrationUID == "" {
		return true, nil
	}

	curMigrationUID := vmi.Status.MigrationState.MigrationUID

	// check to see if the curMigrationUID still exists or is finalized
	objs, err := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, migration.Namespace)

	if err != nil {
		return false, err
	}
	for _, obj := range objs {
		curMigration := obj.(*virtv1.VirtualMachineInstanceMigration)
		if curMigration.UID != curMigrationUID {
			continue
		}

		if curMigration.IsFinal() {
			// If the other job already completed, it's okay to take over the migration.
			return true, nil
		}
		return false, nil
	}

	return true, nil

}

func (c *MigrationController) updateStatus(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod, syncError error) error {

	var pod *k8sv1.Pod = nil
	var attachmentPod *k8sv1.Pod = nil
	conditionManager := controller.NewVirtualMachineInstanceMigrationConditionManager()
	vmiConditionManager := controller.NewVirtualMachineInstanceConditionManager()
	migrationCopy := migration.DeepCopy()

	podExists, attachmentPodExists := len(pods) > 0, false
	if podExists {
		pod = pods[0]

		if attachmentPods, err := controller.AttachmentPods(pod, c.podInformer); err != nil {
			return fmt.Errorf(failedGetAttractionPodsFmt, err)
		} else {
			attachmentPodExists = len(attachmentPods) > 0
			if attachmentPodExists {
				attachmentPod = attachmentPods[0]
			}
		}
	}

	// Remove the finalizer and conditions if the migration has already completed
	if migration.IsFinal() {
		// store the finalized migration state data from the VMI status in the migration object
		migrationCopy.Status.MigrationState = vmi.Status.MigrationState

		// remove the migration finalizaer
		controller.RemoveFinalizer(migrationCopy, virtv1.VirtualMachineInstanceMigrationFinalizer)

		// Status checking of active Migration job.
		//
		// 1. Fail if VMI isn't in running state.
		// 2. Fail if target pod exists and has gone down for any reason.
		// 3. Begin progressing migration state based on VMI's MigrationState status.
	} else if vmi == nil {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed because vmi does not exist.")
		log.Log.Object(migration).Error("vmi does not exist")
	} else if vmi.IsFinal() {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed vmi shutdown during migration.")
		log.Log.Object(migration).Error("Unable to migrate vmi because vmi is shutdown.")
	} else if migration.DeletionTimestamp != nil && !c.isMigrationHandedOff(migration, vmi) {
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed due to being canceled")
		if !conditionManager.HasCondition(migration, virtv1.VirtualMachineInstanceMigrationAbortRequested) {
			condition := virtv1.VirtualMachineInstanceMigrationCondition{
				Type:          virtv1.VirtualMachineInstanceMigrationAbortRequested,
				Status:        k8sv1.ConditionTrue,
				LastProbeTime: v1.Now(),
			}
			migrationCopy.Status.Conditions = append(migrationCopy.Status.Conditions, condition)
		}
		migrationCopy.Status.Phase = virtv1.MigrationFailed
	} else if podExists && podIsDown(pod) {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed because target pod shutdown during migration")
		log.Log.Object(migration).Errorf("target pod %s/%s shutdown during migration", pod.Namespace, pod.Name)
	} else if migration.TargetIsCreated() && !podExists {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration target pod was removed during active migration.")
		log.Log.Object(migration).Error("target pod disappeared during migration")
	} else if migration.TargetIsHandedOff() && vmi.Status.MigrationState == nil {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "VMI's migration state was cleared during the active migration.")
		log.Log.Object(migration).Error("vmi migration state cleared during migration")
	} else if migration.TargetIsHandedOff() &&
		vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.MigrationUID != migration.UID {

		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "VMI's migration state was taken over by another migration job during active migration.")
		log.Log.Object(migration).Error("vmi's migration state was taken over by another migration object")
	} else if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.MigrationUID == migration.UID &&
		vmi.Status.MigrationState.Failed {

		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "source node reported migration failed")
		log.Log.Object(migration).Errorf("VMI %s/%s reported migration failed", vmi.Namespace, vmi.Name)

	} else if migration.DeletionTimestamp != nil && !migration.IsFinal() &&
		!conditionManager.HasCondition(migration, virtv1.VirtualMachineInstanceMigrationAbortRequested) {
		condition := virtv1.VirtualMachineInstanceMigrationCondition{
			Type:          virtv1.VirtualMachineInstanceMigrationAbortRequested,
			Status:        k8sv1.ConditionTrue,
			LastProbeTime: v1.Now(),
		}
		migrationCopy.Status.Conditions = append(migrationCopy.Status.Conditions, condition)
	} else if attachmentPodExists && podIsDown(attachmentPod) {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed because target attachment pod shutdown during migration")
		log.Log.Object(migration).Errorf("target attachment pod %s/%s shutdown during migration", attachmentPod.Namespace, attachmentPod.Name)
	} else {

		switch migration.Status.Phase {
		case virtv1.MigrationPhaseUnset:
			canMigrate, err := c.canMigrateVMI(migration, vmi)
			if err != nil {
				return err
			}

			if canMigrate {
				migrationCopy.Status.Phase = virtv1.MigrationPending
			} else {
				// can not migrate because there is an active migration already
				// in progress for this VMI.
				migrationCopy.Status.Phase = virtv1.MigrationFailed
				c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "VMI is not eligible for migration because another migration job is in progress.")
				log.Log.Object(migration).Error("Migration object ont eligible for migration because another job is in progress")
			}
		case virtv1.MigrationPending:
			if podExists {
				if controller.VMIHasHotplugVolumes(vmi) {
					if attachmentPodExists {
						migrationCopy.Status.Phase = virtv1.MigrationScheduling
					}
				} else {
					migrationCopy.Status.Phase = virtv1.MigrationScheduling
				}
			} else if syncError != nil && strings.Contains(syncError.Error(), "exceeded quota") && !conditionManager.HasCondition(migration, virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota) {
				condition := virtv1.VirtualMachineInstanceMigrationCondition{
					Type:          virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota,
					Status:        k8sv1.ConditionTrue,
					LastProbeTime: v1.Now(),
				}
				migrationCopy.Status.Conditions = append(migrationCopy.Status.Conditions, condition)
			}
		case virtv1.MigrationScheduling:
			if conditionManager.HasCondition(migrationCopy, virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota) {
				conditionManager.RemoveCondition(migrationCopy, virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota)
			}
			if isPodReady(pod) {
				if controller.VMIHasHotplugVolumes(vmi) {
					if attachmentPodExists && isPodReady(attachmentPod) {
						log.Log.Object(migration).Infof("Attachment pod %s for vmi %s/%s is ready", attachmentPod.Name, vmi.Namespace, vmi.Name)
						migrationCopy.Status.Phase = virtv1.MigrationScheduled
					}
				} else {
					migrationCopy.Status.Phase = virtv1.MigrationScheduled
				}
			}
		case virtv1.MigrationScheduled:
			if vmi.Status.MigrationState != nil &&
				vmi.Status.MigrationState.MigrationUID == migration.UID &&
				vmi.Status.MigrationState.TargetNode != "" {
				migrationCopy.Status.Phase = virtv1.MigrationPreparingTarget
			}
		case virtv1.MigrationPreparingTarget:
			if vmi.Status.MigrationState.TargetNode != "" && vmi.Status.MigrationState.TargetNodeAddress != "" {
				migrationCopy.Status.Phase = virtv1.MigrationTargetReady
			}
		case virtv1.MigrationTargetReady:
			if vmi.Status.MigrationState.StartTimestamp != nil {
				migrationCopy.Status.Phase = virtv1.MigrationRunning
			}
		case virtv1.MigrationRunning:
			_, exists := pod.Annotations[virtv1.MigrationTargetReadyTimestamp]
			if !exists && vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp != nil {
				key := patch.EscapeJSONPointer(virtv1.MigrationTargetReadyTimestamp)
				patchOps := fmt.Sprintf(`[{ "op": "add", "path": "/metadata/annotations/%s", "value": "%s" }]`,
					key,
					vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp.String())

				_, err := c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, types.JSONPatchType, []byte(patchOps), v1.PatchOptions{})
				if err != nil {
					return err
				}
			}

			if vmi.Status.MigrationState.Completed &&
				!vmiConditionManager.HasCondition(vmi, virtv1.VirtualMachineInstanceVCPUChange) &&
				!vmiConditionManager.HasCondition(vmi, virtv1.VirtualMachineInstanceMemoryChange) {
				migrationCopy.Status.Phase = virtv1.MigrationSucceeded
				c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulMigrationReason, "Source node reported migration succeeded")
				log.Log.Object(migration).Infof("VMI reported migration succeeded.")
			}
		}
	}

	controller.SetVMIMigrationPhaseTransitionTimestamp(migration, migrationCopy)

	if !equality.Semantic.DeepEqual(migration.Status, migrationCopy.Status) {
		err := c.statusUpdater.UpdateStatus(migrationCopy)
		if err != nil {
			return err
		}

	} else if !equality.Semantic.DeepEqual(migration.Finalizers, migrationCopy.Finalizers) {
		_, err := c.clientset.VirtualMachineInstanceMigration(migrationCopy.Namespace).Update(migrationCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func setTargetPodSELinuxLevel(pod *k8sv1.Pod, vmiSeContext string) error {
	// The target pod may share resources with the sources pod (RWX disks for example)
	// Therefore, it needs to share the same SELinux categories to inherit the same permissions
	// Note: there is a small probablility that the target pod will share the same categories as another pod on its node.
	//   It is a slight security concern, but not as bad as removing categories on all shared objects for the duration of the migration.
	if vmiSeContext == "none" {
		// The SelinuxContext is explicitly set to "none" when SELinux is not present
		return nil
	}
	if vmiSeContext == "" {
		return fmt.Errorf("SELinux context not set on VMI status")
	} else {
		seContext, err := selinux.NewContext(vmiSeContext)
		if err != nil {
			return err
		}
		level, exists := seContext["level"]
		if exists && level != "" {
			// The SELinux context looks like "system_u:object_r:container_file_t:s0:c1,c2", we care about "s0:c1,c2"
			if pod.Spec.SecurityContext == nil {
				pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
			}
			pod.Spec.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{
				Level: level,
			}
		}
	}

	return nil
}

func (c *MigrationController) createTargetPod(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, sourcePod *k8sv1.Pod) error {
	templatePod, err := c.templateService.RenderMigrationManifest(vmi, sourcePod)
	if err != nil {
		return fmt.Errorf("failed to render launch manifest: %v", err)
	}

	antiAffinityTerm := k8sv1.PodAffinityTerm{
		LabelSelector: &v1.LabelSelector{
			MatchLabels: map[string]string{
				virtv1.CreatedByLabel: string(vmi.UID),
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	}
	antiAffinityRule := &k8sv1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{antiAffinityTerm},
	}

	if templatePod.Spec.Affinity == nil {
		templatePod.Spec.Affinity = &k8sv1.Affinity{
			PodAntiAffinity: antiAffinityRule,
		}
	} else if templatePod.Spec.Affinity.PodAntiAffinity == nil {
		templatePod.Spec.Affinity.PodAntiAffinity = antiAffinityRule
	} else {
		templatePod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(templatePod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, antiAffinityTerm)
	}

	templatePod.ObjectMeta.Labels[virtv1.MigrationJobLabel] = string(migration.UID)
	templatePod.ObjectMeta.Annotations[virtv1.MigrationJobNameAnnotation] = migration.Name

	// If cpu model is "host model" allow migration only to nodes that supports this cpu model
	if cpu := vmi.Spec.Domain.CPU; cpu != nil && cpu.Model == virtv1.CPUModeHostModel {
		node, err := c.getNodeForVMI(vmi)

		if err != nil {
			return err
		}

		err = prepareNodeSelectorForHostCpuModel(node, templatePod, sourcePod)
		if err != nil {
			return err
		}
	}

	matchLevelOnTarget := c.clusterConfig.GetMigrationConfiguration().MatchSELinuxLevelOnMigration
	if matchLevelOnTarget == nil || *matchLevelOnTarget {
		err = setTargetPodSELinuxLevel(templatePod, vmi.Status.SelinuxContext)
		if err != nil {
			return err
		}
	}

	// This is used by the functional test to simulate failures
	computeImageOverride, ok := migration.Annotations[virtv1.FuncTestMigrationTargetImageOverrideAnnotation]
	if ok && computeImageOverride != "" {
		for i, container := range templatePod.Spec.Containers {
			if container.Name == "compute" {
				container.Image = computeImageOverride
				templatePod.Spec.Containers[i] = container
				break
			}
		}
	}

	key := controller.MigrationKey(migration)
	c.podExpectations.ExpectCreations(key, 1)
	pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), templatePod, v1.CreateOptions{})
	if err != nil {
		if k8serrors.IsForbidden(err) && strings.Contains(err.Error(), "violates PodSecurity") {
			err = fmt.Errorf("failed to create target pod for vmi %s/%s, it needs a privileged namespace to run: %w", vmi.GetNamespace(), vmi.GetName(), err)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, failedToRenderLaunchManifestErrFormat, err)

		} else {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
			err = fmt.Errorf("failed to create vmi migration target pod: %v", err)
		}
		c.podExpectations.CreationObserved(key)
		return err
	}
	log.Log.Object(vmi).Infof("Created migration target pod %s/%s with uuid %s for migration %s with uuid %s", pod.Namespace, pod.Name, string(pod.UID), migration.Name, string(migration.UID))
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created migration target pod %s", pod.Name)
	return nil
}

func (c *MigrationController) expandPDB(pdb *policyv1.PodDisruptionBudget, vmi *virtv1.VirtualMachineInstance, vmim *virtv1.VirtualMachineInstanceMigration) error {
	minAvailable := 2

	if pdb.Spec.MinAvailable.IntValue() == minAvailable && pdb.Labels[virtv1.MigrationNameLabel] == vmim.Name {
		log.Log.V(4).Object(vmi).Infof("PDB has been already expanded")
		return nil
	}

	patchBytes := []byte(fmt.Sprintf(`{"spec":{"minAvailable": %d},"metadata":{"labels":{"%s": "%s"}}}`, minAvailable, virtv1.MigrationNameLabel, vmim.Name))

	_, err := c.clientset.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Patch(context.Background(), pdb.Name, types.StrategicMergePatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, failedUpdatePodDisruptionBudgetReason, "Error expanding the PodDisruptionBudget %s: %v", pdb.Name, err)
		return err
	}
	log.Log.Object(vmi).Infof("expanding pdb for VMI %s/%s to protect migration %s", vmi.Namespace, vmi.Name, vmim.Name)
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, successfulUpdatePodDisruptionBudgetReason, "Expanded PodDisruptionBudget %s", pdb.Name)
	return nil
}

// handleMigrationBackoff introduce a backoff (when needed) only for migrations
// created by the evacuation controller.
func (c *MigrationController) handleMigrationBackoff(key string, vmi *virtv1.VirtualMachineInstance, migration *virtv1.VirtualMachineInstanceMigration) error {
	if _, exists := migration.Annotations[virtv1.FuncTestForceIgnoreMigrationBackoffAnnotation]; exists {
		return nil
	}
	if _, exists := migration.Annotations[virtv1.EvacuationMigrationAnnotation]; !exists {
		return nil
	}

	migrations, err := c.listEvacuationMigrations(vmi.Namespace, vmi.Name)
	if err != nil {
		return err
	}
	if len(migrations) < 2 {
		return nil
	}

	// Newest first
	sort.Sort(sort.Reverse(vmimCollection(migrations)))
	if migrations[0].UID != migration.UID {
		return nil
	}

	backoff := time.Second * 0
	for _, m := range migrations[1:] {
		if m.Status.Phase == virtv1.MigrationSucceeded {
			break
		}
		if m.DeletionTimestamp != nil {
			continue
		}

		if m.Status.Phase == virtv1.MigrationFailed {
			if backoff == 0 {
				backoff = time.Second * 20
			} else {
				backoff = backoff * 2
			}
		}
	}
	if backoff == 0 {
		return nil
	}

	getFailedTS := func(migration *virtv1.VirtualMachineInstanceMigration) metav1.Time {
		for _, ts := range migration.Status.PhaseTransitionTimestamps {
			if ts.Phase == virtv1.MigrationFailed {
				return ts.PhaseTransitionTimestamp
			}
		}
		return metav1.Time{}
	}

	outOffBackoffTS := getFailedTS(migrations[1]).Add(backoff)
	backoff = outOffBackoffTS.Sub(time.Now())

	if backoff > 0 {
		log.Log.Object(vmi).Errorf("vmi in migration backoff, re-enqueueing after %v", backoff)
		c.Queue.AddAfter(key, backoff)
		return migrationBackoffError
	}
	return nil
}

func (c *MigrationController) handleMarkMigrationFailedOnVMI(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) error {

	// Mark Migration Done on VMI if virt handler never started it.
	// Once virt-handler starts the migration, it's up to handler
	// to finalize it.

	vmiCopy := vmi.DeepCopy()

	now := v1.NewTime(time.Now())
	vmiCopy.Status.MigrationState.StartTimestamp = &now
	vmiCopy.Status.MigrationState.EndTimestamp = &now
	vmiCopy.Status.MigrationState.Failed = true
	vmiCopy.Status.MigrationState.Completed = true

	err := c.patchVMI(vmi, vmiCopy)
	if err != nil {
		log.Log.Reason(err).Object(vmi).Errorf("Failed to patch VMI status to indicate migration %s/%s failed.", migration.Namespace, migration.Name)
		return err
	}
	log.Log.Object(vmi).Infof("Marked Migration %s/%s failed on vmi due to target pod disappearing before migration kicked off.", migration.Namespace, migration.Name)
	c.recorder.Event(vmi, k8sv1.EventTypeWarning, FailedMigrationReason, fmt.Sprintf("VirtualMachineInstance migration uid %s failed. reason: target pod is down", string(migration.UID)))

	return nil
}

func (c *MigrationController) handlePreHandoffMigrationCancel(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	if pod == nil {
		return nil
	}

	c.podExpectations.ExpectDeletions(controller.MigrationKey(migration), []string{controller.PodKey(pod)})
	err := c.clientset.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, v1.DeleteOptions{})
	if err != nil {
		c.podExpectations.DeletionObserved(controller.MigrationKey(migration), controller.PodKey(pod))
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedDeletePodReason, "Error deleting canceled migration target pod: %v", err)
		return fmt.Errorf("cannot delete pending target pod %s/%s for migration although migration is aborted", pod.Name, pod.Namespace)
	}

	reason := "migration canceled"
	log.Log.Object(vmi).Infof("Deleted pending migration target pod %s/%s with uuid %s for migration %s with uuid %s with reason [%s]", pod.Namespace, pod.Name, string(pod.UID), migration.Name, string(migration.UID), reason)
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, reason, pod.Name)
	return nil
}

func (c *MigrationController) handleTargetPodHandoff(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {

	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.MigrationUID == migration.UID {
		// already handed off
		return nil
	}

	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
		MigrationUID: migration.UID,
		TargetNode:   pod.Spec.NodeName,
		SourceNode:   vmi.Status.NodeName,
		TargetPod:    pod.Name,
	}

	// By setting this label, virt-handler on the target node will receive
	// the vmi and prepare the local environment for the migration
	vmiCopy.ObjectMeta.Labels[virtv1.MigrationTargetNodeNameLabel] = pod.Spec.NodeName

	if controller.VMIHasHotplugVolumes(vmiCopy) {
		attachmentPods, err := controller.AttachmentPods(pod, c.podInformer)
		if err != nil {
			return fmt.Errorf(failedGetAttractionPodsFmt, err)
		}
		if len(attachmentPods) > 0 {
			log.Log.Object(migration).Infof("Target attachment pod for vmi %s/%s: %s", vmiCopy.Namespace, vmiCopy.Name, string(attachmentPods[0].UID))
			vmiCopy.Status.MigrationState.TargetAttachmentPodUID = attachmentPods[0].UID
		} else {
			return fmt.Errorf("target attachment pod not found")
		}
	}

	clusterMigrationConfigs := c.clusterConfig.GetMigrationConfiguration().DeepCopy()
	err := c.matchMigrationPolicy(vmiCopy, clusterMigrationConfigs)
	if err != nil {
		return fmt.Errorf("failed to match migration policy: %v", err)
	}

	if !c.isMigrationPolicyMatched(vmiCopy) {
		vmiCopy.Status.MigrationState.MigrationConfiguration = clusterMigrationConfigs
	}

	if controller.VMIHasHotplugCPU(vmi) && vmi.IsCPUDedicated() {
		cpuLimitsCount, err := getTargetPodLimitsCount(pod)
		if err != nil {
			return err
		}
		vmiCopy.ObjectMeta.Labels[virtv1.VirtualMachinePodCPULimitsLabel] = strconv.Itoa(int(cpuLimitsCount))
	}

	if controller.VMIHasHotplugMemory(vmi) {
		memoryReq, err := getTargetPodMemoryRequests(pod)
		if err != nil {
			return err
		}
		vmiCopy.ObjectMeta.Labels[virtv1.VirtualMachinePodMemoryRequestsLabel] = memoryReq
	}

	err = c.patchVMI(vmi, vmiCopy)
	if err != nil {
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedHandOverPodReason, fmt.Sprintf("Failed to set MigrationStat in VMI status. :%v", err))
		return err
	}

	c.addHandOffKey(controller.MigrationKey(migration))
	log.Log.Object(vmi).Infof("Handed off migration %s/%s to target virt-handler.", migration.Namespace, migration.Name)
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulHandOverPodReason, "Migration target pod is ready for preparation by virt-handler.")
	return nil
}

func (c *MigrationController) markMigrationAbortInVmiStatus(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) error {

	if vmi.Status.MigrationState == nil {
		return fmt.Errorf("migration state is nil when trying to mark migratio abortion in vmi status")
	}

	vmiCopy := vmi.DeepCopy()

	vmiCopy.Status.MigrationState.AbortRequested = true
	if !equality.Semantic.DeepEqual(vmi.Status, vmiCopy.Status) {

		newStatus := vmiCopy.Status
		oldStatus := vmi.Status
		patchBytes, err := patch.GenerateTestReplacePatch("/status", oldStatus, newStatus)
		if err != nil {
			return err
		}

		_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, &v1.PatchOptions{})
		if err != nil {
			msg := fmt.Sprintf("failed to set MigrationState in VMI status. :%v", err)
			c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedAbortMigrationReason, msg)
			return fmt.Errorf(msg)
		}
		log.Log.Object(vmi).Infof("Signaled migration %s/%s to be aborted.", migration.Namespace, migration.Name)
		c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulAbortMigrationReason, "Migration is ready to be canceled by virt-handler.")
	}

	return nil
}

func isMigrationProtected(pdb *policyv1.PodDisruptionBudget) bool {
	return pdb.Status.DesiredHealthy == 2 && pdb.Generation == pdb.Status.ObservedGeneration
}

func filterOutOldPDBs(pdbList []*policyv1.PodDisruptionBudget) []*policyv1.PodDisruptionBudget {
	var filteredPdbs []*policyv1.PodDisruptionBudget

	for i := range pdbList {
		if !pdbs.IsPDBFromOldMigrationController(pdbList[i]) {
			filteredPdbs = append(filteredPdbs, pdbList[i])
		}
	}
	return filteredPdbs
}

func (c *MigrationController) handleTargetPodCreation(key string, migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, sourcePod *k8sv1.Pod) error {

	c.migrationStartLock.Lock()
	defer c.migrationStartLock.Unlock()

	// Don't start new migrations if we wait for cache updates on migration target pods
	if c.podExpectations.AllPendingCreations() > 0 {
		c.Queue.AddAfter(key, 1*time.Second)
		return nil
	} else if controller.VMIActivePodsCount(vmi, c.podInformer) > 1 {
		log.Log.Object(migration).Infof("Waiting to schedule target pod for migration because there are already multiple pods running for vmi %s/%s", vmi.Namespace, vmi.Name)
		c.Queue.AddAfter(key, 1*time.Second)
		return nil

	}

	// Don't start new migrations if we wait for migration object updates because of new target pods
	runningMigrations, err := c.findRunningMigrations()
	if err != nil {
		return fmt.Errorf("failed to determin the number of running migrations: %v", err)
	}

	// XXX: Make this configurable, think about limit per node, bandwidth per migration, and so on.
	if len(runningMigrations) >= int(*c.clusterConfig.GetMigrationConfiguration().ParallelMigrationsPerCluster) {
		log.Log.Object(migration).Infof("Waiting to schedule target pod for vmi [%s/%s] migration because total running parallel migration count [%d] is currently at the global cluster limit.", vmi.Namespace, vmi.Name, len(runningMigrations))
		// Let's wait until some migrations are done
		c.Queue.AddAfter(key, time.Second*5)
		return nil
	}

	outboundMigrations, err := c.outboundMigrationsOnNode(vmi.Status.NodeName, runningMigrations)

	if err != nil {
		return err
	}

	if outboundMigrations >= int(*c.clusterConfig.GetMigrationConfiguration().ParallelOutboundMigrationsPerNode) {
		// Let's ensure that we only have two outbound migrations per node
		// XXX: Make this configurable, thinkg about inbound migration limit, bandwidh per migration, and so on.
		log.Log.Object(migration).Infof("Waiting to schedule target pod for vmi [%s/%s] migration because total running parallel outbound migrations on target node [%d] has hit outbound migrations per node limit.", vmi.Namespace, vmi.Name, outboundMigrations)
		c.Queue.AddAfter(key, time.Second*5)
		return nil
	}

	// migration was accepted into the system, now see if we
	// should create the target pod
	if vmi.IsRunning() {
		if migrations.VMIMigratableOnEviction(c.clusterConfig, vmi) {
			pdbs, err := pdbs.PDBsForVMI(vmi, c.pdbInformer)
			if err != nil {
				return err
			}
			// removes pdbs from old implementation from list.
			pdbs = filterOutOldPDBs(pdbs)

			if len(pdbs) < 1 {
				log.Log.Object(vmi).Errorf("Found no PDB protecting the vmi")
				return fmt.Errorf("Found no PDB protecting the vmi %s", vmi.Name)
			}
			pdb := pdbs[0]

			if err := c.expandPDB(pdb, vmi, migration); err != nil {
				return err
			}

			// before proceeding we have to check that the k8s pdb controller has processed
			// the pdb expansion and is actually protecting the VMI migration
			if !isMigrationProtected(pdb) {
				log.Log.V(4).Object(migration).Infof("Waiting for the pdb-controller to protect the migration pods, postponing migration start")
				return nil
			}
		}
		return c.createTargetPod(migration, vmi, sourcePod)
	}
	return nil
}

func (c *MigrationController) createAttachmentPod(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) error {
	sourcePod, err := controller.CurrentVMIPod(vmi, c.podInformer)
	if err != nil {
		return fmt.Errorf("failed to get current VMI pod: %v", err)
	}

	volumes := getHotplugVolumes(vmi, sourcePod)

	volumeNamesPVCMap, err := storagetypes.VirtVolumesToPVCMap(volumes, c.pvcInformer.GetStore(), virtLauncherPod.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get PVC map: %v", err)
	}

	// Reset the hotplug volume statuses to enforce mount
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.VolumeStatus = []virtv1.VolumeStatus{}
	attachmentPodTemplate, err := c.templateService.RenderHotplugAttachmentPodTemplate(volumes, virtLauncherPod, vmiCopy, volumeNamesPVCMap, false)
	if err != nil {
		return fmt.Errorf("failed to render attachment pod template: %v", err)
	}

	if attachmentPodTemplate.ObjectMeta.Labels == nil {
		attachmentPodTemplate.ObjectMeta.Labels = make(map[string]string)
	}

	if attachmentPodTemplate.ObjectMeta.Annotations == nil {
		attachmentPodTemplate.ObjectMeta.Annotations = make(map[string]string)
	}

	attachmentPodTemplate.ObjectMeta.Labels[virtv1.MigrationJobLabel] = string(migration.UID)
	attachmentPodTemplate.ObjectMeta.Annotations[virtv1.MigrationJobNameAnnotation] = migration.Name

	key := controller.MigrationKey(migration)
	c.podExpectations.ExpectCreations(key, 1)

	attachmentPod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), attachmentPodTemplate, v1.CreateOptions{})
	if err != nil {
		c.podExpectations.CreationObserved(key)
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating attachment pod: %v", err)
		return fmt.Errorf("failed to create attachment pod: %v", err)
	}
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created attachment pod %s", attachmentPod.Name)
	return nil
}

func isPodPendingUnschedulable(pod *k8sv1.Pod) bool {

	if pod.Status.Phase != k8sv1.PodPending || pod.DeletionTimestamp != nil {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == k8sv1.PodScheduled &&
			condition.Status == k8sv1.ConditionFalse &&
			condition.Reason == k8sv1.PodReasonUnschedulable {

			return true
		}
	}
	return false
}

func timeSinceCreationSeconds(objectMeta *metav1.ObjectMeta) int64 {

	now := time.Now().UTC().Unix()
	creationTime := objectMeta.CreationTimestamp.Time.UTC().Unix()
	seconds := now - creationTime
	if seconds < 0 {
		seconds = 0
	}

	return seconds
}

func (c *MigrationController) deleteTimedOutTargetPod(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, reason string) error {

	migrationKey, err := controller.KeyFunc(migration)
	if err != nil {
		return err
	}

	c.podExpectations.ExpectDeletions(migrationKey, []string{controller.PodKey(pod)})
	err = c.clientset.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, v1.DeleteOptions{})
	if err != nil {
		c.podExpectations.DeletionObserved(migrationKey, controller.PodKey(pod))
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedDeletePodReason, "Error deleted migration target pod: %v", err)
		return fmt.Errorf("failed to delete vmi migration target pod that reached pending pod timeout period.: %v", err)
	}
	log.Log.Object(vmi).Infof("Deleted pending migration target pod %s/%s with uuid %s for migration %s with uuid %s with reason [%s]", pod.Namespace, pod.Name, string(pod.UID), migration.Name, string(migration.UID), reason)
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, reason, pod.Name)
	return nil
}

func (c *MigrationController) getUnschedulablePendingTimeoutSeconds(migration *virtv1.VirtualMachineInstanceMigration) int64 {
	timeout := c.unschedulablePendingTimeoutSeconds
	customTimeoutStr, ok := migration.Annotations[virtv1.MigrationUnschedulablePodTimeoutSecondsAnnotation]
	if !ok {
		return timeout
	}

	newTimeout, err := strconv.Atoi(customTimeoutStr)
	if err != nil {
		log.Log.Object(migration).Reason(err).Errorf("Unable to parse unschedulable pending timeout value for migration")
		return timeout
	}

	return int64(newTimeout)
}

func (c *MigrationController) getCatchAllPendingTimeoutSeconds(migration *virtv1.VirtualMachineInstanceMigration) int64 {
	timeout := c.catchAllPendingTimeoutSeconds
	customTimeoutStr, ok := migration.Annotations[virtv1.MigrationPendingPodTimeoutSecondsAnnotation]
	if !ok {
		return timeout
	}

	newTimeout, err := strconv.Atoi(customTimeoutStr)
	if err != nil {
		log.Log.Object(migration).Reason(err).Errorf("Unable to parse catch all pending timeout value for migration")
		return timeout
	}

	return int64(newTimeout)
}

func (c *MigrationController) handlePendingPodTimeout(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {

	if pod.Status.Phase != k8sv1.PodPending || pod.DeletionTimestamp != nil || pod.CreationTimestamp.IsZero() {
		// only check if timeout has occurred if pod is pending and not already marked for deletion
		return nil
	}

	migrationKey, err := controller.KeyFunc(migration)
	if err != nil {
		return err
	}

	unschedulableTimeout := c.getUnschedulablePendingTimeoutSeconds(migration)
	catchAllTimeout := c.getCatchAllPendingTimeoutSeconds(migration)
	secondsSpentPending := timeSinceCreationSeconds(&pod.ObjectMeta)

	if isPodPendingUnschedulable(pod) {
		c.alertIfHostModelIsUnschedulable(vmi, pod)
		c.recorder.Eventf(
			migration,
			k8sv1.EventTypeWarning,
			MigrationTargetPodUnschedulable,
			"Migration target pod for VMI [%s/%s] is currently unschedulable.", vmi.Namespace, vmi.Name)
		log.Log.Object(migration).Warningf("Migration target pod for VMI [%s/%s] is currently unschedulable.", vmi.Namespace, vmi.Name)
		if secondsSpentPending >= unschedulableTimeout {
			return c.deleteTimedOutTargetPod(migration, vmi, pod, "unschedulable pod timeout period exceeded")
		} else {
			// Make sure we check this again after some time
			c.Queue.AddAfter(migrationKey, time.Second*time.Duration(unschedulableTimeout-secondsSpentPending))
		}
	}

	if secondsSpentPending >= catchAllTimeout {
		return c.deleteTimedOutTargetPod(migration, vmi, pod, "pending pod timeout period exceeded")
	} else {
		// Make sure we check this again after some time
		c.Queue.AddAfter(migrationKey, time.Second*time.Duration(catchAllTimeout-secondsSpentPending))
	}

	return nil
}

func (c *MigrationController) sync(key string, migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod) error {

	var pod *k8sv1.Pod = nil
	targetPodExists := len(pods) > 0
	if targetPodExists {
		pod = pods[0]
	}

	if vmiDeleted := vmi == nil || vmi.DeletionTimestamp != nil; vmiDeleted {
		return nil
	}

	if migrationFinalizedOnVMI := vmi.Status.MigrationState != nil && vmi.Status.MigrationState.MigrationUID == migration.UID &&
		vmi.Status.MigrationState.EndTimestamp != nil; migrationFinalizedOnVMI {
		return nil
	}

	canMigrate, err := c.canMigrateVMI(migration, vmi)
	if err != nil {
		return err
	}

	if !canMigrate {
		return fmt.Errorf("vmi is inelgible for migration because another migration job is running")
	}

	switch migration.Status.Phase {
	case virtv1.MigrationPending:
		if migration.DeletionTimestamp != nil {
			return c.handlePreHandoffMigrationCancel(migration, vmi, pod)
		}
		if err = c.handleMigrationBackoff(key, vmi, migration); errors.Is(err, migrationBackoffError) {
			warningMsg := fmt.Sprintf("backoff migrating vmi %s/%s", vmi.Namespace, vmi.Name)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, err.Error(), warningMsg)
			return nil
		}

		if !targetPodExists {
			sourcePod, err := controller.CurrentVMIPod(vmi, c.podInformer)
			if err != nil {
				log.Log.Reason(err).Error("Failed to fetch pods for namespace from cache.")
				return err
			}
			if !podExists(sourcePod) {
				// for instance sudden deletes can cause this. In this
				// case we don't have to do anything in the creation flow anymore.
				// Once the VMI is in a final state or deleted the migration
				// will be marked as failed too.
				return nil
			}

			var patches []string
			if !c.clusterConfig.RootEnabled() {
				// The cluster is configured for non-root VMs, ensure the VMI is non-root.
				// If the VMI is root, the migration will be a root -> non-root migration.
				if vmi.Status.RuntimeUser != util.NonRootUID {
					patches = append(patches, fmt.Sprintf(`{ "op": "replace", "path": "/status/runtimeUser", "value": %d }`, util.NonRootUID))
				}

				// This is required in order to be able to update from v0.43-v0.51 to v0.52+
				if vmi.Annotations == nil {
					patches = append(patches, fmt.Sprintf(`{ "op": "add", "path": "/metadata/annotations", "value":  { "%s": "true"} }`, virtv1.DeprecatedNonRootVMIAnnotation))
				} else if _, ok := vmi.Annotations[virtv1.DeprecatedNonRootVMIAnnotation]; !ok {
					patches = append(patches, fmt.Sprintf(`{ "op": "add", "path": "/metadata/annotations/%s", "value": "true"}`, patch.EscapeJSONPointer(virtv1.DeprecatedNonRootVMIAnnotation)))
				}
			} else {
				// The cluster is configured for root VMs, ensure the VMI is root.
				// If the VMI is non-root, the migration will be a non-root -> root migration.
				if vmi.Status.RuntimeUser != util.RootUser {
					patches = append(patches, fmt.Sprintf(`{ "op": "replace", "path": "/status/runtimeUser", "value": %d }`, util.RootUser))
				}

				if vmi.Annotations != nil {
					if _, ok := vmi.Annotations[virtv1.DeprecatedNonRootVMIAnnotation]; ok {
						patches = append(patches, fmt.Sprintf(`{ "op": "remove", "path": "/metadata/annotations/%s"}`, patch.EscapeJSONPointer(virtv1.DeprecatedNonRootVMIAnnotation)))
					}
				}
			}
			if len(patches) != 0 {
				vmi, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, controller.GeneratePatchBytes(patches), &v1.PatchOptions{})
				if err != nil {
					return fmt.Errorf("failed to set VMI RuntimeUser: %v", err)
				}
			}

			return c.handleTargetPodCreation(key, migration, vmi, sourcePod)
		} else if isPodReady(pod) {
			if controller.VMIHasHotplugVolumes(vmi) {
				attachmentPods, err := controller.AttachmentPods(pod, c.podInformer)
				if err != nil {
					return fmt.Errorf(failedGetAttractionPodsFmt, err)
				}
				if len(attachmentPods) == 0 {
					log.Log.Object(migration).Infof("Creating attachment pod for vmi %s/%s on node %s", vmi.Namespace, vmi.Name, pod.Spec.NodeName)
					return c.createAttachmentPod(migration, vmi, pod)
				}
			}
		} else {
			return c.handlePendingPodTimeout(migration, vmi, pod)
		}
	case virtv1.MigrationScheduling:
		if migration.DeletionTimestamp != nil {
			return c.handlePreHandoffMigrationCancel(migration, vmi, pod)
		}

		if targetPodExists {
			return c.handlePendingPodTimeout(migration, vmi, pod)
		}

	case virtv1.MigrationScheduled:
		if migration.DeletionTimestamp != nil && !c.isMigrationHandedOff(migration, vmi) {
			return c.handlePreHandoffMigrationCancel(migration, vmi, pod)
		}

		// once target pod is running, then alert the VMI of the migration by
		// setting the target and source nodes. This kicks off the preparation stage.
		if targetPodExists && isPodReady(pod) {
			return c.handleTargetPodHandoff(migration, vmi, pod)
		}
	case virtv1.MigrationPreparingTarget, virtv1.MigrationTargetReady, virtv1.MigrationFailed:
		if (!targetPodExists || podIsDown(pod)) &&
			vmi.Status.MigrationState != nil &&
			len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts) == 0 &&
			vmi.Status.MigrationState.StartTimestamp == nil &&
			!vmi.Status.MigrationState.Failed &&
			!vmi.Status.MigrationState.Completed {

			return c.handleMarkMigrationFailedOnVMI(migration, vmi)
		}
	case virtv1.MigrationRunning:
		if migration.DeletionTimestamp != nil && vmi.Status.MigrationState != nil {
			err = c.markMigrationAbortInVmiStatus(migration, vmi)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *MigrationController) listMatchingTargetPods(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) ([]*k8sv1.Pod, error) {

	selector, err := v1.LabelSelectorAsSelector(&v1.LabelSelector{
		MatchLabels: map[string]string{
			virtv1.CreatedByLabel:    string(vmi.UID),
			virtv1.AppLabel:          "virt-launcher",
			virtv1.MigrationJobLabel: string(migration.UID),
		},
	})
	if err != nil {
		return nil, err
	}

	objs, err := c.podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, migration.Namespace)
	if err != nil {
		return nil, err
	}

	var pods []*k8sv1.Pod
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		if selector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (c *MigrationController) addMigration(obj interface{}) {
	c.enqueueMigration(obj)
}

func (c *MigrationController) deleteMigration(obj interface{}) {
	c.enqueueMigration(obj)
}

func (c *MigrationController) updateMigration(_, curr interface{}) {
	c.enqueueMigration(curr)
}

func (c *MigrationController) enqueueMigration(obj interface{}) {
	logger := log.Log
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)
	key, err := controller.KeyFunc(migration)
	if err != nil {
		logger.Object(migration).Reason(err).Error("Failed to extract key from migration.")
		return
	}
	c.Queue.Add(key)
}

func (c *MigrationController) getControllerOf(pod *k8sv1.Pod) *v1.OwnerReference {
	t := true
	return &v1.OwnerReference{
		Kind:               virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
		Name:               pod.Annotations[virtv1.MigrationJobNameAnnotation],
		UID:                types.UID(pod.Labels[virtv1.MigrationJobLabel]),
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *MigrationController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineInstanceMigration {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind {
		return nil
	}
	migration, exists, err := c.migrationInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if migration.(*virtv1.VirtualMachineInstanceMigration).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return migration.(*virtv1.VirtualMachineInstanceMigration)
}

// When a pod is created, enqueue the migration that manages it and update its podExpectations.
func (c *MigrationController) addPod(obj interface{}) {
	pod := obj.(*k8sv1.Pod)

	if pod.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deletePod(pod)
		return
	}

	controllerRef := c.getControllerOf(pod)
	migration := c.resolveControllerRef(pod.Namespace, controllerRef)
	if migration == nil {
		return
	}
	migrationKey, err := controller.KeyFunc(migration)
	if err != nil {
		return
	}
	log.Log.V(4).Object(pod).Infof("Pod created")
	c.podExpectations.CreationObserved(migrationKey)
	c.enqueueMigration(migration)
}

// When a pod is updated, figure out what migration manages it and wake them
// up. If the labels of the pod have changed we need to awaken both the old
// and new migration. old and cur must be *v1.Pod types.
func (c *MigrationController) updatePod(old, cur interface{}) {
	curPod := cur.(*k8sv1.Pod)
	oldPod := old.(*k8sv1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	labelChanged := !equality.Semantic.DeepEqual(curPod.Labels, oldPod.Labels)
	if curPod.DeletionTimestamp != nil {
		// having a pod marked for deletion is enough to count as a deletion expectation
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
		}
		return
	}

	curControllerRef := c.getControllerOf(curPod)
	oldControllerRef := c.getControllerOf(oldPod)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if migration := c.resolveControllerRef(oldPod.Namespace, oldControllerRef); migration != nil {
			c.enqueueMigration(migration)
		}
	}

	migration := c.resolveControllerRef(curPod.Namespace, curControllerRef)
	if migration == nil {
		return
	}
	log.Log.V(4).Object(curPod).Infof("Pod updated")
	c.enqueueMigration(migration)
	return
}

// When a resourceQuota is updated, figure out if there are pending migration in the namespace
// if there are we should push them into the queue to accelerate the target creation process
func (c *MigrationController) updateResourceQuota(_, cur interface{}) {
	curResourceQuota := cur.(*k8sv1.ResourceQuota)
	log.Log.V(4).Object(curResourceQuota).Infof("ResourceQuota updated")
	objs, _ := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, curResourceQuota.Namespace)
	for _, obj := range objs {
		migration := obj.(*virtv1.VirtualMachineInstanceMigration)
		if migration.Status.Conditions == nil {
			continue
		}
		for _, cond := range migration.Status.Conditions {
			if cond.Type == virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota {
				c.enqueueMigration(migration)
			}
		}
	}
	return
}

// When a resourceQuota is deleted, figure out if there are pending migration in the namespace
// if there are we should push them into the queue to accelerate the target creation process
func (c *MigrationController) deleteResourceQuota(obj interface{}) {
	resourceQuota := obj.(*k8sv1.ResourceQuota)
	log.Log.V(4).Object(resourceQuota).Infof("ResourceQuota deleted")
	objs, _ := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, resourceQuota.Namespace)
	for _, obj := range objs {
		migration := obj.(*virtv1.VirtualMachineInstanceMigration)
		if migration.Status.Conditions == nil {
			continue
		}
		for _, cond := range migration.Status.Conditions {
			if cond.Type == virtv1.VirtualMachineInstanceMigrationRejectedByResourceQuota {
				c.enqueueMigration(migration)
			}
		}
	}
	return
}

// When a pod is deleted, enqueue the migration that manages the pod and update its podExpectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (c *MigrationController) deletePod(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new migration will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(failedToProcessDeleteNotificationErrMsg)
			return
		}
		pod, ok = tombstone.Obj.(*k8sv1.Pod)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pod %#v", obj)).Error(failedToProcessDeleteNotificationErrMsg)
			return
		}
	}

	controllerRef := c.getControllerOf(pod)
	migration := c.resolveControllerRef(pod.Namespace, controllerRef)
	if migration == nil {
		return
	}
	migrationKey, err := controller.KeyFunc(migration)
	if err != nil {
		return
	}
	c.podExpectations.DeletionObserved(migrationKey, controller.PodKey(pod))
	c.enqueueMigration(migration)
}

func (c *MigrationController) updatePDB(old, cur interface{}) {
	curPDB := cur.(*policyv1.PodDisruptionBudget)
	oldPDB := old.(*policyv1.PodDisruptionBudget)
	if curPDB.ResourceVersion == oldPDB.ResourceVersion {
		return
	}

	// Only process PDBs manipulated by this controller
	migrationName := curPDB.Labels[virtv1.MigrationNameLabel]
	if migrationName == "" {
		return
	}

	objs, err := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, curPDB.Namespace)
	if err != nil {
		return
	}

	for _, obj := range objs {
		vmim := obj.(*virtv1.VirtualMachineInstanceMigration)

		if vmim.Name == migrationName {
			log.Log.V(4).Object(curPDB).Infof("PDB updated")
			c.enqueueMigration(vmim)
		}
	}
}

type vmimCollection []*virtv1.VirtualMachineInstanceMigration

func (c vmimCollection) Len() int {
	return len(c)
}

func (c vmimCollection) Less(i, j int) bool {
	t1 := &c[i].CreationTimestamp
	t2 := &c[j].CreationTimestamp
	return t1.Before(t2)
}

func (c vmimCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *MigrationController) garbageCollectFinalizedMigrations(vmi *virtv1.VirtualMachineInstance) error {

	var finalizedMigrations []string

	migrations, err := c.listMigrationsMatchingVMI(vmi.Namespace, vmi.Name)
	if err != nil {
		return err
	}

	// Oldest first
	sort.Sort(vmimCollection(migrations))
	for _, migration := range migrations {
		if migration.IsFinal() && migration.DeletionTimestamp == nil {
			finalizedMigrations = append(finalizedMigrations, migration.Name)
		}
	}

	// only keep the oldest 5 finalized migration objects
	garbageCollectionCount := len(finalizedMigrations) - defaultFinalizedMigrationGarbageCollectionBuffer

	if garbageCollectionCount <= 0 {
		return nil
	}

	for i := 0; i < garbageCollectionCount; i++ {
		err = c.clientset.VirtualMachineInstanceMigration(vmi.Namespace).Delete(finalizedMigrations[i], &v1.DeleteOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			// This is safe to ignore. It's possible in some
			// scenarios that the migration we're trying to garbage
			// collect has already disappeared. Let's log it as debug
			// and suppress the error in this situation.
			log.Log.Reason(err).Infof("error encountered when garbage collecting migration object %s/%s", vmi.Namespace, finalizedMigrations[i])
		} else if err != nil {
			return err
		}
	}

	return nil
}

func (c *MigrationController) filterMigrations(namespace string, filter func(*virtv1.VirtualMachineInstanceMigration) bool) ([]*virtv1.VirtualMachineInstanceMigration, error) {
	objs, err := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	var migrations []*virtv1.VirtualMachineInstanceMigration
	for _, obj := range objs {
		migration := obj.(*virtv1.VirtualMachineInstanceMigration)

		if filter(migration) {
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
}

// takes a namespace and returns all migrations listening for this vmi
func (c *MigrationController) listMigrationsMatchingVMI(namespace, name string) ([]*virtv1.VirtualMachineInstanceMigration, error) {
	return c.filterMigrations(namespace, func(migration *virtv1.VirtualMachineInstanceMigration) bool {
		return migration.Spec.VMIName == name
	})
}

func (c *MigrationController) listEvacuationMigrations(namespace string, name string) ([]*virtv1.VirtualMachineInstanceMigration, error) {
	return c.filterMigrations(namespace, func(migration *virtv1.VirtualMachineInstanceMigration) bool {
		_, isEvacuation := migration.Annotations[virtv1.EvacuationMigrationAnnotation]
		return migration.Spec.VMIName == name && isEvacuation
	})
}

func (c *MigrationController) addVMI(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)
	if vmi.DeletionTimestamp != nil {
		c.deleteVMI(vmi)
		return
	}

	migrations, err := c.listMigrationsMatchingVMI(vmi.Namespace, vmi.Name)
	if err != nil {
		return
	}
	for _, migration := range migrations {
		c.enqueueMigration(migration)
	}
}

func (c *MigrationController) updateVMI(old, cur interface{}) {
	curVMI := cur.(*virtv1.VirtualMachineInstance)
	oldVMI := old.(*virtv1.VirtualMachineInstance)
	if curVMI.ResourceVersion == oldVMI.ResourceVersion {
		// Periodic resync will send update events for all known VMIs.
		// Two different versions of the same vmi will always
		// have different RVs.
		return
	}
	labelChanged := !equality.Semantic.DeepEqual(curVMI.Labels, oldVMI.Labels)
	if curVMI.DeletionTimestamp != nil {
		// having a DataVOlume marked for deletion is enough
		// to count as a deletion expectation
		c.deleteVMI(curVMI)
		if labelChanged {
			// we don't need to check the oldVMI.DeletionTimestamp
			// because DeletionTimestamp cannot be unset.
			c.deleteVMI(oldVMI)
		}
		return
	}

	migrations, err := c.listMigrationsMatchingVMI(curVMI.Namespace, curVMI.Name)
	if err != nil {
		log.Log.Object(curVMI).Errorf("Error encountered during datavolume update: %v", err)
		return
	}
	for _, migration := range migrations {
		log.Log.V(4).Object(curVMI).Infof("vmi updated for migration %s", migration.Name)
		c.enqueueMigration(migration)
	}
}
func (c *MigrationController) deleteVMI(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)
	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vmi
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(failedToProcessDeleteNotificationErrMsg)
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error(failedToProcessDeleteNotificationErrMsg)
			return
		}
	}
	migrations, err := c.listMigrationsMatchingVMI(vmi.Namespace, vmi.Name)
	if err != nil {
		return
	}
	for _, migration := range migrations {
		log.Log.V(4).Object(vmi).Infof("vmi deleted for migration %s", migration.Name)
		c.enqueueMigration(migration)
	}
}

func (c *MigrationController) outboundMigrationsOnNode(node string, runningMigrations []*virtv1.VirtualMachineInstanceMigration) (int, error) {
	sum := 0
	for _, migration := range runningMigrations {
		if vmi, exists, _ := c.vmiInformer.GetStore().GetByKey(migration.Namespace + "/" + migration.Spec.VMIName); exists {
			if vmi.(*virtv1.VirtualMachineInstance).Status.NodeName == node {
				sum = sum + 1
			}
		}
	}
	return sum, nil
}

// findRunningMigrations calcules how many migrations are running or in flight to be triggered to running
// Migrations which are in running phase are added alongside with migrations which are still pending but
// where we already see a target pod.
func (c *MigrationController) findRunningMigrations() ([]*virtv1.VirtualMachineInstanceMigration, error) {

	// Don't start new migrations if we wait for migration object updates because of new target pods
	notFinishedMigrations := migrations.ListUnfinishedMigrations(c.migrationInformer)
	var runningMigrations []*virtv1.VirtualMachineInstanceMigration
	for _, migration := range notFinishedMigrations {
		if migration.IsRunning() {
			runningMigrations = append(runningMigrations, migration)
			continue
		}
		vmi, exists, err := c.vmiInformer.GetStore().GetByKey(migration.Namespace + "/" + migration.Spec.VMIName)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		pods, err := c.listMatchingTargetPods(migration, vmi.(*virtv1.VirtualMachineInstance))
		if err != nil {
			return nil, err
		}
		if len(pods) > 0 {
			runningMigrations = append(runningMigrations, migration)
		}
	}
	return runningMigrations, nil
}

func (c *MigrationController) getNodeForVMI(vmi *virtv1.VirtualMachineInstance) (*k8sv1.Node, error) {
	obj, exists, err := c.nodeInformer.GetStore().GetByKey(vmi.Status.NodeName)

	if err != nil {
		return nil, fmt.Errorf("cannot get nodes to migrate VMI with host-model CPU. error: %v", err)
	} else if !exists {
		return nil, fmt.Errorf("node \"%s\" associated with vmi \"%s\" does not exist", vmi.Status.NodeName, vmi.Name)
	}

	node := obj.(*k8sv1.Node)
	return node, nil
}

func (c *MigrationController) alertIfHostModelIsUnschedulable(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	fittingNodeFound := false

	if cpu := vmi.Spec.Domain.CPU; cpu == nil || cpu.Model != virtv1.CPUModeHostModel {
		return
	}

	requiredNodeLabels := map[string]string{}
	for key, value := range targetPod.Spec.NodeSelector {
		if strings.HasPrefix(key, virtv1.SupportedHostModelMigrationCPU) || strings.HasPrefix(key, virtv1.CPUFeatureLabel) {
			requiredNodeLabels[key] = value
		}
	}

	nodes := c.nodeInformer.GetStore().List()
	for _, nodeInterface := range nodes {
		node := nodeInterface.(*k8sv1.Node)

		if node.Name == vmi.Status.NodeName {
			continue // avoid checking the VMI's source node
		}

		if isNodeSuitableForHostModelMigration(node, requiredNodeLabels) {
			log.Log.Object(vmi).Infof("Node %s is suitable to run vmi %s host model cpu mode (more nodes may fit as well)", node.Name, vmi.Name)
			fittingNodeFound = true
			break
		}
	}

	if !fittingNodeFound {
		warningMsg := fmt.Sprintf("Migration cannot proceed since no node is suitable to run the required CPU model / required features: %v", requiredNodeLabels)
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, NoSuitableNodesForHostModelMigration, warningMsg)
		log.Log.Object(vmi).Warning(warningMsg)
	}
}

func prepareNodeSelectorForHostCpuModel(node *k8sv1.Node, pod *k8sv1.Pod, sourcePod *k8sv1.Pod) error {
	var hostCpuModel, nodeSelectorKeyForHostModel, hostModelLabelValue string
	migratedAtLeastOnce := false

	// if the vmi already migrated before it should include node selector that consider CPUModelLabel
	for key, value := range sourcePod.Spec.NodeSelector {
		if strings.Contains(key, virtv1.CPUFeatureLabel) || strings.Contains(key, virtv1.SupportedHostModelMigrationCPU) {
			pod.Spec.NodeSelector[key] = value
			migratedAtLeastOnce = true
		}
	}

	if !migratedAtLeastOnce {
		for key, value := range node.Labels {
			if strings.HasPrefix(key, virtv1.HostModelCPULabel) {
				hostCpuModel = strings.TrimPrefix(key, virtv1.HostModelCPULabel)
				hostModelLabelValue = value
			}

			if strings.HasPrefix(key, virtv1.HostModelRequiredFeaturesLabel) {
				requiredFeature := strings.TrimPrefix(key, virtv1.HostModelRequiredFeaturesLabel)
				pod.Spec.NodeSelector[virtv1.CPUFeatureLabel+requiredFeature] = value
			}
		}

		if hostCpuModel == "" {
			return fmt.Errorf("node does not contain labal \"%s\" with information about host cpu model", virtv1.HostModelCPULabel)
		}

		nodeSelectorKeyForHostModel = virtv1.SupportedHostModelMigrationCPU + hostCpuModel
		pod.Spec.NodeSelector[nodeSelectorKeyForHostModel] = hostModelLabelValue

		log.Log.Object(pod).Infof("cpu model label selector (\"%s\") defined for migration target pod", nodeSelectorKeyForHostModel)
	}

	return nil
}

func isNodeSuitableForHostModelMigration(node *k8sv1.Node, requiredNodeLabels map[string]string) bool {
	for key, value := range requiredNodeLabels {
		nodeValue, ok := node.Labels[key]

		if !ok || nodeValue != value {
			return false
		}
	}

	return true
}

func (c *MigrationController) matchMigrationPolicy(vmi *virtv1.VirtualMachineInstance, clusterMigrationConfiguration *virtv1.MigrationConfiguration) error {
	vmiNamespace, err := c.clientset.CoreV1().Namespaces().Get(context.Background(), vmi.Namespace, v1.GetOptions{})
	if err != nil {
		return err
	}

	// Fetch cluster policies
	var policies []v1alpha1.MigrationPolicy
	migrationInterfaceList := c.migrationPolicyInformer.GetStore().List()
	for _, obj := range migrationInterfaceList {
		policy := obj.(*v1alpha1.MigrationPolicy)
		policies = append(policies, *policy)
	}
	policiesListObj := v1alpha1.MigrationPolicyList{Items: policies}

	// Override cluster-wide migration configuration if migration policy is matched
	matchedPolicy := MatchPolicy(&policiesListObj, vmi, vmiNamespace)

	if matchedPolicy == nil {
		log.Log.Object(vmi).Reason(err).Infof("no migration policy matched for VMI %s", vmi.Name)
		return nil
	}

	isUpdated, err := matchedPolicy.GetMigrationConfByPolicy(clusterMigrationConfiguration)
	if err != nil {
		return err
	}

	if isUpdated {
		vmi.Status.MigrationState.MigrationPolicyName = &matchedPolicy.Name
		vmi.Status.MigrationState.MigrationConfiguration = clusterMigrationConfiguration
		log.Log.Object(vmi).Infof("migration is updated by migration policy named %s.", matchedPolicy.Name)
	}

	return nil
}

func (c *MigrationController) isMigrationPolicyMatched(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	migrationPolicyName := vmi.Status.MigrationState.MigrationPolicyName
	return migrationPolicyName != nil && *migrationPolicyName != ""
}

func (c *MigrationController) isMigrationHandedOff(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.MigrationUID == migration.UID {
		return true
	}

	migrationKey := controller.MigrationKey(migration)

	c.handOffLock.Lock()
	defer c.handOffLock.Unlock()

	_, isHandedOff := c.handOffMap[migrationKey]
	return isHandedOff
}

func (c *MigrationController) addHandOffKey(migrationKey string) {
	c.handOffLock.Lock()
	defer c.handOffLock.Unlock()

	c.handOffMap[migrationKey] = struct{}{}
}

func (c *MigrationController) removeHandOffKey(migrationKey string) {
	c.handOffLock.Lock()
	defer c.handOffLock.Unlock()

	delete(c.handOffMap, migrationKey)
}

func getComputeContainer(pod *k8sv1.Pod) *k8sv1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == "compute" {
			return &container
		}
	}
	return nil
}

func getTargetPodLimitsCount(pod *k8sv1.Pod) (int64, error) {
	cc := getComputeContainer(pod)
	if cc == nil {
		return 0, fmt.Errorf("Could not find VMI compute container")
	}

	cpuLimit, ok := cc.Resources.Limits[k8sv1.ResourceCPU]
	if !ok {
		return 0, fmt.Errorf("Could not find dedicated CPU limit in VMI compute container")
	}
	return cpuLimit.Value(), nil
}

func getTargetPodMemoryRequests(pod *k8sv1.Pod) (string, error) {
	cc := getComputeContainer(pod)
	if cc == nil {
		return "", fmt.Errorf("Could not find VMI compute container")
	}

	memReq, ok := cc.Resources.Requests[k8sv1.ResourceMemory]
	if !ok {
		return "", fmt.Errorf("Could not find memory request in VMI compute container")
	}
	return memReq.String(), nil
}
