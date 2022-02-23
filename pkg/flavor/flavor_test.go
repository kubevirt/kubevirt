package flavor_test

import (
	"reflect"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

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
				Name:           defaultProfileName,
				Default:        true,
				DomainTemplate: nil,
			}, {
				Name:           customProfileName1,
				DomainTemplate: nil,
			}, {
				Name:           customProfileName2,
				DomainTemplate: nil,
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
				vm         *v1.VirtualMachine
				vmi        *v1.VirtualMachineInstance
				profile    *flavorv1alpha1.VirtualMachineFlavorProfile
				testFlavor string
			)

			BeforeEach(func() {
				vm = kubecli.NewMinimalVM("testvm")
				vmi = api.NewMinimalVMI("testvmi")

				testFlavor = "TestFlavor"
				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: testFlavor,
					Kind: "VirtualMachineFlavor",
				}

				vmi.Spec = v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				}

				profile = &flavorv1alpha1.VirtualMachineFlavorProfile{
					Name:    "default",
					Default: true,
					DomainTemplate: &flavorv1alpha1.VirtualMachineFlavorDomainTemplateSpec{
						CPU: &v1.CPU{
							Sockets: 2,
							Cores:   1,
							Threads: 1,
						},
					},
				}
			})

			It("passed empty Flavor.Kind down to the VMI expect ClusterFlavor to be used", func() {
				vm.Spec.Flavor.Kind = ""

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(2)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))

				// ClusterFlavor should be set
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(""))
				Expect(vmi.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(testFlavor))
			})

			It("passed ClusterFlavor down to the VMI", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineClusterFlavor"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(2)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))

				// Flavor should be nil
				// ClusterFlavor should be set
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(""))
				Expect(vmi.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(testFlavor))
			})

			It("ignores CPU count if not defined", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"

				const vmiCpuCount = uint32(4)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: vmiCpuCount,
					Cores:   1,
					Threads: 1,
				}

				profile.DomainTemplate = &flavorv1alpha1.VirtualMachineFlavorDomainTemplateSpec{}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(vmiCpuCount))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})

			It("sets CPU count", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"

				vmi.Spec.Domain.CPU = nil

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.CPU).To(Equal(profile.DomainTemplate.CPU))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})

			It("detects CPU count conflict", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"

				const vmiCpuCount = uint32(4)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   vmiCpuCount,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.cpu"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
		})
		Context("with DiskDefaults", func() {
			var (
				vm         *v1.VirtualMachine
				vmi        *v1.VirtualMachineInstance
				profile    *flavorv1alpha1.VirtualMachineFlavorProfile
				testFlavor string
			)

			BeforeEach(func() {
				vm = kubecli.NewMinimalVM("testvm")
				vmi = api.NewMinimalVMI("testvmi")

				testFlavor = "TestFlavor"
				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: testFlavor,
					Kind: "VirtualMachineFlavor",
				}

				profile = &flavorv1alpha1.VirtualMachineFlavorProfile{
					Name:    "default",
					Default: true,
					DevicesDefaults: &flavorv1alpha1.DevicesDefaults{
						DiskDefaults: &flavorv1alpha1.DiskDefaults{},
					},
				}
			})
			It("should apply PreferredDiskBus", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					Name: "Test",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredDiskBus = "virtio"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal("virtio"))
			})
			It("should reject PreferredDiskBus if value differs to vmi disk", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "sata",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredDiskBus = "virtio"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.diskdevice.disk.bus"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredCdromBus", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Bus: "",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredCdromBus = "sata"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.CDRom.Bus).To(Equal("sata"))
			})
			It("should reject PreferredCdromBus if value differs to vmi disk", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Bus: "virtio",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredCdromBus = "sata"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.diskdevice.cdrom.bus"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredLunBus", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Bus: "",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredLunBus = "virtio"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.LUN.Bus).To(Equal("virtio"))
			})
			It("should reject PreferredLunBus if value differs to vmi disk", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Bus: "sata",
						},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredLunBus = "virtio"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.diskdevice.lun.bus"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredCdromBus, PreferredCdromBus and PreferredLunBus", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{},
						},
					},
				}
				profile.DevicesDefaults.DiskDefaults.PreferredDiskBus = "virtio"
				profile.DevicesDefaults.DiskDefaults.PreferredCdromBus = "sata"
				profile.DevicesDefaults.DiskDefaults.PreferredLunBus = "ide"

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal("virtio"))
				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal("virtio"))
				Expect(vmi.Spec.Domain.Devices.Disks[2].DiskDevice.CDRom.Bus).To(Equal("sata"))
				Expect(vmi.Spec.Domain.Devices.Disks[3].DiskDevice.CDRom.Bus).To(Equal("sata"))
				Expect(vmi.Spec.Domain.Devices.Disks[4].DiskDevice.LUN.Bus).To(Equal("ide"))
			})
			It("should apply PreferredDedicatedIoThread", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DedicatedIOThread: pointer.BoolPtr(false),
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredDedicatedIoThread = pointer.BoolPtr(true)

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(*vmi.Spec.Domain.Devices.Disks[0].DedicatedIOThread).To(BeTrue())
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should reject PreferredDedicatedIoThread when disabled in the flavor but enabled in the vmi disk", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DedicatedIOThread: pointer.BoolPtr(true),
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredDedicatedIoThread = pointer.BoolPtr(false)

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.dedicatediothread"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredCache", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredCache = v1.CacheWriteThrough

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].Cache).To(Equal(v1.CacheWriteThrough))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should reject PreferredCache when it differs from the vmi", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					Cache: v1.CacheNone,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredCache = v1.CacheWriteThrough

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.cache"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredIo", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredIo = v1.IONative

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].IO).To(Equal(v1.IONative))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should reject PreferredIo when it differs from the vmi", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					IO: v1.IOThreads,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredIo = v1.IONative

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.io"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredBlockSize.Custom", func() {
				var logical_size uint = 4096
				var physical_size uint = 4096
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{},
					},
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredBlockSize = &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  logical_size,
						Physical: physical_size,
					},
					MatchVolume: nil,
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(vmi.Spec.Domain.Devices.Disks[0].BlockSize.Custom.Logical).To(Equal(logical_size))
				Expect(vmi.Spec.Domain.Devices.Disks[0].BlockSize.Custom.Physical).To(Equal(physical_size))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should reject PreferredBlockSize.Custom when it differs from the vmi", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  1024,
							Physical: 1024,
						},
						MatchVolume: nil,
					},
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredBlockSize = &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  4096,
						Physical: 4096,
					},
					MatchVolume: nil,
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(2))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.blocksize.custom.logical"))
				Expect(conflicts[1].String()).To(Equal("spec.domain.devices.disks.0.blocksize.custom.physical"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should apply PreferredBlockSize.MatchVolume.Enabled", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					BlockSize: &v1.BlockSize{
						MatchVolume: &v1.FeatureState{
							Enabled: pointer.BoolPtr(false),
						},
					},
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredBlockSize = &v1.BlockSize{
					Custom: nil,
					MatchVolume: &v1.FeatureState{
						Enabled: pointer.BoolPtr(true),
					},
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(0))
				Expect(*vmi.Spec.Domain.Devices.Disks[0].BlockSize.MatchVolume.Enabled).To(BeTrue())
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
			It("should reject PreferredBlockSize.MatchVolume.Enabled when disabled in the flavor but enabled in the vmi", func() {
				vm.Spec.Flavor.Kind = "VirtualMachineFlavor"
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					BlockSize: &v1.BlockSize{
						MatchVolume: &v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
					},
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{},
					},
				}}
				profile.DevicesDefaults.DiskDefaults.PreferredBlockSize = &v1.BlockSize{
					MatchVolume: &v1.FeatureState{
						Enabled: pointer.BoolPtr(false),
					},
				}

				conflicts := flavorMethods.ApplyToVmi(k8sfield.NewPath("spec"), profile, vm, vmi)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.domain.devices.disks.0.blocksize.matchvolume.enabled"))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(testFlavor))
			})
		})
	})
})

