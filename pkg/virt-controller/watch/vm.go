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
	"strings"
	"time"

	"github.com/jeevatkm/go-model"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
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
	logging.DefaultLogger().Info().Msg("Starting controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.podInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
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
	isPending, err := c.execute(key.(string))

	if err != nil {
		logging.DefaultLogger().Info().Reason(err).Msgf("reenqueuing VM %v", key)
		c.queue.AddRateLimited(key)
	} else if isPending {
		logging.DefaultLogger().Info().V(4).Msgf("reenqueuing pending VM %v", key)
		c.queue.AddAfter(key, 1*time.Second)
		c.queue.Forget(key)
	} else {
		logging.DefaultLogger().Info().V(4).Msgf("processed VM %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VMController) execute(key string) (bool, error) {

	// Fetch the latest Vm state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return false, err
	}

	// Retrieve the VM
	var vm *kubev1.VirtualMachine
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return false, err
		}
		vm = kubev1.NewVMReferenceFromNameWithNS(namespace, name)
	} else {
		vm = obj.(*kubev1.VirtualMachine)
	}
	logger := logging.DefaultLogger().Object(vm)

	if !exists {
		// Delete VM Pods
		err := c.vmService.DeleteVMPod(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("Deleting VM target Pod failed.")
			return false, err
		}
		logger.Info().Msg("Deleting VM target Pod succeeded.")
		return false, nil
	}

	switch vm.Status.Phase {
	case kubev1.VmPhaseUnset, kubev1.Pending:
		// Schedule the VM

		// Deep copy the object, so that we can safely manipulate it
		vmCopy := kubev1.VirtualMachine{}
		model.Copy(&vmCopy, vm)
		logger = logging.DefaultLogger().Object(&vmCopy)

		// Check if there are already outdated VM Pods
		pods, err := c.vmService.GetRunningVMPods(&vmCopy)
		if err != nil {
			logger.Error().Reason(err).Msg("Fetching VM pods failed.")
			return false, err
		}

		// If there are already pods, delete them before continuing ...
		if len(pods.Items) > 0 {
			logger.Error().Msg("VM Pods do already exist, will clean up before continuing.")
			if err := c.vmService.DeleteVMPod(&vmCopy); err != nil {
				logger.Error().Reason(err).Msg("Deleting VM pods failed.")
				return false, err
			}
			return true, nil
		}

		// Defaulting and setting constants
		// TODO move defaulting to virt-api
		// TODO move constants to virt-handler and remove from the spec
		if vmCopy.Spec.Domain == nil {
			spec := kubev1.NewMinimalDomainSpec()
			vmCopy.Spec.Domain = spec
		}

		// TODO when we move this to virt-api, we have to block that they are set on POST or changed on PUT
		graphics := vmCopy.Spec.Domain.Devices.Graphics
		for i, _ := range graphics {
			if strings.ToLower(graphics[i].Type) == "spice" {
				graphics[i].Port = int32(-1)
				graphics[i].Listen = kubev1.Listen{
					Address: "0.0.0.0",
					Type:    "address",
				}

			}
		}

		// Create a Pod which will be the VM destination
		if err := c.vmService.StartVMPod(&vmCopy); err != nil {
			logger.Error().Reason(err).Msg("Defining a target pod for the VM failed.")
			return false, err
		}

		// Mark the VM as "initialized". After the created Pod above is scheduled by
		// kubernetes, virt-handler can take over.
		vmCopy.Status.Phase = kubev1.Scheduling
		if err := c.restClient.Put().Resource("virtualmachines").Body(&vmCopy).Name(vmCopy.ObjectMeta.Name).Namespace(vmCopy.ObjectMeta.Namespace).Do().Error(); err != nil {
			logger.Error().Reason(err).Msg("Updating the VM state to 'Scheduling' failed.")
			return false, err
		}
		logger.Info().Msg("Handing over the VM to the scheduler succeeded.")
	case kubev1.Scheduling:
		// Target Pod for the VM was already created, check if it is  running and update the VM to Scheduled

		// Deep copy the object, so that we can safely manipulate it
		vmCopy := kubev1.VirtualMachine{}
		model.Copy(&vmCopy, vm)
		logger = logging.DefaultLogger().Object(&vmCopy)

		pods, err := c.vmService.GetRunningVMPods(&vmCopy)
		if err != nil {
			logger.Error().Reason(err).Msg("Fetching VM pods failed.")
			return false, err
		}

		//TODO, we can improve the pod checks here, for now they are as good as before the refactoring
		// So far, no running Pod found, we will sooner or later get a started event.
		// If not, something is wrong and the VM, stay stuck in the Scheduling phase
		if len(pods.Items) == 0 {
			logger.Info().V(3).Msg("No VM target pod in running state found.")
			return false, nil
		}

		// Whatever is going on here, I don't know what to do, don't reprocess this
		if len(pods.Items) > 1 {
			logger.Error().V(3).Msg("More than one VM target pods found.")
			return false, nil
		}

		// Pod is not yet running
		if pods.Items[0].Status.Phase != k8sv1.PodRunning {
			return false, nil
		}

		if verifyReadiness(&pods.Items[0]) == false {
			logger.Info().V(2).Msg("Waiting on all virt-launcher containers to be marked ready")
			return false, nil
		}

		// VM got scheduled
		vmCopy.Status.Phase = kubev1.Scheduled

		// FIXME we store this in the metadata since field selectors are currently not working for TPRs
		if vmCopy.GetObjectMeta().GetLabels() == nil {
			vmCopy.ObjectMeta.Labels = map[string]string{}
		}
		vmCopy.ObjectMeta.Labels[kubev1.NodeNameLabel] = pods.Items[0].Spec.NodeName
		vmCopy.Status.NodeName = pods.Items[0].Spec.NodeName
		if _, err := c.vmService.PutVm(&vmCopy); err != nil {
			logger.Error().Reason(err).Msg("Updating the VM state to 'Scheduled' failed.")
			return false, err
		}
		logger.Info().Msgf("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
	}
	return false, nil
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
		_, hasMigrationLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.MigrationLabel]

		if phase != k8sv1.PodRunning {
			// Filter out non running pods from Queue
			return
		} else if hasMigrationLabel {
			// Filter out migration target pods
			return
		} else if hasDomainLabel == false || hasAppLabel == false {
			// missing required labels
			return
		} else if appLabel != "virt-launcher" {
			// ensure we're looking just for virt-launcher pods
			return
		}
		vmQueue.Add(namespace + "/" + domainLabel)
	}
}
