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

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// RenderConfig abstracts the cluster configuration needed by the rendering
// pipeline (defaults, mutations, network setup). It is satisfied by
// *virtconfig.ClusterConfig when running inside KubeVirt, or by a lightweight
// offline implementation when rendering without a cluster.
//
// The interface is deliberately minimal: it covers what pkg/defaults,
// pkg/virt-api/webhooks/mutating-webhook/mutators, and
// pkg/network/vmispec need. The TemplateService's broader config needs
// are encapsulated behind ManifestRenderer, not exposed here.
//
// GetConfig() is an escape hatch for sub-fields not yet abstracted into
// dedicated methods. As the interface matures, callers should migrate to
// specific getters and GetConfig() should shrink in usage.
type RenderConfig interface {
	// IsFeatureGateEnabled checks whether a named feature gate is active.
	IsFeatureGateEnabled(gate string) bool

	// GetMachineType returns the default machine type for the given architecture.
	GetMachineType(arch string) string
	// GetDefaultArchitecture returns the cluster's default CPU architecture.
	GetDefaultArchitecture() string
	// GetCPUModel returns the default CPU model.
	GetCPUModel() string
	// GetCPURequest returns the default CPU resource request for virt-launcher.
	GetCPURequest() *resource.Quantity

	// IsVMRolloutStrategyLiveUpdate returns whether the live-update rollout
	// strategy is in effect (controls hotplug defaults).
	IsVMRolloutStrategyLiveUpdate() bool
	// GetMaximumCpuSockets returns the maximum number of CPU sockets for hotplug.
	GetMaximumCpuSockets() uint32
	// GetMaxHotplugRatio returns the maximum hotplug ratio.
	GetMaxHotplugRatio() uint32
	// GetMaximumGuestMemory returns the maximum guest memory for hotplug.
	GetMaximumGuestMemory() *resource.Quantity

	// GetDefaultNetworkInterface returns the default network interface type.
	GetDefaultNetworkInterface() string
	// IsBridgeInterfaceOnPodNetworkEnabled returns whether bridge binding
	// on the pod network is allowed.
	IsBridgeInterfaceOnPodNetworkEnabled() bool

	// GetConfigFromKubeVirtCR returns the full KubeVirt CR (used by mutators
	// to read annotations such as EmulatorThreadCompleteToEvenParity).
	GetConfigFromKubeVirtCR() *virtv1.KubeVirt
	// GetQGSSocketPath returns the QGS (Quote Generation Service) socket path
	// for TDX support.
	GetQGSSocketPath() string

	// GetConfig returns the full KubeVirtConfiguration. This is an escape
	// hatch for fields not yet exposed as dedicated methods (e.g.
	// EvictionStrategy, SeccompConfiguration). Prefer specific methods
	// when available.
	GetConfig() *virtv1.KubeVirtConfiguration

	// Methods needed by the TemplateService rendering pipeline.
	AllowEmulation() bool
	GetOVMFPath(arch string) string
	GetDiskVerification() *virtv1.DiskVerification
	GetHypervisor() *virtv1.HypervisorConfiguration
	GetVirtLauncherVerbosity() uint
	GetSELinuxLauncherType() string
	GetDefaultRuntimeClass() string
	GetNodeSelectors() map[string]string
	GetImagePullPolicy() k8sv1.PullPolicy
	GetNetworkBindings() map[string]virtv1.InterfaceBindingPlugin
	GetMemoryOvercommit() int
	GetCPUAllocationRatio() int
	GetClusterCPUArch() string
	GetSupportContainerRequest(typeName virtv1.SupportContainerType, resourceName k8sv1.ResourceName) *resource.Quantity
	GetSupportContainerLimit(typeName virtv1.SupportContainerType, resourceName k8sv1.ResourceName) *resource.Quantity
	GetPermittedHostDevices() *virtv1.PermittedHostDevices
	IsSerialConsoleLogDisabled() bool
}

