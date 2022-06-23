package tests_test

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	flavorapi "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	flavorpkg "kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Flavor and Preferences", func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Flavor validation", func() {
		It("[test_id:TODO] should allow valid flavor", func() {
			flavor := newVirtualMachineFlavor(nil)
			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Preference validation", func() {
		It("[test_id:TODO] should allow valid preference", func() {
			preference := newVirtualMachinePreference()
			_, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
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
			Expect(cause.Message).To(HavePrefix("Failure to find flavor"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing namespaced flavor", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "non-existing-flavor",
				Kind: flavorapi.SingularResourceName,
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Failure to find flavor"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})
	})

	Context("VM with invalid PreferenceMatcher", func() {
		It("[test_id:TODO] should fail to create VM with non-existing cluster preference", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "non-existing-cluster-preference",
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Failure to find preference"))
			Expect(cause.Field).To(Equal("spec.preference"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing namespaced preference", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "non-existing-preference",
				Kind: flavorapi.SingularPreferenceResourceName,
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Failure to find preference"))
			Expect(cause.Field).To(Equal("spec.preference"))
		})
	})

	Context("Flavor and preference application", func() {

		It("[test_id:TODO] should find and apply cluster flavor and preferences when kind isn't provided", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
			)
			flavor := newVirtualMachineClusterFlavor(vmi)
			flavor, err := virtClient.VirtualMachineClusterFlavor().
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachineClusterPreference()
			preference.Spec.CPU = &flavorv1alpha1.CPUPreferences{
				PreferredCPUTopology: flavorv1alpha1.PreferSockets,
			}

			preference, err = virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:TODO] should apply flavor and preferences to VMI", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
			)

			flavor := newVirtualMachineFlavor(vmi)
			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()
			preference.Spec.CPU = &flavorv1alpha1.CPUPreferences{
				PreferredCPUTopology: flavorv1alpha1.PreferSockets,
			}
			preference.Spec.Devices = &flavorv1alpha1.DevicePreferences{
				PreferredDiskBus: v1.DiskBusSATA,
			}
			preference.Spec.Features = &flavorv1alpha1.FeaturePreferences{
				PreferredHyperv: &v1.FeatureHyperv{
					VAPIC: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
					Relaxed: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
				},
			}
			preference.Spec.Firmware = &flavorv1alpha1.FirmwarePreferences{
				PreferredUseBios: pointer.Bool(true),
			}
			// We don't want to break tests randomly so just use the q35 alias for now
			preference.Spec.Machine = &flavorv1alpha1.MachinePreferences{
				PreferredMachineType: "q35",
			}

			preference, err = virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove any requested resources from the VMI before generating the VM
			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)

			// Add the flavor and preference matchers to the VM spec
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: flavorapi.SingularResourceName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: flavorapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert we've used sockets as flavorv1alpha1.PreferSockets was requested
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(flavor.Spec.CPU.Guest))
			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(*flavor.Spec.Memory.Guest))

			// Assert that the correct disk bus is used
			for diskIndex := range vmi.Spec.Domain.Devices.Disks {
				if vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk != nil {
					Expect(vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk.Bus).To(Equal(preference.Spec.Devices.PreferredDiskBus))
				}
			}

			// Assert that the correct features are enabled
			Expect(*vmi.Spec.Domain.Features.Hyperv).To(Equal(*preference.Spec.Features.PreferredHyperv))

			// Assert that the correct firmware preferences are enabled
			Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS).ToNot(BeNil())

			// Assert that the correct machine type preference is applied to the VMI
			Expect(vmi.Spec.Domain.Machine.Type).To(Equal(preference.Spec.Machine.PreferredMachineType))

			// Assert the correct annotations have been set
			Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(flavor.Name))
			Expect(vmi.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(""))
			Expect(vmi.Annotations[v1.PreferenceAnnotation]).To(Equal(preference.Name))
			Expect(vmi.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(""))
		})

		It("[test_id:TODO] should fail if flavor and VM define CPU", func() {
			vmi := tests.NewRandomVMI()

			flavor := newVirtualMachineFlavor(vmi)
			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{Sockets: 1, Cores: 1, Threads: 1}
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: flavorapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal("VM field conflicts with selected Flavor"))
			Expect(cause.Field).To(Equal("spec.template.spec.domain.cpu"))
		})

		DescribeTable("[test_id:TODO] should fail if the VirtualMachine has ", func(resources virtv1.ResourceRequirements, expectedField string) {

			vmi := libvmi.NewCirros(libvmi.WithResourceMemory("1Mi"))
			flavor := newVirtualMachineFlavor(vmi)
			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.Resources = resources
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: flavorapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal("VM field conflicts with selected Flavor"))
			Expect(cause.Field).To(Equal(expectedField))
		},
			Entry("CPU resource requests", virtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU: resource.MustParse("1"),
				},
			}, "spec.template.spec.domain.resources.requests.cpu"),
			Entry("CPU resource limits", virtv1.ResourceRequirements{
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU: resource.MustParse("1"),
				},
			}, "spec.template.spec.domain.resources.limits.cpu"),
			Entry("Memory resource requests", virtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}, "spec.template.spec.domain.resources.requests.memory"),
			Entry("Memory resource limits", virtv1.ResourceRequirements{
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}, "spec.template.spec.domain.resources.limits.memory"),
		)

		It("[test_id:TODO] should apply preferences to default network interface", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
			)

			preference := newVirtualMachineClusterPreference()
			preference.Spec.Devices = &flavorv1alpha1.DevicePreferences{
				PreferredInterfaceModel: "virtio",
			}

			preference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(preference.Spec.Devices.PreferredInterfaceModel))
		})

		It("[test_id:TODO] should apply preferences to default volume disks", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
			)

			preference := newVirtualMachineClusterPreference()
			preference.Spec.Devices = &flavorv1alpha1.DevicePreferences{
				PreferredDiskBus: v1.DiskBusVirtio,
			}

			preference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				Expect(disk.DiskDevice.Disk.Bus).To(Equal(preference.Spec.Devices.PreferredDiskBus))
			}
		})

		It("[test_id:TODO] should store and use ControllerRevisions of VirtualMachineFlavorSpec and VirtualMachinePreferenceSpec", func() {

			var (
				flavorRevision     *appsv1.ControllerRevision
				preferenceRevision *appsv1.ControllerRevision
			)

			vmi := tests.NewRandomVMI()

			By("Creating a VirtualMachineFlavor")
			flavor := newVirtualMachineFlavor(vmi)
			originalFlavorCPUGuest := flavor.Spec.CPU.Guest
			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference := newVirtualMachinePreference()
			preference.Spec = flavorv1alpha1.VirtualMachinePreferenceSpec{
				CPU: &flavorv1alpha1.CPUPreferences{
					PreferredCPUTopology: flavorv1alpha1.PreferSockets,
				},
			}
			preference, err = virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			removeResourcesAndPreferencesFromVMI(vmi)
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: flavorapi.SingularResourceName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: flavorapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVirtualMachine(vm)

			expectedFlavorRevisionName := flavorpkg.GetRevisionName(vm.Name, flavor.Name, flavor.UID, flavor.Generation)
			By("Waiting for a VirtualMachineFlavorSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func() string {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return vm.Spec.Flavor.RevisionName
			}, 300*time.Second, 1*time.Second).Should(Equal(expectedFlavorRevisionName))

			expectedPreferenceRevisionName := flavorpkg.GetRevisionName(vm.Name, preference.Name, preference.UID, preference.Generation)
			By("Waiting for a VirtualMachinePreferenceSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func() string {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return vm.Spec.Preference.RevisionName
			}, 300*time.Second, 1*time.Second).Should(Equal(expectedPreferenceRevisionName))

			By("Checking that ControllerRevisions have been created for the VirtualMachineFlavor and VirtualMachinePreference")
			flavorRevision, err = virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), expectedFlavorRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedFlavorSpecRevision := flavorv1alpha1.VirtualMachineFlavorSpecRevision{}
			stashedFlavorSpec := flavorv1alpha1.VirtualMachineFlavorSpec{}
			Expect(json.Unmarshal(flavorRevision.Data.Raw, &stashedFlavorSpecRevision)).To(Succeed())
			Expect(stashedFlavorSpecRevision.APIVersion).To(Equal(flavor.APIVersion))
			Expect(json.Unmarshal(stashedFlavorSpecRevision.Spec, &stashedFlavorSpec)).To(Succeed())
			Expect(stashedFlavorSpec).To(Equal(flavor.Spec))

			preferenceRevision, err = virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedPreferenceSpecRevision := flavorv1alpha1.VirtualMachinePreferenceSpecRevision{}
			stashedPreferenceSpec := flavorv1alpha1.VirtualMachinePreferenceSpec{}
			Expect(json.Unmarshal(preferenceRevision.Data.Raw, &stashedPreferenceSpecRevision)).To(Succeed())
			Expect(stashedPreferenceSpecRevision.APIVersion).To(Equal(preference.APIVersion))
			Expect(json.Unmarshal(stashedPreferenceSpecRevision.Spec, &stashedPreferenceSpec)).To(Succeed())
			Expect(stashedPreferenceSpec).To(Equal(preference.Spec))

			By("Checking that a VirtualMachineInstance has been created with the VirtualMachineFlavor and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalFlavorCPUGuest))

			By("Updating the VirtualMachineFlavor vCPU count")
			newFlavorCPUGuest := originalFlavorCPUGuest + 1
			flavor.Spec.CPU.Guest = newFlavorCPUGuest
			updatedFlavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).Update(context.Background(), flavor, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedFlavor.Spec.CPU.Guest).To(Equal(newFlavorCPUGuest))

			vm = tests.StopVirtualMachine(vm)
			vm = tests.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the original VirtualMachineFlavor and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalFlavorCPUGuest))

			By("Creating a second VirtualMachine using the now updated VirtualMachineFlavor and original VirtualMachinePreference")
			newVMI := tests.NewRandomVMI()
			removeResourcesAndPreferencesFromVMI(newVMI)
			newVM := tests.NewRandomVirtualMachine(newVMI, false)
			newVM.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: flavorapi.SingularResourceName,
			}
			newVM.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: flavorapi.SingularPreferenceResourceName,
			}
			newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(newVM)
			Expect(err).ToNot(HaveOccurred())

			newVM = tests.StartVirtualMachine(newVM)

			By("Waiting for a VirtualMachineFlavorSpec ControllerRevision to be referenced from the new VirtualMachine")
			Eventually(func() string {
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return newVM.Spec.Flavor.RevisionName
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			By("Ensuring the two VirtualMachines are using different ControllerRevisions of the same VirtualMachineFlavor")
			Expect(newVM.Spec.Flavor.Name).To(Equal(vm.Spec.Flavor.Name))
			Expect(newVM.Spec.Flavor.RevisionName).ToNot(Equal(vm.Spec.Flavor.RevisionName))

			By("Checking that new ControllerRevisions for the updated VirtualMachineFlavor")
			flavorRevision, err = virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), newVM.Spec.Flavor.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedFlavorSpecRevision = flavorv1alpha1.VirtualMachineFlavorSpecRevision{}
			stashedFlavorSpec = flavorv1alpha1.VirtualMachineFlavorSpec{}
			Expect(json.Unmarshal(flavorRevision.Data.Raw, &stashedFlavorSpecRevision)).To(Succeed())
			Expect(stashedFlavorSpecRevision.APIVersion).To(Equal(updatedFlavor.APIVersion))
			Expect(json.Unmarshal(stashedFlavorSpecRevision.Spec, &stashedFlavorSpec)).To(Succeed())
			Expect(stashedFlavorSpec).To(Equal(updatedFlavor.Spec))

			By("Checking that the new VirtualMachineInstance is using the updated VirtualMachineFlavor")
			newVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(newVM.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI.Spec.Domain.CPU.Sockets).To(Equal(newFlavorCPUGuest))

		})

	})
})

