package capabilities

import (
	arch_capabilities "kubevirt.io/kubevirt/pkg/capabilities/arch"
	hypervisor_capabilities "kubevirt.io/kubevirt/pkg/capabilities/hypervisor"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

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

	hypervisor_capabilities.RegisterKvmCapabilities()
	hypervisor_capabilities.RegisterMshvCapabilities()

	arch_capabilities.RegisterAmd64Capabilities()
	arch_capabilities.RegisterArm64Capabilities()
	arch_capabilities.RegisterS390xCapabilities()
}
