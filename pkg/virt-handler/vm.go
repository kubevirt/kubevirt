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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/util/wait"

	"encoding/json"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

func NewController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	virtShareDir string,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	gracefulShutdownInformer cache.SharedIndexInformer,
	watchdogTimeoutSeconds int,
	maxDevices int,
) *VirtualMachineController {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	c := &VirtualMachineController{
		Queue:                    queue,
		recorder:                 recorder,
		clientset:                clientset,
		host:                     host,
		virtShareDir:             virtShareDir,
		vmiInformer:              vmiInformer,
		domainInformer:           domainInformer,
		gracefulShutdownInformer: gracefulShutdownInformer,
		heartBeatInterval:        1 * time.Minute,
		watchdogTimeoutSeconds:   watchdogTimeoutSeconds,
	}

	vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		DeleteFunc: c.deleteDomainFunc,
		UpdateFunc: c.updateDomainFunc,
	})

	gracefulShutdownInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	c.launcherClients = make(map[string]cmdclient.LauncherClient)

	c.kvmController = device_manager.NewDeviceController(c.host, maxDevices)

	return c
}

type VirtualMachineController struct {
	recorder                 record.EventRecorder
	clientset                kubecli.KubevirtClient
	host                     string
	virtShareDir             string
	Queue                    workqueue.RateLimitingInterface
	vmiInformer              cache.SharedIndexInformer
	domainInformer           cache.SharedInformer
	gracefulShutdownInformer cache.SharedIndexInformer
	launcherClients          map[string]cmdclient.LauncherClient
	launcherClientLock       sync.Mutex
	heartBeatInterval        time.Duration
	watchdogTimeoutSeconds   int
	kvmController            *device_manager.DeviceController
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

func (d *VirtualMachineController) updateVMIStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {

	// Don't update the VirtualMachineInstance if it is already in a final state
	if vmi.IsFinal() {
		return nil
	}

	oldStatus := vmi.DeepCopy().Status

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = d.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	controller.NewVirtualMachineInstanceConditionManager().CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")

	if !reflect.DeepEqual(oldStatus, vmi.Status) {
		_, err = d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
		if err != nil {
			return err
		}
	}

	if oldStatus.Phase != vmi.Status.Phase {
		switch vmi.Status.Phase {
		case v1.Running:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Started.String(), "VirtualMachineInstance started.")
		case v1.Succeeded:
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Stopped.String(), "The VirtualMachineInstance was shut down.")
		case v1.Failed:
			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Stopped.String(), "The VirtualMachineInstance crashed.")
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

	go c.kvmController.Run(stopCh)

	// Poplulate the VirtualMachineInstance store with known Domains on the host, to get deletes since the last run
	for _, domain := range c.domainInformer.GetStore().List() {
		d := domain.(*api.Domain)
		c.vmiInformer.GetStore().Add(v1.NewVMIReferenceFromNameWithNS(d.ObjectMeta.Namespace, d.ObjectMeta.Name))
	}

	go c.vmiInformer.Run(stopCh)
	go c.gracefulShutdownInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced, c.vmiInformer.HasSynced, c.gracefulShutdownInformer.HasSynced)

	go c.heartBeat(c.heartBeatInterval, stopCh)

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
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (d *VirtualMachineController) getVMIFromCache(key string) (vmi *v1.VirtualMachineInstance, exists bool, err error) {

	// Fetch the latest Vm state from cache
	obj, exists, err := d.vmiInformer.GetStore().GetByKey(key)

	if err != nil {
		return nil, false, err
	}

	// Retrieve the VirtualMachineInstance
	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			// TODO log and don't retry
			return nil, false, err
		}
		vmi = v1.NewVMIReferenceFromNameWithNS(namespace, name)
	} else {
		vmi = obj.(*v1.VirtualMachineInstance)
	}
	return vmi, exists, nil
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

	// set to true when domain needs to be shutdown.
	shouldShutdown := false
	// set to true when domain needs to be removed from libvirt.
	shouldDelete := false
	// optimization. set to true when processing already deleted domain.
	shouldCleanUp := false
	// set to true when VirtualMachineInstance is active or about to become active.
	shouldUpdate := false

	vmi, vmiExists, err := d.getVMIFromCache(key)
	if err != nil {
		return err
	}

	log.Log.V(3).Infof("Processing vmi %v, existing: %v\n", vmi.Name, vmiExists)
	if vmiExists {
		log.Log.V(3).Infof("vmi is in phase: %v\n", vmi.Status.Phase)
	}

	domain, domainExists, err := d.getDomainFromCache(key)
	if err != nil {
		return err
	}

	// Ignore domains from an older VMI
	if vmiExists && domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		log.Log.Object(vmi).Info("Ignoring domain from an older VMI, will be handled by its own VMI.")
		return nil
	}

	log.Log.V(3).Infof("Domain: existing: %v\n", domainExists)
	if domainExists {
		log.Log.V(3).Infof("Domain status: %v, reason: %v\n", domain.Status.Status, domain.Status.Reason)
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	// Determine if gracefulShutdown has been triggered by virt-launcher
	gracefulShutdown, err := virtlauncher.VmHasGracefulShutdownTrigger(d.virtShareDir, vmi)
	if err != nil {
		return err
	} else if gracefulShutdown && vmi.IsRunning() {
		log.Log.Object(vmi).V(3).Info("Shutting down due to graceful shutdown signal.")
		shouldShutdown = true
	}

	// Determine removal of VirtualMachineInstance from cache should result in deletion.
	if !vmiExists {
		if domainAlive {
			// The VirtualMachineInstance is deleted on the cluster, and domain is alive,
			// then shut down the domain.
			log.Log.Object(vmi).V(3).Info("Shutting down domain for deleted VirtualMachineInstance object.")
			shouldShutdown = true
		} else if domainExists {
			// The VirtualMachineInstance is deleted on the cluster, and domain is not alive
			// then delete the domain.
			log.Log.Object(vmi).V(3).Info("Shutting down domain for deleted VirtualMachineInstance object.")
			shouldDelete = true
		} else {
			// If neither the domain nor the vmi object exist locally,
			// then ensure any remaining local ephemeral data is cleaned up.
			shouldCleanUp = true
		}
	}

	// Determine if VirtualMachineInstance is being deleted.
	if vmiExists && vmi.ObjectMeta.DeletionTimestamp != nil {
		if vmi.IsRunning() || domainAlive {
			log.Log.Object(vmi).V(3).Info("Shutting down domain for VirtualMachineInstance with deletion timestamp.")
			shouldShutdown = true
		} else if domainExists {
			log.Log.Object(vmi).V(3).Info("Deleting domain for VirtualMachineInstance with deletion timestamp.")
			shouldDelete = true
		} else {
			shouldCleanUp = true
		}
	}

	// Determine if domain needs to be deleted as a result of VirtualMachineInstance
	// shutting down naturally (guest internal invoked shutdown)
	if domainExists && vmiExists && vmi.IsFinal() {
		log.Log.Object(vmi).V(3).Info("Removing domain and ephemeral data for finalized vmi.")
		shouldDelete = true
	}

	// Determine if an active (or about to be active) VirtualMachineInstance should be updated.
	if vmiExists && !vmi.IsFinal() {
		// requiring the phase of the domain and VirtualMachineInstance to be in sync is an
		// optimization that prevents unnecessary re-processing VMIs during the start flow.
		phase, err := d.calculateVmPhaseForStatusReason(domain, vmi)
		if err != nil {
			return err
		}
		if vmi.Status.Phase == phase {
			shouldUpdate = true
		}
	}

	var syncErr error

	// Process the VirtualMachineInstance update in this order.
	// * Shutdown and Deletion due to VirtualMachineInstance deletion, process stopping, graceful shutdown trigger, etc...
	// * Cleanup of already shutdown and Deleted VMIs
	// * Update due to spec change and initial start flow.
	if shouldShutdown {
		log.Log.Object(vmi).V(3).Info("Processing shutdown.")
		syncErr = d.processVmShutdown(vmi, domain)
	} else if shouldDelete {
		log.Log.Object(vmi).V(3).Info("Processing deletion.")
		syncErr = d.processVmDelete(vmi, domain)
	} else if shouldCleanUp {
		log.Log.Object(vmi).V(3).Info("Processing local ephemeral data cleanup for shutdown domain.")
		syncErr = d.processVmCleanup(vmi)
	} else if shouldUpdate {
		log.Log.Object(vmi).V(3).Info("Processing vmi update")
		syncErr = d.processVmUpdate(vmi)
	} else {
		log.Log.Object(vmi).V(3).Info("No update processing required")
	}

	if syncErr != nil {
		d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		log.Log.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	// Update the VirtualMachineInstance status, if the VirtualMachineInstance exists
	if vmiExists {
		err = d.updateVMIStatus(vmi.DeepCopy(), domain, syncErr)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Updating the VirtualMachineInstance status failed.")
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	log.Log.Object(vmi).V(3).Info("Synchronization loop succeeded.")
	return nil
}

func (d *VirtualMachineController) injectCloudInitSecrets(vmi *v1.VirtualMachineInstance) error {
	cloudInitSpec := cloudinit.GetCloudInitNoCloudSource(vmi)
	if cloudInitSpec == nil {
		return nil
	}
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())

	err := cloudinit.ResolveSecrets(cloudInitSpec, namespace, d.clientset)
	if err != nil {
		return err
	}
	return nil
}

func (d *VirtualMachineController) processVmCleanup(vmi *v1.VirtualMachineInstance) error {
	err := watchdog.WatchdogFileRemove(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

	err = virtlauncher.VmGracefulShutdownTriggerClear(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

	d.closeLauncherClient(vmi)
	return nil
}

func (d *VirtualMachineController) closeLauncherClient(vmi *v1.VirtualMachineInstance) {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	sockFile := cmdclient.SocketFromUID(d.virtShareDir, string(vmi.GetUID()))

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

func (d *VirtualMachineController) getLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()

	sockFile := cmdclient.SocketFromUID(d.virtShareDir, string(vmi.GetUID()))

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

func (d *VirtualMachineController) processVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := d.getVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}

	// Only attempt to gracefully shutdown if the domain has the ACPI feature enabled
	if isACPIEnabled(vmi, domain) {
		expired, timeLeft := d.hasGracePeriodExpired(domain)
		if !expired {
			if domain.Status.Status != api.Shutdown {
				err = client.ShutdownVirtualMachine(vmi)
				if err != nil && !cmdclient.IsDisconnected(err) {
					// Only report err if it wasn't the result of a disconnect.
					return err
				}

				log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())
				// pending graceful shutdown.
				d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Duration(timeLeft)*time.Second)
				d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), "Signaled Graceful Shutdown")
			} else {
				log.Log.V(4).Object(vmi).Infof("%s is already shutting down.", vmi.GetObjectMeta().GetName())
			}
			return nil
		}
		log.Log.Object(vmi).Infof("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	} else {
		log.Log.Object(vmi).Infof("ACPI feature not available, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	}

	err = client.KillVirtualMachine(vmi)
	if err != nil && !cmdclient.IsDisconnected(err) {
		// Only report err if it wasn't the result of a disconnect.
		//
		// Both virt-launcher and virt-handler are trying to destroy
		// the VirtualMachineInstance at the same time. It's possible the client may get
		// disconnected during the kill request, which shouldn't be
		// considered an error.
		return err
	}

	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), "VirtualMachineInstance stopping")

	return nil
}

