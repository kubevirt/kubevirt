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
	"encoding/json"
	goerror "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/controller"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	pvcutils "kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

func NewController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	ipAddress string,
	virtShareDir string,
	vmiSourceInformer cache.SharedIndexInformer,
	vmiTargetInformer cache.SharedIndexInformer,
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
		ipAddress:                ipAddress,
		virtShareDir:             virtShareDir,
		vmiSourceInformer:        vmiSourceInformer,
		vmiTargetInformer:        vmiTargetInformer,
		domainInformer:           domainInformer,
		gracefulShutdownInformer: gracefulShutdownInformer,
		heartBeatInterval:        1 * time.Minute,
		watchdogTimeoutSeconds:   watchdogTimeoutSeconds,
		migrationProxy:           migrationproxy.NewMigrationProxyManager(virtShareDir),
		podIsolationDetector:     isolation.NewSocketBasedIsolationDetector(virtShareDir),
	}

	vmiSourceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})

	vmiTargetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	ipAddress                string
	virtShareDir             string
	Queue                    workqueue.RateLimitingInterface
	vmiSourceInformer        cache.SharedIndexInformer
	vmiTargetInformer        cache.SharedIndexInformer
	domainInformer           cache.SharedInformer
	gracefulShutdownInformer cache.SharedIndexInformer
	launcherClients          map[string]cmdclient.LauncherClient
	launcherClientLock       sync.Mutex
	heartBeatInterval        time.Duration
	watchdogTimeoutSeconds   int
	kvmController            *device_manager.DeviceController
	migrationProxy           migrationproxy.ProxyManager
	podIsolationDetector     isolation.PodIsolationDetector
}

// Determines if a domain's grace period has expired during shutdown.
// If the grace period has started but not expired, timeLeft represents
// the time in seconds left until the period expires.
// If the grace period has not started, timeLeft will be set to -1.
func (d *VirtualMachineController) hasGracePeriodExpired(dom *api.Domain) (hasExpired bool, timeLeft int64) {

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

	timeLeft = int64(gracePeriod - diff)
	if timeLeft < 1 {
		timeLeft = 1
	}
	return
}

func (d *VirtualMachineController) hasTargetDetectedDomain(vmi *v1.VirtualMachineInstance) (bool, int64) {
	// give the target node 60 seconds to discover the libvirt domain via the domain informer
	// before allowing the VMI to be processed. This closes the gap between the
	// VMI's status getting updated to reflect the new source node, and the domain
	// informer firing the event to alert the source node of the new domain.
	migrationTargetDelayTimeout := 60

	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNodeDomainDetected {

		return true, 0
	}

	nowUnix := time.Now().UTC().Unix()
	migrationEndUnix := vmi.Status.MigrationState.EndTimestamp.Time.UTC().Unix()

	diff := nowUnix - migrationEndUnix

	if diff > int64(migrationTargetDelayTimeout) {
		return false, 0
	}

	timeLeft := int64(migrationTargetDelayTimeout) - diff

	enqueueTime := timeLeft
	if enqueueTime < 5 {
		enqueueTime = 5
	}

	// re-enqueue the key to ensure it gets processed again within the right time.
	d.Queue.AddAfter(controller.VirtualMachineKey(vmi), time.Duration(enqueueTime)*time.Second)

	return false, timeLeft
}

func domainMigrated(domain *api.Domain) bool {
	if domain != nil && domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonMigrated {
		return true
	}
	return false
}

