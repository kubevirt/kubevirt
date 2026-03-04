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

package vm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("syncPCITopologyAnnotationsToVM", func() {
	It("should skip when vmi has no version annotation", func() {
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{},
			},
		}
		vmi := &v1.VirtualMachineInstance{}

		syncPCITopologyAnnotationsToVM(vm, vmi)
		Expect(vm.Spec.Template.ObjectMeta.Annotations).To(BeEmpty())
	})

	It("should skip when vm template already has version annotation", func() {
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.PciTopologyVersionAnnotation: v1.PciTopologyVersionV3,
						},
					},
				},
			},
		}
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.PciTopologyVersionAnnotation:    v1.PciTopologyVersionV2,
					v1.PciInterfaceSlotCountAnnotation: "8",
				},
			},
		}

		syncPCITopologyAnnotationsToVM(vm, vmi)
		Expect(vm.Spec.Template.ObjectMeta.Annotations[v1.PciTopologyVersionAnnotation]).To(Equal(v1.PciTopologyVersionV3))
		Expect(vm.Spec.Template.ObjectMeta.Annotations).NotTo(HaveKey(v1.PciInterfaceSlotCountAnnotation))
	})

	It("should copy v3 annotation from vmi to vm template", func() {
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{},
			},
		}
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.PciTopologyVersionAnnotation: v1.PciTopologyVersionV3,
				},
			},
		}

		syncPCITopologyAnnotationsToVM(vm, vmi)
		Expect(vm.Spec.Template.ObjectMeta.Annotations[v1.PciTopologyVersionAnnotation]).To(Equal(v1.PciTopologyVersionV3))
		Expect(vm.Spec.Template.ObjectMeta.Annotations).NotTo(HaveKey(v1.PciInterfaceSlotCountAnnotation))
	})

	It("should copy v2 annotation and slot count from vmi to vm template", func() {
		vm := &v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"existing": "annotation",
						},
					},
				},
			},
		}
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.PciTopologyVersionAnnotation:    v1.PciTopologyVersionV2,
					v1.PciInterfaceSlotCountAnnotation: "11",
				},
			},
		}

		syncPCITopologyAnnotationsToVM(vm, vmi)
		Expect(vm.Spec.Template.ObjectMeta.Annotations[v1.PciTopologyVersionAnnotation]).To(Equal(v1.PciTopologyVersionV2))
		Expect(vm.Spec.Template.ObjectMeta.Annotations[v1.PciInterfaceSlotCountAnnotation]).To(Equal("11"))
		Expect(vm.Spec.Template.ObjectMeta.Annotations["existing"]).To(Equal("annotation"))
	})

	It("should handle nil vm and vmi", func() {
		syncPCITopologyAnnotationsToVM(nil, nil)
		syncPCITopologyAnnotationsToVM(&v1.VirtualMachine{}, nil)
		syncPCITopologyAnnotationsToVM(nil, &v1.VirtualMachineInstance{})
	})
})
