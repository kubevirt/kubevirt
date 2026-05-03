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

package capabilities

import (
	core_capabilities "kubevirt.io/kubevirt/pkg/capabilities/core"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func RegisterUniversalCapabilities() {
	// Register capability support levels for universal platforms
	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapVsock, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Vsock support is experimental on this platform.",
		GatedBy: featuregate.VSOCKGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapVirtiofsStorage, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "VirtioFS storage support is experimental on this platform.",
		GatedBy: featuregate.VirtIOFSStorageVolumeGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapDownwardMetricsVolume, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Downward metrics volume support is experimental on this platform.",
		GatedBy: featuregate.DownwardMetricsFeatureGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapDownwardMetricsDevice, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Downward metrics device support is experimental on this platform.",
		GatedBy: featuregate.DownwardMetricsFeatureGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapDeclarativeHotplugVolumes, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Declarative hotplug volumes support is experimental on this platform.",
		GatedBy: featuregate.DeclarativeHotplugVolumesGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapHostDevicesPassthrough, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Host devices passthrough support is experimental on this platform.",
		GatedBy: featuregate.HostDevicesGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapHostDisk, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Host disk support is experimental on this platform.",
		GatedBy: featuregate.HostDiskGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapIgnitionSupport, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Ignition support is experimental on this platform.",
		GatedBy: featuregate.IgnitionGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapSidecarHooks, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Sidecar hooks support is experimental on this platform.",
		GatedBy: featuregate.SidecarGate,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapPersistentReservation, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Persistent reservation support is experimental on this platform.",
		GatedBy: featuregate.PersistentReservation,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapVideoConfig, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Video configuration support is experimental on this platform.",
		GatedBy: featuregate.VideoConfig,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapRebootPolicy, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Reboot policy support is experimental on this platform.",
		GatedBy: featuregate.RebootPolicy,
	})

	core_capabilities.AddPlatformCapabilitySupport(core_capabilities.Universal, core_capabilities.CapReservedOverheadMemlock, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Experimental,
		Message: "Reserved overhead memlock support is experimental on this platform.",
		GatedBy: featuregate.ReservedOverheadMemlock,
	})
}

func init() {
	// This is a placeholder for capabilities that are not specific to any particular platform.
	RegisterUniversalCapabilities()
}
