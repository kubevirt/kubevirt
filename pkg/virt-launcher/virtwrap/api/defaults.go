package api

import archdefaulter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api/arch-defaulter"

const (
	DefaultProtocol   = "TCP"
	DefaultVMCIDR     = "10.0.2.0/24"
	DefaultVMIpv6CIDR = "fd10:0:2::/120"
	DefaultBridgeName = "k6t-eth0"
)

func NewDefaulter(arch string) *Defaulter {
	return &Defaulter{
		ArchDefaulter: archdefaulter.NewArchDefaulter(arch),
	}
}

type Defaulter struct {
	ArchDefaulter archdefaulter.ArchDefaulter
}

func (d *Defaulter) setDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		ostype.Arch = d.ArchDefaulter.OSTypeArch()
	}

	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		ostype.Machine = d.ArchDefaulter.OSTypeMachine()
	}
}

func (d *Defaulter) setDefaults_DomainSpec(spec *DomainSpec) {
	spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	if spec.Type == "" {
		spec.Type = "kvm"
	}
}

func (d *Defaulter) setDefaults_SysInfo(sysinfo *SysInfo) {
	if sysinfo.Type == "" {
		sysinfo.Type = "smbios"
	}
}

func (d *Defaulter) setDefaults_Features(spec *DomainSpec) {
	if spec.Features == nil {
		spec.Features = &Features{}
	}
}

func (d *Defaulter) SetObjectDefaults_Domain(in *Domain) {
	d.setDefaults_DomainSpec(&in.Spec)
	d.setDefaults_OSType(&in.Spec.OS.Type)
	if in.Spec.SysInfo != nil {
		d.setDefaults_SysInfo(in.Spec.SysInfo)
	}
	d.setDefaults_Features(&in.Spec)
}
