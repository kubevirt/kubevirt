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
 * Copyright The KubeVirt Authors
 *
 */
package validation

import (
	"slices"

	"kubevirt.io/api/instancetype/v1beta1"
)

func IsPreferredTopologySupported(topology v1beta1.PreferredCPUTopology) bool {
	supportedTopologies := []v1beta1.PreferredCPUTopology{
		v1beta1.DeprecatedPreferSockets,
		v1beta1.DeprecatedPreferCores,
		v1beta1.DeprecatedPreferThreads,
		v1beta1.DeprecatedPreferSpread,
		v1beta1.DeprecatedPreferAny,
		v1beta1.Sockets,
		v1beta1.Cores,
		v1beta1.Threads,
		v1beta1.Spread,
		v1beta1.Any,
	}
	return slices.Contains(supportedTopologies, topology)
}
