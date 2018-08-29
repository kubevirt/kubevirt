/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataVolumeSpec   `json:"spec"`
	Status DataVolumeStatus `json:"status"`
}

type DataVolumeSpec struct {
	Source DataVolumeSource                  `json:"source"`
	PVC    *corev1.PersistentVolumeClaimSpec `json:"pvc"`
}

type DataVolumeSource struct {
	HTTP *DataVolumeSourceHTTP `json:"http,omitempty"`
	S3   *DataVolumeSourceS3   `json:"s3,omitempty"`
}

type DataVolumeSourceS3 struct {
	URL       string `json:"url,omitempty"`
	SecretRef string `json:"secretRef,omitempty"`
}

type DataVolumeSourceHTTP struct {
	URL string `json:"url,omitempty"`
}

type DataVolumeStatus struct {
	Phase DataVolumePhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []DataVolume `json:"items"`
}

type DataVolumePhase string

const (
	PhaseUnset DataVolumePhase = ""

	Pending  DataVolumePhase = "Pending"
	PVCBound DataVolumePhase = "PVCBound"

	ImportScheduled DataVolumePhase = "ImportScheduled"

	ImportInProgress DataVolumePhase = "ImportInProgress"

	Succeeded DataVolumePhase = "Succeeded"
	Failed    DataVolumePhase = "Failed"
	Unknown   DataVolumePhase = "Unknown"
)
