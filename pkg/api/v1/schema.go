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
	UserDataBase64 string `json:"userDataBase64,omitempty"`
	// UserData contains NoCloud inline cloud-init userdata
	// + optional
	UserData string `json:"userData,omitempty"`
}

type DomainSpec struct {
	// Resources describes the Compute Resources required by this vm.
	Resources ResourceRequirements `json:"resources,omitempty"`
	// CPU allow specified the detailed CPU topology inside the vm.
	// +optional
	CPU *CPU `json:"cpu,omitempty"`
	// Machine type
	// +optional
	Machine Machine `json:"machine,omitempty"`
	// Firmware
	// +optional
	Firmware *Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vm.
	// +optional
	Clock *Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv
	// +optional
	Features *Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, ...
	Devices Devices `json:"devices"`
}

type ResourceRequirements struct {
	// Requests is a description of the initial vm resources.
	// Valid resource keys are "memory" and "cpu".
	// +optional
	Requests v1.ResourceList `json:"requests,omitempty"`
}

// CPU allow specifying the CPU topology
type CPU struct {
	// Cores specifies the number of cores inside the vm.
	// Must be a value greater or equal 1.
	Cores uint32 `json:"cores,omitempty"`
}

type Machine struct {
	// QEMU machine type is the actual chipset of the VM.
	Type string `json:"type"`
}

type Firmware struct {
	// UUID reported by the vm bios
	// Defaults to a random generated uid
	UUID types.UID `json:"uuid,omitempty"`
}

type Devices struct {
	// Disks describes disks, cdroms, floppy and luns which are connected to the vm
	Disks []Disk `json:"disks,omitempty"`
	// Watchdog describes a watchdog device which can be added to the vm
	Watchdog *Watchdog `json:"watchdog,omitempty"`
}

type Disk struct {
	// Name is the device name
	Name string `json:"name"`
	// Name of the volume which is referenced
	// Must match the Name of a Volume.
	VolumeName string `json:"volumeName"`
	// DiskDevice specifies as which device the disk should be added to the guest
	// Defaults to Disk
	DiskDevice `json:",inline"`
}

// Represents the target of a volume to mount.
// Only one of its members may be specified.
type DiskDevice struct {
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
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, ide
	Bus string `json:"bus,omitempty"`
	// ReadOnly
	// Defaults to false
	ReadOnly bool `json:"readonly,omitempty"`
}

type LunTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, ide
	Bus string `json:"bus,omitempty"`
	// ReadOnly
	// Defaults to false
	ReadOnly bool `json:"readonly,omitempty"`
}

type FloppyTarget struct {
	// ReadOnly
	// Defaults to false
	ReadOnly bool `json:"readonly,omitempty"`
	// Tray indicates if the tray of the device is open or closed.
	// Allowed values are "open" and "closed"
	// Defaults to closed
	// +optional
	Tray TrayState `json:"tray,omitempty"`
}

// TrayState indicates if a tray of a cdrom or floppy is open or closed
type TrayState string

const (
	// TrayStateOpen indicates that the tray of a cdrom or floppy is open
	TrayStateOpen TrayState = "open"
	// TrayStateClosed indicates that the tray of a cdrom or floppy is closed
	TrayStateClosed TrayState = "closed"
)

type CDRomTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, ide
	Bus string `json:"bus,omitempty"`
	// ReadOnly
	// Defaults to true
	ReadOnly *bool `json:"readonly,omitempty"`
	// Tray indicates if the tray of the device is open or closed.
	// Allowed values are "open" and "closed"
	// Defaults to closed
	// +optional
	Tray TrayState `json:"tray,omitempty"`
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
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vm via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *v1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// CloudInitNoCloud represents a cloud-init NoCloud user-data source.
	// The NoCloud data will be added as a disk to the vm. A proper cloud-init installation is required inside the guest.
	// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
	// +optional
	CloudInitNoCloud *CloudInitNoCloudSource `json:"cloudInitNoCloud,omitempty"`
	// RegistryDisk references a docker image, embedding a qcow or raw disk
	// More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html
	// +optional
	RegistryDisk *RegistryDiskSource `json:"registryDisk,omitempty"`
}

