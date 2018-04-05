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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watch

import (
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"fmt"
	"reflect"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

// Reasons for vm events
const (
	// FailedCreatePodReason is added in an event and in a vm controller condition
	// when a pod for a vm controller is failed to be created.
	FailedCreatePodReason = "FailedCreate"
	// SuccessfulCreatePodReason is added in an event when a pod for a vm controller
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreate"
	// FailedDeletePodReason is added in an event and in a vm controller condition
	// when a pod for a vm controller is failed to be deleted.
	FailedDeletePodReason = "FailedDelete"
	// SuccessfulDeletePodReason is added in an event when a pod for a vm controller
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDelete"
)

func NewVMController(templateService services.TemplateService, vmCache cache.Store, vmInformer cache.SharedIndexInformer, podInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient) *VMController {
	c := &VMController{
		vmService:    services.NewVMService(clientset, templateService),
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		store:        vmCache,
		vmInformer:   vmInformer,
		podInformer:  podInformer,
		recorder:     recorder,
		clientset:    clientset,
		expectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.deletePod,
		UpdateFunc: c.updatePod,
	})

	return c
}

type VMController struct {
	vmService    services.VMService
	clientset    kubecli.KubevirtClient
	queue        workqueue.RateLimitingInterface
	store        cache.Store
	vmInformer   cache.SharedIndexInformer
	podInformer  cache.SharedIndexInformer
	recorder     record.EventRecorder
	expectations *controller.UIDTrackingControllerExpectations
}

func (c *VMController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.podInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping controller.")
}

func (c *VMController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VM %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VM %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VMController) execute(key string) error {

	// Fetch the latest Vm state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return err
	}

	// If the VM gets deleted we let the garbage collector and virt-handler take care of the Pod deletion.
	// TODO as soon as TPRs support graceful deletes this assumption is not true anymore
	if !exists {
		c.expectations.DeleteExpectations(key)
		return nil
	}
	vm := obj.(*virtv1.VirtualMachine)

	// If the VM is exists still, don't process the VM until it is fully initialized.
	// Initialization is handled by the initialization controller and must take place
	// before the VM is acted upon.
	if exists && !isVirtualMachineInitialized(vm) {
		return nil
	}

	selector, err := v1.LabelSelectorAsSelector(&v1.LabelSelector{MatchLabels: map[string]string{virtv1.DomainLabel: vm.Name, virtv1.AppLabel: "virt-launcher"}})

	logger := log.Log.Object(vm)

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.VM(vm.ObjectMeta.Namespace).Get(vm.ObjectMeta.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != vm.ObjectMeta.UID {
			return nil, fmt.Errorf("original VirtualMachine %v/%v is gone: got uid %v, wanted %v", vm.Namespace, vm.Name, fresh.UID, vm.UID)
		}
		return fresh, nil
	})

	// Get all non-terminated pods from the namespace
	pods, err := c.listPodsFromNamespace(vm.Namespace)

	if err != nil {
		logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}

	cm := controller.NewPodControllerRefManager(controller.RealPodControl{Clientset: c.clientset}, vm, selector, virtv1.VMReplicaSetGroupVersionKind, canAdoptFunc)
	pods, err = cm.ClaimPods(pods)
	if err != nil {
		return err
	}

	needsSync := c.expectations.SatisfiedExpectations(key)

	// Copy to allow manipulating the vm
	vm = vm.DeepCopy()

	var syncErr error
	if needsSync && vm.ObjectMeta.DeletionTimestamp != nil {
		syncErr = c.sync(vm, pods)
	}
	return c.updateStatus(vm, pods, syncErr)
}

