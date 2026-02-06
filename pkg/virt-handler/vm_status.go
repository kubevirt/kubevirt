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
	"bytes"
	"context"
	"encoding/json"
	goerror "errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/controller"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	containerdisk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// Volume status helper functions

func domainPausedFailedPostCopy(domain *api.Domain) bool {
	return domain != nil && domain.Status.Status == api.Paused && domain.Status.Reason == api.ReasonPausedPostcopyFailed
}

func canUpdateToMounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeBound || currentPhase == v1.VolumePending || currentPhase == v1.HotplugVolumeAttachedToNode
}

func canUpdateToUnmounted(currentPhase v1.VolumePhase) bool {
	return currentPhase == v1.VolumeReady || currentPhase == v1.HotplugVolumeMounted || currentPhase == v1.HotplugVolumeAttachedToNode
}

func isVMIPausedDuringMigration(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.MigrationState != nil &&
		vmi.Status.MigrationState.Mode == v1.MigrationPaused &&
		!vmi.Status.MigrationState.Completed
}

// Status update methods

func (c *VirtualMachineController) generateEventsForVolumeStatusChange(vmi *v1.VirtualMachineInstance, newStatusMap map[string]v1.VolumeStatus) {
	newStatusMapCopy := make(map[string]v1.VolumeStatus)
	for k, v := range newStatusMap {
		newStatusMapCopy[k] = v
	}
	for _, oldStatus := range vmi.Status.VolumeStatus {
		newStatus, ok := newStatusMap[oldStatus.Name]
		if !ok {
			// status got removed
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, VolumeUnplugged, fmt.Sprintf("Volume %s has been unplugged", oldStatus.Name))
			continue
		}
		if newStatus.Phase != oldStatus.Phase {
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, newStatus.Reason, newStatus.Message)
		}
		delete(newStatusMapCopy, newStatus.Name)
	}
	// Send events for any new statuses.
	for _, v := range newStatusMapCopy {
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v.Reason, v.Message)
	}
}

func (c *VirtualMachineController) updateHotplugVolumeStatus(vmi *v1.VirtualMachineInstance, volumeStatus v1.VolumeStatus, specVolumeMap map[string]struct{}) (v1.VolumeStatus, bool) {
	needsRefresh := false
	if volumeStatus.Target == "" {
		needsRefresh = true
		mounted, err := c.hotplugVolumeMounter.IsMounted(vmi, volumeStatus.Name, volumeStatus.HotplugVolume.AttachPodUID)
		if err != nil {
			c.logger.Object(vmi).Errorf("error occurred while checking if volume is mounted: %v", err)
		}
		if mounted {
			if _, ok := specVolumeMap[volumeStatus.Name]; ok && canUpdateToMounted(volumeStatus.Phase) {
				log.DefaultLogger().Infof("Marking volume %s as mounted in pod, it can now be attached", volumeStatus.Name)
				// mounted, and still in spec, and in phase we can change, update status to mounted.
				volumeStatus.Phase = v1.HotplugVolumeMounted
				volumeStatus.Message = fmt.Sprintf("Volume %s has been mounted in virt-launcher pod", volumeStatus.Name)
				volumeStatus.Reason = VolumeMountedToPodReason
			}
		} else {
			// Not mounted, check if the volume is in the spec, if not update status
			if _, ok := specVolumeMap[volumeStatus.Name]; !ok && canUpdateToUnmounted(volumeStatus.Phase) {
				log.DefaultLogger().Infof("Marking volume %s as unmounted from pod, it can now be detached", volumeStatus.Name)
				// Not mounted.
				volumeStatus.Phase = v1.HotplugVolumeUnMounted
				volumeStatus.Message = fmt.Sprintf("Volume %s has been unmounted from virt-launcher pod", volumeStatus.Name)
				volumeStatus.Reason = VolumeUnMountedFromPodReason
			}
		}
	} else {
		// Successfully attached to VM.
		volumeStatus.Phase = v1.VolumeReady
		volumeStatus.Message = fmt.Sprintf("Successfully attach hotplugged volume %s to VM", volumeStatus.Name)
		volumeStatus.Reason = VolumeReadyReason
	}
	return volumeStatus, needsRefresh
}

func needToComputeChecksums(vmi *v1.VirtualMachineInstance) bool {
	containerDisks := map[string]*v1.Volume{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk != nil {
			containerDisks[volume.Name] = &volume
		}
	}

	for i := range vmi.Status.VolumeStatus {
		_, isContainerDisk := containerDisks[vmi.Status.VolumeStatus[i].Name]
		if !isContainerDisk {
			continue
		}

		if vmi.Status.VolumeStatus[i].ContainerDiskVolume == nil ||
			vmi.Status.VolumeStatus[i].ContainerDiskVolume.Checksum == 0 {
			return true
		}
	}

	if util.HasKernelBootContainerImage(vmi) {
		if vmi.Status.KernelBootStatus == nil {
			return true
		}

		kernelBootContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container

		if kernelBootContainer.KernelPath != "" &&
			(vmi.Status.KernelBootStatus.KernelInfo == nil ||
				vmi.Status.KernelBootStatus.KernelInfo.Checksum == 0) {
			return true

		}

		if kernelBootContainer.InitrdPath != "" &&
			(vmi.Status.KernelBootStatus.InitrdInfo == nil ||
				vmi.Status.KernelBootStatus.InitrdInfo.Checksum == 0) {
			return true

		}
	}

	return false
}

