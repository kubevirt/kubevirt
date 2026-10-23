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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/diskimg"
	hostdisk "kubevirt.io/kubevirt/pkg/storage/host-disk"
	"kubevirt.io/kubevirt/pkg/storage/virtiofs"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	launcherclients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const (
	failedDetectIsolationFmt              = "failed to detect isolation for launcher pod: %v"
	unableCreateVirtLauncherConnectionFmt = "unable to create virt-launcher client connection: %v"
	// This value was determined after consulting with libvirt developers and performing extensive testing.
	parallelMultifdMigrationThreads = uint(8)
)

const (
	//VolumeReadyReason is the reason set when the volume is ready.
	VolumeReadyReason = "VolumeReady"
	//VolumeUnMountedFromPodReason is the reason set when the volume is unmounted from the virtlauncher pod
	VolumeUnMountedFromPodReason = "VolumeUnMountedFromPod"
	//VolumeMountedToPodReason is the reason set when the volume is mounted to the virtlauncher pod
	VolumeMountedToPodReason = "VolumeMountedToPod"
	//VolumeUnplugged is the reason set when the volume is completely unplugged from the VMI
	VolumeUnplugged = "VolumeUnplugged"
	//VMIDefined is the reason set when a VMI is defined
	VMIDefined = "VirtualMachineInstance defined."
	//VMIStarted is the reason set when a VMI is started
	VMIStarted = "VirtualMachineInstance started."
	//VMIShutdown is the reason set when a VMI is shutdown
	VMIShutdown = "The VirtualMachineInstance was shut down."
	//VMICrashed is the reason set when a VMI crashed
	VMICrashed = "The VirtualMachineInstance crashed."
	//VMIAbortingMigration is the reason set when migration is being aborted
	VMIAbortingMigration = "VirtualMachineInstance is aborting migration."
	//VMIMigrating in the reason set when the VMI is migrating
	VMIMigrating = "VirtualMachineInstance is migrating."
	//VMIMigrationTargetPrepared is the reason set when the migration target has been prepared
	VMIMigrationTargetPrepared = "VirtualMachineInstance Migration Target Prepared."
	//VMIStopping is the reason set when the VMI is stopping
	VMIStopping = "VirtualMachineInstance stopping"
	//VMIGracefulShutdown is the reason set when the VMI is gracefully shut down
	VMIGracefulShutdown = "Signaled Graceful Shutdown"
	//VMISignalDeletion is the reason set when the VMI has signal deletion
	VMISignalDeletion = "Signaled Deletion"

	// MemoryHotplugFailedReason is the reason set when the VM cannot hotplug memory
	memoryHotplugFailedReason = "Memory Hotplug Failed"
)

// role identifies the single virt-handler controller that owns a VMI key on this
// node for one pass of the Arbiter.
type role int

const (
	// roleNone means no controller may act on the key in this pass.
	roleNone role = iota
	roleVM
	roleSource
	roleTarget
)

func (r role) String() string {
	switch r {
	case roleVM:
		return "vm"
	case roleSource:
		return "migration-source"
	case roleTarget:
		return "migration-target"
	default:
		return "none"
	}
}

// migrationRole is the single source of truth about which controller owns a VMI.
//
// The VM lifecycle, migration-source and migration-target controllers must never
// act on the same VMI concurrently. Expressing that as three independent "should I
// skip?" predicates, each evaluated against its own informer cache, cannot provide
// that guarantee: the predicates are not complements of each other, and they are
// not even evaluated against the same snapshot. This function replaces them with a
// single decision, taken once per pass, over one snapshot.
//
// The source and the target of a migration are never the same node for a given
// VMI, so migrationTargetOwns and migrationSourceOwns are mutually exclusive by
// construction - virt-controller writes the target node label and
// MigrationState.SourceNode in the same update, and they never name this host at
// once. Their order below therefore only decides precedence against the VM
// controller, not against each other.
//
// Hand-over points are chosen so that the controller giving up ownership is the one
// that publishes the release, rather than both sides racing on a timestamp written
// by somebody else.
func migrationRole(vmi *v1.VirtualMachineInstance, vmiExists bool, domain *api.Domain, host string) role {
	// No VMI left: all that remains is local cleanup of a leftover domain and its
	// ghost record, which only the VM controller does.
	if !vmiExists || vmi == nil {
		return roleVM
	}

	if migrationTargetOwns(vmi, host) {
		return roleTarget
	}

	if migrationSourceOwns(vmi, host) {
		return roleSource
	}

	// A VMI that is going away always goes to the VM controller: local teardown and
	// releasing the ghost record are nobody else's job. This must come before the
	// in-progress check below, because a VMI that dies mid-migration leaves a
	// migration state that never reaches an end timestamp, and standing everybody
	// down there would leak the domain and the ghost record.
	if vmi.DeletionTimestamp != nil || vmi.IsFinal() {
		return roleVM
	}

	// Last line of defence: the cached VMI may not reflect a migration that the
	// domain already knows about. Nobody acts until the VMI catches up - the key is
	// re-queued by the next VMI or domain event.
	if isMigrationInProgress(vmi, domain) {
		return roleNone
	}

	return roleVM
}

