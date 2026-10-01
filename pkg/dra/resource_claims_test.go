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

package dra

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ResourceClaims", func() {
	DescribeTable("should convert VMI resourceClaims to Pod resourceClaims",
		func(resourceClaims []v1.VirtualMachineInstanceResourceClaim, expected []k8sv1.PodResourceClaim) {
			Expect(toPodResourceClaims(resourceClaims)).To(Equal(expected))
		},
		Entry("nil resourceClaims",
			nil,
			nil,
		),
		Entry("empty resourceClaims",
			[]v1.VirtualMachineInstanceResourceClaim{},
			nil,
		),
		Entry("direct and template resourceClaims",
			[]v1.VirtualMachineInstanceResourceClaim{
				{
					Name:              "direct-claim",
					ResourceClaimName: ptr.To("resource-claim"),
				},
				{
					Name:                      "template-claim",
					ResourceClaimTemplateName: ptr.To("resource-claim-template"),
				},
			},
			[]k8sv1.PodResourceClaim{
				{
					Name:              "direct-claim",
					ResourceClaimName: ptr.To("resource-claim"),
				},
				{
					Name:                      "template-claim",
					ResourceClaimTemplateName: ptr.To("resource-claim-template"),
				},
			},
		),
	)

	Describe("PodResourceClaimsForVMI", func() {
		It("should append synthesized CPU pod claim when enabled", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{DedicatedCPUPlacement: true},
					},
				},
			}
			Expect(PodResourceClaimsForVMI(vmi, true)).To(Equal([]k8sv1.PodResourceClaim{
				{
					Name:              CPUClaimRefName,
					ResourceClaimName: ptr.To(CPUResourceClaimName("testvmi")),
				},
			}))
		})

		It("should merge user claims with synthesized CPU claim", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
						{Name: "net-claim", ResourceClaimName: ptr.To("net-rc")},
					},
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{DedicatedCPUPlacement: true},
					},
				},
			}
			Expect(PodResourceClaimsForVMI(vmi, true)).To(Equal([]k8sv1.PodResourceClaim{
				{Name: "net-claim", ResourceClaimName: ptr.To("net-rc")},
				{
					Name:              CPUClaimRefName,
					ResourceClaimName: ptr.To(CPUResourceClaimName("testvmi")),
				},
			}))
		})

		It("should not append CPU claim when synthesis is disabled", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{DedicatedCPUPlacement: true},
					},
				},
			}
			Expect(PodResourceClaimsForVMI(vmi, false)).To(BeNil())
		})
	})

	Describe("ShouldSynthesizeCPUResourceClaim", func() {
		It("should not synthesize for a VMI without dedicated CPUs", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi"},
			}
			Expect(ShouldSynthesizeCPUResourceClaim(vmi)).To(BeFalse())
		})

		It("should not synthesize for NUMA guest mapping passthrough", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							DedicatedCPUPlacement: true,
							NUMA: &v1.NUMA{
								GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
							},
						},
					},
				},
			}
			Expect(ShouldSynthesizeCPUResourceClaim(vmi)).To(BeFalse())
		})

		It("should synthesize for a dedicated CPU VMI", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{DedicatedCPUPlacement: true},
					},
				},
			}
			Expect(ShouldSynthesizeCPUResourceClaim(vmi)).To(BeTrue())
		})
	})
})
