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

// ReservedOverheadMemlock enables using the spec.domain.memory.ReservedOverhead field which
// can specify some required memory overhead as well as whether VM
// memory (and overhead) needs to be locked or not.
// Owner: sig-compute / @bgartzi
// Alpha: v1.8.0
const ReservedOverheadMemlock = "ReservedOverheadMemlock"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ReservedOverheadMemlock, State: featuregate.Alpha})
}

// ReservedOverheadMemlockEnabled returns true when the ReservedOverheadMemlock feature gate is enabled.
func (g ComputeFeatureGates) ReservedOverheadMemlockEnabled() bool {
	return featuregate.GateEnabled(ReservedOverheadMemlock, g.ConfigReader)
}
