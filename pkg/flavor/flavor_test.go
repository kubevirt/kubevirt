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

	Context("Apply flavor Spec to VMI", func() {

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

		Context("Apply flavor.spec.CPU and preference.spec.CPU", func() {

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
					CPU: &flavorv1alpha1.CPUPreferences{
						PreferredCPUTopology: flavorv1alpha1.PreferCores,
					},
				}
			})

			It("in full to VMI", func() {

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

			It("detects CPU conflict", func() {

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

			It("defaults to PreferCores if no CPUPreferences are defined", func() {

				preferenceSpec = &flavorv1alpha1.VirtualMachinePreferenceSpec{
					CPU: &flavorv1alpha1.CPUPreferences{},
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(flavorSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))

			})
		})
		Context("Apply flavor.Spec.Memory", func() {
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

			It("in full to VMI", func() {

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(*flavorSpec.Memory.Guest))
				Expect(*vmi.Spec.Domain.Memory.Hugepages).To(Equal(*flavorSpec.Memory.Hugepages))

			})

			It("detects memory count conflict", func() {

				vmiMemGuest := resource.MustParse("512M")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &vmiMemGuest,
				}

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.memory"))

			})
		})

		// TODO - break this up into smaller more targeted tests
		Context("Preference.Devices ", func() {

			var userDefinedBlockSize *v1.BlockSize

			BeforeEach(func() {

				userDefinedBlockSize = &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  512,
						Physical: 512,
					},
				}
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.BoolPtr(false)
				vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.BoolPtr(false)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					v1.Disk{
						Cache:             v1.CacheWriteBack,
						IO:                v1.IODefault,
						DedicatedIOThread: pointer.BoolPtr(false),
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
						PreferredAutoattachGraphicsDevice:   pointer.BoolPtr(true),
						PreferredAutoattachMemBalloon:       pointer.BoolPtr(true),
						PreferredAutoattachPodInterface:     pointer.BoolPtr(true),
						PreferredAutoattachSerialConsole:    pointer.BoolPtr(true),
						PreferredDiskDedicatedIoThread:      pointer.BoolPtr(true),
						PreferredDisableHotplug:             pointer.BoolPtr(true),
						PreferredUseVirtioTransitional:      pointer.BoolPtr(true),
						PreferredNetworkInterfaceMultiQueue: pointer.BoolPtr(true),
						PreferredBlockMultiQueue:            pointer.BoolPtr(true),
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
					},
				}

			})

			It("in full to VMI", func() {

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

			})

			It("When passed Preference and a disk doesn't have a DiskDevice target defined", func() {

				vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk = nil

				conflicts := flavorMethods.ApplyToVmi(field, flavorSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))

			})
		})
	})
})
