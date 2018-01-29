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
	"os"
	"reflect"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/util/wait"

	"net"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server/client"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

func NewController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
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
		recorder:                 recorder,
		clientset:                clientset,
		host:                     host,
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

	c.launcherClients = make(map[string]cmdclient.LauncherClient)

	return c
}

type VirtualMachineController struct {
	recorder                 record.EventRecorder
	clientset                kubecli.KubevirtClient
	host                     string
	virtShareDir             string
	watchdogTimeoutSeconds   int
	Queue                    workqueue.RateLimitingInterface
	vmInformer               cache.SharedIndexInformer
	domainInformer           cache.SharedInformer
	watchdogInformer         cache.SharedIndexInformer
	gracefulShutdownInformer cache.SharedIndexInformer
	launcherClients          map[string]cmdclient.LauncherClient
	launcherClientLock       sync.Mutex
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
	if dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp != nil {
		startTime = dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp.UTC().Unix()
	}
	gracePeriod := dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds

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

	// Calculate the new VM state based on what libvirt reported
	d.setVmPhaseForStatusReason(domain, vm)

	d.checkFailure(vm, syncError, "Synchronizing with the Domain failed.")

	if !reflect.DeepEqual(oldStatus, vm.Status) {
		_, err = d.clientset.VM(vm.ObjectMeta.Namespace).Update(vm)
		if err != nil {
			return err
		}
	}

	if oldStatus.Phase != vm.Status.Phase {
		switch vm.Status.Phase {
		case v1.Running:
			d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Started.String(), "VM started.")
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
		log.Log.Object(vm).Info("Shutting down due to expired watchdog.")
		shouldShutdownAndDelete = true
		if domainExists {
			// Virt-launcher provids domain state. If virt-launcher
			// is down, the domain needs to be deleted from the cache.
			err = d.domainInformer.GetStore().Delete(domain)
			if err != nil {
				return err
			}
		}
		domainExists = false
		domain = nil
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

// Look up Volumes and PVs and translate them into their primitives (only supports ISCSI and PVs right now)
func MapVolumes(vm *v1.VirtualMachine, clientset kubecli.KubevirtClient, namespace string) (*v1.VirtualMachine, error) {
	vmCopy := vm.DeepCopy()
	logger := log.Log.Object(vm)

	for idx, volume := range vmCopy.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			logger.V(3).Infof("Mapping PersistentVolumeClaim: %s", volume.PersistentVolumeClaim.ClaimName)

			// Look up existing persistent volume
			pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(volume.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})

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
			if err = mapPVToDisk(&volume, pv); err != nil {
				logger.Reason(err).Errorf("Mapping PVC %s failed", pv.Name)
				return vm, err
			}
		}

		// After a PVC translation, ISCSI can be the resolved type, so "if" instead of "else if"
		if volume.ISCSI != nil {
			// FIXME ugly hack to resolve the IP from dns, since qemu is not in the right namespace
			hostPort := strings.Split(volume.ISCSI.TargetPortal, ":")
			ipAddrs, err := net.LookupIP(hostPort[0])
			if err != nil || len(ipAddrs) < 1 {
				return nil, fmt.Errorf("unable to resolve host '%s': %s", hostPort[0], err)
			}
			volume.ISCSI.TargetPortal = ipAddrs[0].String()
			for _, part := range hostPort[1:] {
				volume.ISCSI.TargetPortal = volume.ISCSI.TargetPortal + ":" + part
			}
		}

		// Set the translated volume, necessary since the VolumeSource might have been exchanged
		vmCopy.Spec.Volumes[idx] = volume
	}

	return vmCopy, nil
}

func mapPVToDisk(volume *v1.Volume, pv *k8sv1.PersistentVolume) error {
	if pv.Spec.ISCSI != nil {
		// Take the ISCSI config from the PV and set it on the vm
		volume.ISCSI = pv.Spec.ISCSI
		volume.PersistentVolumeClaim = nil
		return nil
	}
	return fmt.Errorf("referenced PV %s is backed by an unsupported storage type", pv.ObjectMeta.Name)
}

func (d *VirtualMachineController) injectCloudInitSecrets(vm *v1.VirtualMachine) error {
	cloudInitSpec := cloudinit.GetCloudInitNoCloudSource(vm)
	if cloudInitSpec == nil {
		return nil
	}
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())

	err := cloudinit.ResolveSecrets(cloudInitSpec, namespace, d.clientset)
	if err != nil {
		return err
	}
	return nil
}

// injectDiskAuth takes a virtual machine, extracts secrets, synchronizes them with libvirt and returns a map
// of all extrated secrets, for later use when converting from v1.VirtualMachine to api.Domain
func (d *VirtualMachineController) injectDiskAuth(vm *v1.VirtualMachine) (map[string]*k8sv1.Secret, error) {
	secrets := map[string]*k8sv1.Secret{}
	for _, volume := range vm.Spec.Volumes {
		if volume.ISCSI != nil {
			if volume.ISCSI.SecretRef == nil || volume.ISCSI.SecretRef.Name == "" {
				continue
			}

			usageType := "iscsi"
			secretID := volume.ISCSI.SecretRef.Name
			usageID := api.SecretToLibvirtSecret(vm, volume.ISCSI.SecretRef.Name)

			secret, err := d.clientset.CoreV1().Secrets(vm.Namespace).Get(secretID, metav1.GetOptions{})
			if err != nil {
				log.Log.Reason(err).Error("Defining the VM secret failed unable to pull corresponding k8s secret value")
				return nil, err
			}

			secretValue, ok := secret.Data["node.session.auth.password"]
			if ok == false {
				return nil, fmt.Errorf("No password value found in k8s secret %s %v", secretID, err)
			}

			client, err := d.getLauncherClient(vm)
			if err != nil {
				return nil, err
			}

			err = client.SyncSecret(vm, usageType, usageID, string(secretValue))
			if err != nil {
				return nil, err
			}
			secrets[secretID] = secret
		}
	}

	return secrets, nil
}

