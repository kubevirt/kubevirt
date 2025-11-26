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
 * Copyright The KubeVirt Authors.
 *
 */

package instancetype

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

// VirtualMachineInstancetype resource contains quantitative and resource related VirtualMachine configuration
// that can be used by multiple VirtualMachine resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstancetype struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Required spec describing the instancetype
	Spec VirtualMachineInstancetypeSpec
}

// VirtualMachineInstancetypeList is a list of VirtualMachineInstancetype resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstancetypeList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []VirtualMachineInstancetype
}

// VirtualMachineClusterInstancetype is a cluster scoped version of VirtualMachineInstancetype resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterInstancetype struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Required spec describing the instancetype
	Spec VirtualMachineInstancetypeSpec
}

// VirtualMachineClusterInstancetypeList is a list of VirtualMachineClusterInstancetype resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterInstancetypeList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []VirtualMachineClusterInstancetype
}

// VirtualMachineInstancetypeSpec is a description of the VirtualMachineInstancetype or VirtualMachineClusterInstancetype.
//
// CPU and Memory are required attributes with both requiring that their Guest attribute is defined, ensuring a number of vCPUs and amount of RAM is always provided by each instancetype.
type VirtualMachineInstancetypeSpec struct {
	// NodeSelector is a selector which must be true for the vmi to fit on a node.
	// Selector which must match a node's labels for the vmi to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	//
	// NodeSelector is the name of the custom node selector for the instancetype.
	NodeSelector map[string]string

	// If specified, the VMI will be dispatched by specified scheduler.
	// If not specified, the VMI will be dispatched by default scheduler.
	//
	// SchedulerName is the name of the custom K8s scheduler for the instancetype.
	SchedulerName string

	// Required CPU related attributes of the instancetype.
	CPU CPUInstancetype

	// Required Memory related attributes of the instancetype.
	Memory MemoryInstancetype

	// Optionally defines any GPU devices associated with the instancetype.
	GPUs []v1.GPU

	// Optionally defines any HostDevices associated with the instancetype.
	HostDevices []v1.HostDevice

	// Optionally defines the IOThreadsPolicy to be used by the instancetype.
	IOThreadsPolicy *v1.IOThreadsPolicy

	// Optionally specifies the IOThreads options to be used by the instancetype.
	IOThreads *v1.DiskIOThreads

	// Optionally defines the LaunchSecurity to be used by the instancetype.
	LaunchSecurity *v1.LaunchSecurity

	// Optionally defines the required Annotations to be used by the instance type and applied to the VirtualMachineInstance
	Annotations map[string]string
}

// CPUInstancetype contains the CPU related configuration of a given VirtualMachineInstancetypeSpec.
//
// Guest is a required attribute and defines the number of vCPUs to be exposed to the guest by the instancetype.
type CPUInstancetype struct {
	// Required number of vCPUs to expose to the guest.
	//
	// The resulting CPU topology being derived from the optional PreferredCPUTopology attribute of CPUPreferences that itself defaults to PreferSockets.
	Guest uint32

	// Model specifies the CPU model inside the VMI.
	// List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map.
	// It is possible to specify special cases like "host-passthrough" to get the same CPU as the node
	// and "host-model" to get CPU closest to the node one.
	// Defaults to host-model.
	Model *string

	// DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node
	// with enough dedicated pCPUs and pin the vCPUs to it.
	DedicatedCPUPlacement *bool

	// NUMA allows specifying settings for the guest NUMA topology
	NUMA *v1.NUMA

	// IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place
	// the emulator thread on it.
	IsolateEmulatorThread *bool

	// Realtime instructs the virt-launcher to tune the VMI for lower latency, optional for real time workloads
	Realtime *v1.Realtime

	// MaxSockets specifies the maximum amount of sockets that can be hotplugged
	MaxSockets *uint32
}

