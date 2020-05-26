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

import (
	"strings"
)

const (
	CPUManager            = "CPUManager"
	IgnitionGate          = "ExperimentalIgnitionSupport"
	liveMigrationGate     = "LiveMigration"
	CPUNodeDiscoveryGate  = "CPUNodeDiscovery"
	HypervStrictCheckGate = "HypervStrictCheck"
	SidecarGate           = "Sidecar"
	GPUGate               = "GPU"
)

func (c *ClusterConfig) isFeatureGateEnabled(featureGate string) bool {
	return strings.Contains(c.getConfig().FeatureGates, featureGate)
}

func (config *ClusterConfig) CPUManagerEnabled() bool {
	return config.isFeatureGateEnabled(CPUManager)
}

func (config *ClusterConfig) IgnitionEnabled() bool {
	return config.isFeatureGateEnabled(IgnitionGate)
}

func (config *ClusterConfig) LiveMigrationEnabled() bool {
	return config.isFeatureGateEnabled(liveMigrationGate)
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
