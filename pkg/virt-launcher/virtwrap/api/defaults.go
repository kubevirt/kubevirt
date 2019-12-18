package api

import (
	"runtime"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

const (
	resolvConf        = "/etc/resolv.conf"
	DefaultProtocol   = "TCP"
	DefaultVMCIDR     = "10.0.2.0/24"
	DefaultBridgeName = "k6t-eth0"
)

func SetDefaults_Devices(devices *Devices) {
	// Set default memballoon, "none" means that controller disabled
	devices.Ballooning = &Ballooning{
		Model: "none",
	}

}

func SetDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		if runtime.GOARCH == "ppc64le" {
			ostype.Arch = "ppc64le"
		} else {
			ostype.Arch = "x86_64"
		}
	}

	// q35 is an alias of the newest q35 machine type.
	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		if runtime.GOARCH == "ppc64le" {
			ostype.Machine = "pseries"
		} else {
			ostype.Machine = "q35"
		}
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

func GetInterfaceType(iface *v1.Interface) string {
	if iface.Slirp != nil {
		// Slirp configuration works only with e1000 or rtl8139
		if iface.Model != "e1000" && iface.Model != "rtl8139" {
			log.Log.Infof("The network interface type of %s was changed to e1000 due to unsupported interface type by qemu slirp network", iface.Name)
			return "e1000"
		}
		return iface.Model
	}
	if iface.Model != "" {
		return iface.Model
	}
	return "virtio"
}
