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

package core

// Map from capability keys to their definitions
var capabilityDefinitions = map[CapabilityKey]Capability{
	// Experimental capabilities guarded by feature gates
	CapVsock:                     CapVsockDef,
	CapVirtiofsStorage:           CapVirtiofsStorageDef,
	CapDownwardMetricsVolume:     CapDownwardMetricsVolumeDef,
	CapDownwardMetricsDevice:     CapDownwardMetricsDeviceDef,
	CapDeclarativeHotplugVolumes: CapDeclarativeHotplugVolumesDef,
	CapNUMAGuestMapping:          CapNUMAGuestMappingDef,
	CapHostDevicesPassthrough:    CapHostDevicesPassthroughDef,
	CapHostDisk:                  CapHostDiskDef,
	CapIgnitionSupport:           CapIgnitionSupportDef,
	CapSidecarHooks:              CapSidecarHooksDef,
	CapRebootPolicy:              CapRebootPolicyDef,
	CapReservedOverheadMemlock:   CapReservedOverheadMemlockDef,
}

// Getter function to retrieve the definition of a capability by its key
func GetCapabilityDefinition(capabilityKey CapabilityKey) (Capability, bool) {
	capability, exists := capabilityDefinitions[capabilityKey]
	return capability, exists
}

// Map from platform information to the support levels of capabilities
var platformCapabilitySupport = map[Platform]map[CapabilityKey]CapabilitySupport{}

// Function to add support information for a specific capability key for a specific platform
func AddPlatformCapabilitySupport(platform Platform, capabilityKey CapabilityKey, support CapabilitySupport) {
	if platformCapabilitySupport[platform] == nil {
		platformCapabilitySupport[platform] = make(map[CapabilityKey]CapabilitySupport)
	}

	if _, exists := platformCapabilitySupport[platform][capabilityKey]; exists {
		// Throw an error if the capabilityKey is already defined for the platform.
		// This is to prevent accidental overwriting of existing support information.
		panic("Capability support for " + string(capabilityKey) + " already defined for platform " + string(platform))
	}

	platformCapabilitySupport[platform][capabilityKey] = support
}

// Function to retrieve all capabilities and their support level for all platforms
func GetAllPlatformCapabilitySupport() map[Platform]map[CapabilityKey]CapabilitySupport {
	return platformCapabilitySupport
}

// Function to return the support information for all capabilities for a given hypervisor and architecture
func GetCapabilitiesSupportForPlatform(hypervisor, arch string) map[CapabilityKey]CapabilitySupport {
	supports := make(map[CapabilityKey]CapabilitySupport)

	// Start with universal capabilities
	if universalSupports, exists := platformCapabilitySupport[Universal]; exists {
		for capKey, capSupport := range universalSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay hypervisor-specific capabilities
	platformHypervisorKey := Platform(PlatformKeyFromHypervisor(hypervisor))
	if hypervisorSupports, exists := platformCapabilitySupport[platformHypervisorKey]; exists {
		for capKey, capSupport := range hypervisorSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay architecture-specific capabilities
	platformArchKey := Platform(PlatformKeyFromArch(arch))
	if archSupports, exists := platformCapabilitySupport[platformArchKey]; exists {
		for capKey, capSupport := range archSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay hypervisor+arch-specific capabilities
	platformHypervisorArchKey := Platform(PlatformKeyFromHypervisorAndArch(hypervisor, arch))
	if hypervisorArchSupports, exists := platformCapabilitySupport[platformHypervisorArchKey]; exists {
		for capKey, capSupport := range hypervisorArchSupports {
			supports[capKey] = capSupport
		}
	}

	return supports
}
