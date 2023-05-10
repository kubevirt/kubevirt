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
	"encoding/json"

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
	DefaultCPUModel                        = CPUModeHostModel
)

const HotplugDiskDir = "/var/run/kubevirt/hotplug-disks/"

type DiskErrorPolicy string

const (
	DiskErrorPolicyStop     DiskErrorPolicy = "stop"
	DiskErrorPolicyIgnore   DiskErrorPolicy = "ignore"
	DiskErrorPolicyReport   DiskErrorPolicy = "report"
	DiskErrorPolicyEnospace DiskErrorPolicy = "enospace"
)

/*
 ATTENTION: Rerun code generators when comments on structs or fields are modified.
*/

// Represents a disk created on the cluster level
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
type ConfigMapVolumeSource struct {
	v1.LocalObjectReference `json:",inline"`
	// Specify whether the ConfigMap or it's keys must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
	// The volume label of the resulting disk inside the VMI.
	// Different bootstrapping mechanisms require different values.
	// Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
	// +optional
	VolumeLabel string `json:"volumeLabel,omitempty"`
}

// SecretVolumeSource adapts a Secret into a volume.
type SecretVolumeSource struct {
	// Name of the secret in the pod's namespace to use.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#secret
	SecretName string `json:"secretName,omitempty"`
	// Specify whether the Secret or it's keys must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
	// The volume label of the resulting disk inside the VMI.
	// Different bootstrapping mechanisms require different values.
	// Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
	// +optional
	VolumeLabel string `json:"volumeLabel,omitempty"`
}

// DownwardAPIVolumeSource represents a volume containing downward API info.
type DownwardAPIVolumeSource struct {
	// Fields is a list of downward API volume file
	// +optional
	Fields []v1.DownwardAPIVolumeFile `json:"fields,omitempty"`
	// The volume label of the resulting disk inside the VMI.
	// Different bootstrapping mechanisms require different values.
	// Typical values are "cidata" (cloud-init), "config-2" (cloud-init) or "OEMDRV" (kickstart).
	// +optional
	VolumeLabel string `json:"volumeLabel,omitempty"`
}

// ServiceAccountVolumeSource adapts a ServiceAccount into a volume.
type ServiceAccountVolumeSource struct {
	// Name of the service account in the pod's namespace to use.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// DownwardMetricsVolumeSource adds a very small disk to VMIs which contains a limited view of host and guest
// metrics. The disk content is compatible with vhostmd (https://github.com/vhostmd/vhostmd) and vm-dump-metrics.
type DownwardMetricsVolumeSource struct {
}

// Represents a Sysprep volume source.
type SysprepSource struct {
	// Secret references a k8s Secret that contains Sysprep answer file named autounattend.xml that should be attached as disk of CDROM type.
	// + optional
	Secret *v1.LocalObjectReference `json:"secret,omitempty"`
	// ConfigMap references a ConfigMap that contains Sysprep answer file named autounattend.xml that should be attached as disk of CDROM type.
	// + optional
	ConfigMap *v1.LocalObjectReference `json:"configMap,omitempty"`
}

// Represents a cloud-init nocloud user data source.
// More info: http://cloudinit.readthedocs.io/en/latest/topics/datasources/nocloud.html
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
	Machine *Machine `json:"machine,omitempty"`
	// Firmware.
	// +optional
	Firmware *Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv, smm.
	// +optional
	Features *Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, and others
	Devices Devices `json:"devices"`
	// Controls whether or not disks will share IOThreads.
	// Omitting IOThreadsPolicy disables use of IOThreads.
	// One of: shared, auto
	// +optional
	IOThreadsPolicy *IOThreadsPolicy `json:"ioThreadsPolicy,omitempty"`
	// Chassis specifies the chassis info passed to the domain.
	// +optional
	Chassis *Chassis `json:"chassis,omitempty"`
	// Launch Security setting of the vmi.
	// +optional
	LaunchSecurity *LaunchSecurity `json:"launchSecurity,omitempty"`
}

