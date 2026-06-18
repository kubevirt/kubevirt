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

const Root = "Root"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: Root, State: featuregate.Alpha})
}

// RootEnabled returns true when the Root feature gate is enabled.
func (g LegacyFeatureGates) RootEnabled() bool {
	return featuregate.GateEnabled(Root, g.ConfigReader)
}
