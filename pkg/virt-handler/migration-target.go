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
 * Copyright 2025 The KubeVirt Authors.
 *
 */

package virthandler

import (
	"context"
	"encoding/json"
	goerror "errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	hotplug_volume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	launcher_clients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type netBindingPluginMemoryCalculator interface {
	Calculate(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) resource.Quantity
}

type MigrationTargetController struct {
	*BaseController
	capabilities                     *libvirtxml.Caps
	clientset                        kubecli.KubevirtClient
	containerDiskMounter             container_disk.Mounter
	hotplugVolumeMounter             hotplug_volume.VolumeMounter
	queue                            workqueue.TypedRateLimitingInterface[string]
	launcherClients                  launcher_clients.LauncherClientsManager
	migrationIpAddress               string
	migrationProxy                   migrationproxy.ProxyManager
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator
	netConf                          netconf
	podIsolationDetector             isolation.PodIsolationDetector
	recorder                         record.EventRecorder
	virtLauncherFSRunDirPattern      string
	vmiExpectations                  *controller.UIDTrackingControllerExpectations
}

func NewMigrationTargetController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	virtShareDir string,
	virtPrivateDir string,
	kubeletPodsDir string,
	migrationIpAddress string,
	launcherClients launcher_clients.LauncherClientsManager,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	capabilities *libvirtxml.Caps,
	netConf netconf,
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator,
) (*MigrationTargetController, error) {

	baseCtrl, err := NewBaseController(
		host,
		vmiInformer,
		domainInformer,
		clusterConfig,
		podIsolationDetector,
	)
	if err != nil {
		return nil, err
	}

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-target"},
	)

	containerDiskState := filepath.Join(virtPrivateDir, "container-disk-mount-state")
	if err := os.MkdirAll(containerDiskState, 0700); err != nil {
		return nil, err
	}

	hotplugState := filepath.Join(virtPrivateDir, "hotplug-volume-mount-state")
	if err := os.MkdirAll(hotplugState, 0700); err != nil {
		return nil, err
	}

	c := &MigrationTargetController{
		BaseController:                   baseCtrl,
		capabilities:                     capabilities,
		clientset:                        clientset,
		containerDiskMounter:             container_disk.NewMounter(podIsolationDetector, containerDiskState, clusterConfig),
		hotplugVolumeMounter:             hotplug_volume.NewVolumeMounter(hotplugState, kubeletPodsDir, host),
		queue:                            queue,
		launcherClients:                  launcherClients,
		migrationIpAddress:               migrationIpAddress,
		migrationProxy:                   migrationProxy,
		netBindingPluginMemoryCalculator: netBindingPluginMemoryCalculator,
		netConf:                          netConf,
		podIsolationDetector:             podIsolationDetector,
		recorder:                         recorder,
		virtLauncherFSRunDirPattern:      "/proc/%d/root/var/run",
		vmiExpectations:                  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDeleteFunc,
		UpdateFunc: c.updateFunc,
	})
	if err != nil {
		return nil, err
	}

	_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		UpdateFunc: c.updateDomainFunc,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func domainIsActiveOnTarget(domain *api.Domain) bool {
	if domain == nil {
		return false
	}

	// It's possible for the domain to be active on the target node if the domain is
	// 1. Running
	// 2. User initiated Paused
	if domain.Status.Status == api.Running {
		return true
	} else if domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedUser {
		return true
	}
	return false

}

func (c *MigrationTargetController) ackMigrationCompletion(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	vmi.Status.MigrationState.EndTimestamp = domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp
	vmi.Labels[v1.NodeNameLabel] = c.host
	delete(vmi.Labels, v1.OutdatedLauncherImageLabel)
	vmi.Status.LauncherContainerImageVersion = ""
	vmi.Status.NodeName = c.host
	// clean the evacuation node name since have already migrated to a new node
	vmi.Status.EvacuationNodeName = ""
	// update the vmi migrationTransport to indicate that next migration should use unix URI
	// new workloads will set the migrationTransport on their creation, however, legacy workloads
	// can make the switch only after the first migration
	vmi.Status.MigrationTransport = v1.MigrationTransportUnix
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", c.host))
	log.Log.Object(vmi).Info("The target node detected that the migration has completed")
}

