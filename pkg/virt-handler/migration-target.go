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
 * Copyright The KubeVirt Authors.
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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	containerdisk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	hotplugvolume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	launcherclients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	vfsmanager "kubevirt.io/kubevirt/pkg/virt-handler/virtiofs"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type netBindingPluginMemoryCalculator interface {
	Calculate(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) resource.Quantity
}

type passtRepairTargetHandler interface {
	HandleMigrationTarget(*v1.VirtualMachineInstance, func(*v1.VirtualMachineInstance) (string, error)) error
}

type MigrationTargetController struct {
	*BaseController
	capabilities                     *libvirtxml.Caps
	containerDiskMounter             containerdisk.Mounter
	hotplugVolumeMounter             hotplugvolume.VolumeMounter
	migrationIpAddress               string
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator
	netConf                          netconf
	passtRepairHandler               passtRepairTargetHandler
	vfsManager                       *vfsmanager.VirtiofsManager
}

func NewMigrationTargetController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	host string,
	virtPrivateDir string,
	kubeletPodsDir string,
	migrationIpAddress string,
	launcherClients launcherclients.LauncherClientsManager,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	virtLauncherFSRunDirPattern string,
	capabilities *libvirtxml.Caps,
	netConf netconf,
	netStat netstat,
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator,
	passtRepairHandler passtRepairTargetHandler,
) (*MigrationTargetController, error) {

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-target"},
	)
	logger := log.Log.With("controller", "migration-target")

	baseCtrl, err := NewBaseController(
		logger,
		host,
		recorder,
		clientset,
		queue,
		vmiInformer,
		domainInformer,
		clusterConfig,
		podIsolationDetector,
		launcherClients,
		migrationProxy,
		virtLauncherFSRunDirPattern,
		netStat,
	)
	if err != nil {
		return nil, err
	}

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
		containerDiskMounter:             containerdisk.NewMounter(podIsolationDetector, containerDiskState, clusterConfig),
		hotplugVolumeMounter:             hotplugvolume.NewVolumeMounter(hotplugState, kubeletPodsDir, host),
		migrationIpAddress:               migrationIpAddress,
		netBindingPluginMemoryCalculator: netBindingPluginMemoryCalculator,
		netConf:                          netConf,
		passtRepairHandler:               passtRepairHandler,
		vfsManager:                       vfsmanager.NewVirtiofsManager("/pods"),
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
		UpdateFunc: c.updateFunc,
	})
	if err != nil {
		return nil, err
	}

	_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDomainFunc,
		DeleteFunc: c.deleteDomainFunc,
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
	// update the vmi migrationTransport to indicate that the next migration should use unix URI
	// new workloads will set the migrationTransport on creation, however legacy workloads
	// can make the switch only after the first migration
	vmi.Status.MigrationTransport = v1.MigrationTransportUnix
	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Migrated.String(), fmt.Sprintf("The VirtualMachineInstance migrated to node %s.", c.host))
	c.logger.Object(vmi).Info("The target node detected that the migration has completed")
}

