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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

// VirtualMachineInstancetype resource contains quantitative and resource related VirtualMachine configuration
// that can be used by multiple VirtualMachine resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineInstancetype struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Required spec describing the instancetype
	Spec VirtualMachineInstancetypeSpec `json:"spec"`
}

// VirtualMachineInstancetypeList is a list of VirtualMachineInstancetype resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineInstancetypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineInstancetype `json:"items"`
}

// VirtualMachineClusterInstancetype is a cluster scoped version of VirtualMachineInstancetype resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
type VirtualMachineClusterInstancetype struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Required spec describing the instancetype
	Spec VirtualMachineInstancetypeSpec `json:"spec"`
}

// VirtualMachineClusterInstancetypeList is a list of VirtualMachineClusterInstancetype resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterInstancetypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineClusterInstancetype `json:"items"`
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
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the VMI will be dispatched by specified scheduler.
	// If not specified, the VMI will be dispatched by default scheduler.
	//
	// SchedulerName is the name of the custom K8s scheduler for the instancetype.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty"`

	// Required CPU related attributes of the instancetype.
	CPU CPUInstancetype `json:"cpu"`

	// Required Memory related attributes of the instancetype.
	Memory MemoryInstancetype `json:"memory"`

	// Optionally defines any GPU devices associated with the instancetype.
	//
	// +optional
	// +listType=atomic
	GPUs []v1.GPU `json:"gpus,omitempty"`

	// Optionally defines any HostDevices associated with the instancetype.
	//
	// +optional
	// +listType=atomic
	HostDevices []v1.HostDevice `json:"hostDevices,omitempty"`

	// Optionally defines the IOThreadsPolicy to be used by the instancetype.
	//
	// +optional
	IOThreadsPolicy *v1.IOThreadsPolicy `json:"ioThreadsPolicy,omitempty"`

	// Optionally defines the LaunchSecurity to be used by the instancetype.
	//
	// +optional
	LaunchSecurity *v1.LaunchSecurity `json:"launchSecurity,omitempty"`

	// Optionally defines the required Annotations to be used by the instance type and applied to the VirtualMachineInstance
	//
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CPUInstancetype contains the CPU related configuration of a given VirtualMachineInstancetypeSpec.
//
// Guest is a required attribute and defines the number of vCPUs to be exposed to the guest by the instancetype.
type CPUInstancetype struct {

	// Required number of vCPUs to expose to the guest.
	//
	// The resulting CPU topology being derived from the optional PreferredCPUTopology attribute of CPUPreferences that itself defaults to PreferSockets.
	Guest uint32 `json:"guest"`

	// Model specifies the CPU model inside the VMI.
	// List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map.
	// It is possible to specify special cases like "host-passthrough" to get the same CPU as the node
	// and "host-model" to get CPU closest to the node one.
	// Defaults to host-model.
	// +optional
	Model *string `json:"model,omitempty"`

	// DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node
	// with enough dedicated pCPUs and pin the vCPUs to it.
	// +optional
	DedicatedCPUPlacement *bool `json:"dedicatedCPUPlacement,omitempty"`

	// NUMA allows specifying settings for the guest NUMA topology
	// +optional
	NUMA *v1.NUMA `json:"numa,omitempty"`

	// IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place
	// the emulator thread on it.
	// +optional
	IsolateEmulatorThread *bool `json:"isolateEmulatorThread,omitempty"`

	// Realtime instructs the virt-launcher to tune the VMI for lower latency, optional for real time workloads
	// +optional
	Realtime *v1.Realtime `json:"realtime,omitempty"`

	// MaxSockets specifies the maximum amount of sockets that can be hotplugged
	// +optional
	MaxSockets *uint32 `json:"maxSockets,omitempty"`
}

// MemoryInstancetype contains the Memory related configuration of a given VirtualMachineInstancetypeSpec.
//
// Guest is a required attribute and defines the amount of RAM to be exposed to the guest by the instancetype.
type MemoryInstancetype struct {

	// Required amount of memory which is visible inside the guest OS.
	Guest resource.Quantity `json:"guest"`

	// Optionally enables the use of hugepages for the VirtualMachineInstance instead of regular memory.
	// +optional
	Hugepages *v1.Hugepages `json:"hugepages,omitempty"`
	// OvercommitPercent is the percentage of the guest memory which will be overcommitted.
	// This means that the VMIs parent pod (virt-launcher) will request less
	// physical memory by a factor specified by the OvercommitPercent.
	// Overcommits can lead to memory exhaustion, which in turn can lead to crashes. Use carefully.
	// Defaults to 0
	// +optional
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=0
	OvercommitPercent int `json:"overcommitPercent,omitempty"`

	// MaxGuest allows to specify the maximum amount of memory which is visible inside the Guest OS.
	// The delta between MaxGuest and Guest is the amount of memory that can be hot(un)plugged.
	// +optional
	MaxGuest *resource.Quantity `json:"maxGuest,omitempty"`
}

