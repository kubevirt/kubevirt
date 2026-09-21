/* Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright the KubeVirt Authors.
 *
 */
package defaults

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

// Disable ACPI Feature by default on ppc64le, since pseries machine types do not support it
func setPPC64LEDefaultFeatures(spec *v1.VirtualMachineInstanceSpec) {
	featureStateDisabled := v1.FeatureState{Enabled: pointer.P[bool](false)}
	if spec.Domain.Features == nil {
		spec.Domain.Features = &v1.Features{
			ACPI: featureStateDisabled,
		}
	} else if spec.Domain.Features.ACPI.Enabled == nil {
		spec.Domain.Features.ACPI.Enabled = pointer.P[bool](false)
	}
}

func IsPPC64LE(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Architecture == "ppc64le"
}
