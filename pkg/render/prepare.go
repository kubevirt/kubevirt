package render

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/storage/memorydump"
	storagevmispec "kubevirt.io/kubevirt/pkg/storage/vmispec"
)

// networkDefaults is what AttachDefaultDevices needs from cluster config.
// *virtconfig.ClusterConfig and the offline config both satisfy it.
type networkDefaults interface {
	GetDefaultNetworkInterface() string
	IsBridgeInterfaceOnPodNetworkEnabled() bool
}

// PrepareControllerVMI is the controller preparation shared with VMIFromVM.
// It drops a completed memory-dump volume and pairs disks. It does not
// apply VM admission defaults, instancetype lookup, start-paused, or
// changed-block tracking. virt-controller must not call VMIFromVM.
func PrepareControllerVMI(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	if memorydump.HasCompleted(vm) {
		vmi.Spec = *memorydump.RemoveMemoryDumpVolumeFromVMISpec(&vmi.Spec, vm.Status.MemoryDumpRequest.ClaimName)
	}
	storagevmispec.SetDefaultVolumeDisk(&vmi.Spec)
}

// AttachDefaultDevices adds the default input and pod interface. Call it
// after auto-attach preferences so those preferences are already visible.
func AttachDefaultDevices(vmi *v1.VirtualMachineInstance, cfg networkDefaults) error {
	AutoAttachInputDevice(vmi)
	if err := netvmispec.SetDefaultNetworkInterface(cfg, &vmi.Spec); err != nil {
		return fmt.Errorf("failed to set default network interface: %w", err)
	}
	return nil
}
