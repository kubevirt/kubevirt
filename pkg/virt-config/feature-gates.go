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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtconfig

import "kubevirt.io/kubevirt/pkg/virt-config/deprecation"

/*
 This module is intended for determining whether an optional feature is enabled or not at the cluster-level.
*/

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
	// Owner: @lyarwood
	// Alpha: v1.4.0
	//
	// InstancetypeReferencePolicy allows a cluster admin to control how a VirtualMachine references instance types and preferences
	// through the kv.spec.configuration.instancetype.referencePolicy configurable.
	InstancetypeReferencePolicy = "InstancetypeReferencePolicy"
)

func (config *ClusterConfig) isFeatureGateEnabled(featureGate string) bool {
	deprecatedFeature := deprecation.FeatureGateInfo(featureGate)
	if deprecatedFeature != nil && deprecatedFeature.State == deprecation.GA {
		return true
	}

	for _, fg := range config.GetConfig().DeveloperConfiguration.FeatureGates {
		if fg == featureGate {
			return true
		}
	}
	return false
}

func (config *ClusterConfig) ExpandDisksEnabled() bool {
	return config.isFeatureGateEnabled(ExpandDisksGate)
}

func (config *ClusterConfig) CPUManagerEnabled() bool {
	return config.isFeatureGateEnabled(CPUManager)
}

func (config *ClusterConfig) NUMAEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.NUMAFeatureGate)
}

func (config *ClusterConfig) DownwardMetricsEnabled() bool {
	return config.isFeatureGateEnabled(DownwardMetricsFeatureGate)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.isFeatureGateEnabled(IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.LiveMigrationGate)
}

func (config *ClusterConfig) SRIOVLiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.SRIOVLiveMigrationGate)
}

func (config *ClusterConfig) HypervStrictCheckEnabled() bool {
	return config.isFeatureGateEnabled(HypervStrictCheckGate)
}

func (config *ClusterConfig) CPUNodeDiscoveryEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.CPUNodeDiscoveryGate)
}

func (config *ClusterConfig) SidecarEnabled() bool {
	return config.isFeatureGateEnabled(SidecarGate)
}

func (config *ClusterConfig) GPUPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.GPUGate)
}

func (config *ClusterConfig) SnapshotEnabled() bool {
	return config.isFeatureGateEnabled(SnapshotGate)
}

func (config *ClusterConfig) VMExportEnabled() bool {
	return config.isFeatureGateEnabled(VMExportGate)
}

func (config *ClusterConfig) HotplugVolumesEnabled() bool {
	return config.isFeatureGateEnabled(HotplugVolumesGate)
}

func (config *ClusterConfig) HostDiskEnabled() bool {
	return config.isFeatureGateEnabled(HostDiskGate)
}

func (config *ClusterConfig) VirtiofsEnabled() bool {
	return config.isFeatureGateEnabled(VirtIOFSGate)
}

func (config *ClusterConfig) MacvtapEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.MacvtapGate)
}

func (config *ClusterConfig) PasstEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.PasstGate)
}

func (config *ClusterConfig) HostDevicesPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(HostDevicesGate)
}

func (config *ClusterConfig) RootEnabled() bool {
	return config.isFeatureGateEnabled(Root)
}

func (config *ClusterConfig) ClusterProfilerEnabled() bool {
	return config.isFeatureGateEnabled(ClusterProfiler)
}

func (config *ClusterConfig) WorkloadEncryptionSEVEnabled() bool {
	return config.isFeatureGateEnabled(WorkloadEncryptionSEV)
}

func (config *ClusterConfig) DockerSELinuxMCSWorkaroundEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.DockerSELinuxMCSWorkaround)
}

func (config *ClusterConfig) VSOCKEnabled() bool {
	return config.isFeatureGateEnabled(VSOCKGate)
}

func (config *ClusterConfig) CustomSELinuxPolicyDisabled() bool {
	return config.isFeatureGateEnabled(DisableCustomSELinuxPolicy)
}

func (config *ClusterConfig) MediatedDevicesHandlingDisabled() bool {
	return config.isFeatureGateEnabled(DisableMediatedDevicesHandling)
}

func (config *ClusterConfig) KubevirtSeccompProfileEnabled() bool {
	return config.isFeatureGateEnabled(KubevirtSeccompProfile)
}

func (config *ClusterConfig) HotplugNetworkInterfacesEnabled() bool {
	return config.isFeatureGateEnabled(deprecation.HotplugNetworkIfacesGate)
}

func (config *ClusterConfig) PersistentReservationEnabled() bool {
	return config.isFeatureGateEnabled(PersistentReservation)
}

func (config *ClusterConfig) VMPersistentStateEnabled() bool {
	return config.isFeatureGateEnabled(VMPersistentState)
}

func (config *ClusterConfig) MultiArchitectureEnabled() bool {
	return config.isFeatureGateEnabled(MultiArchitecture)
}

func (config *ClusterConfig) VMLiveUpdateFeaturesEnabled() bool {
	return config.isFeatureGateEnabled(VMLiveUpdateFeaturesGate)
}

func (config *ClusterConfig) NetworkBindingPlugingsEnabled() bool {
	return config.isFeatureGateEnabled(NetworkBindingPlugingsGate)
}

func (config *ClusterConfig) AutoResourceLimitsEnabled() bool {
	return config.isFeatureGateEnabled(AutoResourceLimitsGate)
}

func (config *ClusterConfig) AlignCPUsEnabled() bool {
	return config.isFeatureGateEnabled(AlignCPUsGate)
}

func (config *ClusterConfig) VolumesUpdateStrategyEnabled() bool {
	return config.isFeatureGateEnabled(VolumesUpdateStrategy)
}

func (config *ClusterConfig) VolumeMigrationEnabled() bool {
	return config.isFeatureGateEnabled(VolumeMigration)
}

func (config *ClusterConfig) NodeRestrictionEnabled() bool {
	return config.isFeatureGateEnabled(NodeRestrictionGate)
}

func (config *ClusterConfig) DynamicPodInterfaceNamingEnabled() bool {
	return config.isFeatureGateEnabled(DynamicPodInterfaceNamingGate)
}
