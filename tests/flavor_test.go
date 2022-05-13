package tests_test

import (
	"context"
	goerrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	flavorapi "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
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

		tests.BeforeTestCleanup()
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
