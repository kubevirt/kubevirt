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

package network

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: SIG network
// Alpha: v1.9.0
//
// NetworkDevicesWithDRAGate allows users to create VMIs with DRA provisioned Network devices
// specified in spec.networks with resourceClaim type. This enables DRA-managed network
// resources to be attached to VMs using the natural networks API.
const NetworkDevicesWithDRAGate = "NetworkDevicesWithDRA"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: NetworkDevicesWithDRAGate, State: featuregate.Alpha})
}

// NetworkDevicesWithDRAGateEnabled returns true when the NetworkDevicesWithDRA feature gate is enabled.
func (g NetworkFeatureGates) NetworkDevicesWithDRAGateEnabled() bool {
	return featuregate.GateEnabled(NetworkDevicesWithDRAGate, g.ConfigReader)
}
