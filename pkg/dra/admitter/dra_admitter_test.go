/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package admitter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
)

type fakeConfigChecker struct {
	gpuDRAEnabled bool
}

func (f *fakeConfigChecker) GPUsWithDRAGateEnabled() bool {
	return f.gpuDRAEnabled
}

var _ = Describe("DRA Admitter", func() {
	var (
		field   *k8sfield.Path
		checker *fakeConfigChecker
	)

	BeforeEach(func() {
		field = k8sfield.NewPath("spec")
		checker = &fakeConfigChecker{}
	})

	Context("when no GPUs are specified", func() {
		It("should produce no causes", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})
	})

	Context("non-DRA (device-plugin) GPUs", func() {
		It("should accept a GPU with deviceName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name:       "gpu1",
							DeviceName: "vfio.gpu.example.com",
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should reject a GPU without deviceName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("without deviceName"))
		})

		It("should reject a GPU with both deviceName and claimRequest", func() {
			checker.gpuDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name:       "gpu1",
							DeviceName: "vfio.gpu.example.com",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("both deviceName and claimRequest"))
		})
	})

	Context("DRA GPUs with feature gate disabled", func() {
		It("should reject DRA GPUs when the feature gate is off", func() {
			checker.gpuDRAEnabled = false
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("feature gate is not enabled"))
		})
	})

	Context("DRA GPU field validation", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should reject when claimName is nil", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("claimName is required"))
		})

		It("should reject when claimName is empty string", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To(""),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("claimName is required"))
		})

		It("should reject when requestName is nil", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName: ptr.To("claim1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("requestName is required"))
		})

		It("should reject when requestName is empty string", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To(""),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("requestName is required"))
		})

		It("should report two causes when both claimName and requestName are missing", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name:         "gpu1",
							ClaimRequest: &v1.ClaimRequest{},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(2))
		})
	})

	Context("resourceClaims cross-validation", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should reject duplicate names in spec.resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "claim1"},
					{Name: "claim1"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(causes[0].Message).To(ContainSubstring("duplicate resourceClaims name"))
		})

		It("should reject when claimName is not listed in spec.resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "other-claim"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
		})

		It("should reject when one of multiple claims is missing from spec.resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "gpu2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim2"),
									RequestName: ptr.To("req2"),
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
		})
	})

	Context("fully valid DRA GPU specs", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should accept a single valid DRA GPU", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should accept multiple valid DRA GPUs with matching resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "claim1"},
					{Name: "claim2"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "gpu2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim2"),
									RequestName: ptr.To("req2"),
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})
	})

	Context("duplicate claimName/requestName pairs", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should reject two GPUs referencing the same claimName/requestName pair", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "gpu2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(causes[0].Message).To(ContainSubstring("duplicate claimName/requestName"))
		})

		It("should accept two GPUs referencing the same claim but different requests", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "gpu2",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req2"),
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})
	})

	Context("Validator methods", func() {
		It("ValidateCreation should delegate to validateCreationDRA", func() {
			checker.gpuDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			v := NewValidator(field, spec, checker)
			causes := v.ValidateCreation()
			Expect(causes).To(BeEmpty())
		})

		It("Validate should delegate to validateCreationDRA", func() {
			checker.gpuDRAEnabled = false
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("claim1"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			v := NewValidator(field, spec, checker)
			causes := v.Validate()
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("feature gate is not enabled"))
		})
	})
})
