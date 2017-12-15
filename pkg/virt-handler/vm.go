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
	goerror "errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jeevatkm/go-model"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	configdisk "kubevirt.io/kubevirt/pkg/config-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	watchdog "kubevirt.io/kubevirt/pkg/watchdog"
)

func NewController(
	domainManager virtwrap.DomainManager,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	configDiskClient configdisk.ConfigDiskClient,
	virtShareDir string,
	watchdogTimeoutSeconds int,
	vmInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	watchdogInformer cache.SharedIndexInformer,
	gracefulShutdownInformer cache.SharedIndexInformer,
) *VirtualMachineController {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := &VirtualMachineController{
		Queue:                    queue,
		domainManager:            domainManager,
		recorder:                 recorder,
		clientset:                clientset,
		host:                     host,
		configDisk:               configDiskClient,
		virtShareDir:             virtShareDir,
		watchdogTimeoutSeconds:   watchdogTimeoutSeconds,
		vmInformer:               vmInformer,
		domainInformer:           domainInformer,
		watchdogInformer:         watchdogInformer,
		gracefulShutdownInformer: gracefulShutdownInformer,
	}

	vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		DeleteFunc: c.deleteDomainFunc,
		UpdateFunc: c.updateDomainFunc,
	})

	watchdogInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	gracefulShutdownInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	return c
}

type VirtualMachineController struct {
	domainManager            virtwrap.DomainManager
	recorder                 record.EventRecorder
	clientset                kubecli.KubevirtClient
	host                     string
	configDisk               configdisk.ConfigDiskClient
	virtShareDir             string
	watchdogTimeoutSeconds   int
	Queue                    workqueue.RateLimitingInterface
	vmInformer               cache.SharedIndexInformer
	domainInformer           cache.SharedInformer
	watchdogInformer         cache.SharedIndexInformer
	gracefulShutdownInformer cache.SharedIndexInformer
}

// Determines if a domain's grace period has expired during shutdown.
// If the grace period has started but not expired, timeLeft represents
// the time in seconds left until the period expires.
// If the grace period has not started, timeLeft will be set to -1.
func (d *VirtualMachineController) hasGracePeriodExpired(dom *api.Domain) (hasExpired bool, timeLeft int) {

	hasExpired = false
	timeLeft = 0

	if dom == nil {
		hasExpired = true
		return
	}

	startTime := int64(0)
	if dom.Spec.Metadata.GracePeriod.DeletionTimestamp != nil {
		startTime = dom.Spec.Metadata.GracePeriod.DeletionTimestamp.UTC().Unix()
	}
	gracePeriod := dom.Spec.Metadata.GracePeriod.DeletionGracePeriodSeconds

	// If gracePeriod == 0, then there will be no startTime set, deletion
	// should occur immediately during shutdown.
	if gracePeriod == 0 {
		hasExpired = true
		return
	} else if startTime == 0 {
		// If gracePeriod > 0, then the shutdown signal needs to be sent
		// and the gracePeriod start time needs to be set.
		timeLeft = -1
		return
	}

	now := time.Now().UTC().Unix()
	diff := now - startTime

	if diff >= gracePeriod {
		hasExpired = true
		return
	}

	timeLeft = int(gracePeriod - diff)
	if timeLeft < 1 {
		timeLeft = 1
	}
	return
}

func (d *VirtualMachineController) getVMNodeAddress(vm *v1.VirtualMachine) (string, error) {
	node, err := d.clientset.CoreV1().Nodes().Get(vm.Status.NodeName, metav1.GetOptions{})
	if err != nil {
		log.Log.Reason(err).Errorf("fetching source node %s failed", vm.Status.NodeName)
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
		log.Log.Error("VM node is unreachable")
		return "", err
	}

	return addrStr, nil
}

