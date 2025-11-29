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
	Universal Platform = ""
	KVM       Platform = "kvm"
	KVM_AMD64 Platform = "kvm/amd64"
	KVM_S390X Platform = "kvm/s390x"
	KVM_ARM64 Platform = "kvm/arm64"
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