// migrationTargetOwns reports whether the migration target controller owns vmi on
// this node.
//
// Ownership starts when virt-controller labels the VMI with the node its target pod
// was scheduled on, and ends when the target controller itself drops that label,
// which it does as the very last step of finalCleanup. Because the owner publishes
// its own release, the VM controller cannot take over while the target is still
// finalizing - the window that EndTimestamp-based predicates leave wide open.
func migrationTargetOwns(vmi *v1.VirtualMachineInstance, host string) bool {
	if vmi.Labels[v1.MigrationTargetNodeNameLabel] != host {
		return false
	}

	// Once ackMigrationCompletion has moved the VMI onto this node, a terminating VMI
	// is an ordinary local VMI: the VM controller owns the teardown, and its cleanup
	// is a superset of the target's best-effort one.
	if (vmi.DeletionTimestamp != nil || vmi.IsFinal()) && vmiOwnedByNode(vmi, host) {
		return false
	}

	return true
}

// migrationSourceOwns reports whether the migration source controller owns vmi on
// this node. It is deliberately the exact condition under which the source
// controller has work to do, so that the VM controller's own window is its
// complement rather than a second, independently drifting predicate.
func migrationSourceOwns(vmi *v1.VirtualMachineInstance, host string) bool {
	migrationState := vmi.Status.MigrationState
	if migrationState == nil || migrationState.SourceNode != host {
		return false
	}

	// A VMI that is going away is shut down and cleaned up by the VM controller, even
	// mid-migration. The source controller has nothing to contribute there.
	if vmi.DeletionTimestamp != nil {
		return false
	}

	// A terminal VMI is torn down locally by the VM controller, which is also the only
	// one that releases the ghost record. A decentralized source is the one exception,
	// and only for as long as it takes to record on the migration state a failure that
	// something else caused: nothing on this side of a decentralized migration would
	// otherwise do it, and that single update always makes progress, so ownership moves
	// on right after.
	if vmi.IsFinal() {
		return vmi.IsDecentralizedMigration() &&
			vmi.IsMigrationSource() &&
			vmi.Status.Phase == v1.Failed &&
			!isMigrationDone(migrationState)
	}

	if vmi.IsDecentralizedMigration() {
		// Before the two sides have synchronized there is nothing for the source
		// controller to do and the guest is simply running here.
		return vmi.IsMigrationSource()
	}

	// Once the migration is done the source domain is shut off with reason "migrated"
	// and the VM controller reaps it through its orphan path.
	return vmi.Status.NodeName == host && !isMigrationDone(migrationState)
}

func vmiOwnedByNode(vmi *v1.VirtualMachineInstance, host string) bool {
	nodeName, ok := vmi.Labels[v1.NodeNameLabel]

	if ok && nodeName != "" && nodeName == host {
		return true
	}

	return vmi.Status.NodeName != "" && vmi.Status.NodeName == host
}

type netconf interface {
	Setup(vmi *v1.VirtualMachineInstance, networks []v1.Network, launcherPid int) error
	Teardown(vmi *v1.VirtualMachineInstance) error
}

type BaseController struct {
	logger                      *log.FilteredLogger
	host                        string
	virtClient                  kubecli.KubevirtClient
	queue                       workqueue.TypedRateLimitingInterface[string]
	vmiStore                    cache.Store
	domainStore                 cache.Store
	clusterConfig               *virtconfig.ClusterConfig
	podIsolationDetector        isolation.PodIsolationDetector
	launcherClients             launcherclients.LauncherClientsManager
	virtLauncherFSRunDirPattern string
	netStat                     netstat
	recorder                    record.EventRecorder
	hasSynced                   func() bool
	// vmiExpectations is shared by all three controllers through the Arbiter. A VMI
	// update written by one of them has to be observed before any of them processes
	// the key again, otherwise the next owner would act on a snapshot it already
	// invalidated.
	vmiExpectations    *controller.UIDTrackingControllerExpectations
	hypervisorNodeInfo hypervisor.HypervisorNodeInformation
	hypervisorRuntime  hypervisor.VirtRuntime
	pluginStore        cache.Store
	pluginCache        pluginCache
}

