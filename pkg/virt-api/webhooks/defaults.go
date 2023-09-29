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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package webhooks

import (
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func SetDefaultVirtualMachine(clusterConfig *virtconfig.ClusterConfig, vm *v1.VirtualMachine) error {
	if err := setDefaultVirtualMachineInstanceSpec(clusterConfig, &vm.Spec.Template.Spec); err != nil {
		return err
	}
	v1.SetObjectDefaults_VirtualMachine(vm)
	setDefaultHypervFeatureDependencies(&vm.Spec.Template.Spec)
	setDefaultCPUArch(clusterConfig, &vm.Spec.Template.Spec)
	return nil
}

func SetDefaultVirtualMachineInstance(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) error {
	if err := setDefaultVirtualMachineInstanceSpec(clusterConfig, &vmi.Spec); err != nil {
		return err
	}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	setDefaultHypervFeatureDependencies(&vmi.Spec)
	setDefaultCPUArch(clusterConfig, &vmi.Spec)
	setGuestMemoryStatus(vmi)
	return nil
}

func setGuestMemoryStatus(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Memory != nil &&
		vmi.Spec.Domain.Memory.Guest != nil {
		vmi.Status.Memory = &v1.MemoryStatus{
			GuestAtBoot:    vmi.Spec.Domain.Memory.Guest,
			GuestCurrent:   vmi.Spec.Domain.Memory.Guest,
			GuestRequested: vmi.Spec.Domain.Memory.Guest,
		}
	}
}

func setDefaultCPUArch(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	// Do some CPU arch specific setting.
	if IsARM64(spec) {
		log.Log.V(4).Info("Apply Arm64 specific setting")
		SetArm64Defaults(spec)
	} else {
		SetAmd64Defaults(spec)
		setDefaultCPUModel(clusterConfig, spec)
	}
}

func setDefaultHypervFeatureDependencies(spec *v1.VirtualMachineInstanceSpec) {
	// In a future, yet undecided, release either libvirt or QEMU are going to check the hyperv dependencies, so we can get rid of this code.
	// Until that time, we need to handle the hyperv deps to avoid obscure rejections from QEMU later on
	log.Log.V(4).Info("Set HyperV dependencies")
	if err := SetHypervFeatureDependencies(spec); err != nil {
		// HyperV is a special case. If our best-effort attempt fails, we should leave
		// rejection to be performed later on in the validating webhook, and continue here.
		// Please note this means that partial changes may have been performed.
		// This is OK since each dependency must be atomic and independent (in ACID sense),
		// so the VMI configuration is still legal.
		log.Log.V(2).Infof("Failed to set HyperV dependencies: %s", err)
	}
}

func setDefaultVirtualMachineInstanceSpec(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error {
	setDefaultArchitecture(clusterConfig, spec)
	setDefaultMachineType(clusterConfig, spec)
	setDefaultResourceRequests(clusterConfig, spec)
	SetDefaultGuestCPUTopology(clusterConfig, spec)
	setDefaultPullPoliciesOnContainerDisks(clusterConfig, spec)
	setDefaultEvictionStrategy(clusterConfig, spec)
	if err := clusterConfig.SetVMISpecDefaultNetworkInterface(spec); err != nil {
		return err
	}
	util.SetDefaultVolumeDisk(spec)
	return nil
}

func setDefaultEvictionStrategy(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	if spec.EvictionStrategy == nil {
		spec.EvictionStrategy = clusterConfig.GetConfig().EvictionStrategy
	}
}

func setDefaultMachineType(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	machineType := clusterConfig.GetMachineType(spec.Architecture)

	if machine := spec.Domain.Machine; machine != nil {
		if machine.Type == "" {
			machine.Type = machineType
		}
	} else {
		spec.Domain.Machine = &v1.Machine{Type: machineType}
	}

}

func setDefaultPullPoliciesOnContainerDisks(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	for _, volume := range spec.Volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.ImagePullPolicy == "" {
			if strings.HasSuffix(volume.ContainerDisk.Image, ":latest") || !strings.ContainsAny(volume.ContainerDisk.Image, ":@") {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullAlways
			} else {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullIfNotPresent
			}
		}
	}
}

func setDefaultResourceRequests(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	resources := &spec.Domain.Resources

	if !resources.Limits.Cpu().IsZero() && resources.Requests.Cpu().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	if !resources.Limits.Memory().IsZero() && resources.Requests.Memory().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	}

	if _, exists := resources.Requests[k8sv1.ResourceMemory]; !exists {
		var memory *resource.Quantity
		if spec.Domain.Memory != nil && spec.Domain.Memory.Guest != nil {
			memory = spec.Domain.Memory.Guest
		}
		if memory == nil && spec.Domain.Memory != nil && spec.Domain.Memory.Hugepages != nil {
			if hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize); err == nil {
				memory = &hugepagesSize
			}
		}
		if memory != nil && memory.Value() > 0 {
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			overcommit := clusterConfig.GetMemoryOvercommit()
			if overcommit == 100 {
				resources.Requests[k8sv1.ResourceMemory] = *memory
			} else {
				value := (memory.Value() * int64(100)) / int64(overcommit)
				resources.Requests[k8sv1.ResourceMemory] = *resource.NewQuantity(value, memory.Format)
			}
			memoryRequest := resources.Requests[k8sv1.ResourceMemory]
			log.Log.V(4).Infof("Set memory-request to %s as a result of memory-overcommit = %v%%", memoryRequest.String(), overcommit)
		}
	}
	if cpuRequest := clusterConfig.GetCPURequest(); !cpuRequest.Equal(resource.MustParse(virtconfig.DefaultCPURequest)) {
		if _, exists := resources.Requests[k8sv1.ResourceCPU]; !exists {
			if spec.Domain.CPU != nil && spec.Domain.CPU.DedicatedCPUPlacement {
				return
			}
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			resources.Requests[k8sv1.ResourceCPU] = *cpuRequest
		}
	}
}

func SetDefaultGuestCPUTopology(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := spec.Domain.CPU
	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		// create cpu topology struct
		if spec.Domain.CPU == nil {
			spec.Domain.CPU = &v1.CPU{}
		}
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}

		spec.Domain.CPU.Sockets = sockets
		spec.Domain.CPU.Cores = cores
		spec.Domain.CPU.Threads = threads
	}
}

func setDefaultCPUModel(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	// create cpu topology struct
	if spec.Domain.CPU == nil {
		spec.Domain.CPU = &v1.CPU{}
	}

	// if vmi doesn't have cpu model set
	if spec.Domain.CPU.Model == "" {
		if clusterConfigCPUModel := clusterConfig.GetCPUModel(); clusterConfigCPUModel != "" {
			//set is as vmi cpu model
			spec.Domain.CPU.Model = clusterConfigCPUModel
		} else {
			spec.Domain.CPU.Model = v1.DefaultCPUModel
		}
	}
}

func setDefaultArchitecture(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	if spec.Architecture == "" {
		spec.Architecture = clusterConfig.GetDefaultArchitecture()
	}
}
