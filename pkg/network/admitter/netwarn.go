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
)

// maxRecommendedNTPServers matches the +kubebuilder:validation:MaxItems limit
// planned for DHCPOptions.NTPServers. It is not yet enforced: NTPServers has
// never had a count limit before, so enforcing one retroactively could
// reject an existing object with more entries. This warning starts a
// deprecation runway ahead of any future enforcement.
const maxRecommendedNTPServers = 32

// WarnTooManyNTPServers warns when an interface's DHCP NTP server list
// exceeds the number of entries a future KubeVirt release may enforce as a
// hard limit.
func WarnTooManyNTPServers(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []string {
	var warnings []string
	for idx, iface := range spec.Domain.Devices.Interfaces {
		if iface.DHCPOptions == nil {
			continue
		}
		if len(iface.DHCPOptions.NTPServers) > maxRecommendedNTPServers {
			warnings = append(warnings, fmt.Sprintf(
				"%s: has more than %d DHCP NTP servers and may be rejected by a future KubeVirt release",
				field.Child("domain", "devices", "interfaces").Index(idx).String(),
				maxRecommendedNTPServers,
			))
		}
	}
	return warnings
}