// VirtualMachinePreference resource contains optional preferences related to the VirtualMachine.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachinePreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Required spec describing the preferences
	Spec VirtualMachinePreferenceSpec `json:"spec"`
}

// VirtualMachinePreferenceList is a list of VirtualMachinePreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachinePreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []VirtualMachinePreference `json:"items"`
}

// VirtualMachineClusterPreference is a cluster scoped version of the VirtualMachinePreference resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
type VirtualMachineClusterPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Required spec describing the preferences
	Spec VirtualMachinePreferenceSpec `json:"spec"`
}

// VirtualMachineClusterPreferenceList is a list of VirtualMachineClusterPreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineClusterPreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []VirtualMachineClusterPreference `json:"items"`
}

// VirtualMachinePreferenceSpec is a description of the VirtualMachinePreference or VirtualMachineClusterPreference.
type VirtualMachinePreferenceSpec struct {

	// Clock optionally defines preferences associated with the Clock attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	Clock *ClockPreferences `json:"clock,omitempty"`

	// CPU optionally defines preferences associated with the CPU attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	CPU *CPUPreferences `json:"cpu,omitempty"`

	// Devices optionally defines preferences associated with the Devices attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	Devices *DevicePreferences `json:"devices,omitempty"`

	// Features optionally defines preferences associated with the Features attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	Features *FeaturePreferences `json:"features,omitempty"`

	// Firmware optionally defines preferences associated with the Firmware attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	Firmware *FirmwarePreferences `json:"firmware,omitempty"`

	// Machine optionally defines preferences associated with the Machine attribute of a VirtualMachineInstance DomainSpec
	//
	//+optional
	Machine *MachinePreferences `json:"machine,omitempty"`

	// Volumes optionally defines preferences associated with the Volumes attribute of a VirtualMachineInstace DomainSpec
	//
	//+optional
	Volumes *VolumePreferences `json:"volumes,omitempty"`

	// Subdomain of the VirtualMachineInstance
	//
	//+optional
	PreferredSubdomain *string `json:"preferredSubdomain,omitempty"`

	// Grace period observed after signalling a VirtualMachineInstance to stop after which the VirtualMachineInstance is force terminated.
	//
	//+optional
	PreferredTerminationGracePeriodSeconds *int64 `json:"preferredTerminationGracePeriodSeconds,omitempty"`

	// Requirements defines the minium amount of instance type defined resources required by a set of preferences
	//
	//+optional
	Requirements *PreferenceRequirements `json:"requirements,omitempty"`

	// Optionally defines preferred Annotations to be applied to the VirtualMachineInstance
	//
	//+optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// PreferSpreadSocketToCoreRatio defines the ratio to spread vCPUs between cores and sockets, it defaults to 2.
	//
	//+optional
	PreferSpreadSocketToCoreRatio uint32 `json:"preferSpreadSocketToCoreRatio,omitempty"`
}

type VolumePreferences struct {

	// PreffereedStorageClassName optionally defines the preferred storageClass
	//
	//+optional
	PreferredStorageClassName string `json:"preferredStorageClassName,omitempty"`
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
	//
	//+optional
	PreferredCPUTopology *PreferredCPUTopology `json:"preferredCPUTopology,omitempty"`

	//
	//+optional
	SpreadOptions *SpreadOptions `json:"spreadOptions,omitempty"`

	// PreferredCPUFeatures optionally defines a slice of preferred CPU features.
	//
	//+optional
	PreferredCPUFeatures []v1.CPUFeature `json:"preferredCPUFeatures,omitempty"`
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
	//
	//+optional
	Across *SpreadAcross `json:"across,omitempty"`

	// Ratio optionally defines the ratio to spread vCPUs across the guest visible topology:
	//
	// CoresThreads        - 1:2   - Controls the ratio of cores to threads. Only a ratio of 2 is currently accepted.
	// SocketsCores        - 1:N   - Controls the ratio of socket to cores.
	// SocketsCoresThreads - 1:N:2 - Controls the ratio of socket to cores. Each core providing 2 threads.
	//
	// Default: 2
	//
	//+optional
	Ratio *uint32 `json:"ratio,omitempty"`
}