// MemoryInstancetype contains the Memory related configuration of a given VirtualMachineInstancetypeSpec.
//
// Guest is a required attribute and defines the amount of RAM to be exposed to the guest by the instancetype.
type MemoryInstancetype struct {
	// Required amount of memory which is visible inside the guest OS.
	Guest resource.Quantity

	// Optionally enables the use of hugepages for the VirtualMachineInstance instead of regular memory.
	Hugepages *v1.Hugepages

	// OvercommitPercent is the percentage of the guest memory which will be overcommitted.
	// This means that the VMIs parent pod (virt-launcher) will request less
	// physical memory by a factor specified by the OvercommitPercent.
	// Overcommits can lead to memory exhaustion, which in turn can lead to crashes. Use carefully.
	// Defaults to 0
	OvercommitPercent int

	// MaxGuest allows to specify the maximum amount of memory which is visible inside the Guest OS.
	// The delta between MaxGuest and Guest is the amount of memory that can be hot(un)plugged.
	MaxGuest *resource.Quantity
}

// VirtualMachinePreference resource contains optional preferences related to the VirtualMachine.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachinePreference struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Required spec describing the preferences
	Spec VirtualMachinePreferenceSpec
}

// VirtualMachinePreferenceList is a list of VirtualMachinePreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachinePreferenceList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []VirtualMachinePreference
}

// VirtualMachineClusterPreference is a cluster scoped version of the VirtualMachinePreference resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterPreference struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Required spec describing the preferences
	Spec VirtualMachinePreferenceSpec
}

// VirtualMachineClusterPreferenceList is a list of VirtualMachineClusterPreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterPreferenceList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []VirtualMachineClusterPreference
}

// VirtualMachinePreferenceSpec is a description of the VirtualMachinePreference or VirtualMachineClusterPreference.
type VirtualMachinePreferenceSpec struct {
	// Clock optionally defines preferences associated with the Clock attribute of a VirtualMachineInstance DomainSpec
	Clock *ClockPreferences

	// CPU optionally defines preferences associated with the CPU attribute of a VirtualMachineInstance DomainSpec
	CPU *CPUPreferences

	// Devices optionally defines preferences associated with the Devices attribute of a VirtualMachineInstance DomainSpec
	Devices *DevicePreferences

	// Features optionally defines preferences associated with the Features attribute of a VirtualMachineInstance DomainSpec
	Features *FeaturePreferences

	// Firmware optionally defines preferences associated with the Firmware attribute of a VirtualMachineInstance DomainSpec
	Firmware *FirmwarePreferences

	// Machine optionally defines preferences associated with the Machine attribute of a VirtualMachineInstance DomainSpec
	Machine *MachinePreferences

	// Volumes optionally defines preferences associated with the Volumes attribute of a VirtualMachineInstace DomainSpec
	Volumes *VolumePreferences

	// Subdomain of the VirtualMachineInstance
	PreferredSubdomain *string

	// Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
	PreferredTerminationGracePeriodSeconds *int64

	// Requirements defines the minium amount of instance type defined resources required by a set of preferences
	Requirements *PreferenceRequirements

	// Optionally defines preferred Annotations to be applied to the VirtualMachineInstance
	Annotations map[string]string

	// PreferSpreadSocketToCoreRatio defines the ratio to spread vCPUs between cores and sockets, it defaults to 2.
	PreferSpreadSocketToCoreRatio uint32

	// PreferredArchitecture defines a prefeerred architecture for the VirtualMachine
	PreferredArchitecture *string
}

type VolumePreferences struct {
	// PreffereedStorageClassName optionally defines the preferred storageClass
	PreferredStorageClassName string
}

// PreferredCPUTopology defines a preferred CPU topology to be exposed to the guest
type PreferredCPUTopology string

const (
	// Prefer vCPUs to be exposed as cores to the guest
	DeprecatedPreferCores PreferredCPUTopology = "preferCores"

	// Prefer vCPUs to be exposed as sockets to the guest, this is the default for the PreferredCPUTopology attribute of CPUPreferences.
	DeprecatedPreferSockets PreferredCPUTopology = "preferSockets"

	// Prefer vCPUs to be exposed as threads to the guest
	DeprecatedPreferThreads PreferredCPUTopology = "preferThreads"

	// Prefer vCPUs to be spread evenly between cores and sockets with any remaining vCPUs being presented as cores
	DeprecatedPreferSpread PreferredCPUTopology = "preferSpread"

	// Prefer vCPUs to be spread according to VirtualMachineInstanceTemplateSpec
	//
	// If used with VirtualMachineInstanceType it will use sockets as default
	DeprecatedPreferAny PreferredCPUTopology = "preferAny"

	// Prefer vCPUs to be exposed as cores to the guest
	Cores PreferredCPUTopology = "cores"

	// Prefer vCPUs to be exposed as sockets to the guest, this is the default for the PreferredCPUTopology attribute of CPUPreferences.
	Sockets PreferredCPUTopology = "sockets"

	// Prefer vCPUs to be exposed as threads to the guest
	Threads PreferredCPUTopology = "threads"

	// Prefer vCPUs to be spread evenly between cores and sockets with any remaining vCPUs being presented as cores
	Spread PreferredCPUTopology = "spread"

	// Prefer vCPUs to be spread according to VirtualMachineInstanceTemplateSpec
	//
	// If used with VirtualMachineInstanceType it will use sockets as default
	Any PreferredCPUTopology = "any"
)

