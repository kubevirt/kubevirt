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

// Owner: @orenc1
// Alpha: v1.8.0
// Beta: v1.9.0
//
// OptOutRoleAggregation enables the RoleAggregationStrategy field in KubeVirtConfiguration,
// allowing users to opt out of aggregating KubeVirt ClusterRoles to the default Kubernetes roles.
const OptOutRoleAggregation = "OptOutRoleAggregation"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: OptOutRoleAggregation, State: featuregate.Beta})
}

// OptOutRoleAggregationEnabled returns true when the OptOutRoleAggregation feature gate is enabled.
func (g LegacyFeatureGates) OptOutRoleAggregationEnabled() bool {
	return featuregate.GateEnabled(OptOutRoleAggregation, g.ConfigReader)
}
