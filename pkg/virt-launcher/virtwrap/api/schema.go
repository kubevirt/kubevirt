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

package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/precond"
)

// For versioning of the virt-handler and -launcher communication,
// you need to increase the Version const when making changes,
// and make necessary changes in the cmd and notify rpc implementation!
const (
	DomainVersion = "v1"
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

	FSThawed      = "thawed"
	FSFrozen      = "frozen"
	SchedulerFIFO = "fifo"

	HostDevicePCI  = "pci"
	HostDeviceMDev = "mdev"
	HostDeviceUSB  = "usb"
	AddressPCI     = "pci"
	AddressCCW     = "ccw"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Domain struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"ObjectMeta"`
	Spec              DomainSpec
	Status            DomainStatus
}

type DomainStatus struct {
	Status         LifeCycle
	Reason         StateChangeReason
	Interfaces     []InterfaceStatus
	OSInfo         GuestOSInfo
	FSFreezeStatus FSFreeze
}

type DomainSysInfo struct {
	Hostname string
	OSInfo   GuestOSInfo
	Timezone Timezone
}

type GuestOSInfo struct {
	Name          string
	KernelRelease string
	Version       string
	PrettyName    string
	VersionId     string
	KernelVersion string
	Machine       string
	Id            string
}

type InterfaceStatus struct {
	Mac           string
	Ip            string
	IPs           []string
	InterfaceName string
}

type SEVNodeParameters struct {
	PDH       string
	CertChain string
}

type Timezone struct {
	Zone   string
	Offset int
}

type FSFreeze struct {
	Status string
}

type FSDisk struct {
	Serial  string
	BusType string
}

type Filesystem struct {
	Name       string
	Mountpoint string
	Type       string
	UsedBytes  int
	TotalBytes int
	Disk       []FSDisk
}

type User struct {
	Name      string
	Domain    string
	LoginTime float64
}

// DomainGuestInfo represent guest agent info for specific domain
type DomainGuestInfo struct {
	Interfaces     []InterfaceStatus
	OSInfo         *GuestOSInfo
	FSFreezeStatus *FSFreeze
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
	XMLName        xml.Name        `xml:"domain"`
	Type           string          `xml:"type,attr"`
	XmlNS          string          `xml:"xmlns:qemu,attr,omitempty"`
	Name           string          `xml:"name"`
	UUID           string          `xml:"uuid,omitempty"`
	Memory         Memory          `xml:"memory"`
	CurrentMemory  *Memory         `xml:"currentMemory,omitempty"`
	MaxMemory      *MaxMemory      `xml:"maxMemory,omitempty"`
	MemoryBacking  *MemoryBacking  `xml:"memoryBacking,omitempty"`
	OS             OS              `xml:"os"`
	SysInfo        *SysInfo        `xml:"sysinfo,omitempty"`
	Devices        Devices         `xml:"devices"`
	Clock          *Clock          `xml:"clock,omitempty"`
	Resource       *Resource       `xml:"resource,omitempty"`
	QEMUCmd        *Commandline    `xml:"qemu:commandline,omitempty"`
	Metadata       Metadata        `xml:"metadata,omitempty"`
	Features       *Features       `xml:"features,omitempty"`
	CPU            CPU             `xml:"cpu"`
	VCPU           *VCPU           `xml:"vcpu"`
	VCPUs          *VCPUs          `xml:"vcpus"`
	CPUTune        *CPUTune        `xml:"cputune"`
	NUMATune       *NUMATune       `xml:"numatune"`
	IOThreads      *IOThreads      `xml:"iothreads,omitempty"`
	LaunchSecurity *LaunchSecurity `xml:"launchSecurity,omitempty"`
}

type CPUTune struct {
	VCPUPin     []CPUTuneVCPUPin     `xml:"vcpupin"`
	IOThreadPin []CPUTuneIOThreadPin `xml:"iothreadpin,omitempty"`
	EmulatorPin *CPUEmulatorPin      `xml:"emulatorpin"`
}

type NUMATune struct {
	Memory   NumaTuneMemory `xml:"memory"`
	MemNodes []MemNode      `xml:"memnode"`
}

type MemNode struct {
	CellID  uint32 `xml:"cellid,attr"`
	Mode    string `xml:"mode,attr"`
	NodeSet string `xml:"nodeset,attr"`
}

type NumaTuneMemory struct {
	Mode    string `xml:"mode,attr"`
	NodeSet string `xml:"nodeset,attr"`
}

type CPUTuneVCPUPin struct {
	VCPU   uint32 `xml:"vcpu,attr"`
	CPUSet string `xml:"cpuset,attr"`
}

type CPUTuneIOThreadPin struct {
	IOThread uint32 `xml:"iothread,attr"`
	CPUSet   string `xml:"cpuset,attr"`
}

type CPUEmulatorPin struct {
	CPUSet string `xml:"cpuset,attr"`
}

type VCPU struct {
	Placement string `xml:"placement,attr"`
	CPUs      uint32 `xml:",chardata"`
}

type VCPUsVCPU struct {
	ID           uint32 `xml:"id,attr"`
	Enabled      string `xml:"enabled,attr,omitempty"`
	Hotpluggable string `xml:"hotpluggable,attr,omitempty"`
	Order        uint32 `xml:"order,attr,omitempty"`
}

type VCPUs struct {
	VCPU []VCPUsVCPU `xml:"vcpu"`
}

type CPU struct {
	Mode     string       `xml:"mode,attr,omitempty"`
	Model    string       `xml:"model,omitempty"`
	Features []CPUFeature `xml:"feature"`
	Topology *CPUTopology `xml:"topology"`
	NUMA     *NUMA        `xml:"numa,omitempty"`
}

type NUMA struct {
	Cells []NUMACell `xml:"cell"`
}

type NUMACell struct {
	ID           string `xml:"id,attr"`
	CPUs         string `xml:"cpus,attr"`
	Memory       uint64 `xml:"memory,attr,omitempty"`
	Unit         string `xml:"unit,attr,omitempty"`
	MemoryAccess string `xml:"memAccess,attr,omitempty"`
}

type CPUFeature struct {
	Name   string `xml:"name,attr"`
	Policy string `xml:"policy,attr,omitempty"`
}

type CPUTopology struct {
	Sockets uint32 `xml:"sockets,attr,omitempty"`
	Cores   uint32 `xml:"cores,attr,omitempty"`
	Threads uint32 `xml:"threads,attr,omitempty"`
}

type Features struct {
	ACPI       *FeatureEnabled    `xml:"acpi,omitempty"`
	APIC       *FeatureEnabled    `xml:"apic,omitempty"`
	Hyperv     *FeatureHyperv     `xml:"hyperv,omitempty"`
	SMM        *FeatureEnabled    `xml:"smm,omitempty"`
	KVM        *FeatureKVM        `xml:"kvm,omitempty"`
	PVSpinlock *FeaturePVSpinlock `xml:"pvspinlock,omitempty"`
	PMU        *FeatureState      `xml:"pmu,omitempty"`
	VMPort     *FeatureState      `xml:"vmport,omitempty"`
}

const HypervModePassthrough = "passthrough"

type FeatureHyperv struct {
	Mode            string            `xml:"mode,attr,omitempty"`
	Relaxed         *FeatureState     `xml:"relaxed,omitempty"`
	VAPIC           *FeatureState     `xml:"vapic,omitempty"`
	Spinlocks       *FeatureSpinlocks `xml:"spinlocks,omitempty"`
	VPIndex         *FeatureState     `xml:"vpindex,omitempty"`
	Runtime         *FeatureState     `xml:"runtime,omitempty"`
	SyNIC           *FeatureState     `xml:"synic,omitempty"`
	SyNICTimer      *SyNICTimer       `xml:"stimer,omitempty"`
	Reset           *FeatureState     `xml:"reset,omitempty"`
	VendorID        *FeatureVendorID  `xml:"vendor_id,omitempty"`
	Frequencies     *FeatureState     `xml:"frequencies,omitempty"`
	Reenlightenment *FeatureState     `xml:"reenlightenment,omitempty"`
	TLBFlush        *FeatureState     `xml:"tlbflush,omitempty"`
	IPI             *FeatureState     `xml:"ipi,omitempty"`
	EVMCS           *FeatureState     `xml:"evmcs,omitempty"`
}

type FeatureSpinlocks struct {
	State   string  `xml:"state,attr,omitempty"`
	Retries *uint32 `xml:"retries,attr,omitempty"`
}

type SyNICTimer struct {
	Direct *FeatureState `xml:"direct,omitempty"`
	State  string        `xml:"state,attr,omitempty"`
}

type FeaturePVSpinlock struct {
	State string `xml:"state,attr,omitempty"`
}

type FeatureVendorID struct {
	State string `xml:"state,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
}