// updateChecksumInfo is kept for compatibility with older virt-handlers
// that validate checksum calculations in vmi.status. This validation was
// removed in PR #14021, but we had to keep the checksum calculations for upgrades.
// Once we're sure old handlers won't interrupt upgrades, this can be removed.
func (c *VirtualMachineController) updateChecksumInfo(vmi *v1.VirtualMachineInstance, syncError error) error {
	// If the imageVolume feature gate is enabled, upgrade support isn't required,
	// and we can skip the checksum calculation. By the time the feature gate is GA,
	// the checksum calculation should be removed.
	if syncError != nil || vmi.DeletionTimestamp != nil || !needToComputeChecksums(vmi) || c.clusterConfig.ImageVolumeEnabled() {
		return nil
	}

	diskChecksums, err := c.containerDiskMounter.ComputeChecksums(vmi)
	if goerror.Is(err, containerdisk.ErrDiskContainerGone) {
		c.logger.Errorf("cannot compute checksums as containerdisk/kernelboot containers seem to have been terminated")
		return nil
	}
	if err != nil {
		return err
	}

	// containerdisks
	for i := range vmi.Status.VolumeStatus {
		checksum, exists := diskChecksums.ContainerDiskChecksums[vmi.Status.VolumeStatus[i].Name]
		if !exists {
			// not a containerdisk
			continue
		}

		vmi.Status.VolumeStatus[i].ContainerDiskVolume = &v1.ContainerDiskInfo{
			Checksum: checksum,
		}
	}

	// kernelboot
	if util.HasKernelBootContainerImage(vmi) {
		vmi.Status.KernelBootStatus = &v1.KernelBootStatus{}

		if diskChecksums.KernelBootChecksum.Kernel != nil {
			vmi.Status.KernelBootStatus.KernelInfo = &v1.KernelInfo{
				Checksum: *diskChecksums.KernelBootChecksum.Kernel,
			}
		}

		if diskChecksums.KernelBootChecksum.Initrd != nil {
			vmi.Status.KernelBootStatus.InitrdInfo = &v1.InitrdInfo{
				Checksum: *diskChecksums.KernelBootChecksum.Initrd,
			}
		}
	}

	return nil
}

func (c *VirtualMachineController) updateVolumeStatusesFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	// The return value is only used by unit tests
	hasHotplug := false

	if len(vmi.Status.VolumeStatus) == 0 {
		return false
	}

	diskDeviceMap := make(map[string]string)
	if domain != nil {
		for _, disk := range domain.Spec.Devices.Disks {
			// don't care about empty cdroms
			if disk.Source.File != "" || disk.Source.Dev != "" {
				diskDeviceMap[disk.Alias.GetName()] = disk.Target.Device
			}
		}
	}
	specVolumeMap := make(map[string]struct{})
	for _, volume := range vmi.Spec.Volumes {
		specVolumeMap[volume.Name] = struct{}{}
	}
	for _, utilityVolume := range vmi.Spec.UtilityVolumes {
		specVolumeMap[utilityVolume.Name] = struct{}{}
	}
	newStatusMap := make(map[string]v1.VolumeStatus)
	var newStatuses []v1.VolumeStatus
	needsRefresh := false
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		tmpNeedsRefresh := false
		// relying on the fact that target will be "" if not in the map
		// see updateHotplugVolumeStatus
		volumeStatus.Target = diskDeviceMap[volumeStatus.Name]
		if volumeStatus.HotplugVolume != nil {
			hasHotplug = true
			volumeStatus, tmpNeedsRefresh = c.updateHotplugVolumeStatus(vmi, volumeStatus, specVolumeMap)
			needsRefresh = needsRefresh || tmpNeedsRefresh
		}
		if volumeStatus.MemoryDumpVolume != nil {
			volumeStatus, tmpNeedsRefresh = c.updateMemoryDumpInfo(vmi, volumeStatus, domain)
			needsRefresh = needsRefresh || tmpNeedsRefresh
		}
		newStatuses = append(newStatuses, volumeStatus)
		newStatusMap[volumeStatus.Name] = volumeStatus
	}
	sort.SliceStable(newStatuses, func(i, j int) bool {
		return strings.Compare(newStatuses[i].Name, newStatuses[j].Name) == -1
	})
	if needsRefresh {
		c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second)
	}
	c.generateEventsForVolumeStatusChange(vmi, newStatusMap)
	vmi.Status.VolumeStatus = newStatuses

	return hasHotplug
}

