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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package storage

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = SIGDescribe("Hotplug", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	newVirtualMachineInstanceWithContainerDisk := func() (*kubevirtv1.VirtualMachineInstance, *cdiv1.DataVolume) {
		vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
		return tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n"), nil
	}

	createVirtualMachine := func(running bool, template *kubevirtv1.VirtualMachineInstance) *kubevirtv1.VirtualMachine {
		By("Creating VirtualMachine")
		vm := tests.NewRandomVirtualMachine(template, running)
		newVM, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
		Expect(err).ToNot(HaveOccurred())
		return newVM
	}

	deleteVirtualMachine := func(vm *kubevirtv1.VirtualMachine) error {
		return virtClient.VirtualMachine(tests.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
	}

	addVolumeVMWithSource := func(vm *kubevirtv1.VirtualMachine, volumeName, bus string, volumeSource *kubevirtv1.HotplugVolumeSource) {
		Eventually(func() error {
			return virtClient.VirtualMachine(vm.Namespace).AddVolume(vm.Name, &kubevirtv1.AddVolumeOptions{
				Name: volumeName,
				Disk: &kubevirtv1.Disk{
					DiskDevice: kubevirtv1.DiskDevice{
						Disk: &kubevirtv1.DiskTarget{
							Bus: bus,
						},
					},
				},
				VolumeSource: volumeSource,
			})
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	addDVVolumeVM := func(vm *kubevirtv1.VirtualMachine, volumeName, claimName, bus string) {
		addVolumeVMWithSource(vm, volumeName, bus, &kubevirtv1.HotplugVolumeSource{
			DataVolume: &kubevirtv1.DataVolumeSource{
				Name: claimName,
			},
		})
	}

	addPVCVolumeVM := func(vm *kubevirtv1.VirtualMachine, volumeName, claimName, bus string) {
		addVolumeVMWithSource(vm, volumeName, bus, &kubevirtv1.HotplugVolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		})
	}

	removeVolumeVM := func(vm *kubevirtv1.VirtualMachine, volumeName string) {
		err = virtClient.VirtualMachine(vm.Namespace).RemoveVolume(vm.Name, &kubevirtv1.RemoveVolumeOptions{
			Name: volumeName,
		})
		Expect(err).ToNot(HaveOccurred())
	}

	verifyVolumeAndDiskVMAdded := func(vm *kubevirtv1.VirtualMachine, volumeNames ...string) {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		log.Log.Infof("Checking %d volumes", len(volumeNames))
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			if len(updatedVM.Status.VolumeRequests) > 0 {
				return fmt.Errorf("waiting on all VolumeRequests to be processed")
			}

			foundVolume := 0
			foundDisk := 0

			for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
				if _, ok := nameMap[volume.Name]; ok {
					foundVolume++
				}
			}
			for _, disk := range updatedVM.Spec.Template.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok {
					foundDisk++
				}
			}

			if foundDisk != len(volumeNames) {
				return fmt.Errorf("waiting on new disk to appear in template")
			}
			if foundVolume != len(volumeNames) {
				return fmt.Errorf("waiting on new volume to appear in template")
			}

			return nil
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyVolumeAndDiskVMIAdded := func(vmi *kubevirtv1.VirtualMachineInstance, volumeNames ...string) {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		Eventually(func() error {
			updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			foundVolume := 0
			foundDisk := 0

			for _, volume := range updatedVMI.Spec.Volumes {
				if _, ok := nameMap[volume.Name]; ok {
					foundVolume++
				}
			}
			for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok {
					foundDisk++
				}
			}

			if foundDisk != len(volumeNames) {
				return fmt.Errorf("waiting on new disk to appear in template")
			}
			if foundVolume != len(volumeNames) {
				return fmt.Errorf("waiting on new volume to appear in template")
			}

			return nil
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyVolumeAndDiskVMRemoved := func(vm *kubevirtv1.VirtualMachine, volumeNames ...string) {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			if len(updatedVM.Status.VolumeRequests) > 0 {
				return fmt.Errorf("waiting on all VolumeRequests to be processed")
			}

			for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
				if _, ok := nameMap[volume.Name]; ok {
					return fmt.Errorf("waiting on volume to be removed")
				}
			}
			for _, disk := range updatedVM.Spec.Template.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok {
					return fmt.Errorf("waiting on disk to be removed")
				}
			}
			return nil
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyVolumeStatus := func(vmi *kubevirtv1.VirtualMachineInstance, phase kubevirtv1.VolumePhase, volumeNames ...string) {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		Eventually(func() error {
			updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			foundVolume := 0
			for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
				if _, ok := nameMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil {
					if volumeStatus.Phase == phase {
						foundVolume++
					}
				}
			}

			if foundVolume != len(volumeNames) {
				return fmt.Errorf("waiting on volume statuses for hotplug disks to be ready")
			}

			return nil
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyCreateData := func(vmi *kubevirtv1.VirtualMachineInstance, device string) {
		batch := []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 %s\n", device)},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: fmt.Sprintf("sudo mount %s %s\n", device, filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo chmod a+w %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("echo '%s' > %s\n", vmi.UID, filepath.Join("/test", filepath.Base(device), "data/message"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("cat %s\n", filepath.Join("/test", filepath.Base(device), "data/message"))},
			&expect.BExp{R: string(vmi.UID)},
			&expect.BSnd{S: "sync\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "sync\n"},
			&expect.BExp{R: console.PromptExpression},
		}
		Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
	}

	getTargetsFromVolumeStatus := func(vmi *kubevirtv1.VirtualMachineInstance, volumeNames ...string) []string {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		res := make([]string, 0)
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
			if _, ok := nameMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil {
				Expect(volumeStatus.Target).ToNot(BeEmpty())
				res = append(res, fmt.Sprintf("/dev/%s", volumeStatus.Target))
			}
		}
		return res
	}

	Context("Offline VM", func() {
		var (
			vm *kubevirtv1.VirtualMachine
		)
		BeforeEach(func() {
			By("Creating VirtualMachine")
			template, _ := newVirtualMachineInstanceWithContainerDisk()
			vm = createVirtualMachine(false, template)
		})

		AfterEach(func() {
			err := deleteVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should add a volume on an offline VM", func() {
			By("Adding test volumes")
			addPVCVolumeVM(vm, "some-new-volume1", "madeup", "scsi")
			addPVCVolumeVM(vm, "some-new-volume2", "madeup", "scsi")
			By("Verifying the volumes have been added to the template spec")
			verifyVolumeAndDiskVMAdded(vm, "some-new-volume1", "some-new-volume2")
			By("Removing new volumes from VM")
			removeVolumeVM(vm, "some-new-volume1")
			removeVolumeVM(vm, "some-new-volume2")

			verifyVolumeAndDiskVMRemoved(vm, "some-new-volume1", "some-new-volume2")
		})
	})

	Context("Online VM", func() {
		var (
			vm *kubevirtv1.VirtualMachine
			dv *cdiv1.DataVolume
		)

		BeforeEach(func() {
			template := tests.NewRandomFedoraVMIWitGuestAgent()
			vm = createVirtualMachine(true, template)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			By("Creating DataVolume")
			dv = tests.NewRandomBlankDataVolume(tests.NamespaceTestDefault, corev1.ReadWriteOnce)
			_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulDataVolumeImport(dv, 240)
		})

		AfterEach(func() {
			By("Deleting the virtual machine")
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			By("Waiting for VMI to be removed")
			Eventually(func() int {
				vmis, err := virtClient.VirtualMachineInstance(vm.Namespace).List(&metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vmis.Items)
			}, 300*time.Second, 2*time.Second).Should(BeZero(), "The VirtualMachineInstance did not disappear")

			By("Deleting the DataVolume")
			ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
		})

		table.DescribeTable("should add/remove volume", func(volumeName, bus string, addVolumeFunc func(vm *kubevirtv1.VirtualMachine, volumeName, claimName, bus string), waitToStart bool) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if waitToStart {
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			}
			By("Adding volume to running VM")
			addVolumeFunc(vm, "testvolume", dv.Name, "scsi")
			By("Verifying the volume and disk are in the VM and VMI")
			verifyVolumeAndDiskVMAdded(vm, "testvolume")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(vmi, "testvolume")
			By("Verify the volume status of the hotplugged volume is ready")
			verifyVolumeStatus(vmi, kubevirtv1.VolumeReady, "testvolume")
			By("Obtaining the serial console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			Eventually(func() error {
				return console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", targets[0])},
					&expect.BExp{R: targets[0]},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 10)
			}, 40*time.Second, 2*time.Second).Should(Succeed())
			verifyCreateData(vmi, targets[0])
			By("removing volume from VM")
			removeVolumeVM(vm, "testvolume")
			By("Verifying the volume no longer exists in VM")
			verifyVolumeAndDiskVMRemoved(vm, "testvolume")
			Eventually(func() error {
				return console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", targets[0])},
					&expect.BExp{R: fmt.Sprintf("ls: cannot access '%s'", targets[0])},
				}, 10)
			}, 40*time.Second, 2*time.Second).Should(Succeed())
		},
			table.Entry("with DataVolume immediate attach", "testvolume", "scsi", addDVVolumeVM, false),
			table.Entry("with PersistentVolume immediate attach", "testvolume", "scsi", addPVCVolumeVM, false),
			table.Entry("with DataVolume wait for VM to finish starting", "testvolume", "scsi", addDVVolumeVM, true),
			table.Entry("with PersistentVolume wait for VM to finish starting", "testvolume", "scsi", addPVCVolumeVM, true),
		)

		It("Should be able to add and remove multiple volumes", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// By("Obtaining the serial console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			testVolumes := make([]string, 0)
			dvNames := make([]string, 0)
			for i := 0; i < 5; i++ {
				volumeName := fmt.Sprintf("volume%d", i)
				By("Creating DataVolume")
				dv := tests.NewRandomBlankDataVolume(tests.NamespaceTestDefault, corev1.ReadWriteOnce)
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulDataVolumeImport(dv, 240)

				By("Adding volume to running VM")
				addDVVolumeVM(vm, volumeName, dv.Name, "scsi")
				testVolumes = append(testVolumes, volumeName)
				dvNames = append(dvNames, dv.Name)
			}
			defer func(dvNames []string, namespace string) {
				for _, dvName := range dvNames {
					By("Deleting the DataVolume")
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Delete(dvName, &metav1.DeleteOptions{})).To(Succeed())
				}
			}(dvNames, vmi.Namespace)
			By("Verifying the volume and disk are in the VM and VMI")
			verifyVolumeAndDiskVMAdded(vm, testVolumes...)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(vmi, testVolumes...)
			By("Verify the volume status of the hotplugged volume is ready")
			verifyVolumeStatus(vmi, kubevirtv1.VolumeReady, testVolumes...)
			targets := getTargetsFromVolumeStatus(vmi, testVolumes...)
			for i := range testVolumes {
				Eventually(func() error {
					return console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", targets[i])},
						&expect.BExp{R: targets[i]},
						&expect.BSnd{S: "echo $?\n"},
						&expect.BExp{R: console.RetValue("0")},
					}, 10)
				}, 40*time.Second, 2*time.Second).Should(Succeed())
			}
			for _, target := range targets {
				verifyCreateData(vmi, target)
			}
			for i, volumeName := range testVolumes {
				By("removing volume from VM")
				removeVolumeVM(vm, volumeName)
				By("Verifying the volume no longer exists in VM")
				verifyVolumeAndDiskVMRemoved(vm, volumeName)
				Eventually(func() error {
					return console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", targets[i])},
						&expect.BExp{R: fmt.Sprintf("ls: cannot access '%s'", targets[i])},
					}, 10)
				}, 40*time.Second, 2*time.Second).Should(Succeed())
			}
		})

		It("Should be able to add and remove and re-add multiple volumes", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			testVolumes := make([]string, 0)
			dvNames := make([]string, 0)
			for i := 0; i < 5; i++ {
				volumeName := fmt.Sprintf("volume%d", i)
				By("Creating DataVolume")
				dv := tests.NewRandomBlankDataVolume(tests.NamespaceTestDefault, corev1.ReadWriteOnce)
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulDataVolumeImport(dv, 240)
				testVolumes = append(testVolumes, volumeName)
				dvNames = append(dvNames, dv.Name)
			}
			defer func(dvNames []string, namespace string) {
				for _, dvName := range dvNames {
					By("Deleting the DataVolume")
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Delete(dvName, &metav1.DeleteOptions{})).To(Succeed())
				}
			}(dvNames, vmi.Namespace)

			for i := 0; i < 4; i++ {
				By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
				addDVVolumeVM(vm, testVolumes[i], dvNames[i], "scsi")
			}

			By("Verifying the volume and disk are in the VM and VMI")
			verifyVolumeAndDiskVMAdded(vm, testVolumes[:len(testVolumes)-1]...)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(vmi, testVolumes[:len(testVolumes)-1]...)
			By("Verify the volume status of the hotplugged volume is ready")
			verifyVolumeStatus(vmi, kubevirtv1.VolumeReady, testVolumes[:len(testVolumes)-1]...)

			By("removing volume sdc, with dv" + dvNames[2])
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.VolumeStatus[4].Target
			}, 40*time.Second, 2*time.Second).Should(Equal("sdc"))
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.VolumeStatus[5].Target
			}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

			removeVolumeVM(vm, testVolumes[2])
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.VolumeStatus[4].Target
			}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

			By("Adding remaining volume, it should end up in the spot that was just cleared")
			addDVVolumeVM(vm, testVolumes[4], dvNames[4], "scsi")
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volumeStatus := range vmi.Status.VolumeStatus {
					if volumeStatus.Name == testVolumes[4] {
						return volumeStatus.Target
					}
				}
				return ""
			}, 40*time.Second, 2*time.Second).Should(Equal("sdc"))
			By("Adding intermediate volume, it should end up at the end")
			addDVVolumeVM(vm, testVolumes[2], dvNames[2], "scsi")
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volumeStatus := range vmi.Status.VolumeStatus {
					if volumeStatus.Name == testVolumes[2] {
						return volumeStatus.Target
					}
				}
				return ""
			}, 40*time.Second, 2*time.Second).Should(Equal("sde"))

			for _, volumeName := range testVolumes {
				By("removing volume from VM")
				removeVolumeVM(vm, volumeName)
				By("Verifying the volume no longer exists in VM")
				verifyVolumeAndDiskVMRemoved(vm, volumeName)
			}
		})
	})
})
