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

// Owner: sig-compute / @awels
// Alpha: v1.10.0
//
// CrossClusterMigrationProxy enables network proxy support in synchronization controllers
// for cross-cluster live migrations. When enabled along with crossClusterNetwork configuration,
// sync controllers attach to both in-cluster and cross-cluster networks and proxy migration
// traffic between them. This reduces IP address requirements on the cross-cluster network
// from N×(M+2) to N×2 addresses (where N=clusters, M=virt-handlers per cluster).
const CrossClusterMigrationProxy = "CrossClusterMigrationProxy"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: CrossClusterMigrationProxy, State: featuregate.Alpha})
}

func (g ComputeFeatureGates) CrossClusterMigrationProxyEnabled() bool {
	return featuregate.GateEnabled(CrossClusterMigrationProxy, g.ConfigReader)
}