// Represents a docker image with an embedded disk
type RegistryDiskSource struct {
	// Image is the name of the image with the embedded disk
	Image string `json:"image"`
}

// Exactly one of its members must be set.
type ClockOffset struct {
	// UTC sets the guest clock to UTC on each boot. If an offset is specified,
	// guest changes to the clock will be kept during reboots and are not reset.
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

// Represents the clock and timers of a vm
type Clock struct {
	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock
	ClockOffset `json:",inline"`
	// Timer specifies whih timers are attached to the vm
	Timer *Timer `json:"timer,inline"`
}

// Represents all available timers in a vm
type Timer struct {
	// HPET (High Precision Event Timer) - multiple timers with periodic interrupts.
	HPET *HPETTimer `json:"hpet,omitempty"`
	// KVM 	(KVM clock) - lets guests read the host’s wall clock time (paravirtualized). For linux guests.
	KVM *KVMTimer `json:"kvm,omitempty"`
	// PIT (Programmable Interval Timer) - a timer with periodic interrupts.
	PIT *PITTimer `json:"pit,omitempty"`
	// RTC (Real Time Clock) - a continuously running timer with periodic interrupts.
	RTC *RTCTimer `json:"rtc,omitempty"`
	// Hyperv (Hypervclock) - lets guests read the host’s wall clock time (paravirtualized). For windows guests.
	Hyperv *HypervTimer `json:"hyperv,omitempty"`
}

// HPETTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
type HPETTickPolicy string

// PITTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
type PITTickPolicy string

// RTCTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
type RTCTickPolicy string

const (
	// HPETTickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick
	HPETTickPolicyDelay HPETTickPolicy = "delay"
	// HPETTickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete
	HPETTickPolicyCatchup HPETTickPolicy = "catchup"
	// HPETTickPolicyMerge merges the missed tick(s) into one tick and inject. The
	// guest time may be delayed, depending on how the OS reacts to the merging
	// of ticks
	HPETTickPolicyMerge HPETTickPolicy = "merge"
	// HPETTickPolicyDiscard discards all missed ticks.
	HPETTickPolicyDiscard HPETTickPolicy = "discard"

	// PITTickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick
	PITTickPolicyDelay PITTickPolicy = "delay"
	// PITTickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete
	PITTickPolicyCatchup PITTickPolicy = "catchup"
	// PITTickPolicyDiscard discards all missed ticks.
	PITTickPolicyDiscard PITTickPolicy = "discard"

	// RTCTickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick
	RTCTickPolicyDelay RTCTickPolicy = "delay"
	// RTCTickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete
	RTCTickPolicyCatchup RTCTickPolicy = "catchup"
)

// RTCTimerTrack specifies from which source to track the time
type RTCTimerTrack string

const (
	// TrackGuest tracks the guest time
	TrackGuest RTCTimerTrack = "guest"
	// TrackWall tracks the host time
	TrackWall RTCTimerTrack = "wall"
)

type RTCTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup"
	TickPolicy RTCTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
	// Track the guest or the wall clock
	Track RTCTimerTrack `json:"track,omitempty"`
}

type HPETTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup", "merge", "discard"
	TickPolicy HPETTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type PITTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup", "discard"
	TickPolicy PITTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type KVMTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type HypervTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type Features struct {
	// ACPI enables/disables ACPI insidejsondata guest
	// Defaults to enabled
	// +optional
	ACPI FeatureState `json:"acpi,omitempty"`
	// Defaults to the machine type setting
	// +optional
	APIC *FeatureAPIC `json:"apic,omitempty"`
	// Defaults to the machine type setting
	// +optional
	Hyperv *FeatureHyperv `json:"hyperv,omitempty"`
}

