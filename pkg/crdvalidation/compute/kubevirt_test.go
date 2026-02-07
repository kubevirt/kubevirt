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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/crdvalidation"
)

var _ = Describe("KubeVirt Config Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Enum validations", func() {
		It("should reject invalid vmRolloutStrategy enum value", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						VMRolloutStrategy: ptr.To(v1.VMRolloutStrategy("InvalidStrategy")),
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should accept valid vmRolloutStrategy enum values", func() {
			validStrategies := []v1.VMRolloutStrategy{v1.VMRolloutStrategyStage, v1.VMRolloutStrategyLiveUpdate}
			for _, strategy := range validStrategies {
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							VMRolloutStrategy: ptr.To(strategy),
						},
					},
				}

				errs := validator.Validate("kubevirt", kv)
				enumErrs := errs.ByType(crdvalidation.ErrorTypeEnum)
				Expect(enumErrs).To(BeEmpty(), "Expected %s to be a valid vmRolloutStrategy", strategy)
			}
		})

		It("should reject invalid minTLSVersion enum value", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						TLSConfiguration: &v1.TLSConfiguration{
							MinTLSVersion: v1.TLSProtocolVersion("InvalidVersion"),
						},
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should accept valid minTLSVersion enum values", func() {
			validVersions := []v1.TLSProtocolVersion{
				v1.TLSProtocolVersion("VersionTLS10"),
				v1.TLSProtocolVersion("VersionTLS11"),
				v1.TLSProtocolVersion("VersionTLS12"),
				v1.TLSProtocolVersion("VersionTLS13"),
			}
			for _, version := range validVersions {
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							TLSConfiguration: &v1.TLSConfiguration{
								MinTLSVersion: version,
							},
						},
					},
				}

				errs := validator.Validate("kubevirt", kv)
				enumErrs := errs.ByType(crdvalidation.ErrorTypeEnum)
				Expect(enumErrs).To(BeEmpty(), "Expected %s to be a valid minTLSVersion", version)
			}
		})

		It("should reject invalid hypervisor name enum value", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						Hypervisors: []v1.HypervisorConfiguration{
							{Name: "invalid-hypervisor"},
						},
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should accept valid hypervisor name enum values", func() {
			validNames := []string{"kvm", "hyperv-direct"}
			for _, name := range validNames {
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							Hypervisors: []v1.HypervisorConfiguration{
								{Name: name},
							},
						},
					},
				}

				errs := validator.Validate("kubevirt", kv)
				enumErrs := errs.ByType(crdvalidation.ErrorTypeEnum)
				Expect(enumErrs).To(BeEmpty(), "Expected %s to be a valid hypervisor name", name)
			}
		})
	})

	Context("Minimum validations", func() {
		It("should reject memoryOvercommit below minimum of 10", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							MemoryOvercommit: 5,
						},
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept memoryOvercommit at minimum of 10", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							MemoryOvercommit: 10,
						},
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})
	})

	Context("MaxItems validations", func() {
		It("should reject more than 1 hypervisor", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						Hypervisors: []v1.HypervisorConfiguration{
							{Name: "kvm"},
							{Name: "hyperv-direct"},
						},
					},
				},
			}

			errs := validator.Validate("kubevirt", kv)
			Expect(errs.ByType(crdvalidation.ErrorTypeMaxItems)).ToNot(BeEmpty())
		})
	})
})
