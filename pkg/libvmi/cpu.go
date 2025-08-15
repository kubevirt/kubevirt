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

package libvmi

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

func WithCPUCount(cores, threads, sockets uint32) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.Cores = cores
		vmi.Spec.Domain.CPU.Threads = threads
		vmi.Spec.Domain.CPU.Sockets = sockets
	}
}

func WithCPUModel(model string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.Model = model
	}
}

func WithCPUFeature(featureName, policy string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}

		vmi.Spec.Domain.CPU.Features = append(vmi.Spec.Domain.CPU.Features, v1.CPUFeature{
			Name:   featureName,
			Policy: policy,
		})
	}
}

func WithDedicatedCPUPlacement() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
	}
}

func WithRealtimeMask(realtimeMask string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.Realtime = &v1.Realtime{Mask: realtimeMask}
	}
}

func WithNUMAGuestMappingPassthrough() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.NUMA = &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}}
	}
}

func WithArchitecture(arch string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Architecture = arch
	}
}

// WithResourceCPU specifies the vmi CPU resource.
func WithResourceCPU(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

// WithLimitCPU specifies the VMI CPU limit.
func WithLimitCPU(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Limits == nil {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

// WithCPURequest specifies the vmi CPU resource.
func WithCPURequest(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

// WithCPULimit specifies the VMI CPU limit.
func WithCPULimit(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Limits == nil {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

func WithIsolateEmulatorThread() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.IsolateEmulatorThread = true
	}
}
