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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func NewVMController(restClient *rest.RESTClient, vmService services.VMService, queue workqueue.RateLimitingInterface, vmCache cache.Store, vmInformer cache.SharedIndexInformer, podInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient) *VMController {
	return &VMController{
		restClient:  restClient,
		vmService:   vmService,
		queue:       queue,
		store:       vmCache,
		vmInformer:  vmInformer,
		podInformer: podInformer,
		recorder:    recorder,
		clientset:   clientset,
	}
}

type VMController struct {
	restClient  *rest.RESTClient
	vmService   services.VMService
	clientset   kubecli.KubevirtClient
	queue       workqueue.RateLimitingInterface
	store       cache.Store
	vmInformer  cache.SharedIndexInformer
	podInformer cache.SharedIndexInformer
	recorder    record.EventRecorder
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

	// Retrieve the VM
	var vm *kubev1.VirtualMachine
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		vm = kubev1.NewVMReferenceFromNameWithNS(namespace, name)
	} else {
		vm = obj.(*kubev1.VirtualMachine)
	}

	// If the VM is exists still, don't process the VM until it is fully initialized.
	// Initialization is handled by the initialization controller and must take place
	// before the VM is acted upon.
	if exists && !isVirtualMachineInitialized(vm) {
		return nil
	}

	logger := log.Log.Object(vm)

	if !exists {
		// Delete VM Pods
		err := c.vmService.DeleteVMPod(vm)
		if err != nil {
			logger.Reason(err).Error("Deleting VM target Pod failed.")
			return err
		}
		logger.Info("Deleting VM target Pod succeeded.")
		return nil
	}

	switch vm.Status.Phase {
	case kubev1.VmPhaseUnset, kubev1.Pending:
		// Schedule the VM

		// Deep copy the object, so that we can safely manipulate it
		vmCopy := vm.DeepCopy()
		logger = log.Log.Object(vmCopy)

		// Check if there are already outdated VM Pods
		pods, err := c.vmService.GetRunningVMPods(vmCopy)
		if err != nil {
			logger.Reason(err).Error("Fetching VM pods failed.")
			return err
		}

		// If there are already pods, delete them before continuing ...
		if len(pods.Items) > 0 {
			logger.Error("VM Pods do already exist, will clean up before continuing.")
			if err := c.vmService.DeleteVMPod(vmCopy); err != nil {
				logger.Reason(err).Error("Deleting VM pods failed.")
				return err
			}
			// the pod informer will reenqueue the key as a result of it being deleted.
			return nil
		}

		// Defaulting and setting constants
		// TODO move defaulting to virt-api
		kubev1.SetObjectDefaults_VirtualMachine(vmCopy)

		// Create a Pod which will be the VM destination
		if err := c.vmService.StartVMPod(vmCopy); err != nil {
			logger.Reason(err).Error("Defining a target pod for the VM failed.")
			return err
		}

		// Mark the VM as "initialized". After the created Pod above is scheduled by
		// kubernetes, virt-handler can take over.
		vmCopy.Status.Phase = kubev1.Scheduling
		if _, err := c.clientset.VM(vm.Namespace).Update(vmCopy); err != nil {
			logger.Reason(err).Error("Updating the VM state to 'Scheduling' failed.")
			return err
		}
		logger.Info("Handing over the VM to the scheduler succeeded.")
	case kubev1.Scheduling:
		// Target Pod for the VM was already created, check if it is  running and update the VM to Scheduled

		// Deep copy the object, so that we can safely manipulate it
		vmCopy := vm.DeepCopy()
		logger = log.Log.Object(vmCopy)

		pods, err := c.vmService.GetRunningVMPods(vmCopy)
		if err != nil {
			logger.Reason(err).Error("Fetching VM pods failed.")
			return err
		}

		//TODO, we can improve the pod checks here, for now they are as good as before the refactoring
		// So far, no running Pod found, we will sooner or later get a started event.
		// If not, something is wrong and the VM, stay stuck in the Scheduling phase
		if len(pods.Items) == 0 {
			logger.V(3).Info("No VM target pod in running state found.")
			return nil
		}

		// If this occurs, the podinformer should reenqueue the key
		// if one of these pods terminates. This will let virt-controller continue
		// processing the VM.
		if len(pods.Items) > 1 {
			logger.V(3).Error("More than one VM target pods found.")
			return nil
		}

		// Pod is not yet running
		if pods.Items[0].Status.Phase != k8sv1.PodRunning {
			return nil
		}

		if verifyReadiness(&pods.Items[0]) == false {
			logger.V(2).Info("Waiting on all virt-launcher containers to be marked ready")
			return nil
		}

		// VM got scheduled
		vmCopy.Status.Phase = kubev1.Scheduled

		// FIXME we store this in the metadata since field selectors are currently not working for TPRs
		if vmCopy.GetObjectMeta().GetLabels() == nil {
			vmCopy.ObjectMeta.Labels = map[string]string{}
		}
		vmCopy.ObjectMeta.Labels[kubev1.NodeNameLabel] = pods.Items[0].Spec.NodeName
		vmCopy.Status.NodeName = pods.Items[0].Spec.NodeName
		// Copy the POD IP address to the VM
		iface := kubev1.VirtualMachineNetworkInterface{}
		iface.IP = pods.Items[0].Status.PodIP
		vmCopy.Status.Interfaces = []kubev1.VirtualMachineNetworkInterface{iface}

		if _, err := c.vmService.PutVm(vmCopy); err != nil {
			logger.Reason(err).Error("Updating the VM state to 'Scheduled' failed.")
			return err
		}
		logger.Infof("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
	case kubev1.Failed, kubev1.Succeeded:
		err := c.vmService.DeleteVMPod(vm)
		if err != nil {
			logger.Reason(err).Error("Deleting VM target Pod failed.")
			return err
		}
		logger.Info("Deleted VM target Pod for VM in finalized state.")
	}
	return nil
}

func verifyReadiness(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Ready == false {
			return false
		}
	}

	return true
}

func vmLabelHandler(vmQueue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		phase := obj.(*k8sv1.Pod).Status.Phase
		namespace := obj.(*k8sv1.Pod).ObjectMeta.Namespace
		appLabel, hasAppLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.AppLabel]
		domainLabel, hasDomainLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.DomainLabel]

		deleted := false
		if obj.(*k8sv1.Pod).GetObjectMeta().GetDeletionTimestamp() != nil {
			deleted = true
		}

		if hasDomainLabel == false || hasAppLabel == false {
			// missing required labels
			return
		} else if appLabel != "virt-launcher" {
			// ensure we're looking just for virt-launcher pods
			return
		} else if phase != k8sv1.PodRunning && deleted == false {
			// Filter out non running pods from Queue that aren't deleted
			return
		}
		vmQueue.Add(namespace + "/" + domainLabel)
	}
}