func (c *MigrationTargetController) updateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if migrations.MigrationFailed(vmi) {
		log.Log.Object(vmi).V(4).Info("migration has failed, nothing to report on the target node")
		return nil
	}

	domainExists := domain != nil

	// detect domain on target node
	if domainExists && !vmi.Status.MigrationState.TargetNodeDomainDetected {
		// record that we've see the domain populated on the target's node
		log.Log.Object(vmi).Info("The target node received the migrated domain")
		vmi.Status.MigrationState.TargetNodeDomainDetected = true

		// adjust QEMU process memlock limits in order to enable old virt-launcher pod's to
		// perform hotplug host-devices on post migration.
		if err := isolation.AdjustQemuProcessMemoryLimits(c.podIsolationDetector, vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio); err != nil {
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "Failed to update target node qemu memory limits during live migration")
		}

	}

	// detect an active domain on target node
	if domainIsActiveOnTarget(domain) && vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp == nil {

		// record the moment we detected the domain is running.
		// This is used as a trigger to help coordinate when CNI drivers
		// fail over the IP to the new pod.
		vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp = pointer.P(metav1.Now())
		log.Log.Object(vmi).Info("The target node received the running migrated domain")
	}

	// migration is complete, ack it
	if domainExists &&
		domain.Spec.Metadata.KubeVirt.Migration != nil &&
		domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp != nil {
		c.ackMigrationCompletion(vmi, domain)
	}

	if migrations.IsMigrating(vmi) {
		log.Log.Object(vmi).V(4).Info("migration is already in progress")
		return nil
	}

	destSrcPortsMap := c.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
	if len(destSrcPortsMap) == 0 {
		msg := "target migration listener is not up for this vmi"
		log.Log.Object(vmi).Error(msg)
		return fmt.Errorf(msg)
	}

	// advertise target address
	if vmi.Status.MigrationState.TargetNodeAddress != c.migrationIpAddress {
		portsList := make([]string, 0, len(destSrcPortsMap))

		for k := range destSrcPortsMap {
			portsList = append(portsList, k)
		}
		portsStrList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(portsList)), ","), "[]")
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), fmt.Sprintf("Migration Target is listening at %s, on ports: %s", c.migrationIpAddress, portsStrList))
		vmi.Status.MigrationState.TargetNodeAddress = c.migrationIpAddress
		vmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPortsMap
	}

	// If the migrated VMI requires dedicated CPUs, report the new pod CPU set to the source node
	// via the VMI migration status in order to patch the domain pre migration
	if vmi.IsCPUDedicated() {
		err := c.reportDedicatedCPUSetForMigratingVMI(vmi)
		if err != nil {
			return err
		}
		err = c.reportTargetTopologyForMigratingVMI(vmi)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *MigrationTargetController) Run(threadiness int, stopCh chan struct{}) {
	defer c.queue.ShutDown()
	log.Log.Info("Starting virt-handler target controller.")

	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// queue keys for previous Domains on the host that no longer exist
	// in the cache. This ensures we perform local cleanup of deleted VMs.
	for _, domain := range c.domainStore.List() {
		d := domain.(*api.Domain)
		vmiRef := v1.NewVMIReferenceWithUUID(
			d.ObjectMeta.Namespace,
			d.ObjectMeta.Name,
			d.Spec.Metadata.KubeVirt.UID)

		key := controller.VirtualMachineInstanceKey(vmiRef)

		_, exists, _ := c.vmiStore.GetByKey(key)
		if !exists {
			c.queue.Add(key)
		}
	}

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping virt-handler target controller.")
}

func (c *MigrationTargetController) runWorker() {
	for c.Execute() {
	}
}

func (c *MigrationTargetController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *MigrationTargetController) sync(key string, vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	// post migration clean up
	if vmi.Status.MigrationState == nil ||
		(vmi.Status.MigrationState.EndTimestamp != nil &&
			(vmi.Status.MigrationState.Completed || vmi.Status.MigrationState.Failed)) {
		c.migrationProxy.StopTargetListener(string(vmi.UID))
	}

	if domain != nil {
		log.Log.Object(vmi).Infof("VMI is in phase: %v | Domain status: %v, reason: %v", vmi.Status.Phase, domain.Status.Status, domain.Status.Reason)
	} else {
		log.Log.Object(vmi).Infof("VMI is in phase: %v | Domain does not exist", vmi.Status.Phase)
	}

	oldVmi := vmi.DeepCopy()
	oldStatus := oldVmi.Status
	oldLabels := oldVmi.Labels

	syncErr := c.processVMI(vmi)

	if syncErr != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		log.Log.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}
	updateErr := c.updateStatus(vmi, domain)

	if updateErr != nil {
		log.Log.Object(vmi).Reason(updateErr).Error("Updating the migration status failed.")
	}

	// update the VMI if necessary
	if !equality.Semantic.DeepEqual(oldStatus, vmi.Status) || !equality.Semantic.DeepEqual(oldLabels, vmi.Labels) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	if updateErr != nil {
		return updateErr
	}

	log.Log.Object(vmi).V(4).Info("Target synchronization loop succeeded.")
	return nil

}