func (c *VirtualMachineController) updateGuestInfoFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil || domain.Status.OSInfo.Name == "" || vmi.Status.GuestOSInfo.Name == domain.Status.OSInfo.Name {
		return
	}

	vmi.Status.GuestOSInfo.Name = domain.Status.OSInfo.Name
	vmi.Status.GuestOSInfo.Version = domain.Status.OSInfo.Version
	vmi.Status.GuestOSInfo.KernelRelease = domain.Status.OSInfo.KernelRelease
	vmi.Status.GuestOSInfo.PrettyName = domain.Status.OSInfo.PrettyName
	vmi.Status.GuestOSInfo.VersionID = domain.Status.OSInfo.VersionId
	vmi.Status.GuestOSInfo.KernelVersion = domain.Status.OSInfo.KernelVersion
	vmi.Status.GuestOSInfo.Machine = domain.Status.OSInfo.Machine
	vmi.Status.GuestOSInfo.ID = domain.Status.OSInfo.Id
}

func (c *VirtualMachineController) updateAccessCredentialConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

	if domain == nil || domain.Spec.Metadata.KubeVirt.AccessCredential == nil {
		return
	}

	message := domain.Spec.Metadata.KubeVirt.AccessCredential.Message
	status := k8sv1.ConditionFalse
	if domain.Spec.Metadata.KubeVirt.AccessCredential.Succeeded {
		status = k8sv1.ConditionTrue
	}

	add := false
	condition := condManager.GetCondition(vmi, v1.VirtualMachineInstanceAccessCredentialsSynchronized)
	if condition == nil {
		add = true
	} else if condition.Status != status || condition.Message != message {
		// if not as expected, remove, then add.
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceAccessCredentialsSynchronized)
		add = true
	}
	if add {
		newCondition := v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceAccessCredentialsSynchronized,
			LastTransitionTime: metav1.Now(),
			Status:             status,
			Message:            message,
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, newCondition)
		if status == k8sv1.ConditionTrue {
			eventMessage := "Access credentials sync successful."
			if message != "" {
				eventMessage = fmt.Sprintf("Access credentials sync successful: %s", message)
			}
			c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.AccessCredentialsSyncSuccess.String(), eventMessage)
		} else {
			c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.AccessCredentialsSyncFailed.String(),
				fmt.Sprintf("Access credentials sync failed: %s", message),
			)
		}
	}
}

func (c *VirtualMachineController) updateLiveMigrationConditions(vmi *v1.VirtualMachineInstance, condManager *controller.VirtualMachineInstanceConditionManager) {
	// Calculate whether the VM is migratable
	liveMigrationCondition, isBlockMigration := c.calculateLiveMigrationCondition(vmi)
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceIsMigratable) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, *liveMigrationCondition)
	} else {
		cond := condManager.GetCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
		if !equality.Semantic.DeepEqual(cond, liveMigrationCondition) {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
			vmi.Status.Conditions = append(vmi.Status.Conditions, *liveMigrationCondition)
		}
	}
	// Set VMI Migration Method
	if isBlockMigration {
		vmi.Status.MigrationMethod = v1.BlockMigration
	} else {
		vmi.Status.MigrationMethod = v1.LiveMigration
	}
	storageLiveMigCond := c.calculateLiveStorageMigrationCondition(vmi)
	condManager.UpdateCondition(vmi, storageLiveMigCond)
	evictable := migrations.VMIMigratableOnEviction(c.clusterConfig, vmi)
	if evictable && liveMigrationCondition.Status == k8sv1.ConditionFalse {
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, v1.Migrated.String(), "EvictionStrategy is set but vmi is not migratable; %s", liveMigrationCondition.Message)
	}
}

func (c *VirtualMachineController) updateGuestAgentConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) error {

	// Update the condition when GA is connected
	channelConnected := false
	if domain != nil {
		for _, channel := range domain.Spec.Devices.Channels {
			if channel.Target != nil {
				c.logger.V(4).Infof("Channel: %s, %s", channel.Target.Name, channel.Target.State)
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
			LastProbeTime: metav1.Now(),
			Status:        k8sv1.ConditionTrue,
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
	case !channelConnected:
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceAgentConnected)
	}

	if condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
		client, err := c.launcherClients.GetLauncherClient(vmi)
		if err != nil {
			return err
		}

		guestInfo, err := client.GetGuestInfo()
		if err != nil {
			return err
		}

		var supported = false
		var reason = ""

		// For current versions, virt-launcher's supported commands will always contain data.
		// For backwards compatibility: during upgrade from a previous version of KubeVirt,
		// virt-launcher might not provide any supported commands. If the list of supported
		// commands is empty, fall back to previous behavior.
		if len(guestInfo.SupportedCommands) > 0 {
			supported, reason = isGuestAgentSupported(vmi, guestInfo.SupportedCommands)
			c.logger.V(3).Object(vmi).Info(reason)
		} else {
			for _, version := range c.clusterConfig.GetSupportedAgentVersions() {
				supported = supported || regexp.MustCompile(version).MatchString(guestInfo.GAVersion)
			}
			if !supported {
				reason = fmt.Sprintf("Guest agent version '%s' is not supported", guestInfo.GAVersion)
			}
		}

		if !supported {
			if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent) {
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
					Reason:        reason,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
			}
		} else {
			condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceUnsupportedAgent)
		}

	}
	return nil
}

