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

	"github.com/jeevatkm/go-model"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
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
	logging.DefaultLogger().Info().Msg("Starting controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmRSInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
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
		logging.DefaultLogger().Info().Reason(err).Msgf("reenqueuing VirtualMachineReplicaSet %v", key)
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
	rs := obj.(*kubev1.VirtualMachineReplicaSet)

	//TODO default rs if necessary, the aggregated apiserver will do that in the future
	if rs.Spec.Template == nil || rs.Spec.Selector == nil || rs.Spec.Selector.Size() == 0 {
		logging.DefaultLogger().Object(rs).Error().Msg("Invalid controller spec, will not retry processing it.")
		return nil
	}

	//TODO should be added when creating the controller
	// This is a default value
	wantedReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		wantedReplicas = *rs.Spec.Replicas
	}
	selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		return nil
	}

	// get all potentially interesting VMs from the cache
	vms, err := c.listPodsFromNamespace(rs.ObjectMeta.Namespace)

	if err != nil {
		return err
	}

	// make sure we only consider active VMs
	vms = c.filterActiveVMs(vms)

	// make sure the selector of the controller matches and the VMs match
	vms = c.filterMatchingVMs(selector, vms)

	vmCount := int32(len(vms))
	if vmCount < wantedReplicas {
		// create VM
		vm := kubev1.NewVMReferenceFromNameWithNS(rs.ObjectMeta.Namespace, "")
		vm.ObjectMeta.GenerateName = rs.ObjectMeta.Name + "-"
		vm.Spec = rs.Spec.Template.Spec
		// TODO check if vm labels exist, and when make sure that they match. For now just override them
		vm.ObjectMeta.Labels = rs.Spec.Template.ObjectMeta.Labels
		_, err := c.clientset.VM(rs.ObjectMeta.Namespace).Create(vm)
		if err != nil {
			return err
		}
		vmCount += 1
	} else if vmCount > wantedReplicas {
		// delete VM
		// TODO graceful delete only
		deleteCandidate := vms[0]
		if err != nil {
			return err
		}
		c.clientset.VM(rs.ObjectMeta.Namespace).Delete(deleteCandidate.ObjectMeta.Name, &v1.DeleteOptions{})
		vmCount -= 1
	}
	if rs.Status.Replicas != vmCount {
		obj, err = model.Clone(rs)
		if err != nil {
			return err
		}
		rsCopy := obj.(*kubev1.VirtualMachineReplicaSet)
		rsCopy.Status.Replicas = vmCount
		_, err := c.clientset.ReplicaSet(rs.ObjectMeta.Namespace).Update(rsCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

// filterActiveVMs takes a list of VMs and returns all VMs which are not in a final state
// Note that vms which have a deletion timestamp set, are still treated as active.
// This is a difference to Pod ReplicaSets
func (c *VMReplicaSet) filterActiveVMs(vms []kubev1.VM) []kubev1.VM {
	filtered := []kubev1.VM{}
	for _, vm := range vms {
		if !vm.IsFinal() {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// filterMatchingVMs takes a selector and a list of VMs. If the VM labels match the selector it is added to the filtered collection.
// Returns the list of all VMs which match the selector
func (c *VMReplicaSet) filterMatchingVMs(selector labels.Selector, vms []kubev1.VM) []kubev1.VM {
	//TODO take owner reference into account
	filtered := []kubev1.VM{}
	for _, vm := range vms {
		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// listPodsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
func (c *VMReplicaSet) listPodsFromNamespace(namespace string) ([]kubev1.VM, error) {
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []kubev1.VM{}
	for _, obj := range objs {
		vms = append(vms, *obj.(*kubev1.VM))
	}
	return vms, nil
}

// listControllerFromNamespace takes a namespace and returns all VMReplicaSets from the ReplicaSet cache which run in this namespace
func (c *VMReplicaSet) listControllerFromNamespace(namespace string) ([]kubev1.VirtualMachineReplicaSet, error) {
	objs, err := c.vmRSInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	replicaSets := []kubev1.VirtualMachineReplicaSet{}
	for _, obj := range objs {
		rs := obj.(*kubev1.VirtualMachineReplicaSet)
		replicaSets = append(replicaSets, *rs)
	}
	return replicaSets, nil
}

// vmChangeFunc checks if the supplied VM matches a replica set controller in it's namespace
// and wakes the first controller it encounters.
func (c *VMReplicaSet) vmChangeFunc(obj interface{}) {
	vm := obj.(*kubev1.VM)
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		//TODO error handling
		return
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, rs := range controllers {
		selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			// selector is invalid, continue with next controller
			continue
		}

		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			// The first matching rs will be informed
			key, err := cache.MetaNamespaceKeyFunc(&rs)
			if err != nil {
				return
			}
			c.queue.Add(key)
			return
		}

	}
}
