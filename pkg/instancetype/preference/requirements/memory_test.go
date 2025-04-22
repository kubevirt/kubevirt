package requirements_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/requirements"
)

var _ = Describe("Preferences - Requirement - Memory", func() {
	requirementsChecker := requirements.New()

	DescribeTable("should pass when sufficient resources are provided",
		func(instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec,
		) {
			conflict, err := requirementsChecker.Check(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(conflict).ToNot(HaveOccurred())
		},
		Entry("by an instance type for Memory",
			&v1beta1.VirtualMachineInstancetypeSpec{
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("1Gi"),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("1Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{},
		),
		Entry("by a VM for Memory",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("1Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
					},
				},
			},
		),
	)

	DescribeTable("should be rejected when insufficient resources are provided",
		func(instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec, expectedConflict conflict.Conflicts, errSubString string,
		) {
			conflicts, err := requirementsChecker.Check(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errSubString))
			Expect(conflicts).To(Equal(expectedConflict))
		},
		Entry("by an instance type for Memory",
			&v1beta1.VirtualMachineInstancetypeSpec{
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("1Gi"),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			nil,
			conflict.Conflicts{conflict.New("spec", "instancetype")},
			fmt.Sprintf(requirements.InsufficientInstanceTypeMemoryResourcesErrorFmt, "1Gi", "2Gi"),
		),
		Entry("by a VM for Memory",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
					},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Sprintf(requirements.InsufficientVMMemoryResourcesErrorFmt, "1Gi", "2Gi"),
		),
		Entry("by a VM without Memory - bug #14551",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Sprintf(requirements.InsufficientVMMemoryResourcesErrorFmt, "0", "2Gi"),
		),
		Entry("by a VM with Memory but without a Guest value provided - bug #14551",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Sprintf(requirements.InsufficientVMMemoryResourcesErrorFmt, "0", "2Gi"),
		),
	)
})