func (d *VirtualMachineController) updateVMStatus(vm *v1.VirtualMachine, domain *api.Domain, syncError error) (err error) {

	// Don't update the VM if it is already in a final state
	if vm.IsFinal() {
		return nil
	}

	// While the VM is migrating, don't do anything, the Migration Controller is in charge
	if vm.Status.MigrationNodeName != "" {
		return nil
	}

	oldStatus := vm.DeepCopy().Status
	// Make sure that we always deal with an empty instance for later equality checks
	if oldStatus.Graphics == nil {
		oldStatus.Graphics = []v1.VMGraphics{}
	}

	// Calculate the new VM state based on what libvirt reported
	d.setVmPhaseForStatusReason(domain, vm)

	vm.Status.Graphics = []v1.VMGraphics{}

	// Update devices if device status changed
	// TODO needs caching, better position or init fetch
	if domain != nil {
		nodeIP, err := d.getVMNodeAddress(vm)
		if err != nil {
			return err
		}

		vm.Status.Graphics = []v1.VMGraphics{}
		for _, src := range domain.Spec.Devices.Graphics {
			if (src.Type != "spice" && src.Type != "vnc") || src.Port == -1 {
				continue
			}
			dst := v1.VMGraphics{
				Type: src.Type,
				Host: nodeIP,
				Port: src.Port,
			}
			vm.Status.Graphics = append(vm.Status.Graphics, dst)
		}
	}

	d.checkFailure(vm, syncError, "Synchronizing with the Domain failed.")

	if !reflect.DeepEqual(oldStatus, vm.Status) {
		_, err = d.clientset.VM(vm.ObjectMeta.Namespace).Update(vm)
		if err != nil {
			return err
		}
	}

	if oldStatus.Phase != vm.Status.Phase {
		switch vm.Status.Phase {
		case v1.Succeeded:
			d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Stopped.String(), "The VM was shut down.")
		case v1.Failed:
			d.recorder.Event(vm, k8sv1.EventTypeWarning, v1.Stopped.String(), "The VM crashed.")
		}
	}

	return nil
}

func (c *VirtualMachineController) Run(threadiness int, stopCh chan struct{}) {
	defer c.Queue.ShutDown()
	log.Log.Info("Starting virt-handler controller.")

	// Wait for the domain cache to be synced
	go c.domainInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced)

	// Poplulate the VM store with known Domains on the host, to get deletes since the last run
	for _, domain := range c.domainInformer.GetStore().List() {
		d := domain.(*api.Domain)
		c.vmInformer.GetStore().Add(v1.NewVMReferenceFromNameWithNS(d.ObjectMeta.Namespace, d.ObjectMeta.Name))
	}

	// Clean up left over config disks
	err := c.configDisk.UndefineUnseen(c.vmInformer.GetStore())
	if err != nil {
		panic(err)
	}

	// Clean up left over registry disks
	err = registrydisk.CleanupOrphanedEphemeralDisks(c.vmInformer.GetStore())
	if err != nil {
		panic(err)
	}

	go c.vmInformer.Run(stopCh)
	go c.watchdogInformer.Run(stopCh)
	go c.gracefulShutdownInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced, c.vmInformer.HasSynced, c.watchdogInformer.HasSynced, c.gracefulShutdownInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping virt-handler controller.")
}

func (c *VirtualMachineController) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachineController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachine %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachine %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (d *VirtualMachineController) getVMFromCache(key string) (vm *v1.VirtualMachine, exists bool, err error) {

	// Fetch the latest Vm state from cache
	obj, exists, err := d.vmInformer.GetStore().GetByKey(key)

	if err != nil {
		return nil, false, err
	}

	// Retrieve the VM
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			// TODO log and don't retry
			return nil, false, err
		}
		vm = v1.NewVMReferenceFromNameWithNS(namespace, name)
	} else {
		vm = obj.(*v1.VirtualMachine)
	}
	return vm, exists, nil
}

