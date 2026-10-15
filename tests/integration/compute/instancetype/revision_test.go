/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

func (s *instancetypeSuite) describeRevisions() {
	Context("ControllerRevision storage", func() {
		var vm *virtv1.VirtualMachine
		var it *instancetypev1beta1.VirtualMachineInstancetype
		var pref *instancetypev1beta1.VirtualMachinePreference

		BeforeEach(func(ctx SpecContext) {
			var err error
			it, err = s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Create(ctx, newInstancetype(), metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			pref, err = s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, newPreference(), metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithInstancetype(it.Name), libvmi.WithPreference(pref.Name))
		})

		It("should store and reuse revisions after the source changes [test_id:CNV-9098]", func(ctx SpecContext) {
			var err error
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(matcher.HaveControllerRevisionRefs())
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			originalITRevision := vm.Status.InstancetypeRef.ControllerRevisionRef.Name
			originalPrefRevision := vm.Status.PreferenceRef.ControllerRevisionRef.Name
			expectRevisionSpec(ctx, s.f, s.namespace, originalITRevision, &it.Spec)
			expectRevisionSpec(ctx, s.f, s.namespace, originalPrefRevision, &pref.Spec)

			vm = patchVM(ctx, s.f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyAlways))
			originalVMI := waitForScheduledVMI(ctx, s.f, vm)
			Expect(originalVMI.Spec.Domain.CPU.Sockets).To(Equal(it.Spec.CPU.Guest))

			By("updating the source instancetype and preference")
			originalCPUs := it.Spec.CPU.Guest
			payload, err := patch.New(patch.WithReplace("/spec/cpu/guest", originalCPUs+1)).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())
			it, err = s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Patch(
				ctx, it.Name, types.JSONPatchType, payload, metav1.PatchOptions{},
			)
			Expect(err).NotTo(HaveOccurred())
			payload, err = patch.New(patch.WithReplace("/spec/cpu/preferredCPUTopology", instancetypev1beta1.Cores)).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())
			pref, err = s.f.VirtClient().VirtualMachinePreference(s.namespace).Patch(
				ctx, pref.Name, types.JSONPatchType, payload, metav1.PatchOptions{},
			)
			Expect(err).NotTo(HaveOccurred())

			By("recreating the VMI while retaining the VM's stored revisions")
			vm = patchVM(ctx, s.f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyHalted))
			// There is no virt-handler or kubelet in envtest. Complete pod deletion
			// so the real controllers can finish stopping the scheduled VMI.
			Eventually(ctx, matcher.ThisVMI(originalVMI), 30*time.Second, 100*time.Millisecond).Should(
				HaveField("DeletionTimestamp", Not(BeNil())),
			)
			deleteLauncherPods(ctx, s.f, originalVMI)
			Eventually(ctx, matcher.ThisVMI(originalVMI), 30*time.Second, 100*time.Millisecond).Should(matcher.BeGone())
			vm = patchVM(ctx, s.f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyAlways))
			restartedVMI := waitForScheduledVMI(ctx, s.f, vm)
			Expect(restartedVMI.UID).NotTo(Equal(originalVMI.UID))
			Expect(restartedVMI.Spec.Domain.CPU.Sockets).To(Equal(originalCPUs))
			Expect(restartedVMI.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(originalITRevision))
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(originalPrefRevision))

			By("creating another VM which must use the updated source resources")
			newVM := libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithInstancetype(it.Name), libvmi.WithPreference(pref.Name),
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
			newVM, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, newVM, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			newVMI := waitForScheduledVMI(ctx, s.f, newVM)
			Expect(newVMI.Spec.Domain.CPU.Cores).To(Equal(it.Spec.CPU.Guest))
			newVM, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, newVM.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(newVM.Status.InstancetypeRef.ControllerRevisionRef.Name).NotTo(Equal(originalITRevision))
			Expect(newVM.Status.PreferenceRef.ControllerRevisionRef.Name).NotTo(Equal(originalPrefRevision))
			expectRevisionSpec(ctx, s.f, s.namespace, newVM.Status.InstancetypeRef.ControllerRevisionRef.Name, &it.Spec)
			expectRevisionSpec(ctx, s.f, s.namespace, newVM.Status.PreferenceRef.ControllerRevisionRef.Name, &pref.Spec)
		})

		It("should fail if an existing revision contains different data [test_id:CNV-9304]", func(ctx SpecContext) {
			By("precreating a conflicting revision instead of relying on garbage collection")
			conflictingIT := it.DeepCopy()
			conflictingIT.Spec.Memory.Guest.Add(resource.MustParse("10Mi"))
			cr, err := revision.CreateControllerRevision(vm, conflictingIT)
			Expect(err).NotTo(HaveOccurred())
			cr.OwnerReferences = nil
			_, err = s.f.VirtClient().AppsV1().ControllerRevisions(s.namespace).Create(ctx, cr, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways)(vm)
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(And(
				matcher.HaveConditionTrue(virtv1.VirtualMachineFailure),
				HaveField("Status.Conditions", ContainElement(And(
					HaveField("Type", virtv1.VirtualMachineFailure),
					HaveField("Message", ContainSubstring("found existing ControllerRevision with unexpected data")),
				))),
			))
			Expect(matcher.ThisVMIWith(s.namespace, vm.Name)()).To(matcher.BeGone())
		})
	})
}

func expectRevisionSpec(ctx context.Context, f *framework.Framework, namespace, name string, spec interface{}) {
	GinkgoHelper()
	cr, err := f.VirtClient().AppsV1().ControllerRevisions(namespace).Get(ctx, name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	switch expected := spec.(type) {
	case *instancetypev1beta1.VirtualMachineInstancetypeSpec:
		var it instancetypev1beta1.VirtualMachineInstancetype
		Expect(json.Unmarshal(cr.Data.Raw, &it)).To(Succeed())
		Expect(it.Spec).To(Equal(*expected))
	case *instancetypev1beta1.VirtualMachinePreferenceSpec:
		var pref instancetypev1beta1.VirtualMachinePreference
		Expect(json.Unmarshal(cr.Data.Raw, &pref)).To(Succeed())
		Expect(pref.Spec).To(Equal(*expected))
	default:
		Fail(fmt.Sprintf("unexpected revision spec %T", spec))
	}
}

func patchVM(ctx context.Context, f *framework.Framework, vm *virtv1.VirtualMachine, opts ...patch.PatchOption) *virtv1.VirtualMachine {
	GinkgoHelper()
	payload, err := patch.New(opts...).GeneratePayload()
	Expect(err).NotTo(HaveOccurred())
	updatedVM, err := f.VirtClient().VirtualMachine(vm.Namespace).Patch(ctx, vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).NotTo(HaveOccurred())
	return updatedVM
}

func deleteLauncherPods(ctx context.Context, f *framework.Framework, vmi *virtv1.VirtualMachineInstance) {
	GinkgoHelper()
	pods, err := f.K8sClient().CoreV1().Pods(vmi.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, vmi.UID),
	})
	Expect(err).NotTo(HaveOccurred())
	for i := range pods.Items {
		Expect(f.K8sClient().CoreV1().Pods(vmi.Namespace).Delete(ctx, pods.Items[i].Name,
			metav1.DeleteOptions{GracePeriodSeconds: pointer.P(int64(0))})).To(Succeed())
	}
}
