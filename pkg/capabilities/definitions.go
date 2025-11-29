package capabilities

import v1 "kubevirt.io/api/core/v1"

// Capability constants - each represents a feature that may need validation or blocking
const (
	CapVsock CapabilityKey = "domain.devices.vsock"
	// ... all capabilities declared as constants
)

// Define CapVsock capability
var CapVsockDef = Capability{
	IsRequiredBy: func(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
		return vmiSpec.Domain.Devices.AutoattachVSOCK != nil && *vmiSpec.Domain.Devices.AutoattachVSOCK
	},
}