func (d *VirtualMachineController) processVmCleanup(vm *v1.VirtualMachine) error {
	err := watchdog.WatchdogFileRemove(d.virtShareDir, vm)
	if err != nil {
		return err
	}

	err = virtlauncher.VmGracefulShutdownTriggerClear(d.virtShareDir, vm)
	if err != nil {
		return err
	}

	d.closeLauncherClient(vm)
	return nil
}

func (d *VirtualMachineController) closeLauncherClient(vm *v1.VirtualMachine) {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	namespace := vm.ObjectMeta.Namespace
	name := vm.ObjectMeta.Name
	sockFile := cmdclient.SocketFromNamespaceName(d.virtShareDir, namespace, name)

	client, ok := d.launcherClients[sockFile]
	if ok == false {
		return
	}

	client.Close()
	delete(d.launcherClients, sockFile)

	os.RemoveAll(sockFile)
}

// used by unit tests to add mock clients
func (d *VirtualMachineController) addLauncherClient(client cmdclient.LauncherClient, sockFile string) error {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	d.launcherClients[sockFile] = client

	return nil
}

func (d *VirtualMachineController) getLauncherClient(vm *v1.VirtualMachine) (cmdclient.LauncherClient, error) {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	namespace := vm.ObjectMeta.Namespace
	name := vm.ObjectMeta.Name
	sockFile := cmdclient.SocketFromNamespaceName(d.virtShareDir, namespace, name)

	client, ok := d.launcherClients[sockFile]
	if ok {
		return client, nil
	}

	client, err := cmdclient.GetClient(sockFile)
	if err != nil {
		return nil, err
	}

	d.launcherClients[sockFile] = client

	return client, nil
}

func (d *VirtualMachineController) processVmShutdown(vm *v1.VirtualMachine, domain *api.Domain) error {

	clientDisconnected := false

	client, err := d.getLauncherClient(vm)
	if err != nil {
		clientDisconnected = true
	}

	// verify connectivity before processing shutdown.
	// It's possible the pod has already been torn down along with the VM.
	if clientDisconnected == false {
		err := client.Ping()
		if cmdclient.IsDisconnected(err) {
			clientDisconnected = true
		} else if err != nil {
			return err
		}
	}

	// Only attempt to gracefully terminate if we still have a
	// connection established with the pod.
	// If the pod has been torn down, we know the VM has been destroyed.
	if clientDisconnected == false {
		expired, timeLeft := d.hasGracePeriodExpired(domain)
		if expired == false {
			err = client.ShutdownVirtualMachine(vm)
			if err != nil && !cmdclient.IsDisconnected(err) {
				// Only report err if it wasn't the result of a disconnect.
				return err
			}

			log.Log.Object(vm).Infof("Signaled graceful shutdown for %s", vm.GetObjectMeta().GetName())
			// pending graceful shutdown.
			d.Queue.AddAfter(controller.VirtualMachineKey(vm), time.Duration(timeLeft)*time.Second)
			d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), "Signaled Graceful Shutdown")
			return nil
		}

		log.Log.Object(vm).Infof("grace period expired, killing deleted VM %s", vm.GetObjectMeta().GetName())

		err = client.KillVirtualMachine(vm)
		if err != nil && !cmdclient.IsDisconnected(err) {
			// Only report err if it wasn't the result of a disconnect.
			//
			// Both virt-launcher and virt-handler are trying to destroy
			// the VM at the same time. It's possible the client may get
			// disconnected during the kill request, which shouldn't be
			// considered an error.
			return err
		}
	}
	d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Deleted.String(), "VM stopping")

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

	// Synchronize the VM state
	vm, err = MapVolumes(vm, d.clientset, vm.ObjectMeta.Namespace)
	if err != nil {
		return err
	}

	err = d.injectCloudInitSecrets(vm)
	if err != nil {
		return err
	}

	secrets, err := d.injectDiskAuth(vm)
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
	// if not, delete the Domain firs
	client, err := d.getLauncherClient(vm)
	if err != nil {
		return err
	}
	err = client.StartVirtualMachine(vm, secrets)
	if err != nil {
		return err
	}
	d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Created.String(), "VM defined.")

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
		vm.Status.Conditions = append(vm.Status.Conditions, v1.VirtualMachineCondition{
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
	conds := []v1.VirtualMachineCondition{}
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
	domain, ok := obj.(*api.Domain)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		domain, ok = tombstone.Obj.(*api.Domain)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a domain %#v", obj)).Error("Failed to process delete notification")
			return
		}
		return
	}
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
