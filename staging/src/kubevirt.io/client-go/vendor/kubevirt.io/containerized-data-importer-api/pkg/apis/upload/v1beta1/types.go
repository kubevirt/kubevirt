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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UploadTokenRequest is the CR used to initiate a CDI upload
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UploadTokenRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec contains the parameters of the request
	Spec UploadTokenRequestSpec `json:"spec"`

	// Status contains the status of the request
	Status UploadTokenRequestStatus `json:"status"`
}

// UploadTokenRequestSpec defines the parameters of the token request
type UploadTokenRequestSpec struct {
	// PvcName is the name of the PVC to upload to
	PvcName string `json:"pvcName"`
}

// UploadTokenRequestStatus stores the status of a token request
type UploadTokenRequestStatus struct {
	// Token is a JWT token to be inserted in "Authentication Bearer header"
	Token string `json:"token,omitempty"`
}

// UploadTokenRequestList contains a list of UploadTokenRequests
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UploadTokenRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains a list of UploadTokenRequests
	Items []UploadTokenRequest `json:"items"`
}
