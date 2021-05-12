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
 This module is intended for exposing the virtualization configuration that is available at the cluster-level and its default settings.
*/

import (
	"runtime"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
)

const (
	ParallelOutboundMigrationsPerNodeDefault uint32 = 2
	ParallelMigrationsPerClusterDefault      uint32 = 5
	BandwithPerMigrationDefault                     = "64Mi"
	MigrationAllowAutoConverge               bool   = false
	MigrationAllowPostCopy                   bool   = false
	MigrationProgressTimeout                 int64  = 150
	MigrationCompletionTimeoutPerGiB         int64  = 800
	DefaultAMD64MachineType                         = "q35"
	DefaultPPC64LEMachineType                       = "pseries"
	DefaultAARCH64MachineType                       = "virt"
	DefaultCPURequest                               = "100m"
	DefaultMemoryOvercommit                         = 100
	DefaultAMD64EmulatedMachines                    = "q35*,pc-q35*"
	DefaultPPC64LEEmulatedMachines                  = "pseries*"
	DefaultAARCH64EmulatedMachines                  = "virt*"
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
	DefaultSELinuxLauncherType                      = "virt_launcher.process"
	SupportedGuestAgentVersions                     = "2.*,3.*,4.*,5.*"
	DefaultARCHOVMFPath                             = "/usr/share/OVMF"
	DefaultAARCH64OVMFPath                          = "/usr/share/AAVMF"
	DefaultMemBalloonStatsPeriod             uint32 = 10
	DefaultCPUAllocationRatio                       = 10
	DefaultVirtAPILogVerbosity                      = 2
	DefaultVirtControllerLogVerbosity               = 2
	DefaultVirtHandlerLogVerbosity                  = 2
	DefaultVirtLauncherLogVerbosity                 = 2
	DefaultVirtOperatorLogVerbosity                 = 2
)

type DefaultArch struct {
	Architecture string
}

func NewDefaultArch(arch string) *DefaultArch {
	return &DefaultArch{Architecture: arch}
}

func (d *DefaultArch) IsARM64() bool {
	if d.Architecture == "arm64" {
		return true
	}
	return false
}

func (d *DefaultArch) IsPPC64() bool {
	if d.Architecture == "ppc64le" {
		return true
	}
	return false
}

// GetDefaultMachinesForArch set default machine type and supported emulated machines based on architecture
func (d *DefaultArch) GetDefaultMachinesForArch() (string, string) {
	if d.IsPPC64() {
		return DefaultPPC64LEMachineType, DefaultPPC64LEEmulatedMachines
	}
	if d.IsARM64() {
		return DefaultAARCH64MachineType, DefaultAARCH64EmulatedMachines
	}
	return DefaultAMD64MachineType, DefaultAMD64EmulatedMachines
}

var DefaultMachineType, DefaultEmulatedMachines = NewDefaultArch(runtime.GOARCH).GetDefaultMachinesForArch()

func (c *ClusterConfig) GetMemBalloonStatsPeriod() uint32 {
	return *c.GetConfig().MemBalloonStatsPeriod
}

//GetDefaultOVMFPathForArch set default EFI bootloader Path based on architecture
func (d *DefaultArch) GetDefaultOVMFPathForArch() string {
	if d.IsARM64() {
		return DefaultAARCH64OVMFPath
	}
	return DefaultARCHOVMFPath
}

var DefaultOVMFPath = NewDefaultArch(runtime.GOARCH).GetDefaultOVMFPathForArch()

func (c *ClusterConfig) IsUseEmulation() bool {
	return c.GetConfig().DeveloperConfiguration.UseEmulation
}

func (c *ClusterConfig) GetMigrationConfiguration() *v1.MigrationConfiguration {
	return c.GetConfig().MigrationConfiguration
}

func (c *ClusterConfig) GetImagePullPolicy() (policy k8sv1.PullPolicy) {
	return c.GetConfig().ImagePullPolicy
}

