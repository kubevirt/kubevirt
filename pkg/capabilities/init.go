package capabilities

import "kubevirt.io/kubevirt/pkg/virt-config/featuregate"

// Function to register all capabilities universal to KubeVirt
func RegisterUniversalCapabilities() {
	// Register CapVsock support levels for different platforms
	AddPlatformCapabilitySupport(Universal, CapVsock, CapabilitySupport{
		Level:   Experimental,
		Message: "Vsock support is experimental on this platform.",
		GatedBy: featuregate.VSOCKGate,
	})
}

// Function to register all capabilities and their support levels
func Init() {
	RegisterUniversalCapabilities()
	RegisterKvmCapabilities()
	RegisterMshvCapabilities()
}
