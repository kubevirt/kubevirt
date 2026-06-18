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

// Owner: sig-compute / @enp0s3
// Alpha: v1.9.0
//
// VMStatsCollector enables the additional guest agent polling workers
// (frequent/medium/infrequent tiers) that collect raw monitoring data
// for the GetVMStats gRPC RPC.
const VMStatsCollector = "VMStatsCollector"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: VMStatsCollector, State: featuregate.Alpha})
}

// VMStatsCollectorEnabled returns true when the VMStatsCollector feature gate is enabled.
func (g ComputeFeatureGates) VMStatsCollectorEnabled() bool {
	return featuregate.GateEnabled(VMStatsCollector, g.ConfigReader)
}
