package api

const (
	resolvConf        = "/etc/resolv.conf"
	DefaultProtocol   = "TCP"
	DefaultVMCIDR     = "10.0.2.0/24"
	DefaultVMIpv6CIDR = "fd10:0:2::/120"
	DefaultBridgeName = "k6t-eth0"
)

func NewDefaulter(arch string) *Defaulter {
	return &Defaulter{Architecture: arch}
}

type Defaulter struct {
	Architecture string
}

func (d *Defaulter) SetDefaults_Devices(devices *Devices) {
	// Set default memballoon, "none" means that controller disabled
	devices.Ballooning = &Ballooning{
		Model: "none",
	}
}

func (d *Defaulter) SetDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		if d.Architecture == "ppc64le" {
			ostype.Arch = "ppc64le"
		} else {
			ostype.Arch = "x86_64"
		}
	}

	// q35 is an alias of the newest q35 machine type.
	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		if d.Architecture == "ppc64le" {
			ostype.Machine = "pseries"
		} else {
			ostype.Machine = "q35"
		}
	}
}

func (d *Defaulter) SetDefaults_DomainSpec(spec *DomainSpec) {
	spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	if spec.Type == "" {
		spec.Type = "kvm"
	}
}

func (d *Defaulter) SetDefaults_SysInfo(sysinfo *SysInfo) {
	if sysinfo.Type == "" {
		sysinfo.Type = "smbios"
	}
}

func (d *Defaulter) SetObjectDefaults_Domain(in *Domain) {
	d.SetDefaults_DomainSpec(&in.Spec)
	d.SetDefaults_OSType(&in.Spec.OS.Type)
	if in.Spec.SysInfo != nil {
		d.SetDefaults_SysInfo(in.Spec.SysInfo)
	}
	d.SetDefaults_Devices(&in.Spec.Devices)
}
