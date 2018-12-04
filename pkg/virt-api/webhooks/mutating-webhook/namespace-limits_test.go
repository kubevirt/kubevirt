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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package mutating_webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Mutating Webhook Namespace Limits", func() {
	var vmi v1.VirtualMachineInstance
	var namespaceLimit *k8sv1.LimitRange
	var namespaceLimitInformer cache.SharedIndexInformer

	memory, _ := resource.ParseQuantity("64M")
	limitMemory, _ := resource.ParseQuantity("128M")
	zeroMemory, _ := resource.ParseQuantity("0M")

	BeforeEach(func() {
		vmi = v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							"memory": memory,
						},
					},
				},
			},
		}
		namespaceLimit = &k8sv1.LimitRange{
			Spec: k8sv1.LimitRangeSpec{
				Limits: []k8sv1.LimitRangeItem{
					{
						Type: k8sv1.LimitTypeContainer,
						Default: k8sv1.ResourceList{
							k8sv1.ResourceMemory: limitMemory,
						},
					},
				},
			},
		}
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		namespaceLimitInformer.GetIndexer().Add(namespaceLimit)
	})

	When("VMI has limits under spec", func() {
		It("should not apply namespace limits", func() {
			vmiCopy := vmi.DeepCopy()
			vmiCopy.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: memory,
			}
			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(vmiCopy, namespaceLimitInformer)

			Expect(vmiCopy.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("64M"))
		})
	})

	When("VMI does not have limits under spec", func() {
		It("should apply namespace limits", func() {
			vmiCopy := vmi.DeepCopy()
			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(vmiCopy, namespaceLimitInformer)

			Expect(vmiCopy.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("128M"))
		})

		When("namespace limit equals 0", func() {
			It("should not apply namespace limits", func() {
				vmiCopy := vmi.DeepCopy()

				namespaceLimitCopy := namespaceLimit.DeepCopy()
				namespaceLimitCopy.Spec.Limits[0].Default[k8sv1.ResourceMemory] = zeroMemory
				namespaceLimitInformer.GetIndexer().Update(namespaceLimitCopy)

				By("Applying namespace range values on the VMI")
				applyNamespaceLimitRangeValues(vmiCopy, namespaceLimitInformer)
				_, ok := vmiCopy.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]
				Expect(ok).To(Equal(false))
			})

			AfterEach(func() {
				namespaceLimitInformer.GetIndexer().Update(namespaceLimit)
			})
		})

		When("namespace has more than one limit range", func() {
			var additionalNamespaceLimit *k8sv1.LimitRange

			BeforeEach(func() {
				additionalLimitMemory, _ := resource.ParseQuantity("76M")
				additionalNamespaceLimit = &k8sv1.LimitRange{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name: "additional-limit-range",
					},
					Spec: k8sv1.LimitRangeSpec{
						Limits: []k8sv1.LimitRangeItem{
							{
								Type: k8sv1.LimitTypeContainer,
								Default: k8sv1.ResourceList{
									k8sv1.ResourceMemory: additionalLimitMemory,
								},
							},
						},
					},
				}
				namespaceLimitInformer.GetIndexer().Add(additionalNamespaceLimit)
			})

			It("should apply range with minimal limit", func() {
				vmiCopy := vmi.DeepCopy()
				By("Applying namespace range values on the VMI")
				applyNamespaceLimitRangeValues(vmiCopy, namespaceLimitInformer)

				Expect(vmiCopy.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("76M"))
			})
		})
	})
})