func (c *MigrationTargetController) isMigrationTarget(vmi *v1.VirtualMachineInstance) bool {
	migrationTargetNodeName, _ := vmi.Labels[v1.MigrationTargetNodeNameLabel]
	return migrationTargetNodeName != "" && migrationTargetNodeName == c.host
}

func (c *MigrationTargetController) execute(key string) error {
	vmi, vmiExists, err := c.getVMIFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists || vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		log.Log.V(4).Infof("vmi for key %v is terminating, final or does not exists", key)
		return nil
	}

	if !c.vmiExpectations.SatisfiedExpectations(key) {
		log.Log.V(4).Object(vmi).Info("waiting for expectations to be satisfied")
		return nil
	}

	domain, domainExists, _, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		log.Log.V(4).Object(vmi).Infof("Detected stale vmi %s that still needs cleanup before new vmi with identical name/namespace can be processed", vmi.UID)
		return nil
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	if domainExists && !domainAlive {
		log.Log.V(4).Object(vmi).Info("domain is not alive")
		return nil
	}

	if vmi.Status.MigrationState == nil {
		log.Log.Object(vmi).V(4).Info("no migration is in progress")
		return nil
	}

	if !c.isMigrationTarget(vmi) {
		log.Log.Object(vmi).V(4).Info("not a migration target")
		return nil
	}

	return c.sync(key, vmi.DeepCopy(), domain)
}

func migrationNeedsFinalization(migrationState *v1.VirtualMachineInstanceMigrationState) bool {
	return migrationState != nil &&
		migrationState.StartTimestamp != nil &&
		migrationState.EndTimestamp != nil &&
		!migrationState.Completed &&
		!migrationState.Failed
}

func (c *MigrationTargetController) handleTargetMigrationProxy(vmi *v1.VirtualMachineInstance) error {
	// handle starting/stopping target migration proxy
	migrationTargetSockets := []string{}
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// Get the libvirt connection socket file on the destination pod.
	socketFile := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "libvirt/virtqemud-sock"), res.Pid())
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	migrationTargetSockets = append(migrationTargetSockets, socketFile)

	migrationPortsRange := migrationproxy.GetMigrationPortsList(vmi.IsBlockMigration())
	for _, port := range migrationPortsRange {
		key := migrationproxy.ConstructProxyKey(string(vmi.UID), port)
		// a proxy between the target direct qemu channel and the connector in the destination pod
		destSocketFile := migrationproxy.SourceUnixFile(baseDir, key)
		migrationTargetSockets = append(migrationTargetSockets, destSocketFile)
	}
	err = c.migrationProxy.StartTargetListener(string(vmi.UID), migrationTargetSockets)
	if err != nil {
		return err
	}
	return nil
}

func replaceMigratedVolumesStatus(vmi *v1.VirtualMachineInstance) {
	replaceVolsStatus := make(map[string]*v1.PersistentVolumeClaimInfo)
	for _, v := range vmi.Status.MigratedVolumes {
		replaceVolsStatus[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo
	}
	for i, v := range vmi.Status.VolumeStatus {
		if v.PersistentVolumeClaimInfo == nil {
			continue
		}
		if status, ok := replaceVolsStatus[v.PersistentVolumeClaimInfo.ClaimName]; ok {
			vmi.Status.VolumeStatus[i].PersistentVolumeClaimInfo = status
		}
	}
}

func (c *MigrationTargetController) syncVolumes(vmi *v1.VirtualMachineInstance) error {
	// The VolumeStatus is used to retrive additional information for the volume handling.
	// For example, for filesystem PVC, the information are used to create a right size image.
	// In the case of migrated volumes, we need to replace the original volume information with the
	// destination volume properties.
	replaceMigratedVolumesStatus(vmi)
	err := hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	// give containerDisks some time to become ready before throwing errors on retries
	info := c.launcherClients.GetLauncherClientInfo(vmi)
	if ready, err := c.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
		if err != nil {
			return err
		}
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return container_disk.ErrWaitingForDisks
	}

	// Mount container disks
	err = c.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return err
	}

	// Mount hotplug disks
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != types.UID("") {
		cgroupManager, err := getCgroupManager(vmi, c.host)
		if err != nil {
			return err
		}
		if err := c.hotplugVolumeMounter.MountFromPod(vmi, attachmentPodUID, cgroupManager); err != nil {
			return fmt.Errorf("failed to mount hotplug volumes: %v", err)
		}
	}

	return nil
}