func (d *VirtualMachineController) getDomainFromCache(key string) (domain *api.Domain, exists bool, err error) {

	obj, exists, err := d.domainInformer.GetStore().GetByKey(key)

	if err != nil {
		return nil, false, err
	}

	if exists {
		domain = obj.(*api.Domain)
	}
	return domain, exists, nil
}

func (d *VirtualMachineController) execute(key string) error {

	// set to true when domain needs to be shutdown and removed from libvirt.
	shouldShutdownAndDelete := false
	// optimization. set to true when processing already deleted domain.
	shouldCleanUp := false
	// set to true when VM is active or about to become active.
	shouldUpdate := false

	vm, vmExists, err := d.getVMFromCache(key)
	if err != nil {
		return err
	}

	domain, domainExists, err := d.getDomainFromCache(key)
	if err != nil {
		return err
	}

	// Determine if VM's watchdog has expired
	watchdogExpired, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vm)
	if err != nil {
		return err
	} else if watchdogExpired && vm.IsRunning() {
		log.Log.Object(vm).V(3).Info("Shutting down due to expired watchdog.")
		shouldShutdownAndDelete = true
	}

	// Determine if gracefulShutdown has been triggered by virt-launcher
	gracefulShutdown, err := virtlauncher.VmHasGracefulShutdownTrigger(d.virtShareDir, vm)
	if err != nil {
		return err
	} else if gracefulShutdown && vm.IsRunning() {
		log.Log.Object(vm).V(3).Info("Shutting down due to graceful shutdown signal.")
		shouldShutdownAndDelete = true
	}

	// Determine removal of VM from cache should result in deletion.
	if !vmExists {
		// If we don't have the VM in the cache, it could be that it is currently migrating to us
		isDestination, err := d.isMigrationDestination(vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
		if err != nil {
			// unable to determine migration status, we'll try again later.
			return err
		} else if isDestination {
			// OK, this VM is migrating to us, don't interrupt it.
			return nil
		}

		if domainExists {
			// The VM is deleted on the cluster,
			// then continue with processing the deletion on the host.
			log.Log.Object(vm).V(3).Info("Shutting down domain for deleted VM object.")
			shouldShutdownAndDelete = true
		} else {
			// If neither the domain nor the vm object exist locally,
			// then ensure any remaining local ephemeral data is cleaned up.
			shouldCleanUp = true
		}
	}

	// Determine if VM is being deleted.
	if vmExists && vm.ObjectMeta.DeletionTimestamp != nil {
		if vm.IsRunning() || domainExists {
			log.Log.Object(vm).V(3).Info("Shutting down domain for VM with deletion timestamp.")
			shouldShutdownAndDelete = true
		} else {
			shouldCleanUp = true
		}
	}

	// Determine if domain needs to be deleted as a result of VM
	// shutting down naturally (guest internal invoked shutdown)
	if domainExists && vmExists && vm.IsFinal() {
		log.Log.Object(vm).V(3).Info("Removing domain and ephemeral data for finalized vm.")
		shouldShutdownAndDelete = true
	}

	// Determine if an active (or about to be active) VM should be updated.
	if vmExists && !vm.IsFinal() {
		// requiring the phase of the domain and VM to be in sync is an
		// optimization that prevents unnecessary re-processing VMs during the start flow.
		if vm.Status.Phase == d.calculateVmPhaseForStatusReason(domain, vm) {
			shouldUpdate = true
		}
	}

	var syncErr error

	// Process the VM update in this order.
	// * Shutdown and Deletion due to VM deletion, process stopping, graceful shutdown trigger, expired watchdog, etc...
	// * Cleanup of already shutdown and Deleted VMs
	// * Update due to spec change and initial start flow.
	if shouldShutdownAndDelete {
		log.Log.Object(vm).V(3).Info("Processing shutdown.")
		syncErr = d.processVmShutdown(vm, domain)
	} else if shouldCleanUp {
		log.Log.Object(vm).V(3).Info("Processing local ephemeral data cleanup for shutdown domain.")
		syncErr = d.processVmCleanup(vm)
	} else if shouldUpdate {
		log.Log.Object(vm).V(3).Info("Processing vm update")
		syncErr = d.processVmUpdate(vm)
	} else {
		log.Log.Object(vm).V(3).Info("No update processing required")
	}

	if syncErr != nil {
		d.recorder.Event(vm, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		log.Log.Object(vm).Reason(syncErr).Error("Synchronizing the VM failed.")
	}

	// Update the VM status, if the VM exists
	if vmExists {
		err = d.updateVMStatus(vm.DeepCopy(), domain, syncErr)
		if err != nil {
			log.Log.Object(vm).Reason(err).Error("Updating the VM status failed.")
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	log.Log.Object(vm).V(3).Info("Synchronization loop succeeded.")
	return nil
}

// Almost everything in the VM object maps exactly to its domain counterpart
// One exception is persistent volume claims. This function looks up each PV
// and inserts a corrected disk entry into the VM's device map.
func MapPersistentVolumes(vm *v1.VirtualMachine, clientset kubecli.KubevirtClient, namespace string) (*v1.VirtualMachine, error) {
	vmCopy := &v1.VirtualMachine{}
	model.Copy(vmCopy, vm)
	logger := log.Log.Object(vm)

	for idx, disk := range vmCopy.Spec.Domain.Devices.Disks {
		if disk.Type == "PersistentVolumeClaim" {
			logger.V(3).Infof("Mapping PersistentVolumeClaim: %s", disk.Source.Name)

			// Look up existing persistent volume
			pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(disk.Source.Name, metav1.GetOptions{})

			if err != nil {
				logger.Reason(err).Error("unable to look up persistent volume claim")
				return vm, fmt.Errorf("unable to look up persistent volume claim: %v", err)
			}

			if pvc.Status.Phase != k8sv1.ClaimBound {
				logger.Error("attempted use of unbound persistent volume")
				return vm, fmt.Errorf("attempted use of unbound persistent volume claim: %s", pvc.Name)
			}

			// Look up the PersistentVolume this PVC is bound to
			pv, err := clientset.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})

			if err != nil {
				logger.Reason(err).Error("unable to access persistent volume record")
				return vm, fmt.Errorf("unable to access persistent volume record: %v", err)
			}

			logger.Infof("Mapping PVC %s", pv.Name)
			newDisk, err := mapPVToDisk(&disk, pv)

			if err != nil {
				logger.Reason(err).Errorf("Mapping PVC %s failed", pv.Name)
				return vm, err
			}

			vmCopy.Spec.Domain.Devices.Disks[idx] = *newDisk
		} else if disk.Type == "network" {
			newDisk := v1.Disk{}
			model.Copy(&newDisk, disk)

			if disk.Source.Host == nil {
				logger.Error("Missing disk source host")
				return vm, fmt.Errorf("Missing disk source host")
			}

			ipAddrs, err := net.LookupIP(disk.Source.Host.Name)
			if err != nil || ipAddrs == nil || len(ipAddrs) < 1 {
				logger.Reason(err).Errorf("Unable to resolve host '%s'", disk.Source.Host.Name)
				return vm, fmt.Errorf("Unable to resolve host '%s': %s", disk.Source.Host.Name, err)
			}

			newDisk.Source.Host.Name = ipAddrs[0].String()

			vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
		}
	}

	return vmCopy, nil
}

func mapPVToDisk(disk *v1.Disk, pv *k8sv1.PersistentVolume) (*v1.Disk, error) {
	if pv.Spec.ISCSI != nil {
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
			return nil, fmt.Errorf("Unable to resolve host '%s': %s", hostPort[0], err)
		}

		newDisk.Source.Host = &v1.DiskSourceHost{}
		newDisk.Source.Host.Name = ipAddrs[0].String()
		if len(hostPort) > 1 {
			newDisk.Source.Host.Port = hostPort[1]
		}

		// This iscsi device has auth associated with it.
		if pv.Spec.ISCSI.SecretRef != nil && pv.Spec.ISCSI.SecretRef.Name != "" {
			newDisk.Auth = &v1.DiskAuth{
				Secret: &v1.DiskSecret{
					Type:  "iscsi",
					Usage: pv.Spec.ISCSI.SecretRef.Name,
				},
			}
		}
		return &newDisk, nil
	} else {
		err := fmt.Errorf("Referenced PV %s is backed by an unsupported storage type. Only iSCSI is supported.", pv.ObjectMeta.Name)
		return nil, err
	}
}