type FeatureEnabled struct {
}

type Shareable struct{}

type Slice struct {
	Slice SliceType `xml:"slice,omitempty"`
}

type SliceType struct {
	Type   string `xml:"type,attr"`
	Offset int64  `xml:"offset,attr"`
	Size   int64  `xml:"size,attr"`
}

type FeatureState struct {
	State string `xml:"state,attr,omitempty"`
}

type FeatureKVM struct {
	Hidden        *FeatureState `xml:"hidden,omitempty"`
	HintDedicated *FeatureState `xml:"hint-dedicated,omitempty"`
}

type Metadata struct {
	// KubeVirt contains kubevirt related metadata
	// Note: Libvirt only accept one element at metadata root with a specific namespace
	KubeVirt KubeVirtMetadata `xml:"http://kubevirt.io kubevirt"`
}

type KubeVirtMetadata struct {
	UID              types.UID                 `xml:"uid"`
	GracePeriod      *GracePeriodMetadata      `xml:"graceperiod,omitempty"`
	Migration        *MigrationMetadata        `xml:"migration,omitempty"`
	AccessCredential *AccessCredentialMetadata `xml:"accessCredential,omitempty"`
	MemoryDump       *MemoryDumpMetadata       `xml:"memoryDump,omitempty"`
}

type AccessCredentialMetadata struct {
	Succeeded bool   `xml:"succeeded,omitempty"`
	Message   string `xml:"message,omitempty"`
}

