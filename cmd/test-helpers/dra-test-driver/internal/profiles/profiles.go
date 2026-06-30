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

package profiles

import (
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
)

type PerDeviceCDIContainerEdits map[string]*cdiapi.ContainerEdits

// Profile describes a kind of device that can be managed by the driver.
type Profile interface {
	ConfigHandler
	// Name returns the profile name used for CDI vendor identification
	Name() string
	EnumerateDevices() (resourceslice.DriverResources, error)
}

// ConfigHandler handles opaque configuration set for requests in ResourceClaims.
type ConfigHandler interface {
	// SchemeBuilder produces a [runtime.Scheme] for the profile's configuration types.
	SchemeBuilder() runtime.SchemeBuilder
	// Validate returns nil for valid configuration, or an error explaining why the configuration is invalid.
	Validate(config runtime.Object) error
	// ApplyConfig applies a configuration to a set of device allocation
	// results. When `config` is nil, the profile's default configuration should
	// be applied. The claimName is the name of the ResourceClaim being prepared.
	ApplyConfig(claimName string, config runtime.Object, results []*resourceapi.DeviceRequestAllocationResult) (PerDeviceCDIContainerEdits, error)
}