func (d *VirtualMachineController) getUnusedSerialPort(vm *v1.VirtualMachine) (uint, error) {
	// look for an unclaimed serial port.
	for i := 0; i < 100; i++ {
		claimed := false
		for _, serial := range vm.Spec.Domain.Devices.Serials {
			if serial.Target == nil || serial.Target.Port == nil {
				continue
			}

			if *serial.Target.Port == uint(i) {
				claimed = true
				break
			}
		}

		if claimed == false {
			return uint(i), nil
		}
	}

	return 0, goerror.New("No serial port found for console")
}

func (d *VirtualMachineController) cleanupConsoleSockets(vm *v1.VirtualMachine) error {
	// TODO this can go away once qemu is in the pods mount namespace.
	namespace := vm.ObjectMeta.Namespace
	name := vm.ObjectMeta.Name
	unixPath := fmt.Sprintf("%s-private/%s/%s", d.virtShareDir, namespace, name)
	return diskutils.RemoveFile(unixPath)
}

func (d *VirtualMachineController) MapConsoleAccess(vm *v1.VirtualMachine) *v1.VirtualMachine {
	for _, console := range vm.Spec.Domain.Devices.Consoles {
		if console.Type == "pty" && console.Target == nil {
			serialPort, err := d.getUnusedSerialPort(vm)
			if err != nil {
				// this is best effort. Don't block starting the
				// vm because the user is doing something advanced
				// with serial ports that prevents us from mapping
				// the console target.
				return vm
			}

			namespace := vm.ObjectMeta.Namespace
			name := vm.ObjectMeta.Name
			unixPath := fmt.Sprintf("%s-private/%s/%s/virt-serial%d", d.virtShareDir, namespace, name, serialPort)
			vm.Spec.Domain.Devices.Serials = append(vm.Spec.Domain.Devices.Serials, v1.Serial{
				Type: "unix",
				Target: &v1.SerialTarget{
					Port: &serialPort,
				},

				Source: &v1.SerialSource{
					Mode: "bind",
					Path: unixPath,
				},
			})
			err = os.MkdirAll(filepath.Dir(unixPath), 0755)
			if err != nil {
				// TODO
				return vm
			}

			err = diskutils.SetFileOwnership("qemu", filepath.Dir(unixPath))
			if err != nil {
				// TODO
				return vm
			}

		}
	}
	return vm
}

