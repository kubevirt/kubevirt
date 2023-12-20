package hotplug

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/framework/checks"

	"kubevirt.io/kubevirt/tests/libmigration"

	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/flags"
	util2 "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute][Serial]CPU Hotplug", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateFeaturesGate, Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv := util2.GetCurrentKv(virtClient)
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := &v1.VMRolloutStrategy{
			LiveUpdate: &v1.RolloutStrategyLiveUpdate{},
		}
		patchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

		currentKv := util2.GetCurrentKv(virtClient)
		tests.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			tests.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)

	})

	Context("A VM with cpu.maxSockets set higher than cpu.sockets", func() {
		type cpuCount struct {
			enabled  int
			disabled int
		}
		countDomCPUs := func(vmi *v1.VirtualMachineInstance) (count cpuCount) {
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, domSpec.VCPUs).NotTo(BeNil())
			for _, cpu := range domSpec.VCPUs.VCPU {
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
		It("should successfully plug vCPUs", func() {
			By("Creating a running VM with 1 socket and 2 max sockets")
			const (
				maxSockets uint32 = 2
			)

			vmi := libvmi.NewAlpineWithTestTooling(
				libvmi.WithMasqueradeNetworking()...,
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Sockets:    1,
				Cores:      2,
				Threads:    1,
				MaxSockets: maxSockets,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the compute container has 200m CPU")
			compute := tests.GetComputeContainerOfPod(tests.GetVmiPod(virtClient, vmi))

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu := compute.Resources.Requests.Cpu().Value()
			expCpu := resource.MustParse("200m")
			Expect(reqCpu).To(Equal(expCpu.Value()))

			By("Ensuring the libvirt domain has 2 enabled cores and 2 hotpluggable cores")
			Expect(countDomCPUs(vmi)).To(Equal(cpuCount{
				enabled:  2,
				disabled: 2,
			}))

			By("Enabling the second socket")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for hot change CPU condition to appear")
			// Need to wait for the hotplug to begin.
			Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceVCPUChange))

			By("Ensuring live-migration started")
			var migration *v1.VirtualMachineInstanceMigration
			Eventually(func() bool {
				migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(&k8smetav1.ListOptions{})
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
			Eventually(func() cpuCount {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return countDomCPUs(vmi)
			}, 240*time.Second, time.Second).Should(Equal(cpuCount{
				enabled:  4,
				disabled: 0,
			}))

			By("Ensuring the virt-launcher pod now has 400m CPU")
			compute = tests.GetComputeContainerOfPod(tests.GetVmiPod(virtClient, vmi))

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu = compute.Resources.Requests.Cpu().Value()
			expCpu = resource.MustParse("400m")
			Expect(reqCpu).To(Equal(expCpu.Value()))
		})

		It("should successfully plug guaranteed vCPUs", decorators.RequiresTwoWorkerNodesWithCPUManager, func() {
			checks.ExpectAtLeastTwoWorkerNodesWithCPUManager(virtClient)
			const maxSockets uint32 = 3

			By("Creating a running VM with 1 socket and 2 max sockets")
			vmi := libvmi.NewAlpineWithTestTooling(
				libvmi.WithMasqueradeNetworking()...,
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				Sockets:               1,
				Threads:               1,
				DedicatedCPUPlacement: true,
				MaxSockets:            maxSockets,
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the compute container has 2 CPU")
			compute := tests.GetComputeContainerOfPod(tests.GetVmiPod(virtClient, vmi))

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu := compute.Resources.Requests.Cpu().Value()
			expCpu := resource.MustParse("2")
			Expect(reqCpu).To(Equal(expCpu.Value()))

			By("Ensuring the libvirt domain has 2 enabled cores and 4 disabled cores")
			Expect(countDomCPUs(vmi)).To(Equal(cpuCount{
				enabled:  2,
				disabled: 4,
			}))

			By("starting the migration")
			migration := libmigration.New(vm.Name, vm.Namespace)
			migration, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Create(migration, &metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Enabling the second socket")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &k8smetav1.PatchOptions{})
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
			Eventually(func() cpuCount {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return countDomCPUs(vmi)
			}, 30*time.Second, time.Second).Should(Equal(cpuCount{
				enabled:  4,
				disabled: 2,
			}))

			By("Ensuring the virt-launcher pod now has 4 CPU")
			compute = tests.GetComputeContainerOfPod(tests.GetVmiPod(virtClient, vmi))

			Expect(compute).NotTo(BeNil(), "failed to find compute container")
			reqCpu = compute.Resources.Requests.Cpu().Value()
			expCpu = resource.MustParse("4")
			Expect(reqCpu).To(Equal(expCpu.Value()))
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
		_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(kvName, types.JSONPatchType, data, &k8smetav1.PatchOptions{})
		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func beReady() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Ready": BeTrue(),
		}),
	}))
}
