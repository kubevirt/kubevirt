package v1

//go:generate desc -in schema.go

import (
	"encoding/xml"
	"kubevirt.io/kubevirt/pkg/precond"
)

type DomainSpec struct {
	XMLName xml.Name `xml:"domain" json:"-"`
	Name    string   `xml:"name" json:"name"`
	UUID    string   `xml:"uuid,omitempty" json:"uuid,omitempty"`
	Memory  Memory   `xml:"memory" json:"memory"`
	Type    string   `xml:"type,attr" json:"type"`
	OS      OS       `xml:"os" json:"os"`
	SysInfo *SysInfo `xml:"sysinfo,omitempty" json:"sysInfo,omitempty"`
	Devices Devices  `xml:"devices" json:"devices"`
	Clock   *Clock   `xml:"clock,omitempty" json:"clock,omitempty"`
}

type Memory struct {
	Value uint   `xml:",chardata" json:"value"`
	Unit  string `xml:"unit,attr" json:"unit"`
}

type Devices struct {
	Emulator   string      `xml:"emulator" json:"emulator"`
	Interfaces []Interface `xml:"interface" json:"interfaces,omitempty"`
	Channels   []Channel   `xml:"channel" json:"channels,omitempty"`
	Video      []Video     `xml:"video" json:"video,omitempty"`
	Graphics   []Graphics  `xml:"graphics" json:"graphics,omitempty"`
	Ballooning *Ballooning `xml:"memballoon,omitempty" json:"memballoon,omitempty"`
	Disks      []Disk      `xml:"disk" json:"disks,omitempty"`
}

// BEGIN Disk -----------------------------

type Disk struct {
	Device     string      `xml:"device,attr" json:"device"`
	Snapshot   string      `xml:"snapshot,attr" json:"shapshot"`
	Type       string      `xml:"type,attr" json:"type"`
	DiskSource DiskSource  `xml:"source" json:"diskSource"`
	DiskTarget DiskTarget  `xml:"target" json:"diskTarget"`
	Serial     string      `xml:"serial,omitempty" json:"serial,omitempty"`
	Driver     *DiskDriver `xml:"driver,omitempty" json:"driver,omitempty"`
	ReadOnly   *ReadOnly   `xml:"readonly,omitempty" json:"readOnly,omitempty"`
}

type ReadOnly struct{}

type DiskSource struct {
	File          string `xml:"file,attr" json:"file"`
	StartupPolicy string `xml:"startupPolicy,attr,omitempty" json:"startupPolicy,omitempty"`
}

type DiskTarget struct {
	Bus    string `xml:"bus,attr" json:"bus"`
	Device string `xml:"dev,attr" json:"dev"`
}

type DiskDriver struct {
	Cache       string `xml:"cache,attr,omitempty" json:"cache,omitempty"`
	ErrorPolicy string `xml:"error_policy,attr,omitempty" json:"errorPolicy,omitempty"`
	IO          string `xml:"io,attr,omitempty" json:"io,omitempty"`
	Name        string `xml:"name,attr" json:"name"`
	Type        string `xml:"type,attr" json:"type"`
}

// END Disk -----------------------------

// BEGIN Inteface -----------------------------

type Interface struct {
	Address   *Address         `xml:"address,omitempty" json:"address,omitempty"`
	Type      string           `xml:"type,attr" json:"type"`
	Source    InterfaceSource  `xml:"source" json:"source"`
	Target    *InterfaceTarget `xml:"target,omitempty" json:"target,omitempty"`
	Model     *Model           `xml:"model,omitempty" json:"model,omitempty"`
	MAC       *MAC             `xml:"mac,omitempty" json:"mac,omitempty"`
	BandWidth *BandWidth       `xml:"bandwidth,omitempty" json:"bandwidth,omitempty"`
	BootOrder *BootOrder       `xml:"boot,omitempty" json:"boot,omitempty"`
	LinkState *LinkState       `xml:"link,omitempty" json:"link,omitempty"`
	FilterRef *FilterRef       `xml:"filterref,omitempty" json:"filterRef,omitempty"`
	Alias     *Alias           `xml:"alias,omitempty" json:"alias,omitempty"`
}

type LinkState struct {
	State string `xml:"state,attr" json:"state"`
}

type BandWidth struct {
}

type BootOrder struct {
	Order uint `xml:"order,attr" json:"order"`
}

type MAC struct {
	MAC string `xml:"address,attr" json:"address"`
}

type FilterRef struct {
	Filter string `xml:"filter,attr" json:"filter"`
}

type InterfaceSource struct {
	Network string `xml:"network,attr,omitempty" json:"network,omitempty"`
	Device  string `xml:"dev,attr,omitempty" json:"device,omitempty"`
	Bridge  string `xml:"bridge,attr,omitempty" json:"bridge,omitempty"`
}

type Model struct {
	Type string `xml:"type,attr" json:"type"`
}

type InterfaceTarget struct {
	Device string `xml:"dev,attr" json:"dev"`
}

type Alias struct {
	Name string `xml:"name,attr" json:"name"`
}

// END Inteface -----------------------------
//BEGIN OS --------------------

type OS struct {
	Type      OSType    `xml:"type" json:"type"`
	SMBios    *SMBios   `xml:"smbios,omitempty" json:"smBIOS,omitempty"`
	BootOrder []Boot    `xml:"boot" json:"bootOrder"`
	BootMenu  *BootMenu `xml:"bootmenu,omitempty" json:"bootMenu,omitempty"`
	BIOS      *BIOS     `xml:"bios,omitempty" json:"bios,omitempty"`
}

