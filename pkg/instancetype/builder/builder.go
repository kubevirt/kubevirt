package builder

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type InstancetypeSpecOption func(*v1beta1.VirtualMachineInstancetypeSpec)

func NewInstancetype(opts ...InstancetypeSpecOption) *v1beta1.VirtualMachineInstancetype {
	instancetype := v1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newInstancetypeSpec(opts...),
	}
	return &instancetype
}

func NewClusterInstancetype(opts ...InstancetypeSpecOption) *v1beta1.VirtualMachineClusterInstancetype {
	instancetype := v1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "clusterinstancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
		Spec: newInstancetypeSpec(opts...),
	}
	return &instancetype
}

func newInstancetypeSpec(opts ...InstancetypeSpecOption) v1beta1.VirtualMachineInstancetypeSpec {
	spec := &v1beta1.VirtualMachineInstancetypeSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func WithCPUs(vCPUs uint32) InstancetypeSpecOption {
	return func(spec *v1beta1.VirtualMachineInstancetypeSpec) {
		spec.CPU.Guest = vCPUs
	}
}

func WithMemory(memory string) InstancetypeSpecOption {
	return func(spec *v1beta1.VirtualMachineInstancetypeSpec) {
		spec.Memory.Guest = resource.MustParse(memory)
	}
}

func fromVMI(vmi *virtv1.VirtualMachineInstance) InstancetypeSpecOption {
	return func(spec *v1beta1.VirtualMachineInstancetypeSpec) {
		// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
		guestMemory := resource.MustParse("128M")
		if vmi != nil {
			if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
				guestMemory = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
			}
		}
		spec.CPU = v1beta1.CPUInstancetype{
			Guest: uint32(1),
		}
		spec.Memory.Guest = guestMemory
	}
}

func NewInstancetypeFromVMI(vmi *virtv1.VirtualMachineInstance) *v1beta1.VirtualMachineInstancetype {
	return NewInstancetype(
		fromVMI(vmi),
	)
}

func NewClusterInstancetypeFromVMI(vmi *virtv1.VirtualMachineInstance) *v1beta1.VirtualMachineClusterInstancetype {
	return NewClusterInstancetype(
		fromVMI(vmi),
	)
}

type PreferenceSpecOption func(*v1beta1.VirtualMachinePreferenceSpec)

func newPreferenceSpec(opts ...PreferenceSpecOption) v1beta1.VirtualMachinePreferenceSpec {
	spec := &v1beta1.VirtualMachinePreferenceSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func NewPreference(opts ...PreferenceSpecOption) *v1beta1.VirtualMachinePreference {
	preference := &v1beta1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newPreferenceSpec(opts...),
	}
	return preference
}

func NewClusterPreference(opts ...PreferenceSpecOption) *v1beta1.VirtualMachineClusterPreference {
	preference := &v1beta1.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "clusterpreference-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
		Spec: newPreferenceSpec(opts...),
	}
	return preference
}

func WithPreferredCPUTopology(topology v1beta1.PreferredCPUTopology) PreferenceSpecOption {
	return func(spec *v1beta1.VirtualMachinePreferenceSpec) {
		if spec.CPU == nil {
			spec.CPU = &v1beta1.CPUPreferences{}
		}
		spec.CPU.PreferredCPUTopology = pointer.P(topology)
	}
}

func WithPreferredDiskBus(bus virtv1.DiskBus) PreferenceSpecOption {
	return func(spec *v1beta1.VirtualMachinePreferenceSpec) {
		if spec.Devices == nil {
			spec.Devices = &v1beta1.DevicePreferences{}
		}
		spec.Devices.PreferredDiskBus = bus
	}
}
