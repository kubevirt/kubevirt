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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/backup"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnamespace"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Backup", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
		vmi        *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("VM matches cbt label selector, then unmatches", func() {
		vm = libstorage.RenderVMWithDataVolumeTemplate(libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(),
		),
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		volumeName := vm.Spec.Template.Spec.Volumes[0].Name

		By(fmt.Sprintf("Creating VM %s with CBT label", vm.Name))
		_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vm.Status.ChangedBlockTracking
		}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vmi.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		By("Verify CBT overlay exists")
		stdout := libpod.RunCommandOnVmiPod(vmi, []string{"find", backup.PathForCBT(vmi), "-type", "f", "-name", fmt.Sprintf("%s.qcow2", volumeName)})
		Expect(stdout).To(ContainSubstring(backup.GetQCOW2OverlayPath(vmi, volumeName)))

		By("Remove CBT Label")
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		delete(vm.Labels, backup.CBTKey)
		patch, err := patch.New(patch.WithAdd("/metadata/labels", vm.Labels)).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verify CBT state PendingRestart")
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vm.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingPendingRestart))

		By("Restarting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verify CBT state disabled")
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vm.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingDisabled))

		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vmi.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingDisabled))

		By("Verify CBT overlay deleted")
		libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).NotTo(BeNil())

		output, err := exec.ExecuteCommandOnPod(pod, "compute", []string{"ls", "-d", backup.PathForCBT(vmi)})
		Expect(err.Error()).To(ContainSubstring("No such file or directory"))
		Expect(output).To(BeEmpty())
	})
	DescribeTable("Patch to match cbt label selector", func(patchFunc func(vm *v1.VirtualMachine)) {
		vm = libstorage.RenderVMWithDataVolumeTemplate(libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(),
		),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		volumeName := vm.Spec.Template.Spec.Volumes[0].Name

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 360).Should(BeInPhase(v1.Running))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(vm.Status.ChangedBlockTracking).To(BeEmpty())

		patchFunc(vm)

		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vm.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingPendingRestart))

		By("Restarting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vm.Status.ChangedBlockTracking
		}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))
		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return vmi.Status.ChangedBlockTracking
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		stdout := libpod.RunCommandOnVmiPod(vmi, []string{"find", backup.PathForCBT(vmi), "-type", "f", "-name", fmt.Sprintf("%s.qcow2", volumeName)})
		Expect(stdout).To(ContainSubstring(backup.GetQCOW2OverlayPath(vmi, volumeName)))
	},
		Entry("patch vm", func(vm *v1.VirtualMachine) {
			patch, err := patch.New(patch.WithAdd("/metadata/labels", backup.CBTLabel)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

		}),
		Entry("patch vm namespace", func(vm *v1.VirtualMachine) {
			Expect(libnamespace.AddLabelToNamespace(virtClient, vm.Namespace, backup.CBTKey, "true")).ToNot(HaveOccurred())
		}),
	)
}))
