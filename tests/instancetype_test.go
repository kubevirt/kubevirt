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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	k6ttypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences", func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Instancetype validation", func() {
		It("[test_id:CNV-9082] should allow valid instancetype", func() {
			instancetype := newVirtualMachineInstancetype(nil)
			_, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("[test_id:CNV-9083] should reject invalid instancetype", func(instancetype instancetypev1alpha2.VirtualMachineInstancetype) {
			_, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), &instancetype, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueRequired))
		},
			Entry("without CPU defined", instancetypev1alpha2.VirtualMachineInstancetype{
				Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1alpha2.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without CPU.Guest defined", instancetypev1alpha2.VirtualMachineInstancetype{
				Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{},
					Memory: instancetypev1alpha2.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without Memory defined", instancetypev1alpha2.VirtualMachineInstancetype{
				Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest: 1,
					},
				},
			}),
			Entry("without Memory.Guest defined", instancetypev1alpha2.VirtualMachineInstancetype{
				Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest: 1,
					},
					Memory: instancetypev1alpha2.MemoryInstancetype{},
				},
			}),
		)

	})

	Context("Preference validation", func() {
		It("[test_id:CNV-9084] should allow valid preference", func() {
			preference := newVirtualMachinePreference()
			_, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VM with invalid InstancetypeMatcher", func() {
		It("[test_id:CNV-9086] should fail to create VM with non-existing cluster instancetype", func() {
			vmi := libvmi.NewCirros()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "non-existing-cluster-instancetype",
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Failure to find instancetype"))
			Expect(cause.Field).To(Equal("spec.instancetype"))
		})

		It("[test_id:CNV-9089] should fail to create VM with non-existing namespaced instancetype", func() {
			vmi := libvmi.NewCirros()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "non-existing-instancetype",
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Failure to find instancetype"))
			Expect(cause.Field).To(Equal("spec.instancetype"))
		})
	})

	Context("VM with invalid PreferenceMatcher", func() {
		It("[test_id:CNV-9091] should fail to create VM with non-existing cluster preference", func() {
			vmi := libvmi.NewCirros()
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

		It("[test_id:CNV-9090] should fail to create VM with non-existing namespaced preference", func() {
			vmi := libvmi.NewCirros()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "non-existing-preference",
				Kind: instancetypeapi.SingularPreferenceResourceName,
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

	Context("Instancetype and preference application", func() {

		It("[test_id:CNV-9094] should find and apply cluster instancetype and preferences when kind isn't provided", func() {
			vmi := libvmi.NewCirros()

			clusterInstancetype := newVirtualMachineClusterInstancetype(vmi)
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			clusterPreference := newVirtualMachineClusterPreference()
			clusterPreference.Spec.CPU = &instancetypev1alpha2.CPUPreferences{
				PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
			}

			clusterPreference, err = virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: clusterInstancetype.Name,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:CNV-9095] should apply instancetype and preferences to VMI", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()
			preference.Spec.CPU = &instancetypev1alpha2.CPUPreferences{
				PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
			}
			preference.Spec.Devices = &instancetypev1alpha2.DevicePreferences{
				PreferredDiskBus: v1.DiskBusSATA,
			}
			preference.Spec.Features = &instancetypev1alpha2.FeaturePreferences{
				PreferredHyperv: &v1.FeatureHyperv{
					VAPIC: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
					Relaxed: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
				},
			}
			preference.Spec.Firmware = &instancetypev1alpha2.FirmwarePreferences{
				PreferredUseBios: pointer.Bool(true),
			}

			preference, err = virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove any requested resources from the VMI before generating the VM
			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)

			// Add the instancetype and preference matchers to the VM spec
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert we've used sockets as instancetypev1alpha2.PreferSockets was requested
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetype.Spec.CPU.Guest))
			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))

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

			// Assert the correct annotations have been set
			Expect(vmi.Annotations[v1.InstancetypeAnnotation]).To(Equal(instancetype.Name))
			Expect(vmi.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(""))
			Expect(vmi.Annotations[v1.PreferenceAnnotation]).To(Equal(preference.Name))
			Expect(vmi.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(""))
		})

		It("[test_id:CNV-9096] should fail if instancetype and VM define CPU", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{Sockets: 1, Cores: 1, Threads: 1}
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal("VM field conflicts with selected Instancetype"))
			Expect(cause.Field).To(Equal("spec.template.spec.domain.cpu"))
		})

		DescribeTable("[test_id:CNV-9301] should fail if the VirtualMachine has ", func(resources virtv1.ResourceRequirements, expectedField string) {

			vmi := libvmi.NewCirros()
			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.Resources = resources
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal("VM field conflicts with selected Instancetype"))
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

		It("[test_id:CNV-9302] should apply preferences to default network interface", func() {
			vmi := libvmi.NewCirros()

			clusterPreference := newVirtualMachineClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1alpha2.DevicePreferences{
				PreferredInterfaceModel: v1.VirtIO,
			}

			clusterPreference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(clusterPreference.Spec.Devices.PreferredInterfaceModel))
		})

		It("[test_id:CNV-9303] should apply preferences to default volume disks", func() {
			vmi := libvmi.NewCirros()

			clusterPreference := newVirtualMachineClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1alpha2.DevicePreferences{
				PreferredDiskBus: v1.DiskBusVirtio,
			}

			clusterPreference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				Expect(disk.DiskDevice.Disk.Bus).To(Equal(clusterPreference.Spec.Devices.PreferredDiskBus))
			}
		})

		It("[test_id:CNV-9098] should store and use ControllerRevisions of VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec", func() {
			vmi := libvmi.NewCirros()

			By("Creating a VirtualMachineInstancetype")
			instancetype := newVirtualMachineInstancetype(vmi)
			originalInstancetypeCPUGuest := instancetype.Spec.CPU.Guest
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference := newVirtualMachinePreference()
			preference.Spec = instancetypev1alpha2.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1alpha2.CPUPreferences{
					PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
				},
			}
			preference, err = virtClient.VirtualMachinePreference(util.NamespaceTestDefault).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			removeResourcesAndPreferencesFromVMI(vmi)
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
				g.Expect(vm.Spec.Preference.RevisionName).ToNot(BeEmpty())
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype := &instancetypev1alpha2.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(instancetype.Spec))

			preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), vm.Spec.Preference.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedPreference := &instancetypev1alpha2.VirtualMachinePreference{}
			Expect(json.Unmarshal(preferenceRevision.Data.Raw, stashedPreference)).To(Succeed())
			Expect(stashedPreference.Spec).To(Equal(preference.Spec))

			vm = tests.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalInstancetypeCPUGuest))

			By("Updating the VirtualMachineInstancetype vCPU count")
			newInstancetypeCPUGuest := originalInstancetypeCPUGuest + 1
			patchData, err := k6ttypes.GenerateTestReplacePatch("/spec/cpu/guest", originalInstancetypeCPUGuest, newInstancetypeCPUGuest)
			Expect(err).ToNot(HaveOccurred())
			updatedInstancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Patch(context.Background(), instancetype.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedInstancetype.Spec.CPU.Guest).To(Equal(newInstancetypeCPUGuest))

			vm = tests.StopVirtualMachine(vm)
			vm = tests.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the original VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalInstancetypeCPUGuest))

			By("Creating a second VirtualMachine using the now updated VirtualMachineInstancetype and original VirtualMachinePreference")
			newVMI := libvmi.NewCirros()
			removeResourcesAndPreferencesFromVMI(newVMI)
			newVM := tests.NewRandomVirtualMachine(newVMI, false)
			newVM.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			newVM.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}
			newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(newVM)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for a VirtualMachineInstancetypeSpec ControllerRevision to be referenced from the new VirtualMachine")
			Eventually(func() string {
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return newVM.Spec.Instancetype.RevisionName
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			By("Ensuring the two VirtualMachines are using different ControllerRevisions of the same VirtualMachineInstancetype")
			Expect(newVM.Spec.Instancetype.Name).To(Equal(vm.Spec.Instancetype.Name))
			Expect(newVM.Spec.Instancetype.RevisionName).ToNot(Equal(vm.Spec.Instancetype.RevisionName))

			By("Checking that new ControllerRevisions for the updated VirtualMachineInstancetype")
			instancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), newVM.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype = &instancetypev1alpha2.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(updatedInstancetype.Spec))

			newVM = tests.StartVirtualMachine(newVM)

			By("Checking that the new VirtualMachineInstance is using the updated VirtualMachineInstancetype")
			newVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(newVM.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI.Spec.Domain.CPU.Sockets).To(Equal(newInstancetypeCPUGuest))

		})

		It("[test_id:CNV-9304] should fail if stored ControllerRevisions are different", func() {
			vmi := libvmi.NewCirros()

			By("Creating a VirtualMachineInstancetype")
			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			removeResourcesAndPreferencesFromVMI(vmi)
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVirtualMachine(vm)

			By("Waiting for VirtualMachineInstancetypeSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
			}, 5*time.Minute, time.Second).Should(Succeed())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Stopping and removing VM")
			vm = tests.StopVirtualMachine(vm)

			err = virtClient.VirtualMachine(util.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait until ControllerRevision is deleted
			Eventually(func(g Gomega) metav1.StatusReason {
				_, err := virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Get(context.Background(), instancetypeRevision.Name, metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				return errors.ReasonForError(err)
			}, 5*time.Minute, time.Second).Should(Equal(metav1.StatusReasonNotFound))

			By("Creating changed ControllerRevision")
			stashedInstancetype := &instancetypev1alpha2.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())

			stashedInstancetype.Spec.Memory.Guest.Add(resource.MustParse("10M"))

			newInstancetypeRevision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instancetypeRevision.Name,
					Namespace: instancetypeRevision.Namespace,
				},
			}
			newInstancetypeRevision.Data.Raw, err = json.Marshal(stashedInstancetype)
			Expect(err).ToNot(HaveOccurred())

			newInstancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(util.NamespaceTestDefault).Create(context.Background(), newInstancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating and starting the VM and expecting a failure")
			newVm := tests.NewRandomVirtualMachine(vmi, true)
			newVm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			newVm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(newVm)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				foundVm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(newVm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				cond := controller.NewVirtualMachineConditionManager().
					GetCondition(foundVm, v1.VirtualMachineFailure)
				g.Expect(cond).ToNot(BeNil())
				g.Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				g.Expect(cond.Message).To(ContainSubstring("found existing ControllerRevision with unexpected data"))
			}, 5*time.Minute, time.Second).Should(Succeed())
		})
	})
})