func (c *MigrationTargetController) processVMI(vmi *v1.VirtualMachineInstance) error {
	isUnresponsive, isInitialized, err := c.launcherClients.IsLauncherClientUnresponsive(vmi)
	if err != nil {
		return err
	}

	if !isInitialized {
		log.Log.Object(vmi).V(4).Info("launcher client is not initialized")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	} else if isUnresponsive {
		return goerror.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	if migrationNeedsFinalization(vmi.Status.MigrationState) {
		log.Log.Object(vmi).V(4).Info("finalize migration")
		c.finalizeMigration(vmi)
		return nil
	}

	client, err := c.launcherClients.GetLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	if migrations.MigrationFailed(vmi) {
		// if the migration failed, signal the target pod it's okay to exit
		err = client.SignalTargetPodCleanup(vmi)
		if err != nil {
			return err
		}
		log.Log.Object(vmi).Infof("Signaled target pod for failed migration to clean up")
		// nothing left to do here if the migration failed.
		// Re-enqueue to trigger handler final cleanup
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second)
		return nil
	}
	if migrations.IsMigrating(vmi) {
		// If the migration has already started,
		// then there's nothing left to prepare on the target side
		log.Log.Object(vmi).V(4).Info("migration is already in progress")
		return nil
	}

	if err := c.setupNetwork(vmi, netsetup.FilterNetsForMigrationTarget(vmi), c.netConf); err != nil {
		return fmt.Errorf("failed to configure vmi network for migration target: %w", err)
	}

	vmiCopy := vmi.DeepCopy()

	err = c.syncVolumes(vmiCopy)
	if goerror.Is(err, container_disk.ErrWaitingForDisks) {
		log.Log.Object(vmi).V(4).Info("waiting for container disks to become ready")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	}
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to sync Volumes")
		return err
	}
	if err := c.setupDevicesOwnerships(vmiCopy, c.recorder); err != nil {
		return err
	}

	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)
	options.InterfaceDomainAttachment = domainspec.DomainAttachmentByInterfaceName(vmiCopy.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())

	if err := client.SyncMigrationTarget(vmiCopy, options); err != nil {
		return fmt.Errorf("syncing migration target failed: %v", err)
	}
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), VMIMigrationTargetPrepared)

	err = c.handleTargetMigrationProxy(vmiCopy)
	if err != nil {
		return fmt.Errorf("failed to handle post sync migration proxy: %v", err)
	}
	return nil
}

func (c *MigrationTargetController) addDeleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}

func (c *MigrationTargetController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.vmiExpectations.LowerExpectations(key, 1, 0)
		c.queue.Add(key)
	}
}

func (c *MigrationTargetController) addDomainFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *MigrationTargetController) deleteDomainFunc(obj interface{}) {
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
	}
	log.Log.Object(domain).Info("Domain deleted")
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *MigrationTargetController) updateDomainFunc(old, new interface{}) {
	newDomain := new.(*api.Domain)
	oldDomain := old.(*api.Domain)
	if oldDomain.Status.Status != newDomain.Status.Status || oldDomain.Status.Reason != newDomain.Status.Reason {
		log.Log.Object(newDomain).Infof("Domain is in state %s reason %s", newDomain.Status.Status, newDomain.Status.Reason)
	}

	if newDomain.ObjectMeta.DeletionTimestamp != nil {
		log.Log.Object(newDomain).Info("Domain is marked for deletion")
	}

	key, err := controller.KeyFunc(new)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *MigrationTargetController) reportDedicatedCPUSetForMigratingVMI(vmi *v1.VirtualMachineInstance) error {
	cgroupManager, err := getCgroupManager(vmi, c.host)
	if err != nil {
		return err
	}

	cpusetStr, err := cgroupManager.GetCpuSet()
	if err != nil {
		return err
	}

	cpuSet, err := hardware.ParseCPUSetLine(cpusetStr, 50000)
	if err != nil {
		return fmt.Errorf("failed to parse target VMI cpuset: %v", err)
	}

	vmi.Status.MigrationState.TargetCPUSet = cpuSet

	return nil
}

func (c *MigrationTargetController) reportTargetTopologyForMigratingVMI(vmi *v1.VirtualMachineInstance) error {
	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)
	topology, err := json.Marshal(options.Topology)
	if err != nil {
		return err
	}
	vmi.Status.MigrationState.TargetNodeTopology = string(topology)
	return nil
}

