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

// Owner: sig-compute / @fossedihelm
// Alpha: v1.9.0
//
// IOMMUFD enables the IOMMUFD device plugin for passthrough devices.
// When enabled, virt-controller requests devices.kubevirt.io/iommufd for
// every launcher pod. Nodes without /dev/iommu (kernel <6.2) will report
// unhealthy devices, making pods unschedulable there.
// This feature also emits domain-level <iommufd/> and uses virDomainFDAssociate,
// which require a libvirt/QEMU stack that supports fdgroup-based IOMMUFD,
// currently libvirt >= 12.2.
const IOMMUFDGate = "IOMMUFD"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: IOMMUFDGate, State: featuregate.Alpha})
}

// IOMMUFDEnabled returns true when the IOMMUFD feature gate is enabled.
func (g ComputeFeatureGates) IOMMUFDEnabled() bool {
	return featuregate.GateEnabled(IOMMUFDGate, g.ConfigReader)
}