func (c *VirtualMachineController) updatePausedConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {

	// Update paused condition in case VMI was paused / unpaused
	if domain != nil && domain.Status.Status == api.Paused {
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			c.calculatePausedCondition(vmi, domain.Status.Reason)
		}
	} else if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		c.logger.Object(vmi).V(3).Info("Removing paused condition")
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstancePaused)
	}
}

func dumpTargetFile(vmiName, volName string) string {
	targetFileName := fmt.Sprintf("%s-%s-%s.memory.dump", vmiName, volName, time.Now().Format("20060102-150405"))
	return targetFileName
}

func (c *VirtualMachineController) updateMemoryDumpInfo(vmi *v1.VirtualMachineInstance, volumeStatus v1.VolumeStatus, domain *api.Domain) (v1.VolumeStatus, bool) {
	needsRefresh := false
	switch volumeStatus.Phase {
	case v1.HotplugVolumeMounted:
		needsRefresh = true
		c.logger.Object(vmi).V(3).Infof("Memory dump volume %s attached, marking it in progress", volumeStatus.Name)
		volumeStatus.Phase = v1.MemoryDumpVolumeInProgress
		volumeStatus.Message = fmt.Sprintf("Memory dump Volume %s is attached, getting memory dump", volumeStatus.Name)
		volumeStatus.Reason = VolumeMountedToPodReason
		volumeStatus.MemoryDumpVolume.TargetFileName = dumpTargetFile(vmi.Name, volumeStatus.Name)
	case v1.MemoryDumpVolumeInProgress:
		var memoryDumpMetadata *api.MemoryDumpMetadata
		if domain != nil {
			memoryDumpMetadata = domain.Spec.Metadata.KubeVirt.MemoryDump
		}
		if memoryDumpMetadata == nil || memoryDumpMetadata.FileName != volumeStatus.MemoryDumpVolume.TargetFileName {
			// memory dump wasnt triggered yet
			return volumeStatus, needsRefresh
		}
		needsRefresh = true
		if memoryDumpMetadata.StartTimestamp != nil {
			volumeStatus.MemoryDumpVolume.StartTimestamp = memoryDumpMetadata.StartTimestamp
		}
		if memoryDumpMetadata.EndTimestamp != nil && memoryDumpMetadata.Failed {
			c.logger.Object(vmi).Errorf("Memory dump to pvc %s failed: %v", volumeStatus.Name, memoryDumpMetadata.FailureReason)
			volumeStatus.Message = fmt.Sprintf("Memory dump to pvc %s failed: %v", volumeStatus.Name, memoryDumpMetadata.FailureReason)
			volumeStatus.Phase = v1.MemoryDumpVolumeFailed
			volumeStatus.MemoryDumpVolume.EndTimestamp = memoryDumpMetadata.EndTimestamp
		} else if memoryDumpMetadata.Completed {
			c.logger.Object(vmi).V(3).Infof("Marking memory dump to volume %s has completed", volumeStatus.Name)
			volumeStatus.Phase = v1.MemoryDumpVolumeCompleted
			volumeStatus.Message = fmt.Sprintf("Memory dump to Volume %s has completed successfully", volumeStatus.Name)
			volumeStatus.Reason = VolumeReadyReason
			volumeStatus.MemoryDumpVolume.EndTimestamp = memoryDumpMetadata.EndTimestamp
		}
	}

	return volumeStatus, needsRefresh
}

func (c *VirtualMachineController) updateFSFreezeStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {

	if domain == nil || domain.Status.FSFreezeStatus.Status == "" {
		return
	}

	if domain.Status.FSFreezeStatus.Status == api.FSThawed {
		vmi.Status.FSFreezeStatus = ""
	} else {
		vmi.Status.FSFreezeStatus = domain.Status.FSFreezeStatus.Status
	}

}

func IsoGuestVolumePath(namespace, name string, volume *v1.Volume) string {
	const basepath = "/var/run"
	switch {
	case volume.CloudInitNoCloud != nil:
		return filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", namespace, name, "noCloud.iso")
	case volume.CloudInitConfigDrive != nil:
		return filepath.Join(basepath, "kubevirt-ephemeral-disks", "cloud-init-data", namespace, name, "configdrive.iso")
	case volume.ConfigMap != nil:
		return config.GetConfigMapDiskPath(volume.Name)
	case volume.DownwardAPI != nil:
		return config.GetDownwardAPIDiskPath(volume.Name)
	case volume.Secret != nil:
		return config.GetSecretDiskPath(volume.Name)
	case volume.ServiceAccount != nil:
		return config.GetServiceAccountDiskPath()
	case volume.Sysprep != nil:
		return config.GetSysprepDiskPath(volume.Name)
	default:
		return ""
	}
}

