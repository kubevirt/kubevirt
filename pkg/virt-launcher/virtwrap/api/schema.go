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
 * Copyright 2017,2018 Red Hat, Inc.
 *
 */

//go:generate deepcopy-gen -i . --go-header-file ../../../../hack/boilerplate/boilerplate.go.txt
//go:generate defaulter-gen -i . --go-header-file ../../../../hack/boilerplate/boilerplate.go.txt

package api

import (
	"encoding/xml"
	"fmt"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
)

type LifeCycle string
type StateChangeReason string

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

	// NoState reasons
	ReasonNonExistent StateChangeReason = "NonExistent"

	// Pause reasons
	ReasonPausedUnknown        StateChangeReason = "Unknown"
	ReasonPausedUser           StateChangeReason = "User"
	ReasonPausedMigration      StateChangeReason = "Migration"
	ReasonPausedSave           StateChangeReason = "Save"
	ReasonPausedDump           StateChangeReason = "Dump"
	ReasonPausedIOError        StateChangeReason = "IOError"
	ReasonPausedWatchdog       StateChangeReason = "Watchdog"
	ReasonPausedFromSnapshot   StateChangeReason = "FromSnapshot"
	ReasonPausedShuttingDown   StateChangeReason = "ShuttingDown"
	ReasonPausedSnapshot       StateChangeReason = "Snapshot"
	ReasonPausedCrashed        StateChangeReason = "Crashed"
	ReasonPausedStartingUp     StateChangeReason = "StartingUp"
	ReasonPausedPostcopy       StateChangeReason = "Postcopy"
	ReasonPausedPostcopyFailed StateChangeReason = "PostcopyFailed"

	UserAliasPrefix = "ua-"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Domain struct {
	metav1.TypeMeta
	ObjectMeta metav1.ObjectMeta
	Spec       DomainSpec
	Status     DomainStatus
}

type DomainStatus struct {
	Status     LifeCycle
	Reason     StateChangeReason
	Interfaces []InterfaceStatus
}