func (c *ClusterConfig) GetResourceVersion() string {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lastValidConfigResourceVersion
}

func (c *ClusterConfig) GetMachineType() string {
	return c.GetConfig().MachineType
}

func (c *ClusterConfig) GetCPUModel() string {
	return c.GetConfig().CPUModel
}

func (c *ClusterConfig) GetCPURequest() *resource.Quantity {
	return c.GetConfig().CPURequest
}

func (c *ClusterConfig) GetMemoryOvercommit() int {
	return c.GetConfig().DeveloperConfiguration.MemoryOvercommit
}

func (c *ClusterConfig) GetEmulatedMachines() []string {
	return c.GetConfig().EmulatedMachines
}

func (c *ClusterConfig) GetLessPVCSpaceToleration() int {
	return c.GetConfig().DeveloperConfiguration.LessPVCSpaceToleration
}

func (c *ClusterConfig) GetNodeSelectors() map[string]string {
	return c.GetConfig().DeveloperConfiguration.NodeSelectors
}

func (c *ClusterConfig) GetDefaultNetworkInterface() string {
	return c.GetConfig().NetworkConfiguration.NetworkInterface
}

func (c *ClusterConfig) IsSlirpInterfaceEnabled() bool {
	return *c.GetConfig().NetworkConfiguration.PermitSlirpInterface
}

func (c *ClusterConfig) GetSMBIOS() *v1.SMBiosConfiguration {
	return c.GetConfig().SMBIOSConfig
}

func (c *ClusterConfig) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return *c.GetConfig().NetworkConfiguration.PermitBridgeInterfaceOnPodNetwork
}

func (c *ClusterConfig) GetDefaultClusterConfig() *v1.KubeVirtConfiguration {
	return c.defaultConfig
}

func (c *ClusterConfig) GetSELinuxLauncherType() string {
	return c.GetConfig().SELinuxLauncherType
}

func (c *ClusterConfig) GetSupportedAgentVersions() []string {
	return c.GetConfig().SupportedGuestAgentVersions
}

func (c *ClusterConfig) GetOVMFPath() string {
	return c.GetConfig().OVMFPath
}

func (c *ClusterConfig) GetCPUAllocationRatio() int {
	return c.GetConfig().DeveloperConfiguration.CPUAllocationRatio
}

func (c *ClusterConfig) GetPermittedHostDevices() *v1.PermittedHostDevices {
	return c.GetConfig().PermittedHostDevices
}

func (c *ClusterConfig) GetVirtHandlerVerbosity(nodeName string) uint {
	logConf := c.GetConfig().DeveloperConfiguration.LogVerbosity
	if level := logConf.NodeVerbosity[nodeName]; level != 0 {
		return level
	}
	return logConf.VirtHandler
}

func (c *ClusterConfig) GetVirtAPIVerbosity(nodeName string) uint {
	logConf := c.GetConfig().DeveloperConfiguration.LogVerbosity
	if level := logConf.NodeVerbosity[nodeName]; level != 0 {
		return level
	}
	return logConf.VirtAPI
}

func (c *ClusterConfig) GetVirtControllerVerbosity(nodeName string) uint {
	logConf := c.GetConfig().DeveloperConfiguration.LogVerbosity
	if level := logConf.NodeVerbosity[nodeName]; level != 0 {
		return level
	}
	return logConf.VirtController
}

func (c *ClusterConfig) GetVirtLauncherVerbosity() uint {
	logConf := c.GetConfig().DeveloperConfiguration.LogVerbosity
	return logConf.VirtLauncher
}

//GetMinCPUModel return minimal cpu which is used in node-labeller
func (c *ClusterConfig) GetMinCPUModel() string {
	return c.GetConfig().MinCPUModel
}

//GetObsoleteCPUModels return slice of obsolete cpus which are used in node-labeller
func (c *ClusterConfig) GetObsoleteCPUModels() map[string]bool {
	return c.GetConfig().ObsoleteCPUModels
}
