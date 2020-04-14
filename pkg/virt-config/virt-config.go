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
	"runtime"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

const (
	ParallelOutboundMigrationsPerNodeDefault uint32 = 2
	ParallelMigrationsPerClusterDefault      uint32 = 5
	BandwithPerMigrationDefault                     = "64Mi"
	MigrationAllowAutoConverge               bool   = false
	MigrationProgressTimeout                 int64  = 150
	MigrationCompletionTimeoutPerGiB         int64  = 800
	DefaultAMD64MachineType                         = "q35"
	DefaultPPC64LEMachineType                       = "pseries"
	DefaultCPURequest                               = "100m"
	DefaultMemoryOvercommit                         = 100
	DefaultAMD64EmulatedMachines                    = "q35*,pc-q35*"
	DefaultPPC64LEEmulatedMachines                  = "pseries*"
	DefaultLessPVCSpaceToleration                   = 10
	DefaultNodeSelectors                            = ""
	DefaultNetworkInterface                         = "bridge"
	DefaultImagePullPolicy                          = k8sv1.PullIfNotPresent
	DefaultUseEmulation                             = false
	DefaultUnsafeMigrationOverride                  = false
	DefaultPermitSlirpInterface                     = false
	SmbiosConfigDefaultFamily                       = "KubeVirt"
	SmbiosConfigDefaultManufacturer                 = "KubeVirt"
	SmbiosConfigDefaultProduct                      = "None"
	DefaultPermitBridgeInterfaceOnPodNetwork        = true
	DefaultSELinuxLauncherType                      = ""
	SupportedGuestAgentVersions                     = "3.*,4.*"
)

// Set default machine type and supported emulated machines based on architecture
func getDefaultMachinesForArch() (string, string) {
	if runtime.GOARCH == "ppc64le" {
		return DefaultPPC64LEMachineType, DefaultPPC64LEEmulatedMachines
	}
	return DefaultAMD64MachineType, DefaultAMD64EmulatedMachines
}

var DefaultMachineType, DefaultEmulatedMachines = getDefaultMachinesForArch()

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

func (c *ClusterConfig) GetSMBIOS() *cmdv1.SMBios {
	return c.getConfig().SmbiosConfig
}

func (c *ClusterConfig) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return c.getConfig().PermitBridgeInterfaceOnPodNetwork
}

func (c *ClusterConfig) GetSELinuxLauncherType() string {
	return c.getConfig().SELinuxLauncherType
}

func (c *ClusterConfig) GetSupportedAgentVersions() []string {
	return c.getConfig().SupportedGuestAgentVersions
}