func (d *VirtualMachineController) processVmDelete(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := d.getVerifiedLauncherClient(vmi)

	// If the pod has been torn down, we know the VirtualMachineInstance is down.
	if err == nil {

		log.Log.Object(vmi).Infof("Signaled deletion for %s", vmi.GetObjectMeta().GetName())

		// pending deletion.
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), "Signaled Deletion")

		err = client.DeleteDomain(vmi)
		if err != nil && !cmdclient.IsDisconnected(err) {
			// Only report err if it wasn't the result of a disconnect.
			//
			// Both virt-launcher and virt-handler are trying to destroy
			// the VirtualMachineInstance at the same time. It's possible the client may get
			// disconnected during the kill request, which shouldn't be
			// considered an error.
			return err
		}
	}

	return nil

}

func (d *VirtualMachineController) getVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	client, err = d.getLauncherClient(vmi)
	if err != nil {
		return
	}

	// Verify connectivity.
	// It's possible the pod has already been torn down along with the VirtualMachineInstance.
	err = client.Ping()
	return
}

func (d *VirtualMachineController) processVmUpdate(origVMI *v1.VirtualMachineInstance) error {

	vmi := origVMI.DeepCopy()

	isExpired, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vmi)

	if err != nil {
		return err
	} else if isExpired {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with expired watchdog."))
	}

	err = d.injectCloudInitSecrets(vmi)
	if err != nil {
		return err
	}

	client, err := d.getLauncherClient(vmi)
	if err != nil {
		return err
	}
	err = client.SyncVirtualMachine(vmi)
	if err != nil {
		return err
	}
	d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), "VirtualMachineInstance defined.")

	return err
}