func (c *MigrationTargetController) updateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if migrations.MigrationFailed(vmi) {
		c.logger.Object(vmi).V(4).Info("migration has failed, nothing to report on the target node")
		return nil
	}

	domainExists := domain != nil

	// detect domain on target node
	if domainExists && !vmi.Status.MigrationState.TargetNodeDomainDetected {
		// record that we've seen the domain populated on the target's node
		c.logger.Object(vmi).Info("The target node received the migrated domain")
		vmi.Status.MigrationState.TargetNodeDomainDetected = true
		if vmi.Status.MigrationState.TargetState != nil {
			vmi.Status.MigrationState.TargetState.DomainDetected = true
		}

		// adjust QEMU process memlock limits in order to enable old virt-launcher pod's to
		// perform host-devices hotplug post migration.
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
		if vmi.Status.MigrationState.TargetState != nil {
			vmi.Status.MigrationState.TargetState.DomainReadyTimestamp = vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp
		}

		c.logger.Object(vmi).Info("The target node received the running migrated domain")

		cm := controller.NewVirtualMachineInstanceConditionManager()
		cm.RemoveCondition(vmi, v1.VirtualMachineInstanceMigrationRequired)
	}

	// migration is complete, ack it
	if domainExists &&
		domain.Spec.Metadata.KubeVirt.Migration != nil &&
		domain.Spec.Metadata.KubeVirt.Migration.EndTimestamp != nil {
		c.ackMigrationCompletion(vmi, domain)
	}

	if migrations.IsMigrating(vmi) {
		c.logger.Object(vmi).V(4).Info("migration is already in progress")
		return nil
	}

	vmiUID := string(vmi.UID)
	if vmi.Status.MigrationState.SourceState != nil && vmi.Status.MigrationState.SourceState.VirtualMachineInstanceUID != nil {
		vmiUID = string(*vmi.Status.MigrationState.SourceState.VirtualMachineInstanceUID)
	}
	destSrcPortsMap := c.migrationProxy.GetTargetListenerPorts(vmiUID)
	if len(destSrcPortsMap) == 0 {
		msg := "target migration listener is not up for this vmi, giving it time"
		c.logger.Object(vmi).Info(msg)
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	}

	hostAddress := ""
	// advertise target address
	if vmi.Status.MigrationState != nil {
		hostAddress = vmi.Status.MigrationState.TargetNodeAddress
	}
	if hostAddress != c.migrationIpAddress {
		portsList := make([]string, 0, len(destSrcPortsMap))

		for k := range destSrcPortsMap {
			portsList = append(portsList, k)
		}
		portsStrList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(portsList)), ","), "[]")
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), fmt.Sprintf("Migration Target is listening at %s, on ports: %s", c.migrationIpAddress, portsStrList))
		vmi.Status.MigrationState.TargetNodeAddress = c.migrationIpAddress
		vmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPortsMap
		if vmi.Status.MigrationState.TargetState != nil {
			vmi.Status.MigrationState.TargetState.NodeAddress = pointer.P(c.migrationIpAddress)
			vmi.Status.MigrationState.TargetState.DirectMigrationNodePorts = destSrcPortsMap
		}
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
	c.logger.Info("Starting virt-handler target controller.")

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
	c.logger.Info("Stopping virt-handler target controller.")
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
		c.logger.Reason(err).Infof("re-enqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		c.logger.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *MigrationTargetController) updateVMI(vmi *v1.VirtualMachineInstance, oldStatus *v1.VirtualMachineInstanceStatus, oldLabels map[string]string) error {
	// update the VMI if necessary
	if !equality.Semantic.DeepEqual(oldStatus, vmi.Status) || !equality.Semantic.DeepEqual(oldLabels, vmi.Labels) {
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// finalCleanup is the last thing we run on finished migrations.
// If the function completes successfully:
// - On failure, virt-launcher will be notified and the virt-handler-managed volumes will be unmounted
// - All caches related to the VMI and domain will be dropped
// - The VMI will be removed from our informer
// - The migration proxy for the VMI will be stopped
// - The key will not be re-enqueued
func (c *MigrationTargetController) finalCleanup(vmi *v1.VirtualMachineInstance, oldStatus *v1.VirtualMachineInstanceStatus, oldLabels map[string]string, domain *api.Domain) error {
	if domainPausedFailedPostCopy(domain) {
		if vmi.Status.Phase == v1.Running {
			// In this function, we can usually clean up our (target) pod, since the migration is over.
			// However, there is one specific case where we can't: on post-copy migration failure that hasn't been acted on at the VMI level yet.
			// This is because, in that specific scenario, the source of truth is in the libvirt domain running in our pod.
			// Once we terminate our pod, the information is gone.
			c.logger.Object(vmi).Info("we're the target of a failed post-copy but the VMI is still running, waiting a sec")
			c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
			return nil
		}
		if vmi.Status.Phase == v1.Succeeded {
			// Ensuring failed post-copy migrations don't lead to a successful VMI, shouldn't be needed
			c.logger.Object(vmi).Warning("VMI status wrongly set to succeeded, this shouldn't happen, fixing VMI phase")
			vmi.Status.Phase = v1.Failed
		}
	}

	defer c.migrationProxy.StopTargetListener(string(vmi.UID))
	defer c.launcherClients.CloseLauncherClient(vmi)
	client, err := c.launcherClients.GetLauncherClient(vmi)
	if err != nil {
		return err
	}

	if vmi.Status.MigrationState.Failed {
		err = client.SignalTargetPodCleanup(vmi)
		if err != nil {
			c.logger.Object(vmi).Warningf("Failed to signal target pod cleanup: %v, ignoring.", err)
		}
		err = c.unmountVolumes(vmi)
		if err != nil {
			return err
		}
		c.logger.Object(vmi).Infof("Signaled target pod for failed migration to clean up")

		// tear down network cache
		if err = c.netConf.Teardown(vmi); err != nil {
			return fmt.Errorf("failed to delete VMI Network cache files: %s", err.Error())
		}
		c.netStat.Teardown(vmi)
		// The migration failed. As the target virt-handler, the domain doesn't belong to our store anymore
		if err = c.domainStore.Delete(vmi); err != nil {
			return err
		}
	} else {
		options := &cmdv1.VirtualMachineOptions{}
		options.InterfaceMigration = domainspec.BindingMigrationByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())
		if err = client.FinalizeVirtualMachineMigration(vmi, options); err != nil {
			return err
		}
	}

	// Effectively removes the VMI from our VMI informer
	delete(vmi.Labels, v1.MigrationTargetNodeNameLabel)
	delete(vmi.Annotations, v1.CreateMigrationTarget)
	return c.updateVMI(vmi, oldStatus, oldLabels)
}

func (c *MigrationTargetController) sync(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	oldStatus := vmi.Status
	oldLabels := vmi.Labels
	vmi = vmi.DeepCopy()

	// post-migration clean up
	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.EndTimestamp != nil &&
		(vmi.Status.MigrationState.Completed || vmi.Status.MigrationState.Failed) {
		return c.finalCleanup(vmi, &oldStatus, oldLabels, domain)
	}

	if domain != nil {
		c.logger.Object(vmi).Infof("VMI is in phase: %v | Domain status: %v, reason: %v", vmi.Status.Phase, domain.Status.Status, domain.Status.Reason)
	} else {
		c.logger.Object(vmi).Infof("VMI is in phase: %v | Domain does not exist", vmi.Status.Phase)
	}

	syncErr := c.processVMI(vmi)
	if syncErr != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())
		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		c.logger.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	updateErr := c.updateStatus(vmi, domain)
	if updateErr != nil {
		c.logger.Object(vmi).Reason(updateErr).Error("Updating the migration status failed.")
		return updateErr
	}

	// If processVMI is just waiting for something to be ready, we can't and don't need to increase expectations.
	// We can't because the VMI may not update before the thing is ready, deadlocking us
	// We don't need to because every time processVMI is waiting for something it re-adds the key to the queue
	updateVMIErr := c.updateVMI(vmi, &oldStatus, oldLabels)
	if updateVMIErr != nil {
		return updateVMIErr
	}

	c.logger.Object(vmi).V(4).Info("Target synchronization loop done.")
	return syncErr
}

