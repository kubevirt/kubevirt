package capabilities

import v1 "kubevirt.io/api/core/v1"

type CapabilityKey string // e.g., "graphics.vga", "firmware.secureboot.uefi"
type SupportLevel int

const (
	Unregistered SupportLevel = iota // Not registered (default zero value)
	Unsupported                      // Explicitly blocked on this platform
	Experimental                     // Requires feature gate
	Deprecated                       // Supported but discouraged
)

type Platform string

const (
	Universal Platform = "" // Applies to all platforms
)

type Capability struct {
	// function to check if this capability is required by a given VMI
	IsRequiredBy func(vmi *v1.VirtualMachineInstanceSpec) bool
}

// struct to store the extent to which a given capability is supported
type CapabilitySupport struct {
	Level   SupportLevel
	Message string // User-facing explanation
	GatedBy string // Optional: feature gate name
}

func PlatformKeyFromHypervisor(hypervisor string) Platform {
	return Platform(hypervisor + "/")
}

func PlatformKeyFromArch(arch string) Platform {
	return Platform("/" + arch)
}

func PlatformKeyFromHypervisorAndArch(hypervisor, arch string) Platform {
	return Platform(hypervisor + "/" + arch)
}
