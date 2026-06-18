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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

var _ = Describe("ResourceClaim management", func() {

	Context("SetupVMIFromVM ResourceClaim rewrite", func() {
		It("should rewrite resourceClaimTemplateName to resourceClaimName for matching entries", func() {
			templateName := "pgpu-claim-tmpl"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{
						{
							Name:                      "gpu-claim",
							ResourceClaimTemplateName: templateName,
						},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{
									Name:                      "gpu-claim",
									ResourceClaimTemplateName: &templateName,
								},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(1))
			rc := vmi.Spec.ResourceClaims[0]
			Expect(rc.ResourceClaimTemplateName).To(BeNil())
			Expect(rc.ResourceClaimName).ToNot(BeNil())
			Expect(*rc.ResourceClaimName).To(Equal("test-vm-gpu-claim"))
		})

		It("should not rewrite entries without matching ResourceClaimTemplates", func() {
			templateName := "external-claim-tmpl"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy:            runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{
									Name:                      "other-claim",
									ResourceClaimTemplateName: &templateName,
								},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(1))
			rc := vmi.Spec.ResourceClaims[0]
			Expect(rc.ResourceClaimTemplateName).ToNot(BeNil())
			Expect(*rc.ResourceClaimTemplateName).To(Equal("external-claim-tmpl"))
			Expect(rc.ResourceClaimName).To(BeNil())
		})

		It("should not rewrite when ResourceClaimTemplate name does not match claim name", func() {
			templateName := "external-claim-tmpl"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{
						{Name: "gpu-claim", ResourceClaimTemplateName: "gpu-template"},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{
									Name:                      "different-claim",
									ResourceClaimTemplateName: &templateName,
								},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(1))
			rc := vmi.Spec.ResourceClaims[0]
			Expect(rc.ResourceClaimTemplateName).ToNot(BeNil())
			Expect(*rc.ResourceClaimTemplateName).To(Equal("external-claim-tmpl"))
			Expect(rc.ResourceClaimName).To(BeNil())
		})

		It("should preserve direct resourceClaimName references", func() {
			directName := "pre-existing-claim"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{
									Name:              "gpu-claim",
									ResourceClaimName: &directName,
								},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(1))
			rc := vmi.Spec.ResourceClaims[0]
			Expect(rc.ResourceClaimName).ToNot(BeNil())
			Expect(*rc.ResourceClaimName).To(Equal("pre-existing-claim"))
			Expect(rc.ResourceClaimTemplateName).To(BeNil())
		})

		It("should rewrite multiple ResourceClaimTemplates entries", func() {
			tmpl1 := "gpu-tmpl"
			tmpl2 := "nvme-tmpl"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multi-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{
						{Name: "gpu-claim", ResourceClaimTemplateName: "gpu-template"},
						{Name: "nvme-claim", ResourceClaimTemplateName: "nvme-template"},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{Name: "gpu-claim", ResourceClaimTemplateName: &tmpl1},
								{Name: "nvme-claim", ResourceClaimTemplateName: &tmpl2},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(2))
			for _, rc := range vmi.Spec.ResourceClaims {
				Expect(rc.ResourceClaimTemplateName).To(BeNil())
				Expect(rc.ResourceClaimName).ToNot(BeNil())
				expectedName := fmt.Sprintf("multi-vm-%s", rc.Name)
				Expect(*rc.ResourceClaimName).To(Equal(expectedName))
			}
		})

		It("should handle mixed direct and template-based claims", func() {
			templateName := "gpu-tmpl"
			directName := "existing-nvme"
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mixed-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{
						{Name: "gpu-claim", ResourceClaimTemplateName: "gpu-template"},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{
								{Name: "gpu-claim", ResourceClaimTemplateName: &templateName},
								{Name: "nvme-claim", ResourceClaimName: &directName},
							},
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(2))
			Expect(vmi.Spec.ResourceClaims[0].Name).To(Equal("gpu-claim"))
			Expect(vmi.Spec.ResourceClaims[0].ResourceClaimTemplateName).To(BeNil())
			Expect(*vmi.Spec.ResourceClaims[0].ResourceClaimName).To(Equal("mixed-vm-gpu-claim"))
			Expect(vmi.Spec.ResourceClaims[1].Name).To(Equal("nvme-claim"))
			Expect(*vmi.Spec.ResourceClaims[1].ResourceClaimName).To(Equal("existing-nvme"))
		})

		It("should handle empty VMI resourceClaims with non-empty ResourceClaimTemplates", func() {
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "test-uid",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: runStrategyPtr(virtv1.RunStrategyAlways),
					ResourceClaimTemplates: []virtv1.ResourceClaimTemplateEntry{
						{Name: "gpu-claim", ResourceClaimTemplateName: "gpu-template"},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							ResourceClaims: []k8sv1.PodResourceClaim{},
							Domain:         virtv1.DomainSpec{},
						},
					},
				},
			}

			vmi := SetupVMIFromVM(vm)

			Expect(vmi.Spec.ResourceClaims).To(BeEmpty())
		})
	})

	Context("CreateResourceClaimManifest", func() {
		It("should create a ResourceClaim with correct ownership and naming", func() {
			vm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					UID:       "vm-uid-123",
				},
			}
			entry := virtv1.ResourceClaimTemplateEntry{
				Name:                      "gpu-claim",
				ResourceClaimTemplateName: "gpu-template",
			}
			claimTemplate := &resourcev1.ResourceClaimTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gpu-template",
					Namespace: "default",
				},
				Spec: resourcev1.ResourceClaimTemplateSpec{
					Spec: resourcev1.ResourceClaimSpec{
						Devices: resourcev1.DeviceClaim{
							Requests: []resourcev1.DeviceRequest{
								{
									Name: "pgpu",
								},
							},
						},
					},
				},
			}

			rc := watchutil.CreateResourceClaimManifest(entry, claimTemplate, vm)

			Expect(rc.Name).To(Equal("test-vm-gpu-claim"))
			Expect(rc.Namespace).To(Equal("default"))
			Expect(rc.Labels[virtv1.CreatedByLabel]).To(Equal("vm-uid-123"))
			Expect(rc.OwnerReferences).To(HaveLen(1))
			Expect(rc.OwnerReferences[0].Name).To(Equal("test-vm"))
			Expect(*rc.OwnerReferences[0].Controller).To(BeTrue())
			Expect(rc.Spec.Devices.Requests).To(HaveLen(1))
			Expect(rc.Spec.Devices.Requests[0].Name).To(Equal("pgpu"))
		})
	})
})

func runStrategyPtr(s virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachineRunStrategy {
	return &s
}
