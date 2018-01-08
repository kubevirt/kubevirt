package api

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
		Type: "network",
		Source: InterfaceSource{
			Network: "default",
		}},
	}
}

func SetDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"
}

func SetDefaults_DomainSpec(spec *DomainSpec) {
	spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	spec.Type = "qemu"
	spec.CPU.Mode = "host-model"
}

func SetDefaults_SysInfo(sysinfo *SysInfo) {
	sysinfo.Type = "smbios"
}
