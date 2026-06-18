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

import "kubevirt.io/kubevirt/pkg/virt-config/featuregate"

// VirtIOFSGate enables the use of virtiofs for config and storage volumes.
// Discontinued in v1.7.0
const VirtIOFSGate = "ExperimentalVirtiofsSupport"

const VirtioFsFeatureGateDiscontinueMessage = "Virtiofs ExperimentalVirtiofsSupport feature gate is discontinued since v1.7. Please use EnableVirtioFsConfigVolumes or EnableVirtioFsPVC feature gates instead"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:    VirtIOFSGate,
		State:   featuregate.Discontinued,
		Message: VirtioFsFeatureGateDiscontinueMessage,
	})
}

// VirtIOFSEnabled returns true when the ExperimentalVirtiofsSupport feature gate is enabled.
func (g StorageFeatureGates) VirtIOFSEnabled() bool {
	return featuregate.GateEnabled(VirtIOFSGate, g.ConfigReader)
}
