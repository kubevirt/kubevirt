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

})
