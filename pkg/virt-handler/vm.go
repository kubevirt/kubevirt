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
	goerror "errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"golang.org/x/sys/unix"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/executor"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/network/domainspec"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	containerdisk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	deviceManager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat"
	hotplugvolume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	launcherclients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	multipathmonitor "kubevirt.io/kubevirt/pkg/virt-handler/multipath-monitor"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type netstat interface {
	UpdateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error
	Teardown(vmi *v1.VirtualMachineInstance)
}

type downwardMetricsManager interface {
	Run(stopCh chan struct{})
	StartServer(vmi *v1.VirtualMachineInstance, pid int) error
	StopServer(vmi *v1.VirtualMachineInstance)
}

type VirtualMachineController struct {
	*BaseController
	capabilities             *libvirtxml.Caps
	clientset                kubecli.KubevirtClient
	containerDiskMounter     containerdisk.Mounter
	downwardMetricsManager   downwardMetricsManager
	hotplugVolumeMounter     hotplugvolume.VolumeMounter
	hostCpuModel             string
	ioErrorRetryManager      *FailRetryManager
	deviceManagerController  *deviceManager.DeviceController
	heartBeat                *heartbeat.HeartBeat
	heartBeatInterval        time.Duration
	netConf                  netconf
	sriovHotplugExecutorPool *executor.RateLimitedExecutorPool
	vmiExpectations          *controller.UIDTrackingControllerExpectations
	vmiGlobalStore           cache.Store
	multipathSocketMonitor   *multipathmonitor.MultipathSocketMonitor
	cbtHandler               *CBTHandler
}

var getCgroupManager = func(vmi *v1.VirtualMachineInstance, host string) (cgroup.Manager, error) {
	return cgroup.NewManagerFromVM(vmi, host)
}

func NewVirtualMachineController(
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	nodeStore cache.Store,
	host string,
	virtPrivateDir string,
	kubeletPodsDir string,
	launcherClients launcherclients.LauncherClientsManager,
	vmiInformer cache.SharedIndexInformer,
	vmiGlobalStore cache.Store,
	domainInformer cache.SharedInformer,
	maxDevices int,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	migrationProxy migrationproxy.ProxyManager,
	downwardMetricsManager downwardMetricsManager,
	capabilities *libvirtxml.Caps,
	hostCpuModel string,
	netConf netconf,
	netStat netstat,
	cbtHandler *CBTHandler,
) (*VirtualMachineController, error) {

	queue := workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-handler-vm"},
	)
	logger := log.Log.With("controller", "vm")

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
		"/proc/%d/root/var/run",
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

	c := &VirtualMachineController{
		BaseController:           baseCtrl,
		capabilities:             capabilities,
		clientset:                clientset,
		containerDiskMounter:     containerdisk.NewMounter(podIsolationDetector, containerDiskState, clusterConfig),
		downwardMetricsManager:   downwardMetricsManager,
		hotplugVolumeMounter:     hotplugvolume.NewVolumeMounter(hotplugState, kubeletPodsDir, host),
		hostCpuModel:             hostCpuModel,
		ioErrorRetryManager:      NewFailRetryManager("io-error-retry", 10*time.Second, 3*time.Minute, 30*time.Second),
		heartBeatInterval:        1 * time.Minute,
		netConf:                  netConf,
		sriovHotplugExecutorPool: executor.NewRateLimitedExecutorPool(executor.NewExponentialLimitedBackoffCreator()),
		vmiExpectations:          controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		vmiGlobalStore:           vmiGlobalStore,
		multipathSocketMonitor:   multipathmonitor.NewMultipathSocketMonitor(),
		cbtHandler:               cbtHandler,
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDeleteFunc,
		DeleteFunc: c.addDeleteFunc,
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

	permissions := "rw"
	if cgroups.IsCgroup2UnifiedMode() {
		// Need 'rwm' permissions otherwise ebpf filtering program attached by runc
		// will deny probing the device file with 'access' syscall. That in turn
		// will lead to virtqemud failure on VM startup.
		// This has been fixed upstream:
		//   https://github.com/opencontainers/runc/pull/2796
		// but the workaround is still needed to support previous versions without
		// the patch.
		permissions = "rwm"
	}

	c.deviceManagerController = deviceManager.NewDeviceController(
		c.host,
		maxDevices,
		permissions,
		deviceManager.PermanentHostDevicePlugins(maxDevices, permissions),
		clusterConfig,
		nodeStore)
	c.heartBeat = heartbeat.NewHeartBeat(clientset.CoreV1(), c.deviceManagerController, clusterConfig, host)

	return c, nil
}

func (c *VirtualMachineController) Run(threadiness int, stopCh chan struct{}) {
	defer c.queue.ShutDown()
	c.logger.Info("Starting virt-handler vms controller.")

	go c.deviceManagerController.Run(stopCh)

	go c.downwardMetricsManager.Run(stopCh)

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
	c.multipathSocketMonitor.Run()

	heartBeatDone := c.heartBeat.Run(c.heartBeatInterval, stopCh)

	go c.ioErrorRetryManager.Run(stopCh)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-heartBeatDone
	<-stopCh
	c.multipathSocketMonitor.Close()
	c.logger.Info("Stopping virt-handler vms controller.")
}

