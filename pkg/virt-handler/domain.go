/*
 * This file is part of the kubevirt project
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

package virthandler

import (
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

/*
TODO: Define the exact scope of this controller.
For now it looks like we should use domain events to detect unexpected domain changes like crashes or vms going
into pause mode because of resource shortage or cut off connections to storage.
*/
func NewDomainController(vmQueue workqueue.RateLimitingInterface, vmStore cache.Store, informer cache.SharedInformer, restClient rest.RESTClient, recorder record.EventRecorder) (cache.Store, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForWorkqueue(queue))
	dispatch := NewDomainDispatch(vmQueue, vmStore, restClient, recorder)
	return kubecli.NewControllerFromInformer(informer.GetStore(), informer, queue, dispatch)
}

func NewDomainDispatch(vmQueue workqueue.RateLimitingInterface, vmStore cache.Store, restClient rest.RESTClient, recorder record.EventRecorder) kubecli.ControllerDispatch {
	return &DomainDispatch{
		vmQueue:    vmQueue,
		vmStore:    vmStore,
		recorder:   recorder,
		restClient: restClient,
	}
}

type DomainDispatch struct {
	vmQueue    workqueue.RateLimitingInterface
	vmStore    cache.Store
	recorder   record.EventRecorder
	restClient rest.RESTClient
}

func (d *DomainDispatch) Execute(indexer cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {
	obj, exists, err := indexer.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
		return
	}

	var domain *api.Domain
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			queue.AddRateLimited(key)
			return
		}
		domain = api.NewDomainReferenceFromName(name)
		logging.DefaultLogger().Info().Object(domain).Msgf("Domain deleted")
	} else {
		domain = obj.(*api.Domain)
		logging.DefaultLogger().Info().Object(domain).Msgf("Domain is in state %s reason %s", domain.Status.Status, domain.Status.Reason)
	}
	obj, vmExists, err := d.vmStore.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
		return
	}
	if !vmExists || obj.(*v1.VM).GetObjectMeta().GetUID() != domain.GetObjectMeta().GetUID() {
		// The VM is not in the vm cache, or is a VM with a differend uuid, tell the VM controller to investigate it
		d.vmQueue.Add(key)
	} else {
		err := d.setVmPhaseForStatusReason(domain, obj.(*v1.VM))
		if err != nil {
			queue.AddRateLimited(key)
		}
	}

	return
}

func (d *DomainDispatch) setVmPhaseForStatusReason(domain *api.Domain, vm *v1.VM) error {
	flag := false
	if domain.Status.Status == api.Shutoff || domain.Status.Status == api.Crashed {
		switch domain.Status.Reason {
		case api.ReasonCrashed, api.ReasonPanicked:
			vm.Status.Phase = v1.Failed
			d.recorder.Event(vm, k8sv1.EventTypeWarning, v1.Stopped.String(), "The VM crashed.")
			flag = true
		case api.ReasonShutdown, api.ReasonDestroyed, api.ReasonSaved, api.ReasonFromSnapshot:
			vm.Status.Phase = v1.Succeeded
			d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Stopped.String(), "The VM was shut down.")
			flag = true
		}
	}

	if flag {
		logging.DefaultLogger().Info().Object(vm).Msgf("Changing VM phase to %s", vm.Status.Phase)
		return d.restClient.Put().Resource("vms").Body(vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
	}

	return nil
}
