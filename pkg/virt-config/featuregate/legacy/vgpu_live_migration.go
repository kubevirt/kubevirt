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

// Owner: @csomani1
// Alpha: v1.8.0
//
// VGPULiveMigration enables the vGPU hook to run for vGPU live migrations, allowing the
// target XML's mdev UUID to be mutated.
const VGPULiveMigration = "VGPULiveMigration"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: VGPULiveMigration, State: featuregate.Alpha})
}

// VGPULiveMigrationEnabled returns true when the VGPULiveMigration feature gate is enabled.
func (g LegacyFeatureGates) VGPULiveMigrationEnabled() bool {
	return featuregate.GateEnabled(VGPULiveMigration, g.ConfigReader)
}
