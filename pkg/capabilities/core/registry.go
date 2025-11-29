package capabilities

// Define a struct to hold a map from capability keys to their definitions
var CapabilityDefinitions = map[CapabilityKey]Capability{
	CapVsock:        CapVsockDef,
	CapPanicDevices: CapPanicDevicesDef,
	// Add other capabilities here as they are defined
}

// Define a struct to hold a map from platform information to the support levels of capabilities
var PlatformCapabilitySupport = map[Platform]map[CapabilityKey]CapabilitySupport{}

// Define a function to add support information for a specific capability key for a specific platform
func AddPlatformCapabilitySupport(platform Platform, capabilityKey CapabilityKey, support CapabilitySupport) {
	if PlatformCapabilitySupport[platform] == nil {
		PlatformCapabilitySupport[platform] = make(map[CapabilityKey]CapabilitySupport)
	}
	PlatformCapabilitySupport[platform][capabilityKey] = support
}

// Function to return the support information for all capabilities for a given hypervisor and architecture
func GetCapabilitiesSupportForPlatform(hypervisor, arch string) map[CapabilityKey]CapabilitySupport {
	supports := make(map[CapabilityKey]CapabilitySupport)

	// Start with universal capabilities
	if universalSupports, exists := PlatformCapabilitySupport[Universal]; exists {
		for capKey, capSupport := range universalSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay hypervisor-specific capabilities
	platformHypervisorKey := Platform(PlatformKeyFromHypervisor(hypervisor))
	if hypervisorSupports, exists := PlatformCapabilitySupport[platformHypervisorKey]; exists {
		for capKey, capSupport := range hypervisorSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay architecture-specific capabilities
	platformArchKey := Platform(PlatformKeyFromArch(arch))
	if archSupports, exists := PlatformCapabilitySupport[platformArchKey]; exists {
		for capKey, capSupport := range archSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay hypervisor+arch-specific capabilities
	platformHypervisorArchKey := Platform(PlatformKeyFromHypervisorAndArch(hypervisor, arch))
	if hypervisorArchSupports, exists := PlatformCapabilitySupport[platformHypervisorArchKey]; exists {
		for capKey, capSupport := range hypervisorArchSupports {
			supports[capKey] = capSupport
		}
	}

	return supports
}
