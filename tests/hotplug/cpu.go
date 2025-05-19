package hotplug

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	testsmig "kubevirt.io/kubevirt/tests/migration"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]CPU Hotplug", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateRolloutStrategy, Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
	)
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

	Context("with requests without topology", func() {

		It("should be able to start", func() {
			By("Kubevirt CR with default MaxHotplugRatio set to 4")

			By("Run VM with 5 sockets without topology")
			vmi := libvmifact.NewAlpine(libvmi.WithResourceCPU("5000m"))

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			By("Expecting to see VMI that is starting")
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 10*time.Second, 1*time.Second).Should(Exist())
		})
	})

	Context("with Kubevirt CR declaring MaxCpuSockets", func() {

		It("should be able to start", func() {
			By("Kubevirt CR with MaxCpuSockets set to 2")
			kubevirt := libkubevirt.GetCurrentKv(virtClient)
			if kubevirt.Spec.Configuration.LiveUpdateConfiguration == nil {
				kubevirt.Spec.Configuration.LiveUpdateConfiguration = &v1.LiveUpdateConfiguration{}
			}
			kubevirt.Spec.Configuration.LiveUpdateConfiguration.MaxCpuSockets = pointer.P(uint32(2))
			kvconfig.UpdateKubeVirtConfigValueAndWait(kubevirt.Spec.Configuration)

			By("Run VM with 3 sockets")
			vmi := libvmifact.NewAlpine(libvmi.WithCPUCount(1, 1, 3))
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			By("Expecting to see VMI that is starting")
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 10*time.Second, 1*time.Second).Should(Exist())
		})

	})

	Context("A VM with cpu.maxSockets set higher than cpu.sockets", func() {
		type cpuCount struct {
			enabled  int
			disabled int
		}
		countDomCPUs := func(spec *api.DomainSpec) (count cpuCount) {
			ExpectWithOffset(1, spec.VCPUs).NotTo(BeNil())
			for _, cpu := range spec.VCPUs.VCPU {
				if cpu.Enabled == "yes" {
					count.enabled++
				} else {
					ExpectWithOffset(1, cpu.Enabled).To(Equal("no"))
					ExpectWithOffset(1, cpu.Hotpluggable).To(Equal("yes"))
					count.disabled++
				}
			}
			return
		}
		It("[test_id:10811]should successfully plug vCPUs", func() {
			By("Creating a running VM with 1 socket and 2 max sockets")
			const (
				maxSockets uint32 = 2
			)

			vmi := libvmifact.NewAlpineWithTestTooling(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithNetworkInterfaceMultiQueue(true),
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Sockets:    1,
				Cores:      2,
				Threads:    1,
				MaxSockets: maxSockets,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the compute container has 200m CPU")
			compute, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu := compute.Resources.Requests.Cpu().Value()
			expCpu := resource.MustParse("200m")
			Expect(reqCpu).To(Equal(expCpu.Value()))

			By("Ensuring the libvirt domain has 2 enabled cores and 2 hotpluggable cores")
			var domSpec *api.DomainSpec
			Eventually(func() error {
				domSpec, err = libdomain.GetRunningVMIDomainSpec(vmi)
				return err
			}).WithTimeout(20 * time.Second).WithPolling(time.Second).Should(Succeed())

			Expect(countDomCPUs(domSpec)).To(Equal(cpuCount{
				enabled:  2,
				disabled: 2,
			}))

			By("Enabling the second socket")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for hot change CPU condition to appear")
			// Need to wait for the hotplug to begin.
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceVCPUChange))

			By("Ensuring live-migration started")
			var migration *v1.VirtualMachineInstanceMigration
			Eventually(func() bool {
				migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, mig := range migrations.Items {
					if mig.Spec.VMIName == vmi.Name {
						migration = mig.DeepCopy()
						return true
					}
				}
				return false
			}, 30*time.Second, time.Second).Should(BeTrue())
			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Ensuring the libvirt domain has 4 enabled cores")

			Eventually(func() error {
				domSpec, err = libdomain.GetRunningVMIDomainSpec(vmi)
				return err
			}).WithTimeout(20 * time.Second).WithPolling(time.Second).Should(Succeed())

			Expect(countDomCPUs(domSpec)).To(Equal(cpuCount{
				enabled:  4,
				disabled: 0,
			}))

			By("Ensuring the virt-launcher pod now has 400m CPU")
			compute, err = libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu = compute.Resources.Requests.Cpu().Value()
			expCpu = resource.MustParse("400m")
			Expect(reqCpu).To(Equal(expCpu.Value()))

			By("Ensuring the vm doesn't have a RestartRequired condition")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(ThisVM(vm), 4*time.Minute, 2*time.Second).Should(HaveConditionMissingOrFalse(v1.VirtualMachineRestartRequired))

			By("Changing the number of CPU cores")
			patchData, err = patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/cores", 2, 4)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the vm has a RestartRequired condition")
			Eventually(ThisVM(vm), 4*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineRestartRequired))

			By("Restarting the VM and expecting RestartRequired to be gone")
			err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(ThisVM(vm), 4*time.Minute, 2*time.Second).Should(HaveConditionMissingOrFalse(v1.VirtualMachineRestartRequired))
		})

		It("[test_id:10822]should successfully plug guaranteed vCPUs", decorators.RequiresTwoWorkerNodesWithCPUManager, func() {
			checks.ExpectAtLeastTwoWorkerNodesWithCPUManager(virtClient)
			const maxSockets uint32 = 3

			By("Creating a running VM with 1 socket and 2 max sockets")
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				Sockets:               1,
				Threads:               1,
				DedicatedCPUPlacement: true,
				MaxSockets:            maxSockets,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the compute container has 2 CPU")
			compute, err := libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu := compute.Resources.Requests.Cpu().Value()
			expCpu := resource.MustParse("2")
			Expect(reqCpu).To(Equal(expCpu.Value()))

			By("Ensuring the libvirt domain has 2 enabled cores and 4 disabled cores")
			var domSpec *api.DomainSpec
			Eventually(func() error {
				domSpec, err = libdomain.GetRunningVMIDomainSpec(vmi)
				return err
			}).WithTimeout(20 * time.Second).WithPolling(time.Second).Should(Succeed())

			Expect(countDomCPUs(domSpec)).To(Equal(cpuCount{
				enabled:  2,
				disabled: 4,
			}))

			By("starting the migration")
			migration := libmigration.New(vm.Name, vm.Namespace)
			migration, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Enabling the second socket")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for hot change CPU condition to appear")
			// Need to wait for the hotplug to begin.
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceVCPUChange))

			By("Ensuring hotplug ended")
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).ShouldNot(SatisfyAny(
				HaveConditionTrue(v1.VirtualMachineInstanceVCPUChange),
				HaveConditionFalse(v1.VirtualMachineInstanceVCPUChange),
			))

			By("Ensuring the libvirt domain has 4 enabled cores")
			Eventually(func(g Gomega) cpuCount {
				domSpec, err = libdomain.GetRunningVMIDomainSpec(vmi)
				g.Expect(err).NotTo(HaveOccurred())
				return countDomCPUs(domSpec)
			}).WithTimeout(20 * time.Second).WithPolling(time.Second).Should(Equal(cpuCount{
				enabled:  4,
				disabled: 2,
			}))

			By("Ensuring the virt-launcher pod now has 4 CPU")
			compute, err = libpod.LookupComputeContainerFromVmi(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu = compute.Resources.Requests.Cpu().Value()
			expCpu = resource.MustParse("4")
			Expect(reqCpu).To(Equal(expCpu.Value()))
		})
	})

	Context("Abort CPU change", func() {
		It("should cancel the automated workload update", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Sockets:    1,
				Cores:      2,
				Threads:    1,
				MaxSockets: 2,
			}
			By("Limiting the bandwidth of migrations in the test namespace")
			policy := testsmig.PreparePolicyAndVMIWithBandwidthLimitation(vmi, resource.MustParse("1Ki"))
			testsmig.CreateMigrationPolicy(virtClient, policy)
			Eventually(func() *migrationsv1.MigrationPolicy {
				policy, err := virtClient.MigrationPolicy().Get(context.Background(), policy.Name, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return policy
			}, 30*time.Second, time.Second).ShouldNot(BeNil())

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			// Update the CPU number and trigger the workload update
			// and migration
			By("Enabling the second socket to trigger the migration update")
			p, err := patch.New(patch.WithReplace("/spec/template/spec/domain/cpu/sockets", 2)).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, mig := range migrations.Items {
					if mig.Spec.VMIName == vmi.Name {
						return true
					}
				}
				return false
			}, 30*time.Second, time.Second).Should(BeTrue())

			// Add annotation to cancel the workload update
			By("Patching the workload migration abortion annotation")
			vmi.ObjectMeta.Annotations[v1.WorkloadUpdateMigrationAbortionAnnotation] = ""
			p, err = patch.New(patch.WithAdd("/metadata/annotations", vmi.ObjectMeta.Annotations)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return metav1.HasAnnotation(vmi.ObjectMeta, v1.WorkloadUpdateMigrationAbortionAnnotation)
			}, 30*time.Second, time.Second).Should(BeTrue())

			// Wait until the migration is cancelled by the workload
			// updater
			Eventually(func() bool {
				migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, mig := range migrations.Items {
					if mig.Spec.VMIName == vmi.Name {
						return true
					}
				}
				return false
			}, 30*time.Second, time.Second).Should(BeFalse())

		})
	})
})

func patchWorkloadUpdateMethodAndRolloutStrategy(kvName string, virtClient kubecli.KubevirtClient, updateStrategy *v1.KubeVirtWorkloadUpdateStrategy, rolloutStrategy *v1.VMRolloutStrategy) {
	methodData, err := json.Marshal(updateStrategy)
	ExpectWithOffset(1, err).To(Not(HaveOccurred()))
	rolloutData, err := json.Marshal(rolloutStrategy)
	ExpectWithOffset(1, err).To(Not(HaveOccurred()))

	data1 := fmt.Sprintf(`{"op": "replace", "path": "/spec/workloadUpdateStrategy", "value": %s}`, string(methodData))
	data2 := fmt.Sprintf(`{"op": "replace", "path": "/spec/configuration/vmRolloutStrategy", "value": %s}`, string(rolloutData))
	data := []byte(fmt.Sprintf(`[%s, %s]`, data1, data2))

	EventuallyWithOffset(1, func() error {
		_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kvName, types.JSONPatchType, data, k8smetav1.PatchOptions{})
		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}