func (d *VirtualMachineController) updateVMIStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()

	// Don't update the VirtualMachineInstance if it is already in a final state
	if vmi.IsFinal() {
		return nil
	}

	oldStatus := vmi.DeepCopy().Status

	if domain != nil {
		// This is needed to be backwards compatible with vmi's which have status interfaces
		// with the name not being set
		if len(domain.Spec.Devices.Interfaces) == 0 && len(vmi.Status.Interfaces) == 1 && vmi.Status.Interfaces[0].Name == "" {
			for _, network := range vmi.Spec.Networks {
				if network.NetworkSource.Pod != nil {
					vmi.Status.Interfaces[0].Name = network.Name
				}
			}
		}

		if len(domain.Spec.Devices.Interfaces) > 0 || len(domain.Status.Interfaces) > 0 {
			// This calculates the vmi.Status.Interfaces based on the following data sets:
			// - vmi.Status.Interfaces - previously calculated interfaces, this can contains data
			//   set in the controller (pod IP) which can not be deleted, unless overridden by Qemu agent
			// - domain.Spec - interfaces form the Spec
			// - domain.Status.Interfaces - interfaces reported by guest agent (emtpy if Qemu agent not running)
			newInterfaces := []v1.VirtualMachineInstanceNetworkInterface{}

			existingInterfaceStatusByName := map[string]v1.VirtualMachineInstanceNetworkInterface{}
			for _, existingInterfaceStatus := range vmi.Status.Interfaces {
				if existingInterfaceStatus.Name != "" {
					existingInterfaceStatusByName[existingInterfaceStatus.Name] = existingInterfaceStatus
				}
			}

			domainInterfaceStatusByMac := map[string]api.InterfaceStatus{}
			for _, domainInterfaceStatus := range domain.Status.Interfaces {
				domainInterfaceStatusByMac[domainInterfaceStatus.Mac] = domainInterfaceStatus
			}

			// Iterate through all domain.Spec interfaces
			for _, domainInterface := range domain.Spec.Devices.Interfaces {
				interfaceMAC := domainInterface.MAC.MAC
				var newInterface v1.VirtualMachineInstanceNetworkInterface

				if existingInterface, exists := existingInterfaceStatusByName[domainInterface.Alias.Name]; exists {
					// Reuse previously calculated interface from vmi.Status.Interfaces, updating the MAC from domain.Spec
					// Only interfaces defined in domain.Spec are handled here
					newInterface = existingInterface
					newInterface.MAC = interfaceMAC
				} else {
					// If not present in vmi.Status.Interfaces, create a new one based on domain.Spec
					newInterface = v1.VirtualMachineInstanceNetworkInterface{
						MAC:  interfaceMAC,
						Name: domainInterface.Alias.Name,
					}
				}

				// Update IP info based on information from domain.Status.Interfaces (Qemu guest)
				// Remove the interface from domainInterfaceStatusByMac to mark it as handled
				if interfaceStatus, exists := domainInterfaceStatusByMac[interfaceMAC]; exists {
					newInterface.IP = interfaceStatus.Ip
					newInterface.IPs = interfaceStatus.IPs
					newInterface.InterfaceName = interfaceStatus.InterfaceName
					delete(domainInterfaceStatusByMac, interfaceMAC)
				}
				newInterfaces = append(newInterfaces, newInterface)
			}

			// If any of domain.Status.Interfaces were not handled above, it means that the vm contains additional
			// interfaces not defined in domain.Spec (most likely added by user on VM). Add them to vmi.Status.Interfaces
			for interfaceMAC, domainInterfaceStatus := range domainInterfaceStatusByMac {
				newInterface := v1.VirtualMachineInstanceNetworkInterface{
					Name:          domainInterfaceStatus.Name,
					MAC:           interfaceMAC,
					IP:            domainInterfaceStatus.Ip,
					IPs:           domainInterfaceStatus.IPs,
					InterfaceName: domainInterfaceStatus.InterfaceName,
				}
				newInterfaces = append(newInterfaces, newInterface)
			}
			vmi.Status.Interfaces = newInterfaces
		}
	}

	// Only update the VMI's phase if this node owns the VMI.
	if vmi.Status.NodeName != "" && vmi.Status.NodeName != d.host {
		// not owned by this host, likely the result of a migration
		return nil
	}

	// Update migration progress if domain reports anything in the migration metadata.
	if domain != nil && domain.Spec.Metadata.KubeVirt.Migration != nil && vmi.Status.MigrationState != nil && d.isMigrationSource(vmi) {
		migrationMetadata := domain.Spec.Metadata.KubeVirt.Migration
		if migrationMetadata.UID == vmi.Status.MigrationState.MigrationUID {

			if vmi.Status.MigrationState.EndTimestamp == nil && migrationMetadata.EndTimestamp != nil {
				if migrationMetadata.Failed {
					d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("VirtualMachineInstance migration uid %s failed. reason:%s", string(migrationMetadata.UID), migrationMetadata.FailureReason))
				}
			}

			if vmi.Status.MigrationState.StartTimestamp == nil {
				vmi.Status.MigrationState.StartTimestamp = migrationMetadata.StartTimestamp
			}
			if vmi.Status.MigrationState.EndTimestamp == nil {
				vmi.Status.MigrationState.EndTimestamp = migrationMetadata.EndTimestamp
			}
			vmi.Status.MigrationState.Completed = migrationMetadata.Completed
			vmi.Status.MigrationState.Failed = migrationMetadata.Failed
		}
	}

	// handle migrations differently than normal status updates.
	//
	// When a successful migration is detected, we must transfer ownership of the VMI
	// from the source node (this node) to the target node (node the domain was migrated to).
	//
	// Transfer owership by...
	// 1. Marking vmi.Status.MigationState as completed
	// 2. Update the vmi.Status.NodeName to reflect the target node's name
	// 3. Update the VMI's NodeNameLabel annotation to reflect the target node's name
	//
	// After a migration, the VMI's phase is no longer owned by this node. Only the
	// MigrationState status field is elgible to be mutated.
	if domainMigrated(domain) {
		migrationHost := ""
		if vmi.Status.MigrationState != nil {
			migrationHost = vmi.Status.MigrationState.TargetNode
		}

		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.EndTimestamp == nil {
			now := v12.NewTime(time.Now())
			vmi.Status.MigrationState.EndTimestamp = &now
		}

		targetNodeDetectedDomain, timeLeft := d.hasTargetDetectedDomain(vmi)

		// If we can't detect where the migration went to, then we have no
		// way of transfering ownership. The only option here is to move the
		// vmi to failed.  The cluster vmi controller will then tear down the
		// resulting pods.
		if migrationHost == "" {
			// migrated to unknown host.
			vmi.Status.Phase = v1.Failed
			vmi.Status.MigrationState.Completed = true
			vmi.Status.MigrationState.Failed = true

			d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to unknown host."))
		} else if !targetNodeDetectedDomain {
			if timeLeft <= 0 {
				vmi.Status.Phase = v1.Failed
				vmi.Status.MigrationState.Completed = true
				vmi.Status.MigrationState.Failed = true

				d.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance's domain was never observed on the target after the migration completed within the timeout period."))
			} else {
				log.Log.Object(vmi).Info("Waiting on the target node to observe the migrated domain before performing the handoff")
			}
		} else if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.TargetNodeDomainDetected {
			// this is the migration ACK.
			// At this point we know that the migration has completed and that
			// the target node has seen the domain event.
			vmi.Labels[v1.NodeNameLabel] = migrationHost
			vmi.Status.NodeName = migrationHost
			vmi.Status.MigrationState.Completed = true
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", migrationHost))
		}

		if !reflect.DeepEqual(oldStatus, vmi.Status) {
			_, err = d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = d.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	// Cacluate whether the VM is migratable
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceIsMigratable) {
		isBlockMigration, err := d.checkVolumesForMigration(vmi)
		liveMigrationCondition := v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceIsMigratable,
			Status: k8sv1.ConditionTrue,
		}
		if err != nil {
			liveMigrationCondition.Status = k8sv1.ConditionFalse
			liveMigrationCondition.Message = err.Error()
			liveMigrationCondition.Reason = v1.VirtualMachineInstanceReasonDisksNotMigratable
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, liveMigrationCondition)

		// Set VMI Migration Method
		if isBlockMigration {
			vmi.Status.MigrationMethod = v1.BlockMigration
		} else {
			vmi.Status.MigrationMethod = v1.LiveMigration
		}
	}

	// Update the condition when GA is connected
	channelConnected := false
	if domain != nil {
		for _, channel := range domain.Spec.Devices.Channels {
			if channel.Target != nil {
				log.Log.V(4).Infof("Channel: %s, %s", channel.Target.Name, channel.Target.State)
				if channel.Target.Name == "org.qemu.guest_agent.0" {
					if channel.Target.State == "connected" {
						channelConnected = true
					}
				}

			}
		}
	}

	switch {
	case channelConnected && !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected):
		agentCondition := v1.VirtualMachineInstanceCondition{
			Type:          v1.VirtualMachineInstanceAgentConnected,
			LastProbeTime: v12.Now(),
			Status:        k8sv1.ConditionTrue,
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
	case !channelConnected:
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceAgentConnected)
	}

	condManager.CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")

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
		c.vmiSourceInformer.GetStore().Add(
			v1.NewVMIReferenceWithUUID(
				d.ObjectMeta.Namespace,
				d.ObjectMeta.Name,
				d.Spec.Metadata.KubeVirt.UID,
			),
		)
	}

	go c.vmiSourceInformer.Run(stopCh)
	go c.vmiTargetInformer.Run(stopCh)
	go c.gracefulShutdownInformer.Run(stopCh)
	cache.WaitForCacheSync(stopCh, c.domainInformer.HasSynced, c.vmiSourceInformer.HasSynced, c.vmiTargetInformer.HasSynced, c.gracefulShutdownInformer.HasSynced)

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
	obj, exists, err := d.vmiSourceInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		obj, exists, err = d.vmiTargetInformer.GetStore().GetByKey(key)
		if err != nil {
			return nil, false, err
		}
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

