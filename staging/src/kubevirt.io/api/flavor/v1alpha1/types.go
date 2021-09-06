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

	// +listType=map
	// +listMapKey=name
	Profiles []VirtualMachineFlavorProfile `json:"profiles"`
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

	// +listType=map
	// +listMapKey=name
	Profiles []VirtualMachineFlavorProfile `json:"profiles"`
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

// VirtualMachineFlavorProfile contains definitions that will be applied to VirtualMachine.
//
// +k8s:openapi-gen=true
type VirtualMachineFlavorProfile struct {
	// Name specifies the name of this custom profile.
	Name string `json:"name"`

	// Default specifies if this VirtualMachineFlavorProfile is the default for the VirtualMachineFlavor.
	// Zero or one profile can be set to default.
	//
	// +optional
	Default bool `json:"default,omitempty"`

	// DomainTemplate specifies domain that will be used to fill missing values in a VMI domain.
	// Devices filed is not allowed in DomainTemplate.
	//
	// +optional
	DomainTemplate *VirtualMachineFlavorDomainTemplateSpec `json:"domainTemplate,omitempty"`
}

// VirtualMachineFlavorDomainTemplateSpec contains the generic spec definition for the flavor.
// Note that resources and devices are optional unlike within a full DomainSpec.
//
// +k8s:openapi-gen=true
type VirtualMachineFlavorDomainTemplateSpec struct {
	// Resources describes the Compute Resources required by this vmi.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// CPU allow specified the detailed CPU topology inside the vmi.
	// +optional
	CPU *v1.CPU `json:"cpu,omitempty"`
	// Memory allow specifying the VMI memory features.
	// +optional
	Memory *v1.Memory `json:"memory,omitempty"`
	// Machine type.
	// +optional
	Machine *v1.Machine `json:"machine,omitempty"`
	// Firmware.
	// +optional
	Firmware *v1.Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *v1.Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv, smm.
	// +optional
	Features *v1.Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, and others
	// +optional
	Devices v1.Devices `json:"devices,omitempty"`
	// Controls whether or not disks will share IOThreads.
	// Omitting IOThreadsPolicy disables use of IOThreads.
	// One of: shared, auto
	// +optional
	IOThreadsPolicy *v1.IOThreadsPolicy `json:"ioThreadsPolicy,omitempty"`
	// Chassis specifies the chassis info passed to the domain.
	// +optional
	Chassis *v1.Chassis `json:"chassis,omitempty"`
	// Launch Security setting of the vmi.
	// +optional
	LaunchSecurity *v1.LaunchSecurity `json:"launchSecurity,omitempty"`
}