func NewBaseController(
	logger *log.FilteredLogger,
	host string,
	recorder record.EventRecorder,
	virtClient kubecli.KubevirtClient,
	queue workqueue.TypedRateLimitingInterface[string],
	vmiExpectations *controller.UIDTrackingControllerExpectations,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
	launcherClients launcherclients.LauncherClientsManager,
	virtLauncherFSRunDirPattern string,
	netStat netstat,
	hypervisorNodeInfo hypervisor.HypervisorNodeInformation,
	hypervisorRuntime hypervisor.VirtRuntime,
	pluginStore cache.Store,
) (*BaseController, error) {

	c := &BaseController{
		logger:                      logger,
		host:                        host,
		recorder:                    recorder,
		virtClient:                  virtClient,
		queue:                       queue,
		vmiExpectations:             vmiExpectations,
		vmiStore:                    vmiInformer.GetStore(),
		domainStore:                 domainInformer.GetStore(),
		clusterConfig:               clusterConfig,
		podIsolationDetector:        podIsolationDetector,
		launcherClients:             launcherClients,
		virtLauncherFSRunDirPattern: virtLauncherFSRunDirPattern,
		netStat:                     netStat,
		hasSynced:                   func() bool { return domainInformer.HasSynced() && vmiInformer.HasSynced() },
		hypervisorNodeInfo:          hypervisorNodeInfo,
		hypervisorRuntime:           hypervisorRuntime,
		pluginStore:                 pluginStore,
	}

	return c, nil
}

type pluginCache struct {
	mu   sync.Mutex
	data []byte
}

func (c *BaseController) serializePlugins() ([]byte, error) {
	c.pluginCache.mu.Lock()
	defer c.pluginCache.mu.Unlock()

	if c.pluginCache.data != nil {
		return c.pluginCache.data, nil
	}

	data, err := c.buildPluginsJSON()
	if err != nil {
		return nil, err
	}

	c.pluginCache.data = data
	return data, nil
}

func (c *BaseController) buildPluginsJSON() ([]byte, error) {
	if c.pluginStore == nil || !c.clusterConfig.PluginsEnabled() {
		return nil, nil
	}
	objs := c.pluginStore.List()
	if len(objs) == 0 {
		return nil, nil
	}
	var pluginList []pluginv1alpha1.Plugin
	for _, obj := range objs {
		if p, ok := obj.(*pluginv1alpha1.Plugin); ok {
			pluginList = append(pluginList, *p)
		}
	}
	if len(pluginList) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(pluginList)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize plugins: %w", err)
	}
	return data, nil
}

func (c *BaseController) InvalidatePluginsCache() {
	c.pluginCache.mu.Lock()
	defer c.pluginCache.mu.Unlock()

	c.pluginCache.data = nil
}

func (c *BaseController) getVMIFromCache(key string) (vmi *v1.VirtualMachineInstance, exists bool, err error) {
	obj, exists, err := c.vmiStore.GetByKey(key)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			// Invalid keys will be retried forever, but the queue has no reason to contain any
			return nil, false, err
		}
		vmi = libvmi.New(libvmi.WithName(name), libvmi.WithNamespace(namespace))
	} else {
		vmi = obj.(*v1.VirtualMachineInstance)
	}
	return vmi, exists, nil
}

func (c *BaseController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceNode == c.host &&
		(!vmi.IsDecentralizedMigration() || vmi.IsMigrationSource()) &&
		vmi.Status.MigrationState.TargetNodeAddress != "" &&
		!vmi.Status.MigrationState.Completed
}

func (c *BaseController) claimDeviceOwnership(virtLauncherRootMount *safepath.Path, deviceName string) error {
	softwareEmulation := c.clusterConfig.AllowEmulation()
	devicePath, err := safepath.JoinNoFollow(virtLauncherRootMount, filepath.Join("dev", deviceName))
	if err != nil {
		if softwareEmulation && deviceName == c.hypervisorNodeInfo.GetHypervisorDevice() {
			return nil
		}
		return err
	}

	return diskutils.DefaultOwnershipManager.SetFileOwnership(devicePath)
}