type MemoryDumpMetadata struct {
	FileName       string       `xml:"fileName,omitempty"`
	StartTimestamp *metav1.Time `xml:"startTimestamp,omitempty"`
	EndTimestamp   *metav1.Time `xml:"endTimestamp,omitempty"`
	Completed      bool         `xml:"completed,omitempty"`
	Failed         bool         `xml:"failed,omitempty"`
	FailureReason  string       `xml:"failureReason,omitempty"`
}

type MigrationMetadata struct {
	UID            types.UID        `xml:"uid,omitempty"`
	StartTimestamp *metav1.Time     `xml:"startTimestamp,omitempty"`
	EndTimestamp   *metav1.Time     `xml:"endTimestamp,omitempty"`
	Completed      bool             `xml:"completed,omitempty"`
	Failed         bool             `xml:"failed,omitempty"`
	FailureReason  string           `xml:"failureReason,omitempty"`
	AbortStatus    string           `xml:"abortStatus,omitempty"`
	Mode           v1.MigrationMode `xml:"mode,omitempty"`
}

type GracePeriodMetadata struct {
	DeletionGracePeriodSeconds int64        `xml:"deletionGracePeriodSeconds"`
	DeletionTimestamp          *metav1.Time `xml:"deletionTimestamp,omitempty"`
	MarkedForGracefulShutdown  *bool        `xml:"markedForGracefulShutdown,omitempty"`
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

type MaxMemory struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
	Slots uint64 `xml:"slots,attr"`
}

// MemoryBacking mirroring libvirt XML under https://libvirt.org/formatdomain.html#elementsMemoryBacking
type MemoryBacking struct {
	HugePages    *HugePages           `xml:"hugepages,omitempty"`
	Source       *MemoryBackingSource `xml:"source,omitempty"`
	Access       *MemoryBackingAccess `xml:"access,omitempty"`
	Allocation   *MemoryAllocation    `xml:"allocation,omitempty"`
	NoSharePages *NoSharePages        `xml:"nosharepages,omitempty"`
}

type MemoryAllocationMode string

const (
	MemoryAllocationModeImmediate MemoryAllocationMode = "immediate"
)

