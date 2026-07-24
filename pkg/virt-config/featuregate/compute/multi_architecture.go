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

package compute

import "kubevirt.io/kubevirt/pkg/virt-config/featuregate"

// Owner: sig-compute / @lyarwood
// Alpha: v1.0.0
// Deprecated: v1.8.0
//
// MultiArchitecture allows VM/VMIs to request and schedule to an architecture other than that of control plane.
const MultiArchitecture = "MultiArchitecture"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:    MultiArchitecture,
		State:   featuregate.Deprecated,
		Message: "MultiArchitecture has been deprecated since v1.8.0",
	})
}

// MultiArchitectureEnabled returns true when the MultiArchitecture feature gate is enabled.
func (g ComputeFeatureGates) MultiArchitectureEnabled() bool {
	return featuregate.GateEnabled(MultiArchitecture, g.ConfigReader)
}
