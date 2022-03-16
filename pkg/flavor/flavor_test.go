package flavor_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	v1 "kubevirt.io/api/core/v1"
	apiflavor "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Flavor", func() {
	var (
		flavorInformer        cache.SharedIndexInformer
		clusterFlavorInformer cache.SharedIndexInformer
		flavorMethods         flavor.Methods
		vm                    *v1.VirtualMachine
		vmi                   *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		flavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineFlavor{})
		clusterFlavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineClusterFlavor{})
		flavorMethods = flavor.NewMethods(flavorInformer.GetStore(), clusterFlavorInformer.GetStore())
	})

	Context("Find Flavor Spec", func() {

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
		})

		It("returns nil when no flavor is specified", func() {
			vm.Spec.Flavor = nil
			spec, err := flavorMethods.FindFlavorSpec(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(BeNil())
		})

		It("returns error when invalid Flavor Kind is specified", func() {
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "foo",
				Kind: "bar",
			}
			spec, err := flavorMethods.FindFlavorSpec(vm)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("got unexpected kind in FlavorMatcher"))
			Expect(spec).To(BeNil())
		})

		Context("Using global ClusterFlavor", func() {
			var flavor *flavorv1alpha1.VirtualMachineClusterFlavor

			BeforeEach(func() {
				flavor = &flavorv1alpha1.VirtualMachineClusterFlavor{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-flavor",
					},
					Spec: flavorv1alpha1.VirtualMachineFlavorSpec{
						CPU: &v1.CPU{
							Cores: 1,
						},
					},
				}

				err := clusterFlavorInformer.GetStore().Add(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
					Kind: apiflavor.ClusterSingularResourceName,
				}
			})

			It("should find cluster flavor if Kind is not specified", func() {
				vm.Spec.Flavor.Kind = ""

				f, err := flavorMethods.FindFlavorSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*f).To(Equal(flavor.Spec))
			})

			It("returns expected flavor", func() {
				f, err := flavorMethods.FindFlavorSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*f).To(Equal(flavor.Spec))
			})

			It("fails when flavor does not exist", func() {
				vm.Spec.Flavor.Name = "non-existing-flavor"

				_, err := flavorMethods.FindFlavorSpec(vm)
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
					Spec: flavorv1alpha1.VirtualMachineFlavorSpec{
						CPU: &v1.CPU{
							Cores: 2,
						},
					},
				}

				err := flavorInformer.GetStore().Add(flavor)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
					Kind: "VirtualMachineFlavor",
				}
			})

			It("returns expected flavor", func() {
				f, err := flavorMethods.FindFlavorSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*f).To(Equal(flavor.Spec))
			})

			It("fails when flavor does not exist", func() {
				vm.Spec.Flavor.Name = "non-existing-flavor"

				_, err := flavorMethods.FindFlavorSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("fails when flavor is in different namespace", func() {
				vm.Namespace = "other-namespace"

				_, err := flavorMethods.FindFlavorSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	Context("Add flavor name annotations", func() {
		const flavorName = "flavor-name"

		BeforeEach(func() {
			vm = kubecli.NewMinimalVM("testvm")
			vm.Spec.Flavor = &v1.FlavorMatcher{Name: flavorName}
		})

		It("should add flavor name annotation", func() {
			vm.Spec.Flavor.Kind = apiflavor.SingularResourceName

			meta := &metav1.ObjectMeta{}
			flavor.AddFlavorNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.FlavorAnnotation]).To(Equal(flavorName))
			Expect(meta.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(""))
		})

		It("should add cluster flavor name annotation", func() {
			vm.Spec.Flavor.Kind = apiflavor.ClusterSingularResourceName

			meta := &metav1.ObjectMeta{}
			flavor.AddFlavorNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.FlavorAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(flavorName))
		})

		It("should add cluster name annotation, if flavor.kind is empty", func() {
			vm.Spec.Flavor.Kind = ""

			meta := &metav1.ObjectMeta{}
			flavor.AddFlavorNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.FlavorAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(flavorName))
		})
	})

	Context("Apply flavor Spec to VMI", func() {

		var (
			flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec
			field      *field.Path
		)

		BeforeEach(func() {
			vm = kubecli.NewMinimalVM("testvm")
			vm.Namespace = "test-namespace"
			vmi = api.NewMinimalVMI("testvmi")

			vmi.Spec = v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			}
			field = k8sfield.NewPath("spec", "template", "spec")
		})

		Context("Apply flavor.CPU", func() {

			It("in full to VMI", func() {

				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					CPU: &v1.CPU{
						Sockets: 2,
						Cores:   1,
						Threads: 1,
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(2)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))

			})

			It("is skipped if CPU count if not defined in spec", func() {

				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					CPU: nil,
				}

				const vmiCpuCount = uint32(4)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: vmiCpuCount,
					Cores:   1,
					Threads: 1,
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(vmiCpuCount))

			})

			It("detects CPU conflict", func() {

				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					CPU: &v1.CPU{
						Sockets: 2,
						Cores:   1,
						Threads: 1,
					},
				}

				const vmiCpuCount = uint32(4)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   vmiCpuCount,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.cpu"))

			})
		})
	})
})