type MemoryAllocation struct {
	Mode MemoryAllocationMode `xml:"mode,attr"`
}

type MemoryBackingSource struct {
	Type string `xml:"type,attr"`
}

// HugePages mirroring libvirt XML under memoryBacking
type HugePages struct {
	HugePage []HugePage `xml:"page,omitempty"`
}

// HugePage mirroring libvirt XML under hugepages
type HugePage struct {
	Size    string `xml:"size,attr"`
	Unit    string `xml:"unit,attr"`
	NodeSet string `xml:"nodeset,attr"`
}

type MemoryBackingAccess struct {
	Mode string `xml:"mode,attr"`
}

type NoSharePages struct {
}

type MemoryAddress struct {
	Base string `xml:"base,attr"`
}

type MemoryTarget struct {
	Size      Memory         `xml:"size"`
	Requested Memory         `xml:"requested"`
	Current   Memory         `xml:"current"`
	Node      string         `xml:"node"`
	Block     Memory         `xml:"block"`
	Address   *MemoryAddress `xml:"address,omitempty"`
}

type MemoryDevice struct {
	XMLName xml.Name      `xml:"memory"`
	Model   string        `xml:"model,attr"`
	Target  *MemoryTarget `xml:"target"`
	Alias   *Alias        `xml:"alias,omitempty"`
	Address *Address      `xml:"address,omitempty"`
}

type Devices struct {
	Emulator    string             `xml:"emulator,omitempty"`
	Interfaces  []Interface        `xml:"interface"`
	Channels    []Channel          `xml:"channel"`
	HostDevices []HostDevice       `xml:"hostdev,omitempty"`
	Controllers []Controller       `xml:"controller,omitempty"`
	Video       []Video            `xml:"video"`
	Graphics    []Graphics         `xml:"graphics"`
	Ballooning  *MemBalloon        `xml:"memballoon,omitempty"`
	Disks       []Disk             `xml:"disk"`
	Inputs      []Input            `xml:"input"`
	Serials     []Serial           `xml:"serial"`
	Consoles    []Console          `xml:"console"`
	Watchdogs   []Watchdog         `xml:"watchdog,omitempty"`
	Rng         *Rng               `xml:"rng,omitempty"`
	Filesystems []FilesystemDevice `xml:"filesystem,omitempty"`
	Redirs      []RedirectedDevice `xml:"redirdev,omitempty"`
	SoundCards  []SoundCard        `xml:"sound,omitempty"`
	TPMs        []TPM              `xml:"tpm,omitempty"`
	VSOCK       *VSOCK             `xml:"vsock,omitempty"`
	Memory      *MemoryDevice      `xml:"memory,omitempty"`
}

type TPM struct {
	Model   string     `xml:"model,attr"`
	Backend TPMBackend `xml:"backend"`
}

type TPMBackend struct {
	Type            string `xml:"type,attr"`
	Version         string `xml:"version,attr"`
	PersistentState string `xml:"persistent_state,attr,omitempty"`
}

// RedirectedDevice describes a device to be redirected
// See: https://libvirt.org/formatdomain.html#redirected-devices
type RedirectedDevice struct {
	Type   string                 `xml:"type,attr"`
	Bus    string                 `xml:"bus,attr"`
	Source RedirectedDeviceSource `xml:"source"`
}

type RedirectedDeviceSource struct {
	Mode string `xml:"mode,attr"`
	Path string `xml:"path,attr"`
}

type FilesystemDevice struct {
	Type       string            `xml:"type,attr"`
	AccessMode string            `xml:"accessMode,attr"`
	Source     *FilesystemSource `xml:"source,omitempty"`
	Target     *FilesystemTarget `xml:"target,omitempty"`
	Driver     *FilesystemDriver `xml:"driver,omitempty"`
	Binary     *FilesystemBinary `xml:"binary,omitempty"`
}

type FilesystemTarget struct {
	Dir string `xml:"dir,attr,omitempty"`
}

type FilesystemSource struct {
	Dir    string `xml:"dir,attr"`
	Socket string `xml:"socket,attr,omitempty"`
}

type FilesystemDriver struct {
	Type  string `xml:"type,attr"`
	Queue string `xml:"queue,attr,omitempty"`
}