// ManifestRenderer renders a Pod manifest from a VirtualMachineInstance.
// It is satisfied by *services.TemplateService when running inside KubeVirt.
// For offline rendering, the render package constructs a TemplateService
// internally with stub caches and nil client.
type ManifestRenderer interface {
	RenderLaunchManifest(vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

const defaultQGSSocketPath = "/var/run/tdx-qgs/qgs.socket"

type offlineRenderConfig struct {
	config *virtv1.KubeVirtConfiguration
}

func newOfflineRenderConfig(opts Options) *offlineRenderConfig {
	cpuRequest := resource.MustParse("100m")
	diskVerifLimit := resource.NewQuantity(2000*1024*1024, resource.BinarySI)

	config := &virtv1.KubeVirtConfiguration{
		DeveloperConfiguration: &virtv1.DeveloperConfiguration{
			FeatureGates:       opts.FeatureGates,
			MemoryOvercommit:   100,
			CPUAllocationRatio: 10,
			DiskVerification: &virtv1.DiskVerification{
				MemoryLimit: diskVerifLimit,
			},
			LogVerbosity: &virtv1.LogVerbosity{
				VirtLauncher: 2,
			},
		},
		CPURequest:      &cpuRequest,
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		ArchitectureConfiguration: &virtv1.ArchConfiguration{
			DefaultArchitecture: runtime.GOARCH,
			Amd64: &virtv1.ArchSpecificConfiguration{
				MachineType: "q35",
				OVMFPath:    "/usr/share/edk2/ovmf",
			},
			Arm64: &virtv1.ArchSpecificConfiguration{
				MachineType: "virt",
				OVMFPath:    "/usr/share/AAVMF",
			},
			S390x: &virtv1.ArchSpecificConfiguration{
				MachineType: "s390-ccw-virtio",
			},
		},
		NetworkConfiguration: &virtv1.NetworkConfiguration{
			NetworkInterface:                  "bridge",
			PermitBridgeInterfaceOnPodNetwork: pointer.P(true),
		},
		LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{
			MaxHotplugRatio: 4,
		},
		VMRolloutStrategy: pointer.P(virtv1.VMRolloutStrategyLiveUpdate),
		VirtualMachineOptions: &virtv1.VirtualMachineOptions{
			DisableSerialConsoleLog: &virtv1.DisableSerialConsoleLog{},
		},
	}

	return &offlineRenderConfig{config: config}
}

func (c *offlineRenderConfig) IsFeatureGateEnabled(gate string) bool {
	return featuregate.IsEnabled(gate, c.config.DeveloperConfiguration)
}

func (c *offlineRenderConfig) GetMachineType(arch string) string {
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

func (c *offlineRenderConfig) GetDefaultArchitecture() string {
	return c.config.ArchitectureConfiguration.DefaultArchitecture
}

func (c *offlineRenderConfig) GetCPUModel() string {
	return c.config.CPUModel
}

func (c *offlineRenderConfig) GetCPURequest() *resource.Quantity {
	return c.config.CPURequest
}

func (c *offlineRenderConfig) IsVMRolloutStrategyLiveUpdate() bool {
	return c.config.VMRolloutStrategy == nil || *c.config.VMRolloutStrategy == virtv1.VMRolloutStrategyLiveUpdate
}

func (c *offlineRenderConfig) GetMaximumCpuSockets() uint32 {
	if c.config.LiveUpdateConfiguration != nil && c.config.LiveUpdateConfiguration.MaxCpuSockets != nil {
		return *c.config.LiveUpdateConfiguration.MaxCpuSockets
	}
	return 0
}

func (c *offlineRenderConfig) GetMaxHotplugRatio() uint32 {
	if c.config.LiveUpdateConfiguration == nil {
		return 1
	}
	return c.config.LiveUpdateConfiguration.MaxHotplugRatio
}

func (c *offlineRenderConfig) GetMaximumGuestMemory() *resource.Quantity {
	if c.config.LiveUpdateConfiguration != nil {
		return c.config.LiveUpdateConfiguration.MaxGuest
	}
	return nil
}

func (c *offlineRenderConfig) GetDefaultNetworkInterface() string {
	return c.config.NetworkConfiguration.NetworkInterface
}

func (c *offlineRenderConfig) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return *c.config.NetworkConfiguration.PermitBridgeInterfaceOnPodNetwork
}

func (c *offlineRenderConfig) GetConfigFromKubeVirtCR() *virtv1.KubeVirt {
	return nil
}

func (c *offlineRenderConfig) GetQGSSocketPath() string {
	cfg := c.config.ConfidentialCompute
	if cfg == nil || cfg.TDX == nil || cfg.TDX.Attestation == nil || cfg.TDX.Attestation.QgsSocketPath == nil {
		return defaultQGSSocketPath
	}
	return *cfg.TDX.Attestation.QgsSocketPath
}

func (c *offlineRenderConfig) GetConfig() *virtv1.KubeVirtConfiguration {
	return c.config
}

func (c *offlineRenderConfig) AllowEmulation() bool {
	return c.config.DeveloperConfiguration != nil && c.config.DeveloperConfiguration.UseEmulation
}

func (c *offlineRenderConfig) GetOVMFPath(arch string) string {
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

func (c *offlineRenderConfig) GetDiskVerification() *virtv1.DiskVerification {
	return c.config.DeveloperConfiguration.DiskVerification
}

func (c *offlineRenderConfig) GetHypervisor() *virtv1.HypervisorConfiguration {
	if c.IsFeatureGateEnabled("ConfigurableHypervisor") && len(c.config.Hypervisors) > 0 {
		return &c.config.Hypervisors[0]
	}
	return &virtv1.HypervisorConfiguration{Name: virtv1.KvmHypervisorName}
}

func (c *offlineRenderConfig) GetVirtLauncherVerbosity() uint {
	if c.config.DeveloperConfiguration != nil && c.config.DeveloperConfiguration.LogVerbosity != nil {
		return c.config.DeveloperConfiguration.LogVerbosity.VirtLauncher
	}
	return 2
}

func (c *offlineRenderConfig) GetSELinuxLauncherType() string {
	return c.config.SELinuxLauncherType
}

func (c *offlineRenderConfig) GetDefaultRuntimeClass() string {
	return c.config.DefaultRuntimeClass
}

func (c *offlineRenderConfig) GetNodeSelectors() map[string]string {
	if c.config.DeveloperConfiguration != nil {
		return c.config.DeveloperConfiguration.NodeSelectors
	}
	return nil
}

func (c *offlineRenderConfig) GetImagePullPolicy() k8sv1.PullPolicy {
	return c.config.ImagePullPolicy
}

func (c *offlineRenderConfig) GetNetworkBindings() map[string]virtv1.InterfaceBindingPlugin {
	if c.config.NetworkConfiguration != nil {
		return c.config.NetworkConfiguration.Binding
	}
	return nil
}

func (c *offlineRenderConfig) GetMemoryOvercommit() int {
	return c.config.DeveloperConfiguration.MemoryOvercommit
}

func (c *offlineRenderConfig) GetCPUAllocationRatio() int {
	return c.config.DeveloperConfiguration.CPUAllocationRatio
}

func (c *offlineRenderConfig) GetClusterCPUArch() string {
	return runtime.GOARCH
}

func (c *offlineRenderConfig) GetSupportContainerRequest(_ virtv1.SupportContainerType, _ k8sv1.ResourceName) *resource.Quantity {
	return nil
}

func (c *offlineRenderConfig) GetSupportContainerLimit(_ virtv1.SupportContainerType, _ k8sv1.ResourceName) *resource.Quantity {
	return nil
}

func (c *offlineRenderConfig) GetPermittedHostDevices() *virtv1.PermittedHostDevices {
	return c.config.PermittedHostDevices
}

func (c *offlineRenderConfig) IsSerialConsoleLogDisabled() bool {
	return c.config.VirtualMachineOptions != nil && c.config.VirtualMachineOptions.DisableSerialConsoleLog != nil
}
