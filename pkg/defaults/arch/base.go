package arch_defaults

import (
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type ArchDefaults interface {
	SetArchDefaults(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance)
}

type BaseArchDefaults struct {
	hotplugSetter           func(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance)
	defaultFeaturesSetter   func(spec *v1.VirtualMachineInstanceSpec)
	defaultDisksBusSetter   func(spec *v1.VirtualMachineInstanceSpec)
	defaultCPUModelSetter   func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultBootloaderSetter func(spec *v1.VirtualMachineInstanceSpec)
	defaultWatchdogSetter   func(spec *v1.VirtualMachineInstanceSpec)
}

func (b *BaseArchDefaults) SetArchDefaults(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	b.hotplugSetter(clusterConfig, vmi)
	b.defaultFeaturesSetter(&vmi.Spec)
	b.defaultDisksBusSetter(&vmi.Spec)
	b.defaultCPUModelSetter(clusterConfig, &vmi.Spec)
	b.defaultBootloaderSetter(&vmi.Spec)
	b.defaultWatchdogSetter(&vmi.Spec)
}

func NewBaseArchDefaults() *BaseArchDefaults {
	return &BaseArchDefaults{
		hotplugSetter:           setupHotplug,
		defaultFeaturesSetter:   func(spec *v1.VirtualMachineInstanceSpec) {},
		defaultDisksBusSetter:   func(spec *v1.VirtualMachineInstanceSpec) {},
		defaultCPUModelSetter:   setDefaultCPUModel,
		defaultBootloaderSetter: func(spec *v1.VirtualMachineInstanceSpec) {},
		defaultWatchdogSetter:   func(spec *v1.VirtualMachineInstanceSpec) {},
	}
}

func setupHotplug(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	if !clusterConfig.IsVMRolloutStrategyLiveUpdate() {
		return
	}
	setupCPUHotplug(clusterConfig, vmi)
	setupMemoryHotplug(clusterConfig, vmi)
}

func setupCPUHotplug(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU == nil {
		return
	}

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
	if vmi.Spec.Domain.Memory == nil || vmi.Spec.Domain.Memory.MaxGuest != nil {
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