func newVirtualMachineFlavor(vmi *v1.VirtualMachineInstance) *flavorv1alpha1.VirtualMachineFlavor {
	return &flavorv1alpha1.VirtualMachineFlavor{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-flavor-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: newVirtualMachineFlavorSpec(vmi),
	}
}

func newVirtualMachineClusterFlavor(vmi *v1.VirtualMachineInstance) *flavorv1alpha1.VirtualMachineClusterFlavor {
	return &flavorv1alpha1.VirtualMachineClusterFlavor{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-flavor-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: newVirtualMachineFlavorSpec(vmi),
	}
}

func newVirtualMachineFlavorSpec(vmi *v1.VirtualMachineInstance) flavorv1alpha1.VirtualMachineFlavorSpec {
	// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
	m := resource.MustParse("128M")
	if vmi != nil {
		if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
			m = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
		}
	}
	return flavorv1alpha1.VirtualMachineFlavorSpec{
		CPU: flavorv1alpha1.CPUFlavor{
			Guest: uint32(1),
		},
		Memory: flavorv1alpha1.MemoryFlavor{
			Guest: &m,
		},
	}
}

func newVirtualMachinePreference() *flavorv1alpha1.VirtualMachinePreference {
	return &flavorv1alpha1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-preference-",
			Namespace:    util.NamespaceTestDefault,
		},
	}
}

func newVirtualMachineClusterPreference() *flavorv1alpha1.VirtualMachineClusterPreference {
	return &flavorv1alpha1.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-preference-",
			Namespace:    util.NamespaceTestDefault,
		},
	}
}

func removeResourcesAndPreferencesFromVMI(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Domain.CPU = nil
	vmi.Spec.Domain.Memory = nil
	vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
	vmi.Spec.Domain.Features = &v1.Features{}
	vmi.Spec.Domain.Machine = &v1.Machine{}

	for diskIndex := range vmi.Spec.Domain.Devices.Disks {
		if vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk != nil && vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk.Bus != "" {
			vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk.Bus = ""
		}
	}
}
