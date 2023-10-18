package tests_test

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	instancetypepkg "kubevirt.io/kubevirt/pkg/instancetype"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences", decorators.SigCompute, func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Instancetype validation", func() {
		It("[test_id:CNV-9082] should allow valid instancetype", func() {
			instancetype := newVirtualMachineInstancetype(nil)
			_, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("[test_id:CNV-9083] should reject invalid instancetype", func(instancetype instancetypev1beta1.VirtualMachineInstancetype) {
			_, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(nil)).
				Create(context.Background(), &instancetype, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueRequired))
		},
			Entry("without CPU defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without CPU.Guest defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without Memory defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: 1,
					},
				},
			}),
			Entry("without Memory.Guest defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: 1,
					},
					Memory: instancetypev1beta1.MemoryInstancetype{},
				},
			}),
		)

	})

	Context("Preference validation", func() {
		It("[test_id:CNV-9084] should allow valid preference", func() {
			preference := newVirtualMachinePreference()
			_, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
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

			_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
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

			_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
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

			_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
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

			_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
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
	Context("[Serial]with cluster memory overcommit being applied", Serial, func() {
		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			config.DeveloperConfiguration.MemoryOvercommit = 200
			tests.UpdateKubeVirtConfigValueAndWait(config)
		})
		It("should apply memory overcommit instancetype to VMI even with cluster overcommit set", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype.Spec.Memory.OvercommitPercent = 15

			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
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

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))
			expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
			Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
			memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
			Expect(memRequest.Value()).To(Equal(expectedOverhead))

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
			preferredCPUTopology := instancetypev1beta1.PreferSockets
			clusterPreference.Spec.CPU = &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: &preferredCPUTopology,
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

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:CNV-9095] should apply instancetype and preferences to VMI", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype.Spec.Annotations = map[string]string{
				"required-annotation-1": "1",
				"required-annotation-2": "2",
			}
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()
			preferredCPUTopology := instancetypev1beta1.PreferSockets
			preference.Spec.CPU = &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: &preferredCPUTopology,
			}
			preference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredDiskBus: v1.DiskBusSATA,
			}
			preference.Spec.Features = &instancetypev1beta1.FeaturePreferences{
				PreferredHyperv: &v1.FeatureHyperv{
					VAPIC: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
					Relaxed: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
				},
			}
			preference.Spec.Firmware = &instancetypev1beta1.FirmwarePreferences{
				PreferredUseBios: pointer.Bool(true),
			}
			preference.Spec.PreferredTerminationGracePeriodSeconds = pointer.Int64(15)
			preference.Spec.PreferredSubdomain = pointer.String("non-existent-subdomain")
			preference.Spec.Annotations = map[string]string{
				"preferred-annotation-1": "1",
				"preferred-annotation-2": "use-vm-value",
				"required-annotation-1":  "use-instancetype-value",
			}

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove any requested resources from the VMI before generating the VM
			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{
				"preferred-annotation-2": "2",
			}

			// Add the instancetype and preference matchers to the VM spec
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert we've used sockets as instancetypev1beta1.PreferSockets was requested
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

			// Assert that the correct termination grace period are used
			Expect(*vmi.Spec.TerminationGracePeriodSeconds).To(Equal(*preference.Spec.PreferredTerminationGracePeriodSeconds))

			// Assert that the correct subdomain grace period are used
			Expect(vmi.Spec.Subdomain).To(Equal(*preference.Spec.PreferredSubdomain))

			// Assert the correct annotations have been set
			Expect(vmi.Annotations[v1.InstancetypeAnnotation]).To(Equal(instancetype.Name))
			Expect(vmi.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(""))
			Expect(vmi.Annotations[v1.PreferenceAnnotation]).To(Equal(preference.Name))
			Expect(vmi.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(""))
			Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-1", "1"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-2", "2"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-1", "1"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-2", "2"))
		})
		It("should apply memory overcommit instancetype to VMI", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype.Spec.Memory.OvercommitPercent = 15

			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
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

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))

			expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
			Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
			memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
			Expect(memRequest.Value()).To(Equal(expectedOverhead))

		})

		It("[test_id:CNV-9096] should fail if instancetype and VM define CPU", func() {
			vmi := libvmi.NewCirros()

			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{Sockets: 1, Cores: 1, Threads: 1}
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(3))

			cause0 := apiStatus.Status().Details.Causes[0]
			Expect(cause0.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			cpuSocketsField := "spec.template.spec.domain.cpu.sockets"
			Expect(cause0.Message).To(Equal(fmt.Sprintf(instancetypepkg.VMFieldConflictErrorFmt, cpuSocketsField)))
			Expect(cause0.Field).To(Equal(cpuSocketsField))

			cause1 := apiStatus.Status().Details.Causes[1]
			Expect(cause1.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			cpuCoresField := "spec.template.spec.domain.cpu.cores"
			Expect(cause1.Message).To(Equal(fmt.Sprintf(instancetypepkg.VMFieldConflictErrorFmt, cpuCoresField)))
			Expect(cause1.Field).To(Equal(cpuCoresField))

			cause2 := apiStatus.Status().Details.Causes[2]
			Expect(cause2.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			cpuThreadsField := "spec.template.spec.domain.cpu.threads"
			Expect(cause2.Message).To(Equal(fmt.Sprintf(instancetypepkg.VMFieldConflictErrorFmt, cpuThreadsField)))
			Expect(cause2.Field).To(Equal(cpuThreadsField))
		})

		DescribeTable("[test_id:CNV-9301] should fail if the VirtualMachine has ", func(resources virtv1.ResourceRequirements, expectedField string) {

			vmi := libvmi.NewCirros()
			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Template.Spec.Domain.Resources = resources
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]

			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal(fmt.Sprintf(instancetypepkg.VMFieldConflictErrorFmt, expectedField)))
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
			clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredInterfaceModel: v1.VirtIO,
			}

			clusterPreference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(clusterPreference.Spec.Devices.PreferredInterfaceModel))
		})

		It("[test_id:CNV-9303] should apply preferences to default volume disks", func() {
			vmi := libvmi.NewCirros()

			clusterPreference := newVirtualMachineClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
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

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference := newVirtualMachinePreference()
			preferredCPUTopology := instancetypev1beta1.PreferSockets
			preference.Spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
				},
			}
			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
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

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
				g.Expect(vm.Spec.Preference.RevisionName).ToNot(BeEmpty())
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype := &instancetypev1beta1.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(instancetype.Spec))

			preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Spec.Preference.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedPreference := &instancetypev1beta1.VirtualMachinePreference{}
			Expect(json.Unmarshal(preferenceRevision.Data.Raw, stashedPreference)).To(Succeed())
			Expect(stashedPreference.Spec).To(Equal(preference.Spec))

			vm = tests.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalInstancetypeCPUGuest))

			By("Updating the VirtualMachineInstancetype vCPU count")
			newInstancetypeCPUGuest := originalInstancetypeCPUGuest + 1
			patchData, err := patch.GenerateTestReplacePatch("/spec/cpu/guest", originalInstancetypeCPUGuest, newInstancetypeCPUGuest)
			Expect(err).ToNot(HaveOccurred())
			updatedInstancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Patch(context.Background(), instancetype.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedInstancetype.Spec.CPU.Guest).To(Equal(newInstancetypeCPUGuest))

			vm = tests.StopVirtualMachine(vm)
			vm = tests.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the original VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
			newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), newVM)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for a VirtualMachineInstancetypeSpec ControllerRevision to be referenced from the new VirtualMachine")
			Eventually(func() string {
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(context.Background(), newVM.Name, &metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return newVM.Spec.Instancetype.RevisionName
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			By("Ensuring the two VirtualMachines are using different ControllerRevisions of the same VirtualMachineInstancetype")
			Expect(newVM.Spec.Instancetype.Name).To(Equal(vm.Spec.Instancetype.Name))
			Expect(newVM.Spec.Instancetype.RevisionName).ToNot(Equal(vm.Spec.Instancetype.RevisionName))

			By("Checking that new ControllerRevisions for the updated VirtualMachineInstancetype")
			instancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), newVM.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype = &instancetypev1beta1.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(updatedInstancetype.Spec))

			newVM = tests.StartVirtualMachine(newVM)

			By("Checking that the new VirtualMachineInstance is using the updated VirtualMachineInstancetype")
			newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), newVM.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI.Spec.Domain.CPU.Sockets).To(Equal(newInstancetypeCPUGuest))

		})

		It("[test_id:CNV-9304] should fail if stored ControllerRevisions are different", func() {
			vmi := libvmi.NewCirros()

			By("Creating a VirtualMachineInstancetype")
			instancetype := newVirtualMachineInstancetype(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			removeResourcesAndPreferencesFromVMI(vmi)
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVirtualMachine(vm)

			By("Waiting for VirtualMachineInstancetypeSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
			}, 5*time.Minute, time.Second).Should(Succeed())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Stopping and removing VM")
			vm = tests.StopVirtualMachine(vm)

			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait until ControllerRevision is deleted
			Eventually(func(g Gomega) metav1.StatusReason {
				_, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(instancetype)).Get(context.Background(), instancetypeRevision.Name, metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				return errors.ReasonForError(err)
			}, 5*time.Minute, time.Second).Should(Equal(metav1.StatusReasonNotFound))

			By("Creating changed ControllerRevision")
			stashedInstancetype := &instancetypev1beta1.VirtualMachineInstancetype{}
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

			newInstancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), newInstancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating and starting the VM and expecting a failure")
			newVm := tests.NewRandomVirtualMachine(vmi, true)
			newVm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			newVm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), newVm)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				foundVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), newVm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				cond := controller.NewVirtualMachineConditionManager().
					GetCondition(foundVm, v1.VirtualMachineFailure)
				g.Expect(cond).ToNot(BeNil())
				g.Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				g.Expect(cond.Message).To(ContainSubstring("found existing ControllerRevision with unexpected data"))
			}, 5*time.Minute, time.Second).Should(Succeed())
		})

		Context("deprecated API versions", func() {
			const expectedCores = uint32(4)
			var expectedMemory = resource.MustParse("256Mi")

			getV1Alpha1VirtualMachineInstancetypeSpecRevisionBytes := func(withAPIVersion bool) ([]byte, []byte) {
				instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha1.CPUInstancetype{
						Guest: expectedCores,
					},
					Memory: instancetypev1alpha1.MemoryInstancetype{
						Guest: expectedMemory,
					},
				}

				instancetypeSpecBytes, err := json.Marshal(&instancetypeSpec)
				Expect(err).ToNot(HaveOccurred())

				instancetypeSpecRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: "",
					Spec:       instancetypeSpecBytes,
				}
				if withAPIVersion {
					instancetypeSpecRevision.APIVersion = instancetypev1alpha1.SchemeGroupVersion.String()
				}
				instancetypeSpecRevisionBytes, err := json.Marshal(instancetypeSpecRevision)
				Expect(err).ToNot(HaveOccurred())

				preferenceSpec := &instancetypev1alpha1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1alpha1.CPUPreferences{
						PreferredCPUTopology: instancetypev1alpha1.PreferCores,
					},
				}

				preferenceSpecBytes, err := json.Marshal(&preferenceSpec)
				Expect(err).ToNot(HaveOccurred())

				preferenceSpecRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
					APIVersion: "",
					Spec:       preferenceSpecBytes,
				}
				if withAPIVersion {
					preferenceSpecRevision.APIVersion = instancetypev1alpha1.SchemeGroupVersion.String()
				}
				preferenceSpecRevisionBytes, err := json.Marshal(preferenceSpecRevision)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeSpecRevisionBytes, preferenceSpecRevisionBytes
			}

			DescribeTable("should be able to use ControllerRevisions containing ", func(getRevisionData func() ([]byte, []byte)) {
				namespace := testsuite.GetTestNamespace(nil)
				instancetypeBytes, preferenceBytes := getRevisionData()

				instancetypeRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype-revision-",
					},
					Data: runtime.RawExtension{
						Raw: instancetypeBytes,
					},
				}

				instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				preferenceRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference-revision-",
					},
					Data: runtime.RawExtension{
						Raw: preferenceBytes,
					},
				}

				preferenceRevision, err = virtClient.AppsV1().ControllerRevisions(namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := libvmi.NewCirros()
				removeResourcesAndPreferencesFromVMI(vmi)
				vm := tests.NewRandomVirtualMachine(vmi, false)
				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name:         "dummy",
					RevisionName: instancetypeRevision.Name,
				}
				vm.Spec.Preference = &virtv1.PreferenceMatcher{
					Name:         "dummy",
					RevisionName: preferenceRevision.Name,
				}

				vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
				vm = tests.StartVirtualMachine(vm)

				vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(expectedCores))
				Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(expectedMemory))
			},
				Entry("v1alpha1 VirtualMachineInstancetypeSpecRevisions with APIVersion", func() ([]byte, []byte) {
					return getV1Alpha1VirtualMachineInstancetypeSpecRevisionBytes(true)
				}),
				Entry("v1alpha1 VirtualMachineInstancetypeSpecRevisions without APIVersion", func() ([]byte, []byte) {
					return getV1Alpha1VirtualMachineInstancetypeSpecRevisionBytes(false)
				}),
				Entry("v1alpha1 VirtualMachineInstancetype and VirtualMachinePreference", func() ([]byte, []byte) {
					instancetype := instancetypev1alpha1.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineInstancetype",
						},
						Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: expectedCores,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: expectedMemory,
							},
						},
					}
					instancetypeBytes, err := json.Marshal(instancetype)
					Expect(err).ToNot(HaveOccurred())

					preference := instancetypev1alpha1.VirtualMachinePreference{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachinePreference",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachinePreference",
						},
						Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha1.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha1.PreferCores,
							},
						},
					}
					preferenceBytes, err := json.Marshal(preference)
					Expect(err).ToNot(HaveOccurred())

					return instancetypeBytes, preferenceBytes
				}),
				Entry("v1alpha1 VirtualMachineClusterInstancetype and VirtualMachineClusterPreference", func() ([]byte, []byte) {
					instancetype := instancetypev1alpha1.VirtualMachineClusterInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineClusterInstancetype",
						},
						Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: expectedCores,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: expectedMemory,
							},
						},
					}
					instancetypeBytes, err := json.Marshal(instancetype)
					Expect(err).ToNot(HaveOccurred())

					preference := instancetypev1alpha1.VirtualMachineClusterPreference{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterPreference",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineClusterPreference",
						},
						Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha1.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha1.PreferCores,
							},
						},
					}
					preferenceBytes, err := json.Marshal(preference)
					Expect(err).ToNot(HaveOccurred())

					return instancetypeBytes, preferenceBytes
				}),
				Entry("v1alpha2 VirtualMachineInstancetype and VirtualMachinePreference", func() ([]byte, []byte) {
					instancetype := instancetypev1alpha2.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineInstancetype",
						},
						Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha2.CPUInstancetype{
								Guest: expectedCores,
							},
							Memory: instancetypev1alpha2.MemoryInstancetype{
								Guest: expectedMemory,
							},
						},
					}
					instancetypeBytes, err := json.Marshal(instancetype)
					Expect(err).ToNot(HaveOccurred())

					preference := instancetypev1alpha2.VirtualMachinePreference{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
							Kind:       "VirtualMachinePreference",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachinePreference",
						},
						Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha2.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha2.PreferCores,
							},
						},
					}
					preferenceBytes, err := json.Marshal(preference)
					Expect(err).ToNot(HaveOccurred())

					return instancetypeBytes, preferenceBytes
				}),
				Entry("v1alpha2 VirtualMachineClusterInstancetype and VirtualMachineClusterPreference", func() ([]byte, []byte) {
					instancetype := instancetypev1alpha2.VirtualMachineClusterInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineClusterInstancetype",
						},
						Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha2.CPUInstancetype{
								Guest: expectedCores,
							},
							Memory: instancetypev1alpha2.MemoryInstancetype{
								Guest: expectedMemory,
							},
						},
					}
					instancetypeBytes, err := json.Marshal(instancetype)
					Expect(err).ToNot(HaveOccurred())

					preference := instancetypev1alpha2.VirtualMachineClusterPreference{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineClusterPreference",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "VirtualMachineClusterPreference",
						},
						Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha2.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha2.PreferCores,
							},
						},
					}
					preferenceBytes, err := json.Marshal(preference)
					Expect(err).ToNot(HaveOccurred())

					return instancetypeBytes, preferenceBytes
				}),
			)
		})
	})

	Context("with inferFromVolume", func() {
		var (
			err          error
			vm           *v1.VirtualMachine
			instancetype *instancetypev1beta1.VirtualMachineInstancetype
			preference   *instancetypev1beta1.VirtualMachinePreference
			sourceDV     *cdiv1beta1.DataVolume
			namespace    string
		)

		const (
			inferFromVolumeName    = "volume"
			dataVolumeTemplateName = "datatemplate"
		)

		createAndValidateVirtualMachine := func() {
			By("Creating the VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Validating the VirtualMachine")
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
			Expect(vm.Spec.Instancetype.Kind).To(Equal(instancetypeapi.SingularResourceName))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
			Expect(vm.Spec.Preference.Kind).To(Equal(instancetypeapi.SingularPreferenceResourceName))
			Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

			vm = tests.StartVMAndExpectRunning(virtClient, vm)

			By("Validating the VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetype.Spec.CPU.Guest))
		}

		generateDataVolumeTemplatesFromDataVolume := func(dataVolume *cdiv1beta1.DataVolume) []v1.DataVolumeTemplateSpec {
			return []v1.DataVolumeTemplateSpec{{
				ObjectMeta: metav1.ObjectMeta{
					Name: dataVolumeTemplateName,
				},
				Spec: dataVolume.Spec,
			}}
		}

		generateVolumesForDataVolumeTemplates := func() []v1.Volume {
			return []v1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dataVolumeTemplateName,
					},
				},
			}}
		}

		BeforeEach(func() {
			if !libstorage.HasCDI() {
				Skip("instance type and preference inferFromVolume tests require CDI to be installed providing the DataVolume and DataSource CRDs")
			}

			namespace = testsuite.GetTestNamespace(nil)

			By("Creating a VirtualMachineInstancetype")
			instancetype = newVirtualMachineInstancetype(nil)
			instancetype, err = virtClient.VirtualMachineInstancetype(namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference = newVirtualMachinePreference()
			preferredCPUTopology := instancetypev1beta1.PreferCores
			preference.Spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
				},
			}
			preference, err = virtClient.VirtualMachinePreference(namespace).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Creating source DataVolume and PVC")
			sourceDV = libdv.NewDataVolume(
				libdv.WithNamespace(namespace),
				libdv.WithForceBindAnnotation(),
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
			)
			// TODO - Add withDefault{Instancetype,Preference}Label support to libdv
			sourceDV.Labels = map[string]string{
				instancetypeapi.DefaultInstancetypeLabel:     instancetype.Name,
				instancetypeapi.DefaultInstancetypeKindLabel: instancetypeapi.SingularResourceName,
				instancetypeapi.DefaultPreferenceLabel:       preference.Name,
				instancetypeapi.DefaultPreferenceKindLabel:   instancetypeapi.SingularPreferenceResourceName,
			}
			sourceDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, 180, HaveSucceeded())

			// This is the default but it should still be cleared
			failurePolicy := v1.RejectInferFromVolumeFailure

			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
					Namespace:    namespace,
				},
				Spec: v1.VirtualMachineSpec{
					Instancetype: &v1.InstancetypeMatcher{
						InferFromVolume:              inferFromVolumeName,
						InferFromVolumeFailurePolicy: &failurePolicy,
					},
					Preference: &v1.PreferenceMatcher{
						InferFromVolume:              inferFromVolumeName,
						InferFromVolumeFailurePolicy: &failurePolicy,
					},
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{},
						},
					},
					Running: pointer.Bool(false),
				},
			}
		})

		It("should infer defaults from PersistentVolumeClaimVolumeSource", func() {
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: sourceDV.Name,
						},
					},
				},
			}}
			createAndValidateVirtualMachine()
		})

		It("should infer defaults from existing DataVolume with labels", func() {
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: sourceDV.Name,
					},
				},
			}}
			createAndValidateVirtualMachine()
		})

		DescribeTable("should infer defaults from DataVolumeTemplates", func(generateDataVolumeTemplatesFunc func() []v1.DataVolumeTemplateSpec) {
			vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFunc()
			vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()
			createAndValidateVirtualMachine()
		},
			Entry("and DataVolumeSourcePVC",
				func() []v1.DataVolumeTemplateSpec {
					dv := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithPVCSource(sourceDV.Namespace, sourceDV.Name),
						libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
					)
					return []v1.DataVolumeTemplateSpec{{
						ObjectMeta: metav1.ObjectMeta{
							Name: dataVolumeTemplateName,
						},
						Spec: dv.Spec,
					}}
				},
			),
			Entry(", DataVolumeSourceRef and DataSource",
				func() []v1.DataVolumeTemplateSpec {
					By("Creating a DataSource")
					// TODO - Replace with libds?
					dataSource := &cdiv1beta1.DataSource{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "datasource-",
							Namespace:    namespace,
						},
						Spec: cdiv1beta1.DataSourceSpec{
							Source: cdiv1beta1.DataSourceSource{
								PVC: &cdiv1beta1.DataVolumeSourcePVC{
									Name:      sourceDV.Name,
									Namespace: namespace,
								},
							},
						},
					}
					dataSource, err := virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					dataVolume := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
					)

					// TODO - Add WithDataVolumeSourceRef support to libdv and use here
					dataVolume.Spec.SourceRef = &cdiv1beta1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Namespace: &namespace,
						Name:      dataSource.Name,
					}

					return generateDataVolumeTemplatesFromDataVolume(dataVolume)
				},
			),
			Entry(", DataVolumeSourceRef and DataSource with labels",
				func() []v1.DataVolumeTemplateSpec {
					By("Createing a blank DV and PVC without labels")
					blankDV := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithBlankImageSource(),
						libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
					)
					blankDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), blankDV, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					libstorage.EventuallyDV(sourceDV, 180, HaveSucceeded())

					By("Creating a DataSource")
					// TODO - Replace with libds?
					dataSource := &cdiv1beta1.DataSource{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "datasource-",
							Namespace:    namespace,
							Labels: map[string]string{
								instancetypeapi.DefaultInstancetypeLabel:     instancetype.Name,
								instancetypeapi.DefaultInstancetypeKindLabel: instancetypeapi.SingularResourceName,
								instancetypeapi.DefaultPreferenceLabel:       preference.Name,
								instancetypeapi.DefaultPreferenceKindLabel:   instancetypeapi.SingularPreferenceResourceName,
							},
						},
						Spec: cdiv1beta1.DataSourceSpec{
							Source: cdiv1beta1.DataSourceSource{
								PVC: &cdiv1beta1.DataVolumeSourcePVC{
									Name:      blankDV.Name,
									Namespace: namespace,
								},
							},
						},
					}
					dataSource, err := virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					dataVolume := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
					)

					// TODO - Add WithDataVolumeSourceRef support to libdv and use here
					dataVolume.Spec.SourceRef = &cdiv1beta1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Namespace: &namespace,
						Name:      dataSource.Name,
					}

					return generateDataVolumeTemplatesFromDataVolume(dataVolume)
				},
			),
		)

		It("should ignore failure when trying to infer defaults from DataVolumeSpec with unsupported DataVolumeSource when policy is set", func() {
			guestMemory := resource.MustParse("512Mi")
			vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
				Guest: &guestMemory,
			}

			// Inference from blank image source is not supported
			dv := libdv.NewDataVolume(
				libdv.WithNamespace(namespace),
				libdv.WithForceBindAnnotation(),
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce), libdv.PVCWithVolumeSize("1Gi")),
			)
			vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFromDataVolume(dv)
			vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()

			failurePolicy := v1.IgnoreInferFromVolumeFailure
			vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
			vm.Spec.Preference.InferFromVolumeFailurePolicy = &failurePolicy

			By("Creating the VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Validating the VirtualMachine")
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(guestMemory))
		})

		DescribeTable("should reject VM creation when inference was successful but memory and RejectInferFromVolumeFailure were set", func(explicit bool) {
			guestMemory := resource.MustParse("512Mi")
			vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
				Guest: &guestMemory,
			}

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: sourceDV.Name,
						},
					},
				},
			}}

			if explicit {
				failurePolicy := v1.RejectInferFromVolumeFailure
				vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
			}

			By("Creating the VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
			Expect(err).To(MatchError("admission webhook \"virtualmachine-validator.kubevirt.io\" denied the request: VM field spec.template.spec.domain.memory conflicts with selected instance type"))
		},
			Entry("with explicitly setting RejectInferFromVolumeFailure", true),
			Entry("with implicitly setting RejectInferFromVolumeFailure (default)", false),
		)
	})

	Context("instance type with dedicatedCPUPlacement enabled", func() {

		BeforeEach(func() {
			checks.SkipTestIfNoCPUManager()
		})

		It("should be accepted and result in running VirtualMachineInstance", func() {
			vmi := libvmi.NewCirros()

			clusterInstancetype := newVirtualMachineClusterInstancetype(vmi)
			clusterInstancetype.Spec.CPU.DedicatedCPUPlacement = pointer.Bool(true)
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			removeResourcesAndPreferencesFromVMI(vmi)
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: clusterInstancetype.Name,
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVirtualMachine(vm)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that DedicatedCPUPlacement is not set in the VM but enabled in the VMI through the instance type
			Expect(vm.Spec.Template.Spec.Domain.CPU).To(BeNil())
			Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(BeTrue())
		})
	})

	Context("instancetype.kubevirt.io apiVersion compatibility", func() {
		fetchVirtualMachineInstancetypev1alpha1 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineInstancetypes(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineInstancetypev1alpha2 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineInstancetypes(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineInstancetypev1beta1 := func(objName string) {
			_, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterInstancetypev1alpha1 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterInstancetypes().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterInstancetypev1alpha2 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterInstancetypes().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterInstancetypev1beta1 := func(objName string) {
			_, err := virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachinePreferencev1alpha1 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachinePreferences(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachinePreferencev1alpha2 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachinePreferences(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachinePreferencev1beta1 := func(objName string) {
			_, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterPreferencev1alpha1 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterPreferences().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterPreferencev1alpha2 := func(objName string) {
			_, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterPreferences().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		fetchVirtualMachineClusterPreferencev1beta1 := func(objName string) {
			_, err := virtClient.VirtualMachineClusterPreference().Get(context.Background(), objName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		DescribeTable("should create", func(createFunc func() string, v1alpha1FetchFunc func(string), v1alpha2FetchFunc func(string), v1beta1FetchFunc func(string)) {
			// Create the object and then fetch it using the currently supported versions
			objName := createFunc()
			v1alpha1FetchFunc(objName)
			v1alpha2FetchFunc(objName)
			v1beta1FetchFunc(objName)
		},
			Entry("VirtualMachineInstancetype v1alpha1 and fetch using v1alpha1, v1alpha2 and v1beta1",
				func() string {
					createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineInstancetypes(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1alpha1.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "instancetype",
						},
						Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: 1,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: resource.MustParse("128Mi"),
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createdObj.Name
				},
				fetchVirtualMachineInstancetypev1alpha1,
				fetchVirtualMachineInstancetypev1alpha2,
				fetchVirtualMachineInstancetypev1beta1,
			),
			Entry("VirtualMachineInstancetype v1alpha2 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineInstancetypes(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1alpha2.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype",
					},
					Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha2.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1alpha2.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineInstancetypev1alpha1,
				fetchVirtualMachineInstancetypev1alpha2,
				fetchVirtualMachineInstancetypev1beta1,
			),
			Entry("VirtualMachineInstancetype v1beta1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1beta1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype",
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineInstancetypev1alpha1,
				fetchVirtualMachineInstancetypev1alpha2,
				fetchVirtualMachineInstancetypev1beta1,
			),
			Entry("VirtualMachineClusterInstancetype v1alpha1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterInstancetypes().Create(context.Background(), &instancetypev1alpha1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype",
					},
					Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha1.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1alpha1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterInstancetypev1alpha1,
				fetchVirtualMachineClusterInstancetypev1alpha2,
				fetchVirtualMachineClusterInstancetypev1beta1,
			),
			Entry("VirtualMachineClusterInstancetype v1alpha2 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterInstancetypes().Create(context.Background(), &instancetypev1alpha2.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype",
					},
					Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha2.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1alpha2.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterInstancetypev1alpha1,
				fetchVirtualMachineClusterInstancetypev1alpha2,
				fetchVirtualMachineClusterInstancetypev1beta1,
			),
			Entry("VirtualMachineClusterInstancetype v1beta1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), &instancetypev1beta1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype",
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterInstancetypev1alpha1,
				fetchVirtualMachineClusterInstancetypev1alpha2,
				fetchVirtualMachineClusterInstancetypev1beta1,
			),
			Entry("VirtualMachinePreference v1alpha1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachinePreferences(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1alpha1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha1.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha1.PreferCores,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachinePreferencev1alpha1,
				fetchVirtualMachinePreferencev1alpha2,
				fetchVirtualMachinePreferencev1beta1,
			),
			Entry("VirtualMachinePreference v1alpha2 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachinePreferences(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1alpha2.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha2.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha2.PreferCores,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachinePreferencev1alpha1,
				fetchVirtualMachinePreferencev1alpha2,
				fetchVirtualMachinePreferencev1beta1,
			),
			Entry("VirtualMachinePreference v1beta1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				preferredCPUTopology := instancetypev1beta1.PreferCores
				createdObj, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).Create(context.Background(), &instancetypev1beta1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
							PreferredCPUTopology: &preferredCPUTopology,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachinePreferencev1alpha1,
				fetchVirtualMachinePreferencev1alpha2,
				fetchVirtualMachinePreferencev1beta1,
			),
			Entry("VirtualMachineClusterPreference v1alpha1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterPreferences().Create(context.Background(), &instancetypev1alpha1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha1.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha1.PreferCores,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterPreferencev1alpha1,
				fetchVirtualMachineClusterPreferencev1alpha2,
				fetchVirtualMachineClusterPreferencev1beta1,
			),
			Entry("VirtualMachineClusterPreference v1alpha2 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				createdObj, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterPreferences().Create(context.Background(), &instancetypev1alpha2.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha2.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha2.PreferCores,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterPreferencev1alpha1,
				fetchVirtualMachineClusterPreferencev1alpha2,
				fetchVirtualMachineClusterPreferencev1beta1,
			),
			Entry("VirtualMachineClusterPreference v1beta1 and fetch using v1alpha1, v1alpha2 and v1beta1", func() string {
				preferredCPUTopology := instancetypev1beta1.PreferCores
				createdObj, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), &instancetypev1beta1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "preference",
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
							PreferredCPUTopology: &preferredCPUTopology,
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createdObj.Name
			},
				fetchVirtualMachineClusterPreferencev1alpha1,
				fetchVirtualMachineClusterPreferencev1alpha2,
				fetchVirtualMachineClusterPreferencev1beta1,
			),
		)
	})

	Context("VirtualMachine using preference resource requirements", func() {
		var (
			preferCores   = instancetypev1beta1.PreferCores
			preferThreads = instancetypev1beta1.PreferThreads
			preferAny     = instancetypev1beta1.PreferAny
		)

		DescribeTable("should be accepted when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *v1.VirtualMachine) {
			if instancetype != nil {
				instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Instancetype.Name = instancetype.Name
			}

			preference, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference.Name = preference.Name
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("VirtualMachineInstancetype meets CPU requirements",
				&instancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype-",
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
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
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Instancetype: &v1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{},
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
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Instancetype: &v1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{},
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
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
										Cores:   uint32(1),
										Threads: uint32(1),
										Sockets: uint32(2),
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
							PreferredCPUTopology: &preferCores,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
										Cores:   uint32(2),
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
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{},
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
							PreferredCPUTopology: &preferThreads,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
										Cores:   uint32(1),
										Threads: uint32(2),
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
							PreferredCPUTopology: &preferAny,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(4),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
										Cores:   uint32(2),
										Threads: uint32(1),
										Sockets: uint32(2),
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
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									Memory: &v1.Memory{
										Guest: resource.NewQuantity(2*1024*1024*1024, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			),
		)

		DescribeTable("should be rejected when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *v1.VirtualMachine, errorSubString string) {
			if instancetype != nil {
				instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Instancetype.Name = instancetype.Name
			}

			preference, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference.Name = preference.Name
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
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
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Instancetype: &v1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{},
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
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Instancetype: &v1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{},
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
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
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
							PreferredCPUTopology: &preferCores,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
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
							PreferredCPUTopology: &preferThreads,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(2),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
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
							PreferredCPUTopology: &preferAny,
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(4),
							},
						},
					},
				},
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									CPU: &v1.CPU{
										Cores:   uint32(2),
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
				&v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: v1.VirtualMachineSpec{
						Running: pointer.Bool(false),
						Preference: &v1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Domain: v1.DomainSpec{
									Memory: &v1.Memory{
										Guest: resource.NewQuantity(1*1024*1024*1024, resource.BinarySI),
									},
								},
							},
						},
					},
				},
				"insufficient Memory resources",
			),
		)
	})
})

