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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package common

// Reasons for vmi events
const (
	// FailedCreatePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be created.
	FailedCreatePodReason = "FailedCreate"
	// SuccessfulCreatePodReason is added in an event when a pod for a vmi controller
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreate"
	// FailedDeletePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be deleted.
	FailedDeletePodReason = "FailedDelete"
	// SuccessfulDeletePodReason is added in an event when a pod for a vmi controller
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDelete"
	// FailedHandOverPodReason is added in an event and in a vmi controller condition
	// when transferring the pod ownership from the controller to virt-hander fails.
	FailedHandOverPodReason = "FailedHandOver"
	// FailedBackendStorageCreateReason is added in an event when posting a dynamically
	// generated dataVolume to the cluster fails.
	FailedBackendStorageCreateReason = "FailedBackendStorageCreate"
	// SuccessfulHandOverPodReason is added in an event
	// when the pod ownership transfer from the controller to virt-hander succeeds.
	SuccessfulHandOverPodReason = "SuccessfulHandOver"
	// FailedDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume reaches the failed status phase.
	FailedDataVolumeImportReason = "FailedDataVolumeImport"
	// FailedGuaranteePodResourcesReason is added in an event and in a vmi controller condition
	// when a pod has been created without a Guaranteed resources.
	FailedGuaranteePodResourcesReason = "FailedGuaranteeResources"
	// FailedGatherhingClusterTopologyHints is added if the cluster topology hints can't be collected for a VMI by virt-controller
	FailedGatherhingClusterTopologyHints = "FailedGatherhingClusterTopologyHints"
	// FailedPvcNotFoundReason is added in an event
	// when a PVC for a volume was not found.
	FailedPvcNotFoundReason = "FailedPvcNotFound"
	// SuccessfulMigrationReason is added when a migration attempt completes successfully
	SuccessfulMigrationReason = "SuccessfulMigration"
	// FailedMigrationReason is added when a migration attempt fails
	FailedMigrationReason = "FailedMigration"
	// SuccessfulAbortMigrationReason is added when an attempt to abort migration completes successfully
	SuccessfulAbortMigrationReason = "SuccessfulAbortMigration"
	// MigrationTargetPodUnschedulable is added a migration target pod enters Unschedulable phase
	MigrationTargetPodUnschedulable = "migrationTargetPodUnschedulable"
	// FailedAbortMigrationReason is added when an attempt to abort migration fails
	FailedAbortMigrationReason = "FailedAbortMigration"
	// MissingAttachmentPodReason is set when we have a hotplugged volume, but the attachment pod is missing
	MissingAttachmentPodReason = "MissingAttachmentPod"
	// PVCNotReadyReason is set when the PVC is not ready to be hot plugged.
	PVCNotReadyReason = "PVCNotReady"
	// FailedHotplugSyncReason is set when a hotplug specific failure occurs during sync
	FailedHotplugSyncReason = "FailedHotplugSync"
	// ErrImagePullReason is set when an error has occured while pulling an image for a containerDisk VM volume.
	ErrImagePullReason = "ErrImagePull"
	// ImagePullBackOffReason is set when an error has occured while pulling an image for a containerDisk VM volume,
	// and that kubelet is backing off before retrying.
	ImagePullBackOffReason = "ImagePullBackOff"
	// NoSuitableNodesForHostModelMigration is set when a VMI with host-model CPU mode tries to migrate but no node
	// is suitable for migration (since CPU model / required features are not supported)
	NoSuitableNodesForHostModelMigration = "NoSuitableNodesForHostModelMigration"
	// FailedPodPatchReason is set when a pod patch error occurs during sync
	FailedPodPatchReason = "FailedPodPatch"
	// MigrationBackoffReason is set when an error has occured while migrating
	// and virt-controller is backing off before retrying.
	MigrationBackoffReason = "MigrationBackoff"
)
