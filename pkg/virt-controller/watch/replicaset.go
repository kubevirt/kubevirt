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

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sync"

	k8score "k8s.io/api/core/v1"

	"github.com/jeevatkm/go-model"

	"k8s.io/apimachinery/pkg/api/errors"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
)

// Reasons for virtual machine events
const (
	// FailedCreateVirtualMachineReason is added in an event and in a replica set condition
	// when a virtual machine for a replica set is failed to be created.
	FailedCreateVirtualMachineReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineReason is added in an event when a virtual machine for a replica set
	// is successfully created.
	SuccessfulCreateVirtualMachineReason = "SuccessfulCreate"
	// FailedDeleteVirtualMachineReason is added in an event and in a replica set condition
	// when a virtual machine for a replica set is failed to be deleted.
	FailedDeleteVirtualMachineReason = "FailedDelete"
	// SuccessfulDeleteVirtualMachineReason is added in an event when a virtual machine for a replica set
	// is successfully deleted.
	SuccessfulDeleteVirtualMachineReason = "SuccessfulDelete"
)

func NewVMReplicaSet(vmInformer cache.SharedIndexInformer, vmRSInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient) *VMReplicaSet {

	c := &VMReplicaSet{
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:   vmInformer,
		vmRSInformer: vmRSInformer,
		recorder:     recorder,
		clientset:    clientset,
	}

	vmRSInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForWorkqueue(c.queue))

	c.vmInformer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForFunc(c.vmChangeFunc))

	return c
}

type VMReplicaSet struct {
	clientset    kubecli.KubevirtClient
	queue        workqueue.RateLimitingInterface
	vmInformer   cache.SharedIndexInformer
	vmRSInformer cache.SharedIndexInformer
	recorder     record.EventRecorder
}

func (c *VMReplicaSet) Run(threadiness int, stopCh chan struct{}) {
	defer kubecli.HandlePanic()
	defer c.queue.ShutDown()
	logging.DefaultLogger().Info().Msg("Starting VirtualMachineReplicaSet controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmRSInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping VirtualMachineReplicaSet controller.")
}

func (c *VMReplicaSet) runWorker() {
	for c.Execute() {
	}
}