type OSType struct {
	OS      string `xml:",chardata" json:"os"`
	Arch    string `xml:"arch,attr,omitempty" json:"arch,omitempty"`
	Machine string `xml:"machine,attr,omitempty" json:"machine,omitempty"`
}

type SMBios struct {
	Mode string `xml:"mode,attr" json:"mode"`
}

type NVRam struct {
	NVRam    string `xml:",chardata,omitempty" json:"nvRam,omitempty"`
	Template string `xml:"template,attr,omitempty" json:"template,omitempty"`
}

type Boot struct {
	Dev string `xml:"dev,attr" json:"dev"`
}

type BootMenu struct {
	Enabled bool  `xml:"enabled,attr" json:"enabled,omitempty"`
	Timeout *uint `xml:"timeout,attr,omitempty" json:"timeout,omitempty"`
}

// TODO <loader readonly='yes' secure='no' type='rom'>/usr/lib/xen/boot/hvmloader</loader>
type BIOS struct {
}

// TODO <bios useserial='yes' rebootTimeout='0'/>
type Loader struct {
}

type SysInfo struct {
	Type      string  `xml:"type,attr" json:"type"`
	System    []Entry `xml:"system>entry" json:"system"`
	BIOS      []Entry `xml:"bios>entry" json:"bios"`
	BaseBoard []Entry `xml:"baseBoard>entry" json:"baseBoard"`
}

type Entry struct {
	Name  string `xml:"name" json:"name"`
	Value string `xml:",chardata" json:"value"`
}

//END OS --------------------

//BEGIN Clock --------------------

type Clock struct {
}

type Timer struct {
	Name       string `xml:"name,attr" json:"name"`
	TickPolicy string `xml:"tickpolicy,attr,omitempty" json:"tickPolicy,omitempty"`
	Present    string `xml:"present,attr,omitempty" json:"present,omitempty"`
}

//END Clock --------------------

//BEGIN Channel --------------------

type Channel struct {
	Type   string         `xml:"type,attr" json:"type"`
	Source ChannelSource  `xml:"target,omitempty" json:"source,omitempty"`
	Target *ChannelTarget `xml:"target,omitempty" json:"target,omitempty"`
}

type ChannelTarget struct {
	Name    string `xml:"name,attr,omitempty" json:"name,omitempty"`
	Type    string `xml:"type,attr" json:"type"`
	Address string `xml:"address,attr,omitempty" json:"address,omitempty"`
	Port    uint   `xml:"port,attr,omitempty" json:"port,omitempty"`
}

type ChannelSource struct {
	Mode string `xml:"mode,attr" json:"mode"`
	Path string `xml:"path,attr" json:"path"`
}

//END Channel --------------------

//BEGIN Video -------------------
/*
<graphics autoport="yes" defaultMode="secure" listen="0" passwd="*****" passwdValidTo="1970-01-01T00:00:01" port="-1" tlsPort="-1" type="spice" />
*/

type Video struct {
	Model VideoModel `xml:"model"`
}

type VideoModel struct {
	Type   string `xml:"type,attr" json:"type"`
	Heads  uint   `xml:"heads,attr,omitempty" json:"heads,omitempty"`
	Ram    uint   `xml:"ram,attr,omitempty" json:"ram,omitempty"`
	VRam   uint   `xml:"vram,attr,omitempty" json:"vram,omitempty"`
	VGAMem uint   `xml:"vgamem,attr,omitempty" vgamem:"vram,omitempty"`
}

type Graphics struct {
	AutoPort      string `xml:"autoPort,attr,omitempty" json:"autoPort,omitempty"`
	DefaultMode   string `xml:"defaultMode,attr,omitempty" json:"defaultMode,omitempty"`
	Listen        Listen `xml:"listen,omitempty" json:"listen,omitempty"`
	PasswdValidTo string `xml:"passwdValidTo,attr,omitempty" json:"passwdValidTo,omitempty"`
	Port          int    `xml:"port,attr,omitempty" json:"port,omitempty"`
	TLSPort       int    `xml:"tlsPort,attr,omitempty" json:"tlsPort,omitempty"`
	Type          string `xml:"type,attr" json:"type"`
}

type Listen struct {
	Type    string `xml:"type,attr" json:"type"`
	Address string `xml:"address,attr,omitempty" json:"address,omitempty"`
	Network string `xml:"newtork,attr,omitempty" json:"network,omitempty"`
}

type Address struct {
	Type     string `xml:"type,attr" json:"type"`
	Domain   string `xml:"domain,attr" json:"domain"`
	Bus      string `xml:"bus,attr" json:"bus"`
	Slot     string `xml:"slot,attr" json:"slot"`
	Function string `xml:"function,attr" json:"function"`
}

//END Video -------------------

type Ballooning struct {
	Model string `xml:"model,attr" json:"model"`
}

type RandomGenerator struct {
}

// TODO ballooning, rng, cpu ...

func NewMinimalVM(vmName string) *DomainSpec {
	precond.MustNotBeEmpty(vmName)
	domain := DomainSpec{OS: OS{Type: OSType{OS: "hvm"}}, Type: "qemu", Name: vmName}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{Emulator: "/usr/local/bin/qemu-x86_64"}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: InterfaceSource{Network: "default"}},
	}
	return &domain
}
