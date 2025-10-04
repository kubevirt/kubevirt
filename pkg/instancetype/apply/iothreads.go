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
package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/pointer"
)

func applyIOThreads(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if instancetypeSpec.IOThreads == nil || instancetypeSpec.IOThreads.SupplementalPoolThreadCount == nil {
		return nil
	}

	if vmiSpec.Domain.IOThreads != nil && vmiSpec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
		return conflict.Conflicts{baseConflict.NewChild("domain", "ioThreads", "supplementalPoolThreadCount")}
	}

	if vmiSpec.Domain.IOThreads == nil {
		vmiSpec.Domain.IOThreads = &virtv1.DiskIOThreads{}
	}

	vmiSpec.Domain.IOThreads.SupplementalPoolThreadCount = pointer.P(*instancetypeSpec.IOThreads.SupplementalPoolThreadCount)

	return nil
}
