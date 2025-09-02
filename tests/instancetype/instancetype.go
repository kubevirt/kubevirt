//nolint:dupl,lll,gomnd
package instancetype

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

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const timeout = 300 * time.Second

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences", decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Instancetype validation", func() {
		It("[test_id:CNV-9082] should allow valid instancetype", func() {
			instancetype := builder.NewInstancetypeFromVMI(nil)
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
			preference := builder.NewPreference()
			_, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("A VirtualMachine", func() {
		const (
			resourceName               = "nonexistent"
			failureCondition           = "Failure"
			instancetypeNotFoundReason = "FailedFindInstancetype"
			preferenceNotFoundReason   = "FailedFindPreference"
		)
		DescribeTable("referencing a non-existent", func(vmOptionFunc func(*virtv1.VirtualMachine), expectedReason string, createResourceFunc func(string) error) {
			By("Creating the VM and asserting that it doesn't start and has the expected condition in place")
			vm := libvmi.NewVirtualMachine(
				libvmifact.NewGuestless(),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
				vmOptionFunc,
			)
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), time.Minute, time.Second).Should(matcher.HaveConditionTrueWithReason(failureCondition, expectedReason))
			Eventually(matcher.ThisVM(vm), time.Minute, time.Second).Should(matcher.HavePrintableStatus(virtv1.VirtualMachineStatusStopped))
			By("Creating the missing resource and asserting that the condition is removed and the VM starts")
			Expect(createResourceFunc(vm.Namespace)).To(Succeed())
			Eventually(matcher.ThisVM(vm), time.Minute, time.Second).ShouldNot(matcher.HaveConditionTrueWithReason(failureCondition, expectedReason))
			Eventually(matcher.ThisVM(vm), time.Minute, time.Second).Should(matcher.BeReady())
		},
			Entry("[test_id:CNV-9086] cluster instance type should still be created and eventually start when missing resource created",
				libvmi.WithClusterInstancetype(resourceName),
				instancetypeNotFoundReason,
				func(_ string) error {
					instancetype := builder.NewClusterInstancetype(
						builder.WithCPUs(1),
						builder.WithMemory("128Mi"),
					)
					// FIXME(lyarwood): builder should provide WithName()
					instancetype.GenerateName = ""
					instancetype.Name = resourceName
					_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
					return err
				},
			),
			Entry("[test_id:CNV-9089] instance type should still be created and eventually start when missing resource created",
				libvmi.WithInstancetype(resourceName),
				instancetypeNotFoundReason,
				func(namespace string) error {
					instancetype := builder.NewInstancetype(
						builder.WithCPUs(1),
						builder.WithMemory("128Mi"),
					)
					// FIXME(lyarwood): builder should provide WithName()
					instancetype.GenerateName = ""
					instancetype.Name = resourceName
					_, err := virtClient.VirtualMachineInstancetype(namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
					return err
				},
			),
			Entry("[test_id:CNV-9091] cluster preference should still be created and eventually start when missing resource created",
				libvmi.WithClusterPreference(resourceName),
				preferenceNotFoundReason,
				func(_ string) error {
					preference := builder.NewClusterPreference(
						builder.WithPreferredCPUTopology(instancetypev1beta1.Cores),
					)
					// FIXME(lyarwood): builder should provide WithName()
					preference.GenerateName = ""
					preference.Name = resourceName
					_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
					return err
				},
			),
			Entry("[test_id:CNV-9090] preference should still be created and eventually start when missing resource created",
				libvmi.WithPreference(resourceName),
				preferenceNotFoundReason,
				func(namespace string) error {
					preference := builder.NewPreference(
						builder.WithPreferredCPUTopology(instancetypev1beta1.Cores),
					)
					// FIXME(lyarwood): builder should provide WithName()
					preference.GenerateName = ""
					preference.Name = resourceName
					_, err := virtClient.VirtualMachinePreference(namespace).Create(context.Background(), preference, metav1.CreateOptions{})
					return err
				},
			),
		)
	})

	Context("with cluster memory overcommit being applied", Serial, func() {
		BeforeEach(func() {
			kv := libkubevirt.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			config.DeveloperConfiguration.MemoryOvercommit = 200
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		})
		It("should apply memory overcommit instancetype to VMI even with cluster overcommit set", func() {
			// Use an Alpine VMI so we have enough memory in the eventual instance type and launched VMI to get past validation checks
			vmi := libvmifact.NewAlpine()

			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype.Spec.Memory.OvercommitPercent = 15

			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove any requested resources from the VMI before generating the VM
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))
			expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
			Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
			memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
			Expect(memRequest.Value()).To(Equal(expectedOverhead))
		})
	})

	Context("Instancetype and preference application", func() {
		var vmi *virtv1.VirtualMachineInstance
		const (
			preferredTerminationGracePeriodSeconds = 15
			expectedCausesLength                   = 3
		)
		BeforeEach(func() {
			vmi = libvmifact.NewGuestless()
		})

		It("[test_id:CNV-9094] should find and apply cluster instancetype and preferences when kind isn't provided", func() {
			clusterInstancetype := builder.NewClusterInstancetypeFromVMI(vmi)
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			clusterPreference := builder.NewClusterPreference()
			clusterPreference.Spec.CPU = &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
			}

			clusterPreference, err = virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithClusterInstancetype(clusterInstancetype.Name),
				libvmi.WithClusterPreference(clusterPreference.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:CNV-9095] should apply instancetype and preferences to VMI", func() {
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype.Spec.Annotations = map[string]string{
				"required-annotation-1": "1",
				"required-annotation-2": "2",
			}
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := builder.NewPreference()
			preference.Spec.CPU = &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
			}
			preference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredDiskBus: virtv1.DiskBusSATA,
			}
			preference.Spec.Features = &instancetypev1beta1.FeaturePreferences{
				PreferredHyperv: &virtv1.FeatureHyperv{
					VAPIC: &virtv1.FeatureState{
						Enabled: pointer.P(true),
					},
					Relaxed: &virtv1.FeatureState{
						Enabled: pointer.P(true),
					},
				},
			}
			preference.Spec.Firmware = &instancetypev1beta1.FirmwarePreferences{
				PreferredUseBios: pointer.P(true),
			}
			preference.Spec.PreferredTerminationGracePeriodSeconds = pointer.P(int64(preferredTerminationGracePeriodSeconds))
			preference.Spec.PreferredSubdomain = pointer.P("non-existent-subdomain")
			preference.Spec.Annotations = map[string]string{
				"preferred-annotation-1": "1",
				"preferred-annotation-2": "use-vm-value",
				"required-annotation-1":  "use-instancetype-value",
			}

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{
				"preferred-annotation-2": "2",
			}
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert we've used sockets as instancetypev1beta1.Sockets was requested
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
			Expect(vmi.Annotations[virtv1.InstancetypeAnnotation]).To(Equal(instancetype.Name))
			Expect(vmi.Annotations[virtv1.ClusterInstancetypeAnnotation]).To(Equal(""))
			Expect(vmi.Annotations[virtv1.PreferenceAnnotation]).To(Equal(preference.Name))
			Expect(vmi.Annotations[virtv1.ClusterPreferenceAnnotation]).To(Equal(""))
			Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-1", "1"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("required-annotation-2", "2"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-1", "1"))
			Expect(vmi.Annotations).To(HaveKeyWithValue("preferred-annotation-2", "2"))
		})
		It("should apply memory overcommit instancetype to VMI", func() {
			// Use an Alpine VMI so we have enough memory in the eventual instance type and launched VMI to get past validation checks
			vmi = libvmifact.NewAlpine()
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype.Spec.Memory.OvercommitPercent = 15

			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetype.Spec.Memory.Guest))

			expectedOverhead := int64(float32(instancetype.Spec.Memory.Guest.Value()) * (1 - float32(instancetype.Spec.Memory.OvercommitPercent)/100))
			Expect(expectedOverhead).ToNot(Equal(instancetype.Spec.Memory.Guest.Value()))
			memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
			Expect(memRequest.Value()).To(Equal(expectedOverhead))
		})

		It("[test_id:CNV-9096] should fail if instancetype and VM define CPU", func() {
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithInstancetype(instancetype.Name))
			vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{Sockets: 1, Cores: 1, Threads: 1}
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())

			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")
			Expect(apiStatus.Status().Details.Causes).To(HaveLen(expectedCausesLength))

			baseCPUConflict := conflict.New("spec", "template", "spec", "domain", "cpu")

			cause0 := apiStatus.Status().Details.Causes[0]
			Expect(cause0.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			socketsConflict := baseCPUConflict.NewChild("sockets")
			Expect(cause0.Message).To(Equal(socketsConflict.Error()))
			Expect(cause0.Field).To(Equal(socketsConflict.String()))

			cause1 := apiStatus.Status().Details.Causes[1]
			Expect(cause1.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			coresConflict := baseCPUConflict.NewChild("cores")
			Expect(cause1.Message).To(Equal(coresConflict.Error()))
			Expect(cause1.Field).To(Equal(coresConflict.String()))

			cause2 := apiStatus.Status().Details.Causes[2]
			Expect(cause2.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			threadsConflict := baseCPUConflict.NewChild("threads")
			Expect(cause2.Message).To(Equal(threadsConflict.Error()))
			Expect(cause2.Field).To(Equal(threadsConflict.String()))
		})

		DescribeTable("[test_id:CNV-9301] should fail if the VirtualMachine has ", func(resources virtv1.ResourceRequirements, expectedConflict *conflict.Conflict) {
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithInstancetype(instancetype.Name))
			vm.Spec.Template.Spec.Domain.Resources = resources
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())

			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")
			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(cause.Message).To(Equal(expectedConflict.Error()))
			Expect(cause.Field).To(Equal(expectedConflict.String()))
		},
			Entry("CPU resource requests", virtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU: resource.MustParse("1"),
				},
			}, conflict.New("spec.template.spec.domain.resources.requests.cpu")),
			Entry("CPU resource limits", virtv1.ResourceRequirements{
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU: resource.MustParse("1"),
				},
			}, conflict.New("spec.template.spec.domain.resources.limits.cpu")),
			Entry("Memory resource requests", virtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}, conflict.New("spec.template.spec.domain.resources.requests.memory")),
			Entry("Memory resource limits", virtv1.ResourceRequirements{
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}, conflict.New("spec.template.spec.domain.resources.limits.memory")),
		)

		It("[test_id:CNV-9302] should apply preferences to default network interface", func() {
			clusterPreference := builder.NewClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredInterfaceModel: virtv1.VirtIO,
			}

			clusterPreference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithClusterPreference(clusterPreference.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(clusterPreference.Spec.Devices.PreferredInterfaceModel))
		})

		It("[test_id:CNV-9303] should apply preferences to default volume disks", func() {
			clusterPreference := builder.NewClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredDiskBus: virtv1.DiskBusVirtio,
			}

			clusterPreference, err := virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithClusterPreference(clusterPreference.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm.Spec.Template.Spec.Domain.Devices.Disks = []virtv1.Disk{}
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				Expect(disk.DiskDevice.Disk.Bus).To(Equal(clusterPreference.Spec.Devices.PreferredDiskBus))
			}
		})

		It("[test_id:CNV-9098] should store and use ControllerRevisions of VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec", func() {
			By("Creating a VirtualMachineInstancetype")
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			originalInstancetypeCPUGuest := instancetype.Spec.CPU.Guest
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference := builder.NewPreference()
			preference.Spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
				},
			}
			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
			)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec ControllerRevision to be referenced from the VirtualMachine")
			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype := &instancetypev1beta1.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(instancetype.Spec))

			preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedPreference := &instancetypev1beta1.VirtualMachinePreference{}
			Expect(json.Unmarshal(preferenceRevision.Data.Raw, stashedPreference)).To(Succeed())
			Expect(stashedPreference.Spec).To(Equal(preference.Spec))

			vm = libvmops.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalInstancetypeCPUGuest))

			By("Updating the VirtualMachineInstancetype vCPU count")
			newInstancetypeCPUGuest := originalInstancetypeCPUGuest + 1
			patchData, err := patch.GenerateTestReplacePatch("/spec/cpu/guest", originalInstancetypeCPUGuest, newInstancetypeCPUGuest)
			Expect(err).ToNot(HaveOccurred())
			updatedInstancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Patch(context.Background(), instancetype.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedInstancetype.Spec.CPU.Guest).To(Equal(newInstancetypeCPUGuest))

			vm = libvmops.StopVirtualMachine(vm)
			vm = libvmops.StartVirtualMachine(vm)

			By("Checking that a VirtualMachineInstance has been created with the original VirtualMachineInstancetype and VirtualMachinePreference applied")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(originalInstancetypeCPUGuest))

			By("Creating a second VirtualMachine using the now updated VirtualMachineInstancetype and original VirtualMachinePreference")
			newVMI := libvmifact.NewGuestless()
			newVM := libvmi.NewVirtualMachine(newVMI,
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
			)
			newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), newVM, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for a ControllerRevisions to be referenced from the new VirtualMachine")
			Eventually(matcher.ThisVM(newVM)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

			newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), newVM.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the two VirtualMachines are using different ControllerRevisions of the same VirtualMachineInstancetype")
			Expect(newVM.Spec.Instancetype.Name).To(Equal(vm.Spec.Instancetype.Name))
			Expect(newVM.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(Equal(vm.Status.InstancetypeRef.ControllerRevisionRef.Name))

			By("Checking that new ControllerRevisions for the updated VirtualMachineInstancetype")
			instancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), newVM.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			stashedInstancetype = &instancetypev1beta1.VirtualMachineInstancetype{}
			Expect(json.Unmarshal(instancetypeRevision.Data.Raw, stashedInstancetype)).To(Succeed())
			Expect(stashedInstancetype.Spec).To(Equal(updatedInstancetype.Spec))

			newVM = libvmops.StartVirtualMachine(newVM)

			By("Checking that the new VirtualMachineInstance is using the updated VirtualMachineInstancetype")
			newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), newVM.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI.Spec.Domain.CPU.Sockets).To(Equal(newInstancetypeCPUGuest))
		})

		It("[test_id:CNV-9304] should fail if stored ControllerRevisions are different", func() {
			By("Creating a VirtualMachineInstancetype")
			instancetype := builder.NewInstancetypeFromVMI(vmi)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachine")
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithInstancetype(instancetype.Name), libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for ControllerRevisions to be referenced from the VirtualMachine")
			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.HaveInstancetypeControllerRevisionRef())

			By("Checking that ControllerRevisions have been created for the VirtualMachineInstancetype and VirtualMachinePreference")
			instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Stopping and removing VM")
			vm = libvmops.StopVirtualMachine(vm)

			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait until ControllerRevision is deleted
			Eventually(func(g Gomega) metav1.StatusReason {
				_, err = virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(instancetype)).Get(context.Background(), instancetypeRevision.Name, metav1.GetOptions{})
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

			_, err = virtClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), newInstancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating and starting the VM and expecting a failure")
			newVM := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(virtv1.RunStrategyAlways), libvmi.WithInstancetype(instancetype.Name))
			newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), newVM, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				foundVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), newVM.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				cond := controller.NewVirtualMachineConditionManager().
					GetCondition(foundVM, virtv1.VirtualMachineFailure)
				g.Expect(cond).ToNot(BeNil())
				g.Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				g.Expect(cond.Message).To(ContainSubstring("found existing ControllerRevision with unexpected data"))
			}, 5*time.Minute, time.Second).Should(Succeed())
		})
	})

	Context("with inferFromVolume", func() {
		var (
			vm           *virtv1.VirtualMachine
			instancetype *instancetypev1beta1.VirtualMachineInstancetype
			preference   *instancetypev1beta1.VirtualMachinePreference
			sourceDV     *cdiv1beta1.DataVolume
			namespace    string
		)

		const (
			inferFromVolumeName     = "volume"
			dataVolumeTemplateName  = "datatemplate"
			dvSuccessTimeoutSeconds = 180
		)

		createAndValidateVirtualMachine := func() {
			By("Creating the VirtualMachine")
			var err error
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways)(vm)
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
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

			Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

			By("Validating the VirtualMachineInstance")
			var vmi *virtv1.VirtualMachineInstance
			vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetype.Spec.CPU.Guest))
		}

		generateDataVolumeTemplatesFromDataVolume := func(dataVolume *cdiv1beta1.DataVolume) []virtv1.DataVolumeTemplateSpec {
			return []virtv1.DataVolumeTemplateSpec{{
				ObjectMeta: metav1.ObjectMeta{
					Name: dataVolumeTemplateName,
				},
				Spec: dataVolume.Spec,
			}}
		}

		generateVolumesForDataVolumeTemplates := func() []virtv1.Volume {
			return []virtv1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: dataVolumeTemplateName,
					},
				},
			}}
		}

		BeforeEach(func() {
			if !libstorage.HasCDI() {
				Fail("instance type and preference inferFromVolume tests require CDI to be installed providing the DataVolume and DataSource CRDs")
			}

			namespace = testsuite.GetTestNamespace(nil)

			By("Creating a VirtualMachineInstancetype")
			instancetype = builder.NewInstancetypeFromVMI(nil)
			var err error
			instancetype, err = virtClient.VirtualMachineInstancetype(namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VirtualMachinePreference")
			preference = builder.NewPreference()
			preference.Spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
				},
			}
			preference, err = virtClient.VirtualMachinePreference(namespace).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Creating source DataVolume and PVC")
			sourceDV = libdv.NewDataVolume(
				libdv.WithNamespace(namespace),
				libdv.WithForceBindAnnotation(),
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
				libdv.WithDefaultInstancetype(instancetypeapi.SingularResourceName, instancetype.Name),
				libdv.WithDefaultPreference(instancetypeapi.SingularPreferenceResourceName, preference.Name),
			)

			sourceDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, dvSuccessTimeoutSeconds, matcher.HaveSucceeded())

			// This is the default but it should still be cleared
			failurePolicy := virtv1.RejectInferFromVolumeFailure
			runStrategy := virtv1.RunStrategyHalted

			vm = &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vm-",
					Namespace:    namespace,
				},
				Spec: virtv1.VirtualMachineSpec{
					Instancetype: &virtv1.InstancetypeMatcher{
						InferFromVolume:              inferFromVolumeName,
						InferFromVolumeFailurePolicy: &failurePolicy,
					},
					Preference: &virtv1.PreferenceMatcher{
						InferFromVolume:              inferFromVolumeName,
						InferFromVolumeFailurePolicy: &failurePolicy,
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{},
						},
					},
					RunStrategy: &runStrategy,
				},
			}
		})

		It("should infer defaults from PersistentVolumeClaimVolumeSource", func() {
			vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: sourceDV.Name,
						},
					},
				},
			}}
			createAndValidateVirtualMachine()
		})

		It("should infer defaults from existing DataVolume with labels", func() {
			vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: sourceDV.Name,
					},
				},
			}}
			createAndValidateVirtualMachine()
		})

		DescribeTable("should infer defaults from DataVolumeTemplates", func(generateDataVolumeTemplatesFunc func() []virtv1.DataVolumeTemplateSpec) {
			vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFunc()
			vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()
			createAndValidateVirtualMachine()
		},
			Entry("and DataVolumeSourcePVC",
				func() []virtv1.DataVolumeTemplateSpec {
					dv := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithPVCSource(sourceDV.Namespace, sourceDV.Name),
						libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
					)
					return []virtv1.DataVolumeTemplateSpec{{
						ObjectMeta: metav1.ObjectMeta{
							Name: dataVolumeTemplateName,
						},
						Spec: dv.Spec,
					}}
				},
			),
			Entry(", DataVolumeSourceRef and DataSource",
				func() []virtv1.DataVolumeTemplateSpec {
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
						libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
						libdv.WithDataVolumeSourceRef("DataSource", namespace, dataSource.Name),
					)

					return generateDataVolumeTemplatesFromDataVolume(dataVolume)
				},
			),
			Entry(", DataVolumeSourceRef and DataSource with labels",
				func() []virtv1.DataVolumeTemplateSpec {
					By("Creating a blank DV and PVC without labels")
					blankDV := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithBlankImageSource(),
						libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
					)
					blankDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), blankDV, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					libstorage.EventuallyDV(sourceDV, dvSuccessTimeoutSeconds, matcher.HaveSucceeded())

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
					dataSource, err = virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					dataVolume := libdv.NewDataVolume(
						libdv.WithNamespace(namespace),
						libdv.WithForceBindAnnotation(),
						libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
						libdv.WithDataVolumeSourceRef("DataSource", namespace, dataSource.Name),
					)

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
				libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
			)
			vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFromDataVolume(dv)
			vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()

			failurePolicy := virtv1.IgnoreInferFromVolumeFailure
			vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
			vm.Spec.Preference.InferFromVolumeFailurePolicy = &failurePolicy

			By("Creating the VirtualMachine")
			var err error
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
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

			vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
				Name: inferFromVolumeName,
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: sourceDV.Name,
						},
					},
				},
			}}

			if explicit {
				failurePolicy := virtv1.RejectInferFromVolumeFailure
				vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
			}

			By("Creating the VirtualMachine")
			_, err := virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).To(MatchError("admission webhook \"virtualmachine-validator.kubevirt.io\" denied the request: VM field(s) spec.template.spec.domain.memory.guest conflicts with selected instance type"))
		},
			Entry("with explicitly setting RejectInferFromVolumeFailure", true),
			Entry("with implicitly setting RejectInferFromVolumeFailure (default)", false),
		)
	})

	Context("instance type with dedicatedCPUPlacement enabled", decorators.RequiresNodeWithCPUManager, func() {
		It("should be accepted and result in running VirtualMachineInstance", func() {
			vmi := libvmifact.NewGuestless()

			clusterInstancetype := builder.NewClusterInstancetypeFromVMI(vmi)
			clusterInstancetype.Spec.CPU.DedicatedCPUPlacement = pointer.P(true)
			clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithClusterInstancetype(clusterInstancetype.Name), libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that DedicatedCPUPlacement is not set in the VM but enabled in the VMI through the instance type
			Expect(vm.Spec.Template.Spec.Domain.CPU).To(BeNil())
			Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(BeTrue())
		})
	})

	Context("VirtualMachine using preference resource requirements", func() {
		var (
			guestMemory2GB = resource.MustParse("2Gi")
			guestMemory1GB = resource.MustParse("1Gi")
		)
		const (
			providedvCPUs    = 2
			requiredNumvCPUs = 2
			requiredvCPUs    = 4
			providedCores    = 2
			providedSockets  = 2
			providedThreads  = 2
		)

		DescribeTable("should be accepted when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *virtv1.VirtualMachine) {
			var err error
			if instancetype != nil {
				instancetype, err = virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Instancetype.Name = instancetype.Name
			}

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference.Name = preference.Name
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("VirtualMachineInstancetype meets CPU requirements",
				&instancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "instancetype-",
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(providedvCPUs),
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
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Instancetype: &virtv1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Instancetype: &virtv1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
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
								Guest: uint32(requiredNumvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
										Cores:   uint32(1),
										Threads: uint32(1),
										Sockets: uint32(providedSockets),
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
										Cores:   uint32(providedCores),
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Threads),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
										Cores:   uint32(1),
										Threads: uint32(providedThreads),
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Any),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(requiredvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
										Cores:   uint32(providedCores),
										Threads: uint32(1),
										Sockets: uint32(providedSockets),
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									Memory: &virtv1.Memory{
										Guest: &guestMemory2GB,
									},
								},
							},
						},
					},
				},
			),
		)

		DescribeTable("should be rejected when", func(instancetype *instancetypev1beta1.VirtualMachineInstancetype, preference *instancetypev1beta1.VirtualMachinePreference, vm *virtv1.VirtualMachine, errorSubString string) {
			var err error
			if instancetype != nil {
				instancetype, err = virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Instancetype.Name = instancetype.Name
			}

			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference.Name = preference.Name
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
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
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Instancetype: &virtv1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Instancetype: &virtv1.InstancetypeMatcher{
							Kind: "VirtualMachineInstancetype",
						},
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
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
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Threads),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(providedvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
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
							PreferredCPUTopology: pointer.P(instancetypev1beta1.Any),
						},
						Requirements: &instancetypev1beta1.PreferenceRequirements{
							CPU: &instancetypev1beta1.CPUPreferenceRequirement{
								Guest: uint32(requiredvCPUs),
							},
						},
					},
				},
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									CPU: &virtv1.CPU{
										Cores:   uint32(providedCores),
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
			Entry("VirtualMachine does not meet Memory requirements",
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{
									Memory: &virtv1.Memory{
										Guest: &guestMemory1GB,
									},
								},
							},
						},
					},
				},
				"insufficient Memory resources",
			),
			Entry("VirtualMachine does not meet Memory requirements or provide any guest visible memory - bug #14551",
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
				&virtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-",
					},
					Spec: virtv1.VirtualMachineSpec{
						RunStrategy: pointer.P(virtv1.RunStrategyHalted),
						Preference: &virtv1.PreferenceMatcher{
							Kind: "VirtualMachinePreference",
						},
						Template: &virtv1.VirtualMachineInstanceTemplateSpec{
							Spec: virtv1.VirtualMachineInstanceSpec{
								Domain: virtv1.DomainSpec{},
							},
						},
					},
				},
				"insufficient Memory resources",
			),
		)
	})
})
