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

package capabilities

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
)

type CapabilityKey string // e.g., "graphics.vga", "firmware.secureboot.uefi"
type SupportLevel int

const (
	Unregistered SupportLevel = iota // Not registered (default zero value)
	Unsupported                      // Explicitly blocked on this platform
	Experimental                     // Requires feature gate
	Deprecated                       // Supported but discouraged
)

type Platform string

const (
	Universal Platform = "" // Applies to all platforms
)

type Capability struct {
	// Returns all field paths where this capability is required, empty slice if not required
	GetRequiredFields func(vmi *v1.VirtualMachineInstance) []*field.Path
}

// struct to store the extent to which a given capability is supported
type CapabilitySupport struct {
	Level   SupportLevel
	Message string // User-facing explanation
	GatedBy string // Optional: feature gate name
}

func PlatformKeyFromHypervisor(hypervisor string) Platform {
	return Platform(hypervisor + "/")
}

func PlatformKeyFromArch(arch string) Platform {
	return Platform("/" + arch)
}

func PlatformKeyFromHypervisorAndArch(hypervisor, arch string) Platform {
	return Platform(hypervisor + "/" + arch)
}
