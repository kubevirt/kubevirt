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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimejson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/dynamic-resource-allocation/resourceslice"

	"github.com/opencontainers/selinux/go-selinux"
	"k8s.io/klog/v2"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1beta1"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/checkpoint"
	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/profiles"
	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/internal/util"
)

type AllocatableDevices map[string]resourceapi.Device
type PreparedDevices []*PreparedDevice

type OpaqueDeviceConfig struct {
	Requests []string
	Config   runtime.Object
}

type PreparedDevice struct {
	drapbv1.Device
	ContainerEdits *cdiapi.ContainerEdits
	AdminAccess    bool
}

func (pds PreparedDevices) GetDevices() []*drapbv1.Device {
	var devices []*drapbv1.Device
	for _, pd := range pds {
		devices = append(devices, &pd.Device)
	}
	return devices
}

type DeviceState struct {
	sync.Mutex
	driverName      string
	cdi             *CDIHandler
	driverResources resourceslice.DriverResources
	allocatable     AllocatableDevices
	configDecoder   runtime.Decoder
	configHandler   profiles.ConfigHandler

	checkpointPath string
}

func NewDeviceState(config *Config) (*DeviceState, error) {
	driverResources, err := config.profile.EnumerateDevices()
	if err != nil {
		return nil, fmt.Errorf("error enumerating all possible devices: %v", err)
	}

	cdi, err := NewCDIHandler(cdiRoot, config.driverName, config.profile.Name())
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI handler: %v", err)
	}

	err = cdi.CreateCommonSpecFile()
	if err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for common edits: %v", err)
	}

	configScheme := runtime.NewScheme()
	configHandler := config.profile
	sb := configHandler.SchemeBuilder()
	if err := sb.AddToScheme(configScheme); err != nil {
		return nil, fmt.Errorf("create config scheme: %w", err)
	}

	// Set up a json serializer to decode our types.
	configDecoder := runtimejson.NewSerializerWithOptions(
		runtimejson.DefaultMetaFactory,
		configScheme,
		configScheme,
		runtimejson.SerializerOptions{
			Pretty: true,
			// Config objects are defined by users in ResourceClaims. Strict
			// decoding helps prevent mistakes.
			//
			// Note: this flag only produces errors when decoding objects that
			// define duplicate keys. Unknown fields are still silently dropped.
			Strict: true,
		},
	)

	allocatable := make(AllocatableDevices)
	for _, slice := range driverResources.Pools[config.nodeName].Slices {
		for _, device := range slice.Devices {
			allocatable[device.Name] = device
		}
	}

	state := &DeviceState{
		driverName:      config.driverName,
		cdi:             cdi,
		driverResources: driverResources,
		allocatable:     allocatable,
		configDecoder:   configDecoder,
		configHandler:   configHandler,
		checkpointPath:  filepath.Join(config.DriverPluginPath(), DriverPluginCheckpointFile),
	}

	return state, nil
}

func (s *DeviceState) Prepare(ctx context.Context, claim *resourceapi.ResourceClaim) ([]*drapbv1.Device, error) {
	s.Lock()
	defer s.Unlock()

	ckpt, err := checkpoint.Read(s.checkpointPath)
	if err != nil {
		return nil, fmt.Errorf("unable to sync from checkpoint: %v", err)
	}
	restoredDevices, err := s.restoreClaimFromCheckpoint(ckpt, claim)
	if err != nil {
		return nil, fmt.Errorf("unable to restore from checkpoint: %v", err)
	}
	if restoredDevices != nil {
		// Recreate CDI spec file for restored claim
		if err = s.cdi.CreateClaimSpecFile(string(claim.UID), restoredDevices); err != nil {
			return nil, fmt.Errorf("unable to create CDI spec file for claim: %v", err)
		}
		return restoredDevices.GetDevices(), nil
	}

	preparedDevices, err := s.prepareDevices(ctx, claim)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %v", err)
	}
	s.addClaimToCheckpoint(ckpt, claim, preparedDevices)

	klog.FromContext(ctx).Info("Creating CDI spec file for new claim", "uid", claim.UID, "numDevices", len(preparedDevices))
	if err = s.cdi.CreateClaimSpecFile(string(claim.UID), preparedDevices); err != nil {
		return nil, fmt.Errorf("unable to create CDI spec file for claim: %v", err)
	}

	if err := checkpoint.Write(s.checkpointPath, ckpt); err != nil {
		return nil, fmt.Errorf("unable to sync to checkpoint: %v", err)
	}

	return preparedDevices.GetDevices(), nil
}