// DevicePreferences contains various optional Device preferences.
type DevicePreferences struct {

	// PreferredAutoattachGraphicsDevice optionally defines the preferred value of AutoattachGraphicsDevice
	//
	// +optional
	PreferredAutoattachGraphicsDevice *bool `json:"preferredAutoattachGraphicsDevice,omitempty"`

	// PreferredAutoattachMemBalloon optionally defines the preferred value of AutoattachMemBalloon
	//
	// +optional
	PreferredAutoattachMemBalloon *bool `json:"preferredAutoattachMemBalloon,omitempty"`

	// PreferredAutoattachPodInterface optionally defines the preferred value of AutoattachPodInterface
	//
	// +optional
	PreferredAutoattachPodInterface *bool `json:"preferredAutoattachPodInterface,omitempty"`

	// PreferredAutoattachSerialConsole optionally defines the preferred value of AutoattachSerialConsole
	//
	// +optional
	PreferredAutoattachSerialConsole *bool `json:"preferredAutoattachSerialConsole,omitempty"`

	// PreferredAutoattachInputDevice optionally defines the preferred value of AutoattachInputDevice
	//
	// +optional
	PreferredAutoattachInputDevice *bool `json:"preferredAutoattachInputDevice,omitempty"`

	// PreferredDisableHotplug optionally defines the preferred value of DisableHotplug
	//
	// +optional
	PreferredDisableHotplug *bool `json:"preferredDisableHotplug,omitempty"`

	// PreferredVirtualGPUOptions optionally defines the preferred value of VirtualGPUOptions
	//
	// +optional
	PreferredVirtualGPUOptions *v1.VGPUOptions `json:"preferredVirtualGPUOptions,omitempty"`

	// PreferredSoundModel optionally defines the preferred model for Sound devices.
	//
	// +optional
	PreferredSoundModel string `json:"preferredSoundModel,omitempty"`

	// PreferredUseVirtioTransitional optionally defines the preferred value of UseVirtioTransitional
	//
	// +optional
	PreferredUseVirtioTransitional *bool `json:"preferredUseVirtioTransitional,omitempty"`

	// PreferredInputBus optionally defines the preferred bus for Input devices.
	//
	// +optional
	PreferredInputBus v1.InputBus `json:"preferredInputBus,omitempty"`

	// PreferredInputType optionally defines the preferred type for Input devices.
	//
	// +optional
	PreferredInputType v1.InputType `json:"preferredInputType,omitempty"`

	// PreferredDiskBus optionally defines the preferred bus for Disk Disk devices.
	//
	// +optional
	PreferredDiskBus v1.DiskBus `json:"preferredDiskBus,omitempty"`

	// PreferredLunBus optionally defines the preferred bus for Lun Disk devices.
	//
	// +optional
	PreferredLunBus v1.DiskBus `json:"preferredLunBus,omitempty"`

	// PreferredCdromBus optionally defines the preferred bus for Cdrom Disk devices.
	//
	// +optional
	PreferredCdromBus v1.DiskBus `json:"preferredCdromBus,omitempty"`

	// PreferredDedicatedIoThread optionally enables dedicated IO threads for Disk devices using the virtio bus.
	//
	// +optional
	PreferredDiskDedicatedIoThread *bool `json:"preferredDiskDedicatedIoThread,omitempty"`

	// PreferredCache optionally defines the DriverCache to be used by Disk devices.
	//
	// +optional
	PreferredDiskCache v1.DriverCache `json:"preferredDiskCache,omitempty"`

	// PreferredIo optionally defines the QEMU disk IO mode to be used by Disk devices.
	//
	// +optional
	PreferredDiskIO v1.DriverIO `json:"preferredDiskIO,omitempty"`

	// PreferredBlockSize optionally defines the block size of Disk devices.
	//
	// +optional
	PreferredDiskBlockSize *v1.BlockSize `json:"preferredDiskBlockSize,omitempty"`

	// PreferredInterfaceModel optionally defines the preferred model to be used by Interface devices.
	//
	// +optional
	PreferredInterfaceModel string `json:"preferredInterfaceModel,omitempty"`

	// PreferredRng optionally defines the preferred rng device to be used.
	//
	// +optional
	PreferredRng *v1.Rng `json:"preferredRng,omitempty"`

	// PreferredBlockMultiQueue optionally enables the vhost multiqueue feature for virtio disks.
	//
	// +optional
	PreferredBlockMultiQueue *bool `json:"preferredBlockMultiQueue,omitempty"`

	// PreferredNetworkInterfaceMultiQueue optionally enables the vhost multiqueue feature for virtio interfaces.
	//
	// +optional
	PreferredNetworkInterfaceMultiQueue *bool `json:"preferredNetworkInterfaceMultiQueue,omitempty"`

	// PreferredTPM optionally defines the preferred TPM device to be used.
	//
	// +optional
	PreferredTPM *v1.TPMDevice `json:"preferredTPM,omitempty"`

	// PreferredInterfaceMasquerade optionally defines the preferred masquerade configuration to use with each network interface.
	//
	// +optional
	PreferredInterfaceMasquerade *v1.InterfaceMasquerade `json:"preferredInterfaceMasquerade,omitempty"`
}

