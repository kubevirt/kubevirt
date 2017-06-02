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
	"fmt"
	"net/http"
	"strings"

	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

func NewVMController(lw cache.ListerWatcher, domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient, clientset *kubernetes.Clientset, host string) (cache.Store, workqueue.RateLimitingInterface, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	dispatch := NewVMHandlerDispatch(domainManager, recorder, &restClient, clientset, host)

	indexer, informer := kubecli.NewController(lw, queue, &v1.VM{}, dispatch)
	return indexer, queue, informer

}
func NewVMHandlerDispatch(domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient *rest.RESTClient, clientset *kubernetes.Clientset, host string) kubecli.ControllerDispatch {
	return &VMHandlerDispatch{
		domainManager: domainManager,
		recorder:      recorder,
		restClient:    *restClient,
		clientset:     clientset,
		host:          host,
	}
}

type VMHandlerDispatch struct {
	domainManager virtwrap.DomainManager
	recorder      record.EventRecorder
	restClient    rest.RESTClient
	clientset     *kubernetes.Clientset
	host          string
}

func (d *VMHandlerDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {

	// Fetch the latest Vm state from cache
	obj, exists, err := store.GetByKey(key.(string))

	if err != nil {
		queue.AddRateLimited(key)
		return
	}

	// Retrieve the VM
	var vm *v1.VM
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			// TODO do something more smart here
			queue.AddRateLimited(key)
			return
		}
		vm = v1.NewVMReferenceFromName(name)

		// If we don't have the VM in the cache, it could be that it is currently migrating to us
		result := d.restClient.Get().Name(vm.GetObjectMeta().GetName()).Resource("vms").Namespace(kubeapi.NamespaceDefault).Do()
		if result.Error() == nil {
			// So the VM still seems to exist
			fetchedVM, err := result.Get()
			if err != nil {
				// Since there was no fetch error, this should have worked, let's back off
				queue.AddRateLimited(key)
				return
			}
			if fetchedVM.(*v1.VM).Status.MigrationNodeName == d.host {
				// OK, this VM is migrating to us, don't interrupt it
				queue.Forget(key)
				return
			}
		} else if result.Error().(*errors.StatusError).Status().Code != int32(http.StatusNotFound) {
			// Something went wrong, let's try again later
			queue.AddRateLimited(key)
			return
		}
		// The VM is deleted on the cluster, let's go on with the deletion on the host
	} else {
		vm = obj.(*v1.VM)
	}
	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Processing VM update.")

	// Process the VM
	if !exists {
		// Since the VM was not in the cache, we delete it
		err = d.domainManager.KillVM(vm)
	} else if isWorthSyncing(vm) {
		// Synchronize the VM state
		vm, err = MapPersistentVolumes(vm, d.clientset.CoreV1().RESTClient(), kubeapi.NamespaceDefault)

		if err == nil {
			// TODO check if found VM has the same UID like the domain, if not, delete the Domain first

			// Only sync if the VM is not marked as migrating. Everything except shutting down the VM is not permitted when it is migrating.
			// TODO MigrationNodeName should be a pointer
			if vm.Status.MigrationNodeName == "" {
				err = d.domainManager.SyncVM(vm)
			} else {
				queue.Forget(key)
				return
			}
		}

		// Update VM status to running
		if err == nil && vm.Status.Phase != v1.Running {
			obj, err = kubeapi.Scheme.Copy(vm)
			if err == nil {
				vm = obj.(*v1.VM)
				vm.Status.Phase = v1.Running
				err = d.restClient.Put().Resource("vms").Body(vm).
					Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
			}
		}
	}

	if err != nil {
		// Something went wrong, reenqueue the item with a delay
		logging.DefaultLogger().Error().Object(vm).Reason(err).Msg("Synchronizing the VM failed.")
		d.recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
		queue.AddRateLimited(key)
		return
	}

	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Synchronizing the VM succeeded.")
	queue.Forget(key)
	return
}

// Almost everything in the VM object maps exactly to its domain counterpart
// One exception is persistent volume claims. This function looks up each PV
// and inserts a corrected disk entry into the VM's device map.
func MapPersistentVolumes(vm *v1.VM, restClient cache.Getter, namespace string) (*v1.VM, error) {
	vmCopy := &v1.VM{}
	model.Copy(vmCopy, vm)

	for idx, disk := range vmCopy.Spec.Domain.Devices.Disks {
		if disk.Type == "PersistentVolumeClaim" {
			logging.DefaultLogger().V(3).Info().Object(vm).Msgf("Mapping PersistentVolumeClaim: %s", disk.Source.Name)

			// Look up existing persistent volume
			obj, err := restClient.Get().Namespace(namespace).Resource("persistentvolumeclaims").Name(disk.Source.Name).Do().Get()

			if err != nil {
				logging.DefaultLogger().Error().Reason(err).Msg("unable to look up persistent volume claim")
				return vm, fmt.Errorf("unable to look up persistent volume claim: %v", err)
			}

			pvc := obj.(*kubev1.PersistentVolumeClaim)
			if pvc.Status.Phase != kubev1.ClaimBound {
				logging.DefaultLogger().Error().Msg("attempted use of unbound persistent volume")
				return vm, fmt.Errorf("attempted use of unbound persistent volume claim: %s", pvc.Name)
			}

			// Look up the PersistentVolume this PVC is bound to
			// Note: This call is not namespaced!
			obj, err = restClient.Get().Resource("persistentvolumes").Name(pvc.Spec.VolumeName).Do().Get()

			if err != nil {
				logging.DefaultLogger().Error().Reason(err).Msg("unable to access persistent volume record")
				return vm, fmt.Errorf("unable to access persistent volume record: %v", err)
			}
			pv := obj.(*kubev1.PersistentVolume)

			if pv.Spec.ISCSI != nil {
				logging.DefaultLogger().Object(vm).Info().Msg("Mapping iSCSI PVC")
				newDisk := v1.Disk{}

				newDisk.Type = "network"
				newDisk.Device = "disk"
				newDisk.Target = disk.Target
				newDisk.Driver = new(v1.DiskDriver)
				newDisk.Driver.Type = "raw"
				newDisk.Driver.Name = "qemu"

				newDisk.Source.Name = fmt.Sprintf("%s/%d", pv.Spec.ISCSI.IQN, pv.Spec.ISCSI.Lun)
				newDisk.Source.Protocol = "iscsi"

				hostPort := strings.Split(pv.Spec.ISCSI.TargetPortal, ":")
				newDisk.Source.Host = &v1.DiskSourceHost{}
				newDisk.Source.Host.Name = hostPort[0]
				if len(hostPort) > 1 {
					newDisk.Source.Host.Port = hostPort[1]
				}

				vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
			} else {
				logging.DefaultLogger().Object(vm).Error().Msg(fmt.Sprintf("Referenced PV %v is backed by an unsupported storage type", pv))
			}
		}
	}

	return vmCopy, nil
}

func isWorthSyncing(vm *v1.VM) bool {
	return vm.Status.Phase != v1.Succeeded && vm.Status.Phase != v1.Failed
}