type FilesystemBinary struct {
	Path  string                 `xml:"path,attr,omitempty"`
	Xattr string                 `xml:"xattr,attr,omitempty"`
	Cache *FilesystemBinaryCache `xml:"cache,omitempty"`
	Lock  *FilesystemBinaryLock  `xml:"lock,omitempty"`
}

type FilesystemBinaryCache struct {
	Mode string `xml:"mode,attr,omitempty"`
}

type FilesystemBinaryLock struct {
	Posix string `xml:"posix,attr,omitempty"`
	Flock string `xml:"flock,attr,omitempty"`
}

// Input represents input device, e.g. tablet
type Input struct {
	Type    v1.InputType `xml:"type,attr"`
	Bus     v1.InputBus  `xml:"bus,attr"`
	Alias   *Alias       `xml:"alias,omitempty"`
	Address *Address     `xml:"address,omitempty"`
	Model   string       `xml:"model,attr,omitempty"`
}

// BEGIN HostDevice -----------------------------
type HostDevice struct {
	XMLName   xml.Name         `xml:"hostdev"`
	Source    HostDeviceSource `xml:"source"`
	Type      string           `xml:"type,attr"`
	BootOrder *BootOrder       `xml:"boot,omitempty"`
	Managed   string           `xml:"managed,attr,omitempty"`
	Mode      string           `xml:"mode,attr,omitempty"`
	Model     string           `xml:"model,attr,omitempty"`
	Address   *Address         `xml:"address,omitempty"`
	Alias     *Alias           `xml:"alias,omitempty"`
	Display   string           `xml:"display,attr,omitempty"`
	RamFB     string           `xml:"ramfb,attr,omitempty"`
}

type HostDeviceSource struct {
	Address *Address `xml:"address,omitempty"`
}

// END HostDevice -----------------------------

// BEGIN Controller -----------------------------

// Controller represens libvirt controller element https://libvirt.org/formatdomain.html#elementsControllers
type Controller struct {
	Type    string            `xml:"type,attr"`
	Index   string            `xml:"index,attr"`
	Model   string            `xml:"model,attr,omitempty"`
	Driver  *ControllerDriver `xml:"driver,omitempty"`
	Alias   *Alias            `xml:"alias,omitempty"`
	Address *Address          `xml:"address,omitempty"`
}

// END Controller -----------------------------

// BEGIN ControllerDriver
type ControllerDriver struct {
	IOThread *uint  `xml:"iothread,attr,omitempty"`
	Queues   *uint  `xml:"queues,attr,omitempty"`
	IOMMU    string `xml:"iommu,attr,omitempty"`
}

// END ControllerDriver

// BEGIN Disk -----------------------------

type Disk struct {
	Device             string        `xml:"device,attr"`
	Snapshot           string        `xml:"snapshot,attr,omitempty"`
	Type               string        `xml:"type,attr"`
	Source             DiskSource    `xml:"source"`
	Target             DiskTarget    `xml:"target"`
	Serial             string        `xml:"serial,omitempty"`
	Driver             *DiskDriver   `xml:"driver,omitempty"`
	ReadOnly           *ReadOnly     `xml:"readonly,omitempty"`
	Auth               *DiskAuth     `xml:"auth,omitempty"`
	Alias              *Alias        `xml:"alias,omitempty"`
	BackingStore       *BackingStore `xml:"backingStore,omitempty"`
	BootOrder          *BootOrder    `xml:"boot,omitempty"`
	Address            *Address      `xml:"address,omitempty"`
	Model              string        `xml:"model,attr,omitempty"`
	BlockIO            *BlockIO      `xml:"blockio,omitempty"`
	FilesystemOverhead *v1.Percent   `xml:"filesystemOverhead,omitempty"`
	Capacity           *int64        `xml:"capacity,omitempty"`
	ExpandDisksEnabled bool          `xml:"expandDisksEnabled,omitempty"`
	Shareable          *Shareable    `xml:"shareable,omitempty"`
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
	Reservations  *Reservations   `xml:"reservations,omitempty"`
	Slices        []Slice         `xml:"slices,omitempty"`
}

