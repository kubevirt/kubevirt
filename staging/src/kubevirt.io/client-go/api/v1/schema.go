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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package v1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

type IOThreadsPolicy string

const (
	IOThreadsPolicyShared  IOThreadsPolicy = "shared"
	IOThreadsPolicyAuto    IOThreadsPolicy = "auto"
	CPUModeHostPassthrough                 = "host-passthrough"
	CPUModeHostModel                       = "host-model"
)

//go:generate swagger-doc
//go:generate openapi-gen -i . --output-package=kubevirt.io/kubevirt/staging/src/kubevirt.io/client-go/api/v1  --go-header-file ../../../../../../hack/boilerplate/boilerplate.go.txt

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

// Represents a disk created on the cluster level
// ---
// +k8s:openapi-gen=true
type HostDisk struct {
	// The path to HostDisk image located on the cluster
	Path string `json:"path"`
	// Contains information if disk.img exists or should be created
	// allowed options are 'Disk' and 'DiskOrCreate'
	Type HostDiskType `json:"type"`
	// Capacity of the sparse disk
	// +optional
	Capacity resource.Quantity `json:"capacity,omitempty"`
	// Shared indicate whether the path is shared between nodes
	Shared *bool `json:"shared,omitempty"`
}

// ConfigMapVolumeSource adapts a ConfigMap into a volume.
// More info: https://kubernetes.io/docs/concepts/storage/volumes/#configmap
// ---
// +k8s:openapi-gen=true
type ConfigMapVolumeSource struct {
	v1.LocalObjectReference `json:",inline"`
	// Specify whether the ConfigMap or it's keys must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// SecretVolumeSource adapts a Secret into a volume.
// ---
// +k8s:openapi-gen=true
type SecretVolumeSource struct {
	// Name of the secret in the pod's namespace to use.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#secret
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// Specify whether the Secret or it's keys must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// ServiceAccountVolumeSource adapts a ServiceAccount into a volume.
// ---
// +k8s:openapi-gen=true
type ServiceAccountVolumeSource struct {
	// Name of the service account in the pod's namespace to use.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// Represents a cloud-init nocloud user data source.
// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
// ---
// +k8s:openapi-gen=true
type CloudInitNoCloudSource struct {
	// UserDataSecretRef references a k8s secret that contains NoCloud userdata.
	// + optional
	UserDataSecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
	// UserDataBase64 contains NoCloud cloud-init userdata as a base64 encoded string.
	// + optional
	UserDataBase64 string `json:"userDataBase64,omitempty"`
	// UserData contains NoCloud inline cloud-init userdata.
	// + optional
	UserData string `json:"userData,omitempty"`
	// NetworkDataSecretRef references a k8s secret that contains NoCloud networkdata.
	// + optional
	NetworkDataSecretRef *v1.LocalObjectReference `json:"networkDataSecretRef,omitempty"`
	// NetworkDataBase64 contains NoCloud cloud-init networkdata as a base64 encoded string.
	// + optional
	NetworkDataBase64 string `json:"networkDataBase64,omitempty"`
	// NetworkData contains NoCloud inline cloud-init networkdata.
	// + optional
	NetworkData string `json:"networkData,omitempty"`
}

// Represents a cloud-init config drive user data source.
// More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html
// ---
// +k8s:openapi-gen=true
type CloudInitConfigDriveSource struct {
	// UserDataSecretRef references a k8s secret that contains config drive userdata.
	// + optional
	UserDataSecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`
	// UserDataBase64 contains config drive cloud-init userdata as a base64 encoded string.
	// + optional
	UserDataBase64 string `json:"userDataBase64,omitempty"`
	// UserData contains config drive inline cloud-init userdata.
	// + optional
	UserData string `json:"userData,omitempty"`
	// NetworkDataSecretRef references a k8s secret that contains config drive networkdata.
	// + optional
	NetworkDataSecretRef *v1.LocalObjectReference `json:"networkDataSecretRef,omitempty"`
	// NetworkDataBase64 contains config drive cloud-init networkdata as a base64 encoded string.
	// + optional
	NetworkDataBase64 string `json:"networkDataBase64,omitempty"`
	// NetworkData contains config drive inline cloud-init networkdata.
	// + optional
	NetworkData string `json:"networkData,omitempty"`
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
	// Machine type.
	// +optional
	Machine Machine `json:"machine,omitempty"`
	// Firmware.
	// +optional
	Firmware *Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv, smm.
	// +optional
	Features *Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, ...
	Devices Devices `json:"devices"`
	// Controls whether or not disks will share IOThreads.
	// Omitting IOThreadsPolicy disables use of IOThreads.
	// One of: shared, auto
	// +optional
	IOThreadsPolicy *IOThreadsPolicy `json:"ioThreadsPolicy,omitempty"`
	// Chassis specifies the chassis info passed to the domain.
	// +optional
	Chassis *Chassis `json:"chassis,omitempty"`
}

// Chassis specifies the chassis info passed to the domain.
// ---
// +k8s:openapi-gen=true
type Chassis struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Version      string `json:"version,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Asset        string `json:"asset,omitempty"`
	Sku          string `json:"sku,omitempty"`
}

// Represents the firmware blob used to assist in the domain creation process.
// Used for setting the QEMU BIOS file path for the libvirt domain.
// ---
// +k8s:openapi-gen=true
type Bootloader struct {
	// If set (default), BIOS will be used.
	// +optional
	BIOS *BIOS `json:"bios,omitempty"`
	// If set, EFI will be used instead of BIOS.
	// +optional
	EFI *EFI `json:"efi,omitempty"`
}

// If set (default), BIOS will be used.
// ---
// +k8s:openapi-gen=true
type BIOS struct {
}

// If set, EFI will be used instead of BIOS.
// ---
// +k8s:openapi-gen=true
type EFI struct {
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
	// Don't ask the scheduler to take the guest-management overhead into account. Instead
	// put the overhead only into the container's memory limit. This can lead to crashes if
	// all memory is in use on a node. Defaults to false.
	OvercommitGuestOverhead bool `json:"overcommitGuestOverhead,omitempty"`
}

// CPU allows specifying the CPU topology.
// ---
// +k8s:openapi-gen=true
type CPU struct {
	// Cores specifies the number of cores inside the vmi.
	// Must be a value greater or equal 1.
	Cores uint32 `json:"cores,omitempty"`
	// Sockets specifies the number of sockets inside the vmi.
	// Must be a value greater or equal 1.
	Sockets uint32 `json:"sockets,omitempty"`
	// Threads specifies the number of threads inside the vmi.
	// Must be a value greater or equal 1.
	Threads uint32 `json:"threads,omitempty"`
	// Model specifies the CPU model inside the VMI.
	// List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map.
	// It is possible to specify special cases like "host-passthrough" to get the same CPU as the node
	// and "host-model" to get CPU closest to the node one.
	// Defaults to host-model.
	// +optional
	Model string `json:"model,omitempty"`
	// Features specifies the CPU features list inside the VMI.
	// +optional
	Features []CPUFeature `json:"features,omitempty"`
	// DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node
	// with enough dedicated pCPUs and pin the vCPUs to it.
	// +optional
	DedicatedCPUPlacement bool `json:"dedicatedCpuPlacement,omitempty"`
}

// CPUFeature allows specifying a CPU feature.
// ---
// +k8s:openapi-gen=true
type CPUFeature struct {
	// Name of the CPU feature
	Name string `json:"name"`
	// Policy is the CPU feature attribute which can have the following attributes:
	// force    - The virtual CPU will claim the feature is supported regardless of it being supported by host CPU.
	// require  - Guest creation will fail unless the feature is supported by the host CPU or the hypervisor is able to emulate it.
	// optional - The feature will be supported by virtual CPU if and only if it is supported by host CPU.
	// disable  - The feature will not be supported by virtual CPU.
	// forbid   - Guest creation will fail if the feature is supported by host CPU.
	// Defaults to require
	// +optional
	Policy string `json:"policy,omitempty"`
}

// Memory allows specifying the VirtualMachineInstance memory features.
// ---
// +k8s:openapi-gen=true
type Memory struct {
	// Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
	// +optional
	Hugepages *Hugepages `json:"hugepages,omitempty"`
	// Guest allows to specifying the amount of memory which is visible inside the Guest OS.
	// The Guest must lie between Requests and Limits from the resources section.
	// Defaults to the requested memory in the resources section if not specified.
	// + optional
	Guest *resource.Quantity `json:"guest,omitempty"`
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
	// UUID reported by the vmi bios.
	// Defaults to a random generated uid.
	UUID types.UID `json:"uuid,omitempty"`
	// Settings to control the bootloader that is used.
	// +optional
	Bootloader *Bootloader `json:"bootloader,omitempty"`
	// The system-serial-number in SMBIOS
	Serial string `json:"serial,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Devices struct {
	// Disks describes disks, cdroms, floppy and luns which are connected to the vmi.
	Disks []Disk `json:"disks,omitempty"`
	// Watchdog describes a watchdog device which can be added to the vmi.
	Watchdog *Watchdog `json:"watchdog,omitempty"`
	// Interfaces describe network interfaces which are added to the vmi.
	Interfaces []Interface `json:"interfaces,omitempty"`
	// Inputs describe input devices
	Inputs []Input `json:"inputs,omitempty"`
	// Whether to attach a pod network interface. Defaults to true.
	AutoattachPodInterface *bool `json:"autoattachPodInterface,omitempty"`
	// Whether to attach the default graphics device or not.
	// VNC will not be available if set to false. Defaults to true.
	AutoattachGraphicsDevice *bool `json:"autoattachGraphicsDevice,omitempty"`
	// Whether to have random number generator from host
	// +optional
	Rng *Rng `json:"rng,omitempty"`
	// Whether or not to enable virtio multi-queue for block devices
	// +optional
	BlockMultiQueue *bool `json:"blockMultiQueue,omitempty"`
	// If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature
	// +optional
	NetworkInterfaceMultiQueue *bool `json:"networkInterfaceMultiqueue,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Input struct {
	// Bus indicates the bus of input device to emulate.
	// Supported values: virtio, usb.
	Bus string `json:"bus,omitempty"`
	// Type indicated the type of input device.
	// Supported values: tablet.
	Type string `json:"type"`
	// Name is the device name
	Name string `json:"name"`
}

// ---
// +k8s:openapi-gen=true
type Disk struct {
	// Name is the device name
	Name string `json:"name"`
	// DiskDevice specifies as which device the disk should be added to the guest.
	// Defaults to Disk.
	DiskDevice `json:",inline"`
	// BootOrder is an integer value > 0, used to determine ordering of boot devices.
	// Lower values take precedence.
	// Each disk or interface that has a boot order must have a unique value.
	// Disks without a boot order are not tried if a disk with a boot order exists.
	// +optional
	BootOrder *uint `json:"bootOrder,omitempty"`
	// Serial provides the ability to specify a serial number for the disk device.
	// +optional
	Serial string `json:"serial,omitempty"`
	// dedicatedIOThread indicates this disk should have an exclusive IO Thread.
	// Enabling this implies useIOThreads = true.
	// Defaults to false.
	// +optional
	DedicatedIOThread *bool `json:"dedicatedIOThread,omitempty"`
	// Cache specifies which kvm disk cache mode should be used.
	// +optional
	Cache DriverCache `json:"cache,omitempty"`
}

// Represents the target of a volume to mount.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type DiskDevice struct {
	// Attach a volume as a disk to the vmi.
	Disk *DiskTarget `json:"disk,omitempty"`
	// Attach a volume as a LUN to the vmi.
	LUN *LunTarget `json:"lun,omitempty"`
	// Attach a volume as a floppy to the vmi.
	Floppy *FloppyTarget `json:"floppy,omitempty"`
	// Attach a volume as a cdrom to the vmi.
	CDRom *CDRomTarget `json:"cdrom,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type DiskTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus string `json:"bus,omitempty"`
	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`
	// If specified, the virtual disk will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10
	// +optional
	PciAddress string `json:"pciAddress,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type LunTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus string `json:"bus,omitempty"`
	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type FloppyTarget struct {
	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`
	// Tray indicates if the tray of the device is open or closed.
	// Allowed values are "open" and "closed".
	// Defaults to closed.
	// +optional
	Tray TrayState `json:"tray,omitempty"`
}

// TrayState indicates if a tray of a cdrom or floppy is open or closed.
// ---
// +k8s:openapi-gen=true
type TrayState string

const (
	// TrayStateOpen indicates that the tray of a cdrom or floppy is open.
	TrayStateOpen TrayState = "open"
	// TrayStateClosed indicates that the tray of a cdrom or floppy is closed.
	TrayStateClosed TrayState = "closed"
)

// ---
// +k8s:openapi-gen=true
type CDRomTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus string `json:"bus,omitempty"`
	// ReadOnly.
	// Defaults to true.
	ReadOnly *bool `json:"readonly,omitempty"`
	// Tray indicates if the tray of the device is open or closed.
	// Allowed values are "open" and "closed".
	// Defaults to closed.
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
	// Defaults to Disk, if no type is specified.
	VolumeSource `json:",inline"`
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type VolumeSource struct {
	// HostDisk represents a disk created on the cluster level
	// +optional
	HostDisk *HostDisk `json:"hostDisk,omitempty"`
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
	// CloudInitConfigDrive represents a cloud-init Config Drive user-data source.
	// The Config Drive data will be added as a disk to the vmi. A proper cloud-init installation is required inside the guest.
	// More info: https://cloudinit.readthedocs.io/en/latest/topics/datasources/configdrive.html
	// +optional
	CloudInitConfigDrive *CloudInitConfigDriveSource `json:"cloudInitConfigDrive,omitempty"`
	// ContainerDisk references a docker image, embedding a qcow or raw disk.
	// More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html
	// +optional
	ContainerDisk *ContainerDiskSource `json:"containerDisk,omitempty"`
	// Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
	// +optional
	Ephemeral *EphemeralVolumeSource `json:"ephemeral,omitempty"`
	// EmptyDisk represents a temporary disk which shares the vmis lifecycle.
	// More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html
	// +optional
	EmptyDisk *EmptyDiskSource `json:"emptyDisk,omitempty"`
	// DataVolume represents the dynamic creation a PVC for this volume as well as
	// the process of populating that PVC with a disk image.
	// +optional
	DataVolume *DataVolumeSource `json:"dataVolume,omitempty"`
	// ConfigMapSource represents a reference to a ConfigMap in the same namespace.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	// +optional
	ConfigMap *ConfigMapVolumeSource `json:"configMap,omitempty"`
	// SecretVolumeSource represents a reference to a secret data in the same namespace.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret/
	// +optional
	Secret *SecretVolumeSource `json:"secret,omitempty"`
	// ServiceAccountVolumeSource represents a reference to a service account.
	// There can only be one volume of this type!
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccount *ServiceAccountVolumeSource `json:"serviceAccount,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type DataVolumeSource struct {
	// Name represents the name of the DataVolume in the same namespace
	Name string `json:"name"`
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

// EmptyDisk represents a temporary disk which shares the vmis lifecycle.
// ---
// +k8s:openapi-gen=true
type EmptyDiskSource struct {
	// Capacity of the sparse disk.
	Capacity resource.Quantity `json:"capacity"`
}

// Represents a docker image with an embedded disk.
// ---
// +k8s:openapi-gen=true
type ContainerDiskSource struct {
	// Image is the name of the image with the embedded disk.
	Image string `json:"image"`
	// ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
	// Path defines the path to disk file in the container
	Path string `json:"path,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// Exactly one of its members must be set.
// ---
// +k8s:openapi-gen=true
type ClockOffset struct {
	// UTC sets the guest clock to UTC on each boot. If an offset is specified,
	// guest changes to the clock will be kept during reboots and are not reset.
	UTC *ClockOffsetUTC `json:"utc,omitempty"`
	// Timezone sets the guest clock to the specified timezone.
	// Zone name follows the TZ environment variable format (e.g. 'America/New_York').
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
// Zone name follows the TZ environment variable format (e.g. 'America/New_York').
// ---
// +k8s:openapi-gen=true
type ClockOffsetTimezone string

// Represents the clock and timers of a vmi.
// ---
// +k8s:openapi-gen=true
type Clock struct {
	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock.
	ClockOffset `json:",inline"`
	// Timer specifies whih timers are attached to the vmi.
	Timer *Timer `json:"timer,inline"`
}

// Represents all available timers in a vmi.
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

// HPETTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
// ---
// +k8s:openapi-gen=true
type HPETTickPolicy string

// PITTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
// ---
// +k8s:openapi-gen=true
type PITTickPolicy string

// RTCTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
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
	// of ticks.
	HPETTickPolicyMerge HPETTickPolicy = "merge"
	// HPETTickPolicyDiscard discards all missed ticks.
	HPETTickPolicyDiscard HPETTickPolicy = "discard"

	// PITTickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick.
	PITTickPolicyDelay PITTickPolicy = "delay"
	// PITTickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete.
	PITTickPolicyCatchup PITTickPolicy = "catchup"
	// PITTickPolicyDiscard discards all missed ticks.
	PITTickPolicyDiscard PITTickPolicy = "discard"

	// RTCTickPolicyDelay delivers ticks at a constant rate. The guest time will
	// be delayed due to the late tick.
	RTCTickPolicyDelay RTCTickPolicy = "delay"
	// RTCTickPolicyCatchup Delivers ticks at a higher rate to catch up with the
	// missed tick. The guest time should not be delayed once catchup is complete.
	RTCTickPolicyCatchup RTCTickPolicy = "catchup"
)

// RTCTimerTrack specifies from which source to track the time.
// ---
// +k8s:openapi-gen=true
type RTCTimerTrack string

const (
	// TrackGuest tracks the guest time.
	TrackGuest RTCTimerTrack = "guest"
	// TrackWall tracks the host time.
	TrackWall RTCTimerTrack = "wall"
)

// ---
// +k8s:openapi-gen=true
type RTCTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
	// One of "delay", "catchup".
	TickPolicy RTCTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
	// Track the guest or the wall clock.
	Track RTCTimerTrack `json:"track,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type HPETTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
	// One of "delay", "catchup", "merge", "discard".
	TickPolicy HPETTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type PITTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
	// One of "delay", "catchup", "discard".
	TickPolicy PITTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type KVMTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type HypervTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Features struct {
	// ACPI enables/disables ACPI insidejsondata guest.
	// Defaults to enabled.
	// +optional
	ACPI FeatureState `json:"acpi,omitempty"`
	// Defaults to the machine type setting.
	// +optional
	APIC *FeatureAPIC `json:"apic,omitempty"`
	// Defaults to the machine type setting.
	// +optional
	Hyperv *FeatureHyperv `json:"hyperv,omitempty"`
	// SMM enables/disables System Management Mode.
	// TSEG not yet implemented.
	// +optional
	SMM *FeatureState `json:"smm,omitempty"`
}

// Represents if a feature is enabled or disabled.
// ---
// +k8s:openapi-gen=true
type FeatureState struct {
	// Enabled determines if the feature should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type FeatureAPIC struct {
	// Enabled determines if the feature should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// EndOfInterrupt enables the end of interrupt notification in the guest.
	// Defaults to false.
	// +optional
	EndOfInterrupt bool `json:"endOfInterrupt,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type FeatureSpinlocks struct {
	// Enabled determines if the feature should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Retries indicates the number of retries.
	// Must be a value greater or equal 4096.
	// Defaults to 4096.
	// +optional
	Retries *uint32 `json:"spinlocks,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type FeatureVendorID struct {
	// Enabled determines if the feature should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// VendorID sets the hypervisor vendor id, visible to the vmi.
	// String up to twelve characters.
	VendorID string `json:"vendorid,omitempty"`
}

// Hyperv specific features.
// ---
// +k8s:openapi-gen=true
type FeatureHyperv struct {
	// Relaxed instructs the guest OS to disable watchdog timeouts.
	// Defaults to the machine type setting.
	// +optional
	Relaxed *FeatureState `json:"relaxed,omitempty"`
	// VAPIC improves the paravirtualized handling of interrupts.
	// Defaults to the machine type setting.
	// +optional
	VAPIC *FeatureState `json:"vapic,omitempty"`
	// Spinlocks allows to configure the spinlock retry attempts.
	// +optional
	Spinlocks *FeatureSpinlocks `json:"spinlocks,omitempty"`
	// VPIndex enables the Virtual Processor Index to help windows identifying virtual processors.
	// Defaults to the machine type setting.
	// +optional
	VPIndex *FeatureState `json:"vpindex,omitempty"`
	// Runtime improves the time accounting to improve scheduling in the guest.
	// Defaults to the machine type setting.
	// +optional
	Runtime *FeatureState `json:"runtime,omitempty"`
	// SyNIC enables the Synthetic Interrupt Controller.
	// Defaults to the machine type setting.
	// +optional
	SyNIC *FeatureState `json:"synic,omitempty"`
	// SyNICTimer enables Synthetic Interrupt Controller Timers, reducing CPU load.
	// Defaults to the machine type setting.
	// +optional
	SyNICTimer *FeatureState `json:"synictimer,omitempty"`
	// Reset enables Hyperv reboot/reset for the vmi. Requires synic.
	// Defaults to the machine type setting.
	// +optional
	Reset *FeatureState `json:"reset,omitempty"`
	// VendorID allows setting the hypervisor vendor id.
	// Defaults to the machine type setting.
	// +optional
	VendorID *FeatureVendorID `json:"vendorid,omitempty"`
	// Frequencies improves the TSC clock source handling for Hyper-V on KVM.
	// Defaults to the machine type setting.
	// +optional
	Frequencies *FeatureState `json:"frequencies,omitempty"`
	// Reenlightenment enables the notifications on TSC frequency changes.
	// Defaults to the machine type setting.
	// +optional
	Reenlightenment *FeatureState `json:"reenlightenment,omitempty"`
	// TLBFlush improves performances in overcommited environments. Requires vpindex.
	// Defaults to the machine type setting.
	// +optional
	TLBFlush *FeatureState `json:"tlbflush,omitempty"`
	// IPI improves performances in overcommited environments. Requires vpindex.
	// Defaults to the machine type setting.
	// +optional
	IPI *FeatureState `json:"ipi,omitempty"`
	// EVMCS Speeds up L2 vmexits, but disables other virtualization features. Requires vapic.
	// Defaults to the machine type setting.
	// +optional
	EVMCS *FeatureState `json:"evmcs,omitempty"`
}

// WatchdogAction defines the watchdog action, if a watchdog gets triggered.
// ---
// +k8s:openapi-gen=true
type WatchdogAction string

const (
	// WatchdogActionPoweroff will poweroff the vmi if the watchdog gets triggered.
	WatchdogActionPoweroff WatchdogAction = "poweroff"
	// WatchdogActionReset will reset the vmi if the watchdog gets triggered.
	WatchdogActionReset WatchdogAction = "reset"
	// WatchdogActionShutdown will shutdown the vmi if the watchdog gets triggered.
	WatchdogActionShutdown WatchdogAction = "shutdown"
)

// Named watchdog device.
// ---
// +k8s:openapi-gen=true
type Watchdog struct {
	// Name of the watchdog.
	Name string `json:"name"`
	// WatchdogDevice contains the watchdog type and actions.
	// Defaults to i6300esb.
	WatchdogDevice `json:",inline"`
}

// Hardware watchdog device.
// Exactly one of its members must be set.
// ---
// +k8s:openapi-gen=true
type WatchdogDevice struct {
	// i6300esb watchdog device.
	// +optional
	I6300ESB *I6300ESBWatchdog `json:"i6300esb,omitempty"`
}

// i6300esb watchdog device.
// ---
// +k8s:openapi-gen=true
type I6300ESBWatchdog struct {
	// The action to take. Valid values are poweroff, reset, shutdown.
	// Defaults to reset.
	Action WatchdogAction `json:"action,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type Interface struct {
	// Logical name of the interface as well as a reference to the associated networks.
	// Must match the Name of a Network.
	Name string `json:"name"`
	// Interface model.
	// One of: e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio.
	// Defaults to virtio.
	// TODO:(ihar) switch to enums once opengen-api supports them. See: https://github.com/kubernetes/kube-openapi/issues/51
	Model string `json:"model,omitempty"`
	// BindingMethod specifies the method which will be used to connect the interface to the guest.
	// Defaults to Bridge.
	InterfaceBindingMethod `json:",inline"`
	// List of ports to be forwarded to the virtual machine.
	Ports []Port `json:"ports,omitempty"`
	// Interface MAC address. For example: de:ad:00:00:be:af or DE-AD-00-00-BE-AF.
	MacAddress string `json:"macAddress,omitempty"`
	// BootOrder is an integer value > 0, used to determine ordering of boot devices.
	// Lower values take precedence.
	// Each interface or disk that has a boot order must have a unique value.
	// Interfaces without a boot order are not tried.
	// +optional
	BootOrder *uint `json:"bootOrder,omitempty"`
	// If specified, the virtual network interface will be placed on the guests pci address with the specifed PCI address. For example: 0000:81:01.10
	// +optional
	PciAddress string `json:"pciAddress,omitempty"`
	// If specified the network interface will pass additional DHCP options to the VMI
	// +optional
	DHCPOptions *DHCPOptions `json:"dhcpOptions,omitempty"`
}

// Extra DHCP options to use in the interface.
// ---
// +k8s:openapi-gen=true
type DHCPOptions struct {
	// If specified will pass option 67 to interface's DHCP server
	// +optional
	BootFileName string `json:"bootFileName,omitempty"`
	// If specified will pass option 66 to interface's DHCP server
	// +optional
	TFTPServerName string `json:"tftpServerName,omitempty"`
	// If specified will pass the configured NTP server to the VM via DHCP option 042.
	// +optional
	NTPServers []string `json:"ntpServers,omitempty"`
	// If specified will pass extra DHCP options for private use, range: 224-254
	// +optional
	PrivateOptions []DHCPPrivateOptions `json:"privateOptions,omitempty"`
}

// DHCPExtraOptions defines Extra DHCP options for a VM.
type DHCPPrivateOptions struct {
	// Option is an Integer value from 224-254
	// Required.
	Option int `json:"option"`
	// Value is a String value for the Option provided
	// Required.
	Value string `json:"value"`
}

// Represents the method which will be used to connect the interface to the guest.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type InterfaceBindingMethod struct {
	Bridge     *InterfaceBridge     `json:"bridge,omitempty"`
	Slirp      *InterfaceSlirp      `json:"slirp,omitempty"`
	Masquerade *InterfaceMasquerade `json:"masquerade,omitempty"`
	SRIOV      *InterfaceSRIOV      `json:"sriov,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type InterfaceBridge struct{}

// ---
// +k8s:openapi-gen=true
type InterfaceSlirp struct{}

// ---
// +k8s:openapi-gen=true
type InterfaceMasquerade struct{}

// ---
// +k8s:openapi-gen=true
type InterfaceSRIOV struct{}

// Port repesents a port to expose from the virtual machine.
// Default protocol TCP.
// The port field is mandatory
// ---
// +k8s:openapi-gen=true
type Port struct {
	// If specified, this must be an IANA_SVC_NAME and unique within the pod. Each
	// named port in a pod must have a unique name. Name for the port that can be
	// referred to by services.
	// +optional
	Name string `json:"name,omitempty"`
	// Protocol for port. Must be UDP or TCP.
	// Defaults to "TCP".
	// +optional
	Protocol string `json:"protocol,omitempty"`
	// Number of port to expose for the virtual machine.
	// This must be a valid port number, 0 < x < 65536.
	Port int32 `json:"port"`
}

// Network represents a network type and a resource that should be connected to the vm.
// ---
// +k8s:openapi-gen=true
type Network struct {
	// Network name.
	// Must be a DNS_LABEL and unique within the vm.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
	// NetworkSource represents the network type and the source interface that should be connected to the virtual machine.
	// Defaults to Pod, if no type is specified.
	NetworkSource `json:",inline"`
}

// Represents the source resource that will be connected to the vm.
// Only one of its members may be specified.
// ---
// +k8s:openapi-gen=true
type NetworkSource struct {
	Pod    *PodNetwork    `json:"pod,omitempty"`
	Multus *MultusNetwork `json:"multus,omitempty"`
	Genie  *GenieNetwork  `json:"genie,omitempty"`
}

// Represents the stock pod network interface.
// ---
// +k8s:openapi-gen=true
type PodNetwork struct {
	// CIDR for vm network.
	// Default 10.0.2.0/24 if not specified.
	VMNetworkCIDR string `json:"vmNetworkCIDR,omitempty"`
}

// Rng represents the random device passed from host
// ---
// +k8s:openapi-gen=true
type Rng struct {
}

// Represents the genie cni network.
// ---
// +k8s:openapi-gen=true
type GenieNetwork struct {
	// References the CNI plugin name.
	NetworkName string `json:"networkName"`
}

// Represents the multus cni network.
// ---
// +k8s:openapi-gen=true
type MultusNetwork struct {
	// References to a NetworkAttachmentDefinition CRD object. Format:
	// <networkName>, <namespace>/<networkName>. If namespace is not
	// specified, VMI namespace is assumed.
	NetworkName string `json:"networkName"`

	// Select the default network and add it to the
	// multus-cni.io/default-network annotation.
	Default bool `json:"default,omitempty"`
}