func newVirtualMachineInstancetype(vmi *v1.VirtualMachineInstance) *instancetypev1alpha2.VirtualMachineInstancetype {
	return &instancetypev1alpha2.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-instancetype-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: newVirtualMachineInstancetypeSpec(vmi),
	}
}

func newVirtualMachineClusterInstancetype(vmi *v1.VirtualMachineInstance) *instancetypev1alpha2.VirtualMachineClusterInstancetype {
	return &instancetypev1alpha2.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-instancetype-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: newVirtualMachineInstancetypeSpec(vmi),
	}
}

func newVirtualMachineInstancetypeSpec(vmi *v1.VirtualMachineInstance) instancetypev1alpha2.VirtualMachineInstancetypeSpec {
	// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
	guestMemory := resource.MustParse("128M")
	if vmi != nil {
		if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
			guestMemory = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
		}
	}
	return instancetypev1alpha2.VirtualMachineInstancetypeSpec{
		CPU: instancetypev1alpha2.CPUInstancetype{
			Guest: uint32(1),
		},
		Memory: instancetypev1alpha2.MemoryInstancetype{
			Guest: guestMemory,
		},
	}
}

func newVirtualMachinePreference() *instancetypev1alpha2.VirtualMachinePreference {
	return &instancetypev1alpha2.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-preference-",
			Namespace:    util.NamespaceTestDefault,
		},
	}
}

func newVirtualMachineClusterPreference() *instancetypev1alpha2.VirtualMachineClusterPreference {
	return &instancetypev1alpha2.VirtualMachineClusterPreference{
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
