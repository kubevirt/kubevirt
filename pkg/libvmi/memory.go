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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libvmi

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

func WithHugepages(pageSize string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Memory == nil {
			vmi.Spec.Domain.Memory = &v1.Memory{}
		}
		vmi.Spec.Domain.Memory.Hugepages = &v1.Hugepages{PageSize: pageSize}
	}
}

func WithGuestMemory(memory string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Memory == nil {
			vmi.Spec.Domain.Memory = &v1.Memory{}
		}
		quantity := resource.MustParse(memory)
		vmi.Spec.Domain.Memory.Guest = &quantity
	}
}

func WithMaxGuest(memory string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Memory == nil {
			vmi.Spec.Domain.Memory = &v1.Memory{}
		}
		quantity := resource.MustParse(memory)
		vmi.Spec.Domain.Memory.MaxGuest = &quantity
	}
}

// WithResourceMemory specifies the vmi memory resource.
func WithResourceMemory(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(value)
	}
}

// WithLimitMemory specifies the VMI memory limit.
func WithLimitMemory(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Limits == nil {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = resource.MustParse(value)
	}
}
