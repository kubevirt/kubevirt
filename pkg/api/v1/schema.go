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
//go:generate openapi-gen -i . --output-package=kubevirt.io/kubevirt/pkg/api/v1  --go-header-file ../../../hack/boilerplate/boilerplate.go.txt

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

// Represents a cloud-init nocloud user data source
// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type DomainSpec struct {
	// Resources describes the Compute Resources required by this vmi.
	Resources ResourceRequirements `json:"resources,omitempty"`
	// CPU allow specified the detailed CPU topology inside the vmi.
	// +optional
	CPU *CPU `json:"cpu,omitempty"`
	// Memory allow specifying the VMI memory features.
	// +optional
	Memory *Memory `json:"memory,omitempty"`
	// Machine type
	// +optional
	Machine Machine `json:"machine,omitempty"`
	// Firmware
	// +optional
	Firmware *Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv
	// +optional
	Features *Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, ...
	Devices Devices `json:"devices"`
}

// ---
// +k8s:openapi-gen=true
type DomainPresetSpec struct {
	// Resources describes the Compute Resources required by this vmi.
	Resources ResourceRequirements `json:"resources,omitempty"`
	// CPU allow specified the detailed CPU topology inside the vmi.
	// +optional
	CPU *CPU `json:"cpu,omitempty"`
	// Machine type
	// +optional
	Machine Machine `json:"machine,omitempty"`
	// Firmware
	// +optional
	Firmware *Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv
	// +optional
	Features *Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, ...
	// +optional
	Devices Devices `json:"devices,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type ResourceRequirements struct {
	// Requests is a description of the initial vmi resources.
	// Valid resource keys are "memory" and "cpu".
	// +optional
	Requests v1.ResourceList `json:"requests,omitempty"`
	// Limits describes the maximum amount of compute resources allowed.
	// Valid resource keys are "memory" and "cpu".
	// +optional
	Limits v1.ResourceList `json:"limits,omitempty"`
}

// CPU allow specifying the CPU topology
// ---
// +k8s:openapi-gen=true
type CPU struct {
	// Cores specifies the number of cores inside the vmi.
	// Must be a value greater or equal 1.
	Cores uint32 `json:"cores,omitempty"`
}

// Memory allow specifying the VirtualMachineInstance memory features
// ---
// +k8s:openapi-gen=true
type Memory struct {
	// Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
	// +optional
	Hugepages *Hugepages `json:"hugepages,omitempty"`
}

// Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
// ---
// +k8s:openapi-gen=true
type Hugepages struct {
	// PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
	PageSize string `json:"pageSize,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Machine struct {
	// QEMU machine type is the actual chipset of the VirtualMachineInstance.
	Type string `json:"type"`
}

// ---
// +k8s:openapi-gen=true
type Firmware struct {
	// UUID reported by the vmi bios
	// Defaults to a random generated uid
	UUID types.UID `json:"uuid,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Devices struct {
	// Disks describes disks, cdroms, floppy and luns which are connected to the vmi
	Disks []Disk `json:"disks,omitempty"`
	// Watchdog describes a watchdog device which can be added to the vmi
	Watchdog *Watchdog `json:"watchdog,omitempty"`
	// Interfaces describe network interfaces which are added to the vm
	Interfaces []Interface `json:"interfaces,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Disk struct {
	// Name is the device name
	Name string `json:"name"`
	// Name of the volume which is referenced
	// Must match the Name of a Volume.
	VolumeName string `json:"volumeName"`
	// DiskDevice specifies as which device the disk should be added to the guest
	// Defaults to Disk
	DiskDevice `json:",inline"`
	// BootOrder is an integer value > 0, used to determine ordering of boot devices.
	// Lower values take precedence.
	// Disks without a boot order are not tried if a disk with a boot order exists.
	// +optional
	BootOrder *uint `json:"bootOrder,omitempty"`
}

