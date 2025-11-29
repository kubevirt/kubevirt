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

import "kubevirt.io/kubevirt/pkg/virt-config/featuregate"

/*
 This module is intended for determining whether an optional feature is enabled or not at the cluster-level.
*/

func (config *ClusterConfig) isFeatureGateDefined(featureGate string) bool {
	for _, fg := range config.GetConfig().DeveloperConfiguration.FeatureGates {
		if fg == featureGate {
			return true
		}
	}
	return false
}

func (config *ClusterConfig) IsFeatureGateEnabled(featureGate string) bool {
	if fg := featuregate.FeatureGateInfo(featureGate); fg != nil && fg.State == featuregate.GA {
		return true
	}

	if config.isFeatureGateDefined(featureGate) {
		return true
	}
	return false
}

func (config *ClusterConfig) ExpandDisksEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.ExpandDisksGate)
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

func (config *ClusterConfig) SRIOVLiveMigrationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.SRIOVLiveMigrationGate)
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

func (config *ClusterConfig) VMExportEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VMExportGate)
}

func (config *ClusterConfig) HotplugVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HotplugVolumesGate)
}

func (config *ClusterConfig) HostDiskEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HostDiskGate)
}

func (config *ClusterConfig) OldVirtiofsEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VirtIOFSGate)
}

func (config *ClusterConfig) VirtiofsConfigVolumesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VirtIOFSConfigVolumesGate)
}

func (config *ClusterConfig) VirtiofsStorageEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.VirtIOFSStorageVolumeGate)
}

func (config *ClusterConfig) MacvtapEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.MacvtapGate)
}

func (config *ClusterConfig) PasstEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PasstGate)
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

func (config *ClusterConfig) MediatedDevicesHandlingDisabled() bool {
	return config.IsFeatureGateEnabled(featuregate.DisableMediatedDevicesHandling)
}

func (config *ClusterConfig) KubevirtSeccompProfileEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.KubevirtSeccompProfile)
}

func (config *ClusterConfig) HotplugNetworkInterfacesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.HotplugNetworkIfacesGate)
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

func (config *ClusterConfig) PanicDevicesEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PanicDevicesGate)
}

func (config *ClusterConfig) PasstIPStackMigrationEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.PasstIPStackMigration)
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

func (config *ClusterConfig) IncrementalBackupEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.IncrementalBackupGate)
}

func (config *ClusterConfig) MigrationPriorityQueueEnabled() bool {
	return config.IsFeatureGateEnabled(featuregate.MigrationPriorityQueue)
}
