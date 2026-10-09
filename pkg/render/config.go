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

package render

import (
	"runtime"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// offlineConfig is a ClusterConfig built from Options, with no informers
// or Kubernetes client. It satisfies the consumer-defined interfaces used
// by defaults, mutators, and TemplateService.
type offlineConfig struct {
	config *v1.KubeVirtConfiguration
}

var _ defaults.ClusterConfigProvider = (*offlineConfig)(nil)

func newOfflineConfig(opts Options) *offlineConfig {
	cpuRequest := resource.MustParse(virtconfig.DefaultCPURequest)
	diskVerifLimit := resource.NewQuantity(virtconfig.DefaultDiskVerificationMemoryLimitBytes, resource.BinarySI)

	return &offlineConfig{
		config: &v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates:         opts.FeatureGates,
				DisabledFeatureGates: opts.DisabledFeatureGates,
				MemoryOvercommit:     virtconfig.DefaultMemoryOvercommit,
				CPUAllocationRatio:   virtconfig.DefaultCPUAllocationRatio,
				DiskVerification: &v1.DiskVerification{
					MemoryLimit: diskVerifLimit,
				},
				LogVerbosity: &v1.LogVerbosity{
					VirtLauncher: virtconfig.DefaultVirtLauncherLogVerbosity,
				},
			},
			CPURequest:      &cpuRequest,
			ImagePullPolicy: virtconfig.DefaultImagePullPolicy,
			ArchitectureConfiguration: &v1.ArchConfiguration{
				DefaultArchitecture: targetArchitecture(opts),
				Amd64: &v1.ArchSpecificConfiguration{
					MachineType: virtconfig.DefaultAMD64MachineType,
					OVMFPath:    virtconfig.DefaultARCHOVMFPath,
				},
				Arm64: &v1.ArchSpecificConfiguration{
					MachineType: virtconfig.DefaultAARCH64MachineType,
					OVMFPath:    virtconfig.DefaultAARCH64OVMFPath,
				},
				S390x: &v1.ArchSpecificConfiguration{
					MachineType: virtconfig.DefaultS390XMachineType,
					OVMFPath:    virtconfig.DefaultS390xOVMFPath,
				},
			},
			NetworkConfiguration: &v1.NetworkConfiguration{
				NetworkInterface:                  virtconfig.DefaultNetworkInterface,
				PermitBridgeInterfaceOnPodNetwork: pointer.P(virtconfig.DefaultPermitBridgeInterfaceOnPodNetwork),
				Binding:                           opts.NetworkBindings,
			},
			LiveUpdateConfiguration: &v1.LiveUpdateConfiguration{
				MaxHotplugRatio: 4,
			},
			VMRolloutStrategy: pointer.P(v1.VMRolloutStrategyLiveUpdate),
			EvictionStrategy:  pointer.P(v1.EvictionStrategyNone),
		},
	}
}

func (c *offlineConfig) IsFeatureGateEnabled(gate string) bool {
	return featuregate.IsEnabled(gate, c.config.DeveloperConfiguration)
}

func (c *offlineConfig) GetMachineType(arch string) string {
	if c.config.MachineType != "" {
		return c.config.MachineType
	}
	switch arch {
	case "arm64":
		return c.config.ArchitectureConfiguration.Arm64.MachineType
	case "s390x":
		return c.config.ArchitectureConfiguration.S390x.MachineType
	default:
		return c.config.ArchitectureConfiguration.Amd64.MachineType
	}
}

func (c *offlineConfig) GetDefaultArchitecture() string {
	return c.config.ArchitectureConfiguration.DefaultArchitecture
}

func (c *offlineConfig) GetCPUModel() string {
	return c.config.CPUModel
}

func (c *offlineConfig) GetCPURequest() *resource.Quantity {
	return c.config.CPURequest
}

func (c *offlineConfig) IsVMRolloutStrategyLiveUpdate() bool {
	return c.config.VMRolloutStrategy == nil || *c.config.VMRolloutStrategy == v1.VMRolloutStrategyLiveUpdate
}

func (c *offlineConfig) GetMaximumCpuSockets() uint32 {
	if c.config.LiveUpdateConfiguration != nil && c.config.LiveUpdateConfiguration.MaxCpuSockets != nil {
		return *c.config.LiveUpdateConfiguration.MaxCpuSockets
	}
	return 0
}

func (c *offlineConfig) GetMaxHotplugRatio() uint32 {
	if c.config.LiveUpdateConfiguration == nil {
		return 1
	}
	return c.config.LiveUpdateConfiguration.MaxHotplugRatio
}

func (c *offlineConfig) GetMaximumGuestMemory() *resource.Quantity {
	if c.config.LiveUpdateConfiguration != nil {
		return c.config.LiveUpdateConfiguration.MaxGuest
	}
	return nil
}

