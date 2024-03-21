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
