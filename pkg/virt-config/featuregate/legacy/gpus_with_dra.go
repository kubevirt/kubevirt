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

package legacy

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: @alaypatel07
// Alpha: v1.6.0
// Beta: v1.9.0
//
// GPUsWithDRAGate allows users to create VMIs with DRA provisioned GPU devices.
const GPUsWithDRAGate = "GPUsWithDRA"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: GPUsWithDRAGate, State: featuregate.Beta})
}

// GPUsWithDRAGateEnabled returns true when the GPUsWithDRA feature gate is enabled.
func (g LegacyFeatureGates) GPUsWithDRAGateEnabled() bool {
	return featuregate.GateEnabled(GPUsWithDRAGate, g.ConfigReader)
}
