//nolint:lll,dupl
package instancetype

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Preference requirements", decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var (
		guestMemory2GB = resource.MustParse("2Gi")
		guestMemory1GB = resource.MustParse("1Gi")
		virtClient     kubecli.KubevirtClient
	)
	const (
		providedvCPUs    = 2
		requiredNumvCPUs = 2
		requiredvCPUs    = 4
		providedCores    = 2
		providedSockets  = 2
		providedThreads  = 2
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should be accepted when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *virtv1.VirtualMachine) {
		var err error
		if instancetype != nil {
			instancetype, err = virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Instancetype.Name = instancetype.Name
		}

		preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference.Name = preference.Name
		_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("VirtualMachineInstancetype meets CPU requirements",
			&instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(providedvCPUs),
					},
				},
			},
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Instancetype: &virtv1.InstancetypeMatcher{
						Kind: "VirtualMachineInstancetype",
					},
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
		),
		Entry("VirtualMachineInstancetype meets Memory requirements",
			&instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Instancetype: &virtv1.InstancetypeMatcher{
						Kind: "VirtualMachineInstancetype",
					},
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets CPU (preferSockets default) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(requiredNumvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(1),
									Threads: uint32(1),
									Sockets: uint32(providedSockets),
								},
							},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets CPU (preferCores) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(providedCores),
									Threads: uint32(1),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets 1 vCPU requirement through defaults - bug #10047",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(1),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets CPU (preferThreads) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Threads),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(1),
									Threads: uint32(providedThreads),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets CPU (preferAny) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Any),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(requiredvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(providedCores),
									Threads: uint32(1),
									Sockets: uint32(providedSockets),
								},
							},
						},
					},
				},
			},
		),
		Entry("VirtualMachine meets Memory requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Memory: &virtv1.Memory{
									Guest: &guestMemory2GB,
								},
							},
						},
					},
				},
			},
		),
	)

	DescribeTable("should be rejected when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *virtv1.VirtualMachine, errorSubString string) {
		var err error
		if instancetype != nil {
			instancetype, err = virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Instancetype.Name = instancetype.Name
		}

		preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference.Name = preference.Name
		_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failure checking preference requirements"))
		Expect(err.Error()).To(ContainSubstring(errorSubString))
	},
		Entry("VirtualMachineInstancetype does not meet CPU requirements",
			&instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(1),
					},
				},
			},
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Instancetype: &virtv1.InstancetypeMatcher{
						Kind: "VirtualMachineInstancetype",
					},
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
			"insufficient CPU resources",
		),
		Entry("VirtualMachineInstancetype does not meet Memory requirements",
			&instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
			},
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Instancetype: &virtv1.InstancetypeMatcher{
						Kind: "VirtualMachineInstancetype",
					},
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
			"insufficient Memory resources",
		),
		Entry("VirtualMachine does not meet CPU (preferSockets default) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(1),
									Threads: uint32(1),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
			"insufficient CPU resources",
		),
		Entry("VirtualMachine does not meet CPU (preferCores) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(1),
									Threads: uint32(1),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
			"insufficient CPU resources",
		),
		Entry("VirtualMachine does not meet CPU (preferThreads) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Threads),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(providedvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(1),
									Threads: uint32(1),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
			"insufficient CPU resources",
		),
		Entry("VirtualMachine does not meet CPU (preferAny) requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Any),
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(requiredvCPUs),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								CPU: &virtv1.CPU{
									Cores:   uint32(providedCores),
									Threads: uint32(1),
									Sockets: uint32(1),
								},
							},
						},
					},
				},
			},
			"insufficient CPU resources",
		),
		Entry("VirtualMachine does not meet Memory requirements",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Memory: &virtv1.Memory{
									Guest: &guestMemory1GB,
								},
							},
						},
					},
				},
			},
			"insufficient Memory resources",
		),
		Entry("VirtualMachine does not meet Memory requirements or provide any guest visible memory - bug #14551",
			nil,
			&instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
			},
			&virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
				},
				Spec: virtv1.VirtualMachineSpec{
					RunStrategy: pointer.P(virtv1.RunStrategyHalted),
					Preference: &virtv1.PreferenceMatcher{
						Kind: "VirtualMachinePreference",
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
				},
			},
			"insufficient Memory resources",
		),
	)
})