func getDeepZeroFields(field *k8sfield.Path, objVal reflect.Value) []string {
	if objVal.IsZero() {
		// If objVal is a struct with no fields, it counts as non-empty
		if objVal.Kind() == reflect.Struct && objVal.NumField() == 0 {
			return nil
		}
		return []string{field.String()}
	}

	switch objVal.Kind() {
	case reflect.Struct:
		switch obj := objVal.Interface().(type) {
		// Quantity struct should not be checked recursively
		case resource.Quantity:
			if obj.IsZero() {
				return []string{field.String()}
			}
			return nil
		default:
			var res []string
			for i := 0; i < objVal.NumField(); i++ {
				f := objVal.Field(i)
				fName := objVal.Type().Field(i).Name
				res = append(res, getDeepZeroFields(field.Child(fName), f)...)
			}
			return res
		}

	case reflect.Ptr:
		return getDeepZeroFields(field, objVal.Elem())

	case reflect.Slice, reflect.Array:
		if objVal.Len() == 0 {
			return []string{field.String()}
		}
		var res []string
		for i := 0; i < objVal.Len(); i++ {
			item := objVal.Index(i)
			res = append(res, getDeepZeroFields(field.Child(strconv.Itoa(i)), item)...)
		}
		return res

	case reflect.Map:
		if objVal.Len() == 0 {
			return []string{field.String()}
		}
		var res []string
		mapRange := objVal.MapRange()
		for mapRange.Next() {
			key := mapRange.Key()
			value := mapRange.Value()
			res = append(res, getDeepZeroFields(field.Child(key.String()), value)...)
		}
		return res
	}

	return nil
}
