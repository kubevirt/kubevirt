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

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: @bmordeha
// Alpha: v1.8.0
// Beta: v1.9.0
//
// VmiMemoryOverheadReport enables reporting the memory overhead in the VMI status.
// When enabled, the memory overhead is calculated and set in the VMI status.Memory.MemoryOverhead field.
const VmiMemoryOverheadReport = "VmiMemoryOverheadReport"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: VmiMemoryOverheadReport, State: featuregate.Beta})
}

// VmiMemoryOverheadReportEnabled returns true when the VmiMemoryOverheadReport feature gate is enabled.
func (g ComputeFeatureGates) VmiMemoryOverheadReportEnabled() bool {
	return featuregate.GateEnabled(VmiMemoryOverheadReport, g.ConfigReader)
}
