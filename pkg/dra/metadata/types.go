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

// TODO: The types declared here should go away once KEP-5304 is implemented in kuberenetes.

// DeviceMetadata represents the KEP-5304 metadata file structure
// written by DRA drivers at /var/run/dra-device-attributes/{driver}/{claimName}/metadata.json
type DeviceMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// PodClaimName is only present for template-generated claims
	// For pre-existing claims (resourceClaimName), this field is absent
	PodClaimName string          `json:"podClaimName,omitempty"`
	Requests     []DeviceRequest `json:"requests"`
}

// DeviceRequest represents a single request within the claim
type DeviceRequest struct {
	Name    string   `json:"name"`
	Devices []Device `json:"devices"`
}

// Device represents an allocated device with its attributes
type Device struct {
	Driver     string                                                  `json:"driver"`
	Pool       string                                                  `json:"pool"`
	Device     string                                                  `json:"device"`
	Attributes map[resourcev1.QualifiedName]resourcev1.DeviceAttribute `json:"attributes"`
}

// Well-known attribute keys for device identification
const (
	// PCIBusIDAttribute is the standard attribute for PCI device address (passthrough GPUs)
	PCIBusIDAttribute = resourcev1.QualifiedName("resources.kubernetes.io/pciBusID")
	// MDevUUIDAttribute is the attribute for mediated device UUID (vGPUs)
	// Note: This is not yet standardized under resources.kubernetes.io
	MDevUUIDAttribute = resourcev1.QualifiedName("mdevUUID")
)