func (c *VirtualMachineController) updateIsoSizeStatus(vmi *v1.VirtualMachineInstance) {
	var podUID string
	if vmi.Status.Phase != v1.Running {
		return
	}

	for k, v := range vmi.Status.ActivePods {
		if v == vmi.Status.NodeName {
			podUID = string(k)
			break
		}
	}
	if podUID == "" {
		log.DefaultLogger().Warningf("failed to find pod UID for VMI %s", vmi.Name)
		return
	}

	volumes := make(map[string]v1.Volume)
	for _, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		volume, ok := volumes[disk.Name]
		if !ok {
			log.DefaultLogger().Warningf("No matching volume with name %s found", disk.Name)
			continue
		}

		volPath := IsoGuestVolumePath(vmi.Namespace, vmi.Name, &volume)
		if volPath == "" {
			continue
		}

		res, err := c.podIsolationDetector.Detect(vmi)
		if err != nil {
			log.DefaultLogger().Reason(err).Warningf("failed to detect VMI %s", vmi.Name)
			continue
		}

		rootPath, err := res.MountRoot()
		if err != nil {
			log.DefaultLogger().Reason(err).Warningf("failed to detect VMI %s", vmi.Name)
			continue
		}

		safeVolPath, err := rootPath.AppendAndResolveWithRelativeRoot(volPath)
		if err != nil {
			log.DefaultLogger().Warningf("failed to determine file size for volume %s", volPath)
			continue
		}
		fileInfo, err := safepath.StatAtNoFollow(safeVolPath)
		if err != nil {
			log.DefaultLogger().Warningf("failed to determine file size for volume %s", volPath)
			continue
		}

		for i := range vmi.Status.VolumeStatus {
			if vmi.Status.VolumeStatus[i].Name == volume.Name {
				vmi.Status.VolumeStatus[i].Size = fileInfo.Size()
				continue
			}
		}
	}
}

func (c *VirtualMachineController) updateSELinuxContext(vmi *v1.VirtualMachineInstance) error {
	_, present, err := selinux.NewSELinux()
	if err != nil {
		return err
	}
	if present {
		context, err := selinux.GetVirtLauncherContext(vmi)
		if err != nil {
			return err
		}
		vmi.Status.SelinuxContext = context
	} else {
		vmi.Status.SelinuxContext = "none"
	}

	return nil
}

func (c *VirtualMachineController) updateMemoryInfo(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || vmi == nil || domain.Spec.CurrentMemory == nil {
		return nil
	}
	if vmi.Status.Memory == nil {
		vmi.Status.Memory = &v1.MemoryStatus{}
	}
	currentGuest := parseLibvirtQuantity(int64(domain.Spec.CurrentMemory.Value), domain.Spec.CurrentMemory.Unit)
	vmi.Status.Memory.GuestCurrent = currentGuest
	return nil
}

func (c *VirtualMachineController) updateVMIStatusFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	c.updateIsoSizeStatus(vmi)
	err := c.updateSELinuxContext(vmi)
	if err != nil {
		c.logger.Reason(err).Errorf("couldn't find the SELinux context for %s", vmi.Name)
	}
	c.updateGuestInfoFromDomain(vmi, domain)
	c.updateVolumeStatusesFromDomain(vmi, domain)
	c.updateFSFreezeStatus(vmi, domain)
	c.updateBackupStatus(vmi, domain)
	c.updateMachineType(vmi, domain)
	if err = c.updateMemoryInfo(vmi, domain); err != nil {
		return err
	}
	if err = c.cbtHandler.HandleChangedBlockTracking(vmi, domain); err != nil {
		return err
	}
	err = c.netStat.UpdateStatus(vmi, domain)
	return err
}

func (c *VirtualMachineController) updateVMIConditions(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) error {
	c.updateAccessCredentialConditions(vmi, domain, condManager)
	c.updateLiveMigrationConditions(vmi, condManager)
	err := c.updateGuestAgentConditions(vmi, domain, condManager)
	if err != nil {
		return err
	}
	c.updatePausedConditions(vmi, domain, condManager)

	return nil
}

func (c *VirtualMachineController) updateVMIStatus(oldStatus *v1.VirtualMachineInstanceStatus, vmi *v1.VirtualMachineInstance, domain *api.Domain, syncError error) (err error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()

	// Don't update the VirtualMachineInstance if it is already in a final state
	if vmi.IsFinal() {
		return nil
	}

	// Update VMI status fields based on what is reported on the domain
	err = c.updateVMIStatusFromDomain(vmi, domain)
	if err != nil {
		return err
	}

	// Calculate the new VirtualMachineInstance state based on what libvirt reported
	err = c.setVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}

	// Update conditions on VMI Status
	err = c.updateVMIConditions(vmi, domain, condManager)
	if err != nil {
		return err
	}

	// Store containerdisks and kernelboot checksums
	if err := c.updateChecksumInfo(vmi, syncError); err != nil {
		return err
	}

	// Handle sync error
	c.handleSyncError(vmi, condManager, syncError)

	controller.SetVMIPhaseTransitionTimestamp(oldStatus, &vmi.Status)

	// Only issue vmi update if status has changed
	if !equality.Semantic.DeepEqual(*oldStatus, vmi.Status) {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, metav1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.SetExpectations(key, 0, 0)
			return err
		}
	}

	// Record an event on the VMI when the VMI's phase changes
	if oldStatus.Phase != vmi.Status.Phase {
		c.recordPhaseChangeEvent(vmi)
	}

	return nil
}

