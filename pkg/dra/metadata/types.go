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

package metadata

import (
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceMetadata contains metadata about devices allocated to a ResourceClaim.
// It is serialized to versioned JSON files that can be mounted into containers.
// These types mirror the KEP-5304 v1alpha1 API in k8s.io/dynamic-resource-allocation.
type DeviceMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// PodClaimName is only present for template-generated claims.
	// For pre-existing claims (resourceClaimName), this field is absent.
	PodClaimName *string                 `json:"podClaimName,omitempty"`
	Requests     []DeviceMetadataRequest `json:"requests,omitempty"`
}

// DeviceMetadataRequest contains metadata for a single request within a ResourceClaim.
type DeviceMetadataRequest struct {
	Name    string   `json:"name"`
	Devices []Device `json:"devices,omitempty"`
}

// Device contains metadata about a single allocated device.
type Device struct {
	Driver      string                                                  `json:"driver"`
	Pool        string                                                  `json:"pool"`
	Name        string                                                  `json:"name"`
	Attributes  map[resourcev1.QualifiedName]resourcev1.DeviceAttribute `json:"attributes,omitempty"`
	NetworkData *resourcev1.NetworkDeviceData                           `json:"networkData,omitempty"`
}

// Well-known attribute keys for device identification
const (
	// PCIBusIDAttribute is the standard attribute for PCI device address (passthrough GPUs)
	PCIBusIDAttribute = resourcev1.QualifiedName("resource.kubernetes.io/pciBusID")
	// MDevUUIDAttribute is the attribute for mediated device UUID (vGPUs)
	// Note: This is not yet standardized under resource.kubernetes.io
	MDevUUIDAttribute = resourcev1.QualifiedName("mdevUUID")
)