// Chassis specifies the chassis info passed to the domain.
type Chassis struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Version      string `json:"version,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Asset        string `json:"asset,omitempty"`
	Sku          string `json:"sku,omitempty"`
}

// Represents the firmware blob used to assist in the domain creation process.
// Used for setting the QEMU BIOS file path for the libvirt domain.
type Bootloader struct {
	// If set (default), BIOS will be used.
	// +optional
	BIOS *BIOS `json:"bios,omitempty"`
	// If set, EFI will be used instead of BIOS.
	// +optional
	EFI *EFI `json:"efi,omitempty"`
}

// If set (default), BIOS will be used.
type BIOS struct {
	// If set, the BIOS output will be transmitted over serial
	// +optional
	UseSerial *bool `json:"useSerial,omitempty"`
}

// If set, EFI will be used instead of BIOS.
type EFI struct {
	// If set, SecureBoot will be enabled and the OVMF roms will be swapped for
	// SecureBoot-enabled ones.
	// Requires SMM to be enabled.
	// Defaults to true
	// +optional
	SecureBoot *bool `json:"secureBoot,omitempty"`
	// If set to true, Persistent will persist the EFI NVRAM across reboots.
	// Defaults to false
	// +optional
	Persistent *bool `json:"persistent,omitempty"`
}

// If set, the VM will be booted from the defined kernel / initrd.
type KernelBootContainer struct {
	// Image that contains initrd / kernel files.
	Image string `json:"image"`
	// ImagePullSecret is the name of the Docker registry secret required to pull the image. The secret must already exist.
	//+optional
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// The fully-qualified path to the kernel image in the host OS
	//+optional
	KernelPath string `json:"kernelPath,omitempty"`
	// the fully-qualified path to the ramdisk image in the host OS
	//+optional
	InitrdPath string `json:"initrdPath,omitempty"`
}

// Represents the firmware blob used to assist in the kernel boot process.
// Used for setting the kernel, initrd and command line arguments
type KernelBoot struct {
	// Arguments to be passed to the kernel at boot time
	KernelArgs string `json:"kernelArgs,omitempty"`
	// Container defines the container that containes kernel artifacts
	Container *KernelBootContainer `json:"container,omitempty"`
}

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
type CPU struct {
	// Cores specifies the number of cores inside the vmi.
	// Must be a value greater or equal 1.
	Cores uint32 `json:"cores,omitempty"`
	// Sockets specifies the number of sockets inside the vmi.
	// Must be a value greater or equal 1.
	Sockets uint32 `json:"sockets,omitempty"`
	// MaxSockets specifies the maximum amount of sockets that can
	// be hotplugged
	MaxSockets uint32 `json:"maxSockets,omitempty"`
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

	// NUMA allows specifying settings for the guest NUMA topology
	// +optional
	NUMA *NUMA `json:"numa,omitempty"`

	// IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place
	// the emulator thread on it.
	// +optional
	IsolateEmulatorThread bool `json:"isolateEmulatorThread,omitempty"`
	// Realtime instructs the virt-launcher to tune the VMI for lower latency, optional for real time workloads
	// +optional
	Realtime *Realtime `json:"realtime,omitempty"`
}

// Realtime holds the tuning knobs specific for realtime workloads.
type Realtime struct {
	// Mask defines the vcpu mask expression that defines which vcpus are used for realtime. Format matches libvirt's expressions.
	// Example: "0-3,^1","0,2,3","2-3"
	// +optional
	Mask string `json:"mask,omitempty"`
}

// NUMAGuestMappingPassthrough instructs kubevirt to model numa topology which is compatible with the CPU pinning on the guest.
// This will result in a subset of the node numa topology being passed through, ensuring that virtual numa nodes and their memory
// never cross boundaries coming from the node numa mapping.
type NUMAGuestMappingPassthrough struct {
}

type NUMA struct {
	// GuestMappingPassthrough will create an efficient guest topology based on host CPUs exclusively assigned to a pod.
	// The created topology ensures that memory and CPUs on the virtual numa nodes never cross boundaries of host numa nodes.
	// +opitonal
	GuestMappingPassthrough *NUMAGuestMappingPassthrough `json:"guestMappingPassthrough,omitempty"`
}

// CPUFeature allows specifying a CPU feature.
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
type Hugepages struct {
	// PageSize specifies the hugepage size, for x86_64 architecture valid values are 1Gi and 2Mi.
	PageSize string `json:"pageSize,omitempty"`
}

type Machine struct {
	// QEMU machine type is the actual chipset of the VirtualMachineInstance.
	// +optional
	Type string `json:"type"`
}

type Firmware struct {
	// UUID reported by the vmi bios.
	// Defaults to a random generated uid.
	UUID types.UID `json:"uuid,omitempty"`
	// Settings to control the bootloader that is used.
	// +optional
	Bootloader *Bootloader `json:"bootloader,omitempty"`
	// The system-serial-number in SMBIOS
	Serial string `json:"serial,omitempty"`
	// Settings to set the kernel for booting.
	// +optional
	KernelBoot *KernelBoot `json:"kernelBoot,omitempty"`
}

type Devices struct {
	// Fall back to legacy virtio 0.9 support if virtio bus is selected on devices.
	// This is helpful for old machines like CentOS6 or RHEL6 which
	// do not understand virtio_non_transitional (virtio 1.0).
	UseVirtioTransitional *bool `json:"useVirtioTransitional,omitempty"`
	// DisableHotplug disabled the ability to hotplug disks.
	DisableHotplug bool `json:"disableHotplug,omitempty"`
	// Disks describes disks, cdroms and luns which are connected to the vmi.
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
	// Whether to attach the default virtio-serial console or not.
	// Serial console access will not be available if set to false. Defaults to true.
	AutoattachSerialConsole *bool `json:"autoattachSerialConsole,omitempty"`
	// Whether to attach the Memory balloon device with default period.
	// Period can be adjusted in virt-config.
	// Defaults to true.
	// +optional
	AutoattachMemBalloon *bool `json:"autoattachMemBalloon,omitempty"`
	// Whether to attach an Input Device.
	// Defaults to false.
	// +optional
	AutoattachInputDevice *bool `json:"autoattachInputDevice,omitempty"`
	// Whether to attach the VSOCK CID to the VM or not.
	// VSOCK access will be available if set to true. Defaults to false.
	AutoattachVSOCK *bool `json:"autoattachVSOCK,omitempty"`
	// Whether to have random number generator from host
	// +optional
	Rng *Rng `json:"rng,omitempty"`
	// Whether or not to enable virtio multi-queue for block devices.
	// Defaults to false.
	// +optional
	BlockMultiQueue *bool `json:"blockMultiQueue,omitempty"`
	// If specified, virtual network interfaces configured with a virtio bus will also enable the vhost multiqueue feature for network devices. The number of queues created depends on additional factors of the VirtualMachineInstance, like the number of guest CPUs.
	// +optional
	NetworkInterfaceMultiQueue *bool `json:"networkInterfaceMultiqueue,omitempty"`
	//Whether to attach a GPU device to the vmi.
	// +optional
	// +listType=atomic
	GPUs []GPU `json:"gpus,omitempty"`
	// DownwardMetrics creates a virtio serials for exposing the downward metrics to the vmi.
	// +optional
	DownwardMetrics *DownwardMetrics `json:"downwardMetrics,omitempty"`
	// Filesystems describes filesystem which is connected to the vmi.
	// +optional
	// +listType=atomic
	Filesystems []Filesystem `json:"filesystems,omitempty"`
	//Whether to attach a host device to the vmi.
	// +optional
	// +listType=atomic
	HostDevices []HostDevice `json:"hostDevices,omitempty"`
	// To configure and access client devices such as redirecting USB
	// +optional
	ClientPassthrough *ClientPassthroughDevices `json:"clientPassthrough,omitempty"`
	// Whether to emulate a sound device.
	// +optional
	Sound *SoundDevice `json:"sound,omitempty"`
	// Whether to emulate a TPM device.
	// +optional
	TPM *TPMDevice `json:"tpm,omitempty"`
}

// Represent a subset of client devices that can be accessed by VMI. At the
// moment only, USB devices using Usbredir's library and tooling. Another fit
// would be a smartcard with libcacard.
//
// The struct is currently empty as there is no immediate request for
// user-facing APIs. This structure simply turns on USB redirection of
// UsbClientPassthroughMaxNumberOf devices.
type ClientPassthroughDevices struct {
}

// Represents the upper limit allowed by QEMU + KubeVirt.
const (
	UsbClientPassthroughMaxNumberOf = 4
)

// Represents the user's configuration to emulate sound cards in the VMI.
type SoundDevice struct {
	// User's defined name for this sound device
	Name string `json:"name"`
	// We only support ich9 or ac97.
	// If SoundDevice is not set: No sound card is emulated.
	// If SoundDevice is set but Model is not: ich9
	// +optional
	Model string `json:"model,omitempty"`
}

type TPMDevice struct {
	// Persistent indicates the state of the TPM device should be kept accross reboots
	// Defaults to false
	Persistent *bool `json:"persistent,omitempty"`
}

type InputBus string

const (
	InputBusUSB    InputBus = "usb"
	InputBusVirtio InputBus = "virtio"
)

type InputType string

const (
	InputTypeTablet   InputType = "tablet"
	InputTypeKeyboard InputType = "keyboard"
)

type Input struct {
	// Bus indicates the bus of input device to emulate.
	// Supported values: virtio, usb.
	Bus InputBus `json:"bus,omitempty"`
	// Type indicated the type of input device.
	// Supported values: tablet.
	Type InputType `json:"type"`
	// Name is the device name
	Name string `json:"name"`
}

type Filesystem struct {
	// Name is the device name
	Name string `json:"name"`
	// Virtiofs is supported
	Virtiofs *FilesystemVirtiofs `json:"virtiofs"`
}

type FilesystemVirtiofs struct{}

type DownwardMetrics struct{}

type GPU struct {
	// Name of the GPU device as exposed by a device plugin
	Name              string       `json:"name"`
	DeviceName        string       `json:"deviceName"`
	VirtualGPUOptions *VGPUOptions `json:"virtualGPUOptions,omitempty"`
	// If specified, the virtual network interface address and its tag will be provided to the guest via config drive
	// +optional
	Tag string `json:"tag,omitempty"`
}

type VGPUOptions struct {
	Display *VGPUDisplayOptions `json:"display,omitempty"`
}

type VGPUDisplayOptions struct {
	// Enabled determines if a display addapter backed by a vGPU should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Enables a boot framebuffer, until the guest OS loads a real GPU driver
	// Defaults to true.
	// +optional
	RamFB *FeatureState `json:"ramFB,omitempty"`
}

type HostDevice struct {
	Name string `json:"name"`
	// DeviceName is the resource name of the host device exposed by a device plugin
	DeviceName string `json:"deviceName"`
	// If specified, the virtual network interface address and its tag will be provided to the guest via config drive
	// +optional
	Tag string `json:"tag,omitempty"`
}

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
	// Supported values are: CacheNone, CacheWriteThrough.
	// +optional
	Cache DriverCache `json:"cache,omitempty"`
	// IO specifies which QEMU disk IO mode should be used.
	// Supported values are: native, default, threads.
	// +optional
	IO DriverIO `json:"io,omitempty"`
	// If specified, disk address and its tag will be provided to the guest via config drive metadata
	// +optional
	Tag string `json:"tag,omitempty"`
	// If specified, the virtual disk will be presented with the given block sizes.
	// +optional
	BlockSize *BlockSize `json:"blockSize,omitempty"`
	// If specified the disk is made sharable and multiple write from different VMs are permitted
	// +optional
	Shareable *bool `json:"shareable,omitempty"`
	// If specified, it can change the default error policy (stop) for the disk
	// +optional
	ErrorPolicy *DiskErrorPolicy `json:"errorPolicy,omitempty"`
}

// CustomBlockSize represents the desired logical and physical block size for a VM disk.
type CustomBlockSize struct {
	Logical  uint `json:"logical"`
	Physical uint `json:"physical"`
}

// BlockSize provides the option to change the block size presented to the VM for a disk.
// Only one of its members may be specified.
type BlockSize struct {
	Custom      *CustomBlockSize `json:"custom,omitempty"`
	MatchVolume *FeatureState    `json:"matchVolume,omitempty"`
}

// Represents the target of a volume to mount.
// Only one of its members may be specified.
type DiskDevice struct {
	// Attach a volume as a disk to the vmi.
	Disk *DiskTarget `json:"disk,omitempty"`
	// Attach a volume as a LUN to the vmi.
	LUN *LunTarget `json:"lun,omitempty"`
	// Attach a volume as a cdrom to the vmi.
	CDRom *CDRomTarget `json:"cdrom,omitempty"`
}

type DiskBus string

const (
	DiskBusSCSI   DiskBus = "scsi"
	DiskBusSATA   DiskBus = "sata"
	DiskBusVirtio DiskBus = VirtIO
	DiskBusUSB    DiskBus = "usb"
)

type DiskTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, usb.
	Bus DiskBus `json:"bus,omitempty"`
	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`
	// If specified, the virtual disk will be placed on the guests pci address with the specified PCI address. For example: 0000:81:01.10
	// +optional
	PciAddress string `json:"pciAddress,omitempty"`
}