type virtLauncherCriticalSecurebootError struct {
	msg string
}

func (e *virtLauncherCriticalSecurebootError) Error() string { return e.msg }

func (c *VirtualMachineController) handleSyncError(vmi *v1.VirtualMachineInstance, condManager *controller.VirtualMachineInstanceConditionManager, syncError error) {
	var criticalNetErr *neterrors.CriticalNetworkError
	if goerror.As(syncError, &criticalNetErr) {
		c.logger.Errorf("virt-launcher crashed due to a network error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	if _, ok := syncError.(*virtLauncherCriticalSecurebootError); ok {
		c.logger.Errorf("virt-launcher does not support the Secure Boot setting. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}

	if _, ok := syncError.(*vmiIrrecoverableError); ok {
		c.logger.Errorf("virt-launcher reached an irrecoverable error. Updating VMI %s status to Failed", vmi.Name)
		vmi.Status.Phase = v1.Failed
	}
	condManager.CheckFailure(vmi, syncError, "Synchronizing with the Domain failed.")
}

func (c *VirtualMachineController) recordPhaseChangeEvent(vmi *v1.VirtualMachineInstance) {
	switch vmi.Status.Phase {
	case v1.Running:
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Started.String(), VMIStarted)
	case v1.Succeeded:
		c.recorder.Event(vmi, k8sv1.EventTypeNormal, v1.Stopped.String(), VMIShutdown)
	case v1.Failed:
		c.recorder.Event(vmi, k8sv1.EventTypeWarning, v1.Stopped.String(), VMICrashed)
	}
}

// Condition and phase calculation methods

func (c *VirtualMachineController) calculatePausedCondition(vmi *v1.VirtualMachineInstance, reason api.StateChangeReason) {
	now := metav1.NewTime(time.Now())
	switch reason {
	case api.ReasonPausedMigration:
		if !isVMIPausedDuringMigration(vmi) || !c.isMigrationSource(vmi) {
			c.logger.Object(vmi).V(3).Infof("Domain is paused after migration by qemu, no condition needed")
			return
		}
		c.logger.Object(vmi).V(3).Info("Adding paused by migration monitor condition")
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedByMigrationMonitor",
			Message:            "VMI was paused by the migration monitor",
		})
	case api.ReasonPausedUser:
		c.logger.Object(vmi).V(3).Info("Adding paused condition")
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedByUser",
			Message:            "VMI was paused by user",
		})
	case api.ReasonPausedIOError:
		c.logger.Object(vmi).V(3).Info("Adding paused condition")
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstancePaused,
			Status:             k8sv1.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "PausedIOError",
			Message:            "VMI was paused, low-level IO error detected",
		})
	default:
		c.logger.Object(vmi).V(3).Infof("Domain is paused for unknown reason, %s", reason)
	}
}

func newNonMigratableCondition(msg string, reason string) *v1.VirtualMachineInstanceCondition {
	return &v1.VirtualMachineInstanceCondition{
		Type:    v1.VirtualMachineInstanceIsMigratable,
		Status:  k8sv1.ConditionFalse,
		Message: msg,
		Reason:  reason,
	}
}

// NonMigratableReason represents a single reason why a VMI is not migratable
type NonMigratableReason struct {
	Reason  string
	Message string
}

// evaluateCommonMigrationConstraints evaluates all non-volume migration constraints.
// This centralizes the migration checks that are common to both live migration and
// live storage migration, ensuring they stay in sync.
func (c *VirtualMachineController) evaluateCommonMigrationConstraints(vmi *v1.VirtualMachineInstance) []NonMigratableReason {
	var reasons []NonMigratableReason

	if err := c.checkNetworkInterfacesForMigration(vmi); err != nil {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonInterfaceNotMigratable,
			Message: err.Error(),
		})
	}

	if err := c.isHostModelMigratable(vmi); err != nil {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonCPUModeNotMigratable,
			Message: err.Error(),
		})
	}

	if vmiContainsPCIHostDevice(vmi) {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonHostDeviceNotMigratable,
			Message: "VMI uses a PCI host devices",
		})
	}

	if util.IsSEVVMI(vmi) {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonSEVNotMigratable,
			Message: "VMI uses SEV",
		})
	} else if util.IsTDXVMI(vmi) {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonTDXNotMigratable,
			Message: "VMI uses TDX",
		})
	}

	if util.IsSecureExecutionVMI(vmi) {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonSecureExecutionNotMigratable,
			Message: "VMI uses Secure Execution",
		})
	}

	if reservation.HasVMIPersistentReservation(vmi) {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonPRNotMigratable,
			Message: "VMI uses SCSI persistent reservation",
		})
	}

	if tscRequirement := topology.GetTscFrequencyRequirement(vmi); !topology.AreTSCFrequencyTopologyHintsDefined(vmi) && tscRequirement.Type == topology.RequiredForMigration {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonNoTSCFrequencyMigratable,
			Message: tscRequirement.Reason,
		})
	}

	if vmiFeatures := vmi.Spec.Domain.Features; vmiFeatures != nil && vmiFeatures.HypervPassthrough != nil && *vmiFeatures.HypervPassthrough.Enabled {
		reasons = append(reasons, NonMigratableReason{
			Reason:  v1.VirtualMachineInstanceReasonHypervPassthroughNotMigratable,
			Message: "VMI uses hyperv passthrough",
		})
	}

	return reasons
}

