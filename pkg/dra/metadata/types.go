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
	"k8s.io/apimachinery/pkg/runtime"
)

// These types mirror the KEP-5304 v1alpha1 API in
// k8s.io/dynamic-resource-allocation/api/metadata.
// When that module is vendored into KubeVirt, delete this file and update
// imports to point at the upstream package.

const (
	ContainerDir                 = "/var/run/kubernetes.io/dra-device-attributes"
	ResourceClaimsSubDir         = "resourceclaims"
	ResourceClaimTemplatesSubDir = "resourceclaimtemplates"
	MetadataFileSuffix           = "-metadata.json"
)

const SupportedAPIVersion = "metadata.resource.k8s.io/v1alpha1"

func MetadataFileName(driverName string) string {
	return driverName + MetadataFileSuffix
}

// Well-known attribute keys for device identification (KubeVirt-specific).
const (
	PCIBusIDAttribute = resourcev1.QualifiedName("resource.kubernetes.io/pciBusID")
	MDevUUIDAttribute = resourcev1.QualifiedName("mdevUUID")
)

type DeviceMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	PodClaimName      *string                 `json:"podClaimName,omitempty"`
	Requests          []DeviceMetadataRequest `json:"requests,omitempty"`
}

type DeviceMetadataRequest struct {
	Name    string   `json:"name"`
	Devices []Device `json:"devices,omitempty"`
}

type Device struct {
	Driver      string                                                  `json:"driver"`
	Pool        string                                                  `json:"pool"`
	Name        string                                                  `json:"name"`
	Attributes  map[resourcev1.QualifiedName]resourcev1.DeviceAttribute `json:"attributes,omitempty"`
	NetworkData *resourcev1.NetworkDeviceData                           `json:"networkData,omitempty"`
}

func (in *DeviceMetadata) DeepCopyInto(out *DeviceMetadata) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.PodClaimName != nil {
		out.PodClaimName = new(string)
		*out.PodClaimName = *in.PodClaimName
	}
	if in.Requests != nil {
		out.Requests = make([]DeviceMetadataRequest, len(in.Requests))
		for i := range in.Requests {
			in.Requests[i].DeepCopyInto(&out.Requests[i])
		}
	}
}

func (in *DeviceMetadata) DeepCopy() *DeviceMetadata {
	if in == nil {
		return nil
	}
	out := new(DeviceMetadata)
	in.DeepCopyInto(out)
	return out
}

func (in *DeviceMetadata) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *DeviceMetadataRequest) DeepCopyInto(out *DeviceMetadataRequest) {
	*out = *in
	if in.Devices != nil {
		out.Devices = make([]Device, len(in.Devices))
		for i := range in.Devices {
			in.Devices[i].DeepCopyInto(&out.Devices[i])
		}
	}
}

func (in *DeviceMetadataRequest) DeepCopy() *DeviceMetadataRequest {
	if in == nil {
		return nil
	}
	out := new(DeviceMetadataRequest)
	in.DeepCopyInto(out)
	return out
}

func (in *Device) DeepCopyInto(out *Device) {
	*out = *in
	if in.Attributes != nil {
		out.Attributes = make(map[resourcev1.QualifiedName]resourcev1.DeviceAttribute, len(in.Attributes))
		for key, val := range in.Attributes {
			out.Attributes[key] = *val.DeepCopy()
		}
	}
	if in.NetworkData != nil {
		out.NetworkData = new(resourcev1.NetworkDeviceData)
		in.NetworkData.DeepCopyInto(out.NetworkData)
	}
}

func (in *Device) DeepCopy() *Device {
	if in == nil {
		return nil
	}
	out := new(Device)
	in.DeepCopyInto(out)
	return out
}
