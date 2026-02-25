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
	"kubevirt.io/client-go/log"

	// TODO: Replace with k8s.io types when KEP-5304 is implemented in kubernetes
	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

// IsGPUDRA returns true if the GPU is a DRA GPU
func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

// IsHostDeviceDRA returns true if the HostDevice is a DRA HostDevice
func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

// KEP-5304 defines the in-container directory layout for DRA device metadata files.
// See: kubernetes/enhancements#5304, k8s.io/dynamic-resource-allocation/api/metadata
const (
	DefaultMetadataBasePath      = "/var/run/kubernetes.io/dra-device-attributes"
	resourceClaimsSubdir         = "resourceclaims"
	resourceClaimTemplatesSubdir = "resourceclaimtemplates"
	metadataFileSuffix           = "-metadata.json"
)

// GetPCIAddressForClaim returns the PCI address for a device in the given claim and request.
// It lazily reads the KEP-5304 metadata file at lookup time.
func GetPCIAddressForClaim(basePath string, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (string, error) {
	device, err := resolveDevice(basePath, resourceClaims, claimRefName, requestName)
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

// GetMDevUUIDForClaim returns the mdev UUID for a device in the given claim and request.
// It lazily reads the KEP-5304 metadata file at lookup time.
func GetMDevUUIDForClaim(basePath string, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (string, error) {
	device, err := resolveDevice(basePath, resourceClaims, claimRefName, requestName)
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

// resolveDevice finds and reads the metadata file for a specific claim ref and
// request, returning the single device from that request.
func resolveDevice(basePath string, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (*metadata.Device, error) {
	md, err := resolveClaimMetadata(basePath, resourceClaims, claimRefName, requestName)
	if err != nil {
		return nil, err
	}

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
	return nil, fmt.Errorf("request %q not found in metadata for claim %q (available requests: %v)", requestName, md.Name, metadataRequestNames(md))
}

// resolveClaimMetadata reads the metadata file for a claim ref + request pair.
// Direct claims:   {base}/resourceclaims/{claimName}/{requestName}/{driverName}-metadata.json
// Template claims: {base}/resourceclaimtemplates/{podClaimName}/{requestName}/{driverName}-metadata.json
func resolveClaimMetadata(basePath string, resourceClaims []k8sv1.PodResourceClaim, claimRefName, requestName string) (*metadata.DeviceMetadata, error) {
	for _, rc := range resourceClaims {
		if rc.Name != claimRefName {
			continue
		}
		if rc.ResourceClaimName != nil && *rc.ResourceClaimName != "" {
			return readMetadataFromDir(filepath.Join(basePath, resourceClaimsSubdir), *rc.ResourceClaimName, requestName)
		}
		return readMetadataFromDir(filepath.Join(basePath, resourceClaimTemplatesSubdir), rc.Name, requestName)
	}
	return nil, fmt.Errorf("metadata not found for claim %q", claimRefName)
}

// readMetadataFromDir reads the first metadata file matching
// {basePath}/{claimName}/{requestName}/*-metadata.json.
// When multiple drivers serve the same request, their files are merged.
func readMetadataFromDir(basePath, claimName, requestName string) (*metadata.DeviceMetadata, error) {
	pattern := filepath.Join(basePath, claimName, requestName, "*"+metadataFileSuffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob metadata for claim %q request %q: %w", claimName, requestName, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("failed to read metadata for claim %q request %q: no files matching %s", claimName, requestName, pattern)
	}
	var merged metadata.DeviceMetadata
	for _, path := range matches {
		log.Log.Infof("Reading DRA device metadata file %s", path)
		md, err := readMetadataFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata for claim %q request %q: %w", claimName, requestName, err)
		}
		if merged.PodClaimName == nil {
			merged.ObjectMeta = md.ObjectMeta
			merged.PodClaimName = md.PodClaimName
		}
		merged.Requests = append(merged.Requests, md.Requests...)
	}
	return &merged, nil
}

func metadataRequestNames(md *metadata.DeviceMetadata) []string {
	names := make([]string, 0, len(md.Requests))
	for _, req := range md.Requests {
		names = append(names, req.Name)
	}
	return names
}

// readMetadataFile reads a KEP-5304 metadata file. The file is a JSON stream
// containing the same data encoded once per API version (newest first).
// We iterate through the stream and decode the first object whose apiVersion
// we understand, skipping unknown versions so that a driver upgrade does not
// break older consumers.
func readMetadataFile(path string) (*metadata.DeviceMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening metadata file %q: %w", path, err)
	}
	defer f.Close()

	md, err := decodeMetadataFromStream(json.NewDecoder(f))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return md, nil
}

// decodeMetadataFromStream iterates a JSON stream and returns the first
// object whose apiVersion is in metadata.SupportedAPIVersions. This is a
// lightweight equivalent of devicemetadata.DecodeMetadataFromStream that
// avoids the k8s.io/apimachinery runtime scheme dependency.
func decodeMetadataFromStream(dec *json.Decoder) (*metadata.DeviceMetadata, error) {
	var skipped []string
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("read object from metadata stream: %w", err)
		}

		var peek struct {
			APIVersion string `json:"apiVersion"`
		}
		if err := json.Unmarshal(raw, &peek); err != nil {
			skipped = append(skipped, fmt.Sprintf("unmarshal apiVersion: %v", err))
			continue
		}

		if !metadata.SupportedAPIVersions[peek.APIVersion] {
			skipped = append(skipped, peek.APIVersion)
			continue
		}

		var md metadata.DeviceMetadata
		if err := json.Unmarshal(raw, &md); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", peek.APIVersion, err))
			continue
		}
		return &md, nil
	}
	if len(skipped) > 0 {
		return nil, fmt.Errorf("no compatible metadata version found in stream (skipped: %v)", skipped)
	}
	return nil, fmt.Errorf("no metadata objects found in stream")
}

// HasNetworkDRA returns true if any of the networks use DRA.
func HasNetworkDRA(networks []v1.Network) bool {
	for _, net := range networks {
		if IsNetworkDRA(net) {
			return true
		}
	}
	return false
}

// IsNetworkDRA returns true if the network source is a ResourceClaim.
func IsNetworkDRA(net v1.Network) bool {
	return net.NetworkSource.ResourceClaim != nil
}