func (s *DeviceState) Unprepare(ctx context.Context, claimUID types.UID) error {
	s.Lock()
	defer s.Unlock()

	ckpt, err := checkpoint.Read(s.checkpointPath)
	if err != nil {
		return fmt.Errorf("unable to read checkpoint: %v", err)
	}

	if err = s.unprepareDevices(ctx, claimUID, ckpt); err != nil {
		return fmt.Errorf("unprepare failed: %v", err)
	}
	s.removeClaimFromCheckpoint(ckpt, claimUID)

	err = s.cdi.DeleteClaimSpecFile(string(claimUID))
	if err != nil {
		return fmt.Errorf("unable to delete CDI spec file for claim: %v", err)
	}

	if err := checkpoint.Write(s.checkpointPath, ckpt); err != nil {
		return fmt.Errorf("unable to sync to checkpoint: %v", err)
	}

	return nil
}

// prepareDevices performs one-time setup for the devices allocated to a
// ResourceClaim before being consumed by a Pod.
func (s *DeviceState) prepareDevices(ctx context.Context, claim *resourceapi.ResourceClaim) (PreparedDevices, error) {
	// Create directories for each allocated device BEFORE computing device config
	// Directory path format: {base}/{stable-claim-name}/{request-name}/
	// A device.json metadata file is created inside with the device ID
	for _, result := range claim.Status.Allocation.Devices.Results {
		if result.Driver != s.driverName {
			continue
		}
		claimDir, err := s.createClaimDirectory(ctx, claim.Name, result.Request, result.Device)
		if err != nil {
			return nil, fmt.Errorf("failed to create claim directory: %w", err)
		}
		klog.FromContext(ctx).Info("Created directory for claim device",
			"path", claimDir, "claim", claim.Name, "request", result.Request, "device", result.Device)
	}

	// Compute device configuration (which will call ApplyConfig with claim.Name)
	preparedDevices, err := s.computeDeviceConfig(claim)
	if err != nil {
		return nil, err
	}

	return preparedDevices, nil
}

