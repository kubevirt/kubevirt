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
