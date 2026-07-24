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

// Owner: sig-compute / @jschintag
// Alpha: v1.6.0
// Beta: v1.7.0
// GA: v1.9.0
//
// SecureExecution introduces secure execution of VMs on IBM Z architecture.
const SecureExecution = "SecureExecution"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: SecureExecution, State: featuregate.GA})
}

// SecureExecutionEnabled returns true when the SecureExecution feature gate is enabled.
func (g ComputeFeatureGates) SecureExecutionEnabled() bool {
	return featuregate.GateEnabled(SecureExecution, g.ConfigReader)
}
