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

package deprecation

import (
	v1 "kubevirt.io/api/core/v1"
)

const MacvtapDiscontinueMessage = "Macvtap network binding is discontinued since v1.3. Please refer to Kubevirt user guide for alternatives."

func macvtapApiUsed(spec *v1.VirtualMachineInstanceSpec) bool {
	for _, net := range spec.Domain.Devices.Interfaces {
		if net.DeprecatedMacvtap != nil {
			return true
		}
	}
	return false
}
