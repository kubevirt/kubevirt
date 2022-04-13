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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
)

// VirtualMachineFlavor resource contains common VirtualMachine configuration
// that can be used by multiple VirtualMachine resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachineFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// VirtualMachineFlavorSpec for the flavor
	Spec VirtualMachineFlavorSpec `json:"spec"`
}

// VirtualMachineFlavorList is a list of VirtualMachineFlavor resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineFlavor `json:"items"`
}

// VirtualMachineClusterFlavor is a cluster scoped version of VirtualMachineFlavor resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type VirtualMachineClusterFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// VirtualMachineFlavorSpec for the flavor
	Spec VirtualMachineFlavorSpec `json:"spec"`
}

// VirtualMachineClusterFlavorList is a list of VirtualMachineClusterFlavor resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineClusterFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineClusterFlavor `json:"items"`
}

// VirtualMachineFlavorSpec
//
// +k8s:openapi-gen=true
type VirtualMachineFlavorSpec struct {
	CPU CPUFlavor `json:"cpu"`

	Memory MemoryFlavor `json:"memory"`
}

// CPUFlavor
//
// +k8s:openapi-gen=true
type CPUFlavor struct {

	// Number of vCPUs to expose to the guest.
	// The resulting CPU topology being derived from the optional PreferredCPUTopology attribute of CPUPreferences.
	Guest uint32 `json:"guest"`

	// Model specifies the CPU model inside the VMI.
	// List of available models https://github.com/libvirt/libvirt/tree/master/src/cpu_map.
	// It is possible to specify special cases like "host-passthrough" to get the same CPU as the node
	// and "host-model" to get CPU closest to the node one.
	// Defaults to host-model.
	// +optional
	Model string `json:"model,omitempty"`

	// DedicatedCPUPlacement requests the scheduler to place the VirtualMachineInstance on a node
	// with enough dedicated pCPUs and pin the vCPUs to it.
	// +optional
	DedicatedCPUPlacement bool `json:"dedicatedCPUPlacement,omitempty"`

	// NUMA allows specifying settings for the guest NUMA topology
	// +optional
	NUMA *v1.NUMA `json:"numa,omitempty"`

	// IsolateEmulatorThread requests one more dedicated pCPU to be allocated for the VMI to place
	// the emulator thread on it.
	// +optional
	IsolateEmulatorThread bool `json:"isolateEmulatorThread,omitempty"`

	// Realtime instructs the virt-launcher to tune the VMI for lower latency, optional for real time workloads
	// +optional
	Realtime *v1.Realtime `json:"realtime,omitempty"`
}

// FlavorMemory
//
// +k8s:openapi-gen=true
type MemoryFlavor struct {

	// Guest allows to specifying the amount of memory which is visible inside the Guest OS.
	Guest *resource.Quantity `json:"guest,omitempty"`

	// Hugepages allow to use hugepages for the VirtualMachineInstance instead of regular memory.
	// +optional
	Hugepages *v1.Hugepages `json:"hugepages,omitempty"`
}

// VirtualMachinePreference
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachinePreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachinePreferenceSpec `json:"spec"`
}

// VirtualMachinePreferenceList is a list of VirtualMachinePreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachinePreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []VirtualMachinePreference `json:"items"`
}

// VirtualMachineClusterPreference
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type VirtualMachineClusterPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachinePreferenceSpec `json:"spec"`
}

// VirtualMachineClusterPreferenceList is a list of VirtualMachineClusterPreference resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineClusterPreferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []VirtualMachineClusterPreference `json:"items"`
}

// VirtualMachinePreferenceSpec
//
// +k8s:openapi-gen=true
type VirtualMachinePreferenceSpec struct {

	//+optional
	CPU *CPUPreferences `json:"cpu,omitempty"`

	//+optional
	Devices *DevicePreferences `json:"devices,omitempty"`

	//+optional
	Features *FeaturePreferences `json:"features,omitempty"`

	//+optional
	Firmware *FirmwarePreferences `json:"firmware,omitempty"`

	//+optional
	Machine *MachinePreferences `json:"machine,omitempty"`
}

// +k8s:openapi-gen=true
type PreferredCPUTopology string

const (
	PreferSockets PreferredCPUTopology = "preferSockets"
	PreferCores   PreferredCPUTopology = "preferCores"
	PreferThreads PreferredCPUTopology = "preferThreads"
)

// PreferencesCPU
//
// +k8s:openapi-gen=true
type CPUPreferences struct {

	// Defaults to
	//+optional
	PreferredCPUTopology PreferredCPUTopology `json:"preferredCPUTopology,omitempty"`
}

// DevicePreferences contains various optional defaults for Devices.
//
// +k8s:openapi-gen=true
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
	PreferredInputBus string `json:"preferredInputBus,omitempty"`

	// PreferredInputType optionally defines the preferred type for Input devices.
	//
	// +optional
	PreferredInputType string `json:"preferredInputType,omitempty"`

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

	// PreferredDedicatedIoThread optionally enables dedicated IO threads for Disk devices.
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
}

// FeaturePreferences contains various optional defaults for Features.
//
// +k8s:openapi-gen=true
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
//
// +k8s:openapi-gen=true
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
//
// +k8s:openapi-gen=true
type MachinePreferences struct {

	// PreferredMachineType optionally defines the preferred machine type to use.
	//
	// +optional
	PreferredMachineType string `json:"preferredMachineType,omitempty"`
}
