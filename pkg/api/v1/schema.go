package v1

//go:generate swagger-doc

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

import (
	"encoding/xml"
	"kubevirt.io/kubevirt/pkg/precond"
)

type DomainSpec struct {
	XMLName xml.Name `json:"-"`
	Name    string   `json:"name"`
	UUID    string   `json:"uuid,omitempty"`
	Memory  Memory   `json:"memory"`
	Type    string   `json:"type"`
	OS      OS       `json:"os"`
	SysInfo *SysInfo `json:"sysInfo,omitempty"`
	Devices Devices  `json:"devices"`
	Clock   *Clock   `json:"clock,omitempty"`
}

type Memory struct {
	Value uint   `json:"value"`
	Unit  string `json:"unit"`
}

type Devices struct {
	Emulator   string      `json:"emulator"`
	Interfaces []Interface `json:"interfaces,omitempty"`
	Channels   []Channel   `json:"channels,omitempty"`
	Video      []Video     `json:"video,omitempty"`
	Graphics   []Graphics  `json:"graphics,omitempty"`
	Ballooning *Ballooning `json:"memballoon,omitempty"`
	Disks      []Disk      `json:"disks,omitempty"`
}

// BEGIN Disk -----------------------------

type Disk struct {
	Device   string      `json:"device"`
	Snapshot string      `json:"snapshot,omitempty"`
	Type     string      `json:"type"`
	Source   DiskSource  `json:"source"`
	Target   DiskTarget  `json:"target"`
	Serial   string      `json:"serial,omitempty"`
	Driver   *DiskDriver `json:"driver,omitempty"`
	ReadOnly *ReadOnly   `json:"readOnly,omitempty"`
}

type ReadOnly struct{}

type DiskSource struct {
	File          string          `json:"file,omitempty"`
	StartupPolicy string          `json:"startupPolicy,omitempty"`
	Protocol      string          `json:"protocol,omitempty"`
	Name          string          `json:"name,omitempty"`
	Host          *DiskSourceHost `json:"host,omitempty"`
}

type DiskTarget struct {
	Bus    string `json:"bus,omitempty"`
	Device string `json:"dev"`
}

type DiskDriver struct {
	Cache       string `json:"cache,omitempty"`
	ErrorPolicy string `json:"errorPolicy,omitempty"`
	IO          string `json:"io,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
}

type DiskSourceHost struct {
	Name string `json:"name"`
	Port string `json:"port,omitempty"`
}

// END Disk -----------------------------

// BEGIN Inteface -----------------------------

type Interface struct {
	Address   *Address         `json:"address,omitempty"`
	Type      string           `json:"type"`
	Source    InterfaceSource  `json:"source"`
	Target    *InterfaceTarget `json:"target,omitempty"`
	Model     *Model           `json:"model,omitempty"`
	MAC       *MAC             `json:"mac,omitempty"`
	BandWidth *BandWidth       `json:"bandwidth,omitempty"`
	BootOrder *BootOrder       `json:"boot,omitempty"`
	LinkState *LinkState       `json:"link,omitempty"`
	FilterRef *FilterRef       `json:"filterRef,omitempty"`
	Alias     *Alias           `json:"alias,omitempty"`
}

type LinkState struct {
	State string `json:"state"`
}

type BandWidth struct {
}

type BootOrder struct {
	Order uint `json:"order"`
}

type MAC struct {
	MAC string `json:"address"`
}

type FilterRef struct {
	Filter string `json:"filter"`
}

type InterfaceSource struct {
	Network string `json:"network,omitempty"`
	Device  string `json:"device,omitempty"`
	Bridge  string `json:"bridge,omitempty"`
}

type Model struct {
	Type string `json:"type"`
}

type InterfaceTarget struct {
	Device string `json:"dev"`
}

type Alias struct {
	Name string `json:"name"`
}

// END Inteface -----------------------------
//BEGIN OS --------------------

type OS struct {
	Type      OSType    `json:"type"`
	SMBios    *SMBios   `json:"smBIOS,omitempty"`
	BootOrder []Boot    `json:"bootOrder"`
	BootMenu  *BootMenu `json:"bootMenu,omitempty"`
	BIOS      *BIOS     `json:"bios,omitempty"`
}

type OSType struct {
	OS      string `json:"os"`
	Arch    string `json:"arch,omitempty"`
	Machine string `json:"machine,omitempty"`
}

type SMBios struct {
	Mode string `json:"mode"`
}

type NVRam struct {
	NVRam    string `json:"nvRam,omitempty"`
	Template string `json:"template,omitempty"`
}

type Boot struct {
	Dev string `json:"dev"`
}

type BootMenu struct {
	Enabled bool  `json:"enabled,omitempty"`
	Timeout *uint `json:"timeout,omitempty"`
}

// TODO <loader readonly='yes' secure='no' type='rom'>/usr/lib/xen/boot/hvmloader</loader>
type BIOS struct {
}

// TODO <bios useserial='yes' rebootTimeout='0'/>
type Loader struct {
}

type SysInfo struct {
	Type      string  `json:"type"`
	System    []Entry `json:"system"`
	BIOS      []Entry `json:"bios"`
	BaseBoard []Entry `json:"baseBoard"`
}

type Entry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//END OS --------------------

//BEGIN Clock --------------------

type Clock struct {
}

type Timer struct {
	Name       string `json:"name"`
	TickPolicy string `json:"tickPolicy,omitempty"`
	Present    string `json:"present,omitempty"`
}

//END Clock --------------------

//BEGIN Channel --------------------

type Channel struct {
	Type   string         `json:"type"`
	Source ChannelSource  `json:"source,omitempty"`
	Target *ChannelTarget `json:"target,omitempty"`
}

type ChannelTarget struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type"`
	Address string `json:"address,omitempty"`
	Port    uint   `json:"port,omitempty"`
}

type ChannelSource struct {
	Mode string `json:"mode"`
	Path string `json:"path"`
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
	Type   string `json:"type"`
	Heads  uint   `json:"heads,omitempty"`
	Ram    uint   `json:"ram,omitempty"`
	VRam   uint   `json:"vram,omitempty"`
	VGAMem uint   `vgamem:"vram,omitempty"`
}

type Graphics struct {
	AutoPort      string `json:"autoPort,omitempty"`
	DefaultMode   string `json:"defaultMode,omitempty"`
	Listen        Listen `json:"listen,omitempty"`
	PasswdValidTo string `json:"passwdValidTo,omitempty"`
	Port          int32  `json:"port,omitempty"`
	TLSPort       int    `json:"tlsPort,omitempty"`
	Type          string `json:"type"`
}

type Listen struct {
	Type    string `json:"type"`
	Address string `json:"address,omitempty"`
	Network string `json:"network,omitempty"`
}

type Address struct {
	Type     string `json:"type"`
	Domain   string `json:"domain"`
	Bus      string `json:"bus"`
	Slot     string `json:"slot"`
	Function string `json:"function"`
}

//END Video -------------------

type Ballooning struct {
	Model string `json:"model"`
}

type RandomGenerator struct {
}

// TODO ballooning, rng, cpu ...

func NewMinimalDomainSpec(vmName string) *DomainSpec {
	precond.MustNotBeEmpty(vmName)
	domain := DomainSpec{OS: OS{Type: OSType{OS: "hvm"}}, Type: "qemu", Name: vmName}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{Emulator: "/usr/local/bin/qemu-x86_64"}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: InterfaceSource{Network: "default"}},
	}
	return &domain
}
