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
package render_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/render"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Instancetype renderer", func() {
	DescribeTable("propagates finder errors without modifying inputs", func(target string, instancetypeFailure bool) {
		expected := errors.New("spec lookup failed")
		finder := failingSpecFinder{}
		if instancetypeFailure {
			finder.instancetypeError = expected
		} else {
			finder.preferenceError = expected
		}
		handler := render.New(finder, finder)
		vm := libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithPreference("test"))
		vmi := libvmi.New()
		originalVM := vm.DeepCopy()
		originalVMI := vmi.DeepCopy()

		var err error
		switch target {
		case "auto-attach":
			err = handler.ApplyAutoAttachPreferences(vm, vmi)
		case "VM":
			err = handler.ApplyToVM(vm)
		case "VMI":
			err = handler.ApplyToVMI(vm, vmi)
		}
		Expect(errors.Is(err, expected)).To(BeTrue())
		Expect(vm).To(Equal(originalVM))
		Expect(vmi).To(Equal(originalVMI))
	},
		Entry("when looking up auto-attach preferences", "auto-attach", false),
		Entry("when looking up the instancetype for a VM", "VM", true),
		Entry("when looking up preferences for a VM", "VM", false),
		Entry("when looking up the instancetype for a VMI", "VMI", true),
		Entry("when looking up preferences for a VMI", "VMI", false),
	)

	It("isolates supplied specs and nested device data across offline applications", func() {
		instancetype := &v1beta1.VirtualMachineInstancetypeSpec{
			CPU:    v1beta1.CPUInstancetype{Guest: 1},
			Memory: v1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
			GPUs: []virtv1.GPU{{
				Name:         "gpu",
				ClaimRequest: &virtv1.ClaimRequest{ClaimName: "original"},
			}},
		}
		vm := libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithClusterInstancetype("test"))
		handler := render.NewOffline(instancetype, nil)
		Expect(handler.ValidateVM(vm)).To(Succeed())

		first := &virtv1.VirtualMachineInstance{}
		Expect(handler.ApplyToVMI(vm, first)).To(Succeed())
		first.Spec.Domain.Devices.GPUs[0].ClaimName = "changed"

		second := &virtv1.VirtualMachineInstance{}
		Expect(handler.ApplyToVMI(vm, second)).To(Succeed())
		Expect(second.Spec.Domain.Devices.GPUs[0].ClaimName).To(Equal("original"))
		Expect(instancetype.GPUs[0].ClaimName).To(Equal("original"))
	})
})

type failingSpecFinder struct {
	instancetypeError error
	preferenceError   error
}

func (f failingSpecFinder) Find(_ *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	return nil, f.instancetypeError
}

func (f failingSpecFinder) FindPreference(_ *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	return nil, f.preferenceError
}