// Represents the target of a volume to mount.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type DiskDevice struct {
	// Attach a volume as a disk to the vmi
	Disk *DiskTarget `json:"disk,omitempty"`
	// Attach a volume as a LUN to the vmi
	LUN *LunTarget `json:"lun,omitempty"`
	// Attach a volume as a floppy to the vmi
	Floppy *FloppyTarget `json:"floppy,omitempty"`
	// Attach a volume as a cdrom to the vmi
	CDRom *CDRomTarget `json:"cdrom,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type DiskTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, ide
	Bus string `json:"bus,omitempty"`
	// ReadOnly
	// Defaults to false
	ReadOnly bool `json:"readonly,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type LunTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, ide
	Bus string `json:"bus,omitempty"`
	// ReadOnly
	// Defaults to false
	ReadOnly bool `json:"readonly,omitempty"`
}

// ---
// +k8s:openapi-gen=true
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
// ---
// +k8s:openapi-gen=true
type TrayState string

const (
	// TrayStateOpen indicates that the tray of a cdrom or floppy is open
	TrayStateOpen TrayState = "open"
	// TrayStateClosed indicates that the tray of a cdrom or floppy is closed
	TrayStateClosed TrayState = "closed"
)

// ---
// +k8s:openapi-gen=true
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

// Volume represents a named volume in a vmi.
// ---
// +k8s:openapi-gen=true
type Volume struct {
	// Volume's name.
	// Must be a DNS_LABEL and unique within the vmi.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
	// VolumeSource represents the location and type of the mounted volume.
	// Defaults to Disk, if no type is specified
	VolumeSource `json:",inline"`
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type VolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *v1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// CloudInitNoCloud represents a cloud-init NoCloud user-data source.
	// The NoCloud data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest.
	// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
	// +optional
	CloudInitNoCloud *CloudInitNoCloudSource `json:"cloudInitNoCloud,omitempty"`
	// RegistryDisk references a docker image, embedding a qcow or raw disk
	// More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html
	// +optional
	RegistryDisk *RegistryDiskSource `json:"registryDisk,omitempty"`
	// Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
	// +optional
	Ephemeral *EphemeralVolumeSource `json:"ephemeral,omitempty"`
	// EmptyDisk represents a temporary disk which shares the vmis lifecycle
	// More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html
	// +optional
	EmptyDisk *EmptyDiskSource `json:"emptyDisk,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type EphemeralVolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *v1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
}

// EmptyDisk represents a temporary disk which shares the vmis lifecycle
// ---
// +k8s:openapi-gen=true
type EmptyDiskSource struct {
	// Capacity of the sparse disk.
	Capacity resource.Quantity `json:"capacity"`
}

// Represents a docker image with an embedded disk
// ---
// +k8s:openapi-gen=true
type RegistryDiskSource struct {
	// Image is the name of the image with the embedded disk
	Image string `json:"image"`
	// ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// Exactly one of its members must be set.
// ---
// +k8s:openapi-gen=true
type ClockOffset struct {
	// UTC sets the guest clock to UTC on each boot. If an offset is specified,
	// guest changes to the clock will be kept during reboots and are not reset.
	UTC *ClockOffsetUTC `json:"utc,omitempty"`
	// Timezone sets the guest clock to the specified timezone.
	// Zone name follows the TZ environment variable format (e.g. 'America/New_York')
	Timezone *ClockOffsetTimezone `json:"timezone,omitempty"`
}

// UTC sets the guest clock to UTC on each boot.
// ---
// +k8s:openapi-gen=true
type ClockOffsetUTC struct {
	// OffsetSeconds specifies an offset in seconds, relative to UTC. If set,
	// guest changes to the clock will be kept during reboots and not reset.
	OffsetSeconds *int `json:"offsetSeconds,omitempty"`
}

// ClockOffsetTimezone sets the guest clock to the specified timezone.
// Zone name follows the TZ environment variable format (e.g. 'America/New_York')
// ---
// +k8s:openapi-gen=true
type ClockOffsetTimezone string

// Represents the clock and timers of a vmi
// ---
// +k8s:openapi-gen=true
type Clock struct {
	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock
	ClockOffset `json:",inline"`
	// Timer specifies whih timers are attached to the vmi
	Timer *Timer `json:"timer,inline"`
}

// Represents all available timers in a vmi
// ---
// +k8s:openapi-gen=true
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
// ---
// +k8s:openapi-gen=true
type HPETTickPolicy string

// PITTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
// ---
// +k8s:openapi-gen=true
type PITTickPolicy string

// RTCTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
// ---
// +k8s:openapi-gen=true
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
// ---
// +k8s:openapi-gen=true
type RTCTimerTrack string

const (
	// TrackGuest tracks the guest time
	TrackGuest RTCTimerTrack = "guest"
	// TrackWall tracks the host time
	TrackWall RTCTimerTrack = "wall"
)

// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type HPETTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup", "merge", "discard"
	TickPolicy HPETTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type PITTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest
	// One of "delay", "catchup", "discard"
	TickPolicy PITTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type KVMTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type HypervTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
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
// ---
// +k8s:openapi-gen=true
type FeatureState struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type FeatureVendorID struct {
	// Enabled determines if the feature should be enabled or disabled on the guest
	// Defaults to true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// VendorID sets the hypervisor vendor id, visible to the vmi
	// String up to twelve characters
	VendorID string `json:"vendorid,omitempty"`
}

// Hyperv specific features
// ---
// +k8s:openapi-gen=true
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
	// Reset enables Hyperv reboot/reset for the vmi
	// Defaults to the machine type setting
	// +optional
	Reset *FeatureState `json:"reset,omitempty"`
	// VendorID allows setting the hypervisor vendor id
	// Defaults to the machine type setting
	// +optional
	VendorID *FeatureVendorID `json:"vendorid,omitempty"`
}

// WatchdogAction defines the watchdog action, if a watchdog gets triggered
// ---
// +k8s:openapi-gen=true
type WatchdogAction string

const (
	// WatchdogActionPoweroff will poweroff the vmi if the watchdog gets triggered
	WatchdogActionPoweroff WatchdogAction = "poweroff"
	// WatchdogActionReset will reset the vmi if the watchdog gets triggered
	WatchdogActionReset WatchdogAction = "reset"
	// WatchdogActionShutdown will shutdown the vmi if the watchdog gets triggered
	WatchdogActionShutdown WatchdogAction = "shutdown"
)

// Named watchdog device
// ---
// +k8s:openapi-gen=true
type Watchdog struct {
	// Name of the watchdog
	Name string `json:"name"`
	// WatchdogDevice contains the watchdog type and actions
	// Defaults to i6300esb
	WatchdogDevice `json:",inline"`
}

// Hardware watchdog device
// Exactly one of its members must be set.
// ---
// +k8s:openapi-gen=true
type WatchdogDevice struct {
	// i6300esb watchdog device
	// +optional
	I6300ESB *I6300ESBWatchdog `json:"i6300esb,omitempty"`
}

// i6300esb watchdog device
// ---
// +k8s:openapi-gen=true
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

// ---
// +k8s:openapi-gen=true
type Interface struct {
	// Logical name of the interface as well as a reference to the associated networks
	// Must match the Name of a Network
	Name string `json:"name"`
	// BindingMethod specifies the method which will be used to connect the interface to the guest
	// Defaults to Bridge
	InterfaceBindingMethod `json:",inline"`
}

// Represents the method which will be used to connect the interface to the guest.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type InterfaceBindingMethod struct {
	Bridge *InterfaceBridge `json:"bridge,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type InterfaceBridge struct{}

// Network represents a network type and a resource that should be connected to the vm.
// ---
// +k8s:openapi-gen=true
type Network struct {
	// Network name
	// Must be a DNS_LABEL and unique within the vm.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
	// NetworkSource represents the network type and the source interface that should be connected to the virtual machine.
	// Defaults to Pod, if no type is specified
	NetworkSource `json:",inline"`
}

// Represents the source resource that will be connected to the vm.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type NetworkSource struct {
	Pod *PodNetwork `json:"pod,omitempty"`
}

// Represents the stock pod network interface
// ---
// +k8s:openapi-gen=true
type PodNetwork struct{}