func (d *VirtualMachineController) migrationOrphanedSourceNodeExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	if domainExists {
		err := d.processVmDelete(vmi, domain)
		if err != nil {
			return err
		}
		// we can perform the cleanup immediately after
		// the successful delete here because we don't have
		// to report the deletion results on the VMI status
		// in this case.
		err = d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else {
		err := d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *VirtualMachineController) migrationTargetExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	// set to true when preparation of migration target should be aborted.
	shouldAbort := false
	// set to true when VirtualMachineInstance migration target needs to be prepared
	shouldUpdate := false

	if vmiExists && vmi.IsRunning() {
		shouldUpdate = true
	}

	if !vmiExists && vmi.DeletionTimestamp != nil {
		shouldAbort = true
	} else if vmi.IsFinal() {
		shouldAbort = true
	}

	if shouldAbort {
		if domainExists {
			err := d.processVmDelete(vmi, domain)
			if err != nil {
				return err
			}
		}

		err := d.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	} else if shouldUpdate {
		log.Log.Object(vmi).V(3).Info("Processing vmi migration target update")
		vmiCopy := vmi.DeepCopy()

		// if the vmi previous lived on this node, we need to make sure
		// we aren't holding on to a previous client connection that is dead.
		// THis function reaps the client connection if it is dead.
		//
		// A new client connection will be created on demand when needed
		d.removeStaleClientConnections(vmi)

		// prepare the POD for the migration
		err := d.processVmUpdate(vmi)
		if err != nil {
			return err
		}

		if domainExists && vmi.Status.MigrationState != nil {
			// record that we've see the domain populated on the target's node
			log.Log.Object(vmi).Info("The target node received the migrated domain")
			vmiCopy.Status.MigrationState.TargetNodeDomainDetected = true
		}

		// get the migration listener port
		curPort := d.migrationProxy.GetTargetListenerPort(string(vmi.UID))
		if curPort == 0 {
			return fmt.Errorf("target migration listener is not up")
		}

		hostAddress := ""

		// advertise the listener address to the source node
		if vmi.Status.MigrationState != nil {
			hostAddress = vmi.Status.MigrationState.TargetNodeAddress
		}
		curAddress := fmt.Sprintf("%s:%d", d.ipAddress, curPort)
		if hostAddress != curAddress {
			d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), fmt.Sprintf("Migration Target is listening at %s", curAddress))
			vmiCopy.Status.MigrationState.TargetNodeAddress = curAddress
		}

		// update the VMI if necessary
		if !reflect.DeepEqual(vmi.Status, vmiCopy.Status) {
			vmiCopy.Status.MigrationState.TargetNodeAddress = curAddress
			_, err := d.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmiCopy)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}

