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

// Owner: sig-storage / @shellyka13
// Alpha: v1.6.0
//
// IncrementalBackup feature gate enables creating full and incremental backups for virtual machines.
// These backups leverage libvirt's native backup capabilities, providing a storage-agnostic solution.
// To support incremental backups, a QCOW2 overlay must be created on top of the VM's raw disk image.
const IncrementalBackupGate = "IncrementalBackup"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: IncrementalBackupGate, State: featuregate.Alpha})
}

// IncrementalBackupEnabled returns true when the IncrementalBackup feature gate is enabled.
func (g StorageFeatureGates) IncrementalBackupEnabled() bool {
	return featuregate.GateEnabled(IncrementalBackupGate, g.ConfigReader)
}
