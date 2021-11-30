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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("VirtualMachine Mutator", func() {
	var vm *v1.VirtualMachine
	var kvInformer cache.SharedIndexInformer
	var mutator *VMsMutator

	machineTypeFromConfig := "pc-q35-3.0"

	getVMSpecMetaFromResponse := func() (*v1.VirtualMachineSpec, *k8smetav1.ObjectMeta) {
		vmBytes, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VM")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
				Object: runtime.RawExtension{
					Raw: vmBytes,
				},
			},
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

	BeforeEach(func() {
		vm = &v1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
		}
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		mutator = &VMsMutator{}
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	})

	It("should apply defaults on VM create", func() {
		vmSpec, _ := getVMSpecMetaFromResponse()
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

		vmSpec, _ := getVMSpecMetaFromResponse()
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

		vmSpec, _ := getVMSpecMetaFromResponse()
		Expect(vmSpec.Template.Spec.Domain.Machine.Type).To(Equal(vm.Spec.Template.Spec.Domain.Machine.Type))
	})
})
