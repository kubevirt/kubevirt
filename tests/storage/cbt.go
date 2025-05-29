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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/cbt"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnamespace"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("CBT", func() {
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
			libvmi.WithLabels(cbt.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		volumeName := vm.Spec.Template.Spec.Volumes[0].Name

		By(fmt.Sprintf("Creating VM %s with CBT label", vm.Name))
		_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vmi.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		By("Verify CBT overlay exists")
		stdout := libpod.RunCommandOnVmiPod(vmi, []string{"find", cbt.PathForCBT(vmi), "-type", "f", "-name", fmt.Sprintf("%s.qcow2", volumeName)})
		Expect(stdout).To(ContainSubstring(cbt.GetQCOW2OverlayPath(vmi, volumeName)))

		By("Remove CBT Label")
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		delete(vm.Labels, cbt.CBTKey)
		patch, err := patch.New(patch.WithAdd("/metadata/labels", vm.Labels)).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verify CBT state PendingRestart")
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingPendingRestart))

		By("Restarting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())

		By("Verify CBT state disabled")
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return v1.ChangedBlockTrackingUndefined
			}
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingDisabled))

		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return v1.ChangedBlockTrackingUndefined
			}
			return cbt.CBTState(vmi.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingDisabled))

		By("Verify CBT overlay deleted")
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).NotTo(BeNil())

		var cmdOutput string
		cmdOutput, err = exec.ExecuteCommandOnPod(pod, "compute", []string{"ls", "-d", cbt.PathForCBT(vmi)})
		Expect(err.Error()).To(ContainSubstring("No such file or directory"))
		Expect(cmdOutput).To(BeEmpty())
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
		_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 360).Should(BeInPhase(v1.Running))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingUndefined))

		patchFunc(vm)

		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingPendingRestart))

		By("Restarting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))
		Eventually(func() v1.ChangedBlockTrackingState {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vmi.Status.ChangedBlockTracking)
		}, 1*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		stdout := libpod.RunCommandOnVmiPod(vmi, []string{"find", cbt.PathForCBT(vmi), "-type", "f", "-name", fmt.Sprintf("%s.qcow2", volumeName)})
		Expect(stdout).To(ContainSubstring(cbt.GetQCOW2OverlayPath(vmi, volumeName)))
	},
		Entry("patch vm", func(vm *v1.VirtualMachine) {
			patch, err := patch.New(patch.WithAdd("/metadata/labels", cbt.CBTLabel)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

		}),
		Entry("patch vm namespace", func(vm *v1.VirtualMachine) {
			Expect(libnamespace.AddLabelToNamespace(virtClient, vm.Namespace, cbt.CBTKey, "true")).ToNot(HaveOccurred())
		}),
	)

	writeDataToDisk := func(vmi *v1.VirtualMachineInstance) {
		By("Writing data to disk to trigger CBT tracking")
		err := console.RunCommand(vmi, "dd if=/dev/zero of=/tmp/testfile bs=1M count=2 && sync", 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
	}

	createConsumerPod := func(vm *v1.VirtualMachine, pvcName string) *k8sv1.Pod {
		By("Creating consumer pod to inspect qcow2 overlay")

		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		pod := libstorage.RenderPodWithPVC(
			"cbt-consumer-"+rand.String(5),
			[]string{"/bin/bash", "-c", "touch /tmp/startup; while true; do echo hello; sleep 2; done"},
			nil, pvc,
		)

		pod, err = libpod.Run(pod, vm.Namespace)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		return pod
	}

	runQemuImgInfo := func(pod *k8sv1.Pod, filePath string) string {
		By(fmt.Sprintf("Running qemu-img info on %s", filePath))

		containerName := pod.Spec.Containers[0].Name
		stdout, err := exec.ExecuteCommandOnPod(pod, containerName, []string{"/bin/sh", "-c", fmt.Sprintf("qemu-img info %s/%s", libstorage.DefaultPvcMountPath, filePath)})
		Expect(err).ToNot(HaveOccurred())

		return stdout
	}

	cleanupConsumerPod := func(pod *k8sv1.Pod) {
		By(fmt.Sprintf("Cleaning up consumer pod %s", pod.Name))
		err := virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	checkCBTIntegrity := func(vm *v1.VirtualMachine, cbtOverlayPath string) {
		By("Checking CBT data integrity")
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		pvcName := backendstorage.CurrentPVCName(vmi)
		Expect(pvcName).ToNot(BeEmpty(), "Backend storage PVC name should not be empty")

		vm = libvmops.StopVirtualMachine(vm)

		consumerPod := createConsumerPod(vm, pvcName)
		output := runQemuImgInfo(consumerPod, cbtOverlayPath)

		By("Verifying qcow2 file is not corrupted")
		Expect(output).To(ContainSubstring("corrupt: false"))
		cleanupConsumerPod(consumerPod)
	}

	migrateVMI := func(vmi *v1.VirtualMachineInstance) {
		By("Performing live migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
		libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
	}

	restartVM := func(vm *v1.VirtualMachine) {
		By("Restarting VM")
		libvmops.StopVirtualMachine(vm)
		libvmops.StartVirtualMachine(vm)
	}

	testCBTPersistence := func(op string) {
		sc, foundSC := libstorage.GetRWOBlockStorageClass()
		accessMode := k8sv1.ReadWriteOnce
		volumeMode := k8sv1.PersistentVolumeBlock
		if op == "migrate" {
			sc, foundSC = libstorage.GetRWXFileSystemStorageClass()
			accessMode = k8sv1.ReadWriteMany
			volumeMode = k8sv1.PersistentVolumeFilesystem
		}
		if !foundSC {
			Fail(fmt.Sprintf("Fail test when no %s Block storage is not present", accessMode))
		}
		vm = libstorage.RenderVMWithDataVolumeTemplate(libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
			libdv.WithStorage(
				libdv.StorageWithStorageClass(sc),
				libdv.StorageWithVolumeMode(volumeMode),
				libdv.StorageWithAccessMode(accessMode),
				libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine))),
			),
		),
			libvmi.WithLabels(cbt.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		volumeName := vm.Spec.Template.Spec.Volumes[0].Name

		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("Verifying CBT is enabled")
		Eventually(func() v1.ChangedBlockTrackingState {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return cbt.CBTState(vm.Status.ChangedBlockTracking)
		}, 3*time.Minute, 3*time.Second).Should(Equal(v1.ChangedBlockTrackingEnabled))

		By("Writing data to disk to trigger CBT tracking")
		writeDataToDisk(vmi)

		By("Running the requested operations and ensuring CBT is not curropted")

		switch op {
		case "migrate":
			migrateVMI(vmi)
		case "restart":
			restartVM(vm)
		}
		cbtOverlayPath := fmt.Sprintf("/cbt/%s.qcow2", volumeName)
		checkCBTIntegrity(vm, cbtOverlayPath)
	}

	Context("CBT migration with vmStateStorageClass configuration", func() {
		var originalVMStateStorageClass string

		BeforeEach(func() {
			By("Saving original vmStateStorageClass configuration")
			kv := libkubevirt.GetCurrentKv(virtClient)
			originalVMStateStorageClass = kv.Spec.Configuration.VMStateStorageClass

			By("Patching KubeVirt CR to use RWXFilesystem storage class for vmStateStorageClass")
			rwxFsStorageClass, found := libstorage.GetRWXFileSystemStorageClass()
			if !found {
				Fail("RWXFilesystem storage class not found, skipping test")
			}

			// Get current configuration and update vmStateStorageClass
			config := kv.Spec.Configuration.DeepCopy()
			config.VMStateStorageClass = rwxFsStorageClass
			kvconfig.UpdateKubeVirtConfigValueAndWait(*config)
		})

		AfterEach(func() {
			By("Restoring original vmStateStorageClass configuration")
			kv := libkubevirt.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration.DeepCopy()
			config.VMStateStorageClass = originalVMStateStorageClass
			kvconfig.UpdateKubeVirtConfigValueAndWait(*config)
		})

		// NOTE: Currently there is a bug in libvirt where the qcow2 overlay has to be on a shared storage
		// or the migration fails. Will change the test to run with RWO once the bug is fixed.
		// Bug: https://issues.redhat.com/browse/RHEL-113574
		It("should persist CBT data across live migration", Serial, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.RequiresRWXFsVMStateStorageClass, func() {
			testCBTPersistence("migrate")
		})
	})

	It("should persist CBT data across restart", func() {
		testCBTPersistence("restart")
	})
}))
