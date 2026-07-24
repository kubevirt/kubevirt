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
	"strings"
	"testing"

	core_capabilities "kubevirt.io/kubevirt/pkg/capabilities/core"

	// Importing the arch and hypervisor packages to register their capabilities
	// via their init functions
	_ "kubevirt.io/kubevirt/pkg/capabilities/arch"
	_ "kubevirt.io/kubevirt/pkg/capabilities/hypervisor"
)

func TestExperimentalCapabilitiesMustDefineFeatureGate(t *testing.T) {
	supports := core_capabilities.GetAllPlatformCapabilitySupport()

	if len(supports) == 0 {
		t.Fatal("expected at least one registered capability support")
	}

	for platform, platformSupports := range supports {
		for capabilityKey, support := range platformSupports {
			if support.Level == core_capabilities.Experimental && strings.TrimSpace(support.GatedBy) == "" {
				t.Errorf("capability %q on platform %q is Experimental but has an empty GatedBy field", capabilityKey, platform)
			}
		}
	}
}
