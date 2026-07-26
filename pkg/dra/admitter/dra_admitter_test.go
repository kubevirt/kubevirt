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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

const (
	claim1    = "claim1"
	claim2    = "claim2"
	req1      = "req1"
	req2      = "req2"
	template1 = "template1"
)

type fakeConfigChecker struct {
	gpuDRAEnabled        bool
	hostDeviceDRAEnabled bool
}

func resourceClaim(name string) v1.VirtualMachineInstanceResourceClaim {
	return v1.VirtualMachineInstanceResourceClaim{
		Name:              name,
		ResourceClaimName: new(name),
	}
}

func claimRequest(claimName, requestName string) *v1.ClaimRequest {
	return &v1.ClaimRequest{
		ClaimName:   claimName,
		RequestName: requestName,
	}
}

func gpuWithClaimRequest(name, claimName, requestName string) v1.GPU {
	return v1.GPU{
		Name:         name,
		ClaimRequest: claimRequest(claimName, requestName),
	}
}

func gpuWithDeviceName(name, deviceName string) v1.GPU {
	return v1.GPU{
		Name:       name,
		DeviceName: deviceName,
	}
}

func gpuWithDeviceNameAndClaimRequest(name, deviceName string, claimRequest *v1.ClaimRequest) v1.GPU {
	return v1.GPU{
		Name:         name,
		DeviceName:   deviceName,
		ClaimRequest: claimRequest,
	}
}

func hostDeviceWithClaimRequest(name, claimName, requestName string) v1.HostDevice {
	return v1.HostDevice{
		Name:         name,
		ClaimRequest: claimRequest(claimName, requestName),
	}
}

func hostDeviceWithDeviceName(name, deviceName string) v1.HostDevice {
	return v1.HostDevice{
		Name:       name,
		DeviceName: deviceName,
	}
}

func hostDeviceWithDeviceNameAndClaimRequest(name, deviceName string, claimRequest *v1.ClaimRequest) v1.HostDevice {
	return v1.HostDevice{
		Name:         name,
		DeviceName:   deviceName,
		ClaimRequest: claimRequest,
	}
}

func gpuSpec(resourceClaims []v1.VirtualMachineInstanceResourceClaim, gpus ...v1.GPU) *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		ResourceClaims: resourceClaims,
		Domain: v1.DomainSpec{
			Devices: v1.Devices{
				GPUs: gpus,
			},
		},
	}
}

func hostDeviceSpec(resourceClaims []v1.VirtualMachineInstanceResourceClaim, hostDevices ...v1.HostDevice) *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		ResourceClaims: resourceClaims,
		Domain: v1.DomainSpec{
			Devices: v1.Devices{
				HostDevices: hostDevices,
			},
		},
	}
}

func (f *fakeConfigChecker) GPUsWithDRAGateEnabled() bool {
	return f.gpuDRAEnabled
}

func (f *fakeConfigChecker) HostDevicesWithDRAEnabled() bool {
	return f.hostDeviceDRAEnabled
}

