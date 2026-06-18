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

// Owner: @Barakmor1
// Alpha: v1.8.0
// Beta: v1.9.0
//
// LibvirtHooksServerAndClient enables running pre-migration
// hooks on the target virt-launcher pod, allowing domain XML mutations to be applied
// on the target before migration starts.
const LibvirtHooksServerAndClient = "LibvirtHooksServerAndClient"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: LibvirtHooksServerAndClient, State: featuregate.Beta})
}

// LibvirtHooksServerAndClientEnabled returns true when the LibvirtHooksServerAndClient feature gate is enabled.
func (g LegacyFeatureGates) LibvirtHooksServerAndClientEnabled() bool {
	return featuregate.GateEnabled(LibvirtHooksServerAndClient, g.ConfigReader)
}
