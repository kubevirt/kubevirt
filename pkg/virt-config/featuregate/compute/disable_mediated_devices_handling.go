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

// Owner: sig-compute
// Alpha: v1.0.0
// Deprecated: v1.8.0
//
// DisableMediatedDevicesHandling disables the handling of mediated
// devices, its creation and deletion.
const DisableMediatedDevicesHandling = "DisableMDEVConfiguration"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:    DisableMediatedDevicesHandling,
		State:   featuregate.Deprecated,
		Message: "DisableMDEVConfiguration has been deprecated since v1.8.0",
	})
}

// DisableMediatedDevicesHandlingEnabled returns true when the DisableMDEVConfiguration feature gate is enabled.
func (g ComputeFeatureGates) DisableMediatedDevicesHandlingEnabled() bool {
	return featuregate.GateEnabled(DisableMediatedDevicesHandling, g.ConfigReader)
}
