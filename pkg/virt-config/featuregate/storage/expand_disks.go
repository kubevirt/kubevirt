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

package storage

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: sig-storage
// Alpha: v0.48.0
// GA: v1.8.0
//
// ExpandDisks allows for expanding the storage available for in-use virtual machines.
const ExpandDisksGate = "ExpandDisks"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ExpandDisksGate, State: featuregate.GA})
}

// ExpandDisksEnabled returns true when the ExpandDisks feature gate is enabled.
func (g StorageFeatureGates) ExpandDisksEnabled() bool {
	return featuregate.GateEnabled(ExpandDisksGate, g.ConfigReader)
}