type DiskTarget struct {
	Bus    v1.DiskBus `xml:"bus,attr,omitempty"`
	Device string     `xml:"dev,attr,omitempty"`
	Tray   string     `xml:"tray,attr,omitempty"`
}

type DiskDriver struct {
	Cache       string             `xml:"cache,attr,omitempty"`
	ErrorPolicy v1.DiskErrorPolicy `xml:"error_policy,attr,omitempty"`
	IO          v1.DriverIO        `xml:"io,attr,omitempty"`
	Name        string             `xml:"name,attr"`
	Type        string             `xml:"type,attr"`
	IOThread    *uint              `xml:"iothread,attr,omitempty"`
	IOThreads   *DiskIOThreads     `xml:"iothreads"`
	Queues      *uint              `xml:"queues,attr,omitempty"`
	Discard     string             `xml:"discard,attr,omitempty"`
	IOMMU       string             `xml:"iommu,attr,omitempty"`
}

type DiskIOThreads struct {
	IOThread []DiskIOThread `xml:"iothread"`
}

type DiskIOThread struct {
	Id uint32 `xml:"id,attr"`
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

type BlockIO struct {
	LogicalBlockSize  uint `xml:"logical_block_size,attr,omitempty"`
	PhysicalBlockSize uint `xml:"physical_block_size,attr,omitempty"`
}

type Reservations struct {
	Managed            string              `xml:"managed,attr,omitempty"`
	SourceReservations *SourceReservations `xml:"source,omitempty"`
}

type SourceReservations struct {
	Type string `xml:"type,attr"`
	Path string `xml:"path,attr,omitempty"`
	Mode string `xml:"mode,attr,omitempty"`
}

// END Disk -----------------------------

// BEGIN Serial -----------------------------

type Serial struct {
	Type   string        `xml:"type,attr"`
	Target *SerialTarget `xml:"target,omitempty"`
	Source *SerialSource `xml:"source,omitempty"`
	Alias  *Alias        `xml:"alias,omitempty"`
	Log    *SerialLog    `xml:"log,omitempty"`
}

type SerialTarget struct {
	Port *uint `xml:"port,attr,omitempty"`
}

type SerialSource struct {
	Mode string `xml:"mode,attr,omitempty"`
	Path string `xml:"path,attr,omitempty"`
}

type SerialLog struct {
	File   string `xml:"file,attr,omitempty"`
	Append string `xml:"append,attr,omitempty"`
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
	XMLName             xml.Name               `xml:"interface"`
	Address             *Address               `xml:"address,omitempty"`
	Type                string                 `xml:"type,attr"`
	TrustGuestRxFilters string                 `xml:"trustGuestRxFilters,attr,omitempty"`
	Source              InterfaceSource        `xml:"source"`
	Target              *InterfaceTarget       `xml:"target,omitempty"`
	Model               *Model                 `xml:"model,omitempty"`
	MAC                 *MAC                   `xml:"mac,omitempty"`
	MTU                 *MTU                   `xml:"mtu,omitempty"`
	BandWidth           *BandWidth             `xml:"bandwidth,omitempty"`
	BootOrder           *BootOrder             `xml:"boot,omitempty"`
	LinkState           *LinkState             `xml:"link,omitempty"`
	FilterRef           *FilterRef             `xml:"filterref,omitempty"`
	Alias               *Alias                 `xml:"alias,omitempty"`
	Driver              *InterfaceDriver       `xml:"driver,omitempty"`
	Rom                 *Rom                   `xml:"rom,omitempty"`
	ACPI                *ACPI                  `xml:"acpi,omitempty"`
	Backend             *InterfaceBackend      `xml:"backend,omitempty"`
	PortForward         []InterfacePortForward `xml:"portForward,omitempty"`
}

type InterfacePortForward struct {
	Proto   string                      `xml:"proto,attr"`
	Address string                      `xml:"address,attr,omitempty"`
	Dev     string                      `xml:"dev,attr,omitempty"`
	Ranges  []InterfacePortForwardRange `xml:"range,omitempty"`
}

type InterfacePortForwardRange struct {
	Start   uint   `xml:"start,attr"`
	End     uint   `xml:"end,attr,omitempty"`
	To      uint   `xml:"to,attr,omitempty"`
	Exclude string `xml:"exclude,attr,omitempty"`
}

type InterfaceBackend struct {
	Type    string `xml:"type,attr,omitempty"`
	LogFile string `xml:"logFile,attr,omitempty"`
}

type ACPI struct {
	Index uint `xml:"index,attr"`
}

type InterfaceDriver struct {
	Name   string `xml:"name,attr"`
	Queues *uint  `xml:"queues,attr,omitempty"`
	IOMMU  string `xml:"iommu,attr,omitempty"`
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
	Device  string `xml:"dev,attr"`
	Managed string `xml:"managed,attr,omitempty"`
}

type Alias struct {
	name        string
	userDefined bool
}

// Package private, responsible to interact with xml and json marshal/unmarshal
type userAliasMarshal struct {
	Name        string `xml:"name,attr"`
	UserDefined bool   `xml:"-"`
}

type Rom struct {
	Enabled string `xml:"enabled,attr"`
}

func NewUserDefinedAlias(aliasName string) *Alias {
	return &Alias{name: aliasName, userDefined: true}
}

func (alias Alias) GetName() string {
	return alias.name
}

func (alias Alias) IsUserDefined() bool {
	return alias.userDefined
}

func (alias Alias) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	userAlias := userAliasMarshal{Name: alias.name}
	if alias.userDefined {
		userAlias.Name = UserAliasPrefix + userAlias.Name
	}
	return e.EncodeElement(userAlias, start)
}

