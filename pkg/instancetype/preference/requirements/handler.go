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
 * Copyright 2024 Red Hat, Inc.
 *
 */
package requirements

import (
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
)

type Handler struct {
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
	preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec
	vmiSpec          *virtv1.VirtualMachineInstanceSpec
}

func New(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) *Handler {
	return &Handler{
		instancetypeSpec: instancetypeSpec,
		preferenceSpec:   preferenceSpec,
		vmiSpec:          vmiSpec,
	}
}

func (h *Handler) Check() (apply.Conflicts, error) {
	if h.preferenceSpec == nil || h.preferenceSpec.Requirements == nil {
		return nil, nil
	}

	if h.preferenceSpec.Requirements.CPU != nil {
		if conflicts, err := h.checkCPU(); err != nil {
			return conflicts, err
		}
	}

	if h.preferenceSpec.Requirements.Memory != nil {
		if conflicts, err := h.checkMemory(); err != nil {
			return conflicts, err
		}
	}

	return nil, nil
}
