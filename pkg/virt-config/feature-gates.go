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

package virtconfig

import (
	"slices"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

/*
 This module is intended for determining whether an optional feature is enabled or not at the cluster-level.
*/

func (config *ClusterConfig) IsFeatureGateEnabled(featureGate string) bool {
	if fg := featuregate.FeatureGateInfo(featureGate); fg != nil && fg.State == featuregate.GA {
		return true
	}

	if isExplicitlyEnabled := slices.Contains(config.GetConfig().DeveloperConfiguration.FeatureGates, featureGate); isExplicitlyEnabled {
		return true
	}

	if isExplicitlyDisabled := slices.Contains(config.GetConfig().DeveloperConfiguration.DisabledFeatureGates, featureGate); isExplicitlyDisabled {
		return false
	}

	return false
}

func (config *ClusterConfig) CPUManagerEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.CPUManager)
}

func (config *ClusterConfig) NUMAEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.NUMAFeatureGate)
}

func (config *ClusterConfig) DownwardMetricsEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.DownwardMetricsFeatureGate)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.LiveMigrationGate)
}

func (config *ClusterConfig) UtilityVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.UtilityVolumesGate)
}

func (config *ClusterConfig) HypervStrictCheckEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HypervStrictCheckGate)
}

func (config *ClusterConfig) CPUNodeDiscoveryEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.CPUNodeDiscoveryGate)
}

func (config *ClusterConfig) SidecarEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.SidecarGate)
}

func (config *ClusterConfig) GPUPassthroughEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.GPUGate)
}

func (config *ClusterConfig) SnapshotEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.SnapshotGate)
}

func (config *ClusterConfig) HotplugVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HotplugVolumesGate)
}

func (config *ClusterConfig) HostDiskEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HostDiskGate)
}

func (config *ClusterConfig) VirtiofsStorageEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VirtIOFSStorageVolumeGate)
}

func (config *ClusterConfig) HostDevicesPassthroughEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HostDevicesGate)
}

func (config *ClusterConfig) RootEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.Root)
}

func (config *ClusterConfig) WorkloadEncryptionSEVEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.WorkloadEncryptionSEV)
}

func (config *ClusterConfig) WorkloadEncryptionTDXEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.WorkloadEncryptionTDX)
}

func (config *ClusterConfig) DockerSELinuxMCSWorkaroundEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.DockerSELinuxMCSWorkaround)
}

func (config *ClusterConfig) VSOCKEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VSOCKGate)
}

func (config *ClusterConfig) KubevirtSeccompProfileEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.KubevirtSeccompProfile)
}

func (config *ClusterConfig) PersistentReservationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PersistentReservation)
}

func (config *ClusterConfig) MultiArchitectureEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.MultiArchitecture)
}

func (config *ClusterConfig) AlignCPUsEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.AlignCPUsGate)
}

func (config *ClusterConfig) ImageVolumeEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ImageVolume)
}

func (config *ClusterConfig) LibvirtHooksServerAndClientEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.LibvirtHooksServerAndClient)
}

func (config *ClusterConfig) VideoConfigEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VideoConfig)
}

func (config *ClusterConfig) NodeRestrictionEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.NodeRestrictionGate)
}

func (config *ClusterConfig) ObjectGraphEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ObjectGraph)
}

func (config *ClusterConfig) DeclarativeHotplugVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.DeclarativeHotplugVolumesGate)
}

func (config *ClusterConfig) SecureExecutionEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.SecureExecution)
}

func (config *ClusterConfig) PasstBindingEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PasstBinding)
}

func (config *ClusterConfig) DecentralizedLiveMigrationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.DecentralizedLiveMigration)
}

func (config *ClusterConfig) GPUsWithDRAGateEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.GPUsWithDRAGate)
}

func (config *ClusterConfig) HostDevicesWithDRAEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HostDevicesWithDRAGate)
}

func (config *ClusterConfig) ConfigurableHypervisorEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ConfigurableHypervisor)
}

func (config *ClusterConfig) PCINUMAAwareTopologyEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PCINUMAAwareTopologyEnabled)
}

func (config *ClusterConfig) IncrementalBackupEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.IncrementalBackupGate)
}

func (config *ClusterConfig) MigrationPriorityQueueEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.MigrationPriorityQueue)
}

func (config *ClusterConfig) PodSecondaryInterfaceNamingUpgradeEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PodSecondaryInterfaceNamingUpgrade)
}

func (config *ClusterConfig) ExternalNetResourceInjectionEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ExternalNetResourceInjection)
}

func (config *ClusterConfig) RebootPolicyEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.RebootPolicy)
}

func (config *ClusterConfig) VmiMemoryOverheadReportEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VmiMemoryOverheadReport)
}

func (config *ClusterConfig) TemplateEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.Template)
}

func (config *ClusterConfig) ContainerPathVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ContainerPathVolumesGate)
}

func (config *ClusterConfig) ReservedOverheadMemlockEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ReservedOverheadMemlock)
}

func (config *ClusterConfig) OptOutRoleAggregationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.OptOutRoleAggregation)
}

func (config *ClusterConfig) LiveUpdateNADRefEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.LiveUpdateNADRef)
}

func (config *ClusterConfig) VGPULiveMigrationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VGPULiveMigration)
}
