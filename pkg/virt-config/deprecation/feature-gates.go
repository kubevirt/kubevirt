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

package deprecation

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

type State string

const (
	// By default, GAed feature gates are considered enabled and no-op.
	GA = "General Availability"
	// The feature is going to be discontinued next release
	Deprecated     = "Deprecated"
	Discontinued   = "Discontinued"
	WarningPattern = "feature gate %s is deprecated (feature state is %q), therefore it can be safely removed and is redundant. " +
		"For more info, please look at: https://github.com/kubevirt/kubevirt/blob/main/docs/deprecation.md"
)

const (
	LiveMigrationGate      = "LiveMigration"      // GA
	SRIOVLiveMigrationGate = "SRIOVLiveMigration" // GA
	NonRoot                = "NonRoot"            // GA
	PSA                    = "PSA"                // GA
	CPUNodeDiscoveryGate   = "CPUNodeDiscovery"   // GA
	PasstGate              = "Passt"              // Deprecated
	MacvtapGate            = "Macvtap"            // Deprecated
)

type FeatureGate struct {
	Name        string
	State       State
	VmiSpecUsed func(spec *v1.VirtualMachineInstanceSpec) bool
	Message     string
}

var featureGates = map[string]FeatureGate{}

func init() {
	RegisterFeatureGate(FeatureGate{Name: LiveMigrationGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: SRIOVLiveMigrationGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: NonRoot, State: GA})
	RegisterFeatureGate(FeatureGate{Name: PSA, State: GA})
	RegisterFeatureGate(FeatureGate{Name: CPUNodeDiscoveryGate, State: GA})

	RegisterFeatureGate(FeatureGate{Name: PasstGate, State: Discontinued, Message: PasstDiscontinueMessage, VmiSpecUsed: passtApiUsed})
	RegisterFeatureGate(FeatureGate{Name: MacvtapGate, State: Discontinued, Message: MacvtapDiscontinueMessage, VmiSpecUsed: macvtapApiUsed})
}

// RegisterFeatureGate adds a given feature-gate to the FG list
// In case the FG already exists (based on its name), it overrides the
// existing FG.
// If the feature-gate is missing a message, a default one is set.
func RegisterFeatureGate(fg FeatureGate) {
	if fg.Message == "" {
		fg.Message = fmt.Sprintf(WarningPattern, fg.Name, fg.State)
	}
	featureGates[fg.Name] = fg
}

func UnregisterFeatureGate(fgName string) {
	delete(featureGates, fgName)
}

func FeatureGateInfo(featureGate string) *FeatureGate {
	if fg, exist := featureGates[featureGate]; exist {
		return &fg
	}
	return nil
}
