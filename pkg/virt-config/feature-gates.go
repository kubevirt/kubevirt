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

func (config *ClusterConfig) isFeatureGateEnabled(featureGate string) bool {
	if fg := featuregate.FeatureGateInfo(featureGate); fg != nil && fg.State == featuregate.GA {
		return true
	}

	if config.isFeatureGateDefined(featureGate) {
		return true
	}
	return false
}

func (config *ClusterConfig) ExpandDisksEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.ExpandDisksGate)
}

func (config *ClusterConfig) CPUManagerEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.CPUManager)
}

func (config *ClusterConfig) NUMAEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.NUMAFeatureGate)
}

func (config *ClusterConfig) DownwardMetricsEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.DownwardMetricsFeatureGate)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.LiveMigrationGate)
}

func (config *ClusterConfig) SRIOVLiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.SRIOVLiveMigrationGate)
}

func (config *ClusterConfig) HypervStrictCheckEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HypervStrictCheckGate)
}

func (config *ClusterConfig) CPUNodeDiscoveryEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.CPUNodeDiscoveryGate)
}

func (config *ClusterConfig) SidecarEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.SidecarGate)
}

func (config *ClusterConfig) GPUPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.GPUGate)
}

func (config *ClusterConfig) SnapshotEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.SnapshotGate)
}

func (config *ClusterConfig) VMExportEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VMExportGate)
}

func (config *ClusterConfig) HotplugVolumesEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HotplugVolumesGate)
}

func (config *ClusterConfig) HostDiskEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HostDiskGate)
}

func (config *ClusterConfig) OldVirtiofsEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VirtIOFSGate)
}

func (config *ClusterConfig) VirtiofsConfigVolumesEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VirtIOFSConfigVolumesGate)
}

func (config *ClusterConfig) VirtiofsStorageEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VirtIOFSStorageVolumeGate)
}

func (config *ClusterConfig) MacvtapEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.MacvtapGate)
}

func (config *ClusterConfig) PasstEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.PasstGate)
}

func (config *ClusterConfig) HostDevicesPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HostDevicesGate)
}

func (config *ClusterConfig) RootEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.Root)
}

func (config *ClusterConfig) WorkloadEncryptionSEVEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.WorkloadEncryptionSEV)
}

func (config *ClusterConfig) DockerSELinuxMCSWorkaroundEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.DockerSELinuxMCSWorkaround)
}

func (config *ClusterConfig) VSOCKEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VSOCKGate)
}

func (config *ClusterConfig) MediatedDevicesHandlingDisabled() bool {
	return config.isFeatureGateEnabled(featuregate.DisableMediatedDevicesHandling)
}

func (config *ClusterConfig) KubevirtSeccompProfileEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.KubevirtSeccompProfile)
}

func (config *ClusterConfig) HotplugNetworkInterfacesEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HotplugNetworkIfacesGate)
}

func (config *ClusterConfig) PersistentReservationEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.PersistentReservation)
}

func (config *ClusterConfig) MultiArchitectureEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.MultiArchitecture)
}

func (config *ClusterConfig) AlignCPUsEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.AlignCPUsGate)
}

func (config *ClusterConfig) ImageVolumeEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.ImageVolume)
}

func (config *ClusterConfig) VideoConfigEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.VideoConfig)
}

func (config *ClusterConfig) NodeRestrictionEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.NodeRestrictionGate)
}

func (config *ClusterConfig) ObjectGraphEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.ObjectGraph)
}

func (config *ClusterConfig) DeclarativeHotplugVolumesEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.DeclarativeHotplugVolumesGate)
}

func (config *ClusterConfig) SecureExecutionEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.SecureExecution)
}

func (config *ClusterConfig) PanicDevicesEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.PanicDevicesGate)
}

func (config *ClusterConfig) PasstIPStackMigrationEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.PasstIPStackMigration)
}

func (config *ClusterConfig) DecentralizedLiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration)
}

func (config *ClusterConfig) GPUsWithDRAGateEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.GPUsWithDRAGate)
}

func (config *ClusterConfig) HostDevicesWithDRAEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HostDevicesWithDRAGate)
}

func (config *ClusterConfig) HyperVLayeredEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.HyperVLayered)
}

func (config *ClusterConfig) IncrementalBackupEnabled() bool {
	return config.isFeatureGateEnabled(featuregate.IncrementalBackupGate)
}