func (c *BaseController) configureDiskImages(
	vmi *v1.VirtualMachineInstance,
	virtLauncherRootMount *safepath.Path,
	recorder record.EventRecorder) error {
	lessPVCSpaceToleration := c.clusterConfig.GetLessPVCSpaceToleration()
	minimumPVCReserveBytes := c.clusterConfig.GetMinimumReservePVCBytes()

	pvcImages, err := diskimg.ImageBackedPVCImages(vmi)
	if err != nil {
		return err
	}
	creator := diskimg.NewCreator(recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
	if err := creator.Create(vmi, append(hostdisk.Images(vmi), pvcImages...)); err != nil {
		return fmt.Errorf("preparing disk images failed: %w", err)
	}
	return nil
}

func (c *BaseController) configureSEVDeviceOwnership(vmi *v1.VirtualMachineInstance, virtLauncherRootMount *safepath.Path) error {
	if util.IsSEVVMI(vmi) {
		sevDevice, err := safepath.JoinNoFollow(virtLauncherRootMount, filepath.Join("dev", "sev"))
		if err != nil {
			return err
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(sevDevice); err != nil {
			return fmt.Errorf("failed to set SEV device owner: %v", err)
		}
	}
	return nil
}

func (c *BaseController) configureVirtioFS(vmi *v1.VirtualMachineInstance, isolationRes isolation.IsolationResult) error {
	for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
		socketPath, err := isolation.SafeJoin(isolationRes, virtiofs.VirtioFSSocketPath(fs.Name))
		if err != nil {
			return err
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath); err != nil {
			return err
		}
	}
	return nil
}

func (c *BaseController) setupDevicesOwnerships(vmi *v1.VirtualMachineInstance, recorder record.EventRecorder) error {
	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf(failedDetectIsolationFmt, err)
	}

	virtLauncherRootMount, err := isolationRes.MountRoot()
	if err != nil {
		return err
	}

	hypervisorDevice := c.hypervisorNodeInfo.GetHypervisorDevice()
	if err := c.claimDeviceOwnership(virtLauncherRootMount, hypervisorDevice); err != nil {
		return fmt.Errorf("failed to set up file ownership for /dev/%s: %v", hypervisorDevice, err)
	}

	if vmitrait.IsAutoAttachVSOCK(vmi) {
		if err := c.claimDeviceOwnership(virtLauncherRootMount, "vhost-vsock"); err != nil {
			return fmt.Errorf("failed to set up file ownership for /dev/vhost-vsock: %v", err)
		}
	}

	if err := c.configureDiskImages(vmi, virtLauncherRootMount, recorder); err != nil {
		return err
	}

	if err := c.configureSEVDeviceOwnership(vmi, virtLauncherRootMount); err != nil {
		return err
	}

	if vmitrait.IsNonRoot(vmi) {
		if err := c.nonRootSetup(vmi); err != nil {
			return err
		}
	}

	if err := c.configureVirtioFS(vmi, isolationRes); err != nil {
		return err
	}

	return nil
}

func (c *BaseController) setupNetwork(vmi *v1.VirtualMachineInstance, networks []v1.Network, netConf netconf) error {
	if len(networks) == 0 {
		return nil
	}

	isolationRes, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf(failedDetectIsolationFmt, err)
	}

	return netConf.Setup(vmi, networks, isolationRes.Pid())
}

func isMigrationInProgress(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	var domainMigrationMetadata *api.MigrationMetadata
	if vmi != nil &&
		vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.StartTimestamp != nil &&
		vmi.Status.MigrationState.EndTimestamp == nil {
		return true
	}

	if domain != nil {
		domainMigrationMetadata = domain.Spec.Metadata.KubeVirt.Migration

		if domainMigrationMetadata != nil &&
			domainMigrationMetadata.StartTimestamp != nil &&
			domainMigrationMetadata.EndTimestamp == nil {
			return true
		}

		if domain.Status.Status == api.Paused &&
			(domain.Status.Reason == api.ReasonPausedMigration ||
				domain.Status.Reason == api.ReasonPausedStartingUp ||
				domain.Status.Reason == api.ReasonPausedPostcopy) {
			return true
		}
	}

	if vmi != nil && vmi.IsMigrationTarget() {
		return vmi.IsMigrationTarget() && !vmi.IsMigrationCompleted()
	}
	return false
}

func (c *BaseController) passtSocketDirOnHostForVMI(vmi *v1.VirtualMachineInstance) (string, error) {
	path, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return "", err
	}
	return passtSocketDirOnHost(path)
}

func (c *BaseController) checkLauncherClient(vmi *v1.VirtualMachineInstance) (bool, error) {
	isUnresponsive, isInitialized, err := c.launcherClients.IsLauncherClientUnresponsive(vmi)
	if err != nil {
		return true, err
	}
	if !isInitialized {
		c.logger.Object(vmi).V(4).Info("launcher client is not initialized")
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
		return true, nil
	} else if isUnresponsive {
		return true, errors.New(fmt.Sprintf("Can not update a VirtualMachineInstance with unresponsive command server."))
	}

	return false, nil
}
