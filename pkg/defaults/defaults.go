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

package defaults

import (
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func SetVirtualMachineDefaults(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) {
	setDefaultInstancetypeKind(vm)
	setDefaultPreferenceKind(vm)
	setDefaultArchitecture(clusterConfig, &vm.Spec.Template.Spec)
	setVMDefaultMachineType(vm, preferenceSpec, clusterConfig)
	setPreferenceStorageClassName(vm, preferenceSpec)
}

func setVMDefaultMachineType(vm *v1.VirtualMachine, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, clusterConfig *virtconfig.ClusterConfig) {
	// Nothing to do, let's the validating webhook fail later
	if vm.Spec.Template == nil {
		return
	}

	if machine := vm.Spec.Template.Spec.Domain.Machine; machine != nil && machine.Type != "" {
		return
	}

	if vm.Spec.Template.Spec.Domain.Machine == nil {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{}
	}

	if preferenceSpec != nil && preferenceSpec.Machine != nil {
		vm.Spec.Template.Spec.Domain.Machine.Type = preferenceSpec.Machine.PreferredMachineType
	}

	// Only use the cluster default if the user hasn't provided a machine type or referenced a preference with PreferredMachineType
	if vm.Spec.Template.Spec.Domain.Machine.Type == "" {
		vm.Spec.Template.Spec.Domain.Machine.Type = clusterConfig.GetMachineType(vm.Spec.Template.Spec.Architecture)
	}
}

func setPreferenceStorageClassName(vm *v1.VirtualMachine, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) {
	// Nothing to do, let's the validating webhook fail later
	if vm.Spec.Template == nil {
		return
	}

	if preferenceSpec != nil && preferenceSpec.Volumes != nil && preferenceSpec.Volumes.PreferredStorageClassName != "" {
		for _, dv := range vm.Spec.DataVolumeTemplates {
			if dv.Spec.PVC != nil && dv.Spec.PVC.StorageClassName == nil {
				dv.Spec.PVC.StorageClassName = &preferenceSpec.Volumes.PreferredStorageClassName
			}
			if dv.Spec.Storage != nil && dv.Spec.Storage.StorageClassName == nil {
				dv.Spec.Storage.StorageClassName = &preferenceSpec.Volumes.PreferredStorageClassName
			}
		}
	}
}

func setDefaultInstancetypeKind(vm *v1.VirtualMachine) {
	if vm.Spec.Instancetype == nil {
		return
	}

	if vm.Spec.Instancetype.Kind == "" {
		vm.Spec.Instancetype.Kind = apiinstancetype.ClusterSingularResourceName
	}
}

func setDefaultPreferenceKind(vm *v1.VirtualMachine) {
	if vm.Spec.Preference == nil {
		return
	}

	if vm.Spec.Preference.Kind == "" {
		vm.Spec.Preference.Kind = apiinstancetype.ClusterSingularPreferenceResourceName
	}
}

func SetDefaultVirtualMachineInstance(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) error {
	if err := SetDefaultVirtualMachineInstanceSpec(clusterConfig, &vmi.Spec); err != nil {
		return err
	}
	setDefaultFeatures(&vmi.Spec)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	setDefaultHypervFeatureDependencies(&vmi.Spec)
	setDefaultCPUArch(clusterConfig, &vmi.Spec)
	setGuestMemoryStatus(vmi)
	setCurrentCPUTopologyStatus(vmi)
	setupHotplug(clusterConfig, vmi)
	return nil
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
		vmi.Spec.Domain.CPU.MaxSockets = vmi.Spec.Domain.CPU.Sockets * clusterConfig.GetMaxHotplugRatio()
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

func setCurrentCPUTopologyStatus(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU != nil && vmi.Status.CurrentCPUTopology == nil {
		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{
			Sockets: vmi.Spec.Domain.CPU.Sockets,
			Cores:   vmi.Spec.Domain.CPU.Cores,
			Threads: vmi.Spec.Domain.CPU.Threads,
		}
	}
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

func setDefaultFeatures(spec *v1.VirtualMachineInstanceSpec) {
	if IsS390X(spec) {
		setS390xDefaultFeatures(spec)
	}
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

func SetDefaultVirtualMachineInstanceSpec(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error {
	setDefaultArchitecture(clusterConfig, spec)
	setDefaultMachineType(clusterConfig, spec)
	setDefaultResourceRequests(clusterConfig, spec)
	setGuestMemory(spec)
	SetDefaultGuestCPUTopology(clusterConfig, spec)
	setDefaultPullPoliciesOnContainerDisks(spec)
	setDefaultEvictionStrategy(clusterConfig, spec)
	if err := vmispec.SetDefaultNetworkInterface(clusterConfig, spec); err != nil {
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

func setDefaultPullPoliciesOnContainerDisks(spec *v1.VirtualMachineInstanceSpec) {
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

func setGuestMemory(spec *v1.VirtualMachineInstanceSpec) {
	if spec.Domain.Memory != nil &&
		spec.Domain.Memory.Guest != nil {
		return
	}

	if spec.Domain.Memory == nil {
		spec.Domain.Memory = &v1.Memory{}
	}

	switch {
	case !spec.Domain.Resources.Requests.Memory().IsZero():
		spec.Domain.Memory.Guest = spec.Domain.Resources.Requests.Memory()
	case !spec.Domain.Resources.Limits.Memory().IsZero():
		spec.Domain.Memory.Guest = spec.Domain.Resources.Limits.Memory()
	case spec.Domain.Memory.Hugepages != nil:
		if hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize); err == nil {
			spec.Domain.Memory.Guest = &hugepagesSize
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