func (c *VirtualMachineController) calculateLiveMigrationCondition(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstanceCondition, bool) {
	isBlockMigration, blockErr := c.checkVolumesForMigration(vmi)

	// Check common migration constraints (NICs, CPU, devices, etc.)
	commonReasons := c.evaluateCommonMigrationConstraints(vmi)
	if len(commonReasons) > 0 {
		// Preserve existing "first reason wins" behavior
		r := commonReasons[0]
		return newNonMigratableCondition(r.Message, r.Reason), isBlockMigration
	}

	// Check volume-specific constraints
	if blockErr != nil {
		return newNonMigratableCondition(blockErr.Error(), v1.VirtualMachineInstanceReasonDisksNotMigratable), isBlockMigration
	}

	return &v1.VirtualMachineInstanceCondition{
		Type:   v1.VirtualMachineInstanceIsMigratable,
		Status: k8sv1.ConditionTrue,
	}, isBlockMigration
}

func vmiContainsPCIHostDevice(vmi *v1.VirtualMachineInstance) bool {
	return len(vmi.Spec.Domain.Devices.HostDevices) > 0 || len(vmi.Spec.Domain.Devices.GPUs) > 0
}

type multipleNonMigratableCondition struct {
	reasons []NonMigratableReason
}

func newMultipleNonMigratableCondition() *multipleNonMigratableCondition {
	return &multipleNonMigratableCondition{}
}

func (cond *multipleNonMigratableCondition) addNonMigratableReason(r NonMigratableReason) {
	cond.reasons = append(cond.reasons, r)
}

func (cond *multipleNonMigratableCondition) String() string {
	var buffer bytes.Buffer
	for i, r := range cond.reasons {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(fmt.Sprintf("%s: %s", r.Reason, r.Message))
	}
	return buffer.String()
}

func (cond *multipleNonMigratableCondition) generateStorageLiveMigrationCondition() *v1.VirtualMachineInstanceCondition {
	if len(cond.reasons) == 0 {
		return &v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceIsStorageLiveMigratable,
			Status: k8sv1.ConditionTrue,
		}
	}
	return &v1.VirtualMachineInstanceCondition{
		Type:    v1.VirtualMachineInstanceIsStorageLiveMigratable,
		Status:  k8sv1.ConditionFalse,
		Message: cond.String(),
		Reason:  v1.VirtualMachineInstanceReasonNotMigratable,
	}
}

func (c *VirtualMachineController) calculateLiveStorageMigrationCondition(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceCondition {
	multiCond := newMultipleNonMigratableCondition()

	// Evaluate all common migration constraints and collect them
	for _, r := range c.evaluateCommonMigrationConstraints(vmi) {
		multiCond.addNonMigratableReason(r)
	}

	return multiCond.generateStorageLiveMigrationCondition()
}

func (c *VirtualMachineController) setVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) error {
	phase, err := c.calculateVmPhaseForStatusReason(domain, vmi)
	if err != nil {
		return err
	}
	vmi.Status.Phase = phase
	return nil
}

func vmiHasTerminationGracePeriod(vmi *v1.VirtualMachineInstance) bool {
	// if not set we use the default graceperiod
	return vmi.Spec.TerminationGracePeriodSeconds == nil ||
		(vmi.Spec.TerminationGracePeriodSeconds != nil && *vmi.Spec.TerminationGracePeriodSeconds != 0)
}

func domainHasGracePeriod(domain *api.Domain) bool {
	return domain != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod != nil &&
		domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds != 0
}

func isACPIEnabled(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	return (vmiHasTerminationGracePeriod(vmi) || (vmi.Spec.TerminationGracePeriodSeconds == nil && domainHasGracePeriod(domain))) &&
		domain != nil &&
		domain.Spec.Features != nil &&
		domain.Spec.Features.ACPI != nil
}