func (d *VirtualMachineController) injectDiskAuth(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Auth == nil || disk.Auth.Secret == nil || disk.Auth.Secret.Usage == "" {
			continue
		}

		usageIDSuffix := fmt.Sprintf("-%s-%s---", vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
		usageID := disk.Auth.Secret.Usage
		usageType := disk.Auth.Secret.Type
		secretID := usageID

		if strings.HasSuffix(usageID, usageIDSuffix) {
			secretID = strings.TrimSuffix(usageID, usageIDSuffix)
		} else {
			usageID = fmt.Sprintf("%s%s", usageID, usageIDSuffix)
		}

		secret, err := d.clientset.CoreV1().Secrets(vm.ObjectMeta.Namespace).Get(secretID, metav1.GetOptions{})
		if err != nil {
			log.Log.Reason(err).Error("Defining the VM secret failed unable to pull corresponding k8s secret value")
			return nil, err
		}

		secretValue, ok := secret.Data["node.session.auth.password"]
		if ok == false {
			return nil, goerror.New(fmt.Sprintf("No password value found in k8s secret %s %v", secretID, err))
		}

		userValue, ok := secret.Data["node.session.auth.username"]
		if ok == false {
			return nil, goerror.New(fmt.Sprintf("Failed to find username for disk auth %s", secretID))
		}
		vm.Spec.Domain.Devices.Disks[idx].Auth.Username = string(userValue)

		// override the usage id on the VM with the VM specific one.
		// By decoupling usage from the k8s secret name here, this allows
		// multiple VMs to reference the same k8s secret without conflicting
		// with one another.
		vm.Spec.Domain.Devices.Disks[idx].Auth.Secret.Usage = usageID

		err = d.domainManager.SyncVMSecret(vm, usageType, usageID, string(secretValue))
		if err != nil {
			return nil, err
		}
	}

	return vm, nil
}

