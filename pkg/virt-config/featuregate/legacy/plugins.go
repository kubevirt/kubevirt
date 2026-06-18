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

// Owner: @iholder101
// Alpha: v1.9.0
//
// Plugins enables the Plugin CRD for declarative VM extension
// via domain hooks, node hooks, and admission references (VEP-190).
const PluginsGate = "Plugins"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: PluginsGate, State: featuregate.Alpha})
}

// PluginsEnabled returns true when the Plugins feature gate is enabled.
func (g LegacyFeatureGates) PluginsEnabled() bool {
	return featuregate.GateEnabled(PluginsGate, g.ConfigReader)
}
