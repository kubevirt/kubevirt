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

package instancetype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/crdvalidation"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VirtualMachineInstancetype Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Minimum/Maximum validations", func() {
		It("should reject overcommitPercent above maximum of 100", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest:             resource.MustParse("1Gi"),
						OvercommitPercent: 150,
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMaximum)).ToNot(BeEmpty())
		})

		It("should reject overcommitPercent below minimum of 0", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest:             resource.MustParse("1Gi"),
						OvercommitPercent: -10,
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept overcommitPercent within valid range", func() {
			validValues := []int{0, 50, 100}
			for _, value := range validValues {
				instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest:             resource.MustParse("1Gi"),
							OvercommitPercent: value,
						},
					},
				}

				errs := validator.Validate("virtualmachineinstancetype", instancetype)
				minErrs := errs.ByType(crdvalidation.ErrorTypeMinimum)
				maxErrs := errs.ByType(crdvalidation.ErrorTypeMaximum)
				Expect(minErrs).To(BeEmpty(), "Expected %d to pass minimum validation", value)
				Expect(maxErrs).To(BeEmpty(), "Expected %d to pass maximum validation", value)
			}
		})

		It("should reject guest CPU count of 0", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 0},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept guest CPU count of 1", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})

		It("should reject maxSockets of 0", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest:      1,
						MaxSockets: pointer.P(uint32(0)),
					},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept maxSockets of 1", func() {
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest:      1,
						MaxSockets: pointer.P(uint32(1)),
					},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})
	})
})

var _ = Describe("VirtualMachineClusterInstancetype Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Minimum/Maximum validations", func() {
		It("should reject overcommitPercent above maximum of 100", func() {
			instancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest:             resource.MustParse("1Gi"),
						OvercommitPercent: 101,
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMaximum)).ToNot(BeEmpty())
		})

		It("should reject guest CPU count of 0", func() {
			instancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 0},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should reject maxSockets of 0", func() {
			instancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest:      1,
						MaxSockets: pointer.P(uint32(0)),
					},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterinstancetype", instancetype)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})
	})
})

var _ = Describe("VirtualMachinePreference Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Enum validations", func() {
		DescribeTable("should accept valid PreferredCPUTopology values", func(topology instancetypev1beta1.PreferredCPUTopology) {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(topology),
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).To(BeEmpty())
		},
			Entry("preferCores", instancetypev1beta1.DeprecatedPreferCores),
			Entry("preferSockets", instancetypev1beta1.DeprecatedPreferSockets),
			Entry("preferThreads", instancetypev1beta1.DeprecatedPreferThreads),
			Entry("preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
			Entry("preferAny", instancetypev1beta1.DeprecatedPreferAny),
			Entry("cores", instancetypev1beta1.Cores),
			Entry("sockets", instancetypev1beta1.Sockets),
			Entry("threads", instancetypev1beta1.Threads),
			Entry("spread", instancetypev1beta1.Spread),
			Entry("any", instancetypev1beta1.Any),
		)

		It("should reject invalid PreferredCPUTopology value", func() {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferredCPUTopology("invalid")),
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		DescribeTable("should accept valid SpreadAcross values", func(across instancetypev1beta1.SpreadAcross) {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(across),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).To(BeEmpty())
		},
			Entry("SocketsCoresThreads", instancetypev1beta1.SpreadAcrossSocketsCoresThreads),
			Entry("SocketsCores", instancetypev1beta1.SpreadAcrossSocketsCores),
			Entry("CoresThreads", instancetypev1beta1.SpreadAcrossCoresThreads),
		)

		It("should reject invalid SpreadAcross value", func() {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcross("InvalidSpread")),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})
	})

	Context("Minimum validations", func() {
		It("should reject spread ratio of 0", func() {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(0)),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept spread ratio of 1", func() {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(1)),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})

		It("should accept spread ratio of 2", func() {
			preference := &instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(2)),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachinepreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})
	})
})

var _ = Describe("VirtualMachineClusterPreference Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Enum validations", func() {
		It("should reject invalid PreferredCPUTopology value", func() {
			preference := &instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferredCPUTopology("invalid")),
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterpreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should reject invalid SpreadAcross value", func() {
			preference := &instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcross("InvalidSpread")),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterpreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})
	})

	Context("Minimum validations", func() {
		It("should reject spread ratio of 0", func() {
			preference := &instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(0)),
						},
					},
				},
			}

			errs := validator.Validate("virtualmachineclusterpreference", preference)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})
	})
})
