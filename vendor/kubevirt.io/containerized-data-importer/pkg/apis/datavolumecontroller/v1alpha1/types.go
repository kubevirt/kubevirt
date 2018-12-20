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

//go:generate swagger-doc
//go:generate deepcopy-gen -i . --go-header-file ../../../../hack/custom-boilerplate.go.txt
//go:generate openapi-gen -i . --output-package=kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1  --go-header-file ../../../../hack/custom-boilerplate.go.txt

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataVolume provides a representation of our data volume
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataVolumeSpec   `json:"spec"`
	Status DataVolumeStatus `json:"status"`
}

// DataVolumeSpec defines our specification for a DataVolume type
type DataVolumeSpec struct {
	//Source is the src of the data for the requested DataVolume
	Source DataVolumeSource `json:"source"`
	//PVC is a pointer to the PVC Spec we want to use
	PVC *corev1.PersistentVolumeClaimSpec `json:"pvc"`
}

// DataVolumeSource represents the source for our Data Volume, this can be HTTP, S3, Registry or an existing PVC
type DataVolumeSource struct {
	HTTP     *DataVolumeSourceHTTP     `json:"http,omitempty"`
	S3       *DataVolumeSourceS3       `json:"s3,omitempty"`
	Registry *DataVolumeSourceRegistry `json:"registry,omitempty"`
	PVC      *DataVolumeSourcePVC      `json:"pvc,omitempty"`
	Upload   *DataVolumeSourceUpload   `json:"upload,omitempty"`
	Blank    *DataVolumeBlankImage     `json:"blank,omitempty"`
}

// DataVolumeSourcePVC provides the parameters to create a Data Volume from an existing PVC
type DataVolumeSourcePVC struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// DataVolumeBlankImage provides the parameters to create a new raw blank image for the PVC
type DataVolumeBlankImage struct{}

// DataVolumeSourceUpload provides the parameters to create a Data Volume by uploading the source
type DataVolumeSourceUpload struct {
	//Target string `json:"shouldUpload,omitempty"`
}

// DataVolumeSourceS3 provides the parameters to create a Data Volume from an S3 source
type DataVolumeSourceS3 struct {
	//URL is the url of the S3 source
	URL string `json:"url,omitempty"`
	//SecretRef provides the secret reference needed to access the S3 source
	SecretRef string `json:"secretRef,omitempty"`
}

// DataVolumeSourceRegistry provides the parameters to create a Data Volume from an registry source
type DataVolumeSourceRegistry struct {
	//URL is the url of the Registry source
	URL string `json:"url,omitempty"`
	//SecretRef provides the secret reference needed to access the Registry source
	SecretRef string `json:"secretRef,omitempty"`
}

// DataVolumeSourceHTTP provides the parameters to create a Data Volume from an HTTP source
type DataVolumeSourceHTTP struct {
	//URL is the URL of the http source
	URL string `json:"url,omitempty"`
	//SecretRef provides the secret reference needed to access the HTTP source
	SecretRef string `json:"secretRef,omitempty"`
}

// DataVolumeStatus provides the parameters to store the phase of the Data Volume
type DataVolumeStatus struct {
	//Phase is the current phase of the data volume
	Phase DataVolumePhase `json:"phase,omitempty"`
}

//DataVolumeList provides the needed parameters to do request a list of Data Volumes from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of DataVolumes
	Items []DataVolume `json:"items"`
}

// DataVolumePhase is the current phase of the DataVolume
type DataVolumePhase string

const (
	// PhaseUnset represents a data volume with no current phase
	PhaseUnset DataVolumePhase = ""

	// Pending represents a data volume with a current phase of Pending
	Pending DataVolumePhase = "Pending"
	// PVCBound represents a data volume with a current phase of PVCBound
	PVCBound DataVolumePhase = "PVCBound"

	// ImportScheduled represents a data volume with a current phase of ImportScheduled
	ImportScheduled DataVolumePhase = "ImportScheduled"

	// ImportInProgress represents a data volume with a current phase of ImportInProgress
	ImportInProgress DataVolumePhase = "ImportInProgress"

	// CloneScheduled represents a data volume with a current phase of CloneScheduled
	CloneScheduled DataVolumePhase = "CloneScheduled"

	// CloneInProgress represents a data volume with a current phase of CloneInProgress
	CloneInProgress DataVolumePhase = "CloneInProgress"

	// UploadScheduled represents a data volume with a current phase of UploadScheduled
	UploadScheduled DataVolumePhase = "UploadScheduled"

	// UploadReady represents a data volume with a current phase of UploadReady
	UploadReady DataVolumePhase = "UploadReady"

	// Succeeded represents a DataVolumePhase of Succeeded
	Succeeded DataVolumePhase = "Succeeded"
	// Failed represents a DataVolumePhase of Failed
	Failed DataVolumePhase = "Failed"
	// Unknown represents a DataVolumePhase of Unknown
	Unknown DataVolumePhase = "Unknown"
)
