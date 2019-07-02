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
 This module is intended for exposing the virtualization configuration that is availabe at the cluster-level and its default settings.
*/

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ParallelOutboundMigrationsPerNodeDefault uint32 = 2
	ParallelMigrationsPerClusterDefault      uint32 = 5
	BandwithPerMigrationDefault                     = "64Mi"
	MigrationAllowAutoConverge               bool   = false
	MigrationProgressTimeout                 int64  = 150
	MigrationCompletionTimeoutPerGiB         int64  = 800
	DefaultMachineType                              = "q35"
	DefaultCPURequest                               = "100m"
	DefaultMemoryOvercommit                         = 100
	DefaultEmulatedMachines                         = "q35*,pc-q35*"
	DefaultLessPVCSpaceToleration                   = 10
	DefaultNodeSelectors                            = ""
	DefaultNetworkInterface                         = "bridge"
	DefaultImagePullPolicy                          = k8sv1.PullIfNotPresent
	DefaultUseEmulation                             = false
	DefaultUnsafeMigrationOverride                  = false
	DefaultPermitSlirpInterface                     = false
)

func (c *ClusterConfig) IsUseEmulation() bool {
	return c.getConfig().UseEmulation
}

func (c *ClusterConfig) GetMigrationConfig() *MigrationConfig {
	return c.getConfig().MigrationConfig
}

func (c *ClusterConfig) GetImagePullPolicy() (policy k8sv1.PullPolicy) {
	return c.getConfig().ImagePullPolicy
}

func (c *ClusterConfig) GetMachineType() string {
	return c.getConfig().MachineType
}

func (c *ClusterConfig) GetCPUModel() string {
	return c.getConfig().CPUModel
}

func (c *ClusterConfig) GetCPURequest() resource.Quantity {
	return c.getConfig().CPURequest
}

func (c *ClusterConfig) GetMemoryOvercommit() int {
	return c.getConfig().MemoryOvercommit
}

func (c *ClusterConfig) GetEmulatedMachines() []string {
	return c.getConfig().EmulatedMachines
}

func (c *ClusterConfig) GetLessPVCSpaceToleration() int {
	return c.getConfig().LessPVCSpaceToleration
}

func (c *ClusterConfig) GetNodeSelectors() map[string]string {
	return c.getConfig().NodeSelectors
}

func (c *ClusterConfig) GetDefaultNetworkInterface() string {
	return c.getConfig().NetworkInterface
}

func (c *ClusterConfig) IsSlirpInterfaceEnabled() bool {
	return c.getConfig().PermitSlirpInterface
}