type LaunchSecurity struct {
	// AMD Secure Encrypted Virtualization (SEV).
	SEV *SEV `json:"sev,omitempty"`
}

type SEV struct {
	// Guest policy flags as defined in AMD SEV API specification.
	// Note: due to security reasons it is not allowed to enable guest debugging. Therefore NoDebug flag is not exposed to users and is always true.
	Policy *SEVPolicy `json:"policy,omitempty"`
	// If specified, run the attestation process for a vmi.
	// +opitonal
	Attestation *SEVAttestation `json:"attestation,omitempty"`
	// Base64 encoded session blob.
	Session string `json:"session,omitempty"`
	// Base64 encoded guest owner's Diffie-Hellman key.
	DHCert string `json:"dhCert,omitempty"`
}

type SEVPolicy struct {
	// SEV-ES is required.
	// Defaults to false.
	// +optional
	EncryptedState *bool `json:"encryptedState,omitempty"`
}

type SEVAttestation struct {
}

type LunTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus DiskBus `json:"bus,omitempty"`
	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`
	// Reservation indicates if the disk needs to support the persistent reservation for the SCSI disk
	Reservation bool `json:"reservation,omitempty"`
}

// TrayState indicates if a tray of a cdrom is open or closed.
type TrayState string

const (
	// TrayStateOpen indicates that the tray of a cdrom is open.
	TrayStateOpen TrayState = "open"
	// TrayStateClosed indicates that the tray of a cdrom is closed.
	TrayStateClosed TrayState = "closed"
)

type CDRomTarget struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus DiskBus `json:"bus,omitempty"`
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
type VolumeSource struct {
	// HostDisk represents a disk created on the cluster level
	// +optional
	HostDisk *HostDisk `json:"hostDisk,omitempty"`
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
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
	// Represents a Sysprep volume source.
	// +optional
	Sysprep *SysprepSource `json:"sysprep,omitempty"`
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
	// DownwardAPI represents downward API about the pod that should populate this volume
	// +optional
	DownwardAPI *DownwardAPIVolumeSource `json:"downwardAPI,omitempty"`
	// ServiceAccountVolumeSource represents a reference to a service account.
	// There can only be one volume of this type!
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccount *ServiceAccountVolumeSource `json:"serviceAccount,omitempty"`
	// DownwardMetrics adds a very small disk to VMIs which contains a limited view of host and guest
	// metrics. The disk content is compatible with vhostmd (https://github.com/vhostmd/vhostmd) and vm-dump-metrics.
	DownwardMetrics *DownwardMetricsVolumeSource `json:"downwardMetrics,omitempty"`
	// MemoryDump is attached to the virt launcher and is populated with a memory dump of the vmi
	MemoryDump *MemoryDumpVolumeSource `json:"memoryDump,omitempty"`
}

// HotplugVolumeSource Represents the source of a volume to mount which are capable
// of being hotplugged on a live running VMI.
// Only one of its members may be specified.
type HotplugVolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// DataVolume represents the dynamic creation a PVC for this volume as well as
	// the process of populating that PVC with a disk image.
	// +optional
	DataVolume *DataVolumeSource `json:"dataVolume,omitempty"`
}

type DataVolumeSource struct {
	// Name of both the DataVolume and the PVC in the same namespace.
	// After PVC population the DataVolume is garbage collected by default.
	Name string `json:"name"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
}

// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
// Directly attached to the vmi via qemu.
// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
type PersistentVolumeClaimVolumeSource struct {
	v1.PersistentVolumeClaimVolumeSource `json:",inline"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
}

type MemoryDumpVolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the virt launcher
	// +optional
	PersistentVolumeClaimVolumeSource `json:",inline"`
}

type EphemeralVolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *v1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
}

// EmptyDisk represents a temporary disk which shares the vmis lifecycle.
type EmptyDiskSource struct {
	// Capacity of the sparse disk.
	Capacity resource.Quantity `json:"capacity"`
}

// Represents a docker image with an embedded disk.
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
type ClockOffset struct {
	// UTC sets the guest clock to UTC on each boot. If an offset is specified,
	// guest changes to the clock will be kept during reboots and are not reset.
	UTC *ClockOffsetUTC `json:"utc,omitempty"`
	// Timezone sets the guest clock to the specified timezone.
	// Zone name follows the TZ environment variable format (e.g. 'America/New_York').
	Timezone *ClockOffsetTimezone `json:"timezone,omitempty"`
}

// UTC sets the guest clock to UTC on each boot.
type ClockOffsetUTC struct {
	// OffsetSeconds specifies an offset in seconds, relative to UTC. If set,
	// guest changes to the clock will be kept during reboots and not reset.
	OffsetSeconds *int `json:"offsetSeconds,omitempty"`
}

// ClockOffsetTimezone sets the guest clock to the specified timezone.
// Zone name follows the TZ environment variable format (e.g. 'America/New_York').
type ClockOffsetTimezone string

// Represents the clock and timers of a vmi.
// +kubebuilder:pruning:PreserveUnknownFields
type Clock struct {
	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock.
	ClockOffset `json:",inline"`
	// Timer specifies whih timers are attached to the vmi.
	// +optional
	Timer *Timer `json:"timer,omitempty"`
}

// Represents all available timers in a vmi.
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
type HPETTickPolicy string

// PITTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
type PITTickPolicy string

// RTCTickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
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
type RTCTimerTrack string

const (
	// TrackGuest tracks the guest time.
	TrackGuest RTCTimerTrack = "guest"
	// TrackWall tracks the host time.
	TrackWall RTCTimerTrack = "wall"
)

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

type HPETTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
	// One of "delay", "catchup", "merge", "discard".
	TickPolicy HPETTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type PITTimer struct {
	// TickPolicy determines what happens when QEMU misses a deadline for injecting a tick to the guest.
	// One of "delay", "catchup", "discard".
	TickPolicy PITTickPolicy `json:"tickPolicy,omitempty"`
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type KVMTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type HypervTimer struct {
	// Enabled set to false makes sure that the machine type or a preset can't add the timer.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"present,omitempty"`
}