func (d *VirtualMachineController) processVmCleanup(vm *v1.VirtualMachine) error {
	err := d.domainManager.RemoveVMSecrets(vm)
	if err != nil {
		return err
	}

	err = d.cleanupConsoleSockets(vm)
	if err != nil {
		return err
	}

	err = registrydisk.CleanupEphemeralDisks(vm)
	if err != nil {
		return err
	}

	err = watchdog.WatchdogFileRemove(d.virtShareDir, vm)
	if err != nil {
		return err
	}

	err = virtlauncher.VmGracefulShutdownTriggerClear(d.virtShareDir, vm)
	if err != nil {
		return err
	}

	return d.configDisk.Undefine(vm)
}

func (d *VirtualMachineController) processVmShutdown(vm *v1.VirtualMachine, domain *api.Domain) error {

	expired, timeLeft := d.hasGracePeriodExpired(domain)

	if expired == false {
		err := d.domainManager.SignalShutdownVM(vm)
		if err != nil {
			return err
		}
		// pending graceful shutdown.
		d.Queue.AddAfter(controller.VirtualMachineKey(vm), time.Duration(timeLeft)*time.Second)
		return nil
	}

	log.Log.Object(vm).Infof("grace period expired, killing deleted VM %s", vm.GetObjectMeta().GetName())

	err := d.domainManager.KillVM(vm)
	if err != nil {
		return err
	}

	return d.processVmCleanup(vm)

}

func (d *VirtualMachineController) processVmUpdate(vm *v1.VirtualMachine) error {

	hasWatchdog, err := watchdog.WatchdogFileExists(d.virtShareDir, vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Error accessing virt-launcher watchdog file.")
		return err
	}
	if hasWatchdog == false {
		log.Log.Object(vm).Reason(err).Error("Could not detect virt-launcher watchdog file.")
		return goerror.New(fmt.Sprintf("No watchdog file found for vm"))
	}

	isPending, err := d.configDisk.Define(vm)
	if err != nil {
		return err
	}

	if isPending {
		log.Log.Object(vm).V(3).Info("Synchronizing is in a pending state.")
		d.Queue.AddAfter(controller.VirtualMachineKey(vm), 1*time.Second)
		return nil
	}

	// Synchronize the VM state
	vm, err = MapPersistentVolumes(vm, d.clientset, vm.ObjectMeta.Namespace)
	if err != nil {
		return err
	}

	// Map Container Registry Disks to block devices Libvirt can consume
	vm, err = registrydisk.MapRegistryDisks(vm)
	if err != nil {
		return err
	}

	vm, err = d.injectDiskAuth(vm)
	if err != nil {
		return err
	}

	// Map whatever devices are being used for config-init
	vm, err = cloudinit.MapCloudInitDisks(vm)
	if err != nil {
		return err
	}

	// Map serial console access details
	vm = d.MapConsoleAccess(vm)
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
	_, err = d.domainManager.SyncVM(vm)
	return err
}

