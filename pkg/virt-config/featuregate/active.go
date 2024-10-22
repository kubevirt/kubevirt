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

package featuregate

const (
	ExpandDisksGate       = "ExpandDisks"
	CPUManager            = "CPUManager"
	IgnitionGate          = "ExperimentalIgnitionSupport"
	HypervStrictCheckGate = "HypervStrictCheck"
	SidecarGate           = "Sidecar"
	HostDevicesGate       = "HostDevices"
	SnapshotGate          = "Snapshot"
	VMExportGate          = "VMExport"
	HotplugVolumesGate    = "HotplugVolumes"
	HostDiskGate          = "HostDisk"
	VirtIOFSGate          = "ExperimentalVirtiofsSupport"

	DownwardMetricsFeatureGate = "DownwardMetrics"
	Root                       = "Root"
	ClusterProfiler            = "ClusterProfiler"
	WorkloadEncryptionSEV      = "WorkloadEncryptionSEV"
	VSOCKGate                  = "VSOCK"
	// DisableCustomSELinuxPolicy disables the installation of the custom SELinux policy for virt-launcher
	DisableCustomSELinuxPolicy = "DisableCustomSELinuxPolicy"
	// KubevirtSeccompProfile indicate that Kubevirt will install its custom profile and
	// user can tell Kubevirt to use it
	KubevirtSeccompProfile = "KubevirtSeccompProfile"
	// DisableMediatedDevicesHandling disables the handling of mediated
	// devices, its creation and deletion
	DisableMediatedDevicesHandling = "DisableMDEVConfiguration"
	// PersistentReservation enables the use of the SCSI persistent reservation with the pr-helper daemon
	PersistentReservation = "PersistentReservation"
	// VMPersistentState enables persisting backend state files of VMs, such as the contents of the vTPM
	VMPersistentState = "VMPersistentState"
	MultiArchitecture = "MultiArchitecture"
	// VMLiveUpdateFeaturesGate allows updating certain VM fields, such as CPU sockets to enable hot-plug functionality.
	VMLiveUpdateFeaturesGate = "VMLiveUpdateFeatures"
	// NetworkBindingPlugingsGate enables using a plugin to bind the pod and the VM network
	// Alpha: v1.1.0
	// Beta:  v1.4.0
	NetworkBindingPlugingsGate = "NetworkBindingPlugins"
	// AutoResourceLimitsGate enables automatic setting of vmi limits if there is a ResourceQuota with limits associated with the vmi namespace.
	AutoResourceLimitsGate = "AutoResourceLimitsGate"

	// AlignCPUsGate allows emulator thread to assign two extra CPUs if needed to complete even parity.
	AlignCPUsGate = "AlignCPUs"

	// VolumesUpdateStrategy enables to specify the strategy on the volume updates.
	VolumesUpdateStrategy = "VolumesUpdateStrategy"
	// VolumeMigration enables to migrate the storage. It depends on the VolumesUpdateStrategy feature.
	VolumeMigration = "VolumeMigration"
	// Owner: @xpivarc
	// Alpha: v1.3.0
	//
	// NodeRestriction enables Kubelet's like NodeRestriction but for Kubevirt's virt-handler.
	// This feature requires following Kubernetes feature gate "ServiceAccountTokenPodNodeInfo". The feature gate is available
	// in Kubernetes 1.30 as Beta.
	NodeRestrictionGate = "NodeRestriction"
	// DynamicPodInterfaceNaming enables a mechanism to dynamically determine the primary pod interface for KuveVirt virtual machines.
	DynamicPodInterfaceNamingGate = "DynamicPodInterfaceNaming"
)

func init() {
	RegisterFeatureGate(FeatureGate{Name: ExpandDisksGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: CPUManager, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: IgnitionGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HypervStrictCheckGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SidecarGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDevicesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SnapshotGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VMExportGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HotplugVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDiskGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VirtIOFSGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DownwardMetricsFeatureGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: Root, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: ClusterProfiler, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: WorkloadEncryptionSEV, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VSOCKGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DisableCustomSELinuxPolicy, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: KubevirtSeccompProfile, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DisableMediatedDevicesHandling, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PersistentReservation, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VMPersistentState, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: MultiArchitecture, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VMLiveUpdateFeaturesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: BochsDisplayForEFIGuests, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: AutoResourceLimitsGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: AlignCPUsGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VolumesUpdateStrategy, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VolumeMigration, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: NodeRestrictionGate, State: Alpha})

	RegisterFeatureGate(FeatureGate{Name: NetworkBindingPlugingsGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: DynamicPodInterfaceNamingGate, State: Beta})
}