func (c *VirtualMachineController) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachineController) Execute() bool {
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

func (c *VirtualMachineController) execute(key string) error {
	vmi, vmiExists, err := c.getVMIFromCache(key)
	if err != nil {
		return err
	}

	if !vmiExists {
		// the vmiInformer probably has to catch up to the domainInformer
		// which already sees the vmi, so let's fetch it from the global
		// vmi informer to make sure the vmi has actually been deleted
		c.logger.V(4).Infof("fetching vmi for key %v from the global informer", key)
		obj, exists, err := c.vmiGlobalStore.GetByKey(key)
		if err != nil {
			return err
		}
		if exists {
			vmi = obj.(*v1.VirtualMachineInstance)
		}
		vmiExists = exists
	}

	if !vmiExists {
		c.vmiExpectations.DeleteExpectations(key)
	} else if !c.vmiExpectations.SatisfiedExpectations(key) {
		return nil
	}

	domain, domainExists, domainCachedUID, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}
	c.logger.Object(vmi).V(4).Infof("domain exists %v", domainExists)

	if !vmiExists && string(domainCachedUID) != "" {
		// it's possible to discover the UID from cache even if the domain
		// doesn't technically exist anymore
		vmi.UID = domainCachedUID
		c.logger.Object(vmi).Infof("Using cached UID for vmi found in domain cache")
	}

	// As a last effort, if the UID still can't be determined attempt
	// to retrieve it from the ghost record
	if string(vmi.UID) == "" {
		uid := virtcache.GhostRecordGlobalStore.LastKnownUID(key)
		if uid != "" {
			c.logger.Object(vmi).V(3).Infof("ghost record cache provided %s as UID", uid)
			vmi.UID = uid
		}
	}

	if vmiExists && domainExists && domain.Spec.Metadata.KubeVirt.UID != vmi.UID {
		oldVMI := v1.NewVMIReferenceFromNameWithNS(vmi.Namespace, vmi.Name)
		oldVMI.UID = domain.Spec.Metadata.KubeVirt.UID
		expired, initialized, err := c.launcherClients.IsLauncherClientUnresponsive(oldVMI)
		if err != nil {
			return err
		}
		// If we found an outdated domain which is also not alive anymore, clean up
		if !initialized {
			c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
			return nil
		} else if expired {
			c.logger.Object(oldVMI).Infof("Detected stale vmi %s that still needs cleanup before new vmi %s with identical name/namespace can be processed", oldVMI.UID, vmi.UID)
			err = c.processVmCleanup(oldVMI)
			if err != nil {
				return err
			}
			// Make sure we re-enqueue the key to ensure this new VMI is processed
			// after the stale domain is removed
			c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*5)
		}

		return nil
	}

	if domainExists &&
		(domainMigrated(domain) || domain.DeletionTimestamp != nil) {
		c.logger.Object(vmi).V(4).Info("detected orphan vmi")
		return c.deleteVM(vmi)
	}

	if migrations.IsMigrating(vmi) && (vmi.Status.Phase == v1.Failed) {
		c.logger.V(1).Infof("cleaning up VMI key %v as migration is in progress and the vmi is failed", key)
		err = c.processVmCleanup(vmi)
		if err != nil {
			return err
		}
	}

	if vmi.DeletionTimestamp == nil && isMigrationInProgress(vmi, domain) {
		c.logger.V(4).Infof("ignoring key %v as migration is in progress", key)
		return nil
	}

	if vmiExists && !c.isVMIOwnedByNode(vmi) {
		c.logger.Object(vmi).V(4).Info("ignoring vmi as it is not owned by this node")
		return nil
	}

	if vmiExists && vmi.IsMigrationSource() {
		c.logger.Object(vmi).V(4).Info("ignoring vmi as it is a migration source")
		return nil
	}

	return c.sync(key,
		vmi.DeepCopy(),
		vmiExists,
		domain,
		domainExists)

}

type vmiIrrecoverableError struct {
	msg string
}

func (e *vmiIrrecoverableError) Error() string { return e.msg }

func formatIrrecoverableErrorMessage(domain *api.Domain) string {
	msg := "unknown reason"
	if domainPausedFailedPostCopy(domain) {
		msg = "VMI is irrecoverable due to failed post-copy migration"
	}
	return msg
}

// teardownNetwork performs network cache cleanup for a specific VMI.
func (c *VirtualMachineController) teardownNetwork(vmi *v1.VirtualMachineInstance) {
	if string(vmi.UID) == "" {
		return
	}
	if err := c.netConf.Teardown(vmi); err != nil {
		c.logger.Reason(err).Errorf("failed to delete VMI Network cache files: %s", err.Error())
	}
	c.netStat.Teardown(vmi)
}

func (c *VirtualMachineController) deleteVM(vmi *v1.VirtualMachineInstance) error {
	err := c.processVmDelete(vmi)
	if err != nil {
		return err
	}
	// we can perform the cleanup immediately after
	// the successful delete here because we don't have
	// to report the deletion results on the VMI status
	// in this case.
	err = c.processVmCleanup(vmi)
	if err != nil {
		return err
	}

	return nil
}

// Determine if gracefulShutdown has been triggered by virt-launcher
func (c *VirtualMachineController) hasGracefulShutdownTrigger(domain *api.Domain) bool {
	if domain == nil {
		return false
	}
	gracePeriod := domain.Spec.Metadata.KubeVirt.GracePeriod

	return gracePeriod != nil &&
		gracePeriod.MarkedForGracefulShutdown != nil &&
		*gracePeriod.MarkedForGracefulShutdown
}