func (c *offlineConfig) GetDefaultNetworkInterface() string {
	return c.config.NetworkConfiguration.NetworkInterface
}

func (c *offlineConfig) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return *c.config.NetworkConfiguration.PermitBridgeInterfaceOnPodNetwork
}

func targetArchitecture(opts Options) string {
	if opts.Architecture != "" {
		return opts.Architecture
	}
	return runtime.GOARCH
}

func (c *offlineConfig) GetEvictionStrategy() *v1.EvictionStrategy {
	return c.config.EvictionStrategy
}

func (c *offlineConfig) EmulatorThreadCompleteToEvenParity() bool {
	return false
}

func (c *offlineConfig) GetQGSSocketPath() string {
	cfg := c.config.ConfidentialCompute
	if cfg == nil || cfg.TDX == nil || cfg.TDX.Attestation == nil || cfg.TDX.Attestation.QgsSocketPath == nil {
		return virtconfig.DefaultQGSSocketPath
	}
	return *cfg.TDX.Attestation.QgsSocketPath
}

func (c *offlineConfig) GetConfig() *v1.KubeVirtConfiguration {
	return c.config
}

func (c *offlineConfig) AllowEmulation() bool {
	return c.config.DeveloperConfiguration != nil && c.config.DeveloperConfiguration.UseEmulation
}

func (c *offlineConfig) GetEmulationPolicy(vmi *v1.VirtualMachineInstance) v1.EmulationPolicy {
	if vmi != nil && vmi.Spec.EmulationPolicy != nil {
		return *vmi.Spec.EmulationPolicy
	}
	if c.config.EmulationPolicy != nil {
		return *c.config.EmulationPolicy
	}
	return v1.EmulationPolicyNone
}

func (c *offlineConfig) GetOVMFPath(arch string) string {
	if c.config.OVMFPath != "" {
		return c.config.OVMFPath
	}
	switch arch {
	case "arm64":
		return c.config.ArchitectureConfiguration.Arm64.OVMFPath
	case "s390x":
		return c.config.ArchitectureConfiguration.S390x.OVMFPath
	default:
		return c.config.ArchitectureConfiguration.Amd64.OVMFPath
	}
}

func (c *offlineConfig) GetDiskVerification() *v1.DiskVerification {
	return c.config.DeveloperConfiguration.DiskVerification
}

func (c *offlineConfig) GetHypervisor() *v1.HypervisorConfiguration {
	return virtconfig.GetHypervisorFromKvConfig(c.config, c.IsFeatureGateEnabled(featuregate.ConfigurableHypervisor))
}

func (c *offlineConfig) GetVirtLauncherVerbosity() uint {
	if c.config.DeveloperConfiguration != nil && c.config.DeveloperConfiguration.LogVerbosity != nil {
		return c.config.DeveloperConfiguration.LogVerbosity.VirtLauncher
	}
	return virtconfig.DefaultVirtLauncherLogVerbosity
}

func (c *offlineConfig) GetSELinuxLauncherType() string {
	return c.config.SELinuxLauncherType
}

func (c *offlineConfig) GetDefaultRuntimeClass() string {
	return c.config.DefaultRuntimeClass
}

func (c *offlineConfig) GetNodeSelectors() map[string]string {
	if c.config.DeveloperConfiguration != nil {
		return c.config.DeveloperConfiguration.NodeSelectors
	}
	return nil
}

func (c *offlineConfig) GetImagePullPolicy() k8sv1.PullPolicy {
	return c.config.ImagePullPolicy
}

func (c *offlineConfig) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	if c.config.NetworkConfiguration != nil {
		return c.config.NetworkConfiguration.Binding
	}
	return nil
}

func (c *offlineConfig) GetMemoryOvercommit() int {
	return c.config.DeveloperConfiguration.MemoryOvercommit
}

func (c *offlineConfig) GetCPUAllocationRatio() int {
	return c.config.DeveloperConfiguration.CPUAllocationRatio
}

func (c *offlineConfig) GetClusterCPUArch() string {
	return runtime.GOARCH
}

func (c *offlineConfig) GetSupportContainerRequest(v1.SupportContainerType, k8sv1.ResourceName) *resource.Quantity {
	return nil
}

func (c *offlineConfig) GetSupportContainerLimit(v1.SupportContainerType, k8sv1.ResourceName) *resource.Quantity {
	return nil
}

func (c *offlineConfig) GetPermittedHostDevices() *v1.PermittedHostDevices {
	return c.config.PermittedHostDevices
}

func (c *offlineConfig) IsSerialConsoleLogDisabled() bool {
	return c.config.VirtualMachineOptions != nil && c.config.VirtualMachineOptions.DisableSerialConsoleLog != nil
}
