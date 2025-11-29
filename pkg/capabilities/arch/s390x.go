package arch_capabilities

import (
	core_capabilities "kubevirt.io/kubevirt/pkg/capabilities/core"
)

func RegisterS390xCapabilities() {
	// Register capability support levels for S390x architecture
	platformKey := core_capabilities.PlatformKeyFromArch("s390x")
	core_capabilities.AddPlatformCapabilitySupport(platformKey, core_capabilities.CapPanicDevices, core_capabilities.CapabilitySupport{
		Level:   core_capabilities.Unsupported,
		Message: "PanicDevices are unsupported on this platform.",
	})
}
