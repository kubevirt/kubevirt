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

package mutators

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Mutating Webhook Namespace Limits", func() {
	var vmi v1.VirtualMachineInstance
	var namespaceLimit *k8sv1.LimitRange
	var namespaceLimitInformer cache.SharedIndexInformer

	BeforeEach(func() {
		vmi = v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{},
				},
			},
		}
		namespaceLimit = &k8sv1.LimitRange{
			Spec: k8sv1.LimitRangeSpec{
				Limits: []k8sv1.LimitRangeItem{
					{
						Type: k8sv1.LimitTypeContainer,
						Default: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("256M"),
							k8sv1.ResourceCPU:    resource.MustParse("4"),
						},
						DefaultRequest: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("128M"),
							k8sv1.ResourceCPU:    resource.MustParse("1"),
						},
					},
				},
			},
		}
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		namespaceLimitInformer.GetIndexer().Add(namespaceLimit)
	})

	When("VMI has all limits under spec", func() {
		It("should not apply namespace limits", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
				k8sv1.ResourceCPU:    resource.MustParse("2"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("32M"),
				k8sv1.ResourceCPU:    resource.MustParse("500m"),
			}

			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(&vmi, namespaceLimitInformer)

			Expect(vmi.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("64M"))
			Expect(vmi.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("32M"))
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal("500m"))
		})
	})

	When("VMI does not have limits under spec", func() {
		It("should apply all namespace limits", func() {
			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(&vmi, namespaceLimitInformer)

			Expect(vmi.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("256M"))
			Expect(vmi.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal("4"))
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("128M"))
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal("1"))
		})
	})

	When("VMI has some limits under spec", func() {
		It("should apply only missing namespace default limits", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("2"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("32M"),
				k8sv1.ResourceCPU:    resource.MustParse("500m"),
			}

			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(&vmi, namespaceLimitInformer)

			Expect(vmi.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("256M"))
			Expect(vmi.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("32M"))
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal("500m"))
		})

		It("should apply only missing namespace defaultRequest limits", func() {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
				k8sv1.ResourceCPU:    resource.MustParse("2"),
			}
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("32M"),
			}

			By("Applying namespace range values on the VMI")
			applyNamespaceLimitRangeValues(&vmi, namespaceLimitInformer)

			Expect(vmi.Spec.Domain.Resources.Limits.Memory().String()).To(Equal("64M"))
			Expect(vmi.Spec.Domain.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("32M"))
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().String()).To(Equal("1"))
		})
	})
})