func (c *VirtualMachineController) sync(key string,
	vmi *v1.VirtualMachineInstance,
	vmiExists bool,
	domain *api.Domain,
	domainExists bool) error {

	oldStatus := vmi.Status.DeepCopy()
	oldSpec := vmi.Spec.DeepCopy()

	// set to true when domain needs to be shutdown.
	shouldShutdown := false
	// set to true when domain needs to be removed from libvirt.
	shouldDelete := false
	// set to true when VirtualMachineInstance is active or about to become active.
	shouldUpdate := false
	// set to true when unrecoverable domain needs to be destroyed non-gracefully.
	forceShutdownIrrecoverable := false

	c.logger.V(3).Infof("Processing event %v", key)

	if vmiExists && domainExists {
		c.logger.Object(vmi).Infof("VMI is in phase: %v | Domain status: %v, reason: %v", vmi.Status.Phase, domain.Status.Status, domain.Status.Reason)
	} else if vmiExists {
		c.logger.Object(vmi).Infof("VMI is in phase: %v | Domain does not exist", vmi.Status.Phase)
	} else if domainExists {
		vmiRef := v1.NewVMIReferenceWithUUID(domain.ObjectMeta.Namespace, domain.ObjectMeta.Name, domain.Spec.Metadata.KubeVirt.UID)
		c.logger.Object(vmiRef).Infof("VMI does not exist | Domain status: %v, reason: %v", domain.Status.Status, domain.Status.Reason)
	} else {
		c.logger.Info("VMI does not exist | Domain does not exist")
	}

	domainAlive := domainExists &&
		domain.Status.Status != api.Shutoff &&
		domain.Status.Status != api.Crashed &&
		domain.Status.Status != ""

	forceShutdownIrrecoverable = domainExists && domainPausedFailedPostCopy(domain)

	gracefulShutdown := c.hasGracefulShutdownTrigger(domain)
	if gracefulShutdown && vmi.IsRunning() {
		if domainAlive {
			c.logger.Object(vmi).V(3).Info("Shutting down due to graceful shutdown signal.")
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
			c.logger.Object(vmi).V(3).Info("Shutting down domain for deleted VirtualMachineInstance object.")
			shouldShutdown = true
		} else {
			// The VirtualMachineInstance is deleted on the cluster, and domain is not alive
			// then delete the domain.
			c.logger.Object(vmi).V(3).Info("Deleting domain for deleted VirtualMachineInstance object.")
			shouldDelete = true
		}
	}

	// Determine if VirtualMachineInstance is being deleted.
	if vmiExists && vmi.ObjectMeta.DeletionTimestamp != nil {
		if domainAlive {
			c.logger.Object(vmi).V(3).Info("Shutting down domain for VirtualMachineInstance with deletion timestamp.")
			shouldShutdown = true
		} else {
			c.logger.Object(vmi).V(3).Info("Deleting domain for VirtualMachineInstance with deletion timestamp.")
			shouldDelete = true
		}
	}

	// Determine if domain needs to be deleted as a result of VirtualMachineInstance
	// shutting down naturally (guest internal invoked shutdown)
	if vmiExists && vmi.IsFinal() {
		c.logger.Object(vmi).V(3).Info("Removing domain and ephemeral data for finalized vmi.")
		shouldDelete = true
	}

	if !domainAlive && domainExists && !vmi.IsFinal() {
		c.logger.Object(vmi).V(3).Info("Deleting inactive domain for vmi.")
		shouldDelete = true
	}

	// Determine if an active (or about to be active) VirtualMachineInstance should be updated.
	if vmiExists && !vmi.IsFinal() {
		// requiring the phase of the domain and VirtualMachineInstance to be in sync is an
		// optimization that prevents unnecessary re-processing VMIs during the start flow.
		phase, err := c.calculateVmPhaseForStatusReason(domain, vmi)
		if err != nil {
			return err
		}
		if vmi.Status.Phase == phase {
			shouldUpdate = true
		}

		if shouldDelay, delay := c.ioErrorRetryManager.ShouldDelay(string(vmi.UID), func() bool {
			return isIOError(shouldUpdate, domainExists, domain)
		}); shouldDelay {
			shouldUpdate = false
			c.logger.Object(vmi).Infof("Delay vm update for %f seconds", delay.Seconds())
			c.queue.AddAfter(key, delay)
		}
	}

	var syncErr error

	// Process the VirtualMachineInstance update in this order.
	// * Shutdown and Deletion due to VirtualMachineInstance deletion, process stopping, graceful shutdown trigger, etc...
	// * Cleanup of already shutdown and Deleted VMIs
	// * Update due to spec change and initial start flow.
	switch {
	case shouldShutdown:
		c.logger.Object(vmi).V(3).Info("Processing shutdown.")
		syncErr = c.processVmShutdown(vmi, domain)
	case forceShutdownIrrecoverable:
		msg := formatIrrecoverableErrorMessage(domain)
		c.logger.Object(vmi).V(3).Infof("Processing a destruction of an irrecoverable domain - %s.", msg)
		syncErr = c.processVmDestroy(vmi, domain)
		if syncErr == nil {
			syncErr = &vmiIrrecoverableError{msg}
		}
	case shouldDelete:
		c.logger.Object(vmi).V(3).Info("Processing deletion.")
		syncErr = c.deleteVM(vmi)
	case shouldUpdate:
		c.logger.Object(vmi).V(3).Info("Processing vmi update")
		syncErr = c.processVmUpdate(vmi, domain)
	default:
		c.logger.Object(vmi).V(3).Info("No update processing required")
	}
	if syncErr != nil && !vmi.IsFinal() {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.SyncFailed.String(), syncErr.Error())

		// `syncErr` will be propagated anyway, and it will be logged in `re-enqueueing`
		// so there is no need to log it twice in hot path without increased verbosity.
		c.logger.Object(vmi).Reason(syncErr).Error("Synchronizing the VirtualMachineInstance failed.")
	}

	// Update the VirtualMachineInstance status, if the VirtualMachineInstance exists
	if vmiExists {
		vmi.Spec = *oldSpec
		if err := c.updateVMIStatus(oldStatus, vmi, domain, syncErr); err != nil {
			c.logger.Object(vmi).Reason(err).Error("Updating the VirtualMachineInstance status failed.")
			return err
		}
	}

	if syncErr != nil {
		return syncErr
	}

	c.logger.Object(vmi).V(3).Info("Synchronization loop succeeded.")
	return nil

}