func (alias *Alias) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var userAlias userAliasMarshal
	err := d.DecodeElement(&userAlias, &start)
	if err != nil {
		return err
	}
	*alias = Alias{name: userAlias.Name}
	if strings.HasPrefix(alias.name, UserAliasPrefix) {
		alias.userDefined = true
		alias.name = alias.name[len(UserAliasPrefix):]
	}
	return nil
}

func (alias Alias) MarshalJSON() ([]byte, error) {
	userAlias := userAliasMarshal{Name: alias.name, UserDefined: alias.userDefined}
	return json.Marshal(&userAlias)
}

func (alias *Alias) UnmarshalJSON(data []byte) error {
	var userAlias userAliasMarshal
	if err := json.Unmarshal(data, &userAlias); err != nil {
		return err
	}
	*alias = Alias{name: userAlias.Name, userDefined: userAlias.UserDefined}
	return nil
}

// END Inteface -----------------------------
//BEGIN OS --------------------

type OS struct {
	Type       OSType    `xml:"type"`
	ACPI       *OSACPI   `xml:"acpi,omitempty"`
	SMBios     *SMBios   `xml:"smbios,omitempty"`
	BootOrder  []Boot    `xml:"boot"`
	BootMenu   *BootMenu `xml:"bootmenu,omitempty"`
	BIOS       *BIOS     `xml:"bios,omitempty"`
	BootLoader *Loader   `xml:"loader,omitempty"`
	NVRam      *NVRam    `xml:"nvram,omitempty"`
	Kernel     string    `xml:"kernel,omitempty"`
	Initrd     string    `xml:"initrd,omitempty"`
	KernelArgs string    `xml:"cmdline,omitempty"`
}

type OSType struct {
	OS      string `xml:",chardata"`
	Arch    string `xml:"arch,attr,omitempty"`
	Machine string `xml:"machine,attr,omitempty"`
}

type OSACPI struct {
	Table ACPITable `xml:"table,omitempty"`
}

type ACPITable struct {
	Path string `xml:",chardata"`
	Type string `xml:"type,attr,omitempty"`
}

type SMBios struct {
	Mode string `xml:"mode,attr"`
}

type NVRam struct {
	Template string `xml:"template,attr,omitempty"`
	NVRam    string `xml:",chardata"`
}

type Boot struct {
	Dev string `xml:"dev,attr"`
}

type BootMenu struct {
	Enable  string `xml:"enable,attr"`
	Timeout *uint  `xml:"timeout,attr,omitempty"`
}

type Loader struct {
	ReadOnly string `xml:"readonly,attr,omitempty"`
	Secure   string `xml:"secure,attr,omitempty"`
	Type     string `xml:"type,attr,omitempty"`
	Path     string `xml:",chardata"`
}

