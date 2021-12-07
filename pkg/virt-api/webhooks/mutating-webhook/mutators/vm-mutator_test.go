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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/pkg/testutils"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("VirtualMachine Mutator", func() {
	var vm *v1.VirtualMachine
	var kvInformer cache.SharedIndexInformer
	var mutator *VMsMutator

	var flavorMethods *testutils.MockFlavorMethods

	machineTypeFromConfig := "pc-q35-3.0"

	getVMSpecMetaFromResponse := func(operation admissionv1.Operation, oldVM *v1.VirtualMachine) (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		vmBytes, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
				Operation: operation,
			},
		}
		if oldVM != nil {
			vmBytes, err = json.Marshal(oldVM)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.OldObject.Raw = vmBytes
		}
		By("Mutating the VM")
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VM spec from the response")
		vmSpec := &v1.VirtualMachineSpec{}
		vmMeta := &k8smetav1.ObjectMeta{}
		patch := []utiltypes.PatchOperation{
			{Value: vmSpec},
			{Value: vmMeta},
		}
		err = json.Unmarshal(resp.Patch, &patch)
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).NotTo(BeEmpty())

		return vmSpec, vmMeta
	}

	getMutateResponse := func(operation admissionv1.Operation, oldVM *v1.VirtualMachine) *admissionv1.AdmissionResponse {
		vmBytes, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
				Operation: operation,
			},
		}
		if oldVM != nil {
			vmBytes, err = json.Marshal(oldVM)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.OldObject.Raw = vmBytes
		}
		By("Mutating the VM")
		return mutator.Mutate(ar)
	}

	BeforeEach(func() {
		vm = &v1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
		}
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		flavorMethods = testutils.NewMockFlavorMethods()

		flavorMethods.FindFlavorFunc = func(_ *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			return &flavorv1alpha1.VirtualMachineFlavorProfile{
				CPU: &v1.CPU{
					Sockets: 2,
					Cores:   1,
					Threads: 1,
				},
			}, nil
		}

		mutator = &VMsMutator{FlavorMethods: flavorMethods}
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	})

	It("should apply defaults on VM create", func() {
		vmSpec, _ := getVMSpecMetaFromResponse(admissionv1.Create, nil)
		if webhooks.IsPPC64() {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("pseries"))
		} else if webhooks.IsARM64() {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("virt"))
		} else {
			Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal("q35"))
		}
	})

	It("should apply configurable defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vmSpec, _ := getVMSpecMetaFromResponse(admissionv1.Create, nil)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
	})

	It("should not override specified properties with defaults on VM create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					MachineType: machineTypeFromConfig,
				},
			},
		})

		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: "q35"}

		vmSpec, _ := getVMSpecMetaFromResponse(admissionv1.Create, nil)
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})

	It("should apply flavor only if vm is created", func() {
		vm.Spec.Flavor = &v1.FlavorMatcher{
			Name: "test",
		}
		flavorMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, vm *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
			vm.Domain.CPU = &v1.CPU{
				Cores: 3,
			}
			return nil
		}
		vmSpec, _ := getVMSpecMetaFromResponse(admissionv1.Create, nil)
		Expect(vmSpec.Template.Spec.Domain.CPU).ToNot(BeNil(), "cpu topology should not equal nil")
		Expect(vmSpec.Template.Spec.Domain.CPU.Cores).To(Equal(uint32(3)), "cores should equal")
	})

	It("should not apply flavor if vm is updated", func() {
		vm.Spec.Flavor = &v1.FlavorMatcher{
			Name: "test",
		}
		vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
			Cores: 1,
		}

		flavorMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, vm *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
			vm.Domain.CPU = &v1.CPU{
				Cores: 3,
			}
			return nil
		}
		oldVM := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Flavor: &v1.FlavorMatcher{
					Name: "test",
				},
			},
		}

		vmSpec, _ := getVMSpecMetaFromResponse(admissionv1.Update, oldVM)
		Expect(vmSpec.Template.Spec.Domain.CPU).ToNot(BeNil(), "cpu topology shoulds equal nil")
		Expect(vmSpec.Template.Spec.Domain.CPU.Cores).To(Equal(uint32(1)), "cores should equal")
	})

	It("should reject if flavor is not found", func() {
		vm.Spec.Flavor = &v1.FlavorMatcher{
			Name: "test",
		}
		flavorMethods.FindFlavorFunc = func(_ *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			return nil, fmt.Errorf("flavor not found")
		}

		response := getMutateResponse(admissionv1.Create, nil)
		Expect(response.Allowed).To(BeFalse())
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
		Expect(response.Result.Details.Causes[0].Field).To(Equal("spec.flavor"))
	})

	It("should reject if flavor fails to apply to VMI", func() {
		vm.Spec.Flavor = &v1.FlavorMatcher{
			Name: "test",
		}
		var (
			basePath = k8sfield.NewPath("spec", "template", "spec")
			path1    = basePath.Child("example", "path")
			path2    = basePath.Child("domain", "example", "path")
		)
		flavorMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, _ *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
			return flavor.Conflicts{path1, path2}
		}

		response := getMutateResponse(admissionv1.Create, nil)
		Expect(response.Allowed).To(BeFalse())
		Expect(response.Result.Details.Causes).To(HaveLen(2))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(path1.String()))
		Expect(response.Result.Details.Causes[1].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[1].Field).To(Equal(path2.String()))
	})
})