func (d *VirtualMachineController) isMigrationDestination(namespace string, vmName string) (bool, error) {

	// If we don't have the VM in the cache, it could be that it is currently migrating to us
	fetchedVM, err := d.clientset.VM(namespace).Get(vmName, metav1.GetOptions{})
	if err == nil {
		// So the VM still seems to exist

		if fetchedVM.Status.MigrationNodeName == d.host {
			return true, nil
		}
	} else if !errors.IsNotFound(err) {
		// Something went wrong, let's try again later
		return false, err
	}

	// VM object was not found.
	return false, nil
}

func (d *VirtualMachineController) checkFailure(vm *v1.VirtualMachine, syncErr error, reason string) (changed bool) {
	if syncErr != nil && !d.hasCondition(vm, v1.VirtualMachineSynchronized) {
		vm.Status.Conditions = append(vm.Status.Conditions, v1.VMCondition{
			Type:               v1.VirtualMachineSynchronized,
			Reason:             reason,
			Message:            syncErr.Error(),
			LastTransitionTime: metav1.Now(),
			Status:             k8sv1.ConditionFalse,
		})
		return true
	} else if syncErr == nil && d.hasCondition(vm, v1.VirtualMachineSynchronized) {
		d.removeCondition(vm, v1.VirtualMachineSynchronized)
		return true
	}
	return false
}

func (d *VirtualMachineController) hasCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) bool {
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (d *VirtualMachineController) removeCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) {
	conds := []v1.VMCondition{}
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vm.Status.Conditions = conds
}

func (d *VirtualMachineController) setVmPhaseForStatusReason(domain *api.Domain, vm *v1.VirtualMachine) {
	vm.Status.Phase = d.calculateVmPhaseForStatusReason(domain, vm)
}
func (d *VirtualMachineController) calculateVmPhaseForStatusReason(domain *api.Domain, vm *v1.VirtualMachine) v1.VMPhase {

	if domain == nil {
		if !vm.IsRunning() && !vm.IsFinal() {
			return v1.Scheduled
		} else if !vm.IsFinal() {
			// That is unexpected. We should not be able to delete a VM before we stop it.
			// However, if someone directly interacts with libvirt it is possible
			return v1.Failed
		}
	} else {
		switch domain.Status.Status {
		case api.Shutoff, api.Crashed:
			switch domain.Status.Reason {
			case api.ReasonCrashed, api.ReasonPanicked:
				return v1.Failed
			case api.ReasonShutdown, api.ReasonDestroyed, api.ReasonSaved, api.ReasonFromSnapshot:
				return v1.Succeeded
			}
		case api.Running, api.Paused, api.Blocked, api.PMSuspended:
			return v1.Running
		}
	}
	return vm.Status.Phase
}

func (d *VirtualMachineController) addFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) deleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) updateFunc(old, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		d.Queue.Add(key)
	}
}

func (d *VirtualMachineController) addDomainFunc(obj interface{}) {
	domain := obj.(*api.Domain)
	log.Log.Object(domain).Infof("Domain is in state %s reason %s", domain.Status.Status, domain.Status.Reason)
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) deleteDomainFunc(obj interface{}) {
	domain := obj.(*api.Domain)
	log.Log.Object(domain).Info("Domain deleted")
	key, err := controller.KeyFunc(obj)
	if err == nil {
		d.Queue.Add(key)
	}
}
func (d *VirtualMachineController) updateDomainFunc(old, new interface{}) {
	newDomain := new.(*api.Domain)
	oldDomain := old.(*api.Domain)
	if oldDomain.Status.Status != newDomain.Status.Status || oldDomain.Status.Reason != newDomain.Status.Reason {
		log.Log.Object(newDomain).Infof("Domain is in state %s reason %s", newDomain.Status.Status, newDomain.Status.Reason)
	}
	key, err := controller.KeyFunc(new)
	if err == nil {
		d.Queue.Add(key)
	}
}
