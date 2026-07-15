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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	virtv1 "kubevirt.io/api/core/v1"
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
}

// ManifestRenderer renders a Pod manifest from a VirtualMachineInstance.
// It is satisfied by *services.TemplateService when running inside KubeVirt.
// For offline rendering, the render package constructs a TemplateService
// internally with stub caches and nil client.
type ManifestRenderer interface {
	RenderLaunchManifest(vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error)
}