func (d *VirtualMachineController) defaultExecute(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	// set to true when domain needs to be shutdown.
	shouldShutdown := false
	// set to true when domain needs to be removed from libvirt.
	shouldDelete := false
	// optimization. set to true when processing already deleted domain.
	shouldCleanUp := false
	// set to true when VirtualMachineInstance is active or about to become active.
	shouldUpdate := false
	// set true to ensure that no updates to the current VirtualMachineInstance state will occur
	forceIgnoreSync := false

	log.Log.V(3).Infof("Processing vmi %v, existing: %v\n", vmi.Name, vmiExists)
	if vmiExists {
		log.Log.V(3).Infof("vmi is in phase: %v\n", vmi.Status.Phase)
	}

	log.Log.V(3).Infof("Domain: existing: %v\n", domainExists)
	if domainExists {
		log.Log.V(3).Infof("Domain status: %v, reason: %v\n", domain.Status.Status, domain.Status.Reason)
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	domainMigrated := domainExists && domainMigrated(domain)

	// Determine if gracefulShutdown has been triggered by virt-launcher
	gracefulShutdown, err := virtlauncher.VmHasGracefulShutdownTrigger(d.virtShareDir, vmi)
	if err != nil {
		return err
	} else if gracefulShutdown && vmi.IsRunning() {
		if domainAlive {
			log.Log.Object(vmi).V(3).Info("Shutting down due to graceful shutdown signal.")
			shouldShutdown = true
		} else {
			shouldDelete = true
		}
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
		if domainAlive {
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

	// NOTE: This must be the last check that occurs before checking the sync booleans!
	//
	// Special logic for domains migrated from a source node.
	// Don't delete/destroy domain until the handoff occurs.
	if domainMigrated {
		// only allow the sync to occur on the domain once we've done
		// the node handoff. Otherwise we potentially lose the fact that
		// the domain migrated because we'll attempt to delete the locally
		// shut off domain during the sync.
		if vmiExists &&
			!vmi.IsFinal() &&
			vmi.DeletionTimestamp == nil &&
			vmi.Status.NodeName != "" &&
			vmi.Status.NodeName == d.host {

			// If the domain migrated but the VMI still thinks this node
			// is the host, force ignore the sync until the VMI's status
			// is updated to reflect the node the domain migrated to.
			forceIgnoreSync = true
		}
	}

	var syncErr error

	// Process the VirtualMachineInstance update in this order.
	// * Shutdown and Deletion due to VirtualMachineInstance deletion, process stopping, graceful shutdown trigger, etc...
	// * Cleanup of already shutdown and Deleted VMIs
	// * Update due to spec change and initial start flow.
	if forceIgnoreSync {
		log.Log.Object(vmi).V(3).Info("No update processing required: forced ignore")
	} else if shouldShutdown {
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

func (d *VirtualMachineController) execute(key string) error {
	vmi, vmiExists, err := d.getVMIFromCache(key)
	if err != nil {
		return err
	}

	domain, domainExists, err := d.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists && domainExists {
		vmi.UID = domain.Spec.Metadata.KubeVirt.UID
	}

	// As a last effort, if the UID still can't be determined attempt
	// to retrieve it from the watchdog file
	if string(vmi.UID) == "" {
		uid := watchdog.WatchdogFileGetUid(d.virtShareDir, vmi)
		if uid != "" {
			log.Log.Object(vmi).V(3).Infof("Watchdog file provided %s as UID", uid)
			vmi.UID = types.UID(uid)
		}
	}
	if vmiExists && domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		oldVMI := v1.NewVMIReferenceFromNameWithNS(vmi.Namespace, vmi.Name)
		oldVMI.UID = domain.Spec.Metadata.KubeVirt.UID
		expired, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, oldVMI)
		if err != nil {
			return err
		}
		// If we found an outdated domain which is also not alive anymore, clean up
		if expired {
			return d.processVmCleanup(oldVMI)
		}
		// if the watchdog still gets updated, we are not allowed to clean up
		return nil
	}

	// Take different execution paths depending on the state of the migration and the
	// node this is executed on.

	if vmiExists && d.isPreMigrationTarget(vmi) {
		// 1. PRE-MIGRATION TARGET PREPARATION PATH
		//
		// If this node is the target of the vmi's migration, take
		// a different execute path. The target execute path prepares
		// the local environment for the migration, but does not
		// start the VMI
		return d.migrationTargetExecute(key,
			vmi,
			vmiExists,
			domain,
			domainExists)
	} else if vmiExists && d.isOrphanedMigrationSource(vmi) {
		// 3. POST-MIGRATION SOURCE CLEANUP
		//
		// After a migration, the migrated domain still exists in the old
		// source's domain cache. Ensure that any node that isn't currently
		// the target or owner of the VMI handles deleting the domain locally.
		return d.migrationOrphanedSourceNodeExecute(key,
			vmi,
			vmiExists,
			domain,
			domainExists)
	}
	return d.defaultExecute(key,
		vmi,
		vmiExists,
		domain,
		domainExists)

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
	err := virtlauncher.VmGracefulShutdownTriggerClear(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

	d.closeLauncherClient(vmi)

	d.migrationProxy.StopTargetListener(string(vmi.UID))
	d.migrationProxy.StopSourceListener(string(vmi.UID))

	// Watch dog file must be the last thing removed here
	err = watchdog.WatchdogFileRemove(d.virtShareDir, vmi)
	if err != nil {
		return err
	}

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

				// Make sure that we don't hot-loop in case we send the first domain notification
				if timeLeft == -1 {
					timeLeft = 5
					if vmi.Spec.TerminationGracePeriodSeconds != nil && *vmi.Spec.TerminationGracePeriodSeconds < timeLeft {
						timeLeft = *vmi.Spec.TerminationGracePeriodSeconds
					}
				}
				// In case we have a long grace period, we want to resend the graceful shutdown every 5 seconds
				// That's important since a booting OS can miss ACPI signals
				if timeLeft > 5 {
					timeLeft = 5
				}

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

func (d *VirtualMachineController) removeStaleClientConnections(vmi *v1.VirtualMachineInstance) {

	_, err := d.getVerifiedLauncherClient(vmi)
	if err == nil {
		// current client connection is good.
		return
	}

	// remove old stale client connection

	// maps require locks for concurrent access
	d.launcherClientLock.Lock()
	defer d.launcherClientLock.Unlock()
	sockFile := cmdclient.SocketFromUID(d.virtShareDir, string(vmi.GetUID()))

	client, ok := d.launcherClients[sockFile]
	if !ok {
		// no client connection to reap
		return
	}

	// close the connection but do not delete the file
	client.Close()
	delete(d.launcherClients, sockFile)
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

func (d *VirtualMachineController) isOrphanedMigrationSource(vmi *v1.VirtualMachineInstance) bool {
	nodeName, ok := vmi.Labels[v1.NodeNameLabel]

	if ok && nodeName != "" && nodeName != d.host {
		return true
	}

	return false
}

func (d *VirtualMachineController) isPreMigrationTarget(vmi *v1.VirtualMachineInstance) bool {

	migrationTargetNodeName, ok := vmi.Labels[v1.MigrationTargetNodeNameLabel]

	if ok &&
		migrationTargetNodeName != "" &&
		migrationTargetNodeName != vmi.Status.NodeName &&
		migrationTargetNodeName == d.host {
		return true
	}

	return false
}

func (d *VirtualMachineController) checkVolumesForMigration(vmi *v1.VirtualMachineInstance) (blockMigrate bool, err error) {
	// Check if all VMI volumes can be shared between the source and the destination
	// of a live migration. blockMigrate will be returned as false, only if all volumes
	// are shared and the VMI has no local disks
	// Some combinations of disks makes the VMI no suitable for live migration.
	// A relevant error will be returned in this case.
	sharedVol := false
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil {
			sharedVol = true
			_, shared, err := pvcutils.IsSharedPVCFromClient(d.clientset, vmi.Namespace, volSrc.PersistentVolumeClaim.ClaimName)
			if errors.IsNotFound(err) {
				return blockMigrate, fmt.Errorf("persistentvolumeclaim %v not found", volSrc.PersistentVolumeClaim.ClaimName)
			} else if err != nil {
				return blockMigrate, err
			}
			blockMigrate = blockMigrate || !shared
			if !shared {
				return blockMigrate, fmt.Errorf("cannot migrate VMI with non-shared PVCs")
			}
		} else if volSrc.HostDisk != nil {
			shared := false
			if volSrc.HostDisk.Shared != nil {
				shared = *volSrc.HostDisk.Shared
			}
			blockMigrate = blockMigrate || !shared
			if !shared {
				return blockMigrate, fmt.Errorf("cannot migrate VMI with non-shared HostDisk")
			}
			sharedVol = true
		} else if volSrc.CloudInitNoCloud != nil ||
			volSrc.ConfigMap != nil || volSrc.ServiceAccount != nil ||
			volSrc.Secret != nil {
			continue
		} else {
			blockMigrate = true
		}
	}
	if sharedVol && blockMigrate {
		err = fmt.Errorf("cannot migrate VMI with mixed shared and non-shared volumes")
		return
	}
	return
}

func (d *VirtualMachineController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {

	if vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceNode == d.host &&
		vmi.Status.MigrationState.TargetNodeAddress != "" &&
		!vmi.Status.MigrationState.Completed {

		return true
	}
	return false

}

func (d *VirtualMachineController) handleMigrationProxy(vmi *v1.VirtualMachineInstance) error {

	// handle starting/stopping target migration proxy
	if d.isPreMigrationTarget(vmi) {

		res, err := d.podIsolationDetector.Detect(vmi)
		if err != nil {
			return err
		}

		// Get Socket File.
		socketFile := fmt.Sprintf("/proc/%d/root/var/run/libvirt/libvirt-sock", res.Pid())

		err = d.migrationProxy.StartTargetListener(string(vmi.UID), socketFile)
		if err != nil {
			return err
		}
	} else {
		d.migrationProxy.StopTargetListener(string(vmi.UID))
	}

	// handle starting/stopping source migration proxy.
	// start the source proxy once we know the target address
	if d.isMigrationSource(vmi) {
		err := d.migrationProxy.StartSourceListener(string(vmi.UID), vmi.Status.MigrationState.TargetNodeAddress)
		if err != nil {
			return err
		}

	} else {
		d.migrationProxy.StopSourceListener(string(vmi.UID))
	}

	return nil
}

func (d *VirtualMachineController) processVmUpdate(origVMI *v1.VirtualMachineInstance) error {
	vmi := origVMI.DeepCopy()

	isExpired, err := watchdog.WatchdogFileIsExpired(d.watchdogTimeoutSeconds, d.virtShareDir, vmi)

	if err != nil {
		return err
	} else if isExpired {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with expired watchdog."))
	}

	err = hostdisk.ReplacePVCByHostDisk(vmi, d.clientset)
	if err != nil {
		return err
	}

	err = d.injectCloudInitSecrets(vmi)
	if err != nil {
		return err
	}

	client, err := d.getLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("unable to create virt-launcher client connection: %v", err)
	}

	// this adds, removes, and replaces migration proxy connections as needed
	err = d.handleMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle migration proxy: %v", err)
	}

	if d.isPreMigrationTarget(vmi) {
		err = client.SyncMigrationTarget(vmi)
		if err != nil {
			return fmt.Errorf("syncing migration target failed: %v", err)
		}
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), "VirtualMachineInstance Migration Target Prepared.")
	} else if d.isMigrationSource(vmi) {
		err = client.MigrateVirtualMachine(vmi)
		if err != nil {
			return err
		}
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrating.String(), "VirtualMachineInstance is migrating.")

	} else {
		err = client.SyncVirtualMachine(vmi)
		if err != nil {
			return err
		}
		d.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), "VirtualMachineInstance defined.")
	}

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
			case api.ReasonMigrated:
				// if the domain migrated, we no longer know the phase.
				return vmi.Status.Phase, nil
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
			// Label the node if cpu manager is running on it
			// This is a temporary workaround until k8s bug #66525 is resolved
			virtconfig.Init()
			if virtconfig.CPUManagerEnabled() {
				d.updateNodeCpuManagerLabel()
			}
		}, interval, 1.2, true, stopCh)
	}
}

func (d *VirtualMachineController) updateNodeCpuManagerLabel() {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
		return
	}

	isEnabled := false
	for _, entry := range entries {
		content, err := ioutil.ReadFile(entry)
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
			return
		}
		if strings.Contains(string(content), "kubelet") && strings.Contains(string(content), "cpu-manager-policy=static") {
			isEnabled = true
			break
		}
	}

	data := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "%t"}}}`, v1.CPUManager, isEnabled))
	_, err = d.clientset.CoreV1().Nodes().Patch(d.host, types.StrategicMergePatchType, data)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to set a cpu manager label on host %s", d.host)
		return
	}
	log.DefaultLogger().V(4).Infof("Node has CPU Manager running")

}

func isACPIEnabled(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	zero := int64(0)
	return vmi.Spec.TerminationGracePeriodSeconds != &zero &&
		domain != nil &&
		domain.Spec.Features != nil &&
		domain.Spec.Features.ACPI != nil
}
