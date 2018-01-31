package watch

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type VMInitializer struct {
	clientset  kubecli.KubevirtClient
	restClient *rest.RESTClient
	queue      workqueue.RateLimitingInterface
	store      cache.Store
}

const initializerMarking = "virtualmachines.kubevirt.io"
const annotationPrefix = "virtualmachinepreset.admission.kubernetes.io/virtualmachinepreset"

// FIXME: Both the restClient and clientSet are probably not needed.
func NewVMInitializer(restClient *rest.RESTClient, queue workqueue.RateLimitingInterface, vmCache cache.Store, clientset kubecli.KubevirtClient) *VMInitializer {
	vmi := VMInitializer{
		restClient: restClient,
		clientset:  clientset,
		store:      vmCache,
		queue:      queue,
	}
	return &vmi
}

func (c *VMInitializer) initializeVM(vm *kubev1.VirtualMachine) error {
	// All VM's must be marked as initialized or they are held in limbo forever
	// Collect all errors and defer returning until after the update
	logger := log.Log
	errors := []error{}

	logger.Object(vm).Info("Initializing VM")

	allPresets, err := listPresets(c.restClient, vm.GetNamespace(), labels.Everything())
	//list, err := c.clientset.VirtualMachinePresets(a.GetNamespace()).List(labels.Everything())

	matchingPresets, err := filterPresets(allPresets, vm)

	if err != nil {
		logger.Object(vm).Reason(err).Errorf("Error while matching presets to VirtualMachine")
		errors = append(errors, err)
	}

	if len(matchingPresets) != 0 {
		err = checkPresetMergeConflicts(vm, matchingPresets)

		if err != nil {
			logger.Reason(err).Errorf("Conflicting VM Presets")
			errors = append(errors, fmt.Errorf("conflicting vm presets"))
		}

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
	c.clientset.VM(vm.Namespace).Update(vm)

	if len(errors) > 0 {
		utilerrors.NewAggregate(errors)
	}
	return nil
}

// FIXME: this should be integrated into the KubeVirt clientset
func listPresets(restClient *rest.RESTClient, namespace string, selector labels.Selector) ([]kubev1.VirtualMachinePreset, error) {
	options := k8smetav1.ListOptions{LabelSelector: selector.String()}
	presetList := &kubev1.VirtualMachinePresetList{}
	if err := restClient.Get().
		Resource("virtualmachinepresets").
		Namespace(namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(presetList); err != nil {
		return nil, err
	}
	for _, preset := range presetList.Items {
		preset.SetGroupVersionKind(kubev1.VirtualMachinePresetGroupVersionKind)
	}

	return presetList.Items, nil
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

func checkPresetMergeConflicts(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) error {
	//FIXME: implement
	return nil
}

func mergeDomainSpec(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
	if len(presetSpec.Resources.Requests) > 0 {
		if vmSpec.Resources.Requests == nil {
			vmSpec.Resources.Requests = v1.ResourceList{}
		}
		for key, val := range presetSpec.Resources.Requests {
			if vmSpec.Resources.Requests[key] != val {
			}
			vmSpec.Resources.Requests[key] = val
		}
	}
	if presetSpec.CPU != nil {
		presetSpec.CPU.DeepCopyInto(vmSpec.CPU)
	}
	if presetSpec.Firmware != nil {
		presetSpec.Firmware.DeepCopyInto(vmSpec.Firmware)
	}
	if presetSpec.Clock != nil {
		if vmSpec.Clock == nil {
			vmSpec.Clock = new(kubev1.Clock)
		}
		vmSpec.Clock.ClockOffset = presetSpec.Clock.ClockOffset
		if presetSpec.Clock.Timer != nil {
			presetSpec.Clock.Timer.DeepCopyInto(vmSpec.Clock.Timer)
		}
	}
	if presetSpec.Features != nil {
		presetSpec.Features.DeepCopyInto(vmSpec.Features)
	}
	if presetSpec.Devices.Watchdog != nil {
		presetSpec.Devices.Watchdog.DeepCopyInto(vmSpec.Devices.Watchdog)

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
	}
	return nil
}

func applyPresets(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) error {
	for _, preset := range presets {
		err := mergeDomainSpec(preset.Spec.Domain, &vm.Spec.Domain)
		if err != nil {
			return err
		}
	}

	err := annotateVM(vm, presets)
	if err != nil {
		return err
	}
	return nil
}

func removeInitializer(vm *kubev1.VirtualMachine) {
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
		kind := strings.ToLower(preset.Kind)
		annotationKey := fmt.Sprintf("%s.%s/%s", kind, kubev1.GroupName, preset.Name)
		vm.Annotations[annotationKey] = preset.APIVersion
	}

	return nil
}
