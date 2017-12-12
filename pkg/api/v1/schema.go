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

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

//go:generate swagger-doc

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

// Represents a cloud-init nocloud user data source
// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
type CloudInitNoCloudSource struct {
	// UserDataSecretRef references a k8s secret that contains NoCloud userdata
	// + optional
	UserDataSecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
	// UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string
	// + optional
	UserDataBase64 string `json:"userDataBase64"`
}

// Only one of the fields in the CloudInitSpec can be set
type CloudInitSpec struct {
	// Nocloud DataSource
	NoCloudData *CloudInitNoCloudSource `json:"nocloud"`

	// Add future cloud init datasource structures below.
}

type DomainSpec struct {
	Resources ResourceRequirements `json:"resources,omitempty"`
	Firmware  *Firmware            `json:"firmware,omitempty"`
	Clock     *Clock               `json:"clock,omitempty"`
	Features  Features             `json:"features,omitempty"`
	Devices   Devices              `json:"devices"`
}

type ResourceRequirements struct {
	Initial v1.ResourceList
}

type Firmware struct {
	UID types.UID
}

type Devices struct {
	Disks      []Disk      `json:"disks,omitempty"`
	Interfaces []Interface `json:"interfaces,omitempty"`
	Channels   []Channel   `json:"channels,omitempty"`
	Video      []Video     `json:"video,omitempty"`
	Graphics   []Graphics  `json:"graphics,omitempty"`
	Ballooning *Ballooning `json:"memballoon,omitempty"`
	Serials    []Serial    `json:"serials,omitempty"`
	Consoles   []Console   `json:"consoles,omitempty"`
	Watchdog   *Watchdog   `json:"watchdog,omitempty"`
}

type Disk struct {
	// This must match the Name of a Volume.
	Name string `json:"name"`

	// DiskTarget specifies as which device the disk should be added to the guest
	DiskTarget `json:",inline"`
}

// Represents the target of a volume to mount.
// Only one of its members may be specified.
type DiskTargets struct {
	// Attach a volume as a disk to the vm
	Disk *DiskTarget `json:"disk,omitempty"`
	// Attach a volume as a LUN to the vm
	LUN *LunTarget `json:"lun,omitempty"`
	// Attach a volume as a floppy to the vm
	Floppy *FloppyTarget `json:"floppy,omitempty"`
	// Attach a volume as a cdrom to the vm
	CDRom *CDRomTarget `json:"cdrom,omitempty"`
}

type DiskTarget struct {
	DiskBaseTarget `json:",inline"`
}

type LunTarget struct {
	DiskBaseTarget `json:",inline"`
}

type FloppyTarget struct {
	DiskBaseTarget `json:",inline"`
}

type CDRomTarget struct {
	DiskBaseTarget `json:",inline"`
}

type DiskBaseTarget struct {
	// Device indicates the "logical" device name. The actual device name
	// specified is not guaranteed to map to the device name in the guest OS. Treat
	// it as a device ordering hint.
	Device string `json:"dev"`
}

// Volume represents a named volume in a vm.
type Volume struct {
	// Volume's name.
	// Must be a DNS_LABEL and unique within the vm.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
	// VolumeSource represents the location and type of the mounted volume.
	// Defaults to Disk, if no type is specified
	VolumeSource `json:",inline"`
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	// ISCSI represents an ISCSI Disk resource that is attached to a
	// kubelet's host machine and then exposed to the pod.
	// More info: https://releases.k8s.io/HEAD/examples/volumes/iscsi/README.md
	// +optional
	ISCSI *v1.ISCSIVolumeSource
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Made available to the vm as mounted block storage
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *v1.PersistentVolumeClaimVolumeSource
	// CloudInitNoCloud represents a cloud-init NoCloud user-data source.
	// The NoCloud data will be added as a disk to the vm. A proper cloud-init installation is required inside the guest.
	// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
	// +optional
	CloudInitNoCloud *CloudInitNoCloudSource
	// RegistryDisk references a docker image, embedding a qcow or raw disk
	// More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html
	// +optional
	RegistryDisk *RegistryDiskSource
}

// Represents a docker image with an embedded disk
type RegistryDiskSource struct {
	// Image is the name of the image with the embedded disk
	Image string `json:"image"`
}

type ReadOnly struct{}

type DiskSource struct {
	File          string          `json:"file,omitempty"`
	StartupPolicy string          `json:"startupPolicy,omitempty"`
	Protocol      string          `json:"protocol,omitempty"`
	Name          string          `json:"name,omitempty"`
	Host          *DiskSourceHost `json:"host,omitempty"`
}

