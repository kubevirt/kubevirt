package hotplug

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/flags"
	util2 "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute][Serial]CPU Hotplug", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateFeaturesGate, Serial, func() {
	var (
		virtClient  kubecli.KubevirtClient
		workerLabel = "node-role.kubernetes.io/worker"
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv := util2.GetCurrentKv(virtClient)
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		patchWorkloadUpdateMethod(originalKv.Name, virtClient, updateStrategy)

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
				Sockets: 1,
				Cores:   2,
				Threads: 1,
			}
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm.Spec.LiveUpdateFeatures = &v1.LiveUpdateFeatures{
				CPU: &v1.LiveUpdateCPU{
					MaxSockets: pointer.P(maxSockets),
				},
			}

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

			By("Making sure that a parallel CPU change is not allowed during an ongoing hot plug")
			patchData, err = patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 2, 1)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &k8smetav1.PatchOptions{})
			Expect(err).To(HaveOccurred())

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

		It("should successfully plug guaranteed vCPUs", func() {
			var nodes []k8sv1.Node
			virtClient = kubevirt.Client()

			By("getting the list of worker nodes that have cpumanager enabled")
			nodeList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=,%s=%s", workerLabel, "cpumanager", "true"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodeList).ToNot(BeNil())
			nodes = nodeList.Items
			Expect(len(nodes)).To(BeNumerically(">=", 2), "at least two worker nodes with cpumanager are required for migration")
			By("Creating a running VM with 1 socket and 2 max sockets")
			const (
				maxSockets uint32 = 3
			)

			vmi := libvmi.NewAlpineWithTestTooling(
				libvmi.WithMasqueradeNetworking()...,
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				Sockets:               1,
				Threads:               1,
				DedicatedCPUPlacement: true,
			}
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm.Spec.LiveUpdateFeatures = &v1.LiveUpdateFeatures{
				CPU: &v1.LiveUpdateCPU{
					MaxSockets: pointer.P(maxSockets),
				},
			}

			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
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
			migration := tests.NewRandomMigration(vm.Name, vm.Namespace)
			migration, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Create(migration, &metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// This validation is being added here because currently it is very
			// costly to run it in a separate test.
			// Chaning the WorkloadUpdateMethods takes a very long time and
			// restarts the whole control plane
			By("Making sure that a hot CPU change is not allowed during an inflight migration")
			patchData, err := patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &k8smetav1.PatchOptions{})
			Expect(err).To(MatchError(ContainSubstring("cannot update while VMI migration is in progress")))

			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

			By("Enabling the second socket")
			patchData, err = patch.GenerateTestReplacePatch("/spec/template/spec/domain/cpu/sockets", 1, 2)
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

func patchWorkloadUpdateMethod(kvName string, virtClient kubecli.KubevirtClient, updateStrategy *v1.KubeVirtWorkloadUpdateStrategy) {
	methodData, err := json.Marshal(updateStrategy)
	ExpectWithOffset(1, err).To(Not(HaveOccurred()))

	data := []byte(fmt.Sprintf(`[{"op": "replace", "path": "/spec/workloadUpdateStrategy", "value": %s}]`, string(methodData)))

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
