package api

const DefaultBridgeName = "br1"

func SetDefaults_Devices(devices *Devices) {
	// Use vga as video device, since it is better than cirrus
	// and does not require guest drivers
	var heads uint = 1
	var vram uint = 16384
	devices.Video = []Video{
		{
			Model: VideoModel{
				Type:  "vga",
				Heads: &heads,
				VRam:  &vram,
			},
		},
	}

	// For now connect every virtual machine to the default network
	devices.Interfaces = []Interface{{
		Model: &Model{
			Type: "e1000",
		},
		Type: "bridge",
		Source: InterfaceSource{
			Bridge: DefaultBridgeName,
		}},
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
