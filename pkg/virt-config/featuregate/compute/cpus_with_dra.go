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

// Owner: @sreeja1725
// Alpha: v1.10.0
//
// CPUsWithDRAGate allows users to create VMIs with DRA provisioned CPUs.
// Do not disable this gate while a live migration is in progress. virt-handler re-reads it on
// each sync, so turning it off reports a VMI that holds a CPU claim as migratable, and the
// migration fails later in virt-controller.
const CPUsWithDRAGate = "CPUsWithDRA"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: CPUsWithDRAGate, State: featuregate.Alpha})
}

func (g ComputeFeatureGates) CPUsWithDRAGateEnabled() bool {
	return featuregate.GateEnabled(CPUsWithDRAGate, g.ConfigReader)
}
