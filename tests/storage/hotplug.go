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
	"math"
	"path/filepath"
	"strconv"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/log"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtctl "kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	dataMessage             = "data/message"
	addingVolumeRunningVM   = "Adding volume to running VM"
	addingVolumeAgain       = "Adding the same volume again with different name"
	verifyingVolumeDiskInVM = "Verifying the volume and disk are in the VM and VMI"
	removingVolumeFromVM    = "removing volume from VM"
	verifyingVolumeNotExist = "Verifying the volume no longer exists in VM"

	virtCtlNamespace       = "--namespace"
	virtCtlVolumeName      = "--volume-name=%s"
	virtCtlCache           = "--cache=%s"
	verifyCannotAccessDisk = "ls: %s: No such file or directory"

	testNewVolume1 = "some-new-volume1"
	testNewVolume2 = "some-new-volume2"

	waitDiskTemplateError         = "waiting on new disk to appear in template"
	waitVolumeTemplateError       = "waiting on new volume to appear in template"
	waitVolumeRequestProcessError = "waiting on all VolumeRequests to be processed"
)

type addVolumeFunction func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache)
type removeVolumeFunction func(name, namespace, volumeName string, dryRun bool)
type storageClassFunction func() (string, bool)

var _ = SIGDescribe("Hotplug", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	getDryRunOption := func(dryRun bool) []string {
		if dryRun {
			return []string{metav1.DryRunAll}
		}
		return nil
	}

	deleteVirtualMachine := func(vm *v1.VirtualMachine) error {
		return virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
	}

	getAddVolumeOptions := func(volumeName string, bus v1.DiskBus, volumeSource *v1.HotplugVolumeSource, dryRun, useLUN bool, cache v1.DriverCache) *v1.AddVolumeOptions {
		opts := &v1.AddVolumeOptions{
			Name: volumeName,
			Disk: &v1.Disk{
				DiskDevice: v1.DiskDevice{},
				Serial:     volumeName,
			},
			VolumeSource: volumeSource,
			DryRun:       getDryRunOption(dryRun),
		}
		if useLUN {
			opts.Disk.DiskDevice.LUN = &v1.LunTarget{Bus: bus}
		} else {
			opts.Disk.DiskDevice.Disk = &v1.DiskTarget{Bus: bus}
		}
		if cache == v1.CacheNone ||
			cache == v1.CacheWriteThrough ||
			cache == v1.CacheWriteBack {
			opts.Disk.Cache = cache
		}
		return opts
	}
	addVolumeVMIWithSource := func(name, namespace string, volumeOptions *v1.AddVolumeOptions) {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(namespace).AddVolume(context.Background(), name, volumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	addDVVolumeVMI := func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache) {
		addVolumeVMIWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: claimName,
			},
		}, dryRun, false, cache))
	}

	addPVCVolumeVMI := func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache) {
		addVolumeVMIWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun, false, cache))
	}

	addVolumeVMWithSource := func(name, namespace string, volumeOptions *v1.AddVolumeOptions) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).AddVolume(context.Background(), name, volumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	addDVVolumeVM := func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache) {
		addVolumeVMWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: claimName,
			},
		}, dryRun, false, cache))
	}

	addPVCVolumeVM := func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache) {
		addVolumeVMWithSource(name, namespace, getAddVolumeOptions(volumeName, bus, &v1.HotplugVolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun, false, cache))
	}

	addVolumeVirtctl := func(name, namespace, volumeName, claimName string, bus v1.DiskBus, dryRun bool, cache v1.DriverCache) {
		By("Invoking virtlctl addvolume")
		commandAndArgs := []string{virtctl.COMMAND_ADDVOLUME, name, fmt.Sprintf(virtCtlVolumeName, claimName), virtCtlNamespace, namespace}
		if dryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		if cache == v1.CacheNone ||
			cache == v1.CacheWriteThrough ||
			cache == v1.CacheWriteBack {
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlCache, cache))
		}
		addvolumeCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return addvolumeCommand()
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeVolumeVMI := func(name, namespace, volumeName string, dryRun bool) {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(namespace).RemoveVolume(context.Background(), name, &v1.RemoveVolumeOptions{
				Name:   volumeName,
				DryRun: getDryRunOption(dryRun),
			})
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeVolumeVM := func(name, namespace, volumeName string, dryRun bool) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).RemoveVolume(context.Background(), name, &v1.RemoveVolumeOptions{
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
		removeVolumeCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
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
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

	verifyVolumeStatus := func(vmi *v1.VirtualMachineInstance, phase v1.VolumePhase, cache v1.DriverCache, volumeNames ...string) {
		By("Verify the volume status of the hotplugged volume is ready")
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		Eventually(func() error {
			updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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

			// verify disk cache mode in spec
			for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok && disk.Cache != cache {
					return fmt.Errorf("expected disk cache mode is %s, but %s in actual", cache, string(disk.Cache))
				}
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
			currentVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
			&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 -F %s\n", device)},
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
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), tests.NewRandomVirtualMachine(libvmi.NewCirros(), true))
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vm.Status.Ready
		}, 300*time.Second, 1*time.Second).Should(BeTrue())
		return vm
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
		var virtlauncherPod corev1.Pod
		for _, pod := range podList.Items {
			for _, ownerRef := range pod.GetOwnerReferences() {
				if ownerRef.UID == vmi.GetUID() {
					virtlauncherPod = pod
				}
			}
		}
		for _, pod := range podList.Items {
			for _, ownerRef := range pod.GetOwnerReferences() {
				if ownerRef.UID == virtlauncherPod.GetUID() {
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
		dvBlock := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithPVC(
				libdv.PVCWithStorageClass(sc),
				libdv.PVCWithVolumeSize(cd.BlankVolumeSize),
				libdv.PVCWithAccessMode(accessMode),
				libdv.PVCWithVolumeMode(volumeMode),
			),
			libdv.WithForceBindAnnotation(),
		)

		dvBlock, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dvBlock)).Create(context.Background(), dvBlock, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.EventuallyDV(dvBlock, 240, matcher.HaveSucceeded())
		return dvBlock
	}

	verifyAttachDetachVolume := func(vm *v1.VirtualMachine, addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, sc string, volumeMode corev1.PersistentVolumeMode, vmiOnly, waitToStart bool) {
		dv := createDataVolumeAndWaitForImport(sc, volumeMode)

		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if waitToStart {
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)
		}
		By(addingVolumeRunningVM)
		addVolumeFunc(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")
		By(verifyingVolumeDiskInVM)
		if !vmiOnly {
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
		}
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
		verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
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
	}

	Context("Offline VM", func() {
		var (
			vm *v1.VirtualMachine
		)
		BeforeEach(func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), tests.NewRandomVirtualMachine(libvmi.NewCirros(), false))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := deleteVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("Should add volumes on an offline VM", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction) {
			By("Adding test volumes")
			addVolumeFunc(vm.Name, vm.Namespace, testNewVolume1, "madeup", v1.DiskBusSCSI, false, "")
			addVolumeFunc(vm.Name, vm.Namespace, testNewVolume2, "madeup2", v1.DiskBusSCSI, false, "")
			By("Verifying the volumes have been added to the template spec")
			verifyVolumeAndDiskVMAdded(virtClient, vm, testNewVolume1, testNewVolume2)
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
		const (
			numPVs = 3
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Skip("Skip no wffc storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)

			vm = createAndStartWFFCStorageHotplugVM()
		})

		DescribeTable("Should be able to add and use WFFC local storage", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)
			dvNames := make([]string, 0)
			for i := 0; i < numPVs; i++ {
				dv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vm)).Create(context.TODO(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dvNames = append(dvNames, dv.Name)
			}

			for i := 0; i < numPVs; i++ {
				By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
				addVolumeFunc(vm.Name, vm.Namespace, dvNames[i], dvNames[i], v1.DiskBusSCSI, false, "")
			}

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dvNames...)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dvNames...)
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

	Context("[storage-req]", decorators.StorageReq, func() {
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
				sc, exists = libstorage.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when RWXBlock storage class is not present")
				}

				node := findCPUManagerWorkerNode()
				opts := []libvmi.Option{}
				if node != "" {
					opts = append(opts, libvmi.WithNodeSelectorFor(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: node}}))
				}
				vmi := libvmi.NewCirros(opts...)

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), tests.NewRandomVirtualMachine(vmi, true))
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Ready
				}, 300*time.Second, 1*time.Second).Should(BeTrue())
			})

			DescribeTable("should add/remove volume", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly, waitToStart bool) {
				verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, vmiOnly, waitToStart)
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
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				getVmiConsoleAndLogin(vmi)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)
				testVolumes := make([]string, 0)
				for i := 0; i < 5; i++ {
					volumeName := fmt.Sprintf("volume%d", i)
					dv := createDataVolumeAndWaitForImport(sc, volumeMode)
					By(addingVolumeRunningVM)
					addVolumeFunc(vm.Name, vm.Namespace, volumeName, dv.Name, v1.DiskBusSCSI, false, "")
					testVolumes = append(testVolumes, volumeName)
					verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes...)
				}
				By(verifyingVolumeDiskInVM)
				if !vmiOnly {
					verifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes...)
				}
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes...)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes...)
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
				By("Verifying there are no sync errors")
				events, err := virtClient.CoreV1().Events(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, event := range events.Items {
					if event.InvolvedObject.Kind == "VirtualMachineInstance" && event.InvolvedObject.UID == vmi.UID {
						if event.Reason == string(v1.SyncFailed) {
							Fail(fmt.Sprintf("Found sync failed event %v", event))
						}
					}
				}
			},
				Entry("with VMs", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeFilesystem, false),
				Entry("with VMIs", addDVVolumeVMI, removeVolumeVMI, corev1.PersistentVolumeFilesystem, true),
				Entry("with VMs and block", addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeBlock, false),
			)

			DescribeTable("Should be able to add and remove and re-add multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly bool) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)
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
					addVolumeFunc(vm.Name, vm.Namespace, testVolumes[i], dvNames[i], v1.DiskBusSCSI, false, "")
				}

				By(verifyingVolumeDiskInVM)
				if !vmiOnly {
					verifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes[:len(testVolumes)-1]...)
				}
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes[:len(testVolumes)-1]...)
				waitForAttachmentPodToRun(vmi)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes[:len(testVolumes)-1]...)
				verifySingleAttachmentPod(vmi)
				By("removing volume sdc, with dv" + dvNames[2])
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.VolumeStatus[4].Target
				}, 40*time.Second, 2*time.Second).Should(Equal("sdc"))
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.VolumeStatus[5].Target
				}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

				removeVolumeFunc(vm.Name, vm.Namespace, testVolumes[2], false)
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.VolumeStatus[4].Target
				}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

				By("Adding remaining volume, it should end up in the spot that was just cleared")
				addVolumeFunc(vm.Name, vm.Namespace, testVolumes[4], dvNames[4], v1.DiskBusSCSI, false, "")
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, volumeStatus := range vmi.Status.VolumeStatus {
						if volumeStatus.Name == testVolumes[4] {
							return volumeStatus.Target
						}
					}
					return ""
				}, 40*time.Second, 2*time.Second).Should(Equal("sdc"))
				By("Adding intermediate volume, it should end up at the end")
				addVolumeFunc(vm.Name, vm.Namespace, testVolumes[2], dvNames[2], v1.DiskBusSCSI, false, "")
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
				Entry("[Serial] with VMs and block", Serial, addDVVolumeVM, removeVolumeVM, corev1.PersistentVolumeBlock, false),
			)

			It("should permanently add hotplug volume when added to VM, but still unpluggable after restart", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(addingVolumeRunningVM)
				addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dvBlock.Name, v1.DiskBusSCSI, false, "")
				By(verifyingVolumeDiskInVM)
				verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
				verifySingleAttachmentPod(vmi)

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUseable(vmi, []string{"testvolume"})
				Expect(targets).To(HaveLen(1))

				By("stopping VM")
				vm = tests.StopVirtualMachine(vm)

				By("starting VM")
				vm = tests.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume is hotpluggable after a restart")
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

				By("Verifying the hotplug device is auto-mounted during booting")
				getVmiConsoleAndLogin(vmi)
				verifyVolumeAccessible(vmi, targets[0])

				By("Remove volume from a running VM")
				removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume can be unplugged after a restart")
				verifyVolumeNolongerAccessible(vmi, targets[0])
			})

			It("should reject hotplugging a volume with the same name as an existing volume", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(addingVolumeRunningVM)
				err = virtClient.VirtualMachine(vm.Namespace).AddVolume(context.Background(), vm.Name, getAddVolumeOptions("disk0", v1.DiskBusSCSI, &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvBlock.Name,
					},
				}, false, false, ""))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unable to add volume [disk0] because volume with that name already exists"))
			})

			It("should reject hotplugging the same volume with an existing volume name", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(addingVolumeRunningVM)
				addPVCVolumeVMI(vmi.Name, vmi.Namespace, "testvolume", dvBlock.Name, v1.DiskBusSCSI, false, "")

				By(verifyingVolumeDiskInVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

				By(addingVolumeAgain)
				err = virtClient.VirtualMachineInstance(vmi.Namespace).AddVolume(context.Background(), vmi.Name, getAddVolumeOptions(dvBlock.Name, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvBlock.Name,
					},
				}, false, false, ""))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Unable to add volume source [%s] because it already exists", dvBlock.Name)))
			})

			DescribeTable("should reject removing a volume", func(volName, expectedErr string) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(removingVolumeFromVM)
				err = virtClient.VirtualMachine(vm.Namespace).RemoveVolume(context.Background(), vm.Name, &v1.RemoveVolumeOptions{Name: volName})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			},
				Entry("which wasn't hotplugged", "disk0", "Unable to remove volume [disk0] because it is not hotpluggable"),
				Entry("which doesn't exist", "doesntexist", "Unable to remove volume [doesntexist] because it does not exist"),
			)

			It("should allow hotplugging both a filesystem and block volume", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)
				dvFileSystem := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeFilesystem)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)
				getVmiConsoleAndLogin(vmi)

				By(addingVolumeRunningVM)
				addDVVolumeVM(vm.Name, vm.Namespace, "block", dvBlock.Name, v1.DiskBusSCSI, false, "")
				addDVVolumeVM(vm.Name, vm.Namespace, "fs", dvFileSystem.Name, v1.DiskBusSCSI, false, "")
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "block", "fs")

				verifyVolumeStatus(vmi, v1.VolumeReady, "", "block", "fs")
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

			BeforeEach(func() {
				exists := false
				sc, exists = libstorage.GetRWXBlockStorageClass()
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
				for _, node := range libnode.GetAllSchedulableNodes(virtClient).Items {
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
				libnode.AddLabelToNode(sourceNode, hotplugLabelKey, hotplugLabelValue)
				vmi = libvmi.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				vmi.Spec.NodeSelector = map[string]string{hotplugLabelKey: hotplugLabelValue}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
				libnode.AddLabelToNode(targetNode, hotplugLabelKey, hotplugLabelValue)
			})

			AfterEach(func() {
				// Cleanup node labels
				libnode.RemoveLabelFromNode(sourceNode, hotplugLabelKey)
				libnode.RemoveLabelFromNode(targetNode, hotplugLabelKey)
			})

			It("should allow live migration with attached hotplug volumes", func() {
				volumeName := "testvolume"
				volumeMode := corev1.PersistentVolumeBlock
				addVolumeFunc := addDVVolumeVMI
				removeVolumeFunc := removeVolumeVMI
				dv := createDataVolumeAndWaitForImport(sc, volumeMode)

				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)
				By("Verifying the VMI is migrateable")
				Eventually(matcher.ThisVMI(vmi), 90*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceIsMigratable))

				By("Adding volume to running VMI")
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, v1.DiskBusSCSI, false, "")
				By("Verifying the volume and disk are in the VMI")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", volumeName)

				By("Verifying the VMI is still migrateable")
				Eventually(matcher.ThisVMI(vmi), 90*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceIsMigratable))

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUseable(vmi, []string{volumeName})
				Expect(targets).To(HaveLen(1))

				By("Starting the migration multiple times")
				for i := 0; i < numberOfMigrations; i++ {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					sourceAttachmentPods := []string{}
					for _, volumeStatus := range vmi.Status.VolumeStatus {
						if volumeStatus.HotplugVolume != nil {
							sourceAttachmentPods = append(sourceAttachmentPods, volumeStatus.HotplugVolume.AttachPodName)
						}
					}
					Expect(sourceAttachmentPods).To(HaveLen(1))

					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
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
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, v1.DiskBusSCSI, false, "")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", volumeName)
			})
		})

		Context("disk mutating sidecar", func() {
			const (
				hookSidecarImage     = "example-disk-mutation-hook-sidecar"
				sidecarContainerName = "hook-sidecar-0"
				newDiskImgName       = "kubevirt-disk.img"
			)

			var (
				vm *v1.VirtualMachine
				dv *cdiv1.DataVolume
			)

			renderSidecar := func() map[string]string {
				annotation := fmt.Sprintf(`[{"args": ["--version", "v1alpha2"], "image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`,
					flags.KubeVirtUtilityRepoPrefix, hookSidecarImage, flags.KubeVirtUtilityVersionTag)
				return map[string]string{
					"hooks.kubevirt.io/hookSidecars":         annotation,
					"diskimage.vm.kubevirt.io/bootImageName": newDiskImgName,
				}
			}

			BeforeEach(func() {
				if !libstorage.HasCDI() {
					Skip("Skip tests when CDI is not present")
				}
			})

			AfterEach(func() {
				if vm != nil {
					err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					vm = nil
				}
				libstorage.DeleteDataVolume(&dv)
			})

			DescribeTable("should be able to add and remove volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, storageClassFunc storageClassFunction, volumeMode corev1.PersistentVolumeMode, vmiOnly bool) {
				sc, exists := storageClassFunc()
				if !exists {
					Skip("Skip test when appropriate storage class is not available")
				}

				var err error
				url := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)

				storageClass, foundSC := libstorage.GetRWOFileSystemStorageClass()
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				dv = libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithRegistryURLSource(url),
					libdv.WithPVC(
						libdv.PVCWithStorageClass(storageClass),
						libdv.PVCWithVolumeSize("256Mi"),
					),
					libdv.WithForceBindAnnotation(),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the dv import to pvc to finish")
				libstorage.EventuallyDV(dv, 180, matcher.HaveSucceeded())

				By("rename disk image on PVC")
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.RenameImgFile(pvc, newDiskImgName)

				By("start VM with disk mutation sidecar")
				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)
				vmi.ObjectMeta.Annotations = renderSidecar()
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				vm := tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Ready
				}, 300*time.Second, 1*time.Second).Should(BeTrue())

				verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, vmiOnly, true)
			},
				Entry("with DataVolume and running VM", addDVVolumeVM, removeVolumeVM, libstorage.GetRWOBlockStorageClass, corev1.PersistentVolumeFilesystem, false),
				Entry("with DataVolume and VMI directly", addDVVolumeVMI, removeVolumeVMI, libstorage.GetRWOBlockStorageClass, corev1.PersistentVolumeFilesystem, true),
				Entry("[Serial] with Block DataVolume immediate attach", Serial, addDVVolumeVM, removeVolumeVM, libstorage.GetRWOBlockStorageClass, corev1.PersistentVolumeBlock, false),
			)
		})
	})

	Context("delete attachment pod several times", func() {
		var (
			vm       *v1.VirtualMachine
			hpvolume *cdiv1.DataVolume
		)

		BeforeEach(func() {
			if !libstorage.HasCDI() {
				Skip("Skip tests when CDI is not present")
			}
			_, foundSC := libstorage.GetRWXBlockStorageClass()
			if !foundSC {
				Skip("Skip test when block RWX storage is not present")
			}
		})

		AfterEach(func() {
			if vm != nil {
				err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm = nil
			}
		})

		deleteAttachmentPod := func(vmi *v1.VirtualMachineInstance) {
			podName := ""
			for _, volume := range vmi.Status.VolumeStatus {
				if volume.HotplugVolume != nil {
					podName = volume.HotplugVolume.AttachPodName
					break
				}
			}
			Expect(podName).ToNot(BeEmpty())
			foreGround := metav1.DeletePropagationForeground
			err := virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), podName, metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64(0),
				PropagationPolicy:  &foreGround,
			})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				_, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), podName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		}

		It("should remain active", func() {
			checkVolumeName := "checkvolume"
			volumeMode := corev1.PersistentVolumeBlock
			addVolumeFunc := addDVVolumeVMI
			var err error
			storageClass, _ := libstorage.GetRWXBlockStorageClass()

			blankDv := func() *cdiv1.DataVolume {
				return libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithBlankImageSource(),
					libdv.WithPVC(
						libdv.PVCWithStorageClass(storageClass),
						libdv.PVCWithVolumeSize(cd.BlankVolumeSize),
						libdv.PVCWithReadWriteManyAccessMode(),
						libdv.PVCWithVolumeMode(volumeMode),
					),
					libdv.WithForceBindAnnotation(),
				)
			}
			vmi := libvmi.NewCirros()
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
			By("creating blank hotplug volumes")
			hpvolume = blankDv()
			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(hpvolume.Namespace).Create(context.Background(), hpvolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("waiting for the dv import to pvc to finish")
			libstorage.EventuallyDV(dv, 180, matcher.HaveSucceeded())
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("hotplugging the volume check volume")
			addVolumeFunc(vmi.Name, vmi.Namespace, checkVolumeName, hpvolume.Name, v1.DiskBusSCSI, false, "")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, checkVolumeName)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", checkVolumeName)
			getVmiConsoleAndLogin(vmi)

			By("verifying the volume is useable and creating some data on it")
			verifyHotplugAttachedAndUseable(vmi, []string{checkVolumeName})
			targets := getTargetsFromVolumeStatus(vmi, checkVolumeName)
			Expect(targets).ToNot(BeEmpty())
			verifyWriteReadData(vmi, targets[0])
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("deleting the attachment pod a few times, try to make the currently attach volume break")
			for i := 0; i < 10; i++ {
				deleteAttachmentPod(vmi)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			By("verifying the volume has not been disturbed in the VM")
			targets = getTargetsFromVolumeStatus(vmi, checkVolumeName)
			Expect(targets).ToNot(BeEmpty())
			verifyWriteReadData(vmi, targets[0])
		})
	})

	Context("with limit range in namespace", func() {
		var (
			sc                         string
			lr                         *corev1.LimitRange
			orgCdiResourceRequirements *corev1.ResourceRequirements
			originalConfig             v1.KubeVirtConfiguration
		)

		createVMWithRatio := func(memRatio, cpuRatio float64) *v1.VirtualMachine {
			vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), true)

			memLimit := int64(1024 * 1024 * 128) //128Mi
			memRequest := int64(math.Ceil(float64(memLimit) / memRatio))
			memRequestQuantity := resource.NewScaledQuantity(memRequest, 0)
			memLimitQuantity := resource.NewScaledQuantity(memLimit, 0)
			cpuLimit := int64(1)
			cpuRequest := int64(math.Ceil(float64(cpuLimit) / cpuRatio))
			cpuRequestQuantity := resource.NewScaledQuantity(cpuRequest, 0)
			cpuLimitQuantity := resource.NewScaledQuantity(cpuLimit, 0)
			vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory] = *memRequestQuantity
			vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceCPU] = *cpuRequestQuantity
			vm.Spec.Template.Spec.Domain.Resources.Limits = corev1.ResourceList{}
			vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceMemory] = *memLimitQuantity
			vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceCPU] = *cpuLimitQuantity
			vm.Spec.Running = pointer.Bool(true)
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
			return vm
		}

		updateCDIResourceRequirements := func(requirements *corev1.ResourceRequirements) {
			if !libstorage.HasCDI() {
				Skip("Test requires CDI CR to be available")
			}
			orgCdiConfig, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			cdi := libstorage.GetCDI(virtClient)
			orgCdiResourceRequirements = cdi.Spec.Config.PodResourceRequirements
			newCdi := cdi.DeepCopy()
			newCdi.Spec.Config.PodResourceRequirements = requirements
			_, err = virtClient.CdiClient().CdiV1beta1().CDIs().Update(context.Background(), newCdi, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				cdiConfig, _ := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
				if cdiConfig == nil {
					return false
				}
				return cdiConfig.Generation > orgCdiConfig.Generation
			}, 30*time.Second, 1*time.Second).Should(BeTrue())
		}

		updateCDIToRatio := func(memRatio, cpuRatio float64) {
			memLimitQuantity := resource.MustParse("600M")
			memLimit := memLimitQuantity.Value()
			memRequest := int64(math.Ceil(float64(memLimit) / memRatio))
			memRequestQuantity := resource.NewScaledQuantity(memRequest, 0)
			cpuLimitQuantity := resource.MustParse("750m")
			cpuLimit := cpuLimitQuantity.AsDec().UnscaledBig().Int64()
			cpuRequest := int64(math.Ceil(float64(cpuLimit) / cpuRatio))
			cpuRequestQuantity := resource.NewScaledQuantity(cpuRequest, resource.Milli)
			updateCDIResourceRequirements(&corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *cpuRequestQuantity,
					corev1.ResourceMemory: *memRequestQuantity,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    cpuLimitQuantity,
					corev1.ResourceMemory: memLimitQuantity,
				},
			})
		}

		updateKubeVirtToRatio := func(memRatio, cpuRatio float64) {
			memLimitQuantity := resource.MustParse("80M")
			memLimit := memLimitQuantity.Value()
			memRequest := int64(math.Ceil(float64(memLimit) / memRatio))
			memRequestQuantity := resource.NewScaledQuantity(memRequest, 0)
			cpuLimitQuantity := resource.MustParse("100m")
			cpuLimit := cpuLimitQuantity.AsDec().UnscaledBig().Int64()
			cpuRequest := int64(math.Ceil(float64(cpuLimit) / cpuRatio))
			cpuRequestQuantity := resource.NewScaledQuantity(cpuRequest, resource.Milli)
			By("Updating hotplug and container disks ratio to the specified ratio")
			resources := corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *cpuRequestQuantity,
					corev1.ResourceMemory: *memRequestQuantity,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    cpuLimitQuantity,
					corev1.ResourceMemory: memLimitQuantity,
				},
			}
			config := originalConfig.DeepCopy()
			config.SupportContainerResources = []v1.SupportContainerResources{
				{
					Type:      v1.HotplugAttachment,
					Resources: resources,
				},
				{
					Type:      v1.ContainerDisk,
					Resources: resources,
				},
				{
					Type:      v1.GuestConsoleLog,
					Resources: resources,
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(*config)
		}

		createLimitRangeInNamespace := func(namespace string, memRatio, cpuRatio float64) {
			lr = &corev1.LimitRange{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-lr", namespace),
				},
				Spec: corev1.LimitRangeSpec{
					Limits: []corev1.LimitRangeItem{
						{
							Type: corev1.LimitTypeContainer,
							MaxLimitRequestRatio: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%f", memRatio)),
								corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%f", cpuRatio)),
							},
							Max: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1Gi"),
								corev1.ResourceCPU:    resource.MustParse("1"),
							},
							Min: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1Mi"),
								corev1.ResourceCPU:    resource.MustParse("1m"),
							},
						},
					},
				},
			}
			lr, err = virtClient.CoreV1().LimitRanges(namespace).Create(context.Background(), lr, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Ensuring LimitRange exists")
			Eventually(func() error {
				lr, err = virtClient.CoreV1().LimitRanges(namespace).Get(context.Background(), lr.Name, metav1.GetOptions{})
				return err
			}, 30*time.Second, 1*time.Second).Should(BeNil())
		}

		BeforeEach(func() {
			exists := false
			sc, exists = libstorage.GetRWXBlockStorageClass()

			if !exists {
				Skip("Skip test when RWXBlock storage class is not present")
			}
			originalConfig = *util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
		})

		AfterEach(func() {
			if lr != nil {
				err = virtClient.CoreV1().LimitRanges(lr.Namespace).Delete(context.Background(), lr.Name, metav1.DeleteOptions{})
				if !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
				lr = nil
			}
			updateCDIResourceRequirements(orgCdiResourceRequirements)
			orgCdiResourceRequirements = nil
			tests.UpdateKubeVirtConfigValueAndWait(originalConfig)
		})

		// Needs to be serial since I am putting limit range on namespace
		DescribeTable("hotplug volume should have mem ratio same as VMI with limit range applied", func(memRatio, cpuRatio float64) {
			updateCDIToRatio(memRatio, cpuRatio)
			updateKubeVirtToRatio(memRatio, cpuRatio)
			createLimitRangeInNamespace(util.NamespaceTestDefault, memRatio, cpuRatio)
			vm := createVMWithRatio(memRatio, cpuRatio)
			dv := createDataVolumeAndWaitForImport(sc, corev1.PersistentVolumeBlock)

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")
			By(verifyingVolumeDiskInVM)
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
			verifySingleAttachmentPod(vmi)
			By("Verifying request/limit ratio on attachment pod")
			podList, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			var virtlauncherPod, attachmentPod corev1.Pod
			By("Finding virt-launcher pod")
			for _, pod := range podList.Items {
				for _, ownerRef := range pod.GetOwnerReferences() {
					if ownerRef.UID == vmi.GetUID() {
						virtlauncherPod = pod
						break
					}
				}
			}
			// Attachment pod is owned by virt-launcher pod
			for _, pod := range podList.Items {
				for _, ownerRef := range pod.GetOwnerReferences() {
					if ownerRef.UID == virtlauncherPod.GetUID() {
						attachmentPod = pod
						break
					}
				}
			}
			By("Checking hotplug attachment pod ratios")
			Expect(attachmentPod.Name).To(ContainSubstring("hp-volume-"))
			memLimit := attachmentPod.Spec.Containers[0].Resources.Limits.Memory().Value()
			memRequest := attachmentPod.Spec.Containers[0].Resources.Requests.Memory().Value()
			Expect(float64(memRequest) * memRatio).To(BeNumerically(">=", float64(memLimit)))
			cpuLimit := attachmentPod.Spec.Containers[0].Resources.Limits.Cpu().Value()
			cpuRequest := attachmentPod.Spec.Containers[0].Resources.Requests.Cpu().Value()
			Expect(float64(cpuRequest) * cpuRatio).To(BeNumerically(">=", float64(cpuLimit)))

			By("Checking virt-launcher ")
			for _, container := range virtlauncherPod.Spec.Containers {
				memLimit := container.Resources.Limits.Memory().Value()
				memRequest := container.Resources.Requests.Memory().Value()
				Expect(float64(memRequest) * memRatio).To(BeNumerically(">=", float64(memLimit)))
				cpuLimit := container.Resources.Limits.Cpu().Value()
				cpuRequest := container.Resources.Requests.Cpu().Value()
				Expect(float64(cpuRequest) * cpuRatio).To(BeNumerically(">=", float64(cpuLimit)))
			}

			for _, container := range virtlauncherPod.Spec.InitContainers {
				memLimit := container.Resources.Limits.Memory().Value()
				memRequest := container.Resources.Requests.Memory().Value()
				Expect(float64(memRequest) * memRatio).To(BeNumerically(">=", float64(memLimit)))
				cpuLimit := container.Resources.Limits.Cpu().Value()
				cpuRequest := container.Resources.Requests.Cpu().Value()
				Expect(float64(cpuRequest) * cpuRatio).To(BeNumerically(">=", float64(cpuLimit)))
			}

			By("Remove volume from a running VM")
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[Serial]1 to 1 cpu and mem ratio", Serial, float64(1), float64(1)),
			Entry("[Serial]1 to 1 mem ratio, 4 to 1 cpu ratio", Serial, float64(1), float64(4)),
			Entry("[Serial]2 to 1 mem ratio, 4 to 1 cpu ratio", Serial, float64(2), float64(4)),
			Entry("[Serial]2.25 to 1 mem ratio, 5.75 to 1 cpu ratio", Serial, float64(2.25), float64(5.75)),
		)
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
			libstorage.CreateStorageClass(storageClassHostPath, &immediateBinding)
			pvNode := libstorage.CreateHostPathPvWithSizeAndStorageClass(tests.CustomHostPath, testsuite.GetTestNamespace(nil), hotplugPvPath, "1Gi", storageClassHostPath)
			libstorage.CreatePVC(tests.CustomHostPath, testsuite.GetTestNamespace(nil), "1Gi", storageClassHostPath, false)

			opts := []libvmi.Option{}
			if pvNode != "" {
				opts = append(opts, libvmi.WithNodeSelectorFor(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: pvNode}}))
			}
			vmi := libvmi.NewCirros(opts...)

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), tests.NewRandomVirtualMachine(vmi, true))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		})

		AfterEach(func() {
			tests.DeletePvAndPvc(fmt.Sprintf("%s-disk-for-tests", tests.CustomHostPath))
			libstorage.DeleteStorageClass(storageClassHostPath)
		})

		It("should attach a hostpath based volume to running VM", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			name := fmt.Sprintf("disk-%s", tests.CustomHostPath)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", name, v1.DiskBusSCSI, false, "")

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

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
			vmi := libvmi.NewCirros()
			policy := v1.IOThreadsPolicyShared
			vmi.Spec.Domain.IOThreadsPolicy = &policy
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), tests.NewRandomVirtualMachine(vmi, true))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("should allow adding and removing hotplugged volumes", func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip no filesystem storage class available")
			}

			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

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
			libstorage.CreateAllSeparateDeviceHostPathPvs(tests.CustomHostPath, testsuite.GetTestNamespace(nil))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), tests.NewRandomVirtualMachine(libvmi.NewCirros(), true))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		})

		AfterEach(func() {
			libstorage.DeleteAllSeparateDeviceHostPathPvs()
		})

		It("should attach a hostpath based volume to running VM", func() {
			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(
					libdv.PVCWithStorageClass(libstorage.StorageClassHostPathSeparateDevice),
					libdv.PVCWithVolumeSize(cd.BlankVolumeSize),
				),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			addPVCVolumeVMI(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVMI(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	// Some of the functions used here don't behave well in parallel (CreateSCSIDisk).
	// The device is created directly on the node and the addition and removal
	// of the scsi_debug kernel module could create flakiness in parallel.
	Context("[Serial]Hotplug LUN disk", Serial, func() {
		var (
			nodeName, address, device string
			pvc                       *corev1.PersistentVolumeClaim
			pv                        *corev1.PersistentVolume
			vm                        *v1.VirtualMachine
		)

		BeforeEach(func() {
			nodeName = tests.NodeNameWithHandler()
			address, device = tests.CreateSCSIDisk(nodeName, []string{})
			By(fmt.Sprintf("Create PVC with SCSI disk %s", device))
			pv, pvc, err = tests.CreatePVandPVCwithSCSIDisk(nodeName, device, util.NamespaceTestDefault, "scsi-disks", "scsipv", "scsipvc")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// Delete the scsi disk
			tests.RemoveSCSIDisk(nodeName, address)
			Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			err := deleteVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("on an offline VM", func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), tests.NewRandomVirtualMachine(libvmi.NewCirros(), false))
			Expect(err).ToNot(HaveOccurred())
			By("Adding test volumes")
			pv2, pvc2, err := tests.CreatePVandPVCwithSCSIDisk(nodeName, device, util.NamespaceTestDefault, "scsi-disks-test2", "scsipv2", "scsipvc2")
			Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for scsi disk")

			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions(testNewVolume1, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				}},
			}, false, true, ""))
			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions(testNewVolume2, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc2.Name,
				}},
			}, false, true, ""))

			By("Verifying the volumes have been added to the template spec")
			verifyVolumeAndDiskVMAdded(virtClient, vm, testNewVolume1, testNewVolume2)

			By("Removing new volumes from VM")
			removeVolumeVM(vm.Name, vm.Namespace, testNewVolume1, false)
			removeVolumeVM(vm.Name, vm.Namespace, testNewVolume2, false)

			verifyVolumeAndDiskVMRemoved(vm, testNewVolume1, testNewVolume2)
			Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv2.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
		})

		It("on an online VM", func() {
			vmi := libvmi.NewCirros(libvmi.WithNodeSelectorFor(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
			}))

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), tests.NewRandomVirtualMachine(vmi, true))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			By(addingVolumeRunningVM)
			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions("testvolume", v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				}},
			}, false, true, ""))
			By(verifyingVolumeDiskInVM)
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
			getVmiConsoleAndLogin(vmi)
			targets := verifyHotplugAttachedAndUseable(vmi, []string{"testvolume"})
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			By(verifyingVolumeNotExist)
			verifyVolumeAndDiskVMRemoved(vm, "testvolume")
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
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists || !libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
				Skip("Skip no wffc storage class available")
			}
			vm = createAndStartWFFCStorageHotplugVM()
		})

		DescribeTable("should add volume according to options", func(dryRun bool) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vmi)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vmi)).Get(context.TODO(), dv.Name, metav1.GetOptions{})
				return err
			}, 40*time.Second, 2*time.Second).Should(Succeed())

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			addVolumeVirtctl(vmi.Name, vmi.Namespace, "", dv.Name, "", dryRun, v1.CacheWriteBack)
			if dryRun {
				verifyNoVolumeAttached(vmi, dv.Name)
			} else {
				verifyVolumeStatus(vmi, v1.VolumeReady, v1.CacheWriteBack, dv.Name)
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
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)
			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vmi)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vmi)).Get(context.TODO(), dv.Name, metav1.GetOptions{})
				return err
			}, 40*time.Second, 2*time.Second).Should(Succeed())

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			addVolumeVirtctl(vmi.Name, vmi.Namespace, "", dv.Name, "", false, "")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dv.Name)

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, dv.Name)
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)

			removeVolumeVirtctl(vmi.Name, vmi.Namespace, dv.Name, dryRun)
			if dryRun {
				Consistently(func() error {
					verifyVolumeStatus(vmi, v1.VolumeReady, "", dv.Name)
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

func verifyVolumeAndDiskVMAdded(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	log.Log.Infof("Checking %d volumes", len(volumeNames))
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(updatedVM.Status.VolumeRequests) > 0 {
			return fmt.Errorf(waitVolumeRequestProcessError)
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
			return fmt.Errorf(waitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(waitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func verifyVolumeAndDiskVMIAdded(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
			return fmt.Errorf(waitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(waitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}
