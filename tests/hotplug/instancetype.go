package hotplug

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute][Serial]Instance Type and Preference Hotplug", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateFeaturesGate, Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
		vm         *virtv1.VirtualMachine
		vmi        *virtv1.VirtualMachineInstance

		originalSockets uint32
		maxSockets      uint32

		originalGuest resource.Quantity
		maxGuest      resource.Quantity
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv := util.GetCurrentKv(virtClient)
		updateStrategy := &virtv1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []virtv1.WorkloadUpdateMethod{virtv1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(virtv1.VMRolloutStrategyLiveUpdate)
		patchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

		currentKv := util.GetCurrentKv(virtClient)
		tests.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			tests.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)

		vmi = libvmifact.NewAlpineWithTestTooling(
			libnet.WithMasqueradeNetworking()...,
		)
		vmi.Namespace = testsuite.GetTestNamespace(vmi)

		originalSockets = uint32(1)
		maxSockets = originalSockets + 1

		originalGuest = vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
		maxGuest = originalGuest.DeepCopy()
		maxGuest.Add(resource.MustParse("128Mi"))

		vmi.Spec.Domain.CPU = nil
		vmi.Spec.Domain.Memory = nil
		vmi.Spec.Domain.Resources = virtv1.ResourceRequirements{}

		vm = libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())
	})

	DescribeTable("should plug extra resources from new instance type", func(withMaxGuestSockets bool) {
		originalInstancetype := &instancetypev1beta1.VirtualMachineInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "original-instancetype",
				Namespace: vmi.Namespace,
			},
			Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1beta1.CPUInstancetype{
					Guest: originalSockets,
				},
				Memory: instancetypev1beta1.MemoryInstancetype{
					Guest: originalGuest,
				},
			},
		}
		if withMaxGuestSockets {
			originalInstancetype.Spec.CPU.MaxSockets = maxSockets
			originalInstancetype.Spec.Memory.MaxGuest = &maxGuest
		}
		originalInstancetype, err := virtClient.VirtualMachineInstancetype(vmi.Namespace).Create(context.Background(), originalInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		maxInstancetype := &instancetypev1beta1.VirtualMachineInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "max-instancetype",
				Namespace: vmi.Namespace,
			},
			Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1beta1.CPUInstancetype{
					Guest: maxSockets,
				},
				Memory: instancetypev1beta1.MemoryInstancetype{
					Guest: maxGuest,
				},
			},
		}
		if withMaxGuestSockets {
			maxInstancetype.Spec.CPU.MaxSockets = maxSockets
			maxInstancetype.Spec.Memory.MaxGuest = &maxGuest
		}
		maxInstancetype, err = virtClient.VirtualMachineInstancetype(vmi.Namespace).Create(context.Background(), maxInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// TODO - Move this into libvmi as a VMOption for the above NewVirtualMachine call
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			Name: originalInstancetype.Name,
			Kind: instancetypeapi.SingularResourceName,
		}

		vm, err = virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm)
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
		libwait.WaitForSuccessfulVMIStart(vmi)

		By("Switching to the max instance type")
		patchData, err := patch.GeneratePatchPayload(
			patch.PatchOperation{
				Op:    patch.PatchTestOp,
				Path:  "/spec/instancetype/name",
				Value: originalInstancetype.Name,
			},
			patch.PatchOperation{
				Op:    patch.PatchReplaceOp,
				Path:  "/spec/instancetype/name",
				Value: maxInstancetype.Name,
			},
			patch.PatchOperation{
				Op:   patch.PatchRemoveOp,
				Path: "/spec/instancetype/revisionName",
			},
		)
		Expect(err).NotTo(HaveOccurred())
		_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for HotVCPUChange condition to appear")
		Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(virtv1.VirtualMachineInstanceVCPUChange))

		By("Waiting for HotMemoryChange condition to appear")
		Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(virtv1.VirtualMachineInstanceMemoryChange))

		By("Ensuring live-migration started")
		var migration *virtv1.VirtualMachineInstanceMigration
		Eventually(func() bool {
			migrations, err := virtClient.VirtualMachineInstanceMigration(vm.Namespace).List(&metav1.ListOptions{})
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

		By("Waiting for HotVCPUChange condition to disappear")
		Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionMissingOrFalse(virtv1.VirtualMachineInstanceVCPUChange))

		By("Waiting for HotMemoryChange condition to disappear")
		Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionMissingOrFalse(virtv1.VirtualMachineInstanceMemoryChange))

		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(maxSockets))
		Expect(vmi.Status.CurrentCPUTopology).ToNot(BeNil())
		Expect(vmi.Status.CurrentCPUTopology.Sockets).To(Equal(maxSockets))

		Expect(vmi.Spec.Domain.Memory.Guest.Value()).To(Equal(maxGuest.Value()))
		Expect(vmi.Status.Memory).ToNot(BeNil())
		Expect(vmi.Status.Memory.GuestAtBoot.Value()).To(Equal(originalGuest.Value()))
		Expect(vmi.Status.Memory.GuestRequested.Value()).To(Equal(maxGuest.Value()))
		Expect(vmi.Status.Memory.GuestCurrent.Value()).To(Equal(maxGuest.Value()))

		By("Ensuring the vm doesn't have a RestartRequired condition")
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(ThisVM(vm), 4*time.Minute, 2*time.Second).Should(HaveConditionMissingOrFalse(virtv1.VirtualMachineRestartRequired))
	},
		Entry("with maxGuest and maxSockets defined", true),
		Entry("without maxGuest and maxSockets defined", false),
	)
})
