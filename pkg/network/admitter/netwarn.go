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

package admitter

import (
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

// WarnPortsOnNonPodNetwork warns when an interface specifies ports or
// portRanges but its corresponding network is not a pod network. Such
// configuration has no effect today: port-forwarding is only ever applied
// to pod networks, so the ports/portRanges are silently ignored. A future
// KubeVirt release may validate and reject this configuration instead of
// ignoring it.
func WarnPortsOnNonPodNetwork(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []string {
	var warnings []string
	networksByName := vmispec.IndexNetworkSpecByName(spec.Networks)
	for idx, iface := range spec.Domain.Devices.Interfaces {
		network := networksByName[iface.Name]
		if network.Pod != nil {
			continue
		}
		if iface.Ports != nil {
			warnings = append(warnings, fmt.Sprintf(
				"%s: ports have no effect because this interface is not on a pod network, and may be rejected by a future KubeVirt release",
				field.Child("domain", "devices", "interfaces").Index(idx).String(),
			))
		}
		if iface.PortRanges != nil {
			warnings = append(warnings, fmt.Sprintf(
				"%s: portRanges have no effect because this interface is not on a pod network, and may be rejected by a future KubeVirt release",
				field.Child("domain", "devices", "interfaces").Index(idx).String(),
			))
		}
	}
	return warnings
}

// maxRecommendedPorts matches the +kubebuilder:validation:MaxItems limit
// planned for Interface.Ports. It is not yet enforced: Ports has never had a
// count limit before, so enforcing one retroactively could reject an
// existing object with more entries. This warning starts a deprecation
// runway ahead of any future enforcement.
const maxRecommendedPorts = 32

// WarnTooManyPorts warns when an interface's ports list exceeds the number
// of entries a future KubeVirt release may enforce as a hard limit.
func WarnTooManyPorts(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []string {
	var warnings []string
	for idx, iface := range spec.Domain.Devices.Interfaces {
		if len(iface.Ports) > maxRecommendedPorts {
			warnings = append(warnings, fmt.Sprintf(
				"%s: has more than %d ports and may be rejected by a future KubeVirt release",
				field.Child("domain", "devices", "interfaces").Index(idx).String(),
				maxRecommendedPorts,
			))
		}
	}
	return warnings
}