func (c *VirtualMachineController) processVmCleanup(vmi *v1.VirtualMachineInstance) error {
	vmiId := string(vmi.UID)

	c.logger.Object(vmi).Infof("Performing final local cleanup for vmi with uid %s", vmiId)

	c.migrationProxy.StopTargetListener(vmiId)
	c.migrationProxy.StopSourceListener(vmiId)

	c.downwardMetricsManager.StopServer(vmi)

	// Unmount container disks and clean up remaining files
	if err := c.containerDiskMounter.Unmount(vmi); err != nil {
		return err
	}

	// UnmountAll does the cleanup on the "best effort" basis: it is
	// safe to pass a nil cgroupManager.
	cgroupManager, _ := getCgroupManager(vmi, c.host)
	if err := c.hotplugVolumeMounter.UnmountAll(vmi, cgroupManager); err != nil {
		return err
	}

	c.teardownNetwork(vmi)

	c.sriovHotplugExecutorPool.Delete(vmi.UID)

	// Watch dog file and command client must be the last things removed here
	c.launcherClients.CloseLauncherClient(vmi)

	// Remove the domain from cache in the event that we're performing
	// a final cleanup and never received the "DELETE" event. This is
	// possible if the VMI pod goes away before we receive the final domain
	// "DELETE"
	domain := api.NewDomainReferenceFromName(vmi.Namespace, vmi.Name)
	c.logger.Object(domain).Infof("Removing domain from cache during final cleanup")
	return c.domainStore.Delete(domain)
}

func (c *VirtualMachineController) processVmDestroy(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	tryGracefully := false
	return c.helperVmShutdown(vmi, domain, tryGracefully)
}

func (c *VirtualMachineController) processVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	tryGracefully := true
	return c.helperVmShutdown(vmi, domain, tryGracefully)
}

const firstGracefulShutdownAttempt = -1

// Determines if a domain's grace period has expired during shutdown.
// If the grace period has started but not expired, timeLeft represents
// the time in seconds left until the period expires.
// If the grace period has not started, timeLeft will be set to -1.
func (c *VirtualMachineController) hasGracePeriodExpired(terminationGracePeriod *int64, dom *api.Domain) (bool, int64) {
	var hasExpired bool
	var timeLeft int64

	gracePeriod := int64(0)
	if terminationGracePeriod != nil {
		gracePeriod = *terminationGracePeriod
	} else if dom != nil && dom.Spec.Metadata.KubeVirt.GracePeriod != nil {
		gracePeriod = dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds
	}

	// If gracePeriod == 0, then there will be no startTime set, deletion
	// should occur immediately during shutdown.
	if gracePeriod == 0 {
		hasExpired = true
		return hasExpired, timeLeft
	}

	startTime := int64(0)
	if dom != nil && dom.Spec.Metadata.KubeVirt.GracePeriod != nil && dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp != nil {
		startTime = dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp.UTC().Unix()
	}

	if startTime == 0 {
		// If gracePeriod > 0, then the shutdown signal needs to be sent
		// and the gracePeriod start time needs to be set.
		timeLeft = firstGracefulShutdownAttempt
		return hasExpired, timeLeft
	}

	now := time.Now().UTC().Unix()
	diff := now - startTime

	if diff >= gracePeriod {
		hasExpired = true
		return hasExpired, timeLeft
	}

	timeLeft = gracePeriod - diff
	if timeLeft < 1 {
		timeLeft = 1
	}
	return hasExpired, timeLeft
}