func (f *fakeConfigChecker) NetworkDevicesWithDRAGateEnabled() bool {
	return false
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

	Context("resource claim validation", func() {
		It("should accept a direct resource claim reference", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should accept a resource claim template reference", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					Name:                      claim1,
					ResourceClaimTemplateName: new(template1),
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should reject a resource claim without name", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					ResourceClaimName: new(claim1),
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "spec.resourceClaims[0].name is a required field",
				Field:   "spec.resourceClaims[0].name",
			}))
		})

		It("should reject a duplicate resource claim name", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
					resourceClaim(claim1),
					resourceClaim(claim1),
				},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: "duplicate resourceClaims name \"claim1\"",
				Field:   "spec.resourceClaims[1].name",
			}))
		})

		It("should reject an invalid resource claim name", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
					{
						Name:              "../claim1",
						ResourceClaimName: new(claim1),
					},
				},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(HaveField("Field", "spec.resourceClaims[0].name")))
		})

		It("should reject an invalid resourceClaimName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					Name:              claim1,
					ResourceClaimName: new("../claim1"),
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(HaveField("Field", "spec.resourceClaims[0].resourceClaimName")))
		})

		It("should reject an invalid resourceClaimTemplateName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					Name:                      claim1,
					ResourceClaimTemplateName: new("../claim-template"),
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(HaveField("Field", "spec.resourceClaims[0].resourceClaimTemplateName")))
		})

		It("should reject when both claim sources are set", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					Name:                      claim1,
					ResourceClaimName:         new(claim1),
					ResourceClaimTemplateName: new(template1),
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "at most one of resourceClaimName or resourceClaimTemplateName may be specified",
				Field:   "spec.resourceClaims[0]",
			}))
		})

		It("should reject when no claim source is set", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
					Name: claim1,
				}},
			}

			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(ContainElement(metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must specify one of: resourceClaimName, resourceClaimTemplateName",
				Field:   "spec.resourceClaims[0]",
			}))
		})
	})

	DescribeTable("non-DRA device-plugin device with deviceName",
		func(spec func() *v1.VirtualMachineInstanceSpec) {
			causes := validateCreationDRA(field, spec(), checker)
			Expect(causes).To(BeEmpty())
		},
		Entry("GPU",
			func() *v1.VirtualMachineInstanceSpec {
				return gpuSpec(nil, gpuWithDeviceName("gpu1", "vfio.gpu.example.com"))
			},
		),
		Entry("HostDevice",
			func() *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec(nil, hostDeviceWithDeviceName("hd1", "vfio.device.example.com"))
			},
		),
	)

	DescribeTable("non-DRA device-plugin device with deviceName and claimRequest",
		func(enableGate func(), spec func() *v1.VirtualMachineInstanceSpec, fieldPath string) {
			enableGate()
			causes := validateCreationDRA(field, spec(), checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("both deviceName and claimRequest"))
			Expect(causes[0].Field).To(Equal(fieldPath))
		},
		Entry("GPU",
			func() { checker.gpuDRAEnabled = true },
			func() *v1.VirtualMachineInstanceSpec {
				return gpuSpec(nil, gpuWithDeviceNameAndClaimRequest("gpu1", "vfio.gpu.example.com", claimRequest(claim1, req1)))
			},
			"spec.domain.devices.gpus",
		),
		Entry("HostDevice",
			func() { checker.hostDeviceDRAEnabled = true },
			func() *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec(nil, hostDeviceWithDeviceNameAndClaimRequest("hd1", "vfio.device.example.com", claimRequest(claim1, req1)))
			},
			"spec.domain.devices.hostDevices",
		),
	)

	Context("DRA GPUs with feature gate disabled", func() {
		It("should reject DRA GPUs when the feature gate is off", func() {
			checker.gpuDRAEnabled = false
			spec := gpuSpec(nil, gpuWithClaimRequest("gpu1", claim1, req1))
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("feature gate is not enabled"))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus"))
		})
	})

	DescribeTable("DRA device with empty claimName",
		func(enableGate func(), specForClaimRequest func(*v1.ClaimRequest) *v1.VirtualMachineInstanceSpec, fieldPrefix string) {
			enableGate()

			causes := validateCreationDRA(field, specForClaimRequest(claimRequest("", req1)), checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("claimName is required"))
			Expect(causes[0].Field).To(Equal(fieldPrefix + ".claimName"))
		},
		Entry("GPU",
			func() { checker.gpuDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return gpuSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.GPU{
					Name:         "gpu1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.gpus[0]",
		),
		Entry("HostDevice",
			func() { checker.hostDeviceDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.HostDevice{
					Name:         "hd1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.hostDevices[0]",
		),
	)

	DescribeTable("DRA device with empty requestName",
		func(enableGate func(), specForClaimRequest func(*v1.ClaimRequest) *v1.VirtualMachineInstanceSpec, fieldPrefix string) {
			enableGate()

			causes := validateCreationDRA(field, specForClaimRequest(claimRequest(claim1, "")), checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("requestName is required"))
			Expect(causes[0].Field).To(Equal(fieldPrefix + ".requestName"))
		},
		Entry("GPU",
			func() { checker.gpuDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return gpuSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.GPU{
					Name:         "gpu1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.gpus[0]",
		),
		Entry("HostDevice",
			func() { checker.hostDeviceDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.HostDevice{
					Name:         "hd1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.hostDevices[0]",
		),
	)

	DescribeTable("DRA device with empty claimName and requestName",
		func(enableGate func(), specForClaimRequest func(*v1.ClaimRequest) *v1.VirtualMachineInstanceSpec, fieldPrefix string) {
			enableGate()

			causes := validateCreationDRA(field, specForClaimRequest(&v1.ClaimRequest{}), checker)
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal(fieldPrefix + ".claimName"))
			Expect(causes[1].Field).To(Equal(fieldPrefix + ".requestName"))
		},
		Entry("GPU",
			func() { checker.gpuDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return gpuSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.GPU{
					Name:         "gpu1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.gpus[0]",
		),
		Entry("HostDevice",
			func() { checker.hostDeviceDRAEnabled = true },
			func(claimRequest *v1.ClaimRequest) *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, v1.HostDevice{
					Name:         "hd1",
					ClaimRequest: claimRequest,
				})
			},
			"spec.domain.devices.hostDevices[0]",
		),
	)

	Context("resourceClaims cross-validation", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should reject duplicate names in spec.resourceClaims", func() {
			spec := gpuSpec(
				[]v1.VirtualMachineInstanceResourceClaim{
					resourceClaim(claim1),
					resourceClaim(claim1),
				},
				gpuWithClaimRequest("gpu1", claim1, req1),
			)
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(causes[0].Message).To(ContainSubstring("duplicate resourceClaims name"))
			Expect(causes[0].Field).To(Equal("spec.resourceClaims[1].name"))
		})

		It("should reject when GPU claimName is not listed in spec.resourceClaims", func() {
			spec := gpuSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim("other-claim")},
				gpuWithClaimRequest("gpu1", claim1, req1),
			)
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
		})

		It("should reject when one of multiple claims is missing from spec.resourceClaims", func() {
			spec := gpuSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				gpuWithClaimRequest("gpu1", claim1, req1),
				gpuWithClaimRequest("gpu2", claim2, req2),
			)
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
		})
	})

	DescribeTable("valid and duplicate DRA device claimName/requestName pairs",
		func(
			enableGate func(),
			singleSpec func() *v1.VirtualMachineInstanceSpec,
			pairSpec func([]v1.VirtualMachineInstanceResourceClaim, string, string, string, string) *v1.VirtualMachineInstanceSpec,
			duplicateField string,
		) {
			enableGate()

			causes := validateCreationDRA(field, singleSpec(), checker)
			Expect(causes).To(BeEmpty())

			causes = validateCreationDRA(field, pairSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1), resourceClaim(claim2)},
				claim1,
				req1,
				claim2,
				req2,
			), checker)
			Expect(causes).To(BeEmpty())

			causes = validateCreationDRA(field, pairSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				claim1,
				req1,
				claim1,
				req1,
			), checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(causes[0].Message).To(ContainSubstring("duplicate claimName/requestName"))
			Expect(causes[0].Field).To(Equal(duplicateField))

			causes = validateCreationDRA(field, pairSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				claim1,
				req1,
				claim1,
				req2,
			), checker)
			Expect(causes).To(BeEmpty())
		},
		Entry("GPU",
			func() { checker.gpuDRAEnabled = true },
			func() *v1.VirtualMachineInstanceSpec {
				return gpuSpec([]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)}, gpuWithClaimRequest("gpu1", claim1, req1))
			},
			func(
				resourceClaims []v1.VirtualMachineInstanceResourceClaim,
				firstClaim string,
				firstRequest string,
				secondClaim string,
				secondRequest string,
			) *v1.VirtualMachineInstanceSpec {
				return gpuSpec(
					resourceClaims,
					gpuWithClaimRequest("gpu1", firstClaim, firstRequest),
					gpuWithClaimRequest("gpu2", secondClaim, secondRequest),
				)
			},
			"spec.domain.devices.gpus[1]",
		),
		Entry("HostDevice",
			func() { checker.hostDeviceDRAEnabled = true },
			func() *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec(
					[]v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
					hostDeviceWithClaimRequest("hd1", claim1, req1),
				)
			},
			func(
				resourceClaims []v1.VirtualMachineInstanceResourceClaim,
				firstClaim string,
				firstRequest string,
				secondClaim string,
				secondRequest string,
			) *v1.VirtualMachineInstanceSpec {
				return hostDeviceSpec(
					resourceClaims,
					hostDeviceWithClaimRequest("hd1", firstClaim, firstRequest),
					hostDeviceWithClaimRequest("hd2", secondClaim, secondRequest),
				)
			},
			"spec.domain.devices.hostDevices[1]",
		),
	)

	Context("mixed DRA and non-DRA GPUs", func() {
		It("should reject a VMI with both DRA and non-DRA GPUs", func() {
			checker.gpuDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name:       "dp-gpu",
								DeviceName: "vfio.gpu.example.com",
							},
							{
								Name: "dra-gpu",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(ContainSubstring("both DRA and non-DRA GPUs"))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus"))
		})
	})

	Context("GPU index accuracy", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
		})

		It("should report correct index for duplicate when an invalid DRA GPU precedes it", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name:         "gpu-invalid",
								ClaimRequest: &v1.ClaimRequest{RequestName: "req1"},
							},
							{
								Name: "gpu-a",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
								},
							},
							{
								Name: "gpu-dup",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
								},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)

			var claimNameCause, dupCause *metav1.StatusCause
			for i := range causes {
				if causes[i].Type == metav1.CauseTypeFieldValueRequired {
					claimNameCause = &causes[i]
				}
				if causes[i].Type == metav1.CauseTypeFieldValueDuplicate {
					dupCause = &causes[i]
				}
			}

			Expect(claimNameCause).NotTo(BeNil())
			Expect(claimNameCause.Field).To(Equal("spec.domain.devices.gpus[0].claimName"))

			Expect(dupCause).NotTo(BeNil())
			Expect(dupCause.Field).To(Equal("spec.domain.devices.gpus[2]"))
		})
	})

	Context("DRA HostDevices with feature gate disabled", func() {
		It("should reject DRA HostDevices when the feature gate is off", func() {
			checker.hostDeviceDRAEnabled = false
			spec := hostDeviceSpec(nil, hostDeviceWithClaimRequest("hd1", claim1, req1))
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("feature gate is not enabled"))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices"))
		})
	})

	Context("HostDevice resourceClaims cross-validation", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should reject when HostDevice claimName is not listed in spec.resourceClaims", func() {
			spec := hostDeviceSpec(
				[]v1.VirtualMachineInstanceResourceClaim{resourceClaim("other-claim")},
				hostDeviceWithClaimRequest("hd1", claim1, req1),
			)
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Message).To(ContainSubstring("resourceClaims must specify all claims"))
			Expect(causes[0].Field).To(Equal("spec.resourceClaims"))
		})
	})

	Context("mixed DRA and non-DRA HostDevices", func() {
		It("should accept a VMI with both DRA and non-DRA HostDevices", func() {
			checker.hostDeviceDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "dp-hd",
								DeviceName: "vfio.device.example.com",
							},
							{
								Name: "dra-hd",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
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

	Context("HostDevice index accuracy", func() {
		BeforeEach(func() {
			checker.hostDeviceDRAEnabled = true
		})

		It("should report correct index when non-DRA devices precede a DRA device with missing claimName", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "dp-hd",
								DeviceName: "vfio.device.example.com",
							},
							{
								Name:         "dra-hd-invalid",
								ClaimRequest: &v1.ClaimRequest{RequestName: "req1"},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(causes[0].Message).To(ContainSubstring("claimName is required"))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[1].claimName"))
		})

		It("should report correct index for duplicate when non-DRA devices are interspersed", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "dp-hd",
								DeviceName: "vfio.device.example.com",
							},
							{
								Name: "dra-hd-a",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
								},
							},
							{
								Name:       "dp-hd-2",
								DeviceName: "another.device.example.com",
							},
							{
								Name: "dra-hd-dup",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   claim1,
									RequestName: "req1",
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
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[3]"))
		})

		It("should report correct index for invalid DRA HostDevice after non-DRA devices", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "dp-hd-1",
								DeviceName: "vfio.device1.example.com",
							},
							{
								Name:       "dp-hd-2",
								DeviceName: "vfio.device2.example.com",
							},
							{
								Name:         "dra-hd-invalid",
								ClaimRequest: &v1.ClaimRequest{},
							},
						},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal("spec.domain.devices.hostDevices[2].claimName"))
			Expect(causes[1].Field).To(Equal("spec.domain.devices.hostDevices[2].requestName"))
		})
	})

	Context("mixed GPUs and HostDevices sharing resourceClaims", func() {
		BeforeEach(func() {
			checker.gpuDRAEnabled = true
			checker.hostDeviceDRAEnabled = true
		})

		It("should accept when both GPUs and HostDevices reference claims present in resourceClaims", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
					resourceClaim("gpu-claim"),
					resourceClaim("hd-claim"),
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "gpu-claim",
								RequestName: "req1",
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "hd-claim",
								RequestName: "req1",
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
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
					resourceClaim("gpu-claim"),
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "gpu-claim",
								RequestName: "req1",
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "hd-claim",
								RequestName: "req1",
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
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
					resourceClaim("shared-claim"),
				},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "shared-claim",
								RequestName: "gpu-req",
							},
						}},
						HostDevices: []v1.HostDevice{{
							Name: "hd1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   "shared-claim",
								RequestName: "hd-req",
							},
						}},
					},
				},
			}
			causes := validateCreationDRA(field, spec, checker)
			Expect(causes).To(BeEmpty())
		})

		It("should reject when GPU and HostDevice share the same claimName/requestName pair", func() {
			vmi := libvmi.New(
				libvmi.WithResourceClaim(resourceClaim("shared-claim")),
				libvmi.WithGPU(v1.GPU{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   "shared-claim",
						RequestName: "shared-req",
					},
				}),
				libvmi.WithHostDevice(v1.HostDevice{
					Name: "hd1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   "shared-claim",
						RequestName: "shared-req",
					},
				}),
			)
			causes := validateCreationDRA(field, &vmi.Spec, checker)
			Expect(causes).To(Equal([]metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: "duplicate claimName/requestName pair \"shared-claim/shared-req\" between GPUs[0] and HostDevices[0]",
				Field:   "spec.domain.devices.hostDevices[0]",
			}}))
		})
	})

	Context("Validator methods", func() {
		It("ValidateCreation should delegate to validateCreationDRA", func() {
			checker.gpuDRAEnabled = true
			spec := &v1.VirtualMachineInstanceSpec{
				ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{resourceClaim(claim1)},
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{{
							Name: "gpu1",
							ClaimRequest: &v1.ClaimRequest{
								ClaimName:   claim1,
								RequestName: "req1",
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
								ClaimName:   claim1,
								RequestName: "req1",
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
