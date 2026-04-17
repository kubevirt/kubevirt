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
	gpuDRAEnabled        bool
	hostDeviceDRAEnabled bool
}

func (f *fakeConfigChecker) GPUsWithDRAGateEnabled() bool {
	return f.gpuDRAEnabled
}

func (f *fakeConfigChecker) HostDevicesWithDRAEnabled() bool {
	return f.hostDeviceDRAEnabled
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

	Context("when no GPUs or HostDevices are specified", func() {
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].claimName"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].claimName"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].requestName"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].requestName"))
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].claimName"))
			Expect(causes[1].Field).To(Equal("spec.domain.devices.gpus[0].requestName"))
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
			Expect(causes[0].Field).To(Equal("spec.resourceClaims[1].name"))
		})

		It("should reject when GPU claimName is not listed in spec.resourceClaims", func() {
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
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
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
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
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

	Context("duplicate claimName/requestName pairs for GPUs", func() {
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[1]"))
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

	Context("non-DRA (device-plugin) HostDevices", func() {
		It("should accept a HostDevice with deviceName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name:       "hd1",
							DeviceName: "vfio.device.example.com",
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should reject a HostDevice with both deviceName and claimRequest", func() {
			checker.hostDeviceDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name:       "hd1",
							DeviceName: "vfio.device.example.com",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices"))
		})
	})

	Context("DRA HostDevices with feature gate disabled", func() {
		It("should reject DRA HostDevices when the feature gate is off", func() {
			checker.hostDeviceDRAEnabled = false
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices"))
		})
	})

	Context("DRA HostDevice field validation", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should reject when claimName is nil", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[0].claimName"))
		})

		It("should reject when claimName is empty string", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[0].claimName"))
		})

		It("should reject when requestName is nil", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[0].requestName"))
		})

		It("should reject when requestName is empty string", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[0].requestName"))
		})

		It("should report two causes when both claimName and requestName are missing", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name:         "hd1",
							ClaimRequest: &v1.ClaimRequest{},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[0].claimName"))
			Expect(causes[1].Field).To(Equal("spec.domain.devices.hostDevices[0].requestName"))
		})
	})

	Context("HostDevice resourceClaims cross-validation", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should reject when HostDevice claimName is not listed in spec.resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "other-claim"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
		})
	})

	Context("fully valid DRA HostDevice specs", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should accept a single valid DRA HostDevice", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
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

		It("should accept multiple valid DRA HostDevices with matching resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "claim1"},
					{Name: "claim2"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name: "hd1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "hd2",
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

	Context("duplicate claimName/requestName pairs for HostDevices", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should reject two HostDevices referencing the same claimName/requestName pair", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name: "hd1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "hd2",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[1]"))
		})

		It("should accept two HostDevices referencing the same claim but different requests", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{{Name: "claim1"}},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name: "hd1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							},
							{
								Name: "hd2",
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

	Context("mixed GPUs and HostDevices sharing resourceClaims", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
			checker.hostDeviceDRAEnabled = true
		})

		It("should accept when both GPUs and HostDevices reference claims present in resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "gpu-claim"},
					{Name: "hd-claim"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("gpu-claim"),
								RequestName: ptr.To("req1"),
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("hd-claim"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should reject when a HostDevice claim is missing from resourceClaims even if GPU claims are present", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "gpu-claim"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("gpu-claim"),
								RequestName: ptr.To("req1"),
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("hd-claim"),
								RequestName: ptr.To("req1"),
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
		})

		It("should accept when GPUs and HostDevices share the same claim", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []k8sv1.PodResourceClaim{
					{Name: "shared-claim"},
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("shared-claim"),
								RequestName: ptr.To("gpu-req"),
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   ptr.To("shared-claim"),
								RequestName: ptr.To("hd-req"),
							},
						}},
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus"))
		})
	})
})
