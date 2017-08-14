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

package virthandler

import (
	"fmt"
	"net"
	"strings"

	"github.com/jeevatkm/go-model"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
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

func (d *VMHandlerDispatch) getVMNodeAddress(vm *v1.VM) (string, error) {
	node, err := d.clientset.CoreV1().Nodes().Get(vm.Status.NodeName, metav1.GetOptions{})
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msgf("fetching source node %s failed", vm.Status.NodeName)
		return "", err
	}

	addrStr := ""
	for _, addr := range node.Status.Addresses {
		if (addr.Type == k8sv1.NodeInternalIP) && (addrStr == "") {
			addrStr = addr.Address
			break
		}
	}
	if addrStr == "" {
		err := fmt.Errorf("VM node is unreachable")
		logging.DefaultLogger().Error().Msg("VM node is unreachable")
		return "", err
	}

	return addrStr, nil
}

func (d *VMHandlerDispatch) updateVMStatus(vm *v1.VM, cfg *api.DomainSpec) error {
	obj, err := scheme.Scheme.Copy(vm)
	if err != nil {
		return err
	}
	vm = obj.(*v1.VM)

	// XXX When we start supporting hotplug, this needs to be altered.
	// Check if the VM is already marked as running. If yes, don't update the VM.
	// Otherwise we end up in endless controller requeues.
	if vm.Status.Phase == v1.Running {
		return nil
	}

	vm.Status.Phase = v1.Running

	vm.Status.Graphics = []v1.VMGraphics{}

	podIP, err := d.getVMNodeAddress(vm)
	if err != nil {
		return err
	}

	for _, src := range cfg.Devices.Graphics {
		if (src.Type != "spice" && src.Type != "vnc") || src.Port == -1 {
			continue
		}
		dst := v1.VMGraphics{
			Type: src.Type,
			Host: podIP,
			Port: src.Port,
		}
		vm.Status.Graphics = append(vm.Status.Graphics, dst)
	}

	return d.restClient.Put().Resource("vms").Body(vm).
		Name(vm.ObjectMeta.Name).Namespace(vm.ObjectMeta.Namespace).Do().Error()

}

func (d *VMHandlerDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {

	shouldDeleteVm := false

	// Fetch the latest Vm state from cache
	obj, exists, err := store.GetByKey(key.(string))

	if err != nil {
		queue.AddRateLimited(key)
		return
	}

	// Retrieve the VM
	var vm *v1.VM
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			// TODO do something more smart here
			queue.AddRateLimited(key)
			return
		}
		vm = v1.NewVMReferenceFromNameWithNS(namespace, name)
	} else {
		vm = obj.(*v1.VM)
	}

	// Check For Migration before processing vm not in our cache
	if !exists {
		// If we don't have the VM in the cache, it could be that it is currently migrating to us
		isDestination, err := d.isMigrationDestination(vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
		if err != nil {
			// unable to determine migration status, we'll try again later.
			queue.AddRateLimited(key)
			return
		}

		if isDestination {
			// OK, this VM is migrating to us, don't interrupt it.
			queue.Forget(key)
			return
		}
		// The VM is deleted on the cluster, continue with processing the deletion on the host.
		shouldDeleteVm = true
	}
	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Processing VM update.")

	// Process the VM
	err = d.processVmUpdate(vm, shouldDeleteVm)
	if err != nil {
		// Something went wrong, reenqueue the item with a delay
		logging.DefaultLogger().Error().Object(vm).Reason(err).Msg("Synchronizing the VM failed.")
		d.recorder.Event(vm, k8sv1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
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

			pvc := obj.(*k8sv1.PersistentVolumeClaim)
			if pvc.Status.Phase != k8sv1.ClaimBound {
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
			pv := obj.(*k8sv1.PersistentVolume)

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
				ipAddrs, err := net.LookupIP(hostPort[0])
				if err != nil || len(ipAddrs) < 1 {
					logging.DefaultLogger().Error().Reason(err).Msgf("Unable to resolve host '%s'", hostPort[0])
					return vm, fmt.Errorf("Unable to resolve host '%s': %s", hostPort[0], err)
				}

				newDisk.Source.Host = &v1.DiskSourceHost{}
				newDisk.Source.Host.Name = ipAddrs[0].String()
				if len(hostPort) > 1 {
					newDisk.Source.Host.Port = hostPort[1]
				}

				vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
			} else {
				logging.DefaultLogger().Object(vm).Error().Msg(fmt.Sprintf("Referenced PV %v is backed by an unsupported storage type", pv))
			}
		} else if disk.Type == "network" {
			newDisk := v1.Disk{}
			model.Copy(&newDisk, disk)

			if disk.Source.Host == nil {
				logging.DefaultLogger().Error().Msg("Missing disk source host")
				return vm, fmt.Errorf("Missing disk source host")
			}

			ipAddrs, err := net.LookupIP(disk.Source.Host.Name)
			if err != nil || ipAddrs == nil || len(ipAddrs) < 1 {
				logging.DefaultLogger().Error().Reason(err).Msgf("Unable to resolve host '%s'", disk.Source.Host.Name)
				return vm, fmt.Errorf("Unable to resolve host '%s': %s", disk.Source.Host.Name, err)
			}

			newDisk.Source.Host.Name = ipAddrs[0].String()

			vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
		}
	}

	return vmCopy, nil
}

func (d *VMHandlerDispatch) processVmUpdate(vm *v1.VM, shouldDeleteVm bool) error {

	if shouldDeleteVm {
		// Since the VM was not in the cache, we delete it
		return d.domainManager.KillVM(vm)
	} else if isWorthSyncing(vm) == false {
		// nothing to do here.
		return nil
	}

	// Synchronize the VM state
	vm, err := MapPersistentVolumes(vm, d.clientset.CoreV1().RESTClient(), vm.ObjectMeta.Namespace)
	if err != nil {
		return err
	}

	// Map Container Registry Disks to block devices Libvirt can consume
	vm, err = registrydisk.MapRegistryDisks(vm)
	if err != nil {
		return err
	}

	// Map whatever devices are being used for config-init
	vm, err = cloudinit.InjectDomainData(vm)
	if err != nil {
		return err
	}

	// TODO MigrationNodeName should be a pointer
	if vm.Status.MigrationNodeName != "" {
		// Only sync if the VM is not marked as migrating.
		// Everything except shutting down the VM is not
		// permitted when it is migrating.
		return nil
	}

	// TODO check if found VM has the same UID like the domain,
	// if not, delete the Domain first
	newCfg, err := d.domainManager.SyncVM(vm)
	if err != nil {
		return err
	}

	return d.updateVMStatus(vm, newCfg)
}

func (d *VMHandlerDispatch) isMigrationDestination(namespace string, vmName string) (bool, error) {

	// If we don't have the VM in the cache, it could be that it is currently migrating to us
	result := d.restClient.Get().Name(vmName).Resource("vms").Namespace(namespace).Do()
	if result.Error() == nil {
		// So the VM still seems to exist
		fetchedVM, err := result.Get()
		if err != nil {
			return false, err
		}
		if fetchedVM.(*v1.VM).Status.MigrationNodeName == d.host {
			return true, nil
		}
	} else if !errors.IsNotFound(result.Error()) {
		// Something went wrong, let's try again later
		return false, result.Error()
	}

	// VM object was not found.
	return false, nil
}

func isWorthSyncing(vm *v1.VM) bool {
	return vm.Status.Phase != v1.Succeeded && vm.Status.Phase != v1.Failed
}