type Features struct {
	// ACPI enables/disables ACPI inside the guest.
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
	// Configure how KVM presence is exposed to the guest.
	// +optional
	KVM *FeatureKVM `json:"kvm,omitempty"`
	// Notify the guest that the host supports paravirtual spinlocks.
	// For older kernels this feature should be explicitly disabled.
	// +optional
	Pvspinlock *FeatureState `json:"pvspinlock,omitempty"`
}

type SyNICTimer struct {
	Enabled *bool         `json:"enabled,omitempty"`
	Direct  *FeatureState `json:"direct,omitempty"`
}

// Represents if a feature is enabled or disabled.
type FeatureState struct {
	// Enabled determines if the feature should be enabled or disabled on the guest.
	// Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

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
	SyNICTimer *SyNICTimer `json:"synictimer,omitempty"`
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

type FeatureKVM struct {
	// Hide the KVM hypervisor from standard MSR based discovery.
	// Defaults to false
	Hidden bool `json:"hidden,omitempty"`
}

// WatchdogAction defines the watchdog action, if a watchdog gets triggered.
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
type Watchdog struct {
	// Name of the watchdog.
	Name string `json:"name"`
	// WatchdogDevice contains the watchdog type and actions.
	// Defaults to i6300esb.
	WatchdogDevice `json:",inline"`
}

// Hardware watchdog device.
// Exactly one of its members must be set.
type WatchdogDevice struct {
	// i6300esb watchdog device.
	// +optional
	I6300ESB *I6300ESBWatchdog `json:"i6300esb,omitempty"`
}

// i6300esb watchdog device.
type I6300ESBWatchdog struct {
	// The action to take. Valid values are poweroff, reset, shutdown.
	// Defaults to reset.
	Action WatchdogAction `json:"action,omitempty"`
}

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
	// Binding specifies the binding plugin that will be used to connect the interface to the guest.
	// It provides an alternative to InterfaceBindingMethod.
	// version: 1alphav1
	Binding *PluginBinding `json:"binding,omitempty"`
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
	// If specified, the virtual network interface will be placed on the guests pci address with the specified PCI address. For example: 0000:81:01.10
	// +optional
	PciAddress string `json:"pciAddress,omitempty"`
	// If specified the network interface will pass additional DHCP options to the VMI
	// +optional
	DHCPOptions *DHCPOptions `json:"dhcpOptions,omitempty"`
	// If specified, the virtual network interface address and its tag will be provided to the guest via config drive
	// +optional
	Tag string `json:"tag,omitempty"`
	// If specified, the ACPI index is used to provide network interface device naming, that is stable across changes
	// in PCI addresses assigned to the device.
	// This value is required to be unique across all devices and be between 1 and (16*1024-1).
	// +optional
	ACPIIndex int `json:"acpiIndex,omitempty"`
	// State represents the requested operational state of the interface.
	// The (only) value supported is `absent`, expressing a request to remove the interface.
	// +optional
	State InterfaceState `json:"state,omitempty"`
}