// TODO <bios rebootTimeout='0'/>
type BIOS struct {
	UseSerial string `xml:"useserial,attr,omitempty"`
}

type SysInfo struct {
	Type      string  `xml:"type,attr"`
	System    []Entry `xml:"system>entry"`
	BIOS      []Entry `xml:"bios>entry"`
	BaseBoard []Entry `xml:"baseBoard>entry"`
	Chassis   []Entry `xml:"chassis>entry"`
}

type Entry struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

//END OS --------------------
//BEGIN LaunchSecurity --------------------

type LaunchSecurity struct {
	Type            string `xml:"type,attr"`
	Cbitpos         string `xml:"cbitpos,omitempty"`
	ReducedPhysBits string `xml:"reducedPhysBits,omitempty"`
	Policy          string `xml:"policy,omitempty"`
	DHCert          string `xml:"dhCert,omitempty"`
	Session         string `xml:"session,omitempty"`
}

//END LaunchSecurity --------------------
//BEGIN Clock --------------------

type Clock struct {
	Offset     string  `xml:"offset,attr,omitempty"`
	Timezone   string  `xml:"timezone,attr,omitempty"`
	Adjustment string  `xml:"adjustment,attr,omitempty"`
	Timer      []Timer `xml:"timer,omitempty"`
}

type Timer struct {
	Name       string `xml:"name,attr"`
	TickPolicy string `xml:"tickpolicy,attr,omitempty"`
	Present    string `xml:"present,attr,omitempty"`
	Track      string `xml:"track,attr,omitempty"`
	Frequency  string `xml:"frequency,attr,omitempty"`
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

//BEGIN Sound -------------------

type SoundCard struct {
	Alias *Alias `xml:"alias,omitempty"`
	Model string `xml:"model,attr"`
}

//END Sound -------------------

//BEGIN Video -------------------

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
	AutoPort      string          `xml:"autoport,attr,omitempty"`
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
	Type       string `xml:"type,attr"`
	Domain     string `xml:"domain,attr,omitempty"`
	Bus        string `xml:"bus,attr"`
	Slot       string `xml:"slot,attr,omitempty"`
	Function   string `xml:"function,attr,omitempty"`
	Controller string `xml:"controller,attr,omitempty"`
	Target     string `xml:"target,attr,omitempty"`
	Unit       string `xml:"unit,attr,omitempty"`
	UUID       string `xml:"uuid,attr,omitempty"`
	Device     string `xml:"device,attr,omitempty"`
}

//END Video -------------------

//BEGIN VSOCK -------------------

type VSOCK struct {
	Model string `xml:"model,attr,omitempty"`
	CID   CID    `xml:"cid"`
}

type CID struct {
	Auto    string `xml:"auto,attr"`
	Address uint32 `xml:"address,attr,omitempty"`
}

//END VSOCK -------------------

type Stats struct {
	Period uint `xml:"period,attr"`
}

type MemBalloon struct {
	Model             string            `xml:"model,attr"`
	Stats             *Stats            `xml:"stats,omitempty"`
	Address           *Address          `xml:"address,omitempty"`
	Driver            *MemBalloonDriver `xml:"driver,omitempty"`
	FreePageReporting string            `xml:"freePageReporting,attr,omitempty"`
}

type MemBalloonDriver struct {
	IOMMU string `xml:"iommu,attr,omitempty"`
}

type Watchdog struct {
	Model   string   `xml:"model,attr"`
	Action  string   `xml:"action,attr"`
	Alias   *Alias   `xml:"alias,omitempty"`
	Address *Address `xml:"address,omitempty"`
}

// Rng represents the source of entropy from host to VM
type Rng struct {
	// Model attribute specifies what type of RNG device is provided
	Model string `xml:"model,attr"`
	// Backend specifies the source of entropy to be used
	Backend *RngBackend `xml:"backend,omitempty"`
	Address *Address    `xml:"address,omitempty"`
	Driver  *RngDriver  `xml:"driver,omitempty"`
}

type RngDriver struct {
	IOMMU string `xml:"iommu,attr,omitempty"`
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