func (c *VirtualMachineController) calculateVmPhaseForStatusReason(domain *api.Domain, vmi *v1.VirtualMachineInstance) (v1.VirtualMachineInstancePhase, error) {

	if domain == nil {
		switch {
		case vmi.IsScheduled():
			isUnresponsive, isInitialized, err := c.launcherClients.IsLauncherClientUnresponsive(vmi)

			if err != nil {
				return vmi.Status.Phase, err
			}
			if !isInitialized {
				c.queue.AddAfter(controller.VirtualMachineInstanceKey(vmi), time.Second*1)
				return vmi.Status.Phase, err
			} else if isUnresponsive {
				// virt-launcher is gone and VirtualMachineInstance never transitioned
				// from scheduled to Running.
				return v1.Failed, nil
			}
			return v1.Scheduled, nil
		case !vmi.IsRunning() && !vmi.IsFinal():
			return v1.Scheduled, nil
		case !vmi.IsFinal():
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
				if isACPIEnabled(vmi, domain) {
					// When ACPI is available, the domain was tried to be shutdown,
					// and destroyed means that the domain was destroyed after the graceperiod expired.
					// Without ACPI a destroyed domain is ok.
					return v1.Failed, nil
				}
				if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Failed && vmi.Status.MigrationState.Mode == v1.MigrationPostCopy {
					// A VMI that failed a post-copy migration should never succeed
					return v1.Failed, nil
				}
				return v1.Succeeded, nil
			case api.ReasonShutdown, api.ReasonSaved, api.ReasonFromSnapshot:
				return v1.Succeeded, nil
			case api.ReasonMigrated:
				// if the domain migrated, we no longer know the phase.
				return vmi.Status.Phase, nil
			}
		case api.Paused:
			switch domain.Status.Reason {
			case api.ReasonPausedPostcopyFailed:
				return v1.Failed, nil
			default:
				return v1.Running, nil
			}
		case api.Running, api.Blocked, api.PMSuspended:
			return v1.Running, nil
		}
	}
	return vmi.Status.Phase, nil
}

// Additional helper methods

func (c *VirtualMachineController) isHostModelMigratable(vmi *v1.VirtualMachineInstance) error {
	if cpu := vmi.Spec.Domain.CPU; cpu != nil && cpu.Model == v1.CPUModeHostModel {
		if c.hostCpuModel == "" {
			err := fmt.Errorf("the node \"%s\" does not allow migration with host-model", vmi.Status.NodeName)
			c.logger.Object(vmi).Errorf("%s", err.Error())
			return err
		}
	}
	return nil
}

func (c *VirtualMachineController) updateMachineType(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil || vmi == nil {
		return
	}
	if domain.Spec.OS.Type.Machine != "" {
		vmi.Status.Machine = &v1.Machine{Type: domain.Spec.OS.Type.Machine}
	}
}

func parseLibvirtQuantity(value int64, unit string) *resource.Quantity {
	switch unit {
	case "b", "bytes":
		return resource.NewQuantity(value, resource.BinarySI)
	case "KB":
		return resource.NewQuantity(value*1000, resource.DecimalSI)
	case "MB":
		return resource.NewQuantity(value*1000*1000, resource.DecimalSI)
	case "GB":
		return resource.NewQuantity(value*1000*1000*1000, resource.DecimalSI)
	case "TB":
		return resource.NewQuantity(value*1000*1000*1000*1000, resource.DecimalSI)
	case "k", "KiB":
		return resource.NewQuantity(value*1024, resource.BinarySI)
	case "M", "MiB":
		return resource.NewQuantity(value*1024*1024, resource.BinarySI)
	case "G", "GiB":
		return resource.NewQuantity(value*1024*1024*1024, resource.BinarySI)
	case "T", "TiB":
		return resource.NewQuantity(value*1024*1024*1024*1024, resource.BinarySI)
	}
	return nil
}

func (c *VirtualMachineController) updateBackupStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil ||
		domain.Spec.Metadata.KubeVirt.Backup == nil ||
		vmi.Status.ChangedBlockTracking == nil ||
		vmi.Status.ChangedBlockTracking.BackupStatus == nil {
		return
	}
	backupMetadata := domain.Spec.Metadata.KubeVirt.Backup
	// Handle the case where a new backupStatus was initiated but
	// the backupMetadata wasnt reinitialized yet
	if vmi.Status.ChangedBlockTracking.BackupStatus.BackupName != backupMetadata.Name {
		return
	}
	vmi.Status.ChangedBlockTracking.BackupStatus.Completed = backupMetadata.Completed
	if backupMetadata.StartTimestamp != nil {
		vmi.Status.ChangedBlockTracking.BackupStatus.StartTimestamp = backupMetadata.StartTimestamp
	}
	if backupMetadata.EndTimestamp != nil {
		vmi.Status.ChangedBlockTracking.BackupStatus.EndTimestamp = backupMetadata.EndTimestamp
	}
	if backupMetadata.BackupMsg != "" {
		vmi.Status.ChangedBlockTracking.BackupStatus.BackupMsg = &backupMetadata.BackupMsg
	}
	if backupMetadata.CheckpointName != "" {
		vmi.Status.ChangedBlockTracking.BackupStatus.CheckpointName = &backupMetadata.CheckpointName
	}
	if backupMetadata.Volumes != "" {
		var volumes []backupv1.BackupVolumeInfo
		if err := json.Unmarshal([]byte(backupMetadata.Volumes), &volumes); err == nil && len(volumes) > 0 {
			vmi.Status.ChangedBlockTracking.BackupStatus.Volumes = volumes
		}
	}
	// TODO: Handle backup failure (backupMetadata.Failed) and abort status (backupMetadata.AbortStatus)
}
