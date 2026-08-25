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

package dra

import (
	"fmt"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/dynamic-resource-allocation/api/metadata"
	"k8s.io/dynamic-resource-allocation/devicemetadata"

	v1 "kubevirt.io/api/core/v1"
)

var (
	readResourceClaim         = devicemetadata.ReadResourceClaimMetadata
	readResourceClaimTemplate = devicemetadata.ReadResourceClaimTemplateMetadata
)

// IsGPUDRA returns true if the GPU is a DRA GPU
func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

// IsHostDeviceDRA returns true if the HostDevice is a DRA HostDevice
func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

const (
	PCIBusIDAttribute = resourcev1.QualifiedName("resource.kubernetes.io/pciBusID")
	MDevUUIDAttribute = resourcev1.QualifiedName("mdevUUID")
)

// GetPCIAddressForClaim returns the PCI address for a device in the given claim and request.
func GetPCIAddressForClaim(
	resourceClaims []v1.VirtualMachineInstanceResourceClaim,
	claimRefName,
	requestName string,
) (string, error) {
	device, err := deviceLookupFromClaim(resourceClaims, claimRefName, requestName)
	if err != nil {
		return "", err
	}
	if attr, ok := device.Attributes[PCIBusIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("pciBusID not found for claim %q request %q", claimRefName, requestName)
}

// GetMDevUUIDForClaim returns the mdev UUID for a device in the given claim and request.
func GetMDevUUIDForClaim(
	resourceClaims []v1.VirtualMachineInstanceResourceClaim,
	claimRefName,
	requestName string,
) (string, error) {
	device, err := deviceLookupFromClaim(resourceClaims, claimRefName, requestName)
	if err != nil {
		return "", err
	}
	if attr, ok := device.Attributes[MDevUUIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("mdevUUID not found for claim %q request %q", claimRefName, requestName)
}

func deviceLookupFromClaim(
	resourceClaims []v1.VirtualMachineInstanceResourceClaim,
	claimRefName,
	requestName string,
) (*metadata.Device, error) {
	md, err := readClaimMetadata(resourceClaims, claimRefName, requestName)
	if err != nil {
		return nil, err
	}

	for _, req := range md.Requests {
		if req.Name == requestName {
			if len(req.Devices) == 0 {
				return nil, fmt.Errorf("request %q has no devices", requestName)
			}
			if len(req.Devices) > 1 {
				return nil, fmt.Errorf(
					"request %q has %d devices but KubeVirt only supports exactly one device per request (count > 1 is not supported)",
					requestName,
					len(req.Devices),
				)
			}
			return &req.Devices[0], nil
		}
	}
	return nil, fmt.Errorf(
		"request %q not found in metadata for claim %q (available requests: %v)",
		requestName,
		md.Name,
		metadataRequestNames(md),
	)
}

func readClaimMetadata(
	resourceClaims []v1.VirtualMachineInstanceResourceClaim,
	claimRefName,
	requestName string,
) (*metadata.DeviceMetadata, error) {
	for _, rc := range resourceClaims {
		if rc.Name != claimRefName {
			continue
		}
		if rc.ResourceClaimName != nil {
			return readResourceClaim(*rc.ResourceClaimName, requestName)
		}
		return readResourceClaimTemplate(rc.Name, requestName)
	}
	return nil, fmt.Errorf("metadata not found for claim %q", claimRefName)
}

func metadataRequestNames(md *metadata.DeviceMetadata) []string {
	names := make([]string, 0, len(md.Requests))
	for _, req := range md.Requests {
		names = append(names, req.Name)
	}
	return names
}
