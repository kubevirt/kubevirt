package tests_test

import (
	"context"
	goerrors "errors"
	"time"

	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Flavor", func() {
	const (
		namespacedFlavorKind = "VirtualMachineFlavor"
	)

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.BeforeTestCleanup()
	})

	Context("Flavor validation", func() {
		It("[test_id:TODO] should allow valid flavor", func() {
			flavor := newVirtualMachineFlavor()
			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:TODO] should fail flavor with no profiles", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles = []flavorv1alpha1.VirtualMachineFlavorProfile{}

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})

			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(cause.Message).To(HavePrefix("A flavor must have at least one profile"))
			Expect(cause.Field).To(Equal("profiles"))
		})

		It("[test_id:TODO] should fail flavor with multiple default profiles", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles = append(flavor.Profiles, flavorv1alpha1.VirtualMachineFlavorProfile{
				Name:    "second-default",
				Default: true,
			})

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})

			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotSupported))
			Expect(cause.Message).To(HavePrefix("Flavor contains more than one default profile"))
			Expect(cause.Field).To(Equal("profiles"))
		})

		It("[test_id:TODO] should fail with DomainTemplate.Devices.Disks", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles[0].DomainTemplate.Devices.Disks = []v1.Disk{{Name: "test"}}

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})

			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotSupported))
			Expect(cause.Message).To(HavePrefix("Disks is not supported on domainTemplate.devices"))
		})

		It("[test_id:TODO] should allow DomainTemplate.Devices.UseVirtioTransitional", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles[0].DomainTemplate.Devices.UseVirtioTransitional = pointer.BoolPtr(true)

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VM with invalid FlavorMatcher", func() {
		It("[test_id:TODO] should fail to create VM with non-existing cluster flavor", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "non-existing-cluster-flavor",
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing namespaced flavor", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "non-existing-flavor",
				Kind: namespacedFlavorKind,
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing default flavor profile", func() {
			flavor := newVirtualMachineFlavor()
			for i := range flavor.Profiles {
				flavor.Profiles[i].Default = false
			}

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing custom flavor profile", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name:    flavor.Name,
				Kind:    namespacedFlavorKind,
				Profile: "nonexisting-profile",
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})
	})

	Context("Flavor application", func() {
		newVmi := func() *v1.VirtualMachineInstance {
			return tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
			)
		}

		startVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			runStrategyAlways := v1.RunStrategyAlways
			By("Starting the VirtualMachine")

			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = nil
				updatedVM.Spec.RunStrategy = &runStrategyAlways
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance created
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("VMI has the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			return updatedVM
		}

		table.DescribeTable("should apply flavor", func(getFlavorAndVmi func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance), expectedVm func(*v1.VirtualMachineInstance)) {
			flavor, vmi := getFlavorAndVmi()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			startVM(vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVm(vmi)
		},
			table.Entry("[test_id:TODO] resources requests",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Resources.Requests = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}
					vmi := newVmi()
					vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(vmi.Spec.Domain.Resources.Requests).To(
						HaveKeyWithValue(k8sv1.ResourceMemory, resource.MustParse("128Mi")),
					)
				},
			),

			table.Entry("[test_id:TODO] resources limits",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Resources.Limits = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}

					vmi := newVmi()
					vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(vmi.Spec.Domain.Resources.Limits).To(
						HaveKeyWithValue(k8sv1.ResourceMemory, resource.MustParse("128Mi")),
					)
				},
			),

			table.Entry("[test_id:TODO] CPU",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.CPU = &v1.CPU{
						Sockets: 2, Cores: 1, Threads: 1, Model: v1.DefaultCPUModel,
					}
					vmi := newVmi()
					vmi.Spec.Domain.CPU = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(vmi.Spec.Domain.CPU).To(Equal(&v1.CPU{
						Sockets: 2, Cores: 1, Threads: 1, Model: v1.DefaultCPUModel,
					}))
				},
			),

			table.Entry("[test_id:TODO] memory guest",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					memory := resource.MustParse("128Mi")
					flavor.Profiles[0].DomainTemplate.Memory = &v1.Memory{
						Guest: &memory,
					}

					vmi := newVmi()
					vmi.Spec.Domain.Memory = nil
					vmi.Spec.Domain.Resources.Requests = nil
					vmi.Spec.Domain.Resources.Limits = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					memory := resource.MustParse("128Mi")
					Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(memory))
				},
			),

			table.Entry("[test_id:TODO] memory huge pages",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Memory = &v1.Memory{
						Hugepages: &v1.Hugepages{
							PageSize: "2Mi",
						},
					}

					vmi := newVmi()
					vmi.Spec.Domain.Memory = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(vmi.Spec.Domain.Memory.Hugepages.PageSize).To(Equal("2Mi"))
				},
			),

			table.Entry("[test_id:TODO] machine",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Machine = &v1.Machine{
						Type: "q35",
					}

					vmi := newVmi()
					vmi.Spec.Domain.Machine = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(vmi.Spec.Domain.Machine.Type).To(Equal("q35"))
				},
			),

			table.Entry("[test_id:TODO] firmware",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Firmware = &v1.Firmware{
						UUID:   "6d5e3bde-8796-4364-97fe-e210ab9ff161",
						Serial: "123456",
					}

					vmi := newVmi()
					vmi.Spec.Domain.Firmware = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.Firmware).To(Equal(v1.Firmware{
						UUID:   "6d5e3bde-8796-4364-97fe-e210ab9ff161",
						Serial: "123456",
					}))
				},
			),

			table.Entry("[test_id:TODO] clock",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Clock = &v1.Clock{
						ClockOffset: v1.ClockOffset{
							UTC: &v1.ClockOffsetUTC{},
						},
						Timer: &v1.Timer{
							KVM: &v1.KVMTimer{
								Enabled: pointer.BoolPtr(true),
							},
						},
					}

					vmi := newVmi()
					vmi.Spec.Domain.Clock = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.Clock).To(Equal(v1.Clock{
						ClockOffset: v1.ClockOffset{
							UTC: &v1.ClockOffsetUTC{},
						},
						Timer: &v1.Timer{
							KVM: &v1.KVMTimer{
								Enabled: pointer.BoolPtr(true),
							},
						},
					}))
				},
			),

			table.Entry("[test_id:TODO] features",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Features = &v1.Features{
						ACPI: v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
						KVM: &v1.FeatureKVM{
							Hidden: false,
						},
						Pvspinlock: &v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
					}

					vmi := newVmi()
					vmi.Spec.Domain.Features = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.Features).To(Equal(v1.Features{
						ACPI: v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
						KVM: &v1.FeatureKVM{
							Hidden: false,
						},
						Pvspinlock: &v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
					}))
				},
			),

			table.Entry("[test_id:TODO] ioThreadPolicy",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					ioThreadPolicy := v1.IOThreadsPolicyAuto
					flavor.Profiles[0].DomainTemplate.IOThreadsPolicy = &ioThreadPolicy

					vmi := newVmi()
					vmi.Spec.Domain.IOThreadsPolicy = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(v1.IOThreadsPolicyAuto))
				},
			),

			table.Entry("[test_id:TODO] chassis",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Chassis = &v1.Chassis{
						Manufacturer: "manufacturer",
						Version:      "123",
						Serial:       "123456",
						Asset:        "asset",
						Sku:          "12345678",
					}

					vmi := newVmi()
					vmi.Spec.Domain.Chassis = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.Chassis).To(Equal(v1.Chassis{
						Manufacturer: "manufacturer",
						Version:      "123",
						Serial:       "123456",
						Asset:        "asset",
						Sku:          "12345678",
					}))
				},
			),
			table.Entry("[test_id:TODO] Devices.UseVirtioTransitional",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Devices.UseVirtioTransitional = pointer.BoolPtr(true)

					vmi := newVmi()
					vmi.Spec.Domain.Devices.UseVirtioTransitional = nil
					return flavor, vmi
				},
				func(vmi *v1.VirtualMachineInstance) {
					Expect(*vmi.Spec.Domain.Devices.UseVirtioTransitional).To(Equal(true))
				},
			),
		)

		table.DescribeTable("flavor conflicts with VM", func(conflictingField string, getFlavorAndVmi func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance)) {
			flavor, vmi := getFlavorAndVmi()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid), "Expected cause type")
			Expect(cause.Message).To(Equal("VMI field conflicts with selected Flavor profile"), "Expected cause message")
			Expect(cause.Field).To(Equal(conflictingField), "Expected conflicting field")
		},
			table.Entry("[test_id:TODO] resources requests",
				"spec.template.spec.domain.resources.requests.memory",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Resources.Requests = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}

					vmi := newVmi()
					vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] resources limits",
				"spec.template.spec.domain.resources.limits.memory",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Resources.Limits = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}

					vmi := newVmi()
					vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					}
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] CPU",
				"spec.template.spec.domain.cpu",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.CPU = &v1.CPU{Sockets: 2, Cores: 1, Threads: 1}

					vmi := tests.NewRandomVMI()
					vmi.Spec.Domain.CPU = &v1.CPU{Sockets: 1, Cores: 1, Threads: 1}

					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] memory guest",
				"spec.template.spec.domain.memory.guest",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					memory := resource.MustParse("128Mi")
					flavor.Profiles[0].DomainTemplate.Memory = &v1.Memory{
						Guest: &memory,
					}

					vmi := newVmi()
					vmi.Spec.Domain.Memory = &v1.Memory{
						Guest: &memory,
					}
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] memory huge pages",
				"spec.template.spec.domain.memory.hugepages",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					pagesize := "2Mi"
					flavor.Profiles[0].DomainTemplate.Memory = &v1.Memory{
						Hugepages: &v1.Hugepages{
							PageSize: pagesize,
						},
					}

					vmi := newVmi()
					vmi.Spec.Domain.Memory = &v1.Memory{
						Hugepages: &v1.Hugepages{
							PageSize: pagesize,
						},
					}
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] machine",
				"spec.template.spec.domain.machine",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					machine := &v1.Machine{
						Type: "q35",
					}

					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Machine = machine

					vmi := newVmi()
					vmi.Spec.Domain.Machine = machine
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] firmware",
				"spec.template.spec.domain.firmware",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					firmware := &v1.Firmware{
						UUID:   "6d5e3bde-8796-4364-97fe-e210ab9ff161",
						Serial: "123456",
					}

					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Firmware = firmware

					vmi := newVmi()
					vmi.Spec.Domain.Firmware = firmware
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] clock",
				"spec.template.spec.domain.clock",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					clock := &v1.Clock{
						ClockOffset: v1.ClockOffset{
							UTC: &v1.ClockOffsetUTC{},
						},
						Timer: &v1.Timer{
							KVM: &v1.KVMTimer{
								Enabled: pointer.BoolPtr(true),
							},
						},
					}

					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Clock = clock

					vmi := newVmi()
					vmi.Spec.Domain.Clock = clock
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] features",
				"spec.template.spec.domain.features",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					features := &v1.Features{
						ACPI: v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
						KVM: &v1.FeatureKVM{
							Hidden: false,
						},
						Pvspinlock: &v1.FeatureState{
							Enabled: pointer.BoolPtr(true),
						},
					}

					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Features = features

					vmi := newVmi()
					vmi.Spec.Domain.Features = features
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] ioThreadsPolicy",
				"spec.template.spec.domain.ioThreadsPolicy",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					flavor := newVirtualMachineFlavor()
					ioThreadPolicy := v1.IOThreadsPolicyAuto
					flavor.Profiles[0].DomainTemplate.IOThreadsPolicy = &ioThreadPolicy

					vmi := newVmi()
					vmi.Spec.Domain.IOThreadsPolicy = &ioThreadPolicy
					return flavor, vmi
				},
			),

			table.Entry("[test_id:TODO] chassis",
				"spec.template.spec.domain.chassis",
				func() (*flavorv1alpha1.VirtualMachineFlavor, *v1.VirtualMachineInstance) {
					chassis := &v1.Chassis{
						Manufacturer: "manufacturer",
						Version:      "123",
						Serial:       "123456",
						Asset:        "asset",
						Sku:          "12345678",
					}
					flavor := newVirtualMachineFlavor()
					flavor.Profiles[0].DomainTemplate.Chassis = chassis

					vmi := newVmi()
					vmi.Spec.Domain.Chassis = chassis
					return flavor, vmi
				},
			),
		)
	})
})

func newVirtualMachineFlavor() *flavorv1alpha1.VirtualMachineFlavor {
	return &flavorv1alpha1.VirtualMachineFlavor{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-flavor-",
			Namespace:    util.NamespaceTestDefault,
		},
		Profiles: []flavorv1alpha1.VirtualMachineFlavorProfile{{
			Name:           "default",
			Default:        true,
			DomainTemplate: &flavorv1alpha1.VirtualMachineFlavorDomainTemplateSpec{},
		}},
	}
}
