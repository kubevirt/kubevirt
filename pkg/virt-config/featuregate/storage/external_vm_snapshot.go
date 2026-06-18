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

// Owner: @noamasu
// Alpha: v1.10.0
//
// ExternalVMSnapshot feature gate enables online VM snapshots taken with a QEMU
// external snapshot transaction. The guest is only frozen while the qcow2 overlays
// are put in place, and the VolumeSnapshots are then taken of the read-only base images.
const ExternalVMSnapshotGate = "ExternalVMSnapshot"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ExternalVMSnapshotGate, State: featuregate.Alpha})
}

func (g StorageFeatureGates) ExternalVMSnapshotEnabled() bool {
	return featuregate.GateEnabled(ExternalVMSnapshotGate, g.ConfigReader)
}
