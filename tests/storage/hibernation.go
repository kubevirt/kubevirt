/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	hibernationTimeout = 5 * time.Minute
	resumeTimeout      = 5 * time.Minute
)

var _ = Describe(SIG("Hibernation", Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
	)

	BeforeEach(func() {
		kvconfig.EnableFeatureGate(featuregate.UtilityVolumesGate)
		kvconfig.EnableFeatureGate(featuregate.VMHibernation)
		virtClient = kubevirt.Client()

		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(libdv.StorageWithVolumeSize(cd.AlpineVolumeSize)),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv, libvmi.WithRunStrategy(v1.RunStrategyAlways))

		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
	})

	savedState := func() *v1.VirtualMachineSavedState {
		updated, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return updated.Status.SavedState
	}

	savedStatePhase := func() v1.VirtualMachineSavedStatePhase {
		if state := savedState(); state != nil {
			return state.Phase
		}
		return ""
	}

	runningVMI := func() *v1.VirtualMachineInstance {
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi).To(matcher.BeRunning())
		return vmi
	}

	bootID := func() string {
		vmi := runningVMI()
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
		output, err := console.RunCommandAndStoreOutput(vmi, "cat /proc/sys/kernel/random/boot_id", 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		return strings.TrimSpace(output)
	}

	// hibernate waits until the state is saved and the VMI it was saved from is gone.
	hibernate := func() *v1.VirtualMachineSavedState {
		By("Hibernating the VM")
		Expect(virtClient.VirtualMachine(vm.Namespace).Hibernate(context.Background(), vm.Name, &v1.HibernateOptions{})).To(Succeed())
		Eventually(savedStatePhase, hibernationTimeout, time.Second).Should(Equal(v1.HibernationSaved))
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), time.Minute, time.Second).Should(matcher.BeGone())
		return savedState()
	}

	start := func() {
		By("Starting the VM")
		Expect(virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})).To(Succeed())
	}

	resume := func(state *v1.VirtualMachineSavedState) {
		start()
		Eventually(savedStatePhase, resumeTimeout, time.Second).Should(Equal(v1.HibernationRestored))
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), resumeTimeout, time.Second).Should(matcher.BeRunning())

		By("Releasing the PVC the VM resumed from while it runs")
		Eventually(matcher.ThisPVCWith(vm.Namespace, state.ClaimName), 2*time.Minute, time.Second).Should(matcher.BeGone())
		Expect(runningVMI().Spec.UtilityVolumes).To(BeEmpty())
	}

	It("should resume the same boot of the guest, and hibernate it again", func() {
		bootBefore := bootID()

		for cycle := range 2 {
			By(fmt.Sprintf("Hibernating and resuming, cycle %d", cycle+1))
			state := hibernate()
			Expect(state.FileName).ToNot(BeEmpty())
			Expect(matcher.ThisPVCWith(vm.Namespace, state.ClaimName)()).To(matcher.Exist())

			resume(state)
			Expect(bootID()).To(Equal(bootBefore), "a resume continues the same boot")
		}
	})

	DescribeTable("should keep the VM running when a pending hibernation is cancelled", func(cancel func()) {
		vmi := runningVMI()

		By("Hibernating to a storage class that never provisions the PVC, which keeps the hibernation pending")
		Expect(virtClient.VirtualMachine(vm.Namespace).Hibernate(context.Background(), vm.Name, &v1.HibernateOptions{
			StorageClassName: new("hibernation-does-not-exist"),
		})).To(Succeed())
		Eventually(savedStatePhase, time.Minute, time.Second).Should(Equal(v1.HibernationPending))
		Consistently(savedStatePhase, 10*time.Second, time.Second).Should(Equal(v1.HibernationPending))

		cancel()

		Eventually(savedState, time.Minute, time.Second).Should(BeNil())
		updated, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updated.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)), "the run strategy is restored")
		Consistently(func() types.UID { return runningVMI().UID }, 15*time.Second, time.Second).Should(Equal(vmi.UID))
	},
		Entry("by discarding it", func() {
			Expect(virtClient.VirtualMachine(vm.Namespace).DiscardSavedState(context.Background(), vm.Name)).To(Succeed())
		}),
		Entry("by starting the VM", func() {
			Expect(virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})).To(Succeed())
		}),
	)

	It("should keep a saved state that cannot be resumed, retry it on a start, and boot the VM once discarded", func() {
		bootBefore := bootID()
		state := hibernate()

		By("Corrupting the saved state")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), state.ClaimName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		pod := createExecutorPodWithPVC("corrupt-saved-state", pvc)
		_, err = exec.ExecuteCommandOnPod(pod, pod.Spec.Containers[0].Name, []string{
			"/bin/bash", "-c", "truncate -s 4096 " + libstorage.DefaultPvcMountPath + "/" + state.FileName,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})).To(Succeed())
		Eventually(matcher.ThisPod(pod), time.Minute, time.Second).Should(matcher.BeGone())

		start()
		Eventually(savedStatePhase, resumeTimeout, time.Second).Should(Equal(v1.HibernationRestoreFailed))
		Expect(savedState().Message).To(ContainSubstring("Resuming from the saved state failed"))

		By("Holding the VM stopped, without retrying the restore or booting")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), time.Minute, time.Second).Should(matcher.BeGone())
		Consistently(matcher.ThisVMIWith(vm.Namespace, vm.Name), 20*time.Second, time.Second).Should(matcher.BeGone())
		Expect(savedStatePhase()).To(Equal(v1.HibernationRestoreFailed))
		Expect(matcher.ThisPVCWith(vm.Namespace, state.ClaimName)()).To(matcher.Exist())
		Expect(matcher.ThisVMWith(vm.Namespace, vm.Name)()).To(HaveField("Status.PrintableStatus", Equal(v1.VirtualMachineStatusResumeFailed)))

		By("Retrying the resume with a start")
		start()
		Eventually(savedStatePhase, resumeTimeout, time.Second).Should(Equal(v1.HibernationRestoreFailed))
		Expect(matcher.ThisPVCWith(vm.Namespace, state.ClaimName)()).To(matcher.Exist())

		By("Discarding the saved state")
		Expect(virtClient.VirtualMachine(vm.Namespace).DiscardSavedState(context.Background(), vm.Name)).To(Succeed())
		Eventually(savedState, time.Minute, time.Second).Should(BeNil())
		Eventually(matcher.ThisPVCWith(vm.Namespace, state.ClaimName), time.Minute, time.Second).Should(matcher.BeGone())

		By("Booting the VM")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), resumeTimeout, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Expect(bootID()).ToNot(Equal(bootBefore))
	})

	It("should stop a VM deleted while its hibernation is pending, and delete its saved state PVC", func() {
		Expect(virtClient.VirtualMachine(vm.Namespace).Hibernate(context.Background(), vm.Name, &v1.HibernateOptions{
			StorageClassName: new("hibernation-does-not-exist"),
		})).To(Succeed())
		Eventually(savedStatePhase, time.Minute, time.Second).Should(Equal(v1.HibernationPending))
		claimName := savedState().ClaimName

		Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())
		Eventually(matcher.ThisVMWith(vm.Namespace, vm.Name), 3*time.Minute, time.Second).Should(matcher.BeGone())
		Eventually(matcher.ThisPVCWith(vm.Namespace, claimName), 2*time.Minute, time.Second).Should(matcher.BeGone())
	})

	It("should resume a VM that runs once with its run strategy", func() {
		updated, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updated.Spec.RunStrategy = new(v1.RunStrategyOnce)
		_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), updated, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		state := hibernate()
		Expect(state.RunStrategy).To(HaveValue(Equal(v1.RunStrategyOnce)))
		resume(state)
		updated, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updated.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyOnce)), "a resume must not turn Once into Always")
	})

	It("should refuse to hibernate a VM whose disks do not persist", func() {
		ephemeral := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
			libvmi.WithMemoryRequest("128Mi"),
		), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		ephemeral, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), ephemeral, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(ephemeral.Namespace, ephemeral.Name), 5*time.Minute, time.Second).Should(matcher.BeRunning())

		err = virtClient.VirtualMachine(ephemeral.Namespace).Hibernate(context.Background(), ephemeral.Name, &v1.HibernateOptions{})
		Expect(err).To(MatchError(ContainSubstring("does not persist")))
	})

	It("should refuse to hibernate a VM with a vTPM", func() {
		withTPM := libstorage.RenderVMWithDataVolumeTemplate(libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(libdv.StorageWithVolumeSize(cd.AlpineVolumeSize)),
		), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		withTPM.Spec.Template.Spec.Domain.Devices.TPM = &v1.TPMDevice{}
		withTPM, err := virtClient.VirtualMachine(withTPM.Namespace).Create(context.Background(), withTPM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(withTPM.Namespace, withTPM.Name), 12*time.Minute, 2*time.Second).Should(matcher.BeRunning())

		err = virtClient.VirtualMachine(withTPM.Namespace).Hibernate(context.Background(), withTPM.Name, &v1.HibernateOptions{})
		Expect(err).To(MatchError(ContainSubstring("vTPM")))
	})
}))
