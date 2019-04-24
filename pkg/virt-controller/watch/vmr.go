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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

func NewVMRestoreController(
	vmInformer cache.SharedIndexInformer,
	vmsInformer cache.SharedIndexInformer,
	vmrInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMRestoreController {

	c := &VMRestoreController{
		Queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:   vmInformer,
		vmsInformer:  vmsInformer,
		vmrInformer:  vmrInformer,
		recorder:     recorder,
		clientset:    clientset,
		expectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmrInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMRestore,
		DeleteFunc: c.deleteVMRestore,
		UpdateFunc: c.updateVMRestore,
	})

	return c
}

type VMRestoreController struct {
	clientset    kubecli.KubevirtClient
	Queue        workqueue.RateLimitingInterface
	vmInformer   cache.SharedIndexInformer
	vmsInformer  cache.SharedIndexInformer
	vmrInformer  cache.SharedIndexInformer
	recorder     record.EventRecorder
	expectations *controller.UIDTrackingControllerExpectations
}

func (c *VMRestoreController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachineRestore controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmsInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachineRestore controller.")
}

func (c *VMRestoreController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMRestoreController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineRestore %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineRestore %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMRestoreController) execute(key string) error {
	obj, exists, err := c.vmrInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	vmr := obj.(*virtv1.VirtualMachineRestore)

	logger := log.Log.Object(vmr)
	logger.Info("Started processing the restore")

	// get all potentially interesting VMs from the cache
	obj, exists, err = c.vmInformer.GetStore().GetByKey(vmr.Namespace + "/" + vmr.Name)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		logger.Reason(err).Errorf("Failed to fetch VirtualMachine %s", vmr.Namespace + "/" + vmr.Name)
		return nil
	}
	vm := obj.(*virtv1.VirtualMachine)
	logger.Infof("found VM: %s", vm.Name)

	// get all potentially interesting VMSs from the cache
	obj, exists, err = c.vmsInformer.GetStore().GetByKey(vmr.Namespace + "/" + vmr.Spec.VirtualMachineSnapshot)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		logger.Reason(err).Errorf("Failed to fetch VirtualMachineSnapshot %s", vmr.Namespace + "/" + vmr.Spec.VirtualMachineSnapshot)
		return nil
	}
	vms := obj.(*virtv1.VirtualMachineSnapshot)
	logger.Infof("found VMS: %s", vms.Name)

	// check whether it can do snapshot
	doRestore, err := shouldDoRestore(vmr, vms, vm)
	var restoredOn metav1.Time
	var restored, updatedStatus bool

	logger.Infof("Doing restore: %t", doRestore)

	if doRestore {
		restored, err = c.doRestore(vmr, vms, vm)
		if err != nil {
			log.Log.Object(vm).Errorf("Cannot restore VM: %s", err.Error())
		}
		if restored {
			logger.Infof("Restored VM: %s", vm.Name)
			restoredOn = metav1.Now()
		}
	}

	if restored {
		vmr, updatedStatus = updateVMRestoreStatus(vmr, restoredOn)
	}

	if updatedStatus {
		// update the snapshot in cluster
		err = c.restUpdateVirtualMachineRestore(vmr)
		if err != nil {
			logger.Errorf("Cannot update the VirtualMachineRestore in ETCD: %s", err.Error())
		}
	}

	return nil
}

func (c *VMRestoreController) addVMRestore(obj interface{}) {
	c.enqueueVMRestore(obj)
}

func (c *VMRestoreController) deleteVMRestore(obj interface{}) {
	c.enqueueVMRestore(obj)
}

func (c *VMRestoreController) updateVMRestore(old, curr interface{}) {
	c.enqueueVMRestore(curr)
}

func (c *VMRestoreController) enqueueVMRestore(obj interface{}) {
	logger := log.Log
	vmr := obj.(*virtv1.VirtualMachineRestore)
	key, err := controller.KeyFunc(vmr)
	if err != nil {
		logger.Object(vmr).Reason(err).Error("Failed to extract vmrKey from VirtualMachineRestore.")
	}
	c.Queue.Add(key)
}

// restUpdateVirtualMachineSnapshot call the REST API and update the VirtualMachineSnapshot in the cluster
func (c *VMRestoreController) restUpdateVirtualMachineRestore(vmr *virtv1.VirtualMachineRestore) error {
	_, err := c.clientset.VirtualMachineRestore(vmr.ObjectMeta.Namespace).Update(vmr)
	return err
}

func (c *VMRestoreController) doRestore(vmr *virtv1.VirtualMachineRestore, vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (bool, error) {

	newVM := vms.Status.VirtualMachine
	newVM.ResourceVersion = vm.ResourceVersion

	updatedVM, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Update(newVM)
	if err != nil {
		log.Log.Errorf("Cannot update VM: %s", newVM.Name)
		return false, err
	}

	log.Log.Object(vm).Infof("updatedVM: %s", updatedVM.Spec.Template.Spec.Domain.Resources.Requests.Memory())

	return true, nil
}

func (c *VMRestoreController) updateResourceVersion(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	obj, exist, err := c.vmInformer.GetStore().GetByKey(vm.Namespace + "/" + vm.Name)
	if !exist {
		log.Log.Object(vm).Errorf("VM %s not found in cache.", vm.Name)
	}
	if err != nil {
		log.Log.Object(vm).Errorf("Cannot load current VM: %s from cache %s", vm.Name, err.Error())
		return nil, err
	}
	clusterVM := obj.(*virtv1.VirtualMachine)
	updatedVM := vm.DeepCopy()
	updatedVM.ResourceVersion = clusterVM.ResourceVersion

	return clusterVM, nil
}

func shouldDoRestore(vmr *virtv1.VirtualMachineRestore, vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (bool, error) {
	if vmr.Status.RestoredOn != nil {
		// VirtualMachine already restored from this restore
		// configuration. Cannot perform another one.
		log.Log.Infof("VM %s is already restored", vm.Name)
		return false, nil
	}

	if vm.Spec.Running == true {
		// VirtualMachine have to be shutdown to perform restore
		log.Log.Infof("VM: %s is running", vm.Name)
		return false, nil
	}

	if vms.Status.VirtualMachine == nil {
		// Nothing to restore from
		log.Log.Infof("VMS: %s does not have VM cloned", vms.Name)
		return false, nil
	}

	return true, nil
}

func updateVMRestoreStatus(vmr *virtv1.VirtualMachineRestore, restoredOn metav1.Time) (*virtv1.VirtualMachineRestore, bool) {
	updatedVMR := vmr.DeepCopy()
	updatedVMR.Status.RestoredOn = &restoredOn

	return updatedVMR, true
}
