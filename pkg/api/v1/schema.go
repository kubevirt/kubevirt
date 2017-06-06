/*
 * This file is part of the kubevirt project
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

type DomainSpec struct {
	Memory  Memory   `json:"memory"`
	Type    string   `json:"type"`
	OS      OS       `json:"os"`
	SysInfo *SysInfo `json:"sysInfo,omitempty"`
	Devices Devices  `json:"devices"`
}

type Memory struct {
	Value uint   `json:"value"`
	Unit  string `json:"unit"`
}

type Devices struct {
	Interfaces []Interface `json:"interfaces,omitempty"`
	Channels   []Channel   `json:"channels,omitempty"`
	Video      []Video     `json:"video,omitempty"`
	Graphics   []Graphics  `json:"graphics,omitempty"`
	Ballooning *Ballooning `json:"memballoon,omitempty"`
	Disks      []Disk      `json:"disks,omitempty"`
	Serials    []Serial    `json:"serials,omitempty"`
	Consoles   []Console   `json:"consoles,omitempty"`
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

// BEGIN Serial -----------------------------

type Serial struct {
	Target *SerialTarget `json:"target,omitempty"`
}

type SerialTarget struct {
	Port *uint `json:"port,omitempty"`
}

// END Serial -----------------------------

// BEGIN Console -----------------------------

type Console struct {
	Target *ConsoleTarget `json:"target,omitempty"`
}

type ConsoleTarget struct {
	Type *string `json:"type,omitempty"`
	Port *uint   `json:"port,omitempty"`
}

// END Serial -----------------------------

// BEGIN Inteface -----------------------------

type Interface struct {
	Address   *Address   `json:"address,omitempty"`
	Model     *Model     `json:"model,omitempty"`
	MAC       *MAC       `json:"mac,omitempty"`
	BootOrder *BootOrder `json:"boot,omitempty"`
	LinkState *LinkState `json:"link,omitempty"`
}

type LinkState struct {
	State string `json:"state"`
}

type BootOrder struct {
	Order uint `json:"order"`
}

type MAC struct {
	MAC string `json:"address"`
}

type Model struct {
	Type string `json:"type"`
}

// END Inteface -----------------------------
//BEGIN OS --------------------

type OS struct {
	Type     OSType    `json:"type"`
	SMBios   *SMBios   `json:"smBIOS,omitempty"`
	BootMenu *BootMenu `json:"bootMenu,omitempty"`
}

type OSType struct {
	OS      string `json:"os"`
	Arch    string `json:"arch,omitempty"`
	Machine string `json:"machine,omitempty"`
}

type SMBios struct {
	Mode string `json:"mode"`
}

type BootMenu struct {
	Enabled bool  `json:"enabled,omitempty"`
	Timeout *uint `json:"timeout,omitempty"`
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

//BEGIN Channel --------------------

type Channel struct {
	Target ChannelTarget `json:"target"`
}

type ChannelTarget struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

//END Channel --------------------

//BEGIN Video -------------------

type Video struct {
	Type   string `json:"type"`
	Heads  *uint  `json:"heads,omitempty"`
	Ram    *uint  `json:"ram,omitempty"`
	VRam   *uint  `json:"vRam,omitempty"`
	VGAMem *uint  `json:"vgaMem,omitempty"`
}

type Graphics struct {
	Type string `json:"type"`
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

func NewMinimalDomainSpec() *DomainSpec {
	domain := DomainSpec{OS: OS{Type: OSType{OS: "hvm"}}, Type: "qemu"}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{}
	domain.Devices.Interfaces = []Interface{Interface{}}
	return &domain
}
