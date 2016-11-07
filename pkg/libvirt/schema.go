package libvirt

import (
	"encoding/xml"
	"kubevirt/core/pkg/precond"
)

type Domain struct {
	XMLName xml.Name `xml:"domain"`
	Name    string   `xml:"name"`
	Memory  Memory   `xml:"memory"`
	Type    string   `xml:"type,attr"`
	OS      string   `xml:"os>type"`
	Devices Devices  `xml:"devices"`
}

type Memory struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

type Devices struct {
	Emulator   Emulator    `xml:"emulator"`
	Interfaces []Interface `xml:"interface"`
}

type Emulator struct {
	Value string `xml:",chardata"`
}

type Interface struct {
	Type   string `xml:"type,attr"`
	Source Source `xml:"source"`
}

type Source struct {
	Network string `xml:"network,attr"`
}

func NewMinimalVM(vmName string) *Domain {
	precond.MustNotBeEmpty(vmName)
	domain := Domain{OS: "hvm", Type: "qemu", Name: vmName}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{Emulator: Emulator{Value: "/usr/local/bin/qemu-x86_64"}}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: Source{Network: "kubevirt-net"}},
		{Type: "network", Source: Source{Network: "default"}},
	}
	return &domain
}
