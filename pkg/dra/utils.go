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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	// TODO: Replace with k8s.io types when KEP-5304 is implemented in kubernetes
	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

// IsGPUDRA returns true if the GPU is a DRA GPU
func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

// IsHostDeviceDRA returns true if the HostDevice is a DRA GPU
func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

// DRA Downward API - reads KEP-5304 metadata files and provides device attribute lookups
const (
	DefaultMetadataBasePath = "/var/run/dra-device-attributes"
)

// DownwardAPIAttributes holds resolved DRA device attributes indexed by VMI claim ref.
type DownwardAPIAttributes struct {
	resolvedMetadata map[string]*metadata.DeviceMetadata
}

// NewDownwardAPIAttributes loads all metadata files from the default path
// and resolves them against the provided VMI resource claims.
func NewDownwardAPIAttributes(resourceClaims []k8sv1.PodResourceClaim) (*DownwardAPIAttributes, error) {
	return NewDownwardAPIAttributesWithBasePath(DefaultMetadataBasePath, resourceClaims)
}

// NewDownwardAPIAttributesWithBasePath loads metadata files from a custom base path (for testing).
func NewDownwardAPIAttributesWithBasePath(basePath string, resourceClaims []k8sv1.PodResourceClaim) (*DownwardAPIAttributes, error) {
	attrs := &DownwardAPIAttributes{
		resolvedMetadata: make(map[string]*metadata.DeviceMetadata),
	}

	claimsFromRC, claimsFromRCT, err := loadMetadataFiles(basePath)
	if err != nil {
		return attrs, err
	}

	// Resolve metadata by vmi.spec.resourceClaims[].name so that during
	// domain XML creation, host device builders can look up device attributes
	// using the same name that appears in gpu.ClaimRequest.ClaimName.
	for _, rc := range resourceClaims {
		if rc.ResourceClaimName != nil && *rc.ResourceClaimName != "" {
			attrs.resolvedMetadata[rc.Name] = claimsFromRC[*rc.ResourceClaimName]
		} else {
			attrs.resolvedMetadata[rc.Name] = claimsFromRCT[rc.Name]
		}
	}

	return attrs, nil
}

func loadMetadataFiles(basePath string) (claimsFromRC, claimsFromRCT map[string]*metadata.DeviceMetadata, err error) {
	claimsFromRC = make(map[string]*metadata.DeviceMetadata)
	claimsFromRCT = make(map[string]*metadata.DeviceMetadata)

	// KEP-5304 container path: {base}/{claimName}/{requestName}/{driverName}-metadata.json
	pattern := filepath.Join(basePath, "*", "*", "*-metadata.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to glob metadata files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var md metadata.DeviceMetadata
		if err := json.Unmarshal(data, &md); err != nil {
			continue
		}

		// Every metadata file is indexed by its ObjectMeta.Name (the actual
		// ResourceClaim name) for when the VMI spec references a ResourceClaim
		// directly. When the VMI spec uses a ResourceClaimTemplate, the file
		// is additionally indexed by PodClaimName, because the VMI spec only
		// knows the pod-level reference name, not the auto-generated
		// ResourceClaim name. A single metadata file may therefore appear in
		// both maps.
		if md.PodClaimName != nil {
			claimsFromRCT[*md.PodClaimName] = &md
		}
		claimsFromRC[md.Name] = &md
	}

	return claimsFromRC, claimsFromRCT, nil
}

// GetPCIAddressForClaim returns the PCI address for a device in the given claim and request.
func (c *DownwardAPIAttributes) GetPCIAddressForClaim(claimName, requestName string) (string, error) {
	md := c.resolvedMetadata[claimName]
	if md == nil {
		return "", fmt.Errorf("metadata not found for claim %q", claimName)
	}

	device, err := getDeviceForRequest(md, requestName)
	if err != nil {
		return "", err
	}

	if attr, ok := device.Attributes[metadata.PCIBusIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("pciBusID not found for claim %q request %q", claimName, requestName)
}

// GetMDevUUIDForClaim returns the mdev UUID for a device in the given claim and request.
func (c *DownwardAPIAttributes) GetMDevUUIDForClaim(claimName, requestName string) (string, error) {
	md := c.resolvedMetadata[claimName]
	if md == nil {
		return "", fmt.Errorf("metadata not found for claim %q", claimName)
	}

	device, err := getDeviceForRequest(md, requestName)
	if err != nil {
		return "", err
	}

	if attr, ok := device.Attributes[metadata.MDevUUIDAttribute]; ok {
		if attr.StringValue != nil && *attr.StringValue != "" {
			return *attr.StringValue, nil
		}
	}
	return "", fmt.Errorf("mdevUUID not found for claim %q request %q", claimName, requestName)
}

func getDeviceForRequest(md *metadata.DeviceMetadata, requestName string) (*metadata.Device, error) {
	for _, req := range md.Requests {
		if req.Name == requestName {
			if len(req.Devices) == 0 {
				return nil, fmt.Errorf("request %q has no devices", requestName)
			}
			if len(req.Devices) > 1 {
				return nil, fmt.Errorf("request %q has %d devices but KubeVirt only supports exactly one device per request (count > 1 is not supported)", requestName, len(req.Devices))
			}
			return &req.Devices[0], nil
		}
	}
	return nil, fmt.Errorf("request %q not found in metadata", requestName)
}