func (c *MigrationTargetController) isMigrationTarget(vmi *v1.VirtualMachineInstance) bool {
	migrationTargetNodeName, _ := vmi.Labels[v1.MigrationTargetNodeNameLabel]
	return migrationTargetNodeName == c.host
}

func (c *MigrationTargetController) execute(key string) error {
	vmi, vmiExists, err := c.getVMIFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists {
		c.logger.V(4).Infof("vmi for key %v does not exists", key)
		return nil
	}

	if vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		c.logger.V(4).Infof("vmi for key %v is terminating or final, doing only a best-effort cleanup", key)
		_ = c.netConf.Teardown(vmi)
		c.netStat.Teardown(vmi)
		return nil
	}

	domain, domainExists, domUID, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}

	if domainExists && domUID != "" && domUID != vmi.UID {
		err = c.domainStore.Delete(domain)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("had to delete stale domain (UID %s) that didn't match current VMI (UID %s)", domUID, vmi.UID)
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	if domainExists && !domainAlive {
		c.logger.V(4).Object(vmi).Info("domain is not alive")
		return nil
	}

	if vmi.Status.MigrationState == nil {
		c.logger.Object(vmi).V(4).Info("no migration is in progress")
		return nil
	}

	if !c.isMigrationTarget(vmi) {
		c.logger.Object(vmi).V(4).Info("not a migration target")
		return nil
	}

	return c.sync(vmi, domain)
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
	var migrationTargetSockets []string
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	vmiUID := string(vmi.UID)
	if vmi.Status.MigrationState.SourceState != nil && vmi.Status.MigrationState.SourceState.VirtualMachineInstanceUID != nil {
		vmiUID = string(*vmi.Status.MigrationState.SourceState.VirtualMachineInstanceUID)
	}

	// Get the libvirt connection socket file on the destination pod.
	socketFile := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "libvirt/virtqemud-sock"), res.Pid())
	// the migration-proxy is no longer shared via host mount, so we
	// pass in the virt-launcher's baseDir to reach the unix sockets.
	baseDir := fmt.Sprintf(filepath.Join(c.virtLauncherFSRunDirPattern, "kubevirt"), res.Pid())
	migrationTargetSockets = append(migrationTargetSockets, socketFile)

	migrationPortsRange := migrationproxy.GetMigrationPortsList(vmi.IsBlockMigration())
	for _, port := range migrationPortsRange {
		key := migrationproxy.ConstructProxyKey(vmiUID, port)
		// a proxy between the target direct qemu channel and the connector in the destination pod
		destSocketFile := migrationproxy.SourceUnixFile(baseDir, key)
		migrationTargetSockets = append(migrationTargetSockets, destSocketFile)
	}
	err = c.migrationProxy.StartTargetListener(vmiUID, migrationTargetSockets)
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
	// The VolumeStatus is used to retrieve additional information for the volume handling.
	// For example, for filesystem PVC, the information is used to create a right size image.
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
		return containerdisk.ErrWaitingForDisks
	}

	// Mount container disks
	err = c.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return err
	}

	// Mount hotplug disks
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != "" {
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

func (c *MigrationTargetController) unmountVolumes(vmi *v1.VirtualMachineInstance) error {
	// The VolumeStatus is used to retrieve additional information for the volume handling.
	// For example, for filesystem PVC, the information is used to create a right size image.
	// In the case of migrated volumes, we need to replace the original volume information with the
	// destination volume properties.
	replaceMigratedVolumesStatus(vmi)
	err := hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	if err = c.containerDiskMounter.Unmount(vmi); err != nil {
		return err
	}

	// Mount hotplug disks
	if attachmentPodUID := vmi.Status.MigrationState.TargetAttachmentPodUID; attachmentPodUID != "" {
		cgroupManager, err := getCgroupManager(vmi, c.host)
		if err != nil {
			return err
		}
		if err = c.hotplugVolumeMounter.Unmount(vmi, cgroupManager); err != nil {
			return fmt.Errorf("failed to unmount hotplug volumes: %v", err)
		}
	}

	return nil
}

// processVMI handles the necessary operations to prepare/cleanup for/after a migration.
// It returns an error and a boolean informing the caller if the key was re-enqueued by us.
func (c *MigrationTargetController) processVMI(vmi *v1.VirtualMachineInstance) error {
	if migrationNeedsFinalization(vmi.Status.MigrationState) {
		c.logger.Object(vmi).V(4).Info("finalize migration")
		return c.finalizeMigration(vmi)
	}

	client, err := c.launcherClients.GetLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	shouldReturn, err := c.checkLauncherClient(vmi)
	if shouldReturn {
		return err
	}

	if migrations.IsMigrating(vmi) {
		// If the migration has already started,
		// then there's nothing left to prepare on the target side
		c.logger.Object(vmi).V(4).Info("migration is already in progress")
		return nil
	}

	vmi = vmi.DeepCopy()

	err = c.syncVolumes(vmi)
	if goerror.Is(err, containerdisk.ErrWaitingForDisks) {
		c.logger.Object(vmi).V(4).Info("waiting for container disks to become ready")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return nil
	}
	if err != nil {
		c.logger.Object(vmi).Reason(err).Error("Failed to sync Volumes")
		return err
	}

	// Look for placeholder virtiofs sockets and launch the dispatcher
	if err := c.vfsManager.StartVirtiofsDispatcher(vmi); err != nil {
		return fmt.Errorf("failed to start the virtiofs dispatcher: %w", err)
	}

	if err := c.setupNetwork(vmi, netsetup.FilterNetsForMigrationTarget(vmi), c.netConf); err != nil {
		return fmt.Errorf("failed to configure vmi network for migration target: %w", err)
	}

	if err := c.setupDevicesOwnerships(vmi, c.recorder); err != nil {
		return err
	}

	options := virtualMachineOptions(nil, 0, nil, c.capabilities, c.clusterConfig)
	options.InterfaceDomainAttachment = domainspec.DomainAttachmentByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())

	if c.clusterConfig.PasstIPStackMigrationEnabled() {
		if err := c.passtRepairHandler.HandleMigrationTarget(vmi, c.passtSocketDirOnHostForVMI); err != nil {
			c.logger.Object(vmi).Warningf("failed to call passt-repair for migration target, %v", err)
		}
	}

	if err := client.SyncMigrationTarget(vmi, options); err != nil {
		return fmt.Errorf("syncing migration target failed: %v", err)
	}

	err = c.handleTargetMigrationProxy(vmi)
	if err != nil {
		return fmt.Errorf("failed to handle post sync migration proxy: %v", err)
	}

	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.PreparingTarget.String(), VMIMigrationTargetPrepared)

	return nil
}

