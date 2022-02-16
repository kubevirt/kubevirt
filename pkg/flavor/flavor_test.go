package flavor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Flavor", func() {
	var (
		flavorInformer        cache.SharedIndexInformer
		clusterFlavorInformer cache.SharedIndexInformer
		flavorMethods         flavor.Methods
	)

	BeforeEach(func() {
		flavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineFlavor{})
		clusterFlavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineClusterFlavor{})
		flavorMethods = flavor.NewMethods(flavorInformer.GetStore(), clusterFlavorInformer.GetStore())
	})

	Context("Find Flavor profile", func() {
		const (
			defaultProfileName = "default"
			customProfileName1 = "custom-profile-1"
			customProfileName2 = "custom-profile-2"
		)

		var (
			vm             *v1.VirtualMachine
			flavorProfiles []flavorv1alpha1.VirtualMachineFlavorProfile
		)

		BeforeEach(func() {
			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "test-vm-namespace",
				},
				Spec: v1.VirtualMachineSpec{
					Flavor: &v1.FlavorMatcher{},
				},
			}

			flavorProfiles = []flavorv1alpha1.VirtualMachineFlavorProfile{{
				Name:    defaultProfileName,
				Default: true,
				CPU:     &v1.CPU{Sockets: 2, Cores: 1, Threads: 1},
			}, {
				Name: customProfileName1,
				CPU:  &v1.CPU{Sockets: 4, Cores: 1, Threads: 1},
			}, {
				Name: customProfileName2,
				CPU:  &v1.CPU{Sockets: 6, Cores: 1, Threads: 1},
			}}
		})

		It("returns nil when no flavor is specified", func() {
			vm.Spec.Flavor = nil
			profile, err := flavorMethods.FindProfile(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(profile).To(BeNil())
		})

		Context("Using global ClusterFlavor", func() {
			var flavor *flavorv1alpha1.VirtualMachineClusterFlavor

			BeforeEach(func() {
				flavor = &flavorv1alpha1.VirtualMachineClusterFlavor{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-flavor",
					},
					Profiles: flavorProfiles,
				}

				err := clusterFlavorInformer.GetStore().Add(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
				}
			})

			It("should find cluster flavor if Kind is not specified", func() {
				vm.Spec.Flavor.Kind = ""

				profile, err := flavorMethods.FindProfile(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(profile).ToNot(BeNil())
			})

			It("returns default profile when no profile is specified", func() {
				vm.Spec.Flavor.Profile = ""

				profile, err := flavorMethods.FindProfile(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*profile).To(Equal(flavorProfiles[0]))
			})

			It("returns custom profile when specified", func() {
				vm.Spec.Flavor.Profile = flavorProfiles[1].Name

				profile, err := flavorMethods.FindProfile(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*profile).To(Equal(flavorProfiles[1]))
			})

			It("fails when default profile does not exist", func() {
				for i := range flavor.Profiles {
					flavor.Profiles[i].Default = false
				}

				err := clusterFlavorInformer.GetStore().Update(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor.Profile = ""

				_, err = flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flavor does not specify a default profile"))
			})

			It("fails when custom profile does not exist", func() {
				vm.Spec.Flavor.Profile = "non-existing-profile"

				_, err := flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flavor does not have a profile with name"))
			})

			It("fails when flavor does not exist", func() {
				vm.Spec.Flavor.Name = "non-existing-flavor"

				_, err := flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("Using namespaced Flavor", func() {
			var flavor *flavorv1alpha1.VirtualMachineFlavor

			BeforeEach(func() {
				flavor = &flavorv1alpha1.VirtualMachineFlavor{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-flavor",
						Namespace: vm.Namespace,
					},
					Profiles: flavorProfiles,
				}

				err := flavorInformer.GetStore().Add(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
					Kind: "VirtualMachineFlavor",
				}
			})

			It("returns default profile when no profile is specified", func() {
				vm.Spec.Flavor.Profile = ""

				profile, err := flavorMethods.FindProfile(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*profile).To(Equal(flavorProfiles[0]))
			})

			It("returns custom profile when specified", func() {
				vm.Spec.Flavor.Profile = flavorProfiles[1].Name

				profile, err := flavorMethods.FindProfile(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*profile).To(Equal(flavorProfiles[1]))
			})

			It("fails when default profile does not exist", func() {
				for i := range flavor.Profiles {
					flavor.Profiles[i].Default = false
				}

				err := flavorInformer.GetStore().Update(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor.Profile = ""

				_, err = flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flavor does not specify a default profile"))
			})

			It("fails when custom profile does not exist", func() {
				vm.Spec.Flavor.Profile = "non-existing-profile"

				_, err := flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flavor does not have a profile with name"))
			})

			It("fails when flavor does not exist", func() {
				vm.Spec.Flavor.Name = "non-existing-flavor"

				_, err := flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("fails when flavor is in different namespace", func() {
				vm.Namespace = "other-namespace"

				_, err := flavorMethods.FindProfile(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	Context("Apply flavor to VMI", func() {
		Context("CPU count", func() {
			var (
				vmiSpec *v1.VirtualMachineInstanceSpec
				profile *flavorv1alpha1.VirtualMachineFlavorProfile
			)

			BeforeEach(func() {
				vmiSpec = &v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				}

				profile = &flavorv1alpha1.VirtualMachineFlavorProfile{
					CPU: &v1.CPU{
						Sockets: 2,
						Cores:   1,
						Threads: 1,
					},
				}
			})

			It("ignores CPU count if not defined", func() {
				const vmiCpuCount = uint32(4)
				vmiSpec.Domain.CPU = &v1.CPU{
					Sockets: vmiCpuCount,
					Cores:   1,
					Threads: 1,
				}

				profile.CPU = nil

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vmiSpec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmiSpec.Domain.CPU.Sockets).To(Equal(vmiCpuCount))
				Expect(vmiSpec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmiSpec.Domain.CPU.Threads).To(Equal(uint32(1)))
			})

			It("sets CPU count", func() {
				vmiSpec.Domain.CPU = nil

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vmiSpec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmiSpec.Domain.CPU).To(Equal(profile.CPU))
			})

			It("detects CPU count conflict", func() {
				const vmiCpuCount = uint32(4)
				vmiSpec.Domain.CPU = &v1.CPU{
					Cores:   vmiCpuCount,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vmiSpec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.cpu"))
			})
		})
	})
})
