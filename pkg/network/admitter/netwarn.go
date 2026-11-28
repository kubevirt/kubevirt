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

// recommendedMaxNameLength matches the DNS_LABEL length limit already
// documented on Network.Name and Interface.Name. It is not yet enforced:
// retroactively enforcing it on these old, previously unbounded fields would
// be unsafe (see docs/api/design_guidelines.md §4.2.3). This warning starts a
// deprecation runway ahead of any future enforcement.
const recommendedMaxNameLength = 63

func WarnLongNetworkNames(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []string {
	var warnings []string
	for idx, network := range spec.Networks {
		if len(network.Name) > recommendedMaxNameLength {
			warnings = append(warnings, fmt.Sprintf(
				"%s is longer than %d characters and may be rejected by a future KubeVirt release",
				field.Child("networks").Index(idx).Child("name").String(), recommendedMaxNameLength,
			))
		}
	}
	return warnings
}

func WarnLongInterfaceNames(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []string {
	var warnings []string
	for idx, iface := range spec.Domain.Devices.Interfaces {
		if len(iface.Name) > recommendedMaxNameLength {
			warnings = append(warnings, fmt.Sprintf(
				"%s is longer than %d characters and may be rejected by a future KubeVirt release",
				field.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(), recommendedMaxNameLength,
			))
		}
	}
	return warnings
}