func (d *VirtualMachineController) setVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) error {
	phase, err := d.calculateVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}
	vmi.Status.Phase = phase
	return nil
}
func (d *VirtualMachineController) calculateVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) (v1.VirtualMachineInstancePhase, error) {

	if domain == nil {
		if vmi.IsScheduled() {
			isExpired, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vmi)

			if err != nil {
				return vmi.Status.Phase, err
			}

			if isExpired {
				// virt-launcher is gone and VirtualMachineInstance never transitioned
				// from scheduled to Running.
				return v1.Failed, nil
			}
			return v1.Scheduled, nil
		} else if !vmi.IsRunning() && !vmi.IsFinal() {
			return v1.Scheduled, nil
		} else if !vmi.IsFinal() {
			// That is unexpected. We should not be able to delete a VirtualMachineInstance before we stop it.
			// However, if someone directly interacts with libvirt it is possible
			return v1.Failed, nil
		}
	} else {

		switch domain.Status.Status {
		case api.Shutoff, api.Crashed:
			switch domain.Status.Reason {
			case api.ReasonCrashed, api.ReasonPanicked:
				return v1.Failed, nil
			case api.ReasonDestroyed:
				// When ACPI is available, the domain was tried to be shutdown,
				// and destroyed means that the domain was destroyed after the graceperiod expired.
				// Without ACPI a destroyed domain is ok.
				if isACPIEnabled(vmi, domain) {
					return v1.Failed, nil
				}
				return v1.Succeeded, nil
			case api.ReasonShutdown, api.ReasonSaved, api.ReasonFromSnapshot:
				return v1.Succeeded, nil
			}
		case api.Running, api.Paused, api.Blocked, api.PMSuspended:
			return v1.Running, nil
		}
	}
	return vmi.Status.Phase, nil
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

func (d *VirtualMachineController) heartBeat(interval time.Duration, stopCh chan struct{}) {
	for {
		wait.JitterUntil(func() {
			now, err := json.Marshal(v12.Now())
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't determine date")
				return
			}
			data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}, "annotations": {"%s": %s}}}`, v1.NodeSchedulable, v1.VirtHandlerHeartbeat, string(now)))
			_, err = d.clientset.CoreV1().Nodes().Patch(d.host, types.StrategicMergePatchType, data)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Can't patch node %s", d.host)
				return
			}
			log.DefaultLogger().V(4).Infof("Heartbeat sent")
		}, interval, 1.2, true, stopCh)
	}
}

func isACPIEnabled(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	zero := int64(0)
	return vmi.Spec.TerminationGracePeriodSeconds != &zero &&
		domain.Spec.Features != nil &&
		domain.Spec.Features.ACPI != nil
}