func (c *VMController) updateStatus(vm *virtv1.VirtualMachine, pods []*k8sv1.Pod, syncErr error) error {

	vmCopy := vm.DeepCopy()

	errorsMatch := (syncErr != nil) == controller.NewVirtualMachineConditionManager().HasCondition(vmCopy, virtv1.VirtualMachineSynchronized)

	ownedByHandler := !vmCopy.IsUnprocessed() && !vmCopy.IsScheduling() && !vmCopy.IsFinal()

	if ownedByHandler {
		// TODO introduce unknown state if the node is down
		return nil
	}

	if len(pods) > 0 {
		podIsReady := podIsReady(pods[0])

		podIsTerminating := podIsDownOrGoingDown(pods[0])

		podIsDown := podIsDown(pods[0])

		if !podIsTerminating && !podIsReady {
			vmCopy.Status.Phase = virtv1.Scheduling
		} else if !podIsTerminating && podIsReady {
			vmCopy.Status.Interfaces = []virtv1.VirtualMachineNetworkInterface{
				{
					IP: pods[0].Status.PodIP,
				},
			}
			vmCopy.Status.Phase = virtv1.Scheduled
			vmCopy.ObjectMeta.Labels[virtv1.NodeNameLabel] = pods[0].Spec.NodeName
		} else if podIsDown {
			vmCopy.Status.Phase = virtv1.Failed
		}
	} else {
		// nothing to do we don't own the VM anymore
	}

	if !errorsMatch {
		reason := "FailedCreate"
		if len(pods) > 0 {
			reason = "FailedDelete"
		}
		if syncErr != nil {
			vmCopy.Status.Conditions = append(vmCopy.Status.Conditions, virtv1.VirtualMachineCondition{
				Type:               virtv1.VirtualMachineSynchronized,
				Reason:             reason,
				Message:            syncErr.Error(),
				LastTransitionTime: v1.Now(),
				Status:             k8sv1.ConditionTrue,
			})

		} else {
			c.removeCondition(vmCopy, virtv1.VirtualMachineSynchronized)
		}
	}

	// If we detect a change on the vm status, we update the vm
	if !reflect.DeepEqual(vm.Status, vmCopy.Status) {
		_, err := c.clientset.VM(vm.Namespace).Update(vmCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func podIsReady(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Ready == false {
			return false
		}
	}

	return true
}

func podIsDownOrGoingDown(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed || pod.DeletionTimestamp != nil
}

func podIsDown(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

func (c *VMController) sync(vm *virtv1.VirtualMachine, pods []*k8sv1.Pod) (err error) {

	vmKey := controller.VirtualMachineKey(vm)

	if vm.IsUnprocessed() && len(pods) == 0 {
		c.expectations.ExpectCreations(vmKey, 1)
		pod, err := c.vmService.StartVMPod(vm)
		if err != nil {
			c.recorder.Eventf(vm, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
			c.expectations.CreationObserved(vmKey)
			return err
		}
		c.recorder.Eventf(vm, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil
	} else if vm.IsFinal() && len(pods) > 0 {

		var wg sync.WaitGroup
		wg.Add(len(pods))
		errChan := make(chan error, len(pods))
		c.expectations.ExpectDeletions(vmKey, controller.PodKeys(pods))
		for i := 0; i < len(pods); i++ {
			go func(pod *k8sv1.Pod) {
				defer wg.Done()
				err := c.clientset.CoreV1().Pods(vm.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
				if err != nil {
					c.expectations.DeleteExpectations(vmKey)
					c.recorder.Eventf(vm, k8sv1.EventTypeWarning, FailedDeletePodReason, "Error deleting virtual machine pod %s: %v", pod.Name, err)
					errChan <- err
					return
				}
				c.recorder.Eventf(vm, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted virtual machine pod: %v", pod.Name)
			}(pods[i])
		}
		wg.Wait()

		select {
		case err := <-errChan:
			// Only return the first error which occurred, the others will most likely be equal errors
			return err
		default:
		}
	} else if len(pods) > 1 {
		return fmt.Errorf("Found %d matching pods where only one should exist", len(pods))
	}
	return nil
}

func extractKeyFromPod(pod *k8sv1.Pod) string {
	namespace := pod.ObjectMeta.Namespace
	appLabel, hasAppLabel := pod.ObjectMeta.Labels[virtv1.AppLabel]
	domainLabel, hasDomainLabel := pod.ObjectMeta.Labels[virtv1.DomainLabel]

	if hasDomainLabel == false || hasAppLabel == false || appLabel != "virt-launcher" {
		// missing required labels
		return ""
	}
	return namespace + "/" + domainLabel
}

// When a pod is created, enqueue the replica set that manages it and update its expectations.
func (c *VMController) addPod(obj interface{}) {
	pod := obj.(*k8sv1.Pod)

	if pod.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deletePod(pod)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := controller.GetControllerOf(pod); controllerRef != nil {
		vm := c.resolveControllerRef(pod.Namespace, controllerRef)
		if vm == nil {
			return
		}
		vmKey, err := controller.KeyFunc(vm)
		if err != nil {
			return
		}
		log.Log.V(4).Object(pod).Infof("Pod created")
		c.expectations.CreationObserved(vmKey)
		c.enqueueVirtualMachine(vm)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching vms and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	vms := c.getMatchingControllers(pod)
	if len(vms) == 0 {
		return
	}
	log.Log.V(4).Object(pod).Infof("Orphan Pod created")
	for _, vm := range vms {
		c.enqueueVirtualMachine(vm)
	}
}

// When a pod is updated, figure out what replica set/s manage it and wake them
// up. If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (c *VMController) updatePod(old, cur interface{}) {
	curPod := cur.(*k8sv1.Pod)
	oldPod := old.(*k8sv1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	labelChanged := !reflect.DeepEqual(curPod.Labels, oldPod.Labels)
	if curPod.DeletionTimestamp != nil {
		// when a pod is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the virt-handler actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect an rs to create more replicas asap, not wait
		// until the virt-handler actually deletes the pod. This is different from the Phase of a pod changing, because
		// an rs never initiates a phase change, and so is never asleep waiting for the same.
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
		}
		return
	}

	curControllerRef := controller.GetControllerOf(curPod)
	oldControllerRef := controller.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vm := c.resolveControllerRef(oldPod.Namespace, oldControllerRef); vm != nil {
			c.enqueueVirtualMachine(vm)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		vm := c.resolveControllerRef(curPod.Namespace, curControllerRef)
		if vm == nil {
			return
		}
		log.Log.V(4).Object(curPod).Infof("Pod updated")
		c.enqueueVirtualMachine(vm)
		// TODO: MinReadySeconds in the Pod will generate an Available condition to be added in
		// Update once we support the available conect on the vm
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	if labelChanged || controllerRefChanged {
		vms := c.getMatchingControllers(curPod)
		if len(vms) == 0 {
			return
		}
		log.Log.V(4).Object(curPod).Infof("Orphan Pod updated")
		for _, vm := range vms {
			c.enqueueVirtualMachine(vm)
		}
	}
}

// When a pod is deleted, enqueue the replica set that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (c *VMController) deletePod(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new vm will not be woken up till the periodic resync.
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

	controllerRef := controller.GetControllerOf(pod)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	vm := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vm == nil {
		return
	}
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(vmKey, controller.PodKey(pod))
	c.enqueueVirtualMachine(vm)
}

func (c *VMController) addVirtualMachine(obj interface{}) {
	c.enqueueVirtualMachine(obj)
}

func (c *VMController) deleteVirtualMachine(obj interface{}) {
	c.enqueueVirtualMachine(obj)
}

func (c *VMController) updateVirtualMachine(old, curr interface{}) {
	c.enqueueVirtualMachine(curr)
}

func (c *VMController) enqueueVirtualMachine(obj interface{}) {
	logger := log.Log
	vm := obj.(*virtv1.VirtualMachine)
	key, err := controller.KeyFunc(vm)
	if err != nil {
		logger.Object(vm).Reason(err).Error("Failed to extract key from virtualmachine.")
	}
	c.queue.Add(key)
}

// getMatchingController returns the first vm which matches the labels of the VM from the listener cache.
// If there are no matching controllers, a NotFound error is returned.
func (c *VMController) getMatchingControllers(pod *k8sv1.Pod) (vms []*virtv1.VirtualMachine) {
	var key string
	if key = extractKeyFromPod(pod); key == "" {
		return nil
	}
	obj, exists, err := c.store.GetByKey(key)
	if err != nil || !exists {
		return nil
	}
	vm := obj.(*virtv1.VirtualMachine)
	return []*virtv1.VirtualMachine{vm}
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *VMController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return nil
	}
	vm, exists, err := c.store.GetByKey(namespace + "/" + controllerRef.Name)
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

// listPodsFromNamespace takes a namespace and returns all Pods from the pod cache which run in this namespace
func (c *VMController) listPodsFromNamespace(namespace string) ([]*k8sv1.Pod, error) {
	objs, err := c.podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	pods := []*k8sv1.Pod{}
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		pods = append(pods, pod)
	}
	return pods, nil
}

func (c *VMController) hasCondition(vm *virtv1.VirtualMachine, cond virtv1.VirtualMachineConditionType) bool {
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *VMController) removeCondition(vm *virtv1.VirtualMachine, cond virtv1.VirtualMachineConditionType) {
	var conds []virtv1.VirtualMachineCondition
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vm.Status.Conditions = conds
}
