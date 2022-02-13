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
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"kubevirt.io/client-go/log"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"

	virtctl "kubevirt.io/kubevirt/pkg/virtctl/vm"
)

const (
	dataMessage             = "data/message"
	addingVolumeRunningVM   = "Adding volume to running VM"
	verifyingVolumeDiskInVM = "Verifying the volume and disk are in the VM and VMI"
	removingVolumeFromVM    = "removing volume from VM"
	verifyingVolumeNotExist = "Verifying the volume no longer exists in VM"

	virtCtlNamespace       = "--namespace"
	virtCtlVolumeName      = "--volume-name=%s"
	verifyCannotAccessDisk = "ls: %s: No such file or directory"

	waitVolumeRequestProcessError = "waiting on all VolumeRequests to be processed"

	testNewVolume1 = "some-new-volume1"
	testNewVolume2 = "some-new-volume2"
)

type addVolumeFunction func(name, namespace, volumeName, claimName, bus string, dryRun bool)
type removeVolumeFunction func(name, namespace, volumeName string, dryRun bool)

var _ = SIGDescribe("Hotplug", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	getDryRunOption := func(dryRun bool) []string {
		if dryRun {
			return []string{metav1.DryRunAll}
		}
		return nil
	}

	newVirtualMachineInstanceWithContainerDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
		vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
		return tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n"), nil
	}

	createVirtualMachine := func(running bool, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
		By("Creating VirtualMachine")
		vm := tests.NewRandomVirtualMachine(template, running)
		newVM, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
		Expect(err).ToNot(HaveOccurred())
		return newVM
	}

	deleteVirtualMachine := func(vm *v1.VirtualMachine) error {
		return virtClient.VirtualMachine(util.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
	}

	getAddVolumeOptions := func(volumeName, bus string, volumeSource *v1.HotplugVolumeSource, dryRun bool) *v1.AddVolumeOptions {
		return &v1.AddVolumeOptions{
			Name: volumeName,
			Disk: &v1.Disk{
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: bus,
					},
				},
				Serial: volumeName,
			},
			VolumeSource: volumeSource,
			DryRun:       getDryRunOption(dryRun),
		}
	}
	addVolumeVMIWithSource := func(name, namespace string, volumeOptions *v1.AddVolumeOptions) {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(namespace).AddVolume(name, volumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	addDVVolumeVMI := func(name, namespace, volumeName, claimName, bus string, dryRun bool) {
		addVolumeVMIWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: claimName,
			},
		}, dryRun))
	}

	addPVCVolumeVMI := func(name, namespace, volumeName, claimName, bus string, dryRun bool) {
		addVolumeVMIWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun))
	}

	addVolumeVMWithSource := func(name, namespace string, volumeOptions *v1.AddVolumeOptions) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).AddVolume(name, volumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	addDVVolumeVM := func(name, namespace, volumeName, claimName, bus string, dryRun bool) {
		addVolumeVMWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: claimName,
			},
		}, dryRun))
	}

	addPVCVolumeVM := func(name, namespace, volumeName, claimName, bus string, dryRun bool) {
		addVolumeVMWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun))
	}

	addVolumeVirtctl := func(name, namespace, volumeName, claimName, bus string, dryRun bool) {
		By("Invoking virtlctl addvolume")
		commandAndArgs := []string{virtctl.COMMAND_ADDVOLUME, name, fmt.Sprintf(virtCtlVolumeName, claimName), virtCtlNamespace, namespace}
		if dryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		addvolumeCommand := tests.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return addvolumeCommand()
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeVolumeVMI := func(name, namespace, volumeName string, dryRun bool) {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(namespace).RemoveVolume(name, &v1.RemoveVolumeOptions{
				Name:   volumeName,
				DryRun: getDryRunOption(dryRun),
			})
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeVolumeVM := func(name, namespace, volumeName string, dryRun bool) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).RemoveVolume(name, &v1.RemoveVolumeOptions{
				Name:   volumeName,
				DryRun: getDryRunOption(dryRun),
			})
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeVolumeVirtctl := func(name, namespace, volumeName string, dryRun bool) {
		By("Invoking virtlctl removevolume")
		commandAndArgs := []string{virtctl.COMMAND_REMOVEVOLUME, name, fmt.Sprintf(virtCtlVolumeName, volumeName), virtCtlNamespace, namespace}
		if dryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		removeVolumeCommand := tests.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return removeVolumeCommand()
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyVolumeAndDiskVMRemoved := func(vm *v1.VirtualMachine, volumeNames ...string) {
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
				return fmt.Errorf(waitVolumeRequestProcessError)
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
		}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyVolumeStatus := func(vmi *v1.VirtualMachineInstance, phase v1.VolumePhase, volumeNames ...string) {
		By("Verify the volume status of the hotplugged volume is ready")
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
				log.Log.Infof("Volume Status, name: %s, target [%s], phase:%s, reason: %s", volumeStatus.Name, volumeStatus.Target, volumeStatus.Phase, volumeStatus.Reason)
				if _, ok := nameMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil && volumeStatus.Target != "" {
					if volumeStatus.Phase == phase {
						foundVolume++
					}
				}
			}

			if foundVolume != len(volumeNames) {
				return fmt.Errorf("waiting on volume statuses for hotplug disks to be ready")
			}

			return nil
		}, 360*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyNoVolumeAttached := func(vmi *v1.VirtualMachineInstance, volumeNames ...string) {
		By("Verify that the number of attached volumes does not change")
		volumeNamesMap := make(map[string]struct{}, len(volumeNames))
		for _, volumeName := range volumeNames {
			volumeNamesMap[volumeName] = struct{}{}
		}
		Consistently(func() error {
			currentVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}
			foundVolume := 0
			for _, volumeStatus := range currentVMI.Status.VolumeStatus {
				if _, ok := volumeNamesMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil && volumeStatus.Target != "" {
					if volumeStatus.Phase == v1.VolumeReady {
						foundVolume++
					}
				}
			}
			if foundVolume != 0 {
				return fmt.Errorf("a volume has been attached")
			}
			return nil
		}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	verifyCreateData := func(vmi *v1.VirtualMachineInstance, device string) {
		batch := []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 %s\n", device)},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: fmt.Sprintf("sudo mount %s %s\n", device, filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo chmod a+w %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("echo '%s' > %s\n", vmi.UID, filepath.Join("/test", filepath.Base(device), dataMessage))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("cat %s\n", filepath.Join("/test", filepath.Base(device), dataMessage))},
			&expect.BExp{R: string(vmi.UID)},
			&expect.BSnd{S: syncName},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: syncName},
			&expect.BExp{R: console.PromptExpression},
		}
		Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
	}

	verifyWriteReadData := func(vmi *v1.VirtualMachineInstance, device string) {
		dataFile := filepath.Join("/test", filepath.Base(device), dataMessage)
		batch := []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("echo '%s' > %s\n", vmi.UID, dataFile)},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: tests.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("cat %s\n", dataFile)},
			&expect.BExp{R: string(vmi.UID)},
			&expect.BSnd{S: syncName},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: syncName},
			&expect.BExp{R: console.PromptExpression},
		}
		Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
	}

	verifyVolumeAccessible := func(vmi *v1.VirtualMachineInstance, volumeName string) {
		Eventually(func() error {
			return console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", volumeName)},
				&expect.BExp{R: volumeName},
				&expect.BSnd{S: tests.EchoLastReturnValue},
				&expect.BExp{R: console.RetValue("0")},
			}, 10)
		}, 40*time.Second, 2*time.Second).Should(Succeed())
	}

	verifyVolumeNolongerAccessible := func(vmi *v1.VirtualMachineInstance, volumeName string) {
		Eventually(func() error {
			return console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("sudo ls %s\n", volumeName)},
				&expect.BExp{R: fmt.Sprintf(verifyCannotAccessDisk, volumeName)},
			}, 5)
		}, 90*time.Second, 2*time.Second).Should(Succeed())
	}

	waitForAttachmentPodToRun := func(vmi *v1.VirtualMachineInstance) {
		namespace := vmi.GetNamespace()
		uid := vmi.GetUID()

		labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

		pods, err := virtClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
		Expect(err).ToNot(HaveOccurred(), "Should list pods")

		var virtlauncher *corev1.Pod
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				virtlauncher = &pod
				break
			}
		}
		Expect(virtlauncher).ToNot(BeNil(), "Should find running virtlauncher pod")
		Eventually(func() bool {
			podList, err := virtClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, pod := range podList.Items {
				for _, owner := range pod.OwnerReferences {
					if owner.UID == virtlauncher.UID {
						By(fmt.Sprintf("phase: %s", pod.Status.Phase))
						return pod.Status.Phase == corev1.PodRunning
					}
				}
			}
			return false
		}, 270*time.Second, 2*time.Second).Should(BeTrue())
	}

	getTargetsFromVolumeStatus := func(vmi *v1.VirtualMachineInstance, volumeNames ...string) []string {
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

	createAndStartWFFCStorageHotplugVM := func() *v1.VirtualMachine {
		template := libvmi.NewCirros()
		vm := createVirtualMachine(true, template)
		Eventually(func() bool {
			vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vm.Status.Ready
		}, 300*time.Second, 1*time.Second).Should(BeTrue())
		return vm
	}

	checkNoProvisionerStorageClassPVs := func(storageClassName string) {
		sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		if sc.Provisioner != "" && sc.Provisioner != "kubernetes.io/no-provisioner" {
			return
		}

		// Verify we have at least 3 available file system PVs
		pvList, err := virtClient.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		count := 0
		for _, pv := range pvList.Items {
			if pv.Spec.StorageClassName != storageClassName || pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil || len(pv.Spec.NodeAffinity.Required.NodeSelectorTerms) == 0 || (pv.Spec.VolumeMode != nil && *pv.Spec.VolumeMode == corev1.PersistentVolumeBlock) {
				// Not a local volume filesystem PV
				continue
			}
			if pv.Spec.ClaimRef == nil {
				count++
			}
		}
		if count < 3 {
			Skip("Not enough available filesystem local storage PVs available")
		}
	}

	verifyHotplugAttachedAndUseable := func(vmi *v1.VirtualMachineInstance, names []string) []string {
		targets := getTargetsFromVolumeStatus(vmi, names...)
		for _, target := range targets {
			verifyVolumeAccessible(vmi, target)
			verifyCreateData(vmi, target)
		}
		return targets
	}

	verifySingleAttachmentPod := func(vmi *v1.VirtualMachineInstance) {
		podList, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		attachmentPodCount := 0
		for _, pod := range podList.Items {
			for _, ownerRef := range pod.GetOwnerReferences() {
				if ownerRef.UID == vmi.GetUID() {
					attachmentPodCount++
				}
			}
		}
		Expect(attachmentPodCount).To(Equal(1), "Number of attachment pods is not 1: %s", attachmentPodCount)
	}

	getVmiConsoleAndLogin := func(vmi *v1.VirtualMachineInstance) {
		By("Obtaining the serial console")
		Expect(console.LoginToCirros(vmi)).To(Succeed())
	}

	createDataVolumeAndWaitForImport := func(sc string, volumeMode corev1.PersistentVolumeMode) *cdiv1.DataVolume {
		accessMode := corev1.ReadWriteOnce
		if volumeMode == corev1.PersistentVolumeBlock {
			accessMode = corev1.ReadWriteMany
		}
		By("Creating DataVolume")
		dvBlock := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, sc, "64Mi", accessMode, volumeMode)
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dvBlock.Namespace).Create(context.Background(), dvBlock, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		Eventually(ThisDV(dvBlock), 240).Should(HaveSucceeded())
		return dvBlock
	}

	Context("Offline VM", func() {
		var (
			vm *v1.VirtualMachine
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

		DescribeTable("Should add volumes on an offline VM", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction) {
			By("Adding test volumes")
			addVolumeFunc(vm.Name, vm.Namespace, testNewVolume1, "madeup", "scsi", false)
			addVolumeFunc(vm.Name, vm.Namespace, testNewVolume2, "madeup", "scsi", false)
			By("Verifying the volumes have been added to the template spec")
			tests.VerifyVolumeAndDiskVMAdded(virtClient, vm, testNewVolume1, testNewVolume2)
			By("Removing new volumes from VM")
			removeVolumeFunc(vm.Name, vm.Namespace, testNewVolume1, false)
			removeVolumeFunc(vm.Name, vm.Namespace, testNewVolume2, false)

			verifyVolumeAndDiskVMRemoved(vm, testNewVolume1, testNewVolume2)
		},
			Entry("with DataVolume", addDVVolumeVM, removeVolumeVM),
			Entry("with PersistentVolume", addPVCVolumeVM, removeVolumeVM),
		)
	})

	Context("WFFC storage", func() {
		var (
			vm *v1.VirtualMachine
			sc string
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = tests.GetRWOFileSystemStorageClass()
			if !exists || !tests.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Skip("Skip no wffc storage class available")
			}
			checkNoProvisionerStorageClassPVs(sc)

			vm = createAndStartWFFCStorageHotplugVM()
		})

		DescribeTable("Should be able to add and use WFFC local storage", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction) {
			tests.SkipIfNonRoot(virtClient, "root owned disk.img")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			dvNames := make([]string, 0)
			for i := 0; i < 3; i++ {
				dv := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, sc, "64Mi", corev1.ReadWriteOnce, corev1.PersistentVolumeFilesystem)
				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.TODO(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())
				dvNames = append(dvNames, dv.Name)
			}

			for i := 0; i < 3; i++ {
				By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
				addVolumeFunc(vm.Name, vm.Namespace, dvNames[i], dvNames[i], "scsi", false)
			}

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, dvNames...)
			verifyVolumeStatus(vmi, v1.VolumeReady, dvNames...)
			getVmiConsoleAndLogin(vmi)
			verifyHotplugAttachedAndUseable(vmi, dvNames)
			verifySingleAttachmentPod(vmi)
			for _, volumeName := range dvNames {
				By("removing volume " + volumeName + " from VM")
				removeVolumeFunc(vm.Name, vm.Namespace, volumeName, false)
			}
			for _, volumeName := range dvNames {
				verifyVolumeNolongerAccessible(vmi, volumeName)
			}
		},
			Entry("calling endpoints directly", addDVVolumeVMI, removeVolumeVMI),
			Entry("using virtctl", addVolumeVirtctl, removeVolumeVirtctl),
		)
	})

	Context("[storage-req]", func() {
		Context("Online VM", func() {
			var (
				vm *v1.VirtualMachine
				sc string
			)

			findCPUManagerWorkerNode := func() string {
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
					LabelSelector: "node-role.kubernetes.io/worker",
				})
				Expect(err).ToNot(HaveOccurred())
				for _, node := range nodes.Items {
					nodeLabels := node.GetLabels()

					for label, val := range nodeLabels {
						if label == v1.CPUManager && val == "true" {
							return node.Name
						}
					}
				}
				return ""
			}

			BeforeEach(func() {
				exists := false
				sc, exists = tests.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when RWXBlock storage class is not present")
				}

				template := libvmi.NewCirros()
				node := findCPUManagerWorkerNode()
				if node != "" {
					template.Spec.NodeSelector = make(map[string]string)
					template.Spec.NodeSelector[corev1.LabelHostname] = node
				}
				vm = createVirtualMachine(true, template)
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Ready
				}, 300*time.Second, 1*time.Second).Should(BeTrue())
			})

			DescribeTable("should add/remove volume", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly, waitToStart bool) {
				dv := createDataVolumeAndWaitForImport(sc, volumeMode)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if waitToStart {
					tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
				}
				By(addingVolumeRunningVM)
				addVolumeFunc(vm.Name, vm.Namespace, "testvolume", dv.Name, "scsi", false)
				By(verifyingVolumeDiskInVM)
				if !vmiOnly {
					tests.VerifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
				}
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUseable(vmi, []string{"testvolume"})
				verifySingleAttachmentPod(vmi)
				By(removingVolumeFromVM)
				removeVolumeFunc(vm.Name, vm.Namespace, "testvolume", false)
				if !vmiOnly {
					By(verifyingVolumeNotExist)
					verifyVolumeAndDiskVMRemoved(vm, "testvolume")
				}
				verifyVolumeNolongerAccessible(vmi, targets[0])
			},
				Entry("with DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false, false),
				Entry("with PersistentVolume immediate attach", addPVCVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false, false),
				Entry("with DataVolume wait for VM to finish starting", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false, true),
				Entry("with PersistentVolume wait for VM to finish starting", addPVCVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false, true),
				Entry("with DataVolume immediate attach, VMI directly", addDVVolumeVMI, removeVolumeVMI, corev1.PersistentVolumeFilesystem, true, false),
				Entry("with PersistentVolume immediate attach, VMI directly", addPVCVolumeVMI, removeVolumeVMI, corev1.PersistentVolumeFilesystem, true, false),
				Entry("with Block DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeBlock, false, false),
			)

			DescribeTable("Should be able to add and remove multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly bool) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				getVmiConsoleAndLogin(vmi)
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
				testVolumes := make([]string, 0)
				for i := 0; i < 5; i++ {
					volumeName := fmt.Sprintf("volume%d", i)
					dv := createDataVolumeAndWaitForImport(sc, volumeMode)
					By(addingVolumeRunningVM)
					addVolumeFunc(vm.Name, vm.Namespace, volumeName, dv.Name, "scsi", false)
					testVolumes = append(testVolumes, volumeName)
					verifyVolumeStatus(vmi, v1.VolumeReady, testVolumes...)
				}
				By(verifyingVolumeDiskInVM)
				if !vmiOnly {
					tests.VerifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes...)
				}
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes...)
				verifyVolumeStatus(vmi, v1.VolumeReady, testVolumes...)
				targets := verifyHotplugAttachedAndUseable(vmi, testVolumes)
				verifySingleAttachmentPod(vmi)
				for _, volumeName := range testVolumes {
					By("removing volume " + volumeName + " from VM")
					removeVolumeFunc(vm.Name, vm.Namespace, volumeName, false)
					if !vmiOnly {
						By(verifyingVolumeNotExist)
						verifyVolumeAndDiskVMRemoved(vm, volumeName)
					}
				}
				for i := range testVolumes {
					verifyVolumeNolongerAccessible(vmi, targets[i])
				}
			},
				Entry("with VMs", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false),
				Entry("with VMIs", addDVVolumeVMI, removeVolumeVMI, corev1.PersistentVolumeFilesystem, true),
				Entry("with VMs and block", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeBlock, false),
			)

			DescribeTable("Should be able to add and remove and re-add multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly bool) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
				testVolumes := make([]string, 0)
				dvNames := make([]string, 0)
				for i := 0; i < 5; i++ {
					volumeName := fmt.Sprintf("volume%d", i)
					dv := createDataVolumeAndWaitForImport(sc, volumeMode)
					testVolumes = append(testVolumes, volumeName)
					dvNames = append(dvNames, dv.Name)
				}

				for i := 0; i < 4; i++ {
					By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
					addVolumeFunc(vm.Name, vm.Namespace, testVolumes[i], dvNames[i], "scsi", false)
				}

				By(verifyingVolumeDiskInVM)
				if !vmiOnly {
					tests.VerifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes[:len(testVolumes)-1]...)
				}
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes[:len(testVolumes)-1]...)
				waitForAttachmentPodToRun(vmi)
				verifyVolumeStatus(vmi, v1.VolumeReady, testVolumes[:len(testVolumes)-1]...)
				verifySingleAttachmentPod(vmi)
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

				removeVolumeFunc(vm.Name, vm.Namespace, testVolumes[2], false)
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.VolumeStatus[4].Target
				}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

				By("Adding remaining volume, it should end up in the spot that was just cleared")
				addVolumeFunc(vm.Name, vm.Namespace, testVolumes[4], dvNames[4], "scsi", false)
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
				addVolumeFunc(vm.Name, vm.Namespace, testVolumes[2], dvNames[2], "scsi", false)
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
				verifySingleAttachmentPod(vmi)
				for _, volumeName := range testVolumes {
					By(removingVolumeFromVM)
					removeVolumeFunc(vm.Name, vm.Namespace, volumeName, false)
					if !vmiOnly {
						By(verifyingVolumeNotExist)
						verifyVolumeAndDiskVMRemoved(vm, volumeName)
					}
				}
			},
				Entry("with VMs", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false),
				Entry("with VMIs", addDVVolumeVMI, removeVolumeVMI, corev1.PersistentVolumeFilesystem, true),
				Entry("[Serial] with VMs and block", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeBlock, false),
			)

			It("should permanently add hotplug volume when added to VM, but still unpluggable after restart", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)

				By(addingVolumeRunningVM)
				addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dvBlock.Name, "scsi", false)
				By(verifyingVolumeDiskInVM)
				tests.VerifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")
				verifySingleAttachmentPod(vmi)

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUseable(vmi, []string{"testvolume"})
				Expect(len(targets)).To(Equal(1))

				By("stopping VM")
				vm = tests.StopVirtualMachine(vm)

				By("starting VM")
				vm = tests.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume is hotpluggable after a restart")
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")

				By("Verifying the hotplug device is auto-mounted during booting")
				getVmiConsoleAndLogin(vmi)
				verifyVolumeAccessible(vmi, targets[0])

				By("Remove volume from a running VM")
				removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume can be unplugged after a restart")
				verifyVolumeNolongerAccessible(vmi, targets[0])
			})

			It("should reject hotplugging a volume with the same name as an existing volume", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)

				By(addingVolumeRunningVM)
				err = virtClient.VirtualMachine(vm.Namespace).AddVolume(vm.Name, getAddVolumeOptions("disk0", "scsi", &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvBlock.Name,
					},
				}, false))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("conflicts with an existing volume of the same name on the vmi template"))
			})

			It("should allow hotplugging both a filesystem and block volume", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)
				dvFileSystem := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeFilesystem)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
				getVmiConsoleAndLogin(vmi)

				By(addingVolumeRunningVM)
				addDVVolumeVM(vm.Name, vm.Namespace, "block", dvBlock.Name, "scsi", false)
				addDVVolumeVM(vm.Name, vm.Namespace, "fs", dvFileSystem.Name, "scsi", false)
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "block", "fs")

				verifyVolumeStatus(vmi, v1.VolumeReady, "block", "fs")
				targets := getTargetsFromVolumeStatus(vmi, "block", "fs")
				for i := 0; i < 2; i++ {
					verifyVolumeAccessible(vmi, targets[i])
				}
				verifySingleAttachmentPod(vmi)
				removeVolumeVMI(vmi.Name, vmi.Namespace, "block", false)
				removeVolumeVMI(vmi.Name, vmi.Namespace, "fs", false)

				for i := 0; i < 2; i++ {
					verifyVolumeNolongerAccessible(vmi, targets[i])
				}
			})
		})

		Context("VMI migration", func() {
			var (
				vmi *v1.VirtualMachineInstance
				sc  string

				numberOfMigrations int
				sourceNode         string
				targetNode         string
			)

			const (
				hotplugLabelKey   = "kubevirt-test-migration-with-hotplug-disks"
				hotplugLabelValue = "true"
			)

			verifyIsMigratable := func(vmi *v1.VirtualMachineInstance, expectedValue bool) {
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					if err != nil {
						return false
					}
					for _, condition := range vmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceIsMigratable {
							return condition.Status == corev1.ConditionTrue
						}
					}
					return vmi.Status.Phase == v1.Failed
				}, 90*time.Second, 1*time.Second).Should(Equal(expectedValue))
			}

			BeforeEach(func() {
				exists := false
				sc, exists = tests.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when RWXBlock storage class is not present")
				}

				// Workaround for the issue with CPU manager and runc prior to version v1.0.0:
				// CPU manager periodically updates cgroup settings via the container runtime
				// interface. Runc prior to version v1.0.0 drops all 'custom' cgroup device
				// rules on 'update' and that causes a race with live migration when block volumes
				// are hotplugged. Try to setup the test in a way so that the VMI is migrated to
				// a node without CPU manager.
				sourceNode = ""
				targetNode = ""
				for _, node := range util.GetAllSchedulableNodes(virtClient).Items {
					labels := node.GetLabels()
					if val, ok := labels[v1.CPUManager]; ok && val == "true" {
						// Use a node with CPU manager as migration source
						sourceNode = node.Name
					} else {
						// Use a node without CPU manager as migration target
						targetNode = node.Name
					}
				}
				if sourceNode == "" || targetNode == "" {
					Skip("Two schedulable nodes are required for migration tests")
				} else {
					numberOfMigrations = 1
				}
				// Ensure the virt-launcher pod is scheduled on the chosen source node and then
				// migrated to the proper target.
				tests.AddLabelToNode(sourceNode, hotplugLabelKey, hotplugLabelValue)
				vmi, _ = newVirtualMachineInstanceWithContainerDisk()
				vmi.Spec.NodeSelector = map[string]string{hotplugLabelKey: hotplugLabelValue}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
				tests.AddLabelToNode(targetNode, hotplugLabelKey, hotplugLabelValue)
			})

			AfterEach(func() {
				// Cleanup node labels
				tests.RemoveLabelFromNode(sourceNode, hotplugLabelKey)
				tests.RemoveLabelFromNode(targetNode, hotplugLabelKey)
			})

			It("should allow live migration with attached hotplug volumes", func() {
				volumeName := "testvolume"
				volumeMode := corev1.PersistentVolumeBlock
				addVolumeFunc := addDVVolumeVMI
				removeVolumeFunc := removeVolumeVMI
				dv := createDataVolumeAndWaitForImport(sc, volumeMode)

				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
				By("Verifying the VMI is migrateable")
				verifyIsMigratable(vmi, true)

				By("Adding volume to running VMI")
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, "scsi", false)
				By("Verifying the volume and disk are in the VMI")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, volumeName)

				By("Verifying the VMI is still migrateable")
				verifyIsMigratable(vmi, true)

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUseable(vmi, []string{volumeName})
				Expect(len(targets) == 1).To(BeTrue())

				By("Starting the migration multiple times")
				for i := 0; i < numberOfMigrations; i++ {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					sourceAttachmentPods := []string{}
					for _, volumeStatus := range vmi.Status.VolumeStatus {
						if volumeStatus.HotplugVolume != nil {
							sourceAttachmentPods = append(sourceAttachmentPods, volumeStatus.HotplugVolume.AttachPodName)
						}
					}
					Expect(len(sourceAttachmentPods) == 1).To(BeTrue())

					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
					By("Verifying the volume is still accessible and usable")
					verifyVolumeAccessible(vmi, targets[0])
					verifyWriteReadData(vmi, targets[0])

					By("Verifying the source attachment pods are deleted")
					Eventually(func() bool {
						_, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), sourceAttachmentPods[0], metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 60*time.Second, 1*time.Second).Should(BeTrue())
				}

				By("Verifying the volume can be detached and reattached after migration")
				removeVolumeFunc(vmi.Name, vmi.Namespace, volumeName, false)
				verifyVolumeNolongerAccessible(vmi, targets[0])
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, "scsi", false)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, volumeName)
			})
		})
	})

	Context("hostpath", func() {
		var (
			vm *v1.VirtualMachine
		)

		const (
			hotplugPvPath = "/mnt/local-storage/hotplug-test"
		)

		storageClassHostPath := "host-path"
		immediateBinding := storagev1.VolumeBindingImmediate

		BeforeEach(func() {
			tests.CreateStorageClass(storageClassHostPath, &immediateBinding)
			pvNode := tests.CreateHostPathPvWithSizeAndStorageClass(tests.CustomHostPath, hotplugPvPath, "1Gi", storageClassHostPath)
			tests.CreatePVC(tests.CustomHostPath, "1Gi", storageClassHostPath, false)
			template := libvmi.NewCirros()
			if pvNode != "" {
				template.Spec.NodeSelector = make(map[string]string)
				template.Spec.NodeSelector[corev1.LabelHostname] = pvNode
			}
			vm = createVirtualMachine(true, template)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		}, 120)

		AfterEach(func() {
			tests.DeletePvAndPvc(fmt.Sprintf("%s-disk-for-tests", tests.CustomHostPath))
			tests.DeleteStorageClass(storageClassHostPath)
		})

		It("should attach a hostpath based volume to running VM", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)

			By(addingVolumeRunningVM)
			name := fmt.Sprintf("disk-%s", tests.CustomHostPath)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", name, "scsi", false)

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVMI(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	Context("iothreads", func() {
		var (
			vm *v1.VirtualMachine
		)

		BeforeEach(func() {
			template := libvmi.NewCirros()
			policy := v1.IOThreadsPolicyShared
			template.Spec.Domain.IOThreadsPolicy = &policy
			vm = createVirtualMachine(true, template)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		}, 120)

		It("should allow adding and removing hotplugged volumes", func() {
			sc, exists := tests.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip no filesystem storage class available")
			}
			dv := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, sc, "64Mi", corev1.ReadWriteOnce, corev1.PersistentVolumeFilesystem)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)

			By(addingVolumeRunningVM)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", dv.Name, "scsi", false)

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVMI(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	Context("hostpath-separate-device", func() {
		var (
			vm *v1.VirtualMachine
		)

		BeforeEach(func() {
			tests.CreateAllSeparateDeviceHostPathPvs(tests.CustomHostPath)
			vm = createVirtualMachine(true, libvmi.NewCirros())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		}, 120)

		AfterEach(func() {
			tests.DeleteAllSeparateDeviceHostPathPvs()
		})

		It("should attach a hostpath based volume to running VM", func() {
			dv := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, tests.StorageClassHostPathSeparateDevice, "64Mi", corev1.ReadWriteOnce, corev1.PersistentVolumeFilesystem)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)

			By(addingVolumeRunningVM)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", dv.Name, "scsi", false)

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.VerifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVMI(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	Context("virtctl", func() {
		var (
			vm *v1.VirtualMachine
			sc string
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = tests.GetRWOFileSystemStorageClass()
			if !exists || !tests.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Skip("Skip no wffc storage class available")
			}
			vm = createAndStartWFFCStorageHotplugVM()
		})

		DescribeTable("should add volume according to options", func(dryRun bool) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			dv := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, sc, "64Mi", corev1.ReadWriteOnce, corev1.PersistentVolumeFilesystem)
			_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			Eventually(func() error {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.TODO(), dv.Name, metav1.GetOptions{})
				return err
			}, 40*time.Second, 2*time.Second).Should(Succeed())

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			addVolumeVirtctl(vmi.Name, vmi.Namespace, "", dv.Name, "", dryRun)
			if dryRun {
				verifyNoVolumeAttached(vmi, dv.Name)
			} else {
				verifyVolumeStatus(vmi, v1.VolumeReady, dv.Name)
				getVmiConsoleAndLogin(vmi)
				targets := getTargetsFromVolumeStatus(vmi, dv.Name)
				verifyVolumeAccessible(vmi, targets[0])
				verifySingleAttachmentPod(vmi)
			}
		},
			Entry("with default", false),
			Entry("[test_id:7803]with dry-run", true),
		)

		DescribeTable("should remove volume according to options", func(dryRun bool) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 240)
			dv := tests.NewRandomBlankDataVolume(util.NamespaceTestDefault, sc, "64Mi", corev1.ReadWriteOnce, corev1.PersistentVolumeFilesystem)
			_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			Eventually(func() error {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.TODO(), dv.Name, metav1.GetOptions{})
				return err
			}, 40*time.Second, 2*time.Second).Should(Succeed())

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			addVolumeVirtctl(vmi.Name, vmi.Namespace, "", dv.Name, "", false)
			verifyVolumeStatus(vmi, v1.VolumeReady, dv.Name)

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, dv.Name)
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)

			removeVolumeVirtctl(vmi.Name, vmi.Namespace, dv.Name, dryRun)
			if dryRun {
				Consistently(func() error {
					verifyVolumeStatus(vmi, v1.VolumeReady, dv.Name)
					getVmiConsoleAndLogin(vmi)
					targets := getTargetsFromVolumeStatus(vmi, dv.Name)
					verifyVolumeAccessible(vmi, targets[0])
					verifySingleAttachmentPod(vmi)
					return nil
				}, 60*time.Second, 1*time.Second).Should(BeNil())
			} else {
				verifyVolumeNolongerAccessible(vmi, targets[0])
			}
		},
			Entry("with default", false),
			Entry("[test_id:7829]with dry-run", true),
		)
	})
})
