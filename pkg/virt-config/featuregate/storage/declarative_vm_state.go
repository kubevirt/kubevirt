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
// Alpha: v1.10.0
//
// DeclarativeVMState enables the declarative virtualMachineState API field,
// which references or templates the PVC that backs the VirtualMachineState
// (persistent UEFI, TPM and CBT state). The gate controls API admission only:
// VMs already created with the field keep using the declarative path even if
// the gate is later disabled.
const DeclarativeVMState = "DeclarativeVMState"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: DeclarativeVMState, State: featuregate.Alpha})
}

func (g StorageFeatureGates) DeclarativeVMStateEnabled() bool {
	return featuregate.GateEnabled(DeclarativeVMState, g.ConfigReader)
}
