/*
 * Copyright The Kubernetes Authors.
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
 */

package network

import (
	"fmt"
	"os"
	"path/filepath"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdispec "tags.cncf.io/container-device-interface/specs-go"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/profiles"
	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/util"
)

const (
	ProfileName = "hostpath"

	// HostBaseDir is where directories are created on the host.
	HostBaseDir = "/var/run/kubevirt/cdi"
)

type Profile struct {
	nodeName   string
	numDevices int
}

func NewProfile(nodeName string, numDevices int) Profile {
	return Profile{
		nodeName:   nodeName,
		numDevices: numDevices,
	}
}

// Name returns the profile name for CDI vendor identification.
func (p Profile) Name() string {
	return ProfileName
}

// EnumerateDevices advertises the available network directory devices.
// This is called at driver startup (discovery time).
func (p Profile) EnumerateDevices() (resourceslice.DriverResources, error) {
	// Create the base directory at discovery time
	if err := os.MkdirAll(HostBaseDir, 0755); err != nil {
		return resourceslice.DriverResources{}, fmt.Errorf("failed to create base directory %s: %w", HostBaseDir, err)
	}

	var devices []resourceapi.Device

	// Create N simple devices (just slots for directory claims)
	for i := 0; i < p.numDevices; i++ {
		devices = append(devices, resourceapi.Device{
			Name: fmt.Sprintf("hostpath-%d", i),
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
				"index": {
					IntValue: new(int64(i)),
				},
				"type": {
					StringValue: new("kubevirt-network-directory"),
				},
			},
		})
	}

	resources := resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			p.nodeName: {
				Slices: []resourceslice.Slice{{Devices: devices}},
			},
		},
	}

	return resources, nil
}

// SchemeBuilder implements profiles.ConfigHandler.
// No custom config needed for network directories.
func (p Profile) SchemeBuilder() runtime.SchemeBuilder {
	return runtime.NewSchemeBuilder()
}

// Validate implements profiles.ConfigHandler.
// No custom config to validate.
func (p Profile) Validate(config runtime.Object) error {
	if config != nil {
		return fmt.Errorf("configuration not supported for network profile")
	}
	return nil
}

// ApplyConfig creates a directory per claim and mounts it via CDI.
// Note: The actual directory creation happens in state.prepareDevices().
// This function only configures the CDI mount specification using the stable claim name.
func (p Profile) ApplyConfig(claimName string, config runtime.Object, results []*resourceapi.DeviceRequestAllocationResult) (profiles.PerDeviceCDIContainerEdits, error) {
	perDeviceEdits := make(profiles.PerDeviceCDIContainerEdits)

	// Extract migration-stable portion of claim name
	stableClaimName := util.ExtractStableClaimName(claimName)

	for _, result := range results {
		// Build directory path: {base}/{stable-claim-name}/{request-name}/
		// The device ID is stored in a device.json file inside this directory
		claimDir := filepath.Join(HostBaseDir, stableClaimName, result.Request)

		edits := &cdispec.ContainerEdits{
			Env: []string{
				fmt.Sprintf("KUBEVIRT_HOSTPATH_DEVICE=%s", result.Device),
				fmt.Sprintf("KUBEVIRT_HOSTPATH_PATH=%s", claimDir),
				fmt.Sprintf("KUBEVIRT_HOSTPATH_REQUEST=%s", result.Request),
			},
			Mounts: []*cdispec.Mount{
				{
					HostPath:      claimDir,
					ContainerPath: claimDir,
					Options:       []string{"rbind", "z"},
				},
			},
		}

		perDeviceEdits[result.Device] = &cdiapi.ContainerEdits{ContainerEdits: edits}
	}

	return perDeviceEdits, nil
}
