package capabilities

// Define a struct to hold a map from capability keys to their definitions
var CapabilityDefinitions = map[CapabilityKey]Capability{
	CapVsock: CapVsockDef,
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

	// First overlay hypervisor-specific capabilities
	platformKey := Platform(hypervisor)
	if hypervisorSupports, exists := PlatformCapabilitySupport[platformKey]; exists {
		for capKey, capSupport := range hypervisorSupports {
			supports[capKey] = capSupport
		}
	}

	// Then overlay hypervisor+arch-specific capabilities
	platformArchKey := Platform(hypervisor + "/" + arch)
	if hypervisorArchSupports, exists := PlatformCapabilitySupport[platformArchKey]; exists {
		for capKey, capSupport := range hypervisorArchSupports {
			supports[capKey] = capSupport
		}
	}

	return supports
}