type DiskTargetTmp struct {
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

// Exactly one of its members must be set.
type ClockOffset struct {
	// UTC sets the guest clock to UTC on each boot. If an offset is specified,
	// guest changes to the clock will be kept during reboots and not reset.
	UTC *ClockOffsetUTC `json:"utc,omitempty"`
	// Timezone sets the guest clock to the specified timezone.
	// Zone name follows the TZ environment variable format (e.g. 'America/New_York')
	Timezone *ClockOffsetTimezone `json:"timezone,omitempty"`
}

// UTC sets the guest clock to UTC on each boot.
type ClockOffsetUTC struct {
	// OffsetSeconds specifies an offset in seconds, relative to UTC. If set,
	// guest changes to the clock will be kept during reboots and not reset.
	OffsetSeconds *int `json:"offsetSeconds,omitempty"`
}

// ClockOffsetTimezone sets the guest clock to the specified timezone.
// Zone name follows the TZ environment variable format (e.g. 'America/New_York')
type ClockOffsetTimezone string

type Clock struct {
	ClockOffset `json:",inline"`
	Timer       `json:",inline"`
}

type Timer struct {
	// HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
	HPET *TimerAttrs `json:"hpet,omitempty"`
	// KVM 	(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests.
	KVM *TimerAttrs `json:"kvm,omitempty"`
	// PIT (Programmable Interval Timer) - a timer with periodic interrupts.
	PIT *TimerAttrs `json:"pit,omitempty"`
	// RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
	RTC *RTCTimerAttrs `json:"trc,omitempty"`
	// Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
	Hyperv *TimerAttrs `json:"hyperv,omitempty"`
}

// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
type TickPolicy string

const (
	// TickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick
	TickPolicyDelay TickPolicy = "delay"
	// TickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete
	TickPolicyCatchup TickPolicy = "catchup"
	// TickPolicyMerge merges the missed tick(s) into one tick and inject. The
	// guest time may be delayed, depending on how the OS reacts to the merging
	// of ticks
	TickPolicyMerge TickPolicy = "merge"
	// TickPolicyDiscard discards all missed ticks.
	TickPolicyDiscard TickPolicy = "discard"
)

type RTCTimerTrack string

const (
	// TrackGuest tracks the guest time
	TrackGuest RTCTimerTrack = "guest"
	// TrackWall tracks the host time
	TrackWall RTCTimerTrack = "wall"
)

type RTCTimerAttrs struct {
	TimerAttrs `json:",inline"`
	Track      RTCTimerTrack `json:"track,omitempty"`
}

type TimerAttrs struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup", "merge", "discard"
	TickPolicy TickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type Features struct {
	// Defaults to the machine type setting
	// +optional
	// TODO: Should it even be possible to disable ACPI?
	ACPI *FeatureState `json:"acpi,omitempty"`
	// Defaults to the machine type setting
	// +optional
	APIC *FeatureState `json:"apic,omitempty"`
	// Defaults to the machine type setting
	// +optional
	Hyperv *FeatureHyperv `json:"hyperv,omitempty"`
}

type FeatureState struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

type FeatureAPIC struct {
	FeatureState `json:",inline"`
	// EndOfInterrupt enables the end of interrupt notification in the guest
	// Defaults to false
	// +optional
	EndOfInterrupt bool `json:"endOfInterrupt,omitempty"`
}

type FeatureSpinlocks struct {
	FeatureState `json:",inline"`
	// Spinlocks indicates how many spinlocks are made available
	// Must be a value greater or equal 4096
	// Defaults to 4096
	// +optional
	Spinlocks *uint32
}

type FeatureVendorID struct {
	FeatureState `json:",inline"`
	VendorID     string
}

type FeatureHyperv struct {
	// Relaxed relaxes constraints on timer
	// Defaults to the machine type setting
	// +optional
	Relaxed *FeatureState
	// VAPIC indicates weather virtual APIC is enabled
	// Defaults to the machine type setting
	// +optional
	VAPIC *FeatureState
	// Spiinlocks
	// Spinlocks indicates if spinlocks should be made available to the guest
	// +optional
	Spinlocks *FeatureSpinlocks
	// VPIndex
	// Defaults to the machine type setting
	// +optional
	VPIndex *FeatureState
	// Runtime
	// Defaults to the machine type setting
	// +optional
	Runtime *FeatureState
	// SyNIC
	// Defaults to the machine type setting
	// +optional
	SyNIC *FeatureState
	// SyNICTimer
	// Defaults to the machine type setting
	// +optional
	SyNICTimer *FeatureState
	// Reset
	// Defaults to the machine type setting
	// +optional
	Reset *FeatureState
	// VendorID
	// Defaults to the machine type setting
	// +optional
	VendorID *FeatureVendorID
}

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
	domain := DomainSpec{}
	domain.Resources.Initial[v1.ResourceMemory] = resource.MustParse("8192Ki")
	domain.Devices = Devices{}
	domain.Devices.Interfaces = []Interface{
		{Type: "network", Source: InterfaceSource{Network: "default"}},
	}
	return &domain
}
