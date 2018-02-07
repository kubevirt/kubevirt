package watch

import (
	"fmt"
	"reflect"

	"k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type VirtualMachineInitializer struct {
	vmPresetInformer cache.SharedIndexInformer
	clientset        kubecli.KubevirtClient
}

const initializerMarking = "presets.virtualmachines.kubevirt.io"

// FIXME: Both the restClient and clientSet are probably not needed.
func NewVirtualMachineInitializer(vmPresetInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient) *VirtualMachineInitializer {
	vmi := VirtualMachineInitializer{
		vmPresetInformer: vmPresetInformer,
		clientset:        clientset,
	}
	return &vmi
}

func (c *VirtualMachineInitializer) initializeVirtualMachine(vm *kubev1.VirtualMachine) error {
	// All VM's must be marked as initialized or they are held in limbo forever
	// Collect all errors and defer returning until after the update
	logger := log.Log
	errors := []error{}

	logger.Object(vm).Info("Initializing VirtualMachine")

	allPresets, err := listPresets(c.vmPresetInformer, vm.GetNamespace())

	matchingPresets, err := filterPresets(allPresets, vm)

	if err != nil {
		logger.Object(vm).Reason(err).Errorf("Error while matching presets to VirtualMachine")
		errors = append(errors, err)
	}

	if len(matchingPresets) != 0 {
		err = applyPresets(vm, matchingPresets)
		if err != nil {
			// A more specific error should have been logged during the applyPresets call.
			// We don't know *which* preset in the list was problematic at this level.
			logger.Object(vm).Errorf("Unable to apply presets to virtualmachine: %v", err)
			errors = append(errors, err)
		}
	}

	logger.Object(vm).Info("Marking VM as initialized and updating")
	removeInitializer(vm)
	_, err = c.clientset.VM(vm.Namespace).Update(vm)
	if err != nil {
		logger.Object(vm).Errorf("Could not update VM. VM will not appear in the cluster: %v", err)
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

// FIXME: There is probably a way to set up the vmPresetInformer such that
// items are already partitioned into namespaces (and can just be listed)
func listPresets(vmPresetInformer cache.SharedIndexInformer, namespace string) ([]kubev1.VirtualMachinePreset, error) {
	result := []kubev1.VirtualMachinePreset{}
	for _, obj := range vmPresetInformer.GetStore().List() {
		preset := kubev1.VirtualMachinePreset{}
		obj.(*kubev1.VirtualMachinePreset).DeepCopyInto(&preset)
		if preset.Namespace == namespace {
			result = append(result, preset)
		}
	}
	return result, nil
}

// filterPresets returns list of VirtualMachinePresets which match given VirtualMachine.
func filterPresets(list []kubev1.VirtualMachinePreset, vm *kubev1.VirtualMachine) ([]kubev1.VirtualMachinePreset, error) {
	var matchingPresets []kubev1.VirtualMachinePreset

	logger := log.Log

	for _, preset := range list {
		selector, err := k8smetav1.LabelSelectorAsSelector(&preset.Spec.Selector)
		if err != nil {
			logger.Reason(err).Errorf("label selector conversion failed: %v for selector: %v", preset.Spec.Selector, err)
			return nil, fmt.Errorf("label selector conversion failed: %v for selector: %v", preset.Spec.Selector, err)
		}

		// check if the pod labels match the selector
		if !selector.Matches(labels.Set(vm.Labels)) {
			continue
		}
		logger.Infof("VirtualMachinePreset %s matches labels of VM %s", preset.GetName(), vm.GetName())
		matchingPresets = append(matchingPresets, preset)
	}
	return matchingPresets, nil
}

func diskDeviceToDeviceName(dev kubev1.DiskDevice) string {
	if dev.Disk != nil {
		return dev.Disk.Device
	}
	if dev.LUN != nil {
		return dev.LUN.Device
	}
	if dev.Floppy != nil {
		return dev.Floppy.Device
	}
	if dev.CDRom != nil {
		return dev.CDRom.Device
	}
	return ""
}

func checkPresetMergeConflicts(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
	errors := []error{}
	if len(presetSpec.Resources.Requests) > 0 {
		for key, presetReq := range presetSpec.Resources.Requests {
			if vmReq, ok := vmSpec.Resources.Requests[key]; ok {
				if presetReq != vmReq {
					errors = append(errors, fmt.Errorf("spec.resources.requests[%s]: %v != %v", key, presetReq, vmReq))
				}
			}
		}
	}
	if presetSpec.CPU != nil && vmSpec.CPU != nil {
		if !reflect.DeepEqual(presetSpec.CPU, vmSpec.CPU) {
			errors = append(errors, fmt.Errorf("spec.cpu: %v != %v", presetSpec.CPU, vmSpec.CPU))
		}
	}
	if presetSpec.Firmware != nil && vmSpec.Firmware != nil {
		if !reflect.DeepEqual(presetSpec.Firmware, vmSpec.Firmware) {
			errors = append(errors, fmt.Errorf("spec.firmware: %v != %v", presetSpec.Firmware, vmSpec.Firmware))
		}
	}
	if presetSpec.Clock != nil && vmSpec.Clock != nil {
		if !reflect.DeepEqual(presetSpec.Clock.ClockOffset, vmSpec.Clock.ClockOffset) {
			errors = append(errors, fmt.Errorf("spec.clock.clockoffset: %v != %v", presetSpec.Clock.ClockOffset, vmSpec.Clock.ClockOffset))
		}
		if presetSpec.Clock.Timer != nil && vmSpec.Clock.Timer != nil {
			if !reflect.DeepEqual(presetSpec.Clock.Timer, vmSpec.Clock.Timer) {
				errors = append(errors, fmt.Errorf("spec.clock.timer: %v != %v", presetSpec.Clock.Timer, vmSpec.Clock.Timer))
			}
		}
	}
	if presetSpec.Features != nil && vmSpec.Features != nil {
		if !reflect.DeepEqual(presetSpec.Features, vmSpec.Features) {
			errors = append(errors, fmt.Errorf("spec.features: %v != %v", presetSpec.Features, vmSpec.Features))
		}
	}
	if presetSpec.Devices.Watchdog != nil && vmSpec.Devices.Watchdog != nil {
		if !reflect.DeepEqual(presetSpec.Devices.Watchdog, vmSpec.Devices.Watchdog) {
			errors = append(errors, fmt.Errorf("spec.devices.watchdog: %v != %v", presetSpec.Devices.Watchdog, vmSpec.Devices.Watchdog))
		}
	}
	nameMap := make(map[string]kubev1.Disk)
	volumeNameMap := make(map[string]kubev1.Disk)
	diskDeviceMap := make(map[string]kubev1.Disk)

	for _, vmDev := range vmSpec.Devices.Disks {
		nameMap[vmDev.Name] = vmDev
		volumeNameMap[vmDev.VolumeName] = vmDev
		diskDeviceMap[diskDeviceToDeviceName(vmDev.DiskDevice)] = vmDev
	}
	for _, presetDev := range presetSpec.Devices.Disks {
		if vmDev, conflict := nameMap[presetDev.Name]; conflict {
			if !reflect.DeepEqual(presetDev, vmDev) {
				errors = append(errors, fmt.Errorf("spec.devices.disk[%s]: conflicting disk with same name", presetDev.Name))
			}
		}
		if vmDev, conflict := volumeNameMap[presetDev.VolumeName]; conflict {
			if !reflect.DeepEqual(presetDev, vmDev) {
				errors = append(errors, fmt.Errorf("spec.devices.disk[%s]: conflicting disk with same volume name", presetDev.Name))
			}
		}
		if vmDev, conflict := diskDeviceMap[diskDeviceToDeviceName(presetDev.DiskDevice)]; conflict {
			if !reflect.DeepEqual(presetDev, vmDev) {
				errors = append(errors, fmt.Errorf("spec.devices.disk[%s]: conflicting device", presetDev.Name))
			}
		}
	}
	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

func mergeDomainSpec(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
	err := checkPresetMergeConflicts(presetSpec, vmSpec)
	if err != nil {
		return err
	}
	if len(presetSpec.Resources.Requests) > 0 {
		if vmSpec.Resources.Requests == nil {
			vmSpec.Resources.Requests = v1.ResourceList{}
		}
		for key, val := range presetSpec.Resources.Requests {
			vmSpec.Resources.Requests[key] = val
		}
	}
	if presetSpec.CPU != nil {
		if vmSpec.CPU == nil {
			vmSpec.CPU = &kubev1.CPU{}
		}
		presetSpec.CPU.DeepCopyInto(vmSpec.CPU)
	}
	if presetSpec.Firmware != nil {
		if vmSpec.Firmware == nil {
			vmSpec.Firmware = &kubev1.Firmware{}
		}
		presetSpec.Firmware.DeepCopyInto(vmSpec.Firmware)
	}
	if presetSpec.Clock != nil {
		if vmSpec.Clock == nil {
			vmSpec.Clock = &kubev1.Clock{}
		}
		vmSpec.Clock.ClockOffset = presetSpec.Clock.ClockOffset
		if presetSpec.Clock.Timer != nil {
			if vmSpec.Clock.Timer == nil {
				vmSpec.Clock.Timer = &kubev1.Timer{}
			}
			presetSpec.Clock.Timer.DeepCopyInto(vmSpec.Clock.Timer)
		}
	}
	if presetSpec.Features != nil {
		if vmSpec.Features == nil {
			vmSpec.Features = &kubev1.Features{}
		}
		presetSpec.Features.DeepCopyInto(vmSpec.Features)
	}
	if presetSpec.Devices.Watchdog != nil {
		if vmSpec.Devices.Watchdog == nil {
			vmSpec.Devices.Watchdog = &kubev1.Watchdog{}
		}
		presetSpec.Devices.Watchdog.DeepCopyInto(vmSpec.Devices.Watchdog)
	}
	// Devices in the VM should appear first (for mount point ordering)
	// Append all devices from preset, but ignore duplicates.
	deviceSet := make(map[string]bool)
	for _, vmDev := range vmSpec.Devices.Disks {
		deviceSet[vmDev.Name] = true
	}
	for _, presetDev := range presetSpec.Devices.Disks {
		if !deviceSet[presetDev.Name] {
			vmSpec.Devices.Disks = append(vmSpec.Devices.Disks, presetDev)
		}
	}
	return nil
}

func applyPresets(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) error {
	for _, preset := range presets {
		err := mergeDomainSpec(preset.Spec.Domain, &vm.Spec.Domain)
		if err != nil {
			return fmt.Errorf("unable to apply preset '%s' to virtual machine '%s': %v", preset.Name, vm.Name, err)
		}
	}

	err := annotateVM(vm, presets)
	if err != nil {
		return err
	}
	return nil
}

func removeInitializer(vm *kubev1.VirtualMachine) {
	if vm.Initializers == nil {
		// If Initializers is nil, there's nothing to remove.
		return
	}
	newInitilizers := []k8smetav1.Initializer{}
	for _, i := range vm.Initializers.Pending {
		if i.Name != initializerMarking {
			newInitilizers = append(newInitilizers, i)
		}
	}
	vm.Initializers.Pending = newInitilizers
}

func annotateVM(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) error {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{}
	}
	for _, preset := range presets {
		annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", kubev1.GroupName, preset.Name)
		vm.Annotations[annotationKey] = kubev1.GroupVersion.String()
	}
	return nil
}