type InterfaceStatus struct {
	Name          string
	Mac           string
	Ip            string
	IPs           []string
	InterfaceName string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DomainList struct {
	metav1.TypeMeta
	ListMeta metav1.ListMeta
	Items    []Domain
}

// DomainSpec represents the actual conversion to libvirt XML. The fields must be
// tagged, and they must correspond to the libvirt domain as described in
// https://libvirt.org/formatdomain.html.
type DomainSpec struct {
	XMLName       xml.Name       `xml:"domain"`
	Type          string         `xml:"type,attr"`
	XmlNS         string         `xml:"xmlns:qemu,attr,omitempty"`
	Name          string         `xml:"name"`
	UUID          string         `xml:"uuid,omitempty"`
	Memory        Memory         `xml:"memory"`
	MemoryBacking *MemoryBacking `xml:"memoryBacking,omitempty"`
	OS            OS             `xml:"os"`
	SysInfo       *SysInfo       `xml:"sysinfo,omitempty"`
	Devices       Devices        `xml:"devices"`
	Clock         *Clock         `xml:"clock,omitempty"`
	Resource      *Resource      `xml:"resource,omitempty"`
	QEMUCmd       *Commandline   `xml:"qemu:commandline,omitempty"`
	Metadata      Metadata       `xml:"metadata,omitempty"`
	Features      *Features      `xml:"features,omitempty"`
	CPU           CPU            `xml:"cpu"`
	VCPU          *VCPU          `xml:"vcpu"`
	CPUTune       *CPUTune       `xml:"cputune"`
	IOThreads     *IOThreads     `xml:"iothreads,omitempty"`
}

type CPUTune struct {
	VCPUPin     []CPUTuneVCPUPin     `xml:"vcpupin"`
	IOThreadPin []CPUTuneIOThreadPin `xml:"iothreadpin,omitempty"`
}

type CPUTuneVCPUPin struct {
	VCPU   uint   `xml:"vcpu,attr"`
	CPUSet string `xml:"cpuset,attr"`
}

type CPUTuneIOThreadPin struct {
	IOThread uint   `xml:"iothread,attr"`
	CPUSet   string `xml:"cpuset,attr"`
}

type VCPU struct {
	Placement string `xml:"placement,attr"`
	CPUs      uint32 `xml:",chardata"`
}

type CPU struct {
	Mode     string       `xml:"mode,attr,omitempty"`
	Model    string       `xml:"model,omitempty"`
	Topology *CPUTopology `xml:"topology"`
}

type CPUTopology struct {
	Sockets uint32 `xml:"sockets,attr,omitempty"`
	Cores   uint32 `xml:"cores,attr,omitempty"`
	Threads uint32 `xml:"threads,attr,omitempty"`
}

type Features struct {
	ACPI   *FeatureEnabled `xml:"acpi,omitempty"`
	APIC   *FeatureEnabled `xml:"apic,omitempty"`
	Hyperv *FeatureHyperv  `xml:"hyperv,omitempty"`
}

type FeatureHyperv struct {
	Relaxed    *FeatureState     `xml:"relaxed,omitempty"`
	VAPIC      *FeatureState     `xml:"vapic,omitempty"`
	Spinlocks  *FeatureSpinlocks `xml:"spinlocks,omitempty"`
	VPIndex    *FeatureState     `xml:"vpindex,omitempty"`
	Runtime    *FeatureState     `xml:"runtime,omitempty"`
	SyNIC      *FeatureState     `xml:"synic,omitempty"`
	SyNICTimer *FeatureState     `xml:"stimer,omitempty"`
	Reset      *FeatureState     `xml:"reset,omitempty"`
	VendorID   *FeatureVendorID  `xml:"vendor_id,omitempty"`
}

type FeatureSpinlocks struct {
	State   string  `xml:"state,attr,omitempty"`
	Retries *uint32 `xml:"retries,attr,omitempty"`
}

type FeatureVendorID struct {
	State string `xml:"state,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
}

type FeatureEnabled struct {
}

type FeatureState struct {
	State string `xml:"state,attr,omitempty"`
}

type Metadata struct {
	// KubeVirt contains kubevirt related metadata
	// Note: Libvirt only accept one element at metadata root with a specific namespace
	KubeVirt KubeVirtMetadata `xml:"http://kubevirt.io kubevirt"`
}

type KubeVirtMetadata struct {
	UID         types.UID            `xml:"uid"`
	GracePeriod *GracePeriodMetadata `xml:"graceperiod,omitempty"`
	Migration   *MigrationMetadata   `xml:"migration,omitempty"`
}

type MigrationMetadata struct {
	UID            types.UID    `xml:"uid,omitempty"`
	StartTimestamp *metav1.Time `xml:"startTimestamp,omitempty"`
	EndTimestamp   *metav1.Time `xml:"endTimestamp,omitempty"`
	Completed      bool         `xml:"completed,omitempty"`
	Failed         bool         `xml:"failed,omitempty"`
	FailureReason  string       `xml:"failureReason,omitempty"`
}

type GracePeriodMetadata struct {
	DeletionGracePeriodSeconds int64        `xml:"deletionGracePeriodSeconds"`
	DeletionTimestamp          *metav1.Time `xml:"deletionTimestamp,omitempty"`
}

type Commandline struct {
	QEMUEnv []Env `xml:"qemu:env,omitempty"`
	QEMUArg []Arg `xml:"qemu:arg,omitempty"`
}

type Env struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type Arg struct {
	Value string `xml:"value,attr"`
}

type Resource struct {
	Partition string `xml:"partition"`
}

type Memory struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

// MemoryBacking mirroring libvirt XML under https://libvirt.org/formatdomain.html#elementsMemoryBacking
type MemoryBacking struct {
	HugePages *HugePages `xml:"hugepages,omitempty"`
}

// HugePages mirroring libvirt XML under memoryBacking
type HugePages struct {
	HugePage []HugePage `xml:"page,omitempty"`
}

// HugePage mirroring libvirt XML under hugepages
type HugePage struct {
	Size string `xml:"size,attr"`
	Unit string `xml:"unit,attr"`
}

type Devices struct {
	Emulator    string       `xml:"emulator,omitempty"`
	Interfaces  []Interface  `xml:"interface"`
	Channels    []Channel    `xml:"channel"`
	HostDevices []HostDevice `xml:"hostdev,omitempty"`
	Controllers []Controller `xml:"controller,omitempty"`
	Video       []Video      `xml:"video"`
	Graphics    []Graphics   `xml:"graphics"`
	Ballooning  *Ballooning  `xml:"memballoon,omitempty"`
	Disks       []Disk       `xml:"disk"`
	Serials     []Serial     `xml:"serial"`
	Consoles    []Console    `xml:"console"`
	Watchdog    *Watchdog    `xml:"watchdog,omitempty"`
	Rng         *Rng         `xml:"rng,omitempty"`
}

// BEGIN HostDevice -----------------------------
type HostDevice struct {
	Source    HostDeviceSource `xml:"source"`
	Type      string           `xml:"type,attr"`
	BootOrder *BootOrder       `xml:"boot,omitempty"`
	Managed   string           `xml:"managed,attr"`
}

type HostDeviceSource struct {
	Address *Address `xml:"address,omitempty"`
}

// END HostDevice -----------------------------

// BEGIN Controller -----------------------------

// Controller represens libvirt controller element https://libvirt.org/formatdomain.html#elementsControllers
type Controller struct {
	Type   string            `xml:"type,attr"`
	Index  string            `xml:"index,attr"`
	Model  string            `xml:"model,attr,omitempty"`
	Driver *ControllerDriver `xml:"driver,omitempty"`
}

// END Controller -----------------------------

// BEGIN ControllerDriver
type ControllerDriver struct {
	IOThread *uint `xml:"iothread,attr,omitempty"`
}

// END ControllerDriver

// BEGIN Disk -----------------------------

type Disk struct {
	Device       string        `xml:"device,attr"`
	Snapshot     string        `xml:"snapshot,attr,omitempty"`
	Type         string        `xml:"type,attr"`
	Source       DiskSource    `xml:"source"`
	Target       DiskTarget    `xml:"target"`
	Serial       string        `xml:"serial,omitempty"`
	Driver       *DiskDriver   `xml:"driver,omitempty"`
	ReadOnly     *ReadOnly     `xml:"readonly,omitempty"`
	Auth         *DiskAuth     `xml:"auth,omitempty"`
	Alias        *Alias        `xml:"alias,omitempty"`
	BackingStore *BackingStore `xml:"backingStore,omitempty"`
	BootOrder    *BootOrder    `xml:"boot,omitempty"`
	Address      *Address      `xml:"address,omitempty"`
}

type DiskAuth struct {
	Username string      `xml:"username,attr"`
	Secret   *DiskSecret `xml:"secret,omitempty"`
}

type DiskSecret struct {
	Type  string `xml:"type,attr"`
	Usage string `xml:"usage,attr,omitempty"`
	UUID  string `xml:"uuid,attr,omitempty"`
}

type ReadOnly struct{}

type DiskSource struct {
	Dev           string          `xml:"dev,attr,omitempty"`
	File          string          `xml:"file,attr,omitempty"`
	StartupPolicy string          `xml:"startupPolicy,attr,omitempty"`
	Protocol      string          `xml:"protocol,attr,omitempty"`
	Name          string          `xml:"name,attr,omitempty"`
	Host          *DiskSourceHost `xml:"host,omitempty"`
}

type DiskTarget struct {
	Bus    string `xml:"bus,attr,omitempty"`
	Device string `xml:"dev,attr,omitempty"`
	Tray   string `xml:"tray,attr,omitempty"`
}

type DiskDriver struct {
	Cache       string `xml:"cache,attr,omitempty"`
	ErrorPolicy string `xml:"error_policy,attr,omitempty"`
	IO          string `xml:"io,attr,omitempty"`
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	IOThread    *uint  `xml:"iothread,attr,omitempty"`
	Queues      *uint  `xml:"queues,attr,omitempty"`
}

type DiskSourceHost struct {
	Name string `xml:"name,attr"`
	Port string `xml:"port,attr,omitempty"`
}

type BackingStore struct {
	Type   string              `xml:"type,attr,omitempty"`
	Format *BackingStoreFormat `xml:"format,omitempty"`
	Source *DiskSource         `xml:"source,omitempty"`
}

type BackingStoreFormat struct {
	Type string `xml:"type,attr"`
}

// END Disk -----------------------------

// BEGIN Serial -----------------------------

type Serial struct {
	Type   string        `xml:"type,attr"`
	Target *SerialTarget `xml:"target,omitempty"`
	Source *SerialSource `xml:"source,omitempty"`
	Alias  *Alias        `xml:"alias,omitempty"`
}

type SerialTarget struct {
	Port *uint `xml:"port,attr,omitempty"`
}

type SerialSource struct {
	Mode string `xml:"mode,attr,omitempty"`
	Path string `xml:"path,attr,omitempty"`
}

// END Serial -----------------------------

// BEGIN Console -----------------------------

type Console struct {
	Type   string         `xml:"type,attr"`
	Target *ConsoleTarget `xml:"target,omitempty"`
	Source *ConsoleSource `xml:"source,omitempty"`
	Alias  *Alias         `xml:"alias,omitempty"`
}

type ConsoleTarget struct {
	Type *string `xml:"type,attr,omitempty"`
	Port *uint   `xml:"port,attr,omitempty"`
}

type ConsoleSource struct {
	Mode string `xml:"mode,attr,omitempty"`
	Path string `xml:"path,attr,omitempty"`
}

// END Serial -----------------------------

// BEGIN Inteface -----------------------------

type Interface struct {
	Address             *Address         `xml:"address,omitempty"`
	Type                string           `xml:"type,attr"`
	TrustGuestRxFilters string           `xml:"trustGuestRxFilters,attr,omitempty"`
	Source              InterfaceSource  `xml:"source"`
	Target              *InterfaceTarget `xml:"target,omitempty"`
	Model               *Model           `xml:"model,omitempty"`
	MAC                 *MAC             `xml:"mac,omitempty"`
	MTU                 *MTU             `xml:"mtu,omitempty"`
	BandWidth           *BandWidth       `xml:"bandwidth,omitempty"`
	BootOrder           *BootOrder       `xml:"boot,omitempty"`
	LinkState           *LinkState       `xml:"link,omitempty"`
	FilterRef           *FilterRef       `xml:"filterref,omitempty"`
	Alias               *Alias           `xml:"alias,omitempty"`
	Driver              *InterfaceDriver `xml:"driver,omitempty"`
}

type InterfaceDriver struct {
	Name   string `xml:"name,attr"`
	Queues *uint  `xml:"queues,attr,omitempty"`
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

type MTU struct {
	Size string `xml:"size,attr"`
}

type FilterRef struct {
	Filter string `xml:"filter,attr"`
}

type InterfaceSource struct {
	Network string   `xml:"network,attr,omitempty"`
	Device  string   `xml:"dev,attr,omitempty"`
	Bridge  string   `xml:"bridge,attr,omitempty"`
	Mode    string   `xml:"mode,attr,omitempty"`
	Address *Address `xml:"address,omitempty"`
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

type UserAlias Alias

func (alias Alias) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	userAlias := UserAlias(alias)
	userAlias.Name = UserAliasPrefix + userAlias.Name
	return e.EncodeElement(userAlias, start)
}

func (alias *Alias) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var userAlias UserAlias
	err := d.DecodeElement(&userAlias, &start)
	if err != nil {
		return err
	}
	*alias = Alias(userAlias)
	alias.Name = alias.Name[len(UserAliasPrefix):]
	return nil
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
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

//END OS --------------------

//BEGIN Clock --------------------

type Clock struct {
	Offset     string  `xml:"offset,attr,omitempty"`
	Adjustment string  `xml:"adjustment,attr,omitempty"`
	Timer      []Timer `xml:"timer,omitempty"`
}

type Timer struct {
	Name       string `xml:"name,attr"`
	TickPolicy string `xml:"tickpolicy,attr,omitempty"`
	Present    string `xml:"present,attr,omitempty"`
	Track      string `xml:"track,attr,omitempty"`
}

//END Clock --------------------

//BEGIN Channel --------------------

type Channel struct {
	Type   string         `xml:"type,attr"`
	Source *ChannelSource `xml:"source,omitempty"`
	Target *ChannelTarget `xml:"target,omitempty"`
}

type ChannelTarget struct {
	Name    string `xml:"name,attr,omitempty"`
	Type    string `xml:"type,attr"`
	Address string `xml:"address,attr,omitempty"`
	Port    uint   `xml:"port,attr,omitempty"`
	State   string `xml:"state,attr,omitempty"`
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
	AutoPort      string          `xml:"autoPort,attr,omitempty"`
	DefaultMode   string          `xml:"defaultMode,attr,omitempty"`
	Listen        *GraphicsListen `xml:"listen,omitempty"`
	PasswdValidTo string          `xml:"passwdValidTo,attr,omitempty"`
	Port          int32           `xml:"port,attr,omitempty"`
	TLSPort       int             `xml:"tlsPort,attr,omitempty"`
	Type          string          `xml:"type,attr"`
}

type GraphicsListen struct {
	Type    string `xml:"type,attr"`
	Address string `xml:"address,attr,omitempty"`
	Network string `xml:"newtork,attr,omitempty"`
	Socket  string `xml:"socket,attr,omitempty"`
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

type Watchdog struct {
	Model  string `xml:"model,attr"`
	Action string `xml:"action,attr"`
	Alias  *Alias `xml:"alias,omitempty"`
}

// Rng represents the source of entropy from host to VM
type Rng struct {
	// Model attribute specifies what type of RNG device is provided
	Model string `xml:"model,attr"`
	// Backend specifies the source of entropy to be used
	Backend *RngBackend `xml:"backend,omitempty"`
}

// RngRate sets the limiting factor how to read from entropy source
type RngRate struct {
	// Period define how long is the read period
	Period uint32 `xml:"period,attr"`
	// Bytes define how many bytes can guest read from entropy source
	Bytes uint32 `xml:"bytes,attr"`
}

// RngBackend is the backend device used
type RngBackend struct {
	// Model is source model
	Model string `xml:"model,attr"`
	// specifies the source of entropy to be used
	Source string `xml:",chardata"`
}

type IOThreads struct {
	IOThreads uint `xml:",chardata"`
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

func NewMinimalDomainSpec(vmiName string) *DomainSpec {
	precond.MustNotBeEmpty(vmiName)
	domain := &DomainSpec{}
	domain.Name = vmiName
	domain.Memory = Memory{Unit: "MB", Value: 9}
	domain.Devices = Devices{}
	return domain
}

func NewMinimalDomain(name string) *Domain {
	return NewMinimalDomainWithNS(kubev1.NamespaceDefault, name)
}

func NewMinimalDomainWithUUID(name string, uuid types.UID) *Domain {
	domain := NewMinimalDomainWithNS(kubev1.NamespaceDefault, name)
	domain.Spec.Metadata = Metadata{
		KubeVirt: KubeVirtMetadata{
			UID: uuid,
		},
	}
	return domain
}

func NewMinimalDomainWithNS(namespace string, name string) *Domain {
	domain := NewDomainReferenceFromName(namespace, name)
	domain.Spec = *NewMinimalDomainSpec(namespace + "_" + name)
	return domain
}

func NewDomainReferenceFromName(namespace string, name string) *Domain {
	return &Domain{
		Spec: DomainSpec{},
		ObjectMeta: metav1.ObjectMeta{
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

// VMINamespaceKeyFunc constructs the domain name with a namespace prefix i.g.
// namespace_name.
func VMINamespaceKeyFunc(vmi *v1.VirtualMachineInstance) string {
	domName := fmt.Sprintf("%s_%s", vmi.Namespace, vmi.Name)
	return domName
}
