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
 */

package builder

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type InstancetypeSpecOption func(*instancetypev1alpha1.VirtualMachineInstancetypeSpec)

func NewInstancetype(opts ...InstancetypeSpecOption) *instancetypev1alpha1.VirtualMachineInstancetype {
	instancetype := instancetypev1alpha1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newInstancetypeSpec(opts...),
	}
	return &instancetype
}

func NewClusterInstancetype(opts ...InstancetypeSpecOption) *instancetypev1alpha1.VirtualMachineClusterInstancetype {
	instancetype := instancetypev1alpha1.VirtualMachineClusterInstancetype{
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

func newInstancetypeSpec(opts ...InstancetypeSpecOption) instancetypev1alpha1.VirtualMachineInstancetypeSpec {
	spec := &instancetypev1alpha1.VirtualMachineInstancetypeSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func WithCPUs(vCPUs uint32) InstancetypeSpecOption {
	return func(spec *instancetypev1alpha1.VirtualMachineInstancetypeSpec) {
		spec.CPU.Guest = vCPUs
	}
}

func WithMemory(memory string) InstancetypeSpecOption {
	return func(spec *instancetypev1alpha1.VirtualMachineInstancetypeSpec) {
		spec.Memory.Guest = resource.MustParse(memory)
	}
}

func fromVMI(vmi *v1.VirtualMachineInstance) InstancetypeSpecOption {
	return func(spec *instancetypev1alpha1.VirtualMachineInstancetypeSpec) {
		// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
		guestMemory := resource.MustParse("128M")
		if vmi != nil {
			if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
				guestMemory = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
			}
		}
		spec.CPU = instancetypev1alpha1.CPUInstancetype{
			Guest: uint32(1),
		}
		spec.Memory.Guest = guestMemory
	}
}

func NewInstancetypeFromVMI(vmi *v1.VirtualMachineInstance) *instancetypev1alpha1.VirtualMachineInstancetype {
	return NewInstancetype(
		fromVMI(vmi),
	)
}

func NewClusterInstancetypeFromVMI(vmi *v1.VirtualMachineInstance) *instancetypev1alpha1.VirtualMachineClusterInstancetype {
	return NewClusterInstancetype(
		fromVMI(vmi),
	)
}

type PreferenceSpecOption func(*instancetypev1alpha1.VirtualMachinePreferenceSpec)

func newPreferenceSpec(opts ...PreferenceSpecOption) instancetypev1alpha1.VirtualMachinePreferenceSpec {
	spec := &instancetypev1alpha1.VirtualMachinePreferenceSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func NewPreference(opts ...PreferenceSpecOption) *instancetypev1alpha1.VirtualMachinePreference {
	preference := &instancetypev1alpha1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newPreferenceSpec(opts...),
	}
	return preference
}

func NewClusterPreference(opts ...PreferenceSpecOption) *instancetypev1alpha1.VirtualMachineClusterPreference {
	preference := &instancetypev1alpha1.VirtualMachineClusterPreference{
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

func WithPreferredCPUTopology(topology instancetypev1alpha1.PreferredCPUTopology) PreferenceSpecOption {
	return func(spec *instancetypev1alpha1.VirtualMachinePreferenceSpec) {
		if spec.CPU == nil {
			spec.CPU = &instancetypev1alpha1.CPUPreferences{}
		}
		spec.CPU.PreferredCPUTopology = topology
	}
}