func newVirtualMachineInstancetype(vmi *v1.VirtualMachineInstance) *instancetypev1beta1.VirtualMachineInstancetype {
	return &instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
		Spec: newVirtualMachineInstancetypeSpec(vmi),
	}
}

func newVirtualMachineClusterInstancetype(vmi *v1.VirtualMachineInstance) *instancetypev1beta1.VirtualMachineClusterInstancetype {
	return &instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-instancetype-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
		Spec: newVirtualMachineInstancetypeSpec(vmi),
	}
}

func newVirtualMachineInstancetypeSpec(vmi *v1.VirtualMachineInstance) instancetypev1beta1.VirtualMachineInstancetypeSpec {
	// Copy the amount of memory set within the VMI so our tests don't randomly start using more resources
	guestMemory := resource.MustParse("128M")
	if vmi != nil {
		if _, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
			guestMemory = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory].DeepCopy()
		}
	}
	return instancetypev1beta1.VirtualMachineInstancetypeSpec{
		CPU: instancetypev1beta1.CPUInstancetype{
			Guest: uint32(1),
		},
		Memory: instancetypev1beta1.MemoryInstancetype{
			Guest: guestMemory,
		},
	}
}

func newVirtualMachinePreference() *instancetypev1beta1.VirtualMachinePreference {
	return &instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
		},
	}
}

func newVirtualMachineClusterPreference() *instancetypev1beta1.VirtualMachineClusterPreference {
	return &instancetypev1beta1.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-cluster-preference-",
			Namespace:    testsuite.GetTestNamespace(nil),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		},
	}
}

func removeResourcesAndPreferencesFromVMI(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Domain.CPU = nil
	vmi.Spec.Domain.Memory = nil
	vmi.Spec.TerminationGracePeriodSeconds = nil
	vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
	vmi.Spec.Domain.Features = &v1.Features{}
	vmi.Spec.Domain.Machine = &v1.Machine{}

	for diskIndex := range vmi.Spec.Domain.Devices.Disks {
		if vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk != nil && vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk.Bus != "" {
			vmi.Spec.Domain.Devices.Disks[diskIndex].DiskDevice.Disk.Bus = ""
		}
	}
}