func (c *VMReplicaSet) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		logging.DefaultLogger().Info().Reason(err).Msgf("re-enqueuing VirtualMachineReplicaSet %v", key)
		c.queue.AddRateLimited(key)
	} else {
		logging.DefaultLogger().Info().V(4).Msgf("processed VirtualMachineReplicaSet %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VMReplicaSet) execute(key string) error {

	obj, exists, err := c.vmRSInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		return nil
	}
	rs := obj.(*virtv1.VirtualMachineReplicaSet)

	log := logging.DefaultLogger().Object(rs)

	//TODO default rs if necessary, the aggregated apiserver will do that in the future
	if rs.Spec.Template == nil || rs.Spec.Selector == nil || len(rs.Spec.Template.ObjectMeta.Labels) == 0 {
		log.Error().Msg("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		log.Error().Reason(err).Msg("Invalid selector on replicaset, will not re-enqueue.")
		return nil
	}

	// get all potentially interesting VMs from the cache
	vms, err := c.listVMsFromNamespace(rs.ObjectMeta.Namespace)

	if err != nil {
		log.Error().Reason(err).Msg("Failed to fetch vms for namespace from cache.")
		return err
	}

	// make sure we only consider active VMs
	vms = c.filterActiveVMs(vms)

	// make sure the selector of the controller matches and the VMs match
	vms = c.filterMatchingVMs(selector, vms)

	// Scale up or down
	scaleErr := c.scale(rs, vms)

	if scaleErr != nil {
		log.Error().Reason(err).Msg("Scaling the replicaset failed.")
	}

	clone, err := model.Clone(rs)

	if err != nil {
		log.Error().Reason(err).Msg("Cloning the replicaset failed.")
		return nil
	}
	rsCopy := clone.(*virtv1.VirtualMachineReplicaSet)

	err = c.updateStatus(rsCopy, vms, scaleErr)
	if err != nil {
		log.Error().Reason(err).Msg("Updating the replicaset status failed.")
	}

	return err
}

func (c *VMReplicaSet) scale(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine) error {

	diff := c.calcDiff(rs, vms)

	if diff == 0 {
		return nil
	}

	// Every delete request can fail, give the channel enough room, to not block the go routines
	errChan := make(chan error, abs(diff))

	var wg sync.WaitGroup
	wg.Add(abs(diff))

	if diff > 0 {
		// We have to delete VMs
		for i := 0; i < diff; i++ {
			go func(idx int) {
				defer wg.Done()
				deleteCandidate := &vms[idx]
				// TODO graceful delete
				err := c.clientset.VM(rs.ObjectMeta.Namespace).Delete(deleteCandidate.ObjectMeta.Name, &v1.DeleteOptions{})
				// Don't log an error if it is already deleted
				if err != nil {
					c.recorder.Eventf(deleteCandidate, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting: %v", err)
					errChan <- err
					return
				}
				// If already deleted, don't log an event
				if !errors.IsNotFound(err) {
					c.recorder.Eventf(deleteCandidate, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Deleted virtual machine: %v", deleteCandidate.ObjectMeta.UID)
				}
			}(i)
		}

	} else if diff < 0 {
		// We have to create VMs
		basename := c.getVirtualMachineBaseName(rs)
		for i := diff; i < 0; i++ {
			go func() {
				defer wg.Done()
				vm := virtv1.NewVMReferenceFromNameWithNS(rs.ObjectMeta.Namespace, "")
				vm.ObjectMeta = rs.Spec.Template.ObjectMeta
				vm.ObjectMeta.Name = ""
				vm.ObjectMeta.GenerateName = basename
				vm.Spec = rs.Spec.Template.Spec
				// TODO check if vm labels exist, and when make sure that they match. For now just override them
				vm.ObjectMeta.Labels = rs.Spec.Template.ObjectMeta.Labels
				vm, err := c.clientset.VM(rs.ObjectMeta.Namespace).Create(vm)
				if err != nil {
					c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error deleting: %v", err)
					errChan <- err
					return
				}
				c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Created virtual machine: %v", vm.ObjectMeta.Name)
			}()
		}
	}
	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred, the others will most likely be equal errors
		return err
	default:
	}
	return nil
}

// filterActiveVMs takes a list of VMs and returns all VMs which are not in a final state
// Note that vms which have a deletion timestamp set, are still treated as active.
// This is a difference to Pod ReplicaSets
func (c *VMReplicaSet) filterActiveVMs(vms []virtv1.VirtualMachine) []virtv1.VirtualMachine {
	filtered := []virtv1.VirtualMachine{}
	for _, vm := range vms {
		if !vm.IsFinal() {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// filterMatchingVMs takes a selector and a list of VMs. If the VM labels match the selector it is added to the filtered collection.
// Returns the list of all VMs which match the selector
func (c *VMReplicaSet) filterMatchingVMs(selector labels.Selector, vms []virtv1.VirtualMachine) []virtv1.VirtualMachine {
	//TODO take owner reference into account
	filtered := []virtv1.VirtualMachine{}
	for _, vm := range vms {
		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
func (c *VMReplicaSet) listVMsFromNamespace(namespace string) ([]virtv1.VirtualMachine, error) {
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []virtv1.VirtualMachine{}
	for _, obj := range objs {
		vms = append(vms, *obj.(*virtv1.VirtualMachine))
	}
	return vms, nil
}

// listControllerFromNamespace takes a namespace and returns all VMReplicaSets from the ReplicaSet cache which run in this namespace
func (c *VMReplicaSet) listControllerFromNamespace(namespace string) ([]virtv1.VirtualMachineReplicaSet, error) {
	objs, err := c.vmRSInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	replicaSets := []virtv1.VirtualMachineReplicaSet{}
	for _, obj := range objs {
		rs := obj.(*virtv1.VirtualMachineReplicaSet)
		replicaSets = append(replicaSets, *rs)
	}
	return replicaSets, nil
}

// vmChangeFunc checks if the supplied VM matches a replica set controller in it's namespace
// and wakes the first controller which matches the VM labels.
func (c *VMReplicaSet) vmChangeFunc(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)
	log := logging.DefaultLogger()
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		log.Error().Object(vm).Reason(err).Msg("Failed to fetch replicasets for namespace of the VM from cache.")
		return
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, rs := range controllers {
		selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			log.Error().Object(&rs).Reason(err).Msg("Failed to fetch replicasets for namespace from cache.")
			continue
		}

		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			// The first matching rs will be informed
			key, err := cache.MetaNamespaceKeyFunc(&rs)
			if err != nil {
				log.Error().Object(&rs).Reason(err).Msg("Failed to extract key from replicaset.")
				return
			}
			c.queue.Add(key)
			return
		}

	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (c *VMReplicaSet) getCondition(rs *virtv1.VirtualMachineReplicaSet, cond virtv1.VMReplicaSetConditionType) *virtv1.VMReplicaSetCondition {
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (c *VMReplicaSet) updateStatus(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine, scaleErr error) error {

	diff := c.calcDiff(rs, vms)

	if scaleErr != nil {
		// If an error occured, only update to filtered pod count
		rs.Status.Replicas = int32(len(vms))
	} else {
		// If no error occured we have reached our required scale number
		rs.Status.Replicas = int32(len(vms) - diff)
	}

	if scaleErr != nil && c.getCondition(rs, virtv1.VMReplicaSetReplicaFailure) == nil {
		var reason string
		if diff < 0 {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}

		rs.Status.Conditions = []virtv1.VMReplicaSetCondition{
			{
				Type:               virtv1.VMReplicaSetReplicaFailure,
				Reason:             reason,
				Message:            scaleErr.Error(),
				LastTransitionTime: v1.Now(),
				Status:             k8score.ConditionTrue,
			},
		}

	} else if scaleErr == nil && c.getCondition(rs, virtv1.VMReplicaSetReplicaFailure) != nil {
		rs.Status.Conditions = []virtv1.VMReplicaSetCondition{}
	}

	_, err := c.clientset.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
	return err
}

func (c *VMReplicaSet) calcDiff(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine) int {
	// TODO default this on the aggregated api server
	wantedReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		wantedReplicas = *rs.Spec.Replicas
	}

	return len(vms) - int(wantedReplicas)
}

func (c *VMReplicaSet) getVirtualMachineBaseName(replicaset *virtv1.VirtualMachineReplicaSet) string {

	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(replicaset.Spec.Template.ObjectMeta.Name) > 0 {
		return replicaset.Spec.Template.ObjectMeta.Name
	}
	if len(replicaset.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return replicaset.Spec.Template.ObjectMeta.GenerateName
	}
	return replicaset.ObjectMeta.Name
}
