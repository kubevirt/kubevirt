package vgpuhook

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirtxml"
)

// VGPUDedicatedHook mutates the mdev uuid for the target's domain XML in vGPU live migrations
func VGPUDedicatedHook(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if len(vmi.Spec.Domain.Devices.GPUs) == 0 {
		return nil
	}
	if len(vmi.Spec.Domain.Devices.GPUs) != 1 {
		return fmt.Errorf("the migrating vmi can only have one vGPU")
	}
	if len(vmi.Spec.Domain.Devices.HostDevices) != 0 {
		return fmt.Errorf("the migrating vmi cannot have any non vGPU hostdevices")
	}

	mdevUUID, ok := vmi.Annotations["kubevirt.io/target-mdev-uuid"]
	if !ok {
		return fmt.Errorf("missing vmi annotation target-mdev-uuid")
	}

	// need to check for type=mdev so we don't try to migrate a passthrough GPU
	if len(domain.Devices.Hostdevs) == 1 && domain.Devices.Hostdevs[0].SubsysMDev != nil {
		domain.Devices.Hostdevs[0].SubsysMDev.Source.Address.UUID = mdevUUID
	} else {
		return fmt.Errorf("failed to retrieve mdev vGPU from domain")
	}

	log.Log.Object(vmi).Info("vGPU-hook: mdev uuid mutation completed")
	return nil
}