// FeaturePreferences contains various optional defaults for Features.
type FeaturePreferences struct {

	// PreferredAcpi optionally enables the ACPI feature
	//
	// +optional
	PreferredAcpi *v1.FeatureState `json:"preferredAcpi,omitempty"`

	// PreferredApic optionally enables and configures the APIC feature
	//
	// +optional
	PreferredApic *v1.FeatureAPIC `json:"preferredApic,omitempty"`

	// PreferredHyperv optionally enables and configures HyperV features
	//
	// +optional
	PreferredHyperv *v1.FeatureHyperv `json:"preferredHyperv,omitempty"`

	// PreferredKvm optionally enables and configures KVM features
	//
	// +optional
	PreferredKvm *v1.FeatureKVM `json:"preferredKvm,omitempty"`

	// PreferredPvspinlock optionally enables the Pvspinlock feature
	//
	// +optional
	PreferredPvspinlock *v1.FeatureState `json:"preferredPvspinlock,omitempty"`

	// PreferredSmm optionally enables the SMM feature
	//
	// +optional
	PreferredSmm *v1.FeatureState `json:"preferredSmm,omitempty"`
}

// FirmwarePreferences contains various optional defaults for Firmware.
type FirmwarePreferences struct {

	// PreferredUseBios optionally enables BIOS
	//
	// +optional
	PreferredUseBios *bool `json:"preferredUseBios,omitempty"`

	// PreferredUseBiosSerial optionally transmitts BIOS output over the serial.
	//
	// Requires PreferredUseBios to be enabled.
	//
	// +optional
	PreferredUseBiosSerial *bool `json:"preferredUseBiosSerial,omitempty"`

	// PreferredUseEfi optionally enables EFI
	//
	// +optional
	PreferredUseEfi *bool `json:"preferredUseEfi,omitempty"`

	// PreferredUseSecureBoot optionally enables SecureBoot and the OVMF roms will be swapped for SecureBoot-enabled ones.
	//
	// Requires PreferredUseEfi and PreferredSmm to be enabled.
	//
	// +optional
	PreferredUseSecureBoot *bool `json:"preferredUseSecureBoot,omitempty"`
}

// MachinePreferences contains various optional defaults for Machine.
type MachinePreferences struct {

	// PreferredMachineType optionally defines the preferred machine type to use.
	//
	// +optional
	PreferredMachineType string `json:"preferredMachineType,omitempty"`
}

// ClockPreferences contains various optional defaults for Clock.
type ClockPreferences struct {

	// ClockOffset allows specifying the UTC offset or the timezone of the guest clock.
	//
	// +optional
	PreferredClockOffset *v1.ClockOffset `json:"preferredClockOffset,omitempty"`

	// Timer specifies whih timers are attached to the vmi.
	//
	// +optional
	PreferredTimer *v1.Timer `json:"preferredTimer,omitempty"`
}

type PreferenceRequirements struct {

	// Required CPU related attributes of the instancetype.
	//
	//+optional
	CPU *CPUPreferenceRequirement `json:"cpu,omitempty"`

	// Required Memory related attributes of the instancetype.
	//
	//+optional
	Memory *MemoryPreferenceRequirement `json:"memory,omitempty"`
}

type CPUPreferenceRequirement struct {

	// Minimal number of vCPUs required by the preference.
	Guest uint32 `json:"guest"`
}

type MemoryPreferenceRequirement struct {

	// Minimal amount of memory required by the preference.
	Guest resource.Quantity `json:"guest"`
}
