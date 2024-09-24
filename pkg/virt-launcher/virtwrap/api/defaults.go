package api

const (
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

func (d *Defaulter) IsPPC64() bool {
	return d.Architecture == "ppc64le"
}

func (d *Defaulter) IsARM64() bool {
	return d.Architecture == "arm64"
}

func (d *Defaulter) IsS390X() bool {
	return d.Architecture == "s390x"
}

func (d *Defaulter) SetDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		switch {
		case d.IsPPC64():
			ostype.Arch = "ppc64le"
		case d.IsARM64():
			ostype.Arch = "aarch64"
		case d.IsS390X():
			ostype.Arch = "s390x"
		default:
			ostype.Arch = "x86_64"
		}
	}

	// q35 is an alias of the newest q35 machine type.
	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		switch {
		case d.IsPPC64():
			ostype.Machine = "pseries"
		case d.IsARM64():
			ostype.Machine = "virt"
		case d.IsS390X():
			ostype.Machine = "s390-ccw-virtio"
		default:
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
}
