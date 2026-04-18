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
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/dra/devicemetadata"
	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

type MetadataReader interface {
	ReadResourceClaimMetadata(claimName, requestName string) (*metadata.DeviceMetadata, error)
	ReadResourceClaimTemplateMetadata(podClaimName, requestName string) (*metadata.DeviceMetadata, error)
}

type metadataReader struct {
	basePath string
}

func NewMetadataReader() MetadataReader {
	return &metadataReader{basePath: metadata.ContainerDir}
}

func NewMetadataReaderWithBasePath(basePath string) MetadataReader {
	return &metadataReader{basePath: basePath}
}

func (r *metadataReader) ReadResourceClaimMetadata(claimName, requestName string) (*metadata.DeviceMetadata, error) {
	dir := filepath.Join(r.basePath, metadata.ResourceClaimsSubDir, claimName, requestName)
	return devicemetadata.ReadRequestDir(dir)
}

func (r *metadataReader) ReadResourceClaimTemplateMetadata(podClaimName, requestName string) (*metadata.DeviceMetadata, error) {
	dir := filepath.Join(r.basePath, metadata.ResourceClaimTemplatesSubDir, podClaimName, requestName)
	return devicemetadata.ReadRequestDir(dir)
}

func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

func GetPCIAddressForClaim(reader MetadataReader, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (string, error) {
	device, err := resolveDevice(reader, resourceClaims, claimRefName, requestName)
	if err != nil {
		return "", err
	}

	if attr, ok := device.Attributes[metadata.PCIBusIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("pciBusID not found for claim %q request %q", claimRefName, requestName)
}

func GetMDevUUIDForClaim(reader MetadataReader, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (string, error) {
	device, err := resolveDevice(reader, resourceClaims, claimRefName, requestName)
	if err != nil {
		return "", err
	}

	if attr, ok := device.Attributes[metadata.MDevUUIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("mdevUUID not found for claim %q request %q", claimRefName, requestName)
}

func resolveDevice(reader MetadataReader, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (*metadata.Device, error) {
	md, err := resolveClaimMetadata(reader, resourceClaims, claimRefName, requestName)
	if err != nil {
		return nil, err
	}
	return singleDeviceForRequest(md, requestName)
}

// singleDeviceForRequest returns the single device allocated to requestName.
// Upstream ReadRequestDir may produce duplicate DeviceMetadataRequest entries
// with the same Name when multiple drivers contribute to a single request
// (kubernetes/kubernetes#138352). This function collects devices across all
// matching entries before enforcing the single-device constraint.
func singleDeviceForRequest(md *metadata.DeviceMetadata, requestName string) (*metadata.Device, error) {
	var devices []metadata.Device
	for _, req := range md.Requests {
		if req.Name == requestName {
			devices = append(devices, req.Devices...)
		}
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("request %q not found in metadata for claim %q (available requests: %v)", requestName, md.Name, metadataRequestNames(md))
	}
	if len(devices) > 1 {
		return nil, fmt.Errorf("request %q has %d devices but KubeVirt only supports exactly one device per request (count > 1 is not supported)", requestName, len(devices))
	}
	return &devices[0], nil
}

func resolveClaimMetadata(reader MetadataReader, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (*metadata.DeviceMetadata, error) {
	for _, rc := range resourceClaims {
		if rc.Name != claimRefName {
			continue
		}
		if rc.ResourceClaimName != nil && *rc.ResourceClaimName != "" {
			return reader.ReadResourceClaimMetadata(*rc.ResourceClaimName, requestName)
		}
		return reader.ReadResourceClaimTemplateMetadata(rc.Name, requestName)
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