type InterfaceState string

const (
	InterfaceStateAbsent InterfaceState = "absent"
)

// Extra DHCP options to use in the interface.
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

func (d *DHCPOptions) UnmarshalJSON(data []byte) error {
	type DHCPOptionsAlias DHCPOptions
	var dhcpOptionsAlias DHCPOptionsAlias

	if err := json.Unmarshal(data, &dhcpOptionsAlias); err != nil {
		return err
	}

	for i, ntpServer := range dhcpOptionsAlias.NTPServers {
		if sanitizedIP, err := sanitizeIP(ntpServer); err == nil {
			dhcpOptionsAlias.NTPServers[i] = sanitizedIP
		}
	}

	*d = DHCPOptions(dhcpOptionsAlias)
	return nil
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
type InterfaceBindingMethod struct {
	Bridge     *InterfaceBridge     `json:"bridge,omitempty"`
	Slirp      *InterfaceSlirp      `json:"slirp,omitempty"`
	Masquerade *InterfaceMasquerade `json:"masquerade,omitempty"`
	SRIOV      *InterfaceSRIOV      `json:"sriov,omitempty"`
	Macvtap    *InterfaceMacvtap    `json:"macvtap,omitempty"`
	Passt      *InterfacePasst      `json:"passt,omitempty"`
}

// InterfaceBridge connects to a given network via a linux bridge.
type InterfaceBridge struct{}

// InterfaceSlirp connects to a given network using QEMU user networking mode.
type InterfaceSlirp struct{}

// InterfaceMasquerade connects to a given network using netfilter rules to nat the traffic.
type InterfaceMasquerade struct{}

// InterfaceSRIOV connects to a given network by passing-through an SR-IOV PCI device via vfio.
type InterfaceSRIOV struct{}

// InterfaceMacvtap connects to a given network by extending the Kubernetes node's L2 networks via a macvtap interface.
type InterfaceMacvtap struct{}

// InterfacePasst connects to a given network.
type InterfacePasst struct{}

// PluginBinding represents a binding implemented in a plugin.
type PluginBinding struct {
	// Name references to the binding name as denined in the kubevirt CR.
	// version: 1alphav1
	Name string `json:"name"`
}

// Port represents a port to expose from the virtual machine.
// Default protocol TCP.
// The port field is mandatory
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

type AccessCredentialSecretSource struct {
	// SecretName represents the name of the secret in the VMI's namespace
	SecretName string `json:"secretName"`
}

type ConfigDriveSSHPublicKeyAccessCredentialPropagation struct{}

// AuthorizedKeysFile represents a path within the guest
// that ssh public keys should be propagated to
type AuthorizedKeysFile struct {
	// FilePath represents the place on the guest that the authorized_keys
	// file should be writen to. This is expected to be a full path including
	// both the base directory and file name.
	FilePath string `json:"filePath"`
}

type QemuGuestAgentUserPasswordAccessCredentialPropagation struct{}

type QemuGuestAgentSSHPublicKeyAccessCredentialPropagation struct {
	// Users represents a list of guest users that should have the ssh public keys
	// added to their authorized_keys file.
	// +listType=set
	Users []string `json:"users"`
}

// SSHPublicKeyAccessCredentialSource represents where to retrieve the ssh key
// credentials
// Only one of its members may be specified.
type SSHPublicKeyAccessCredentialSource struct {
	// Secret means that the access credential is pulled from a kubernetes secret
	// +optional
	Secret *AccessCredentialSecretSource `json:"secret,omitempty"`
}

// SSHPublicKeyAccessCredentialPropagationMethod represents the method used to
// inject a ssh public key into the vm guest.
// Only one of its members may be specified.
type SSHPublicKeyAccessCredentialPropagationMethod struct {
	// ConfigDrivePropagation means that the ssh public keys are injected
	// into the VM using metadata using the configDrive cloud-init provider
	// +optional
	ConfigDrive *ConfigDriveSSHPublicKeyAccessCredentialPropagation `json:"configDrive,omitempty"`

	// QemuGuestAgentAccessCredentailPropagation means ssh public keys are
	// dynamically injected into the vm at runtime via the qemu guest agent.
	// This feature requires the qemu guest agent to be running within the guest.
	// +optional
	QemuGuestAgent *QemuGuestAgentSSHPublicKeyAccessCredentialPropagation `json:"qemuGuestAgent,omitempty"`
}

// SSHPublicKeyAccessCredential represents a source and propagation method for
// injecting ssh public keys into a vm guest
type SSHPublicKeyAccessCredential struct {
	// Source represents where the public keys are pulled from
	Source SSHPublicKeyAccessCredentialSource `json:"source"`

	// PropagationMethod represents how the public key is injected into the vm guest.
	PropagationMethod SSHPublicKeyAccessCredentialPropagationMethod `json:"propagationMethod"`
}

// UserPasswordAccessCredentialSource represents where to retrieve the user password
// credentials
// Only one of its members may be specified.
type UserPasswordAccessCredentialSource struct {
	// Secret means that the access credential is pulled from a kubernetes secret
	// +optional
	Secret *AccessCredentialSecretSource `json:"secret,omitempty"`
}

// UserPasswordAccessCredentialPropagationMethod represents the method used to
// inject a user passwords into the vm guest.
// Only one of its members may be specified.
type UserPasswordAccessCredentialPropagationMethod struct {
	// QemuGuestAgentAccessCredentailPropagation means passwords are
	// dynamically injected into the vm at runtime via the qemu guest agent.
	// This feature requires the qemu guest agent to be running within the guest.
	// +optional
	QemuGuestAgent *QemuGuestAgentUserPasswordAccessCredentialPropagation `json:"qemuGuestAgent,omitempty"`
}

// UserPasswordAccessCredential represents a source and propagation method for
// injecting user passwords into a vm guest
// Only one of its members may be specified.
type UserPasswordAccessCredential struct {
	// Source represents where the user passwords are pulled from
	Source UserPasswordAccessCredentialSource `json:"source"`

	// propagationMethod represents how the user passwords are injected into the vm guest.
	PropagationMethod UserPasswordAccessCredentialPropagationMethod `json:"propagationMethod"`
}

// AccessCredential represents a credential source that can be used to
// authorize remote access to the vm guest
// Only one of its members may be specified.
type AccessCredential struct {
	// SSHPublicKey represents the source and method of applying a ssh public
	// key into a guest virtual machine.
	// +optional
	SSHPublicKey *SSHPublicKeyAccessCredential `json:"sshPublicKey,omitempty"`
	// UserPassword represents the source and method for applying a guest user's
	// password
	// +optional
	UserPassword *UserPasswordAccessCredential `json:"userPassword,omitempty"`
}

// Network represents a network type and a resource that should be connected to the vm.
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
type NetworkSource struct {
	Pod    *PodNetwork    `json:"pod,omitempty"`
	Multus *MultusNetwork `json:"multus,omitempty"`
}

// Represents the stock pod network interface.
type PodNetwork struct {
	// CIDR for vm network.
	// Default 10.0.2.0/24 if not specified.
	VMNetworkCIDR string `json:"vmNetworkCIDR,omitempty"`

	// IPv6 CIDR for the vm network.
	// Defaults to fd10:0:2::/120 if not specified.
	VMIPv6NetworkCIDR string `json:"vmIPv6NetworkCIDR,omitempty"`
}

func (podNet *PodNetwork) UnmarshalJSON(data []byte) error {
	type PodNetworkAlias PodNetwork
	var podNetAlias PodNetworkAlias

	if err := json.Unmarshal(data, &podNetAlias); err != nil {
		return err
	}

	if sanitizedCIDR, err := sanitizeCIDR(podNetAlias.VMNetworkCIDR); err == nil {
		podNetAlias.VMNetworkCIDR = sanitizedCIDR
	}

	*podNet = PodNetwork(podNetAlias)
	return nil
}

// Rng represents the random device passed from host
type Rng struct {
}

// Represents the multus cni network.
type MultusNetwork struct {
	// References to a NetworkAttachmentDefinition CRD object. Format:
	// <networkName>, <namespace>/<networkName>. If namespace is not
	// specified, VMI namespace is assumed.
	NetworkName string `json:"networkName"`

	// Select the default network and add it to the
	// multus-cni.io/default-network annotation.
	Default bool `json:"default,omitempty"`
}

// CPUTopology allows specifying the amount of cores, sockets
// and threads.
type CPUTopology struct {
	// Cores specifies the number of cores inside the vmi.
	// Must be a value greater or equal 1.
	Cores uint32 `json:"cores,omitempty"`
	// Sockets specifies the number of sockets inside the vmi.
	// Must be a value greater or equal 1.
	Sockets uint32 `json:"sockets,omitempty"`
	// Threads specifies the number of threads inside the vmi.
	// Must be a value greater or equal 1.
	Threads uint32 `json:"threads,omitempty"`
}