func (c *MigrationTargetController) hotplugCPU(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()

	removeVMIVCPUChangeConditionAndLabel := func() {
		delete(vmi.Labels, v1.VirtualMachinePodCPULimitsLabel)
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceVCPUChange)
	}
	defer removeVMIVCPUChangeConditionAndLabel()

	if !vmiConditions.HasCondition(vmi, v1.VirtualMachineInstanceVCPUChange) {
		return nil
	}

	if vmi.IsCPUDedicated() {
		cpuLimitStr, ok := vmi.Labels[v1.VirtualMachinePodCPULimitsLabel]
		if !ok || len(cpuLimitStr) == 0 {
			return fmt.Errorf("cannot read CPU limit from VMI annotation")
		}

		cpuLimit, err := strconv.Atoi(cpuLimitStr)
		if err != nil {
			return fmt.Errorf("cannot parse CPU limit from VMI annotation: %v", err)
		}

		vcpus := hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)
		if vcpus > int64(cpuLimit) {
			return fmt.Errorf("number of requested VCPUS (%d) exceeds the limit (%d)", vcpus, cpuLimit)
		}
	}

	options := virtualMachineOptions(
		nil,
		0,
		nil,
		c.capabilities,
		c.clusterConfig)

	if err := client.SyncVirtualMachineCPUs(vmi, options); err != nil {
		return err
	}

	if vmi.Status.CurrentCPUTopology == nil {
		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{}
	}

	vmi.Status.CurrentCPUTopology.Sockets = vmi.Spec.Domain.CPU.Sockets
	vmi.Status.CurrentCPUTopology.Cores = vmi.Spec.Domain.CPU.Cores
	vmi.Status.CurrentCPUTopology.Threads = vmi.Spec.Domain.CPU.Threads

	return nil
}

func (c *MigrationTargetController) hotplugMemory(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()

	removeVMIMemoryChangeLabel := func() {
		delete(vmi.Labels, v1.VirtualMachinePodMemoryRequestsLabel)
		delete(vmi.Labels, v1.MemoryHotplugOverheadRatioLabel)
	}
	defer removeVMIMemoryChangeLabel()

	if !vmiConditions.HasCondition(vmi, v1.VirtualMachineInstanceMemoryChange) {
		return nil
	}

	podMemReqStr := vmi.Labels[v1.VirtualMachinePodMemoryRequestsLabel]
	podMemReq, err := resource.ParseQuantity(podMemReqStr)
	if err != nil {
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
		return fmt.Errorf("cannot parse Memory requests from VMI label: %v", err)
	}

	overheadRatio := vmi.Labels[v1.MemoryHotplugOverheadRatioLabel]
	requiredMemory := services.GetMemoryOverhead(vmi, runtime.GOARCH, &overheadRatio)
	requiredMemory.Add(
		c.netBindingPluginMemoryCalculator.Calculate(vmi, c.clusterConfig.GetNetworkBindings()),
	)

	requiredMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())

	if podMemReq.Cmp(requiredMemory) < 0 {
		vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
		return fmt.Errorf("amount of requested guest memory (%s) exceeds the launcher memory request (%s)", vmi.Spec.Domain.Memory.Guest.String(), podMemReqStr)
	}

	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)

	if err := client.SyncVirtualMachineMemory(vmi, options); err != nil {
		// mark hotplug as failed
		vmiConditions.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstanceMemoryChange,
			Status:  k8sv1.ConditionFalse,
			Reason:  memoryHotplugFailedReason,
			Message: "memory hotplug failed, the VM configuration is not supported",
		})
		return err
	}

	vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
	vmi.Status.Memory.GuestRequested = vmi.Spec.Domain.Memory.Guest
	return nil
}

func removeMigratedVolumes(vmi *v1.VirtualMachineInstance) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	vmiConditions.RemoveCondition(vmi, v1.VirtualMachineInstanceVolumesChange)
	vmi.Status.MigratedVolumes = nil
}

func (c *MigrationTargetController) finalizeMigration(vmi *v1.VirtualMachineInstance) error {
	const errorMessage = "failed to finalize migration"

	client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("%s: %v", errorMessage, err)
	}

	if err := c.hotplugCPU(vmi, client); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to change vCPUs")
	}

	if err := c.hotplugMemory(vmi, client); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to update guest memory")
	}
	removeMigratedVolumes(vmi)

	options := &cmdv1.VirtualMachineOptions{}
	options.InterfaceMigration = domainspec.BindingMigrationByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())
	if err := client.FinalizeVirtualMachineMigration(vmi, options); err != nil {
		log.Log.Object(vmi).Reason(err).Error(errorMessage)
		return fmt.Errorf("%s: %v", errorMessage, err)
	}

	vmi.Status.MigrationState.Completed = true
	delete(vmi.Labels, v1.MigrationTargetNodeNameLabel)

	return nil
}