// CPUPreferences contains various optional CPU preferences.
type CPUPreferences struct {
	// PreferredCPUTopology optionally defines the preferred guest visible CPU topology, defaults to PreferSockets.
	PreferredCPUTopology *PreferredCPUTopology

	SpreadOptions *SpreadOptions

	// PreferredCPUFeatures optionally defines a slice of preferred CPU features.
	PreferredCPUFeatures []v1.CPUFeature
}

type SpreadAcross string

const (
	// Spread vCPUs across sockets, cores and threads
	SpreadAcrossSocketsCoresThreads SpreadAcross = "SocketsCoresThreads"

	// Spread vCPUs across sockets and cores
	SpreadAcrossSocketsCores SpreadAcross = "SocketsCores"

	// Spread vCPUs across cores and threads
	SpreadAcrossCoresThreads SpreadAcross = "CoresThreads"
)

type SpreadOptions struct {
	// Across optionally defines how to spread vCPUs across the guest visible topology.
	// Default: SocketsCores
	Across *SpreadAcross

	// Ratio optionally defines the ratio to spread vCPUs across the guest visible topology:
	//
	// CoresThreads        - 1:2   - Controls the ratio of cores to threads. Only a ratio of 2 is currently accepted.
	// SocketsCores        - 1:N   - Controls the ratio of socket to cores.
	// SocketsCoresThreads - 1:N:2 - Controls the ratio of socket to cores. Each core providing 2 threads.
	//
	// Default: 2
	Ratio *uint32
}

// DevicePreferences contains various optional Device preferences.
type DevicePreferences struct {
	// PreferredAutoattachGraphicsDevice optionally defines the preferred value of AutoattachGraphicsDevice
	PreferredAutoattachGraphicsDevice *bool

	// PreferredAutoattachMemBalloon optionally defines the preferred value of AutoattachMemBalloon
	PreferredAutoattachMemBalloon *bool

	// PreferredAutoattachPodInterface optionally defines the preferred value of AutoattachPodInterface
	PreferredAutoattachPodInterface *bool

	// PreferredAutoattachSerialConsole optionally defines the preferred value of AutoattachSerialConsole
	PreferredAutoattachSerialConsole *bool

	// PreferredAutoattachInputDevice optionally defines the preferred value of AutoattachInputDevice
	PreferredAutoattachInputDevice *bool

	// PreferredDisableHotplug optionally defines the preferred value of DisableHotplug
	PreferredDisableHotplug *bool

	// PreferredVirtualGPUOptions optionally defines the preferred value of VirtualGPUOptions
	PreferredVirtualGPUOptions *v1.VGPUOptions

	// PreferredSoundModel optionally defines the preferred model for Sound devices.
	PreferredSoundModel string

	// PreferredUseVirtioTransitional optionally defines the preferred value of UseVirtioTransitional
	PreferredUseVirtioTransitional *bool

	// PreferredInputBus optionally defines the preferred bus for Input devices.
	PreferredInputBus v1.InputBus

	// PreferredInputType optionally defines the preferred type for Input devices.
	PreferredInputType v1.InputType

	// PreferredDiskBus optionally defines the preferred bus for Disk Disk devices.
	PreferredDiskBus v1.DiskBus

	// PreferredLunBus optionally defines the preferred bus for Lun Disk devices.
	PreferredLunBus v1.DiskBus

	// PreferredCdromBus optionally defines the preferred bus for Cdrom Disk devices.
	PreferredCdromBus v1.DiskBus

	// PreferredDedicatedIoThread optionally enables dedicated IO threads for Disk devices using the virtio bus.
	PreferredDiskDedicatedIoThread *bool

	// PreferredCache optionally defines the DriverCache to be used by Disk devices.
	PreferredDiskCache v1.DriverCache

	// PreferredIo optionally defines the QEMU disk IO mode to be used by Disk devices.
	PreferredDiskIO v1.DriverIO

	// PreferredBlockSize optionally defines the block size of Disk devices.
	PreferredDiskBlockSize *v1.BlockSize

	// PreferredInterfaceModel optionally defines the preferred model to be used by Interface devices.
	PreferredInterfaceModel string

	// PreferredRng optionally defines the preferred rng device to be used.
	PreferredRng *v1.Rng

	// PreferredBlockMultiQueue optionally enables the vhost multiqueue feature for virtio disks.
	PreferredBlockMultiQueue *bool

	// PreferredNetworkInterfaceMultiQueue optionally enables the vhost multiqueue feature for virtio interfaces.
	PreferredNetworkInterfaceMultiQueue *bool

	// PreferredTPM optionally defines the preferred TPM device to be used.
	PreferredTPM *v1.TPMDevice

	// PreferredInterfaceMasquerade optionally defines the preferred masquerade configuration to use with each network interface.
	PreferredInterfaceMasquerade *v1.InterfaceMasquerade

	// PreferredPanicDeviceModel optionally defines the preferred panic device model to use with panic devices.
	PreferredPanicDeviceModel *v1.PanicDeviceModel
}

