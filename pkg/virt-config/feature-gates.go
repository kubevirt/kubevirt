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

/*
 This module is intended for determining whether an optional feature is enabled or not at the cluster-level.
*/

const (
	ExpandDisksGate   = "ExpandDisks"
	CPUManager        = "CPUManager"
	NUMAFeatureGate   = "NUMA"
	IgnitionGate      = "ExperimentalIgnitionSupport"
	LiveMigrationGate = "LiveMigration"
	// SRIOVLiveMigrationGate enables Live Migration for VM's with network SR-IOV interfaces.
	SRIOVLiveMigrationGate     = "SRIOVLiveMigration"
	CPUNodeDiscoveryGate       = "CPUNodeDiscovery"
	HypervStrictCheckGate      = "HypervStrictCheck"
	SidecarGate                = "Sidecar"
	GPUGate                    = "GPU"
	HostDevicesGate            = "HostDevices"
	SnapshotGate               = "Snapshot"
	VMExportGate               = "VMExport"
	HotplugVolumesGate         = "HotplugVolumes"
	HostDiskGate               = "HostDisk"
	VirtIOFSGate               = "ExperimentalVirtiofsSupport"
	MacvtapGate                = "Macvtap"
	PasstGate                  = "Passt"
	DownwardMetricsFeatureGate = "DownwardMetrics"
	NonRootDeprecated          = "NonRootExperimental"
	NonRoot                    = "NonRoot"
	Root                       = "Root"
	ClusterProfiler            = "ClusterProfiler"
	WorkloadEncryptionSEV      = "WorkloadEncryptionSEV"
	// DockerSELinuxMCSWorkaround sets the SELinux level of all the non-compute virt-launcher containers to "s0".
	DockerSELinuxMCSWorkaround = "DockerSELinuxMCSWorkaround"
	PSA                        = "PSA"
	VSOCKGate                  = "VSOCK"
	// DisableCustomSELinuxPolicy disables the installation of the custom SELinux policy for virt-launcher
	DisableCustomSELinuxPolicy = "DisableCustomSELinuxPolicy"
	// KubevirtSeccompProfile indicate that Kubevirt will install its custom profile and
	// user can tell Kubevirt to use it
	KubevirtSeccompProfile = "KubevirtSeccompProfile"
	// DisableMediatedDevicesHandling disables the handling of mediated
	// devices, its creation and deletion
	DisableMediatedDevicesHandling = "DisableMDEVConfiguration"
	// HotplugNetworkIfacesGate enables the virtio network interface hotplug feature
	HotplugNetworkIfacesGate = "HotplugNICs"
	// PersistentReservation enables the use of the SCSI persistent reservation with the pr-helper daemon
	PersistentReservation = "PersistentReservation"
	// VMPersistentState enables persisting backend state files of VMs, such as the contents of the vTPM
	VMPersistentState = "VMPersistentState"
	Multiarchitecture = "MultiArchitecture"
	// VMLiveUpdateFeaturesGate allows updating ceratin VM fields, such as CPU sockets to enable hot-plug functionality.
	VMLiveUpdateFeaturesGate = "VMLiveUpdateFeatures"
)

var deprecatedFeatureGates = [...]string{
	LiveMigrationGate,
	SRIOVLiveMigrationGate,
	NonRoot,
	NonRootDeprecated,
	PSA,
}

func (config *ClusterConfig) isFeatureGateEnabled(featureGate string) bool {
	if config.IsFeatureGateDeprecated(featureGate) {
		// Deprecated feature gates are considered enabled and no-op.
		// For more info about deprecation policy: https://github.com/kubevirt/kubevirt/blob/main/docs/deprecation.md
		return true
	}

	for _, fg := range config.GetConfig().DeveloperConfiguration.FeatureGates {
		if fg == featureGate {
			return true
		}
	}
	return false
}

func (config *ClusterConfig) IsFeatureGateDeprecated(featureGate string) bool {
	for _, deprecatedFeatureGate := range deprecatedFeatureGates {
		if featureGate == deprecatedFeatureGate {
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
	return config.isFeatureGateEnabled(NUMAFeatureGate)
}

func (config *ClusterConfig) DownwardMetricsEnabled() bool {
	return config.isFeatureGateEnabled(DownwardMetricsFeatureGate)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.isFeatureGateEnabled(IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(LiveMigrationGate)
}

func (config *ClusterConfig) SRIOVLiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(SRIOVLiveMigrationGate)
}

func (config *ClusterConfig) HypervStrictCheckEnabled() bool {
	return config.isFeatureGateEnabled(HypervStrictCheckGate)
}

func (config *ClusterConfig) CPUNodeDiscoveryEnabled() bool {
	return config.isFeatureGateEnabled(CPUNodeDiscoveryGate)
}

func (config *ClusterConfig) SidecarEnabled() bool {
	return config.isFeatureGateEnabled(SidecarGate)
}

func (config *ClusterConfig) GPUPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(GPUGate)
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
	return config.isFeatureGateEnabled(MacvtapGate)
}

func (config *ClusterConfig) PasstEnabled() bool {
	return config.isFeatureGateEnabled(PasstGate)
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
	return config.isFeatureGateEnabled(DockerSELinuxMCSWorkaround)
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
	return config.isFeatureGateEnabled(HotplugNetworkIfacesGate)
}

func (config *ClusterConfig) PersistentReservationEnabled() bool {
	return config.isFeatureGateEnabled(PersistentReservation)
}

func (config *ClusterConfig) VMPersistentStateEnabled() bool {
	return config.isFeatureGateEnabled(VMPersistentState)
}
func (config *ClusterConfig) MultiArchitectureEnabled() bool {
	return config.isFeatureGateEnabled(Multiarchitecture)
}
func (config *ClusterConfig) VMLiveUpdateFeaturesEnabled() bool {
	return config.isFeatureGateEnabled(VMLiveUpdateFeaturesGate)
}
