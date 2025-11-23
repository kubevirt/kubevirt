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
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Defaults interface {
	SetVirtualMachineDefaults(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient) error
	SetDefaultVirtualMachineInstance(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) error
	SetDefaultVirtualMachineInstanceSpec(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error
}

func SetVirtualMachineDefaults(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient) {
	setDefaultArchitectureFromDataSource(clusterConfig, vm, virtClient)
	setDefaultArchitecture(clusterConfig, &vm.Spec.Template.Spec)
	setVMDefaultMachineType(vm, clusterConfig)

	vmispec.SetDefaultNetworkInterface(clusterConfig, &vm.Spec.Template.Spec)

}

func setupHotplug(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	if !clusterConfig.IsVMRolloutStrategyLiveUpdate() {
		return
	}
	setupCPUHotplug(clusterConfig, vmi)
	setupMemoryHotplug(clusterConfig, vmi)
}

func setupCPUHotplug(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU.MaxSockets == 0 {
		maxSockets := clusterConfig.GetMaximumCpuSockets()
		if vmi.Spec.Domain.CPU.Sockets > maxSockets && maxSockets != 0 {
			maxSockets = vmi.Spec.Domain.CPU.Sockets
		}
		vmi.Spec.Domain.CPU.MaxSockets = maxSockets
	}

	if vmi.Spec.Domain.CPU.MaxSockets == 0 {
		// Each machine type will have different maximum for vcpus,
		// lets choose 512 as upper bound
		const maxVCPUs = 512

		vmi.Spec.Domain.CPU.MaxSockets = vmi.Spec.Domain.CPU.Sockets * clusterConfig.GetMaxHotplugRatio()
		totalVCPUs := vmi.Spec.Domain.CPU.MaxSockets * vmi.Spec.Domain.CPU.Cores * vmi.Spec.Domain.CPU.Threads
		if totalVCPUs > maxVCPUs {
			adjustedSockets := maxVCPUs / (vmi.Spec.Domain.CPU.Cores * vmi.Spec.Domain.CPU.Threads)
			vmi.Spec.Domain.CPU.MaxSockets = max(adjustedSockets, vmi.Spec.Domain.CPU.Sockets)
		}
	}
}

func setupMemoryHotplug(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Memory.MaxGuest != nil {
		return
	}

	var maxGuest *resource.Quantity
	switch {
	case clusterConfig.GetMaximumGuestMemory() != nil:
		maxGuest = clusterConfig.GetMaximumGuestMemory()
	case vmi.Spec.Domain.Memory.Guest != nil:
		maxGuest = resource.NewQuantity(vmi.Spec.Domain.Memory.Guest.Value()*int64(clusterConfig.GetMaxHotplugRatio()), resource.BinarySI)
	}

	if err := memory.ValidateLiveUpdateMemory(&vmi.Spec, maxGuest); err != nil {
		// memory hotplug is not compatible with this VM configuration
		log.Log.V(2).Object(vmi).Infof("memory-hotplug disabled: %s", err)
		return
	}

	vmi.Spec.Domain.Memory.MaxGuest = maxGuest
}

func setDefaultCPUArch(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	// Do some CPU arch specific setting.
	switch {
	case IsARM64(spec):
		log.Log.V(4).Info("Apply Arm64 specific setting")
		SetArm64Defaults(spec)
	case IsS390X(spec):
		log.Log.V(4).Info("Apply s390x specific setting")
		SetS390xDefaults(spec)
	default:
		SetAmd64Defaults(spec)
	}
	setDefaultCPUModel(clusterConfig, spec)
}
