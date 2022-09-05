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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	App = "virt-exporter"
)

// VirtualMachineExport defines the operation of exporting a VM source
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineExport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineExportSpec `json:"spec"`

	// +optional
	Status *VirtualMachineExportStatus `json:"status,omitempty"`
}

// VirtualMachineExportList is a list of VirtualMachineExport resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineExportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// +listType=atomic
	Items []VirtualMachineExport `json:"items"`
}

// VirtualMachineExportSpec is the spec for a VirtualMachineExport resource
type VirtualMachineExportSpec struct {
	Source corev1.TypedLocalObjectReference `json:"source"`

	// TokenSecretRef is the name of the secret that contains the token used by the export server pod
	TokenSecretRef string `json:"tokenSecretRef"`
}

// VirtualMachineExportPhase is the current phase of the VirtualMachineExport
type VirtualMachineExportPhase string

const (
	// Pending means the Virtual Machine export is pending
	Pending VirtualMachineExportPhase = "Pending"
	// Ready means the Virtual Machine export is ready
	Ready VirtualMachineExportPhase = "Ready"
	// Terminated means the Virtual Machine export is terminated and no longer available
	Terminated VirtualMachineExportPhase = "Terminated"
	// Skipped means the export is invalid in a way so the exporter pod cannot start, and we are skipping creating the exporter server pod.
	Skipped VirtualMachineExportPhase = "Skipped"
)

// VirtualMachineExportStatus is the status for a VirtualMachineExport resource
type VirtualMachineExportStatus struct {
	// +optional
	Phase VirtualMachineExportPhase `json:"phase,omitempty"`

	// +optional
	Links *VirtualMachineExportLinks `json:"links,omitempty"`

	// +optional
	// ServiceName is the name of the service created associated with the Virtual Machine export. It will be used to
	// create the internal URLs for downloading the images
	ServiceName string `json:"serviceName,omitempty"`

	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
}

// VirtualMachineExportLinks contains the links that point the exported VM resources
type VirtualMachineExportLinks struct {
	// +optional
	Internal *VirtualMachineExportLink `json:"internal,omitempty"`
	// +optional
	External *VirtualMachineExportLink `json:"external,omitempty"`
}

// VirtualMachineExportLink contains a list of volumes available for export, as well as the URLs to obtain these volumes
type VirtualMachineExportLink struct {
	// Cert is the public CA certificate base64 encoded
	Cert string `json:"cert"`

	// Volumes is a list of available volumes to export
	// +listType=map
	// +listMapKey=name
	// +optional
	Volumes []VirtualMachineExportVolume `json:"volumes"`
}

// VirtualMachineExportVolume contains the name and available formats for the exported volume
type VirtualMachineExportVolume struct {
	// Name is the name of the exported volume
	Name string `json:"name"`
	// +listType=map
	// +listMapKey=format
	// +optional
	Formats []VirtualMachineExportVolumeFormat `json:"formats,omitempty"`
}

type ExportVolumeFormat string

const (
	// KubeVirtRaw is the volume in RAW format
	KubeVirtRaw ExportVolumeFormat = "raw"
	// KubeVirtGZ is the volume in gzipped RAW format.
	KubeVirtGz ExportVolumeFormat = "gzip"
	// Dir is an uncompressed directory, which points to the root of a PersistentVolumeClaim, exposed using a FileServer https://pkg.go.dev/net/http#FileServer
	Dir ExportVolumeFormat = "dir"
	// ArchiveGz is a tarred and gzipped version of the root of a PersistentVolumeClaim
	ArchiveGz ExportVolumeFormat = "tar.gz"
)

// VirtualMachineExportVolumeFormat contains the format type and URL to get the volume in that format
type VirtualMachineExportVolumeFormat struct {
	// Format is the format of the image at the specified URL
	Format ExportVolumeFormat `json:"format"`
	// Url is the url that contains the volume in the format specified
	Url string `json:"url"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionReady is the "ready" condition type
	ConditionReady ConditionType = "Ready"
	// ConditionPVC is the condition of the PVC we are exporting
	ConditionPVC ConditionType = "PVCReady"
	// ConditionVolumesCreated is the condition to see if volumes are created from volume snapshots
	ConditionVolumesCreated ConditionType = "VolumesCreated"
)

// Condition defines conditions
type Condition struct {
	Type ConditionType `json:"type"`

	Status corev1.ConditionStatus `json:"status"`

	// +optional
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`

	// +optional
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}
