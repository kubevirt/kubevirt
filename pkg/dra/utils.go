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

// IsAllDRAGPUsReconciled checks if all GPUs with DRA in the VMI spec have corresponding status entries populated
// with either a PCI address (pGPU) or an mdev UUID (vGPU).  It is used by both virt-handler and virt-controller
// to decide whether GPU-related DRA reconciliation is complete.
func IsAllDRAGPUsReconciled(vmi *v1.VirtualMachineInstance, status *v1.DeviceStatus) bool {
	draGPUNames := make(map[string]struct{})
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if gpu.ClaimRequest != nil {
			draGPUNames[gpu.Name] = struct{}{}
		}
	}
	if len(draGPUNames) == 0 {
		return true
	}

	reconciledCount := 0
	if status != nil {
		for _, gpuStatus := range status.GPUStatuses {
			if _, isDRAGPU := draGPUNames[gpuStatus.Name]; !isDRAGPU {
				continue
			}

			if gpuStatus.DeviceResourceClaimStatus != nil &&
				gpuStatus.DeviceResourceClaimStatus.ResourceClaimName != nil &&
				gpuStatus.DeviceResourceClaimStatus.Name != nil &&
				gpuStatus.DeviceResourceClaimStatus.Attributes != nil &&
				(gpuStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil ||
					gpuStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil) {
				reconciledCount++
			}
		}
	}
	return reconciledCount == len(draGPUNames)
}

// IsAllDRAHostDevicesReconciled checks if all HostDevices with DRA in the VMI spec have corresponding status entries populated
// with either a PCI address (e.g., SR-IOV) or an mdev UUID when mediated devices are used. It mirrors the semantics of
// IsAllDRAGPUsReconciled but operates on spec.domain.devices.hostDevices instead of GPUs.
func IsAllDRAHostDevicesReconciled(vmi *v1.VirtualMachineInstance, status *v1.DeviceStatus) bool {
	draHostDeviceNames := make(map[string]struct{})
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if hd.ClaimRequest != nil {
			draHostDeviceNames[hd.Name] = struct{}{}
		}
	}
	if len(draHostDeviceNames) == 0 {
		return true
	}

	reconciledCount := 0
	if status != nil {
		for _, hdStatus := range status.HostDeviceStatuses {
			if _, isDRAHostDev := draHostDeviceNames[hdStatus.Name]; !isDRAHostDev {
				continue
			}
			if hdStatus.DeviceResourceClaimStatus != nil &&
				hdStatus.DeviceResourceClaimStatus.ResourceClaimName != nil &&
				hdStatus.DeviceResourceClaimStatus.Name != nil &&
				hdStatus.DeviceResourceClaimStatus.Attributes != nil &&
				(hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil ||
					hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil) {
				reconciledCount++
			}
		}
	}
	return reconciledCount == len(draHostDeviceNames)
}

// IsGPUDRA returns true if the GPU is a DRA GPU
func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

// IsHostDeviceDRA returns true if the HostDevice is a DRA GPU
func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

// DRA Metadata Cache - reads KEP-5304 metadata files and provides device attribute lookups

const (
	DefaultMetadataBasePath = "/var/run/dra-device-attributes"
	MetadataFileName        = "metadata.json"
)

// DRAFileData holds resolved DRA metadata indexed by VMI claim ref.
type DRAFileData struct {
	resolvedMetadata map[string]*metadata.DeviceMetadata
}

// NewDRAFileData creates a new cache, loads all metadata files from the default path,
// and resolves them against the provided VMI resource claims.
func NewDRAFileData(resourceClaims []k8sv1.PodResourceClaim) (*DRAFileData, error) {
	return NewDRAFileDataWithBasePath(DefaultMetadataBasePath, resourceClaims)
}

// NewDRAFileDataWithBasePath creates a new cache with a custom base path (for testing).
func NewDRAFileDataWithBasePath(basePath string, resourceClaims []k8sv1.PodResourceClaim) (*DRAFileData, error) {
	cache := &DRAFileData{
		resolvedMetadata: make(map[string]*metadata.DeviceMetadata),
	}

	claimsFromRC, claimsFromRCT, err := loadMetadataFiles(basePath)
	if err != nil {
		return cache, err
	}

	for _, rc := range resourceClaims {
		if rc.ResourceClaimName != nil && *rc.ResourceClaimName != "" {
			cache.resolvedMetadata[rc.Name] = claimsFromRC[*rc.ResourceClaimName]
		} else {
			cache.resolvedMetadata[rc.Name] = claimsFromRCT[rc.Name]
		}
	}

	return cache, nil
}

func loadMetadataFiles(basePath string) (claimsFromRC, claimsFromRCT map[string]*metadata.DeviceMetadata, err error) {
	claimsFromRC = make(map[string]*metadata.DeviceMetadata)
	claimsFromRCT = make(map[string]*metadata.DeviceMetadata)

	pattern := filepath.Join(basePath, "*", "*", MetadataFileName)
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

		if md.PodClaimName != "" {
			claimsFromRCT[md.PodClaimName] = &md
		}
		claimsFromRC[md.Name] = &md
	}

	return claimsFromRC, claimsFromRCT, nil
}

// GetPCIAddressForClaim returns the PCI address for a device in the given claim and request.
func (c *DRAFileData) GetPCIAddressForClaim(claimName, requestName string) (string, error) {
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
func (c *DRAFileData) GetMDevUUIDForClaim(claimName, requestName string) (string, error) {
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
			return &req.Devices[0], nil
		}
	}
	return nil, fmt.Errorf("request %q not found in metadata", requestName)
}