func (c *VirtualMachineController) helperVmShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain, tryGracefully bool) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)
	if err != nil {
		return err
	}

	if domainHasGracePeriod(domain) && tryGracefully {
		if expired, timeLeft := c.hasGracePeriodExpired(vmi.Spec.TerminationGracePeriodSeconds, domain); !expired {
			return c.handleVMIShutdown(vmi, domain, client, timeLeft)
		}
		c.logger.Object(vmi).Infof("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
	} else {
		c.logger.Object(vmi).Infof("Graceful shutdown not set, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())
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

	c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMIStopping)

	return nil
}

func (c *VirtualMachineController) handleVMIShutdown(vmi *v1.VirtualMachineInstance, domain *api.Domain, client cmdclient.LauncherClient, timeLeft int64) error {
	if domain.Status.Status != api.Shutdown {
		return c.shutdownVMI(vmi, client, timeLeft)
	}
	c.logger.V(4).Object(vmi).Infof("%s is already shutting down.", vmi.GetObjectMeta().GetName())
	return nil
}

func (c *VirtualMachineController) shutdownVMI(vmi *v1.VirtualMachineInstance, client cmdclient.LauncherClient, timeLeft int64) error {
	err := client.ShutdownVirtualMachine(vmi)
	if err != nil && !cmdclient.IsDisconnected(err) {
		// Only report err if it wasn't the result of a disconnect.
		//
		// Both virt-launcher and virt-handler are trying to destroy
		// the VirtualMachineInstance at the same time. It's possible the client may get
		// disconnected during the kill request, which shouldn't be
		// considered an error.
		return err
	}

	c.logger.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

	// Only create a VMIGracefulShutdown event for the first attempt as we can
	// easily hit the default burst limit of 25 for the
	// EventSourceObjectSpamFilter when gracefully shutting down VMIs with a
	// large TerminationGracePeriodSeconds value set. Hitting this limit can
	// result in the eventual VMIShutdown event being dropped.
	if timeLeft == firstGracefulShutdownAttempt {
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.ShuttingDown.String(), VMIGracefulShutdown)
	}

	// Make sure that we don't hot-loop in case we send the first domain notification
	if timeLeft == firstGracefulShutdownAttempt {
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
	c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Duration(timeLeft)*time.Second)
	return nil
}

func (c *VirtualMachineController) processVmDelete(vmi *v1.VirtualMachineInstance) error {

	// Only attempt to shutdown/destroy if we still have a connection established with the pod.
	client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)

	// If the pod has been torn down, we know the VirtualMachineInstance is down.
	if err == nil {

		c.logger.Object(vmi).Infof("Signaled deletion for %s", vmi.GetObjectMeta().GetName())

		// pending deletion.
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Deleted.String(), VMISignalDeletion)

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

func (c *VirtualMachineController) isVMIOwnedByNode(vmi *v1.VirtualMachineInstance) bool {
	nodeName, ok := vmi.Labels[v1.NodeNameLabel]

	if ok && nodeName != "" && nodeName == c.host {
		return true
	}

	return vmi.Status.NodeName != "" && vmi.Status.NodeName == c.host
}

func (c *VirtualMachineController) checkNetworkInterfacesForMigration(vmi *v1.VirtualMachineInstance) error {
	return netvmispec.VerifyVMIMigratable(vmi, c.clusterConfig.GetNetworkBindings())
}

func isReadOnlyDisk(disk *v1.Disk) bool {
	isReadOnlyCDRom := disk.CDRom != nil && (disk.CDRom.ReadOnly == nil || *disk.CDRom.ReadOnly)

	return isReadOnlyCDRom
}

func (c *VirtualMachineController) checkVolumesForMigration(vmi *v1.VirtualMachineInstance) (blockMigrate bool, err error) {
	volumeStatusMap := make(map[string]v1.VolumeStatus)

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	if len(vmi.Status.MigratedVolumes) > 0 {
		blockMigrate = true
	}

	filesystems := storagetypes.GetFilesystemsFromVolumes(vmi)

	// Check if all VMI volumes can be shared between the source and the destination
	// of a live migration. blockMigrate will be returned as false, only if all volumes
	// are shared and the VMI has no local disks
	// Some combinations of disks makes the VMI no suitable for live migration.
	// A relevant error will be returned in this case.
	for _, volume := range vmi.Spec.Volumes {
		volSrc := volume.VolumeSource
		if volSrc.PersistentVolumeClaim != nil || volSrc.DataVolume != nil {
			var claimName string
			if volSrc.PersistentVolumeClaim != nil {
				claimName = volSrc.PersistentVolumeClaim.ClaimName
			} else {
				claimName = volSrc.DataVolume.Name
			}

			volumeStatus, ok := volumeStatusMap[volume.Name]

			if !ok || volumeStatus.PersistentVolumeClaimInfo == nil {
				return true, fmt.Errorf("cannot migrate VMI: Unable to determine if PVC %v is shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", claimName)
			} else if !storagetypes.HasSharedAccessMode(volumeStatus.PersistentVolumeClaimInfo.AccessModes) && !storagetypes.IsMigratedVolume(volumeStatus.Name, vmi) {
				return true, fmt.Errorf("cannot migrate VMI: PVC %v is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", claimName)
			}

		} else if volSrc.HostDisk != nil {
			// Check if this is a translated PVC.
			volumeStatus, ok := volumeStatusMap[volume.Name]
			if ok && volumeStatus.PersistentVolumeClaimInfo != nil {
				if !storagetypes.HasSharedAccessMode(volumeStatus.PersistentVolumeClaimInfo.AccessModes) && !storagetypes.IsMigratedVolume(volumeStatus.Name, vmi) {
					return true, fmt.Errorf("cannot migrate VMI: PVC %v is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)", volumeStatus.PersistentVolumeClaimInfo.ClaimName)
				} else {
					continue
				}
			}

			shared := volSrc.HostDisk.Shared != nil && *volSrc.HostDisk.Shared
			if !shared {
				return true, fmt.Errorf("cannot migrate VMI with non-shared HostDisk")
			}
		} else {
			if _, ok := filesystems[volume.Name]; ok {
				c.logger.Object(vmi).Infof("Volume %s is shared with virtiofs, allow live migration", volume.Name)
				continue
			}

			isVolumeUsedByReadOnlyDisk := false
			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				if isReadOnlyDisk(&disk) && disk.Name == volume.Name {
					isVolumeUsedByReadOnlyDisk = true
					break
				}
			}

			if isVolumeUsedByReadOnlyDisk {
				continue
			}

			if vmi.Status.MigrationMethod == "" || vmi.Status.MigrationMethod == v1.LiveMigration {
				c.logger.Object(vmi).Infof("migration is block migration because of %s volume", volume.Name)
			}
			blockMigrate = true
		}
	}
	return
}