func (c *MigrationTargetController) addFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *MigrationTargetController) deleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *MigrationTargetController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
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
			c.logger.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		domain, ok = tombstone.Obj.(*api.Domain)
		if !ok {
			c.logger.Reason(fmt.Errorf("tombstone contained object that is not a domain %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	c.logger.Object(domain).Info("Domain deleted")
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *MigrationTargetController) updateDomainFunc(old, new interface{}) {
	newDomain := new.(*api.Domain)
	oldDomain := old.(*api.Domain)
	if oldDomain.Status.Status != newDomain.Status.Status || oldDomain.Status.Reason != newDomain.Status.Reason {
		c.logger.Object(newDomain).Infof("Domain is in state %s reason %s", newDomain.Status.Status, newDomain.Status.Reason)
	}

	if newDomain.ObjectMeta.DeletionTimestamp != nil {
		c.logger.Object(newDomain).Info("Domain is marked for deletion")
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
		c.logger.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to change vCPUs")
	}

	if err := c.hotplugMemory(vmi, client); err != nil {
		c.logger.Object(vmi).Reason(err).Error(errorMessage)
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), "failed to update guest memory")
	}
	removeMigratedVolumes(vmi)

	options := &cmdv1.VirtualMachineOptions{}
	options.InterfaceMigration = domainspec.BindingMigrationByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())
	if err := client.FinalizeVirtualMachineMigration(vmi, options); err != nil {
		c.logger.Object(vmi).Reason(err).Error(errorMessage)
		return fmt.Errorf("%s: %v", errorMessage, err)
	}

	vmi.Status.MigrationState.Completed = true
	return nil
}
