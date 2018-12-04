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
	"fmt"
	"reflect"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type MigrationController struct {
	templateService   services.TemplateService
	clientset         kubecli.KubevirtClient
	Queue             workqueue.RateLimitingInterface
	vmiInformer       cache.SharedIndexInformer
	podInformer       cache.SharedIndexInformer
	migrationInformer cache.SharedIndexInformer
	recorder          record.EventRecorder
	podExpectations   *controller.UIDTrackingControllerExpectations
}

func NewMigrationController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *MigrationController {

	c := &MigrationController{
		templateService:   templateService,
		Queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:       vmiInformer,
		podInformer:       podInformer,
		migrationInformer: migrationInformer,
		recorder:          recorder,
		clientset:         clientset,
		podExpectations:   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMI,
		DeleteFunc: c.deleteVMI,
		UpdateFunc: c.updateVMI,
	})

	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.deletePod,
		UpdateFunc: c.updatePod,
	})

	c.migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addMigration,
		DeleteFunc: c.deleteMigration,
		UpdateFunc: c.updateMigration,
	})

	return c
}

func (c *MigrationController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting migration controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.migrationInformer.HasSynced)

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
		return nil
	}
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)
	logger := log.Log.Object(migration)

	vmiObj, vmiExists, err := c.vmiInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", migration.Namespace, migration.Spec.VMIName))
	if err != nil {
		return err
	}

	if !vmiExists {
		if migration.DeletionTimestamp == nil {
			logger.V(3).Infof("Deleting migration for deleted vmi %s/%s", migration.Namespace, migration.Spec.VMIName)
			err := c.clientset.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &v1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
		// nothing to process for a migration that's being deleted
		return nil
	}

	vmi = vmiObj.(*virtv1.VirtualMachineInstance)
	targetPods, err = c.listMatchingTargetPods(migration, vmi)
	if err != nil {
		return err
	}

	needsSync := c.podExpectations.SatisfiedExpectations(key) && vmiExists

	logger.V(4).Infof("processing migration: needsSync %t, hasVMI %t, targetPod len %d", needsSync, vmiExists, len(targetPods))

	var syncErr error

	if needsSync && !migration.IsFinal() {
		syncErr = c.sync(migration, vmi, targetPods)
	}

	err = c.updateStatus(migration, vmi, targetPods, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil {
		return syncErr
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

func (c *MigrationController) updateStatus(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod, syncErr error) error {

	var pod *k8sv1.Pod = nil

	// Nothing to do here if the migration has already completed
	if migration.IsFinal() {
		return nil
	}

	podExists := len(pods) > 0
	if podExists {
		pod = pods[0]
	}

	migrationCopy := migration.DeepCopy()

	// Status checking of active Migration job.
	//
	// 1. Fail if VMI isn't in running state.
	// 2. Fail if target pod exists and has gone down for any reason.
	// 3. Begin progressing migration state based on VMI's MigrationState status.
	if vmi == nil {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed because vmi does not exist.")
		log.Log.Object(migration).Error("vmi does not exist")
	} else if vmi.IsFinal() {
		migrationCopy.Status.Phase = virtv1.MigrationFailed
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Migration failed vmi shutdown during migration.")
		log.Log.Object(migration).Error("Unable to migrate vmi because vmi is shutdown.")
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
		c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedMigrationReason, "Source node reported migration failed")
		log.Log.Object(migration).Error("VMI reported migration failed.")
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
				migrationCopy.Status.Phase = virtv1.MigrationScheduling
			}
		case virtv1.MigrationScheduling:
			if isPodReady(pod) {
				migrationCopy.Status.Phase = virtv1.MigrationScheduled
			}
		case virtv1.MigrationScheduled:
			if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNode != "" {
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
			if vmi.Status.MigrationState.Completed {
				migrationCopy.Status.Phase = virtv1.MigrationSucceeded
				c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulMigrationReason, "Source node reported migration succeeded")
				log.Log.Object(migration).Infof("VMI reported migration succeeded.")
			}
		}
	}

	if !reflect.DeepEqual(migration.Status, migrationCopy.Status) {
		_, err := c.clientset.VirtualMachineInstanceMigration(migration.Namespace).Update(migrationCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *MigrationController) createTargetPod(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) error {

	templatePod, err := c.templateService.RenderLaunchManifest(vmi)
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
	templatePod.ObjectMeta.Annotations[virtv1.MigrationJobNameAnnotation] = string(migration.Name)

	// TODO libvirt requires unique host names for each target and source
	templatePod.Spec.Hostname = ""

	key := controller.MigrationKey(migration)
	c.podExpectations.ExpectCreations(key, 1)
	pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(templatePod)
	if err != nil {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
		c.podExpectations.CreationObserved(key)
		return fmt.Errorf("failed to create vmi migration target pod: %v", err)
	}
	c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created migration target pod %s", pod.Name)
	return nil
}

func (c *MigrationController) sync(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod) error {

	var pod *k8sv1.Pod = nil
	podExists := len(pods) > 0
	if podExists {
		pod = pods[0]
	}

	if vmi == nil || vmi.DeletionTimestamp != nil {
		// nothing to do with a deleted vmi
		return nil
	} else if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.MigrationUID == migration.UID &&
		vmi.Status.MigrationState.EndTimestamp != nil {

		// nothing to do here, the migration is done
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
		// migration was accepted into the system, now see if we
		// should create the target pod
		if !podExists && vmi.IsRunning() {
			return c.createTargetPod(migration, vmi)
		}
		return nil
	case virtv1.MigrationScheduled:
		// once target pod is scheduled, alert the VMI of the migration by
		// setting the target and source nodes. This kicks off the preparation stage.
		if podExists && !podIsDown(pod) {
			vmiCopy := vmi.DeepCopy()
			vmiCopy.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
				MigrationUID: migration.UID,
				TargetNode:   pod.Spec.NodeName,
				SourceNode:   vmi.Status.NodeName,
			}

			// By setting this label, virt-handler on the target node will receive
			// the vmi and prepare the local environment for the migration
			vmiCopy.ObjectMeta.Labels[virtv1.MigrationTargetNodeNameLabel] = pod.Spec.NodeName

			if !reflect.DeepEqual(vmi.Status, vmiCopy.Status) ||
				!reflect.DeepEqual(vmi.Labels, vmiCopy.Labels) {
				_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Update(vmiCopy)
				if err != nil {
					c.recorder.Eventf(migration, k8sv1.EventTypeWarning, FailedHandOverPodReason, fmt.Sprintf("Failed to set MigrationStat in VMI status. :%v", err))
					return err
				}
				c.recorder.Eventf(migration, k8sv1.EventTypeNormal, SuccessfulHandOverPodReason, "Migration target pod is ready for preparation by virt-handler.")
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

	objs, err := c.podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, migration.Namespace)
	if err != nil {
		return nil, err
	}

	pods := []*k8sv1.Pod{}
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

func (c *MigrationController) updateMigration(old, curr interface{}) {
	c.enqueueMigration(curr)
}

func (c *MigrationController) enqueueMigration(obj interface{}) {
	logger := log.Log
	migration := obj.(*virtv1.VirtualMachineInstanceMigration)
	key, err := controller.KeyFunc(migration)
	if err != nil {
		logger.Object(migration).Reason(err).Error("Failed to extract key from migration.")
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

	labelChanged := !reflect.DeepEqual(curPod.Labels, oldPod.Labels)
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
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
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
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		pod, ok = tombstone.Obj.(*k8sv1.Pod)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pod %#v", obj)).Error("Failed to process delete notification")
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

// takes a namespace and returns all migrations listening for this vmi
func (c *MigrationController) listMigrationsMatchingVMI(namespace string, name string) ([]*virtv1.VirtualMachineInstanceMigration, error) {
	objs, err := c.migrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	migrations := []*virtv1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*virtv1.VirtualMachineInstanceMigration)

		if migration.Spec.VMIName == name {
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
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
	labelChanged := !reflect.DeepEqual(curVMI.Labels, oldVMI.Labels)
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
		log.Log.V(4).Object(curVMI).Errorf("Error encountered during datavolume update: %v", err)
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
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error("Failed to process delete notification")
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
