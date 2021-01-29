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
	CPUManager            = "CPUManager"
	IgnitionGate          = "ExperimentalIgnitionSupport"
	LiveMigrationGate     = "LiveMigration"
	CPUNodeDiscoveryGate  = "CPUNodeDiscovery"
	HypervStrictCheckGate = "HypervStrictCheck"
	SidecarGate           = "Sidecar"
	GPUGate               = "GPU"
	HostDevicesGate       = "HostDevices"
	SnapshotGate          = "Snapshot"
	HotplugVolumesGate    = "HotplugVolumes"
	HostDiskGate          = "HostDisk"
	VirtIOFSGate          = "ExperimentalVirtiofsSupport"
	MacvtapGate           = "Macvtap"
)

func (c *ClusterConfig) isFeatureGateEnabled(featureGate string) bool {
	for _, fg := range c.GetConfig().DeveloperConfiguration.FeatureGates {
		if fg == featureGate {
			return true
		}
	}
	return false
}

func (config *ClusterConfig) CPUManagerEnabled() bool {
	return config.isFeatureGateEnabled(CPUManager)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.isFeatureGateEnabled(IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(LiveMigrationGate)
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

func (config *ClusterConfig) HostDevicesPassthroughEnabled() bool {
	return config.isFeatureGateEnabled(HostDevicesGate)
}
