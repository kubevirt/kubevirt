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

// Owner: sig-compute / @lyarwood
// Alpha: v1.10.0
//
// ARM64SecureBoot enables UEFI Secure Boot for ARM64 guests using
// libvirt firmware auto-selection and the QEMU uefi-vars device.
// Requires QEMU 10.0+, libvirt with uefi-vars/varstore support,
// and edk2-aarch64 firmware with Secure Boot templates. These
// components are only available in CentOS Stream 10 based
// virt-launcher images.
const ARM64SecureBoot = "ARM64SecureBoot"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ARM64SecureBoot, State: featuregate.Alpha})
}

func (g ComputeFeatureGates) ARM64SecureBootEnabled() bool {
	return featuregate.GateEnabled(ARM64SecureBoot, g.ConfigReader)
}
