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
	"math"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
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

var _ = Describe(SIG("Hotplug", func() {
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
		return virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
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
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun, false, cache))
	}

	addVolumeVMWithSource := func(name, namespace string, volumeOptions *v1.AddVolumeOptions) {
		var err error

		// Try at least 3 times, this is done because `AddVolume` is inherently racy in the way it's implemented
		// as when it patches the VM Status it expects the field `volumeRequests` to be there (by using a test json patch op),
		// but at the same time virt-controller trims this field when all requests have been satisfied.
		// To avoid hitting these we should explicitly try multiple times.
		for i := 0; i < 3; i++ {
			err = virtClient.VirtualMachine(namespace).AddVolume(context.Background(), name, volumeOptions)
			if err == nil {
				break
			}
		}
		Expect(err).ToNot(HaveOccurred())
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
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		}, dryRun, false, cache))
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

	verifyVolumeAndDiskVMRemoved := func(vm *v1.VirtualMachine, volumeNames ...string) {
		nameMap := make(map[string]bool)
		for _, volumeName := range volumeNames {
			nameMap[volumeName] = true
		}
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if len(updatedVM.Status.VolumeRequests) > 0 {
				return fmt.Errorf(waitVolumeRequestProcessError)
			}

			for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
				if _, ok := nameMap[volume.Name]; ok {
					return fmt.Errorf("waiting on VM volume to be removed")
				}
			}
			for _, disk := range updatedVM.Spec.Template.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok {
					return fmt.Errorf("waiting on VM disk to be removed")
				}
			}

			updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return err
			}

			for _, volume := range updatedVMI.Spec.Volumes {
				if _, ok := nameMap[volume.Name]; ok {
					return fmt.Errorf("waiting on VMI volume to be removed")
				}
			}
			for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok {
					return fmt.Errorf("waiting on VMI disk to be removed")
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
			updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
			currentVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
			&expect.BSnd{S: console.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: fmt.Sprintf("sudo mount %s %s\n", device, filepath.Join("/test", filepath.Base(device)))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: console.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo mkdir -p %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: console.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("sudo chmod a+w %s\n", filepath.Join("/test", filepath.Base(device), "data"))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: console.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
			&expect.BSnd{S: fmt.Sprintf("echo '%s' > %s\n", vmi.UID, filepath.Join("/test", filepath.Base(device), dataMessage))},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: console.EchoLastReturnValue},
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
			&expect.BSnd{S: console.EchoLastReturnValue},
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
				&expect.BSnd{S: console.EchoLastReturnValue},
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

		var virtlauncher *k8sv1.Pod
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
						return pod.Status.Phase == k8sv1.PodRunning
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
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		return vm
	}

	createBootableHotplugVM := func(storageClass string) *v1.VirtualMachine {
		opts := []libvmi.Option{
			libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
		}
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithStorageClass(storageClass),
				libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros))),
			),
		)
		vm := libvmi.NewVirtualMachine(
			libstorage.RenderVMIWithHotplugDataVolume(dv.Name, dv.Namespace, opts...),
			libvmi.WithDataVolumeTemplate(dv),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		return vm
	}

	verifyHotplugAttachedAndUsable := func(vmi *v1.VirtualMachineInstance, names []string) []string {
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
		var virtlauncherPod k8sv1.Pod
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

	createDataVolumeAndWaitForImport := func(sc string, volumeMode k8sv1.PersistentVolumeMode) *cdiv1.DataVolume {
		accessMode := k8sv1.ReadWriteOnce
		if volumeMode == k8sv1.PersistentVolumeBlock {
			accessMode = k8sv1.ReadWriteMany
		}

		By("Creating DataVolume")
		dvBlock := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithStorage(
				libdv.StorageWithStorageClass(sc),
				libdv.StorageWithVolumeSize(cd.BlankVolumeSize),
				libdv.StorageWithAccessMode(accessMode),
				libdv.StorageWithVolumeMode(volumeMode),
			),
		)

		dvBlock, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dvBlock)).Create(context.Background(), dvBlock, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.EventuallyDV(dvBlock, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
		return dvBlock
	}

	verifyAttachDetachVolume := func(obj metav1.Object,
		addVolumeFunc addVolumeFunction,
		removeVolumeFunc removeVolumeFunction,
		sc string,
		volumeMode k8sv1.PersistentVolumeMode,
		bus v1.DiskBus,
		waitToStart bool,
	) {
		vm, isVM := obj.(*v1.VirtualMachine)
		dv := createDataVolumeAndWaitForImport(sc, volumeMode)

		vmi, err := virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if waitToStart {
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)
		}
		By(addingVolumeRunningVM)
		addVolumeFunc(obj.GetName(), obj.GetNamespace(), "testvolume", dv.Name, bus, false, "")
		By(verifyingVolumeDiskInVM)
		if isVM {
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
		}
		vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
		verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
		getVmiConsoleAndLogin(vmi)
		targets := verifyHotplugAttachedAndUsable(vmi, []string{"testvolume"})
		verifySingleAttachmentPod(vmi)
		By(removingVolumeFromVM)
		removeVolumeFunc(obj.GetName(), obj.GetNamespace(), "testvolume", false)
		if isVM {
			By(verifyingVolumeNotExist)
			verifyVolumeAndDiskVMRemoved(vm, "testvolume")
		}
		verifyVolumeNolongerAccessible(vmi, targets[0])
	}

	addRemoveReAddTest := func(obj metav1.Object, addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, sc string, volumeMode k8sv1.PersistentVolumeMode) {
		vm, isVM := obj.(*v1.VirtualMachine)
		vmi, err := virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
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
			addVolumeFunc(obj.GetName(), obj.GetNamespace(), testVolumes[i], dvNames[i], v1.DiskBusSCSI, false, "")
		}

		By(verifyingVolumeDiskInVM)
		if isVM {
			verifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes[:len(testVolumes)-1]...)
		}
		vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		verifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes[:len(testVolumes)-1]...)
		waitForAttachmentPodToRun(vmi)
		verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes[:len(testVolumes)-1]...)
		verifySingleAttachmentPod(vmi)
		By("removing volume sdc, with dv" + dvNames[2])
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vmi.Status.VolumeStatus[4].Target
		}, 40*time.Second, 2*time.Second).Should(Equal("sdc"))
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vmi.Status.VolumeStatus[5].Target
		}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

		removeVolumeFunc(obj.GetName(), obj.GetNamespace(), testVolumes[2], false)
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vmi.Status.VolumeStatus[4].Target
		}, 40*time.Second, 2*time.Second).Should(Equal("sdd"))

		By("Adding remaining volume, it should end up in the spot that was just cleared")
		addVolumeFunc(obj.GetName(), obj.GetNamespace(), testVolumes[4], dvNames[4], v1.DiskBusSCSI, false, "")
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return libstorage.LookupVolumeTargetPath(vmi, testVolumes[4])
		}, 80*time.Second, 2*time.Second).Should(Equal("/dev/sdc"))
		By("Adding intermediate volume, it should end up at the end")
		addVolumeFunc(obj.GetName(), obj.GetNamespace(), testVolumes[2], dvNames[2], v1.DiskBusSCSI, false, "")
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return libstorage.LookupVolumeTargetPath(vmi, testVolumes[2])
		}, 80*time.Second, 2*time.Second).Should(Equal("/dev/sde"))
		verifySingleAttachmentPod(vmi)
		for _, volumeName := range testVolumes {
			By(removingVolumeFromVM)
			removeVolumeFunc(obj.GetName(), obj.GetNamespace(), volumeName, false)
			if isVM {
				By(verifyingVolumeNotExist)
				verifyVolumeAndDiskVMRemoved(vm, volumeName)
			}
		}
	}

	Context("Offline VM", func() {
		var (
			vm *v1.VirtualMachine
		)
		BeforeEach(func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmi.NewVirtualMachine(libvmifact.NewCirros()), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := deleteVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("Should add volumes on an offline VM", decorators.StorageCritical, func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction) {
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

	Context("Offline VM with a block volume", decorators.RequiresRWXBlock, func() {
		var (
			vm *v1.VirtualMachine
			sc string
		)

		BeforeEach(func() {
			var exists bool

			sc, exists = libstorage.GetRWXBlockStorageClass()
			if !exists {
				Fail("Fail test when RWXBlock storage class is not present")
			}

			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros))),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteMany),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
				),
			)
			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace, libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData("#!/bin/bash\necho 'hello'\n")))

			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), libvmi.NewVirtualMachine(vmi), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should be able to boot from block volume", decorators.StorageCritical, func() {
			dvName := "disk0"
			vm = createBootableHotplugVM(sc)
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dvName)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dvName)
			getVmiConsoleAndLogin(vmi)
			verifySingleAttachmentPod(vmi)
		})

		DescribeTable("Should start with a hotplug block", decorators.StorageCritical, func(addVolumeFunc addVolumeFunction) {
			dv := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)

			By("Adding a hotplug block volume")
			addVolumeFunc(vm.Name, vm.Namespace, dv.Name, dv.Name, v1.DiskBusSCSI, false, "")

			By("Verifying the volume has been added to the template spec")
			verifyVolumeAndDiskVMAdded(virtClient, vm, dv.Name)

			By("Starting the VM")
			vm = libvmops.StartVirtualMachine(vm)
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the volume is attached and usable")
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dv.Name)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dv.Name)
			getVmiConsoleAndLogin(vmi)
			targets := verifyHotplugAttachedAndUsable(vmi, []string{dv.Name})
			Expect(targets).To(HaveLen(1))
		},
			Entry("DataVolume", addDVVolumeVM),
			Entry("PersistentVolume", addPVCVolumeVM),
		)

		It("Should preserve access to block devices if virt-handler crashes", Serial, func() {
			blockDevices := []string{"/dev/disk0"}

			By("Adding a hotplug block volume")
			dv := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)
			blockDevices = append(blockDevices, fmt.Sprintf("/var/run/kubevirt/hotplug-disks/%s", dv.Name))
			addDVVolumeVM(vm.Name, vm.Namespace, dv.Name, dv.Name, v1.DiskBusSCSI, false, "")

			By("Verifying the volume has been added to the template spec")
			verifyVolumeAndDiskVMAdded(virtClient, vm, dv.Name)

			By("Starting the VM")
			vm = libvmops.StartVirtualMachine(vm)
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the volume is attached and usable")
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dv.Name)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dv.Name)
			getVmiConsoleAndLogin(vmi)
			targets := verifyHotplugAttachedAndUsable(vmi, []string{dv.Name})
			Expect(targets).To(HaveLen(1))

			By("Deleting virt-handler pod")
			virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				err := virtClient.CoreV1().
					Pods(virtHandlerPod.GetObjectMeta().GetNamespace()).
					Delete(context.Background(), virtHandlerPod.GetObjectMeta().GetName(), metav1.DeleteOptions{})
				return err
			}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "virt-handler pod is expected to be deleted")

			By("Waiting for virt-handler pod to restart")
			Eventually(func() bool {
				virtHandlerPod, err = libnode.GetVirtHandlerPod(virtClient, vmi.Status.NodeName)
				return err == nil && virtHandlerPod.Status.Phase == k8sv1.PodRunning
			}, 60*time.Second, 1*time.Second).Should(BeTrue(), "virt-handler pod is expected to be restarted")

			By("Adding another hotplug block volume")
			dv = createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)
			blockDevices = append(blockDevices, fmt.Sprintf("/var/run/kubevirt/hotplug-disks/%s", dv.Name))
			addDVVolumeVM(vm.Name, vm.Namespace, dv.Name, dv.Name, v1.DiskBusSCSI, false, "")

			By("Verifying the volume is attached and usable")
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dv.Name)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dv.Name)
			getVmiConsoleAndLogin(vmi)
			targets = verifyHotplugAttachedAndUsable(vmi, []string{dv.Name})
			Expect(targets).To(HaveLen(1))

			By("Verifying the block devices are still accessible")
			for _, dev := range blockDevices {
				By(fmt.Sprintf("Verifying %s", dev))
				output := libpod.RunCommandOnVmiPod(vmi, []string{
					"dd", fmt.Sprintf("if=%s", dev), "of=/dev/null", "bs=1", "count=1", "status=none",
				})
				Expect(output).To(BeEmpty())
			}
		})
	})

	Context("WFFC storage", decorators.RequiresWFFCStorageClass, func() {
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
				Fail("fail test, no wffc storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)
		})

		It("Should be able to boot from WFFC local storage", decorators.StorageCritical, func() {
			dvName := "disk0"
			vm = createBootableHotplugVM(sc)
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dvName)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dvName)
			getVmiConsoleAndLogin(vmi)
			verifySingleAttachmentPod(vmi)
		})

		It("Should be able to add and use WFFC local storage", func() {
			vm = createAndStartWFFCStorageHotplugVM()
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)
			dvNames := make([]string, 0)
			for i := 0; i < numPVs; i++ {
				dv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithVolumeSize(cd.BlankVolumeSize)),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(vm)).Create(context.TODO(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dvNames = append(dvNames, dv.Name)
			}

			for i := 0; i < numPVs; i++ {
				By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
				addDVVolumeVM(vm.Name, vm.Namespace, dvNames[i], dvNames[i], v1.DiskBusSCSI, false, "")
			}

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, dvNames...)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", dvNames...)
			getVmiConsoleAndLogin(vmi)
			verifyHotplugAttachedAndUsable(vmi, dvNames)
			verifySingleAttachmentPod(vmi)
			for _, volumeName := range dvNames {
				By("removing volume " + volumeName + " from VM")
				removeVolumeVM(vm.Name, vm.Namespace, volumeName, false)
			}
			for _, volumeName := range dvNames {
				verifyVolumeNolongerAccessible(vmi, volumeName)
			}
		})
	})

	Context("[storage-req]", decorators.StorageReq, func() {
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

		validateDryRun := func(obj metav1.Object, addVolumeFunc addVolumeFunction, sc string, volumeMode k8sv1.PersistentVolumeMode) {
			dv := createDataVolumeAndWaitForImport(sc, volumeMode)

			vmi, err := virtClient.VirtualMachineInstance(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			addVolumeFunc(obj.GetName(), obj.GetNamespace(), "testvolume", dv.Name, v1.DiskBusSCSI, true, "")
			verifyNoVolumeAttached(vmi, "testvolume")
		}

		Context("VMI", decorators.RequiresRWXBlock, func() {
			var (
				vmi *v1.VirtualMachineInstance
				sc  string
			)

			BeforeEach(func() {
				exists := false
				sc, exists = libstorage.GetRWXBlockStorageClass()
				if !exists {
					Fail("Fail test when RWXBlock storage class is not present")
				}

				node := findCPUManagerWorkerNode()
				opts := []libvmi.Option{}
				if node != "" {
					opts = append(opts, libvmi.WithNodeSelectorFor(node))
				}
				vmi = libvmifact.NewCirros(opts...)

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVMI(vmi)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeRunning())
			})

			DescribeTable("should add/remove volume", decorators.StorageCritical, func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode, waitToStart bool) {
				verifyAttachDetachVolume(vmi, addVolumeFunc, removeVolumeFunc, sc, volumeMode, v1.DiskBusSCSI, waitToStart)
			},
				Entry("with DataVolume immediate attach, VMI directly", addDVVolumeVMI, removeVolumeVMI, k8sv1.PersistentVolumeFilesystem, false),
				Entry("with PersistentVolume immediate attach, VMI directly", addPVCVolumeVMI, removeVolumeVMI, k8sv1.PersistentVolumeFilesystem, false),
			)

			DescribeTable("should not add/remove volume with dry run", func(addVolumeFunc addVolumeFunction, volumeMode k8sv1.PersistentVolumeMode) {
				validateDryRun(vmi, addVolumeFunc, sc, volumeMode)
			},
				Entry("with DataVolume immediate attach, VMI directly", addDVVolumeVMI, k8sv1.PersistentVolumeFilesystem),
				Entry("with PersistentVolume immediate attach, VMI directly", addPVCVolumeVMI, k8sv1.PersistentVolumeFilesystem),
			)

			DescribeTable("Should be able to add and remove and re-add multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode) {
				addRemoveReAddTest(vmi, addVolumeFunc, removeVolumeFunc, sc, volumeMode)
			},
				Entry("with VMIs", addDVVolumeVMI, removeVolumeVMI, k8sv1.PersistentVolumeFilesystem),
			)
		})

		Context("Online VM", decorators.RequiresRWXBlock, func() {
			var (
				vm *v1.VirtualMachine
				sc string
			)

			BeforeEach(func() {
				exists := false
				sc, exists = libstorage.GetRWXBlockStorageClass()
				if !exists {
					Fail("Fail test when RWXBlock storage class is not present")
				}

				node := findCPUManagerWorkerNode()
				opts := []libvmi.Option{}
				if node != "" {
					opts = append(opts, libvmi.WithNodeSelectorFor(node))
				}
				vmi := libvmifact.NewCirros(opts...)

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
			})

			Context("with legacy hotplug", Serial, func() {
				BeforeEach(func() {
					kvconfig.DisableFeatureGate(featuregate.DeclarativeHotplugVolumesGate)
					kvconfig.EnableFeatureGate(featuregate.HotplugVolumesGate)
				})

				AfterEach(func() {
					kvconfig.DisableFeatureGate(featuregate.HotplugVolumesGate)
					kvconfig.EnableFeatureGate(featuregate.DeclarativeHotplugVolumesGate)
				})

				DescribeTable("should add/remove volume", decorators.StorageCritical, func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode, waitToStart bool) {
					verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, v1.DiskBusSCSI, waitToStart)
				},
					Entry("with DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, false),
					Entry("with PersistentVolume immediate attach", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, false),
					Entry("with DataVolume wait for VM to finish starting", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, true),
					Entry("with PersistentVolume wait for VM to finish starting", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, true),
					Entry("with Block DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock, false),
				)
			})

			Context("with declarative hotplug", func() {
				DescribeTable("should add/remove volume", decorators.StorageCritical, func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode, waitToStart bool) {
					verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, v1.DiskBusSCSI, waitToStart)
				},
					Entry("with DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, false),
					Entry("with PersistentVolume immediate attach", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, false),
					Entry("with DataVolume wait for VM to finish starting", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, true),
					Entry("with PersistentVolume wait for VM to finish starting", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, true),
					Entry("with Block DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock, false),
				)
			})
			DescribeTable("should add/remove volume", decorators.StorageCritical, func(
				addVolumeFunc addVolumeFunction,
				removeVolumeFunc removeVolumeFunction,
				volumeMode k8sv1.PersistentVolumeMode,
				bus v1.DiskBus,
				vmiOnly, waitToStart bool,
			) {
				verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, bus, waitToStart)
			},
				Entry("with DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusSCSI, false, false),
				Entry("with PersistentVolume immediate attach", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusSCSI, false, false),
				Entry("with DataVolume wait for VM to finish starting", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusSCSI, false, true),
				Entry("with PersistentVolume wait for VM to finish starting", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusSCSI, false, true),
				Entry("with Block DataVolume immediate attach", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock, v1.DiskBusSCSI, false, false),
				Entry("with DataVolume immediate attach (virtio)", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusVirtio, false, false),
				Entry("with PersistentVolume immediate attach (virtio)", addPVCVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, v1.DiskBusVirtio, false, false),
			)

			DescribeTable("should not add/remove volume with dry run", func(addVolumeFunc addVolumeFunction, volumeMode k8sv1.PersistentVolumeMode) {
				validateDryRun(vm, addVolumeFunc, sc, volumeMode)
			},
				Entry("with DataVolume immediate attach", addDVVolumeVM, k8sv1.PersistentVolumeFilesystem),
				Entry("with PersistentVolume immediate attach", addPVCVolumeVM, k8sv1.PersistentVolumeFilesystem),
				Entry("with Block DataVolume immediate attach", addDVVolumeVM, k8sv1.PersistentVolumeBlock),
			)

			DescribeTable("Should be able to add and remove multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode, vmiOnly bool) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, testVolumes...)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes...)
				targets := verifyHotplugAttachedAndUsable(vmi, testVolumes)
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
				Entry("with VMs", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem, false),
				Entry("with VMs and block", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock, false),
			)

			DescribeTable("Should be able to add and remove and re-add multiple volumes", func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode) {
				addRemoveReAddTest(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode)
			},
				Entry("with VMs", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem),
				Entry(" with VMs and block", Serial, addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock),
			)

			It("should allow to hotplug 75 volumes simultaneously", decorators.LargeStoragePoolRequired, func() {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				const howManyVolumes = 75
				var wg sync.WaitGroup

				dvNames := make([]string, howManyVolumes)
				testVolumes := make([]string, howManyVolumes)
				dvReadyChannel := make(chan int, howManyVolumes)

				wg.Add(howManyVolumes)
				for i := 0; i < howManyVolumes; i++ {
					testVolumes[i] = fmt.Sprintf("volume%d", i)

					By("Creating Volume" + testVolumes[i])
					go func(volumeNo int) {
						defer GinkgoRecover()
						defer wg.Done()

						dv := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeFilesystem)
						dvNames[volumeNo] = dv.Name
						dvReadyChannel <- volumeNo
					}(i)
				}

				go func() {
					wg.Wait()
					close(dvReadyChannel)
				}()

				for i := range dvReadyChannel {
					By("Adding volume " + strconv.Itoa(i) + " to running VM, dv name:" + dvNames[i])
					addDVVolumeVM(vm.Name, vm.Namespace, testVolumes[i], dvNames[i], v1.DiskBusSCSI, false, "")
				}

				By(verifyingVolumeDiskInVM)
				verifyVolumeAndDiskVMAdded(virtClient, vm, testVolumes[:len(testVolumes)-1]...)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", testVolumes...)
			})

			It("[QUARANTINE] should permanently add hotplug volume when added to VM, but still unpluggable after restart", decorators.Quarantine, func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(addingVolumeRunningVM)
				addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dvBlock.Name, v1.DiskBusSCSI, false, "")
				By(verifyingVolumeDiskInVM)
				verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
				verifySingleAttachmentPod(vmi)

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUsable(vmi, []string{"testvolume"})
				Expect(targets).To(HaveLen(1))

				By("stopping VM")
				vm = libvmops.StopVirtualMachine(vm)

				By("starting VM")
				vm = libvmops.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume is hotpluggable after a restart")
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

				By("Verifying the hotplug device is auto-mounted during booting")
				getVmiConsoleAndLogin(vmi)
				verifyVolumeAccessible(vmi, targets[0])

				By("Remove volume from a running VM")
				removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying that the hotplugged volume can be unplugged after a restart")
				verifyVolumeNolongerAccessible(vmi, targets[0])
			})

			It("should reject hotplugging a volume with the same name as an existing volume", func() {
				dvBlock := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
				dvBlock := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)

				By(addingVolumeRunningVM)
				addPVCVolumeVM(vmi.Name, vmi.Namespace, "testvolume", dvBlock.Name, v1.DiskBusSCSI, false, "")

				By(verifyingVolumeDiskInVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
				verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

				By(addingVolumeAgain)
				err = virtClient.VirtualMachine(vmi.Namespace).AddVolume(context.Background(), vmi.Name, getAddVolumeOptions(dvBlock.Name, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvBlock.Name,
					},
				}, false, false, ""))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Unable to add volume source [%s] because it already exists", dvBlock.Name)))
			})

			DescribeTable("should reject removing a volume", func(volName, expectedErr string) {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
				dvBlock := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)
				dvFileSystem := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeFilesystem)

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
				removeVolumeVM(vmi.Name, vmi.Namespace, "block", false)
				removeVolumeVM(vmi.Name, vmi.Namespace, "fs", false)

				for i := 0; i < 2; i++ {
					verifyVolumeNolongerAccessible(vmi, targets[i])
				}
			})
		})

		Context("VMI migration", decorators.RequiresRWXBlock, func() {
			var (
				vmi *v1.VirtualMachineInstance
				sc  string
			)

			containerDiskVMIFunc := func() *v1.VirtualMachineInstance {
				return libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
			}
			persistentDiskVMIFunc := func() *v1.VirtualMachineInstance {
				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize(cd.CirrosVolumeSize),
						libdv.StorageWithReadWriteManyAccessMode(),
						libdv.StorageWithBlockVolumeMode(),
					),
				)
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi := libvmi.New(
					libvmi.WithDataVolume("disk0", dataVolume.Name),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					// Stir things up, /dev/urandom access will be needed
					libvmi.WithRng(),
				)

				return vmi
			}

			BeforeEach(func() {
				exists := false
				sc, exists = libstorage.GetRWXBlockStorageClass()
				if !exists {
					Fail("Fail test when RWXBlock storage class is not present")
				}
			})

			DescribeTable("should allow live migration with attached hotplug volumes", decorators.StorageCritical, func(vmiFunc func() *v1.VirtualMachineInstance) {
				vmi = vmiFunc()
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)
				volumeName := "testvolume"
				volumeMode := k8sv1.PersistentVolumeBlock
				addVolumeFunc := addDVVolumeVMI
				removeVolumeFunc := removeVolumeVMI
				dv := createDataVolumeAndWaitForImport(sc, volumeMode)

				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(240),
				)
				By("Verifying the VMI is migratable")
				Eventually(matcher.ThisVMI(vmi), 90*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceIsMigratable))

				By("Adding volume to running VMI")
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, v1.DiskBusSCSI, false, "")
				By("Verifying the volume and disk are in the VMI")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", volumeName)

				By("Verifying the VMI is still migratable")
				Eventually(matcher.ThisVMI(vmi), 90*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceIsMigratable))

				By("Verifying the volume is attached and usable")
				getVmiConsoleAndLogin(vmi)
				targets := verifyHotplugAttachedAndUsable(vmi, []string{volumeName})
				Expect(targets).To(HaveLen(1))

				By("Starting the migration")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				sourceAttachmentPods := []string{}
				for _, volumeStatus := range vmi.Status.VolumeStatus {
					if volumeStatus.HotplugVolume != nil {
						sourceAttachmentPods = append(sourceAttachmentPods, volumeStatus.HotplugVolume.AttachPodName)
					}
				}
				Expect(sourceAttachmentPods).To(HaveLen(1))

				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				By("Verifying the volume is still accessible and usable")
				verifyVolumeAccessible(vmi, targets[0])
				verifyWriteReadData(vmi, targets[0])

				By("Verifying the source attachment pods are deleted")
				Eventually(func() error {
					_, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), sourceAttachmentPods[0], metav1.GetOptions{})
					return err
				}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

				By("Verifying the volume can be detached and reattached after migration")
				removeVolumeFunc(vmi.Name, vmi.Namespace, volumeName, false)
				verifyVolumeNolongerAccessible(vmi, targets[0])
				addVolumeFunc(vmi.Name, vmi.Namespace, volumeName, dv.Name, v1.DiskBusSCSI, false, "")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyVolumeAndDiskVMIAdded(virtClient, vmi, volumeName)
				verifyVolumeStatus(vmi, v1.VolumeReady, "", volumeName)
			},
				Entry("containerDisk VMI", containerDiskVMIFunc),
				Entry("persistent disk VMI", persistentDiskVMIFunc),
			)
		})

		Context("disk mutating sidecar", func() {
			const (
				hookSidecarImage = "example-disk-mutation-hook-sidecar"
				newDiskImgName   = "kubevirt-disk.img"
			)

			var (
				vm *v1.VirtualMachine
				dv *cdiv1.DataVolume
			)

			BeforeEach(func() {
				if !libstorage.HasCDI() {
					Fail("Fail tests when CDI is not present")
				}
			})

			AfterEach(func() {
				if vm != nil {
					err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					vm = nil
				}
			})

			DescribeTable("should be able to add and remove volumes", decorators.RequiresBlockStorage, func(addVolumeFunc addVolumeFunction, removeVolumeFunc removeVolumeFunction, volumeMode k8sv1.PersistentVolumeMode) {
				// Some permutations of this test want a filesystem on top of a block device
				sc, exists := libstorage.GetRWOBlockStorageClass()
				if !exists {
					Fail("Fail test when block storage class is not available")
				}

				var err error
				url := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)

				storageClass, foundSC := libstorage.GetRWOFileSystemStorageClass()
				if !foundSC {
					Fail("Fail test when Filesystem storage is not present")
				}

				dv = libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithRegistryURLSource(url),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(storageClass),
						libdv.StorageWithVolumeSize("500Mi"),
						libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
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
				renameImgFile(pvc, newDiskImgName)

				By("start VM with disk mutation sidecar")
				hookSidecarsValue := fmt.Sprintf(`[{"args": ["--version", "v1alpha2"], "image": "%s", "imagePullPolicy": "IfNotPresent"}]`,
					libregistry.GetUtilityImageFromRegistry(hookSidecarImage))
				vmi := libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace,
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData("#!/bin/bash\necho 'hello'\n")),
					libvmi.WithAnnotation("hooks.kubevirt.io/hookSidecars", hookSidecarsValue),
					libvmi.WithAnnotation("diskimage.vm.kubevirt.io/bootImageName", newDiskImgName),
				)

				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

				verifyAttachDetachVolume(vm, addVolumeFunc, removeVolumeFunc, sc, volumeMode, v1.DiskBusSCSI, true)
			},
				Entry("with DataVolume and running VM", addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeFilesystem),
				Entry(" with Block DataVolume immediate attach", Serial, addDVVolumeVM, removeVolumeVM, k8sv1.PersistentVolumeBlock),
			)
		})
	})

	Context("delete attachment pod several times", decorators.RequiresRWXBlock, func() {
		const quotaName = "pod-limit-quota"
		var (
			vm       *v1.VirtualMachine
			hpvolume *cdiv1.DataVolume
		)

		BeforeEach(func() {
			if !libstorage.HasCDI() {
				Fail("Fail tests when CDI is not present")
			}
			_, foundSC := libstorage.GetRWXBlockStorageClass()
			if !foundSC {
				Fail("Fail test when block RWX storage is not present")
			}
		})

		AfterEach(func() {
			if vm != nil {
				err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
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
				GracePeriodSeconds: pointer.P(int64(0)),
				PropagationPolicy:  &foreGround,
			})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), podName, metav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		}

		createPodLimitingResourceQuota := func(namespace string) {
			rq := &k8sv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      quotaName,
					Namespace: namespace,
				},
				Spec: k8sv1.ResourceQuotaSpec{
					Hard: k8sv1.ResourceList{
						k8sv1.ResourcePods: resource.MustParse("1"), // Only 1 pod is allowed which is the virt-launcher
					},
				},
			}

			_, err := virtClient.CoreV1().ResourceQuotas(namespace).Create(context.Background(), rq, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		DescribeTable("should remain active", func(limitHotplugPodCreation bool, hotplugPodDeletionTimes int) {
			checkVolumeName := "checkvolume"
			volumeMode := k8sv1.PersistentVolumeBlock
			addVolumeFunc := addDVVolumeVM
			var err error
			storageClass, _ := libstorage.GetRWXBlockStorageClass()

			blankDv := func() *cdiv1.DataVolume {
				return libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithBlankImageSource(),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(storageClass),
						libdv.StorageWithVolumeSize(cd.BlankVolumeSize),
						libdv.StorageWithReadWriteManyAccessMode(),
						libdv.StorageWithVolumeMode(volumeMode),
					),
				)
			}
			vmi := libvmifact.NewCirros()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
			By("creating blank hotplug volumes")
			hpvolume = blankDv()
			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(hpvolume.Namespace).Create(context.Background(), hpvolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("waiting for the dv import to pvc to finish")
			libstorage.EventuallyDV(dv, 180, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("hotplugging the volume check volume")
			addVolumeFunc(vmi.Name, vmi.Namespace, checkVolumeName, hpvolume.Name, v1.DiskBusSCSI, false, "")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, checkVolumeName)
			verifyVolumeStatus(vmi, v1.VolumeReady, "", checkVolumeName)
			getVmiConsoleAndLogin(vmi)

			By("verifying the volume is usable and creating some data on it")
			verifyHotplugAttachedAndUsable(vmi, []string{checkVolumeName})
			targets := getTargetsFromVolumeStatus(vmi, checkVolumeName)
			Expect(targets).ToNot(BeEmpty())
			verifyWriteReadData(vmi, targets[0])
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("deleting the attachment pod, try to make the currently attached volume break")
			if limitHotplugPodCreation {
				createPodLimitingResourceQuota(vmi.Namespace)
			}
			for range hotplugPodDeletionTimes {
				deleteAttachmentPod(vmi)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			By("verifying the volume has not been disturbed in the VM")
			targets = getTargetsFromVolumeStatus(vmi, checkVolumeName)
			Expect(targets).ToNot(BeEmpty())
			verifyWriteReadData(vmi, targets[0])

			if limitHotplugPodCreation {
				By("verifying the VM state has not changed to paused")
				Consistently(matcher.ThisVM(vm), 30*time.Second, 1*time.Second).Should(Not(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused)))
			}
		},
			Entry("when deleting the hotplug pod and turning it unschedulable via a ResourceQuota", true, 1),
			Entry("when repeatedly deleting the hotplug pod and letting it reschedule", false, 10),
		)
	})

	Context("with limit range in namespace", decorators.RequiresRWXBlock, func() {
		var (
			sc                         string
			lr                         *k8sv1.LimitRange
			orgCdiResourceRequirements *k8sv1.ResourceRequirements
			originalConfig             v1.KubeVirtConfiguration
		)

		createVMWithRatio := func(memRatio, cpuRatio float64) *v1.VirtualMachine {
			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))

			memLimit := int64(1024 * 1024 * 128) //128Mi
			memRequest := int64(math.Ceil(float64(memLimit) / memRatio))
			memRequestQuantity := resource.NewScaledQuantity(memRequest, 0)
			memLimitQuantity := resource.NewScaledQuantity(memLimit, 0)
			cpuLimit := int64(1)
			cpuRequest := int64(math.Ceil(float64(cpuLimit) / cpuRatio))
			cpuRequestQuantity := resource.NewScaledQuantity(cpuRequest, 0)
			cpuLimitQuantity := resource.NewScaledQuantity(cpuLimit, 0)
			vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = *memRequestQuantity
			vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = *cpuRequestQuantity
			vm.Spec.Template.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
			vm.Spec.Template.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = *memLimitQuantity
			vm.Spec.Template.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = *cpuLimitQuantity
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
			return vm
		}

		updateCDIResourceRequirements := func(requirements *k8sv1.ResourceRequirements) {
			if !libstorage.HasCDI() {
				Fail("Test requires CDI CR to be available")
			}
			orgCdiConfig, err := virtClient.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), "config", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			cdi := libstorage.GetCDI(virtClient)
			orgCdiResourceRequirements = cdi.Spec.Config.PodResourceRequirements
			patchSet := patch.New(
				patch.WithAdd(fmt.Sprintf("/spec/config/podResourceRequirements"), requirements),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.CdiClient().CdiV1beta1().CDIs().Patch(context.Background(), cdi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
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
			updateCDIResourceRequirements(&k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    *cpuRequestQuantity,
					k8sv1.ResourceMemory: *memRequestQuantity,
				},
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    cpuLimitQuantity,
					k8sv1.ResourceMemory: memLimitQuantity,
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
			resources := v1.ResourceRequirementsWithoutClaims{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    *cpuRequestQuantity,
					k8sv1.ResourceMemory: *memRequestQuantity,
				},
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    cpuLimitQuantity,
					k8sv1.ResourceMemory: memLimitQuantity,
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
			kvconfig.UpdateKubeVirtConfigValueAndWait(*config)
		}

		createLimitRangeInNamespace := func(namespace string, memRatio, cpuRatio float64) {
			lr = &k8sv1.LimitRange{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-lr", namespace),
				},
				Spec: k8sv1.LimitRangeSpec{
					Limits: []k8sv1.LimitRangeItem{
						{
							Type: k8sv1.LimitTypeContainer,
							MaxLimitRequestRatio: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse(fmt.Sprintf("%f", memRatio)),
								k8sv1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%f", cpuRatio)),
							},
							Max: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("1Gi"),
								k8sv1.ResourceCPU:    resource.MustParse("1"),
							},
							Min: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("1Mi"),
								k8sv1.ResourceCPU:    resource.MustParse("1m"),
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
				Fail("Fail test when RWXBlock storage class is not present")
			}
			originalConfig = *libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
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
			kvconfig.UpdateKubeVirtConfigValueAndWait(originalConfig)
		})

		// Needs to be serial since I am putting limit range on namespace
		DescribeTable("hotplug volume should have mem ratio same as VMI with limit range applied", func(memRatio, cpuRatio float64) {
			updateCDIToRatio(memRatio, cpuRatio)
			updateKubeVirtToRatio(memRatio, cpuRatio)
			createLimitRangeInNamespace(testsuite.NamespaceTestDefault, memRatio, cpuRatio)
			vm := createVMWithRatio(memRatio, cpuRatio)
			dv := createDataVolumeAndWaitForImport(sc, k8sv1.PersistentVolumeBlock)

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")
			By(verifyingVolumeDiskInVM)
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
			verifySingleAttachmentPod(vmi)
			By("Verifying request/limit ratio on attachment pod")
			podList, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			var virtlauncherPod, attachmentPod k8sv1.Pod
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
			_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[test_id:10002]1 to 1 cpu and mem ratio", Serial, float64(1), float64(1)),
			Entry("[test_id:10003]1 to 1 mem ratio, 4 to 1 cpu ratio", Serial, float64(1), float64(4)),
			Entry("[test_id:10004]2 to 1 mem ratio, 4 to 1 cpu ratio", Serial, float64(2), float64(4)),
			Entry("[test_id:10005]2.25 to 1 mem ratio, 5.75 to 1 cpu ratio", Serial, float64(2.25), float64(5.75)),
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
			pvNode := libstorage.CreateHostPathPvWithSizeAndStorageClass(customHostPath, testsuite.GetTestNamespace(nil), hotplugPvPath, "1Gi", storageClassHostPath)
			libstorage.CreatePVC(customHostPath, testsuite.GetTestNamespace(nil), "1Gi", storageClassHostPath, false)

			opts := []libvmi.Option{}
			if pvNode != "" {
				opts = append(opts, libvmi.WithNodeSelectorFor(pvNode))
			}
			vmi := libvmifact.NewCirros(opts...)

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		})

		AfterEach(func() {
			deletePvAndPvc(fmt.Sprintf("%s-disk-for-tests", customHostPath))
			libstorage.DeleteStorageClass(storageClassHostPath)
		})

		It("should attach a hostpath based volume to running VM", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			name := fmt.Sprintf("disk-%s", customHostPath)
			addPVCVolumeVM(vm.Name, vm.Namespace, "testvolume", name, v1.DiskBusSCSI, false, "")

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	Context("iothreads", func() {
		var (
			vm *v1.VirtualMachine
		)

		createVM := func(policy v1.IOThreadsPolicy) {
			vmi := libvmifact.NewCirros()
			vmi.Spec.Domain.IOThreadsPolicy = &policy
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		}

		addDVVolumeWithDedicatedIO := func(name, namespace, volumeName, claimName string) {
			avr := getAddVolumeOptions(volumeName, v1.DiskBusVirtio, &v1.HotplugVolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: claimName,
				},
			}, false, false, "")
			avr.Disk.DedicatedIOThread = pointer.P(true)

			addVolumeVMWithSource(name, namespace, avr)
		}

		verifyDedicatedIO := func(vmiSpec *v1.VirtualMachineInstanceSpec, volumeName string) {
			Expect(vmiSpec.Domain.Devices.Disks).ToNot(BeEmpty())
			for _, disk := range vmiSpec.Domain.Devices.Disks {
				if disk.Name == volumeName {
					Expect(disk.Disk).ToNot(BeNil())
					Expect(disk.Disk.Bus).To(Equal(v1.DiskBusVirtio))
					Expect(disk.DedicatedIOThread).ToNot(BeNil())
					Expect(*disk.DedicatedIOThread).To(BeTrue())
					return
				}
			}
			Fail(fmt.Sprintf("Disk %s not found in VMI spec", volumeName))
		}

		DescribeTable("should allow adding and removing hotplugged volumes", func(dedicatedIO bool) {
			threadPolicy := v1.IOThreadsPolicyShared
			if dedicatedIO {
				threadPolicy = v1.IOThreadsPolicyAuto
			}
			createVM(threadPolicy)

			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Fail("Fail no filesystem storage class available")
			}

			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithVolumeSize(cd.BlankVolumeSize)),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			if dedicatedIO {
				addDVVolumeWithDedicatedIO(vm.Name, vm.Namespace, "testvolume", dv.Name)
			} else {
				addDVVolumeVM(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")
			}

			By(verifyingVolumeDiskInVM)
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if dedicatedIO {
				verifyDedicatedIO(&vm.Spec.Template.Spec, "testvolume")
			}

			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if dedicatedIO {
				verifyDedicatedIO(&vmi.Spec, "testvolume")
			}

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		},
			Entry("without dedicated IO and shared policy", false),
			Entry("with dedicated IO and auto policy", true),
		)
	})

	Context("hostpath-separate-device", func() {
		var (
			vm *v1.VirtualMachine
		)

		BeforeEach(func() {
			libstorage.CreateAllSeparateDeviceHostPathPvs(customHostPath, testsuite.GetTestNamespace(nil))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		})

		AfterEach(func() {
			libstorage.DeleteAllSeparateDeviceHostPathPvs()
		})

		It("should attach a hostpath based volume to running VM", func() {
			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(libstorage.StorageClassHostPathSeparateDevice),
					libdv.StorageWithVolumeSize(cd.BlankVolumeSize),
				),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.TODO(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(240),
			)

			By(addingVolumeRunningVM)
			addPVCVolumeVM(vm.Name, vm.Namespace, "testvolume", dv.Name, v1.DiskBusSCSI, false, "")

			By(verifyingVolumeDiskInVM)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")

			getVmiConsoleAndLogin(vmi)
			targets := getTargetsFromVolumeStatus(vmi, "testvolume")
			verifyVolumeAccessible(vmi, targets[0])
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})

	// Some of the functions used here don't behave well in parallel (CreateSCSIDisk).
	// The device is created directly on the node and the addition and removal
	// of the scsi_debug kernel module could create flakiness in parallel.
	Context("Hotplug LUN disk", Serial, func() {
		var (
			nodeName, address, device string
			pvc                       *k8sv1.PersistentVolumeClaim
			pv                        *k8sv1.PersistentVolume
			vm                        *v1.VirtualMachine
		)

		BeforeEach(func() {
			nodeName = libnode.GetNodeNameWithHandler()
			address, device = CreateSCSIDisk(nodeName, []string{})
			By(fmt.Sprintf("Create PVC with SCSI disk %s", device))
			pv, pvc, err = CreatePVandPVCwithSCSIDisk(nodeName, device, testsuite.NamespaceTestDefault, "scsi-disks", "scsipv", "scsipvc")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			// Delete the scsi disk
			RemoveSCSIDisk(nodeName, address)
			Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			err := deleteVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("on an offline VM", func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), libvmi.NewVirtualMachine(libvmifact.NewCirros()), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Adding test volumes")
			pv2, pvc2, err := CreatePVandPVCwithSCSIDisk(nodeName, device, testsuite.NamespaceTestDefault, "scsi-disks-test2", "scsipv2", "scsipvc2")
			Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for scsi disk")

			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions(testNewVolume1, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				}},
			}, false, true, ""))
			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions(testNewVolume2, v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
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
			vmi := libvmifact.NewCirros(libvmi.WithNodeSelectorFor(nodeName))

			vm, err = virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways)), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

			By(addingVolumeRunningVM)
			addVolumeVMWithSource(vm.Name, vm.Namespace, getAddVolumeOptions("testvolume", v1.DiskBusSCSI, &v1.HotplugVolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				}},
			}, false, true, ""))
			By(verifyingVolumeDiskInVM)
			verifyVolumeAndDiskVMAdded(virtClient, vm, "testvolume")

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			verifyVolumeAndDiskVMIAdded(virtClient, vmi, "testvolume")
			verifyVolumeStatus(vmi, v1.VolumeReady, "", "testvolume")
			getVmiConsoleAndLogin(vmi)
			targets := verifyHotplugAttachedAndUsable(vmi, []string{"testvolume"})
			verifySingleAttachmentPod(vmi)
			By(removingVolumeFromVM)
			removeVolumeVM(vm.Name, vm.Namespace, "testvolume", false)
			By(verifyingVolumeNotExist)
			verifyVolumeAndDiskVMRemoved(vm, "testvolume")
			verifyVolumeNolongerAccessible(vmi, targets[0])
		})
	})
}))

func verifyVolumeAndDiskVMAdded(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	log.Log.Infof("Checking %d volumes", len(volumeNames))
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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

func renameImgFile(pvc *k8sv1.PersistentVolumeClaim, newName string) {
	args := []string{fmt.Sprintf("mv %s %s && ls -al %s", filepath.Join(libstorage.DefaultPvcMountPath, "disk.img"), filepath.Join(libstorage.DefaultPvcMountPath, newName), libstorage.DefaultPvcMountPath)}

	By("renaming disk.img")
	pod := libstorage.RenderPodWithPVC("rename-disk-img-pod", []string{"/bin/bash", "-c"}, args, pvc)

	virtClient := kubevirt.Client()
	pod, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(matcher.ThisPod(pod), 120).Should(matcher.BeInPhase(k8sv1.PodSucceeded))
}