// unprepareDevices undoes any side-effects produced by
// [DeviceState.prepareDevices].
func (s *DeviceState) unprepareDevices(ctx context.Context, claimUID types.UID, ckpt *checkpoint.Checkpoint) error {
	// Find the claim in the checkpoint to get its name and devices
	for _, preparedClaim := range ckpt.PreparedClaims {
		if preparedClaim.UID == claimUID {
			// Delete directories for all devices allocated to this claim
			var errs []error
			for _, deviceName := range preparedClaim.Devices {
				requestName := preparedClaim.DeviceRequests[deviceName]
				if err := s.deleteClaimDirectory(ctx, preparedClaim.Name, requestName); err != nil {
					klog.FromContext(ctx).Error(err, "Failed to delete claim directory",
						"claim", preparedClaim.Name, "request", requestName, "device", deviceName)
					errs = append(errs, err)
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("failed to delete %d claim directories", len(errs))
			}
			break
		}
	}

	return nil
}

// computeDeviceConfig computes the CDI config for devices allocated to the claim
// designated for this driver. It is called each time the kubelet tells the
// driver to prepare a claim which may occur more than once, and therefore
// should be deterministic and produce no side-effects. Non-deterministic data or
// side-effects should be produced by [DeviceState.prepareDevices] directly and
// recorded in the checkpoint by [DeviceState.addClaimToCheckpoint].
func (s *DeviceState) computeDeviceConfig(claim *resourceapi.ResourceClaim) (PreparedDevices, error) {
	if claim.Status.Allocation == nil {
		return nil, fmt.Errorf("claim not yet allocated")
	}
	// Check if any device request has admin access
	hasAdminAccess := s.checkAdminAccess(claim)

	// Retrieve the full set of device configs for the driver.
	configs, err := GetOpaqueDeviceConfigs(
		s.configDecoder,
		s.driverName,
		claim.Status.Allocation.Devices.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting opaque device configs: %v", err)
	}

	// Add the default device config to the front of the config list with the
	// lowest precedence. This guarantees there will be at least one config in
	// the list with len(Requests) == 0 for the lookup below.
	configs = slices.Insert(configs, 0, &OpaqueDeviceConfig{})

	// Look through the configs and figure out which one will be applied to
	// each device allocation result based on their order of precedence.
	configResultsMap := make(map[runtime.Object][]*resourceapi.DeviceRequestAllocationResult)
	for _, result := range claim.Status.Allocation.Devices.Results {
		// The claim may include allocations meant for other drivers.
		if result.Driver != s.driverName {
			continue
		}
		if _, exists := s.allocatable[result.Device]; !exists {
			return nil, fmt.Errorf("requested device is not allocatable: %v", result.Device)
		}

		for _, c := range slices.Backward(configs) {
			if len(c.Requests) == 0 || slices.Contains(c.Requests, result.Request) {
				configResultsMap[c.Config] = append(configResultsMap[c.Config], &result)
				break
			}
		}
	}

	// Apply all configs associated with devices that need to be prepared.
	// Track container edits generated from applying the config to the set
	// of device allocation results.
	perDeviceCDIContainerEdits := make(profiles.PerDeviceCDIContainerEdits)
	for config, results := range configResultsMap {
		// Apply the config to the list of results associated with it.
		containerEdits, err := s.configHandler.ApplyConfig(claim.Name, config, results)
		if err != nil {
			return nil, fmt.Errorf("error applying config: %w", err)
		}

		// Merge any new container edits with the overall per device map.
		for k, v := range containerEdits {
			perDeviceCDIContainerEdits[k] = v
		}
	}

	// Walk through each config and its associated device allocation results
	// and construct the list of prepared devices to return.
	var preparedDevices PreparedDevices
	for _, results := range configResultsMap {
		for _, result := range results {
			device := &PreparedDevice{
				Device: drapbv1.Device{
					RequestNames: []string{result.Request},
					PoolName:     result.Pool,
					DeviceName:   result.Device,
					CDIDeviceIDs: s.cdi.GetClaimDevices(string(claim.UID), []string{result.Device}),
				},
				ContainerEdits: perDeviceCDIContainerEdits[result.Device],
				AdminAccess:    hasAdminAccess,
			}
			preparedDevices = append(preparedDevices, device)
		}
	}

	return preparedDevices, nil
}

// addClaimToCheckpoint updates the checkpoint with results of preparing the
// devices for the claim. If any parts of the [PreparedDevices] are
// non-deterministic or expensive to recompute, then those should also be added
// to the checkpoint here.
func (s *DeviceState) addClaimToCheckpoint(ckpt *checkpoint.Checkpoint, claim *resourceapi.ResourceClaim, _ PreparedDevices) {
	// Extract device names and request mappings from the allocation
	var devices []string
	deviceRequests := make(map[string]string)
	if claim.Status.Allocation != nil {
		for _, result := range claim.Status.Allocation.Devices.Results {
			// Only add devices belonging to this driver
			if result.Driver != s.driverName {
				continue
			}
			devices = append(devices, result.Device)
			deviceRequests[result.Device] = result.Request
		}
	}

	ckpt.PreparedClaims = append(ckpt.PreparedClaims, checkpoint.PreparedClaim{
		UID:            claim.UID,
		Name:           claim.Name,
		Devices:        devices,
		DeviceRequests: deviceRequests,
	})
}

// removeClaimFromCheckpoint updates the checkpoint to remove all data
// associated with the claim.
func (*DeviceState) removeClaimFromCheckpoint(ckpt *checkpoint.Checkpoint, claimUID types.UID) {
	ckpt.PreparedClaims = slices.DeleteFunc(ckpt.PreparedClaims, func(c checkpoint.PreparedClaim) bool { return c.UID == claimUID })
}

// restoreClaimFromCheckpoint returns the device definitions for devices already prepared
// for the given claim. If the claim has not yet been prepared, it returns nil.
func (s *DeviceState) restoreClaimFromCheckpoint(ckpt *checkpoint.Checkpoint, claim *resourceapi.ResourceClaim) (PreparedDevices, error) {
	if slices.ContainsFunc(ckpt.PreparedClaims, func(c checkpoint.PreparedClaim) bool { return c.UID == claim.UID }) {
		// If [DeviceState.addClaimToCheckpoint] associated any other data with
		// the claim in the checkpoint, then that should be added to the
		// returned [PreparedDevices] here.
		return s.computeDeviceConfig(claim)
	}
	return nil, nil
}

// checkAdminAccess determines if a resource claim requires admin access.
func (s *DeviceState) checkAdminAccess(claim *resourceapi.ResourceClaim) bool {
	if claim != nil && claim.Status.Allocation != nil {
		for _, result := range claim.Status.Allocation.Devices.Results {
			if result.AdminAccess != nil && *result.AdminAccess {
				return true
			}
		}
	}
	return false
}

// GetOpaqueDeviceConfigs returns an ordered list of the configs contained in possibleConfigs for this driver.
//
// Configs can either come from the resource claim itself or from the device
// class associated with the request. Configs coming directly from the resource
// claim take precedence over configs coming from the device class. Moreover,
// configs found later in the list of configs attached to its source take
// precedence over configs found earlier in the list for that source.
//
// All of the configs relevant to the driver from the list of possibleConfigs
// will be returned in order of precedence (from lowest to highest). If no
// configs are found, nil is returned.
func GetOpaqueDeviceConfigs(
	decoder runtime.Decoder,
	driverName string,
	possibleConfigs []resourceapi.DeviceAllocationConfiguration,
) ([]*OpaqueDeviceConfig, error) {
	// Collect all configs in order of reverse precedence.
	var classConfigs []resourceapi.DeviceAllocationConfiguration
	var claimConfigs []resourceapi.DeviceAllocationConfiguration
	var candidateConfigs []resourceapi.DeviceAllocationConfiguration
	for _, config := range possibleConfigs {
		switch config.Source {
		case resourceapi.AllocationConfigSourceClass:
			classConfigs = append(classConfigs, config)
		case resourceapi.AllocationConfigSourceClaim:
			claimConfigs = append(claimConfigs, config)
		default:
			return nil, fmt.Errorf("invalid config source: %v", config.Source)
		}
	}
	candidateConfigs = append(candidateConfigs, classConfigs...)
	candidateConfigs = append(candidateConfigs, claimConfigs...)

	// Decode all configs that are relevant for the driver.
	var resultConfigs []*OpaqueDeviceConfig
	for _, config := range candidateConfigs {
		// If this is nil, the driver doesn't support some future API extension
		// and needs to be updated.
		if config.Opaque == nil {
			return nil, fmt.Errorf("only opaque parameters are supported by this driver")
		}

		// Configs for different drivers may have been specified because a
		// single request can be satisfied by different drivers. This is not
		// an error -- drivers must skip over other driver's configs in order
		// to support this.
		if config.Opaque.Driver != driverName {
			continue
		}

		decodedConfig, err := runtime.Decode(decoder, config.Opaque.Parameters.Raw)
		if err != nil {
			return nil, fmt.Errorf("error decoding config parameters: %w", err)
		}

		resultConfig := &OpaqueDeviceConfig{
			Requests: config.Requests,
			Config:   decodedConfig,
		}

		resultConfigs = append(resultConfigs, resultConfig)
	}

	return resultConfigs, nil
}

// DeviceMetadata stores device-specific information in the claim directory.
type DeviceMetadata struct {
	DeviceID string `json:"device_id"`
}

// createClaimDirectory creates a subdirectory for the claim+request.
// Directory path format: {base}/{stable-claim-name}/{request-name}/.
// The stable-claim-name is derived from the full claim name to be migration-stable.
// Creates a device.json metadata file inside with the device ID.
// Sets permissions to 0775, ownership to 107:107, and SELinux label to container_file_t.
func (s *DeviceState) createClaimDirectory(ctx context.Context, claimName string, requestName string, deviceName string) (string, error) {
	const baseDir = "/var/run/kubevirt/cdi"
	const qemuUID = 107
	const qemuGID = 107

	// Extract migration-stable portion of claim name
	stableClaimName := util.ExtractStableClaimName(claimName)
	klog.FromContext(ctx).Info("Creating directory",
		"fullClaim", claimName, "stableClaim", stableClaimName, "request", requestName, "device", deviceName)

	claimDir := filepath.Join(baseDir, stableClaimName, requestName)

	// Create directory structure with final permissions (0775) to avoid race condition
	if err := os.MkdirAll(claimDir, 0775); err != nil {
		return "", err
	}

	// Set ownership and SELinux label for all created directories
	// so containers running as uid 107 can access them
	claimRoot := filepath.Join(baseDir, stableClaimName)

	for _, dir := range []string{claimRoot, claimDir} {
		// Set ownership to qemu user/group
		if err := os.Chown(dir, qemuUID, qemuGID); err != nil {
			return "", fmt.Errorf("failed to chown %s: %w", dir, err)
		}
		// Set SELinux label for container access (shared across containers)
		// Always try to set the label; if SELinux is disabled or not supported, it will fail silently
		if err := selinux.SetFileLabel(dir, "system_u:object_r:container_file_t:s0"); err != nil {
			// Log but don't fail - SELinux might not be enabled
			klog.FromContext(ctx).Info("Could not set SELinux label (this is OK if SELinux is disabled)", "path", dir, "error", err)
		} else {
			klog.FromContext(ctx).Info("Successfully set SELinux label to container_file_t", "path", dir)
		}
	}

	// Create device metadata file
	metadataPath := filepath.Join(claimDir, "device.json")
	metadata := DeviceMetadata{
		DeviceID: deviceName,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal device metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write device metadata: %w", err)
	}

	// Set ownership and permissions on metadata file
	if err := os.Chown(metadataPath, qemuUID, qemuGID); err != nil {
		return "", fmt.Errorf("failed to chown metadata file: %w", err)
	}

	if err := selinux.SetFileLabel(metadataPath, "system_u:object_r:container_file_t:s0"); err != nil {
		klog.FromContext(ctx).Info("Could not set SELinux label (this is OK if SELinux is disabled)", "path", metadataPath, "error", err)
	}

	klog.FromContext(ctx).Info("Created device metadata file", "path", metadataPath, "deviceID", deviceName)

	return claimDir, nil
}

// deleteClaimDirectory removes the claim directory.
func (s *DeviceState) deleteClaimDirectory(ctx context.Context, claimName string, requestName string) error {
	const baseDir = "/var/run/kubevirt/cdi"
	// Use stable claim name for consistency
	stableClaimName := util.ExtractStableClaimName(claimName)
	claimDir := filepath.Join(baseDir, stableClaimName, requestName)
	klog.FromContext(ctx).Info("Deleting directory",
		"fullClaim", claimName, "stableClaim", stableClaimName, "path", claimDir)
	return os.RemoveAll(claimDir)
}
