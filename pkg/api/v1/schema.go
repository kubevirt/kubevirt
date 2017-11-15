/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package v1

//go:generate swagger-doc

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

// http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
type CloudInitDataSourceNoCloud struct {
	// Reference to a k8s secret that contains NoCloud userdata
	UserDataSecretRef string `json:"userDataSecretRef"`
	// The NoCloud cloud-init userdata as a base64 encoded string
	UserDataBase64 string `json:"userDataBase64"`
	// The NoCloud cloud-init metadata as a base64 encoded string
	MetaDataBase64 string `json:"metaDataBase64"`
}

// Only one of the fields in the CloudInitSpec can be set
type CloudInitSpec struct {
	// Nocloud DataSource
	NoCloudData *CloudInitDataSourceNoCloud `json:"nocloud"`

	// Add future cloud init datasource structures below.
}

type DomainSpec struct {
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
	Emulator   string      `json:"emulator,omitempty"`
	Interfaces []Interface `json:"interfaces,omitempty"`
	Channels   []Channel   `json:"channels,omitempty"`
	Video      []Video     `json:"video,omitempty"`
	Graphics   []Graphics  `json:"graphics,omitempty"`
	Ballooning *Ballooning `json:"memballoon,omitempty"`
	Disks      []Disk      `json:"disks,omitempty"`
	Serials    []Serial    `json:"serials,omitempty"`
	Consoles   []Console   `json:"consoles,omitempty"`
	Watchdog   *Watchdog   `json:"watchdog,omitempty"`
}

// BEGIN Disk -----------------------------

type Disk struct {
	Device    string         `json:"device"`
	Snapshot  string         `json:"snapshot,omitempty"`
	Type      string         `json:"type"`
	Source    DiskSource     `json:"source"`
	Target    DiskTarget     `json:"target"`
	Serial    string         `json:"serial,omitempty"`
	Driver    *DiskDriver    `json:"driver,omitempty"`
	ReadOnly  *ReadOnly      `json:"readOnly,omitempty"`
	Auth      *DiskAuth      `json:"auth,omitempty"`
	CloudInit *CloudInitSpec `json:"cloudinit,omitempty"`
}

type DiskAuth struct {
	Username string      `json:"username"`
	Secret   *DiskSecret `json:"secret,omitempty"`
}

type DiskSecret struct {
	Type  string `json:"type"`
	Usage string `json:"usage"`
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

// BEGIN Serial -----------------------------

type Serial struct {
	Type   string        `json:"type"`
	Target *SerialTarget `json:"target,omitempty"`
}

type SerialTarget struct {
	Port *uint `json:"port,omitempty"`
}

// END Serial -----------------------------

// BEGIN Console -----------------------------

type Console struct {
	Type   string         `json:"type"`
	Target *ConsoleTarget `json:"target,omitempty"`
}

type ConsoleTarget struct {
	Type *string `json:"type,omitempty"`
	Port *uint   `json:"port,omitempty"`
}

// END Serial -----------------------------

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
	BootOrder []Boot    `json:"bootOrder,omitempty"`
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
	Type   string `json:"type"`
	Heads  *uint  `json:"heads,omitempty"`
	Ram    *uint  `json:"ram,omitempty"`
	VRam   *uint  `json:"vRam,omitempty"`
	VGAMem *uint  `json:"vgaMem,omitempty"`
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

// Hardware watchdog device
type Watchdog struct {
	// Defines what watchdog model to use, typically 'i6300esb'
	Model string `json:"model"`
	// The action to take. poweroff, reset, shutdown, pause, dump.
	Action string `json:"action"`
}

// TODO ballooning, rng, cpu ...

func NewMinimalDomainSpec() *DomainSpec {
	domain := DomainSpec{OS: OS{Type: OSType{OS: "hvm"}}, Type: "qemu"}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: InterfaceSource{Network: "default"}},
	}
	return &domain
}
