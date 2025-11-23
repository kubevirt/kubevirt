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

package defaults

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	arch_defaults "kubevirt.io/kubevirt/pkg/defaults/arch"
	base_defaults "kubevirt.io/kubevirt/pkg/defaults/base"
	kvm_defaults "kubevirt.io/kubevirt/pkg/defaults/kvm"
	mshv_defaults "kubevirt.io/kubevirt/pkg/defaults/mshv"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Defaults interface {
	SetVirtualMachineDefaults(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient)
	SetDefaultVirtualMachineInstance(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) error
	SetDefaultVirtualMachineInstanceSpec(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error
}

func NewDefault(clusterConfig *virtconfig.ClusterConfig) Defaults {
	switch clusterConfig.GetHypervisor().Name {
	case v1.HyperVLayeredHypervisorName:
		mshvDefaults := mshv_defaults.MSHVDefaults{
			BaseDefaults: *base_defaults.NewBaseDefaults(
				arch_defaults.NewAmd64ArchDefaults(),
				arch_defaults.NewArm64ArchDefaults(),
				arch_defaults.NewS390xArchDefaults(),
			),
		}
		return &mshvDefaults
	default:
		return &kvm_defaults.KVMDefaults{
			BaseDefaults: *base_defaults.NewBaseDefaults(
				arch_defaults.NewAmd64ArchDefaults(),
				arch_defaults.NewArm64ArchDefaults(),
				arch_defaults.NewS390xArchDefaults(),
			),
		}
	}
}