// FeaturePreferences contains various optional defaults for Features.
type FeaturePreferences struct {
	// PreferredAcpi optionally enables the ACPI feature
	PreferredAcpi *v1.FeatureState

	// PreferredApic optionally enables and configures the APIC feature
	PreferredApic *v1.FeatureAPIC

	// PreferredHyperv optionally enables and configures HyperV features
	PreferredHyperv *v1.FeatureHyperv

	// PreferredKvm optionally enables and configures KVM features
	PreferredKvm *v1.FeatureKVM

	// PreferredPvspinlock optionally enables the Pvspinlock feature
	PreferredPvspinlock *v1.FeatureState

	// PreferredSmm optionally enables the SMM feature
	PreferredSmm *v1.FeatureState
}

// FirmwarePreferences contains various optional defaults for Firmware.
type FirmwarePreferences struct {
	// PreferredUseBios optionally enables BIOS
	PreferredUseBios *bool

	// PreferredUseBiosSerial optionally transmitts BIOS output over the serial.
	//
	// Requires PreferredUseBios to be enabled.
	PreferredUseBiosSerial *bool

	// PreferredUseEfi optionally enables EFI
	//
	// Deprecated: Will be removed with v1beta2 or v1
	DeprecatedPreferredUseEfi *bool

	// PreferredUseSecureBoot optionally enables SecureBoot and the OVMF roms will be swapped for SecureBoot-enabled ones.
	//
	// Requires PreferredUseEfi and PreferredSmm to be enabled.
	//
	// Deprecated: Will be removed with v1beta2 or v1
	DeprecatedPreferredUseSecureBoot *bool

	// PreferredEfi optionally enables EFI
	PreferredEfi *v1.EFI
}

// MachinePreferences contains various optional defaults for Machine.
type MachinePreferences struct {
	// PreferredMachineType optionally defines the preferred machine type to use.
	PreferredMachineType string
}

// ClockPreferences contains various optional defaults for Clock.
type ClockPreferences struct {
	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock.
	PreferredClockOffset *v1.ClockOffset

	// Timer specifies whih timers are attached to the vmi.
	PreferredTimer *v1.Timer
}

type PreferenceRequirements struct {
	// Required CPU related attributes of the instancetype.
	CPU *CPUPreferenceRequirement

	// Required Memory related attributes of the instancetype.
	Memory *MemoryPreferenceRequirement

	// Required Architecture of the VM referencing this preference
	Architecture *string
}

type CPUPreferenceRequirement struct {
	// Minimal number of vCPUs required by the preference.
	Guest uint32
}

type MemoryPreferenceRequirement struct {
	// Minimal amount of memory required by the preference.
	Guest resource.Quantity
}
