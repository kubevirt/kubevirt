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

package api

import (
	"encoding/xml"
	"reflect"

	"github.com/jeevatkm/go-model"
	kubev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/mapper"
	"kubevirt.io/kubevirt/pkg/precond"
)

type LifeCycle string
type StateChangeReason string

func init() {
	// TODO the whole mapping registration can be done be an automatic process with reflection
	mapper.AddConversion(&Memory{}, &v1.Memory{})
	mapper.AddConversion(&OS{}, &v1.OS{})
	mapper.AddConversion(&Devices{}, &v1.Devices{})
	mapper.AddPtrConversion((**Clock)(nil), (**v1.Clock)(nil))
	mapper.AddPtrConversion((**SysInfo)(nil), (**v1.SysInfo)(nil))
	mapper.AddConversion(&Channel{}, &v1.Channel{})
	mapper.AddConversion(&Interface{}, &v1.Interface{})
	mapper.AddConversion(&Graphics{}, &v1.Graphics{})
	mapper.AddPtrConversion((**Ballooning)(nil), (**v1.Ballooning)(nil))
	mapper.AddConversion(&Disk{}, &v1.Disk{})
	mapper.AddConversion(&DiskSource{}, &v1.DiskSource{})
	mapper.AddPtrConversion((**DiskSourceHost)(nil), (**v1.DiskSourceHost)(nil))
	mapper.AddConversion(&DiskTarget{}, &v1.DiskTarget{})
	mapper.AddPtrConversion((**DiskDriver)(nil), (**v1.DiskDriver)(nil))
	mapper.AddPtrConversion((**ReadOnly)(nil), (**v1.ReadOnly)(nil))
	mapper.AddPtrConversion((**Address)(nil), (**v1.Address)(nil))
	mapper.AddConversion(&Serial{}, &v1.Serial{})
	mapper.AddPtrConversion((**SerialTarget)(nil), (**v1.SerialTarget)(nil))
	mapper.AddConversion(&Console{}, &v1.Console{})
	mapper.AddPtrConversion((**ConsoleTarget)(nil), (**v1.ConsoleTarget)(nil))
	mapper.AddConversion(&InterfaceSource{}, &v1.InterfaceSource{})
	mapper.AddPtrConversion((**InterfaceTarget)(nil), (**v1.InterfaceTarget)(nil))
	mapper.AddPtrConversion((**Model)(nil), (**v1.Model)(nil))
	mapper.AddPtrConversion((**MAC)(nil), (**v1.MAC)(nil))
	mapper.AddPtrConversion((**BandWidth)(nil), (**v1.BandWidth)(nil))
	mapper.AddPtrConversion((**BootOrder)(nil), (**v1.BootOrder)(nil))
	mapper.AddPtrConversion((**LinkState)(nil), (**v1.LinkState)(nil))
	mapper.AddPtrConversion((**FilterRef)(nil), (**v1.FilterRef)(nil))
	mapper.AddPtrConversion((**Alias)(nil), (**v1.Alias)(nil))
	mapper.AddConversion(&OSType{}, &v1.OSType{})
	mapper.AddPtrConversion((**SMBios)(nil), (**v1.SMBios)(nil))
	mapper.AddConversion(&Boot{}, &v1.Boot{})
	mapper.AddPtrConversion((**BootMenu)(nil), (**v1.BootMenu)(nil))
	mapper.AddPtrConversion((**BIOS)(nil), (**v1.BIOS)(nil))
	mapper.AddConversion(&Entry{}, &v1.Entry{})
	mapper.AddConversion(&ChannelSource{}, &v1.ChannelSource{})
	mapper.AddPtrConversion((**ChannelTarget)(nil), (**v1.ChannelTarget)(nil))
	mapper.AddConversion(&VideoModel{}, &v1.Video{})
	mapper.AddConversion(&Listen{}, &v1.Listen{})
	mapper.AddPtrConversion((**DiskAuth)(nil), (**v1.DiskAuth)(nil))
	mapper.AddPtrConversion((**DiskSecret)(nil), (**v1.DiskSecret)(nil))

	model.AddConversion(&Video{}, &v1.Video{}, func(in reflect.Value) (reflect.Value, error) {
		out := v1.Video{}
		errs := model.Copy(&out, in.Interface().(Video).Model)
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion(&v1.Video{}, &Video{}, func(in reflect.Value) (reflect.Value, error) {
		out := Video{}
		errs := model.Copy(&out.Model, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
}

const (
	NoState     LifeCycle = "NoState"
	Running     LifeCycle = "Running"
	Blocked     LifeCycle = "Blocked"
	Paused      LifeCycle = "Paused"
	Shutdown    LifeCycle = "ShuttingDown"
	Shutoff     LifeCycle = "Shutoff"
	Crashed     LifeCycle = "Crashed"
	PMSuspended LifeCycle = "PMSuspended"

	// Common reasons
	ReasonUnknown StateChangeReason = "Unknown"

	// ShuttingDown reasons
	ReasonUser StateChangeReason = "User"

	// Shutoff reasons
	ReasonShutdown     StateChangeReason = "Shutdown"
	ReasonDestroyed    StateChangeReason = "Destroyed"
	ReasonMigrated     StateChangeReason = "Migrated"
	ReasonCrashed      StateChangeReason = "Crashed"
	ReasonPanicked     StateChangeReason = "Panicked"
	ReasonSaved        StateChangeReason = "Saved"
	ReasonFailed       StateChangeReason = "Failed"
	ReasonFromSnapshot StateChangeReason = "FromSnapshot"
)

type Domain struct {
	metav1.TypeMeta
	ObjectMeta kubev1.ObjectMeta
	Spec       DomainSpec
	Status     DomainStatus
}

func (in *Domain) DeepCopyInto(out *Domain) {
	v, err := model.Clone(in)
	if err != nil {
		panic(err)
	}
	out = v.(*Domain)
	return
}

func (in *Domain) DeepCopy() *Domain {
	if in == nil {
		return nil
	}
	out := new(Domain)
	in.DeepCopyInto(out)
	return out
}

func (in *Domain) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

type DomainStatus struct {
	Status LifeCycle
	Reason StateChangeReason
}

type DomainList struct {
	metav1.TypeMeta
	ListMeta metav1.ListMeta
	Items    []Domain
}

func (in *DomainList) DeepCopyInto(out *DomainList) {
	v, err := model.Clone(in)
	if err != nil {
		panic(err)
	}
	out = v.(*DomainList)
	return
}

func (in *DomainList) DeepCopy() *DomainList {
	if in == nil {
		return nil
	}
	out := new(DomainList)
	in.DeepCopyInto(out)
	return out
}

func (in *DomainList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DomainSpec represents the actual conversion to libvirt XML. The fields must be
// tagged, and they must correspond to the libvirt domain as described in
// https://libvirt.org/formatdomain.html.
type DomainSpec struct {
	XMLName  xml.Name     `xml:"domain"`
	Type     string       `xml:"type,attr"`
	XmlNS    string       `xml:"xmlns:qemu,attr,omitempty"`
	Name     string       `xml:"name"`
	UUID     string       `xml:"uuid,omitempty"`
	Memory   Memory       `xml:"memory"`
	OS       OS           `xml:"os"`
	SysInfo  *SysInfo     `xml:"sysinfo,omitempty"`
	Devices  Devices      `xml:"devices"`
	Clock    *Clock       `xml:"clock,omitempty"`
	Resource *Resource    `xml:"resource,omitempty"`
	QEMUCmd  *Commandline `xml:"qemu:commandline,omitempty"`
}

type Commandline struct {
	QEMUEnv []Env `xml:"qemu:env,omitempty"`
}

type Env struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type Resource struct {
	Partition string `xml:"partition"`
}

type Memory struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

type Devices struct {
	Emulator   string      `xml:"emulator,omitempty"`
	Interfaces []Interface `xml:"interface"`
	Channels   []Channel   `xml:"channel"`
	Video      []Video     `xml:"video"`
	Graphics   []Graphics  `xml:"graphics"`
	Ballooning *Ballooning `xml:"memballoon,omitempty"`
	Disks      []Disk      `xml:"disk"`
	Serials    []Serial    `xml:"serial"`
	Consoles   []Console   `xml:"console"`
}

// BEGIN Disk -----------------------------

type Disk struct {
	Device   string      `xml:"device,attr"`
	Snapshot string      `xml:"snapshot,attr,omitempty"`
	Type     string      `xml:"type,attr"`
	Source   DiskSource  `xml:"source"`
	Target   DiskTarget  `xml:"target"`
	Serial   string      `xml:"serial,omitempty"`
	Driver   *DiskDriver `xml:"driver,omitempty"`
	ReadOnly *ReadOnly   `xml:"readonly,omitempty"`
	Auth     *DiskAuth   `xml:"auth,omitempty"`
}

type DiskAuth struct {
	Username string      `xml:"username,attr"`
	Secret   *DiskSecret `xml:"secret,omitempty"`
}

type DiskSecret struct {
	Type  string `xml:"type,attr"`
	Usage string `xml:"usage,attr"`
}

type ReadOnly struct{}

type DiskSource struct {
	File          string          `xml:"file,attr,omitempty"`
	StartupPolicy string          `xml:"startupPolicy,attr,omitempty"`
	Protocol      string          `xml:"protocol,attr,omitempty"`
	Name          string          `xml:"name,attr,omitempty"`
	Host          *DiskSourceHost `xml:"host,omitempty"`
}

type DiskTarget struct {
	Bus    string `xml:"bus,attr,omitempty"`
	Device string `xml:"dev,attr"`
}

type DiskDriver struct {
	Cache       string `xml:"cache,attr,omitempty"`
	ErrorPolicy string `xml:"error_policy,attr,omitempty"`
	IO          string `xml:"io,attr,omitempty"`
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
}

type DiskSourceHost struct {
	Name string `xml:"name,attr"`
	Port string `xml:"port,attr,omitempty"`
}

// END Disk -----------------------------

// BEGIN Serial -----------------------------

type Serial struct {
	Type   string        `xml:"type,attr"`
	Target *SerialTarget `xml:"target,omitempty"`
}

type SerialTarget struct {
	Port *uint `xml:"port,attr,omitempty"`
}

// END Serial -----------------------------

// BEGIN Console -----------------------------

type Console struct {
	Type   string         `xml:"type,attr"`
	Target *ConsoleTarget `xml:"target,omitempty"`
}

type ConsoleTarget struct {
	Type *string `xml:"type,attr,omitempty"`
	Port *uint   `xml:"port,attr,omitempty"`
}

// END Serial -----------------------------

// BEGIN Inteface -----------------------------

type Interface struct {
	Address   *Address         `xml:"address,omitempty"`
	Type      string           `xml:"type,attr"`
	Source    InterfaceSource  `xml:"source"`
	Target    *InterfaceTarget `xml:"target,omitempty"`
	Model     *Model           `xml:"model,omitempty"`
	MAC       *MAC             `xml:"mac,omitempty"`
	BandWidth *BandWidth       `xml:"bandwidth,omitempty"`
	BootOrder *BootOrder       `xml:"boot,omitempty"`
	LinkState *LinkState       `xml:"link,omitempty"`
	FilterRef *FilterRef       `xml:"filterref,omitempty"`
	Alias     *Alias           `xml:"alias,omitempty"`
}

type LinkState struct {
	State string `xml:"state,attr"`
}

type BandWidth struct {
}

type BootOrder struct {
	Order uint `xml:"order,attr"`
}

type MAC struct {
	MAC string `xml:"address,attr"`
}

type FilterRef struct {
	Filter string `xml:"filter,attr"`
}

type InterfaceSource struct {
	Network string `xml:"network,attr,omitempty"`
	Device  string `xml:"dev,attr,omitempty"`
	Bridge  string `xml:"bridge,attr,omitempty"`
}

type Model struct {
	Type string `xml:"type,attr"`
}

type InterfaceTarget struct {
	Device string `xml:"dev,attr"`
}

type Alias struct {
	Name string `xml:"name,attr"`
}

// END Inteface -----------------------------
//BEGIN OS --------------------

type OS struct {
	Type       OSType    `xml:"type"`
	SMBios     *SMBios   `xml:"smbios,omitempty"`
	BootOrder  []Boot    `xml:"boot"`
	BootMenu   *BootMenu `xml:"bootmenu,omitempty"`
	BIOS       *BIOS     `xml:"bios,omitempty"`
	Kernel     string    `xml:"kernel,omitempty"`
	Initrd     string    `xml:"initrd,omitempty"`
	KernelArgs string    `xml:"cmdline,omitempty"`
}

type OSType struct {
	OS      string `xml:",chardata"`
	Arch    string `xml:"arch,attr,omitempty"`
	Machine string `xml:"machine,attr,omitempty"`
}

type SMBios struct {
	Mode string `xml:"mode,attr"`
}

type NVRam struct {
	NVRam    string `xml:",chardata,omitempty"`
	Template string `xml:"template,attr,omitempty"`
}

type Boot struct {
	Dev string `xml:"dev,attr"`
}

type BootMenu struct {
	Enabled bool  `xml:"enabled,attr"`
	Timeout *uint `xml:"timeout,attr,omitempty"`
}

// TODO <loader readonly='yes' secure='no' type='rom'>/usr/lib/xen/boot/hvmloader</loader>
type BIOS struct {
}

// TODO <bios useserial='yes' rebootTimeout='0'/>
type Loader struct {
}

type SysInfo struct {
	Type      string  `xml:"type,attr"`
	System    []Entry `xml:"system>entry"`
	BIOS      []Entry `xml:"bios>entry"`
	BaseBoard []Entry `xml:"baseBoard>entry"`
}

type Entry struct {
	Name  string `xml:"name"`
	Value string `xml:",chardata"`
}

//END OS --------------------

//BEGIN Clock --------------------

type Clock struct {
}

type Timer struct {
	Name       string `xml:"name,attr"`
	TickPolicy string `xml:"tickpolicy,attr,omitempty"`
	Present    string `xml:"present,attr,omitempty"`
}

//END Clock --------------------

//BEGIN Channel --------------------

type Channel struct {
	Type   string         `xml:"type,attr"`
	Source ChannelSource  `xml:"source,omitempty"`
	Target *ChannelTarget `xml:"target,omitempty"`
}

type ChannelTarget struct {
	Name    string `xml:"name,attr,omitempty"`
	Type    string `xml:"type,attr"`
	Address string `xml:"address,attr,omitempty"`
	Port    uint   `xml:"port,attr,omitempty"`
}

type ChannelSource struct {
	Mode string `xml:"mode,attr"`
	Path string `xml:"path,attr"`
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
	Type   string `xml:"type,attr"`
	Heads  *uint  `xml:"heads,attr,omitempty"`
	Ram    *uint  `xml:"ram,attr,omitempty"`
	VRam   *uint  `xml:"vram,attr,omitempty"`
	VGAMem *uint  `xml:"vgamem,attr,omitempty"`
}

type Graphics struct {
	AutoPort      string `xml:"autoPort,attr,omitempty"`
	DefaultMode   string `xml:"defaultMode,attr,omitempty"`
	Listen        Listen `xml:"listen,omitempty"`
	PasswdValidTo string `xml:"passwdValidTo,attr,omitempty"`
	Port          int32  `xml:"port,attr,omitempty"`
	TLSPort       int    `xml:"tlsPort,attr,omitempty"`
	Type          string `xml:"type,attr"`
}

type Listen struct {
	Type    string `xml:"type,attr"`
	Address string `xml:"address,attr,omitempty"`
	Network string `xml:"newtork,attr,omitempty"`
}

type Address struct {
	Type     string `xml:"type,attr"`
	Domain   string `xml:"domain,attr"`
	Bus      string `xml:"bus,attr"`
	Slot     string `xml:"slot,attr"`
	Function string `xml:"function,attr"`
}

//END Video -------------------

type Ballooning struct {
	Model string `xml:"model,attr"`
}

type RandomGenerator struct {
}

// TODO ballooning, rng, cpu ...

type SecretUsage struct {
	Type   string `xml:"type,attr"`
	Target string `xml:"target,omitempty"`
}

type SecretSpec struct {
	XMLName     xml.Name    `xml:"secret"`
	Ephemeral   string      `xml:"ephemeral,attr"`
	Private     string      `xml:"private,attr"`
	Description string      `xml:"description,omitempty"`
	Usage       SecretUsage `xml:"usage,omitempty"`
}

func NewMinimalDomainSpec(vmName string) *DomainSpec {
	precond.MustNotBeEmpty(vmName)
	domain := DomainSpec{OS: OS{Type: OSType{OS: "hvm"}}, Type: "qemu", Name: vmName}
	domain.Memory = Memory{Unit: "KiB", Value: 8192}
	domain.Devices = Devices{}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: InterfaceSource{Network: "default"}},
	}
	return &domain
}

func NewMinimalDomain(name string) *Domain {
	return NewMinimalDomainWithNS(kubev1.NamespaceDefault, name)
}

func NewMinimalDomainWithNS(namespace string, name string) *Domain {
	domain := NewDomainReferenceFromName(namespace, name)
	domain.Spec = *NewMinimalDomainSpec(name)
	return domain
}

func NewDomainReferenceFromName(namespace string, name string) *Domain {
	return &Domain{
		Spec: DomainSpec{},
		ObjectMeta: kubev1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: DomainStatus{},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "1.2.2",
			Kind:       "Domain",
		},
	}
}

func (d *Domain) SetState(state LifeCycle, reason StateChangeReason) {
	d.Status.Status = state
	d.Status.Reason = reason
}

// Required to satisfy Object interface
func (d *Domain) GetObjectKind() schema.ObjectKind {
	return &d.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (d *Domain) GetObjectMeta() metav1.Object {
	return &d.ObjectMeta
}

// Required to satisfy Object interface
func (dl *DomainList) GetObjectKind() schema.ObjectKind {
	return &dl.TypeMeta
}

// Required to satisfy ListMetaAccessor interface
func (dl *DomainList) GetListMeta() meta.List {
	return &dl.ListMeta
}