func (c *VirtualMachineController) affinePitThread(vmi *v1.VirtualMachineInstance) error {
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	var Mask unix.CPUSet
	Mask.Zero()
	qemuprocess, err := res.GetQEMUProcess()
	if err != nil {
		return err
	}
	qemupid := qemuprocess.Pid()
	if qemupid == -1 {
		return nil
	}

	pitpid, err := res.KvmPitPid()
	if err != nil {
		return err
	}
	if pitpid == -1 {
		return nil
	}
	if vmi.IsRealtimeEnabled() {
		param := schedParam{priority: 2}
		err = schedSetScheduler(pitpid, schedFIFO, param)
		if err != nil {
			return fmt.Errorf("failed to set FIFO scheduling and priority 2 for thread %d: %w", pitpid, err)
		}
	}
	vcpus, err := getVCPUThreadIDs(qemupid)
	if err != nil {
		return err
	}
	vpid, ok := vcpus["0"]
	if ok == false {
		return nil
	}
	vcpupid, err := strconv.Atoi(vpid)
	if err != nil {
		return err
	}
	err = unix.SchedGetaffinity(vcpupid, &Mask)
	if err != nil {
		return err
	}
	return unix.SchedSetaffinity(pitpid, &Mask)
}

func (c *VirtualMachineController) configureHousekeepingCgroup(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager) error {
	if err := cgroupManager.CreateChildCgroup("housekeeping", "cpuset"); err != nil {
		c.logger.Reason(err).Error("CreateChildCgroup ")
		return err
	}

	key := controller.VirtualMachineInstanceKey(vmi)
	domain, domainExists, _, err := c.getDomainFromCache(key)
	if err != nil {
		return err
	}
	// bail out if domain does not exist
	if domainExists == false {
		return nil
	}

	if domain.Spec.CPUTune == nil || domain.Spec.CPUTune.EmulatorPin == nil {
		return nil
	}

	hkcpus, err := hardware.ParseCPUSetLine(domain.Spec.CPUTune.EmulatorPin.CPUSet, 100)
	if err != nil {
		return err
	}

	c.logger.V(3).Object(vmi).Infof("housekeeping cpu: %v", hkcpus)

	err = cgroupManager.SetCpuSet("housekeeping", hkcpus)
	if err != nil {
		return err
	}

	tids, err := cgroupManager.GetCgroupThreads()
	if err != nil {
		return err
	}
	hktids := make([]int, 0, 10)

	for _, tid := range tids {
		proc, err := ps.FindProcess(tid)
		if err != nil {
			c.logger.Object(vmi).Errorf("Failure to find process: %s", err.Error())
			return err
		}
		if proc == nil {
			return fmt.Errorf("failed to find process with tid: %d", tid)
		}
		comm := proc.Executable()
		if strings.Contains(comm, "CPU ") && strings.Contains(comm, "KVM") {
			continue
		}
		hktids = append(hktids, tid)
	}

	c.logger.V(3).Object(vmi).Infof("hk thread ids: %v", hktids)
	for _, tid := range hktids {
		err = cgroupManager.AttachTID("cpuset", "housekeeping", tid)
		if err != nil {
			c.logger.Object(vmi).Errorf("Error attaching tid %d: %v", tid, err.Error())
			return err
		}
	}

	return nil
}

func (c *VirtualMachineController) vmUpdateHelperDefault(vmi *v1.VirtualMachineInstance, domainExists bool) error {
	client, err := c.launcherClients.GetLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf(unableCreateVirtLauncherConnectionFmt, err)
	}

	preallocatedVolumes := c.getPreallocatedVolumes(vmi)

	err = hostdisk.ReplacePVCByHostDisk(vmi)
	if err != nil {
		return err
	}

	cgroupManager, err := getCgroupManager(vmi, c.host)
	if err != nil {
		return err
	}

	var errorTolerantFeaturesError []error
	readyToProceed, err := c.handleVMIState(vmi, cgroupManager, &errorTolerantFeaturesError)
	if err != nil {
		return err
	}

	if !readyToProceed {
		return nil
	}

	// Synchronize the VirtualMachineInstance state
	err = c.syncVirtualMachine(client, vmi, preallocatedVolumes)
	if err != nil {
		return err
	}

	// Post-sync housekeeping
	err = c.handleHousekeeping(vmi, cgroupManager, domainExists)
	if err != nil {
		return err
	}

	return errors.NewAggregate(errorTolerantFeaturesError)
}

// handleVMIState: Decides whether to call handleRunningVMI or handleStartingVMI based on the VMI's state.
func (c *VirtualMachineController) handleVMIState(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, errorTolerantFeaturesError *[]error) (bool, error) {
	if vmi.IsRunning() {
		return true, c.handleRunningVMI(vmi, cgroupManager, errorTolerantFeaturesError)
	} else if !vmi.IsFinal() {
		return c.handleStartingVMI(vmi, cgroupManager)
	}
	return true, nil
}

