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
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virtiofs"
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

type netconf interface {
	Setup(vmi *v1.VirtualMachineInstance, networks []v1.Network, launcherPid int) error
	Teardown(vmi *v1.VirtualMachineInstance) error
}

type BaseController struct {
	host                 string
	vmiStore             cache.Store
	domainStore          cache.Store
	clusterConfig        *virtconfig.ClusterConfig
	podIsolationDetector isolation.PodIsolationDetector
	hasSynced            func() bool
}

func NewBaseController(
	host string,
	vmiInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
	clusterConfig *virtconfig.ClusterConfig,
	podIsolationDetector isolation.PodIsolationDetector,
) (*BaseController, error) {

	c := &BaseController{
		host:                 host,
		vmiStore:             vmiInformer.GetStore(),
		domainStore:          domainInformer.GetStore(),
		clusterConfig:        clusterConfig,
		podIsolationDetector: podIsolationDetector,
		hasSynced:            func() bool { return domainInformer.HasSynced() && vmiInformer.HasSynced() },
	}

	return c, nil
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
		vmi = v1.NewVMIReferenceFromNameWithNS(namespace, name)
	} else {
		vmi = obj.(*v1.VirtualMachineInstance)
	}
	return vmi, exists, nil
}

func (c *BaseController) getDomainFromCache(key string) (domain *api.Domain, exists bool, cachedUID types.UID, err error) {

	obj, exists, err := c.domainStore.GetByKey(key)

	if err != nil {
		return nil, false, "", err
	}

	if exists {
		domain = obj.(*api.Domain)
		cachedUID = domain.Spec.Metadata.KubeVirt.UID

		// We're using the DeletionTimestamp to signify that the
		// Domain is deleted rather than sending the DELETE watch event.
		if domain.ObjectMeta.DeletionTimestamp != nil {
			exists = false
			domain = nil
		}
	}
	return domain, exists, cachedUID, nil
}

func (c *BaseController) isMigrationSource(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.SourceNode == c.host &&
		vmi.IsMigrationSource() &&
		vmi.Status.MigrationState.TargetNodeAddress != "" &&
		!vmi.Status.MigrationState.Completed {
}

func (c *BaseController) claimDeviceOwnership(virtLauncherRootMount *safepath.Path, deviceName string) error {
	softwareEmulation := c.clusterConfig.AllowEmulation()
	devicePath, err := safepath.JoinNoFollow(virtLauncherRootMount, filepath.Join("dev", deviceName))
	if err != nil {
		if softwareEmulation && deviceName == "kvm" {
			return nil
		}
		return err
	}

	return diskutils.DefaultOwnershipManager.SetFileOwnership(devicePath)
}

func (c *BaseController) configureHostDisks(
	vmi *v1.VirtualMachineInstance,
	virtLauncherRootMount *safepath.Path,
	recorder record.EventRecorder) error {
	lessPVCSpaceToleration := c.clusterConfig.GetLessPVCSpaceToleration()
	minimumPVCReserveBytes := c.clusterConfig.GetMinimumReservePVCBytes()

	hostDiskCreator := hostdisk.NewHostDiskCreator(recorder, lessPVCSpaceToleration, minimumPVCReserveBytes, virtLauncherRootMount)
	if err := hostDiskCreator.Create(vmi); err != nil {
		return fmt.Errorf("preparing host-disks failed: %v", err)
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

	err = c.claimDeviceOwnership(virtLauncherRootMount, "kvm")
	if err != nil {
		return fmt.Errorf("failed to set up file ownership for /dev/kvm: %v", err)
	}

	if util.IsAutoAttachVSOCK(vmi) {
		if err := c.claimDeviceOwnership(virtLauncherRootMount, "vhost-vsock"); err != nil {
			return fmt.Errorf("failed to set up file ownership for /dev/vhost-vsock: %v", err)
		}
	}

	if err := c.configureHostDisks(vmi, virtLauncherRootMount, recorder); err != nil {
		return err
	}

	if err := c.configureSEVDeviceOwnership(vmi, virtLauncherRootMount); err != nil {
		return err
	}

	if util.IsNonRootVMI(vmi) {
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

	return false
}
