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

// Owner: @lyarwood
// Alpha: v1.4.0
// Beta: v1.5.0
// GA: v1.6.0
//
// InstancetypeReferencePolicy allows a cluster admin to control how a VirtualMachine references
// instance types and preferences through the kv.spec.configuration.instancetype.referencePolicy configurable.
const InstancetypeReferencePolicy = "InstancetypeReferencePolicy"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: InstancetypeReferencePolicy, State: featuregate.GA})
}

// InstancetypeReferencePolicyEnabled returns true when the InstancetypeReferencePolicy feature gate is enabled.
func (g LegacyFeatureGates) InstancetypeReferencePolicyEnabled() bool {
	return featuregate.GateEnabled(InstancetypeReferencePolicy, g.ConfigReader)
}