// handleRunningVMI contains the logic specifically for running VMs (hotplugging in running state, metrics, network updates)
func (c *VirtualMachineController) handleRunningVMI(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, errorTolerantFeaturesError *[]error) error {
	if err := c.hotplugSriovInterfaces(vmi); err != nil {
		c.logger.Object(vmi).Error(err.Error())
	}

	if err := c.hotplugVolumeMounter.Mount(vmi, cgroupManager); err != nil {
		if !goerror.Is(err, os.ErrNotExist) {
			return err
		}
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "HotplugFailed", err.Error())
	}

	if err := c.getMemoryDump(vmi); err != nil {
		return err
	}

	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf(failedDetectIsolationFmt, err)
	}

	if err := c.downwardMetricsManager.StartServer(vmi, isolationRes.Pid()); err != nil {
		return err
	}

	if err := c.setupNetwork(vmi, netsetup.FilterNetsForLiveUpdate(vmi), c.netConf); err != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "NicHotplug", err.Error())
		*errorTolerantFeaturesError = append(*errorTolerantFeaturesError, err)
	}

	return nil
}

// handleStartingVMI: Contains the logic for starting VMs (container disks, initial network setup, device ownership).
func (c *VirtualMachineController) handleStartingVMI(
	vmi *v1.VirtualMachineInstance,
	cgroupManager cgroup.Manager,
) (bool, error) {
	// give containerDisks some time to become ready before throwing errors on retries
	info := c.launcherClients.GetLauncherClientInfo(vmi)
	if ready, err := c.containerDiskMounter.ContainerDisksReady(vmi, info.NotInitializedSince); !ready {
		if err != nil {
			return false, err
		}
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return false, nil
	}

	var err error
	err = c.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return false, err
	}

	if err := c.hotplugVolumeMounter.Mount(vmi, cgroupManager); err != nil {
		if !goerror.Is(err, os.ErrNotExist) {
			return false, err
		}
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, "HotplugFailed", err.Error())
	}

	if !c.hotplugVolumesReady(vmi) {
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return false, nil
	}

	if c.clusterConfig.GPUsWithDRAGateEnabled() {
		if !drautil.IsAllDRAGPUsReconciled(vmi, vmi.Status.DeviceStatus) {
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, "WaitingForDRAGPUAttributes",
				"Waiting for Dynamic Resource Allocation GPU attributes to be reconciled")
			return false, nil
		}
	}

	if err := c.setupNetwork(vmi, netsetup.FilterNetsForVMStartup(vmi), c.netConf); err != nil {
		return false, fmt.Errorf("failed to configure vmi network: %w", err)
	}

	if err := c.setupDevicesOwnerships(vmi, c.recorder); err != nil {
		return false, err
	}

	if err := c.adjustResources(vmi); err != nil {
		return false, err
	}

	if c.shouldWaitForSEVAttestation(vmi) {
		return false, nil
	}

	return true, nil
}

func (c *VirtualMachineController) adjustResources(vmi *v1.VirtualMachineInstance) error {
	err := c.podIsolationDetector.AdjustResources(vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio)
	if err != nil {
		return fmt.Errorf("failed to adjust resources: %v", err)
	}
	return nil
}

func (c *VirtualMachineController) shouldWaitForSEVAttestation(vmi *v1.VirtualMachineInstance) bool {
	if util.IsSEVAttestationRequested(vmi) {
		sev := vmi.Spec.Domain.LaunchSecurity.SEV
		// Wait for the session parameters to be provided
		return sev.Session == "" || sev.DHCert == ""
	}
	return false
}

func (c *VirtualMachineController) syncVirtualMachine(client cmdclient.LauncherClient, vmi *v1.VirtualMachineInstance, preallocatedVolumes []string) error {
	smbios := c.clusterConfig.GetSMBIOS()
	period := c.clusterConfig.GetMemBalloonStatsPeriod()

	options := virtualMachineOptions(smbios, period, preallocatedVolumes, c.capabilities, c.clusterConfig)
	options.InterfaceDomainAttachment = domainspec.DomainAttachmentByInterfaceName(vmi.Spec.Domain.Devices.Interfaces, c.clusterConfig.GetNetworkBindings())

	err := client.SyncVirtualMachine(vmi, options)
	if err != nil {
		if strings.Contains(err.Error(), "EFI OVMF rom missing") {
			return &virtLauncherCriticalSecurebootError{fmt.Sprintf("mismatch of Secure Boot setting and bootloaders: %v", err)}
		}
	}

	return err
}

func (c *VirtualMachineController) handleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domainExists bool) error {
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		err := c.configureHousekeepingCgroup(vmi, cgroupManager)
		if err != nil {
			return err
		}
	}

	// Configure vcpu scheduler for realtime workloads and affine PIT thread for dedicated CPU
	if vmi.IsRealtimeEnabled() && !vmi.IsRunning() && !vmi.IsFinal() {
		c.logger.Object(vmi).Info("Configuring vcpus for real time workloads")
		if err := c.configureVCPUScheduler(vmi); err != nil {
			return err
		}
	}
	if vmi.IsCPUDedicated() && !vmi.IsRunning() && !vmi.IsFinal() {
		c.logger.V(3).Object(vmi).Info("Affining PIT thread")
		if err := c.affinePitThread(vmi); err != nil {
			return err
		}
	}
	if !domainExists {
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Created.String(), VMIDefined)
	}

	if vmi.IsRunning() {
		// Umount any disks no longer mounted
		if err := c.hotplugVolumeMounter.Unmount(vmi, cgroupManager); err != nil {
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) getPreallocatedVolumes(vmi *v1.VirtualMachineInstance) []string {
	var preallocatedVolumes []string
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.PersistentVolumeClaimInfo != nil && volumeStatus.PersistentVolumeClaimInfo.Preallocated {
			preallocatedVolumes = append(preallocatedVolumes, volumeStatus.Name)
		}
	}
	return preallocatedVolumes
}

