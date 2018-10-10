package api

const (
	resolvConf        = "/etc/resolv.conf"
	DefaultProtocol   = "TCP"
	DefaultVMCIDR     = "10.0.2.0/24"
	DefaultBridgeName = "k6t-eth0"
)

func SetDefaults_Devices(devices *Devices) {
	// Set default controllers, "none" means that controller disabled
	devices.Controllers = []Controller{
		{
			Type:  "usb",
			Index: "0",
			Model: "none",
		},
	}
	// Set default memballoon, "none" means that controller disabled
	devices.Ballooning = &Ballooning{
		Model: "none",
	}

}

func SetDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		ostype.Arch = "x86_64"
	}

	// q35 is an alias of the newest q35 machine type.
	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		ostype.Machine = "q35"
	}
}

func SetDefaults_DomainSpec(spec *DomainSpec) {
	spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	if spec.Type == "" {
		spec.Type = "kvm"
	}
}

func SetDefaults_SysInfo(sysinfo *SysInfo) {
	sysinfo.Type = "smbios"
}