// Represents if a feature is enabled or disabled
type FeatureState struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

type FeatureAPIC struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// EndOfInterrupt enables the end of interrupt notification in the guest
	// Defaults to false
	// +optional
	EndOfInterrupt bool `json:"endOfInterrupt,omitempty"`
}

type FeatureSpinlocks struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Retries indicates the number of retries
	// Must be a value greater or equal 4096
	// Defaults to 4096
	// +optional
	Retries *uint32 `json:"spinlocks,omitempty"`
}

type FeatureVendorID struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// VendorID sets the hypervisor vendor id, visible to the vm
	// String up to twelve characters
	VendorID string `json:"vendorid, omitempty"`
}

// Hyperv specific features
type FeatureHyperv struct {
	// Relaxed relaxes constraints on timer
	// Defaults to the machine type setting
	// +optional
	Relaxed *FeatureState `json:"relaxed,omitempty"`
	// VAPIC indicates whether virtual APIC is enabled
	// Defaults to the machine type setting
	// +optional
	VAPIC *FeatureState `json:"vapic,omitempty"`
	// Spinlocks indicates if spinlocks should be made available to the guest
	// +optional
	Spinlocks *FeatureSpinlocks `json:"spinlocks,omitempty"`
	// VPIndex enables the Virtual Processor Index to help windows identifying virtual processors
	// Defaults to the machine type setting
	// +optional
	VPIndex *FeatureState `json:"vpindex,omitempty"`
	// Runtime
	// Defaults to the machine type setting
	// +optional
	Runtime *FeatureState `json:"runtime,omitempty"`
	// SyNIC enable Synthetic Interrupt Controller
	// Defaults to the machine type setting
	// +optional
	SyNIC *FeatureState `json:"synic,omitempty"`
	// SyNICTimer enable Synthetic Interrupt Controller timer
	// Defaults to the machine type setting
	// +optional
	SyNICTimer *FeatureState `json:"synictimer,omitempty"`
	// Reset enables Hyperv reboot/reset for the vm
	// Defaults to the machine type setting
	// +optional
	Reset *FeatureState `json:"reset,omitempty"`
	// VendorID allows setting the hypervisor vendor id
	// Defaults to the machine type setting
	// +optional
	VendorID *FeatureVendorID `json:"vendorid,omitempty"`
}

// WatchdogAction defines the watchdog action, if a watchdog gets triggered
type WatchdogAction string

const (
	// WatchdogActionPoweroff will poweroff the vm if the watchdog gets triggered
	WatchdogActionPoweroff WatchdogAction = "poweroff"
	// WatchdogActionReset will reset the vm if the watchdog gets triggered
	WatchdogActionReset WatchdogAction = "reset"
	// WatchdogActionShutdown will shutdown the vm if the watchdog gets triggered
	WatchdogActionShutdown WatchdogAction = "shutdown"
)

// Named watchdog device
type Watchdog struct {
	// Name of the watchdog
	Name string `json:"name"`
	// WatchdogDevice contains the watchdog type and actions
	// Defaults to i6300esb
	WatchdogDevice `json:",inline"`
}

// Hardware watchdog device
// Exactly one of its members must be set.
type WatchdogDevice struct {
	// i6300esb watchdog device
	// +optional
	I6300ESB *I6300ESBWatchdog `json:"i6300esb,omitempty"`
}

// i6300esb watchdog device
type I6300ESBWatchdog struct {
	// The action to take. Valid values are poweroff, reset, shutdown.
	// Defaults to reset
	Action WatchdogAction `json:"action,omitempty"`
}

// TODO ballooning, rng, cpu ...

func NewMinimalDomainSpec() DomainSpec {
	domain := DomainSpec{}
	domain.Resources.Requests = v1.ResourceList{
		v1.ResourceMemory: resource.MustParse("8192Ki"),
	}
	return domain
}