func (c *VirtualMachineController) hotplugSriovInterfaces(vmi *v1.VirtualMachineInstance) error {
	sriovSpecInterfaces := netvmispec.FilterSRIOVInterfaces(vmi.Spec.Domain.Devices.Interfaces)

	sriovSpecIfacesNames := netvmispec.IndexInterfaceSpecByName(sriovSpecInterfaces)
	attachedSriovStatusIfaces := netvmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, func(iface v1.VirtualMachineInstanceNetworkInterface) bool {
		_, exist := sriovSpecIfacesNames[iface.Name]
		return exist && netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceDomain) &&
			netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceMultusStatus)
	})

	desiredSriovMultusPluggedIfaces := netvmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, func(iface v1.VirtualMachineInstanceNetworkInterface) bool {
		_, exist := sriovSpecIfacesNames[iface.Name]
		return exist && netvmispec.ContainsInfoSource(iface.InfoSource, netvmispec.InfoSourceMultusStatus)
	})

	if len(desiredSriovMultusPluggedIfaces) == len(attachedSriovStatusIfaces) {
		c.sriovHotplugExecutorPool.Delete(vmi.UID)
		return nil
	}

	rateLimitedExecutor := c.sriovHotplugExecutorPool.LoadOrStore(vmi.UID)
	return rateLimitedExecutor.Exec(func() error {
		return c.hotplugSriovInterfacesCommand(vmi)
	})
}

func (c *VirtualMachineController) hotplugSriovInterfacesCommand(vmi *v1.VirtualMachineInstance) error {
	const errMsgPrefix = "failed to hot-plug SR-IOV interfaces"

	client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := isolation.AdjustQemuProcessMemoryLimits(c.podIsolationDetector, vmi, c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio); err != nil {
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, err.Error(), err.Error())
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	c.logger.V(3).Object(vmi).Info("sending hot-plug host-devices command")
	if err := client.HotplugHostDevices(vmi); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return nil
}

func memoryDumpPath(volumeStatus v1.VolumeStatus) string {
	target := hotplugdisk.GetVolumeMountDir(volumeStatus.Name)
	dumpPath := filepath.Join(target, volumeStatus.MemoryDumpVolume.TargetFileName)
	return dumpPath
}

func (c *VirtualMachineController) getMemoryDump(vmi *v1.VirtualMachineInstance) error {
	const errMsgPrefix = "failed to getting memory dump"

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.MemoryDumpVolume == nil || volumeStatus.Phase != v1.MemoryDumpVolumeInProgress {
			continue
		}
		client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}

		c.logger.V(3).Object(vmi).Info("sending memory dump command")
		err = client.VirtualMachineMemoryDump(vmi, memoryDumpPath(volumeStatus))
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}
	}

	return nil
}

func (c *VirtualMachineController) hotplugVolumesReady(vmi *v1.VirtualMachineInstance) bool {
	hasHotplugVolume := false
	for _, v := range vmi.Spec.Volumes {
		if storagetypes.IsHotplugVolume(&v) {
			hasHotplugVolume = true
			break
		}
	}
	if len(vmi.Spec.UtilityVolumes) > 0 {
		hasHotplugVolume = true
	}
	if !hasHotplugVolume {
		return true
	}
	if len(vmi.Status.VolumeStatus) == 0 {
		return false
	}
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.HotplugVolume != nil && !(vs.Phase == v1.VolumeReady || vs.Phase == v1.HotplugVolumeMounted) {
			// wait for volume to be mounted
			return false
		}
	}
	return true
}

func (c *VirtualMachineController) processVmUpdate(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	shouldReturn, err := c.checkLauncherClient(vmi)
	if shouldReturn {
		return err
	}

	return c.vmUpdateHelperDefault(vmi, domain != nil)
}

func (c *VirtualMachineController) addDeleteFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.vmiExpectations.SetExpectations(key, 0, 0)
		c.queue.Add(key)
	}
}

func (c *VirtualMachineController) updateFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.vmiExpectations.SetExpectations(key, 0, 0)
		c.queue.Add(key)
	}
}

func (c *VirtualMachineController) addDomainFunc(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) deleteDomainFunc(obj interface{}) {
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
	c.logger.V(3).Object(domain).Info("Domain deleted")
	key, err := controller.KeyFunc(obj)
	if err == nil {
		c.queue.Add(key)
	}
}
func (c *VirtualMachineController) updateDomainFunc(_, new interface{}) {
	key, err := controller.KeyFunc(new)
	if err == nil {
		c.queue.Add(key)
	}
}

func isIOError(shouldUpdate, domainExists bool, domain *api.Domain) bool {
	return shouldUpdate && domainExists && domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedIOError
}
