//nolint:lll
package instancetype

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preference Revisions", decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("[test_id:CNV-9098] should store and use ControllerRevisions of VirtualMachineInstancetypeSpec and VirtualMachinePreferenceSpec", func() {
		By("Creating a VirtualMachineInstancetype")
		vmi := libvmifact.NewGuestless()
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
		vmi := libvmifact.NewGuestless()
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
