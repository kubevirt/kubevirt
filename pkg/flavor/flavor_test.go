package flavor_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	v1 "kubevirt.io/api/core/v1"
	apiflavor "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Flavor and Preferences", func() {
	var (
		flavorInformer            cache.SharedIndexInformer
		clusterFlavorInformer     cache.SharedIndexInformer
		preferenceInformer        cache.SharedIndexInformer
		clusterPreferenceInformer cache.SharedIndexInformer
		flavorMethods             flavor.Methods
		vm                        *v1.VirtualMachine
		vmi                       *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		flavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineFlavor{})
		clusterFlavorInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineClusterFlavor{})
		preferenceInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachinePreference{})
		clusterPreferenceInformer, _ = testutils.NewFakeInformerFor(&flavorv1alpha1.VirtualMachineClusterPreference{})
		flavorMethods = flavor.NewMethods(flavorInformer.GetStore(), clusterFlavorInformer.GetStore(), preferenceInformer.GetStore(), clusterPreferenceInformer.GetStore())
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
						CPU: flavorv1alpha1.CPUFlavor{
							Guest: uint32(2),
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
						CPU: flavorv1alpha1.CPUFlavor{
							Guest: uint32(2),
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

	Context("Find Preference Spec", func() {

		BeforeEach(func() {
			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "test-vm-namespace",
				},
				Spec: v1.VirtualMachineSpec{
					Preference: &v1.PreferenceMatcher{},
				},
			}
		})

		It("returns nil when no preference is specified", func() {
			vm.Spec.Preference = nil
			preference, err := flavorMethods.FindPreferenceSpec(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preference).To(BeNil())
		})

		It("returns error when invalid Preference Kind is specified", func() {
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "foo",
				Kind: "bar",
			}
			spec, err := flavorMethods.FindPreferenceSpec(vm)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("got unexpected kind in PreferenceMatcher"))
			Expect(spec).To(BeNil())
		})

		Context("Using global ClusterPreference", func() {
			var preference *flavorv1alpha1.VirtualMachineClusterPreference

			BeforeEach(func() {
				preference = &flavorv1alpha1.VirtualMachineClusterPreference{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-preference",
					},
					Spec: flavorv1alpha1.VirtualMachinePreferenceSpec{
						CPU: &flavorv1alpha1.CPUPreferences{
							PreferredCPUTopology: flavorv1alpha1.PreferCores,
						},
					},
				}

				err := clusterPreferenceInformer.GetStore().Add(preference)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiflavor.ClusterSingularPreferenceResourceName,
				}
			})

			It("should find cluster preference spec if Kind is not specified", func() {
				vm.Spec.Preference.Kind = ""

				s, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(preference.Spec))
			})

			It("returns expected preference spec", func() {
				s, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(preference.Spec))
			})

			It("fails when preference does not exist", func() {
				vm.Spec.Preference.Name = "non-existing-flavor"

				_, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("Using namespaced Preference", func() {
			var preference *flavorv1alpha1.VirtualMachinePreference

			BeforeEach(func() {
				preference = &flavorv1alpha1.VirtualMachinePreference{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-preference",
						Namespace: vm.Namespace,
					},
					Spec: flavorv1alpha1.VirtualMachinePreferenceSpec{
						CPU: &flavorv1alpha1.CPUPreferences{
							PreferredCPUTopology: flavorv1alpha1.PreferCores,
						},
					},
				}

				err := preferenceInformer.GetStore().Add(preference)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiflavor.SingularPreferenceResourceName,
				}
			})

			It("returns expected preference spec", func() {
				s, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(preference.Spec))
			})

			It("fails when preference does not exist", func() {
				vm.Spec.Preference.Name = "non-existing-preference"

				_, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("fails when preference is in different namespace", func() {
				vm.Namespace = "other-namespace"

				_, err := flavorMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	Context("Add preference name annotations", func() {
		const preferenceName = "preference-name"

		BeforeEach(func() {
			vm = kubecli.NewMinimalVM("testvm")
			vm.Spec.Preference = &v1.PreferenceMatcher{Name: preferenceName}
		})

		It("should add preference name annotation", func() {
			vm.Spec.Preference.Kind = apiflavor.SingularPreferenceResourceName

			meta := &metav1.ObjectMeta{}
			flavor.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(preferenceName))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(""))
		})

		It("should add cluster preference name annotation", func() {
			vm.Spec.Preference.Kind = apiflavor.ClusterSingularPreferenceResourceName

			meta := &metav1.ObjectMeta{}
			flavor.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(preferenceName))
		})

		It("should add cluster name annotation, if preference.kind is empty", func() {
			vm.Spec.Preference.Kind = ""

			meta := &metav1.ObjectMeta{}
			flavor.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(preferenceName))
		})
	})

	Context("Apply", func() {

		var (
			flavorSpec     *flavorv1alpha1.VirtualMachineFlavorSpec
			preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec
			field          *field.Path
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

		Context("flavor.spec.CPU and preference.spec.CPU", func() {

			BeforeEach(func() {

				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					CPU: flavorv1alpha1.CPUFlavor{
						Guest:                 uint32(2),
						Model:                 "host-passthrough",
						DedicatedCPUPlacement: true,
						IsolateEmulatorThread: true,
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
						Realtime: &v1.Realtime{
							Mask: "0-3,^1",
						},
					},
				}
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					CPU: &flavorv1alpha1.CPUPreferences{},
				}
			})

			It("should default to PreferCores", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(flavorSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(flavorSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(flavorSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(flavorSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*flavorSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*flavorSpec.CPU.Realtime))

			})

			It("should apply in full with PreferCores selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = flavorv1alpha1.PreferCores

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(flavorSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(flavorSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(flavorSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(flavorSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*flavorSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*flavorSpec.CPU.Realtime))

			})

			It("should apply in full with PreferThreads selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = flavorv1alpha1.PreferThreads

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(flavorSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(flavorSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(flavorSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(flavorSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*flavorSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*flavorSpec.CPU.Realtime))

			})

			It("should apply in full with PreferSockets selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = flavorv1alpha1.PreferSockets

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(flavorSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(flavorSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(flavorSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(flavorSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*flavorSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*flavorSpec.CPU.Realtime))

			})

			It("should return a conflict if vmi.Spec.Domain.CPU already defined", func() {

				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					CPU: flavorv1alpha1.CPUFlavor{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   4,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.cpu"))

			})
		})
		Context("flavor.Spec.Memory", func() {
			BeforeEach(func() {
				flavorMem := resource.MustParse("512M")
				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					Memory: flavorv1alpha1.MemoryFlavor{
						Guest: &flavorMem,
						Hugepages: &v1.Hugepages{
							PageSize: "1Gi",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(*flavorSpec.Memory.Guest))
				Expect(*vmi.Spec.Domain.Memory.Hugepages).To(Equal(*flavorSpec.Memory.Hugepages))

			})

			It("should detect memory conflict", func() {

				vmiMemGuest := resource.MustParse("512M")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &vmiMemGuest,
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.memory"))

			})
		})
		Context("flavor.Spec.ioThreadsPolicy", func() {

			var flavorPolicy v1.IOThreadsPolicy

			BeforeEach(func() {
				flavorPolicy = v1.IOThreadsPolicyShared
				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					IOThreadsPolicy: &flavorPolicy,
				}
			})

			It("should apply to VMI", func() {
				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(*flavorSpec.IOThreadsPolicy))
			})

			It("should detect IOThreadsPolicy conflict", func() {
				vmi.Spec.Domain.IOThreadsPolicy = &flavorPolicy

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.ioThreadsPolicy"))
			})
		})

		Context("flavor.Spec.LaunchSecurity", func() {

			BeforeEach(func() {
				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					LaunchSecurity: &v1.LaunchSecurity{
						SEV: &v1.SEV{},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.LaunchSecurity).To(Equal(*flavorSpec.LaunchSecurity))
			})

			It("should detect LaunchSecurity conflict", func() {
				vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
					SEV: &v1.SEV{},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.launchSecurity"))
			})
		})

		Context("flavor.Spec.GPUs", func() {

			BeforeEach(func() {
				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					GPUs: []v1.GPU{
						v1.GPU{
							Name:       "barfoo",
							DeviceName: "vendor.com/gpu_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.GPUs).To(Equal(flavorSpec.GPUs))

			})

			It("should detect GPU conflict", func() {

				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
					v1.GPU{
						Name:       "foobar",
						DeviceName: "vendor.com/gpu_name",
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.gpus"))

			})
		})

		Context("flavor.Spec.HostDevices", func() {

			BeforeEach(func() {
				flavorSpec = &flavorv1alpha1.VirtualMachineFlavorSpec{
					HostDevices: []v1.HostDevice{
						v1.HostDevice{
							Name:       "foobar",
							DeviceName: "vendor.com/device_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.HostDevices).To(Equal(flavorSpec.HostDevices))

			})

			It("should detect HostDevice conflict", func() {

				vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
					v1.HostDevice{
						Name:       "foobar",
						DeviceName: "vendor.com/device_name",
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.hostDevices"))

			})
		})

		// TODO - break this up into smaller more targeted tests
		Context("Preference.Devices", func() {

			var userDefinedBlockSize *v1.BlockSize

			BeforeEach(func() {

				userDefinedBlockSize = &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  512,
						Physical: 512,
					},
				}
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.Bool(false)
				vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.Bool(false)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					v1.Disk{
						Cache:             v1.CacheWriteBack,
						IO:                v1.IODefault,
						DedicatedIOThread: pointer.Bool(false),
						BlockSize:         userDefinedBlockSize,
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusSCSI,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{},
						},
					},
				}
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					v1.Input{
						Bus:  "usb",
						Type: "tablet",
					},
					v1.Input{},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					v1.Interface{
						Model: "e1000",
					},
					v1.Interface{},
				}
				vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{}

				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Devices: &flavorv1alpha1.DevicePreferences{
						PreferredAutoattachGraphicsDevice:   pointer.Bool(true),
						PreferredAutoattachMemBalloon:       pointer.Bool(true),
						PreferredAutoattachPodInterface:     pointer.Bool(true),
						PreferredAutoattachSerialConsole:    pointer.Bool(true),
						PreferredDiskDedicatedIoThread:      pointer.Bool(true),
						PreferredDisableHotplug:             pointer.Bool(true),
						PreferredUseVirtioTransitional:      pointer.Bool(true),
						PreferredNetworkInterfaceMultiQueue: pointer.Bool(true),
						PreferredBlockMultiQueue:            pointer.Bool(true),
						PreferredDiskBlockSize: &v1.BlockSize{
							Custom: &v1.CustomBlockSize{
								Logical:  4096,
								Physical: 4096,
							},
						},
						PreferredDiskCache:      v1.CacheWriteThrough,
						PreferredDiskIO:         v1.IONative,
						PreferredDiskBus:        v1.DiskBusVirtio,
						PreferredCdromBus:       v1.DiskBusSCSI,
						PreferredLunBus:         v1.DiskBusSATA,
						PreferredInputBus:       "virtio",
						PreferredInputType:      "tablet",
						PreferredInterfaceModel: "virtio",
						PreferredSoundModel:     "ac97",
						PreferredRng:            &v1.Rng{},
						PreferredTPM:            &v1.TPMDevice{},
					},
				}

			})

			It("should apply to VMI", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.AutoattachMemBalloon).To(BeFalse())
				Expect(vmi.Spec.Domain.Devices.Disks[0].Cache).To(Equal(v1.CacheWriteBack))
				Expect(vmi.Spec.Domain.Devices.Disks[0].IO).To(Equal(v1.IODefault))
				Expect(*vmi.Spec.Domain.Devices.Disks[0].DedicatedIOThread).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.Disks[0].BlockSize).To(Equal(*userDefinedBlockSize))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(v1.DiskBusSCSI))
				Expect(vmi.Spec.Domain.Devices.Disks[2].DiskDevice.CDRom.Bus).To(Equal(v1.DiskBusSATA))
				Expect(vmi.Spec.Domain.Devices.Disks[4].DiskDevice.LUN.Bus).To(Equal(v1.DiskBusSATA))
				Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal("usb"))
				Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal("tablet"))
				Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal("e1000"))

				// Assert that everything that isn't defined in the VM/VMI should use Preferences
				Expect(*vmi.Spec.Domain.Devices.AutoattachPodInterface).To(Equal(*preferenceSpec.Devices.PreferredAutoattachPodInterface))
				Expect(*vmi.Spec.Domain.Devices.AutoattachSerialConsole).To(Equal(*preferenceSpec.Devices.PreferredAutoattachSerialConsole))
				Expect(vmi.Spec.Domain.Devices.DisableHotplug).To(Equal(*preferenceSpec.Devices.PreferredDisableHotplug))
				Expect(*vmi.Spec.Domain.Devices.UseVirtioTransitional).To(Equal(*preferenceSpec.Devices.PreferredUseVirtioTransitional))
				Expect(vmi.Spec.Domain.Devices.Disks[1].Cache).To(Equal(preferenceSpec.Devices.PreferredDiskCache))
				Expect(vmi.Spec.Domain.Devices.Disks[1].IO).To(Equal(preferenceSpec.Devices.PreferredDiskIO))
				Expect(*vmi.Spec.Domain.Devices.Disks[1].DedicatedIOThread).To(Equal(*preferenceSpec.Devices.PreferredDiskDedicatedIoThread))
				Expect(*vmi.Spec.Domain.Devices.Disks[1].BlockSize).To(Equal(*preferenceSpec.Devices.PreferredDiskBlockSize))
				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))
				Expect(vmi.Spec.Domain.Devices.Disks[3].DiskDevice.CDRom.Bus).To(Equal(preferenceSpec.Devices.PreferredCdromBus))
				Expect(vmi.Spec.Domain.Devices.Disks[5].DiskDevice.LUN.Bus).To(Equal(preferenceSpec.Devices.PreferredLunBus))
				Expect(vmi.Spec.Domain.Devices.Inputs[1].Bus).To(Equal(preferenceSpec.Devices.PreferredInputBus))
				Expect(vmi.Spec.Domain.Devices.Inputs[1].Type).To(Equal(preferenceSpec.Devices.PreferredInputType))
				Expect(vmi.Spec.Domain.Devices.Interfaces[1].Model).To(Equal(preferenceSpec.Devices.PreferredInterfaceModel))
				Expect(vmi.Spec.Domain.Devices.Sound.Model).To(Equal(preferenceSpec.Devices.PreferredSoundModel))
				Expect(*vmi.Spec.Domain.Devices.Rng).To(Equal(*preferenceSpec.Devices.PreferredRng))
				Expect(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue).To(Equal(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue))
				Expect(*vmi.Spec.Domain.Devices.BlockMultiQueue).To(Equal(*preferenceSpec.Devices.PreferredBlockMultiQueue))
				Expect(*vmi.Spec.Domain.Devices.TPM).To(Equal(*preferenceSpec.Devices.PreferredTPM))

			})

			It("Should apply when a VMI disk doesn't have a DiskDevice target defined", func() {

				vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk = nil

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))

			})
		})

		Context("Preference.Features", func() {

			BeforeEach(func() {
				spinLockRetries := uint32(32)
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Features: &flavorv1alpha1.FeaturePreferences{
						PreferredAcpi: &v1.FeatureState{},
						PreferredApic: &v1.FeatureAPIC{
							Enabled:        pointer.Bool(true),
							EndOfInterrupt: false,
						},
						PreferredHyperv: &v1.FeatureHyperv{
							Relaxed: &v1.FeatureState{},
							VAPIC:   &v1.FeatureState{},
							Spinlocks: &v1.FeatureSpinlocks{
								Enabled: pointer.Bool(true),
								Retries: &spinLockRetries,
							},
							VPIndex: &v1.FeatureState{},
							Runtime: &v1.FeatureState{},
							SyNIC:   &v1.FeatureState{},
							SyNICTimer: &v1.SyNICTimer{
								Enabled: pointer.Bool(true),
								Direct:  &v1.FeatureState{},
							},
							Reset: &v1.FeatureState{},
							VendorID: &v1.FeatureVendorID{
								Enabled:  pointer.Bool(true),
								VendorID: "1234",
							},
							Frequencies:     &v1.FeatureState{},
							Reenlightenment: &v1.FeatureState{},
							TLBFlush:        &v1.FeatureState{},
							IPI:             &v1.FeatureState{},
							EVMCS:           &v1.FeatureState{},
						},
						PreferredKvm: &v1.FeatureKVM{
							Hidden: true,
						},
						PreferredPvspinlock: &v1.FeatureState{},
						PreferredSmm:        &v1.FeatureState{},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Features.ACPI).To(Equal(*preferenceSpec.Features.PreferredAcpi))
				Expect(*vmi.Spec.Domain.Features.APIC).To(Equal(*preferenceSpec.Features.PreferredApic))
				Expect(*vmi.Spec.Domain.Features.Hyperv).To(Equal(*preferenceSpec.Features.PreferredHyperv))
				Expect(*vmi.Spec.Domain.Features.KVM).To(Equal(*preferenceSpec.Features.PreferredKvm))
				Expect(*vmi.Spec.Domain.Features.Pvspinlock).To(Equal(*preferenceSpec.Features.PreferredPvspinlock))
				Expect(*vmi.Spec.Domain.Features.SMM).To(Equal(*preferenceSpec.Features.PreferredSmm))
			})

			It("should apply when some HyperV features already defined in the VMI", func() {

				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &v1.FeatureHyperv{
						EVMCS: &v1.FeatureState{
							Enabled: pointer.Bool(false),
						},
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Features.Hyperv.EVMCS.Enabled).To(BeFalse())

			})
		})

		Context("Preference.Firmware", func() {

			It("should apply BIOS preferences full to VMI", func() {
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Firmware: &flavorv1alpha1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(true),
						PreferredUseBiosSerial: pointer.Bool(true),
						PreferredUseEfi:        pointer.Bool(false),
						PreferredUseSecureBoot: pointer.Bool(false),
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(Equal(*preferenceSpec.Firmware.PreferredUseBiosSerial))
			})

			It("should apply SecureBoot preferences full to VMI", func() {
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Firmware: &flavorv1alpha1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(false),
						PreferredUseBiosSerial: pointer.Bool(false),
						PreferredUseEfi:        pointer.Bool(true),
						PreferredUseSecureBoot: pointer.Bool(true),
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(Equal(*preferenceSpec.Firmware.PreferredUseSecureBoot))
			})
		})

		Context("Preference.Machine", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Machine: &flavorv1alpha1.MachinePreferences{
						PreferredMachineType: "q35-rhel-8.0",
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Machine.Type).To(Equal(preferenceSpec.Machine.PreferredMachineType))
			})
		})
		Context("Preference.Clock", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					Clock: &flavorv1alpha1.ClockPreferences{
						PreferredClockOffset: &v1.ClockOffset{
							UTC: &v1.ClockOffsetUTC{
								OffsetSeconds: pointer.Int(30),
							},
						},
						PreferredTimer: &v1.Timer{
							Hyperv: &v1.HypervTimer{},
						},
					},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*&vmi.Spec.Domain.Clock.ClockOffset).To(Equal(*preferenceSpec.Clock.PreferredClockOffset))
				Expect(*vmi.Spec.Domain.Clock.Timer).To(Equal(*preferenceSpec.Clock.PreferredTimer))
			})
		})
	})
})
