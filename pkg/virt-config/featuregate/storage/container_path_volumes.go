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

// Owner: sig-storage / @mhenriks
// Alpha: v1.8.0
//
// ContainerPathVolumes enables exposing virt-launcher volumeMount paths to the VM
// via virtiofs. This allows VMs to access credentials and tokens injected into pods
// by external systems such as AWS IRSA, GKE Workload Identity, or TEE attestation.
const ContainerPathVolumesGate = "ContainerPathVolumes"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ContainerPathVolumesGate, State: featuregate.Alpha})
}

// ContainerPathVolumesEnabled returns true when the ContainerPathVolumes feature gate is enabled.
func (g StorageFeatureGates) ContainerPathVolumesEnabled() bool {
	return featuregate.GateEnabled(ContainerPathVolumesGate, g.ConfigReader)
}
