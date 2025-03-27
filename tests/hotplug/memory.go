package hotplug

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/migration"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Memory Hotplug", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateRolloutStrategy, Serial, func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv := libkubevirt.GetCurrentKv(virtClient)
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
		patchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

		currentKv := libkubevirt.GetCurrentKv(virtClient)
		config.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
	})

	Context("A VM with memory liveUpdate enabled", func() {
		createHotplugVM := func(sockets *uint32, maxSockets uint32, opts ...libvmi.Option) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
			vmiOpts := append([]libvmi.Option{},
				libnet.WithMasqueradeNetworking(),
				libvmi.WithResourceMemory("1Gi"),
			)
			vmiOpts = append(vmiOpts, opts...)
			vmi := libvmifact.NewAlpine(vmiOpts...)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.Memory = &v1.Memory{
				Guest: pointer.P(resource.MustParse("1Gi")),
			}

			if sockets != nil {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 1,
				}
			}

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			if maxSockets != 0 {
				vm.Spec.Template.Spec.Domain.CPU.MaxSockets = maxSockets
			}

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			EventuallyWithOffset(1, ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)
			return vm, vmi
		}

		parseCurrentDomainMemory := func(spec *api.DomainSpec) *resource.Quantity {
			ExpectWithOffset(1, spec.CurrentMemory).NotTo(BeNil())
			memory, err := resource.ParseQuantity(fmt.Sprintf("%vKi", spec.CurrentMemory.Value))
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &memory
		}

		DescribeTable("[test_id:10823]should successfully hotplug memory", func(opts ...libvmi.Option) {
			By("Creating a VM")
			guest := resource.MustParse("1Gi")
			vm, vmi := createHotplugVM(nil, 0, opts...)

			By("Limiting the bandwidth of migrations in the test namespace")
			migrationBandwidthLimit := resource.MustParse("1Ki")
			migration.CreateMigrationPolicy(virtClient, migration.PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

			By("Ensuring the compute container has the expected memory")
			compute, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqMemory := compute.Resources.Requests.Memory().Value()
			Expect(reqMemory).To(BeNumerically(">=", guest.Value()))

			By("Hotplug additional memory")
			newGuestMemory := resource.MustParse("1042Mi")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/memory/guest", guest.String(), newGuestMemory.String())
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for HotMemoryChange condition to appear")
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceMemoryChange))

			By("Ensuring live-migration started")
			migration := getVMIMigration(virtClient, vmi)
			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Ensuring the libvirt domain has more available guest memory")
			Eventually(func(g Gomega) int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				spec, err := libdomain.GetRunningVMIDomainSpec(vmi)
				g.Expect(err).To(Not(HaveOccurred()))
				return parseCurrentDomainMemory(spec).Value()
			}, 240*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Ensuring the VMI has more available guest memory")
			Eventually(func() int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Status.Memory.GuestCurrent.Value()
			}, 240*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Ensuring the virt-launcher pod now has more memory")
			compute, err = libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqMemory = compute.Resources.Requests.Memory().Value()
			Expect(reqMemory).To(BeNumerically(">=", newGuestMemory.Value()))
		},
			Entry("with a common VM"),
			Entry("with 2Mi pagesize hugepages VM", decorators.RequiresHugepages2Mi, libvmi.WithHugepages("2Mi")),
		)

		It("[test_id:10824]after a hotplug memory and a restart the new memory value should be the base for the VM", func() {
			By("Creating a VM")
			guest := resource.MustParse("1Gi")
			vm, vmi := createHotplugVM(nil, 0)

			By("Hotplug additional memory")
			newGuestMemory := resource.MustParse("1042Mi")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/memory/guest", guest.String(), newGuestMemory.String())
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that hotplug was successful")
			Eventually(func() int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Status.Memory.GuestCurrent.Value()
			}, 360*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Stopping the VM")
			stopOptions := &v1.StopOptions{GracePeriod: pointer.P(int64(0))}
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, stopOptions)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

			By("Restarting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 480*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Checking the new guest memory base value")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(newGuestMemory.Value()))
		})

		It("[test_id:10825]should successfully hotplug Memory and CPU in parallel", func() {
			By("Creating a VM")
			guest := resource.MustParse("1Gi")
			newSockets := uint32(2)
			vm, vmi := createHotplugVM(pointer.P(uint32(1)), newSockets)

			By("Hotplug Memory and CPU")
			newGuestMemory := resource.MustParse("1042Mi")
			patchData, err := patch.New(
				patch.WithTest("/spec/template/spec/domain/memory/guest", guest.String()),
				patch.WithReplace("/spec/template/spec/domain/memory/guest", newGuestMemory.String()),
				patch.WithTest("/spec/template/spec/domain/cpu/sockets", vmi.Spec.Domain.CPU.Sockets),
				patch.WithReplace("/spec/template/spec/domain/cpu/sockets", newSockets),
			).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that Memory hotplug was successful")
			Eventually(func() int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Status.Memory.GuestCurrent.Value()
			}, 360*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Checking that CPU hotplug was successful")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Status.CurrentCPUTopology.Sockets == newSockets
			}, 360*time.Second, time.Second).Should(BeTrue())

			By("Checking the correctness of the VMI memory requests")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().Value()).To(Equal(newGuestMemory.Value()))
		})

		It("should successfully hotplug memory when adding guest.memory to a VM", func() {
			By("Creating a VM")
			guest := resource.MustParse("1Gi")
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(guest.String()))
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.Memory = &v1.Memory{}

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			EventuallyWithOffset(1, ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the compute container has the expected memory")
			compute, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqMemory := compute.Resources.Requests.Memory().Value()
			Expect(reqMemory).To(BeNumerically(">=", guest.Value()))

			By("Hotplug additional memory")
			newMemory := resource.MustParse("1042Mi")
			patchSet := patch.New(
				patch.WithAdd("/spec/template/spec/domain/memory/guest", newMemory.String()),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).NotTo(HaveOccurred())

			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for HotMemoryChange condition to appear")
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceMemoryChange))

			By("Ensuring live-migration started")
			migration := getVMIMigration(virtClient, vmi)
			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Ensuring the libvirt domain has more available guest memory")
			Eventually(func(g Gomega) int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				spec, err := libdomain.GetRunningVMIDomainSpec(vmi)
				g.Expect(err).To(Not(HaveOccurred()))
				return parseCurrentDomainMemory(spec).Value()
			}, 240*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Ensuring the VMI has more available guest memory")
			Eventually(func() int64 {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Status.Memory.GuestCurrent.Value()
			}, 240*time.Second, time.Second).Should(BeNumerically(">", guest.Value()))

			By("Ensuring the virt-launcher pod now has more memory")
			compute, err = libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqMemory = compute.Resources.Requests.Memory().Value()
			Expect(reqMemory).To(BeNumerically(">=", newMemory.Value()))
		})

		// This is needed as the first hotplug attaches the virtio-mem device
		// while the next ones only update the device. This test exercises
		// both cases
		It("should successfully hotplug memory twice", func() {
			By("Creating a VM")
			vm, vmi := createHotplugVM(nil, 0)

			for _, newMemory := range []*resource.Quantity{pointer.P(resource.MustParse("1028Mi")), pointer.P(resource.MustParse("1042Mi"))} {
				oldGuestMemory := vm.Spec.Template.Spec.Domain.Memory.Guest

				By("Ensuring the compute container has the expected memory")
				compute, err := libpod.LookupComputeContainerFromVmi(vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(compute).NotTo(BeNil(), "failed to find compute container")
				reqMemory := compute.Resources.Requests.Memory().Value()
				Expect(reqMemory).To(BeNumerically(">=", oldGuestMemory.Value()))

				By("Hotplug some memory")
				patchSet := patch.New(
					patch.WithAdd("/spec/template/spec/domain/memory/guest", newMemory.String()),
				)
				patchBytes, err := patchSet.GeneratePayload()
				Expect(err).NotTo(HaveOccurred())

				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for HotMemoryChange condition to appear")
				Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceMemoryChange))

				By("Ensuring live-migration started")
				migration := getVMIMigration(virtClient, vmi)
				libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

				By("Ensuring the libvirt domain has more available guest memory")
				Eventually(func(g Gomega) int64 {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					spec, err := libdomain.GetRunningVMIDomainSpec(vmi)
					g.Expect(err).To(Not(HaveOccurred()))
					return parseCurrentDomainMemory(spec).Value()
				}, 240*time.Second, time.Second).Should(BeNumerically(">", oldGuestMemory.Value()))

				By("Ensuring the VMI has more available guest memory")
				Eventually(func() int64 {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return vmi.Status.Memory.GuestCurrent.Value()
				}, 240*time.Second, time.Second).Should(BeNumerically(">", oldGuestMemory.Value()))

				By("Ensuring the virt-launcher pod now has more memory")
				compute, err = libpod.LookupComputeContainerFromVmi(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(compute).NotTo(BeNil(), "failed to find compute container")
				reqMemory = compute.Resources.Requests.Memory().Value()
				Expect(reqMemory).To(BeNumerically(">=", newMemory.Value()))
			}
		})

		It("should detect a failed memory hotplug", func() {
			By("Creating a VM")
			guest := resource.MustParse("1Gi")
			vmi := libvmifact.NewAlpine(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithAnnotation(v1.FuncTestMemoryHotplugFailAnnotation, ""),
				libvmi.WithResourceMemory(guest.String()),
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guest}

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			EventuallyWithOffset(1, ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Hotplug additional memory")
			newMemory := resource.MustParse("1042Mi")
			patchSet := patch.New(
				patch.WithAdd("/spec/template/spec/domain/memory/guest", newMemory.String()),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).NotTo(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Detect failed memory hotplug")
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionFalse(v1.VirtualMachineInstanceMemoryChange))
			Eventually(ThisVM(vm), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineRestartRequired))

			By("Checking that migration has been marked as succeeded")
			migration := getVMIMigration(virtClient, vmi)
			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)
		})
	})
})

func getVMIMigration(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceMigration {
	var migration *v1.VirtualMachineInstanceMigration
	EventuallyWithOffset(1, func() bool {
		listOpts := k8smetav1.ListOptions{
			LabelSelector: fmt.Sprintf("kubevirt.io/vmi-name=%s", vmi.Name),
		}
		migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(context.Background(), listOpts)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		if len(migrations.Items) > 0 {
			migration = migrations.Items[0].DeepCopy()
			return true
		}
		return false
	}, 30*time.Second, time.Second).Should(BeTrue(), "A migration should be created")
	return migration
}
