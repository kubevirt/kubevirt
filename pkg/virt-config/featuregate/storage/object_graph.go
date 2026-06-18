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

// Owner: sig-storage / @alromeros
// Alpha: v1.6.0
//
// ObjectGraph introduces a new subresource for VMs and VMIs.
// This subresource returns a structured list of k8s objects that are related
// to the specified VM or VMI, enabling better dependency tracking.
const ObjectGraph = "ObjectGraph"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ObjectGraph, State: featuregate.Alpha})
}

// ObjectGraphEnabled returns true when the ObjectGraph feature gate is enabled.
func (g StorageFeatureGates) ObjectGraphEnabled() bool {
	return featuregate.GateEnabled(ObjectGraph, g.ConfigReader)
}
