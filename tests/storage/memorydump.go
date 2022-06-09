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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/client-go/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libvmi"

	virtctl "kubevirt.io/kubevirt/pkg/virtctl/vm"
)

const (
	verifierPodName                  = "verifier"
	memoryDumpPVCName                = "fs-pvc"
	memoryDumpPVCName2               = "fs-pvc2"
	memoryDumpSmallPVCName           = "fs-pvc-small"
	virtCtlClaimName                 = "--claim-name=%s"
	waitMemoryDumpRequest            = "waiting on memory dump request in vm status"
	waitMemoryDumpPvcVolume          = "waiting on memory dump pvc in vm"
	waitMemoryDumpRequestRemove      = "waiting on memory dump request to be remove from vm status"
	waitMemoryDumpPvcVolumeRemove    = "waiting on memory dump pvc to be remove from vm volumes"
	waitMemoryDumpCompletion         = "waiting on memory dump completion in vm, phase: %s"
	waitVMIMemoryDumpPvcVolume       = "waiting memory dump not to be in vmi volumes list"
	waitVMIMemoryDumpPvcVolumeStatus = "waiting memory dump not to be in vmi volumeStatus list"
)

type memoryDumpFunction func(name, namespace, claimNames string)
type removeMemoryDumpFunction func(name, namespace string)

var _ = SIGDescribe("Memory dump", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	createVirtualMachine := func(running bool, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
		By("Creating VirtualMachine")
		vm := tests.NewRandomVirtualMachine(template, running)
		newVM, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
		Expect(err).ToNot(HaveOccurred())
		return newVM
	}

	createAndStartVM := func() *v1.VirtualMachine {
		template := libvmi.NewCirros()
		vm := createVirtualMachine(true, template)
		Eventually(func() bool {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false
			}
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vm.Status.Ready && vmi.Status.Phase == v1.Running
		}, 180*time.Second, time.Second).Should(BeTrue())

		return vm
	}

	waitDeleted := func(deleteFunc func() error) {
		Eventually(func() bool {
			err := deleteFunc()
			if errors.IsNotFound(err) {
				return true
			}
			Expect(err).ToNot(HaveOccurred())
			return false
		}, 180*time.Second, time.Second).Should(BeTrue())
	}

	deleteVirtualMachine := func(vm *v1.VirtualMachine) {
		waitDeleted(func() error {
			return virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
		})
		vm = nil
	}

	deletePod := func(pod *k8sv1.Pod) {
		waitDeleted(func() error {
			return virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		})
		pod = nil
	}

	deletePVC := func(pvc *k8sv1.PersistentVolumeClaim) {
		waitDeleted(func() error {
			return virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
		})
		pvc = nil
	}

	createMemoryDumpPVC := func(name, sc string, size resource.Quantity) *k8sv1.PersistentVolumeClaim {
		volumeMode := k8sv1.PersistentVolumeFilesystem
		createdPvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				AccessModes:      []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				VolumeMode:       &volumeMode,
				StorageClassName: &sc,
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": size,
					},
				},
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		return createdPvc
	}

	verifyMemoryDumpNotOnVMI := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			foundVolume := false
			for _, volume := range updatedVMI.Spec.Volumes {
				if volume.Name == memoryDumpPVC {
					foundVolume = true
					break
				}
			}
			if foundVolume {
				return fmt.Errorf(waitVMIMemoryDumpPvcVolume)
			}

			foundVolumeStatus := false
			for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
				if volumeStatus.Name == memoryDumpPVC {
					foundVolumeStatus = true
					break
				}
			}

			if foundVolumeStatus {
				return fmt.Errorf(waitVMIMemoryDumpPvcVolumeStatus)
			}
			return nil
		}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	waitAndVerifyMemoryDumpCompletion := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}
			if updatedVM.Status.MemoryDumpRequest == nil {
				return fmt.Errorf(waitMemoryDumpRequest)
			}

			if updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
				return fmt.Errorf(fmt.Sprintf(waitMemoryDumpCompletion, updatedVM.Status.MemoryDumpRequest.Phase))
			}

			foundPvc := false
			for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
				if volume.Name == memoryDumpPVC {
					foundPvc = true
					break
				}
			}

			if !foundPvc {
				return fmt.Errorf(waitMemoryDumpPvcVolume)
			}

			return nil
		}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	}

	waitAndVerifyMemoryDumpDissociation := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}
			if updatedVM.Status.MemoryDumpRequest != nil {
				return fmt.Errorf(waitMemoryDumpRequestRemove)
			}

			foundPvc := false
			for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
				if volume.Name == memoryDumpPVC {
					foundPvc = true
					break
				}
			}

			if foundPvc {
				return fmt.Errorf(waitMemoryDumpPvcVolumeRemove)
			}

			return nil
		}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	}

	verifyMemoryDumpOutput := func(memoryDumpPVC *k8sv1.PersistentVolumeClaim, previousOutput string, shouldEqual bool) string {
		executorPod := tests.CreateExecutorPodWithPVC(verifierPodName, memoryDumpPVC)
		lsOutput, err := tests.ExecuteCommandOnPod(
			virtClient,
			executorPod,
			executorPod.Spec.Containers[0].Name,
			[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s", libstorage.DefaultPvcMountPath)},
		)
		lsOutput = strings.TrimSpace(lsOutput)
		log.Log.Infof("%s", lsOutput)
		Expect(err).ToNot(HaveOccurred())
		wcOutput, err := tests.ExecuteCommandOnPod(
			virtClient,
			executorPod,
			executorPod.Spec.Containers[0].Name,
			[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s | wc -l", libstorage.DefaultPvcMountPath)},
		)
		wcOutput = strings.TrimSpace(wcOutput)
		log.Log.Infof("%s", wcOutput)
		Expect(err).ToNot(HaveOccurred())

		Expect(strings.Contains(lsOutput, "memory.dump")).To(BeTrue())
		// Could be that a 'lost+found' directory is in it, check if the
		// response is more then 1 then it is only 2 with `lost+found` directory
		if strings.Compare("1", wcOutput) != 0 {
			Expect(wcOutput).To(Equal("2"))
			Expect(strings.Contains(lsOutput, "lost+found")).To(BeTrue())
		}
		if previousOutput != "" && shouldEqual {
			Expect(lsOutput).To(Equal(previousOutput))
		} else {
			Expect(lsOutput).ToNot(Equal(previousOutput))
		}

		deletePod(executorPod)
		return lsOutput
	}

	memoryDumpVMSubresource := func(vmName, namespace, claimName string) {
		Eventually(func() error {
			memoryDumpRequest := &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: claimName,
			}

			return virtClient.VirtualMachine(namespace).MemoryDump(vmName, memoryDumpRequest)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	memoryDumpVirtctl := func(name, namespace, claimName string) {
		By("Invoking virtlctl memory dump")
		commandAndArgs := []string{virtctl.COMMAND_MEMORYDUMP, "get", name, virtCtlNamespace, namespace}
		if claimName != "" {
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
		}
		memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return memorydumpCommand()
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeMemoryDumpVMSubresource := func(vmName, namespace string) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).RemoveMemoryDump(vmName)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeMemoryDumpVirtctl := func(name, namespace string) {
		By("Invoking virtlctl remove memory dump")
		commandAndArgs := []string{virtctl.COMMAND_MEMORYDUMP, "remove", name, virtCtlNamespace, namespace}
		removeMemorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return removeMemorydumpCommand()
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	Context("Memory dump with existing PVC", func() {
		var (
			vm                 *v1.VirtualMachine
			memoryDumpPVC      *k8sv1.PersistentVolumeClaim
			memoryDumpPVC2     *k8sv1.PersistentVolumeClaim
			memoryDumpSmallPVC *k8sv1.PersistentVolumeClaim
			sc                 string
		)
		const (
			numPVs = 2
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip no filesystem storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)

			vm = createAndStartVM()

			size, _ := resource.ParseQuantity("500Mi")
			memoryDumpPVC = createMemoryDumpPVC(memoryDumpPVCName, sc, size)
		})

		AfterEach(func() {
			if vm != nil {
				deleteVirtualMachine(vm)
			}
			if memoryDumpPVC != nil {
				deletePVC(memoryDumpPVC)
			}
			if memoryDumpPVC2 != nil {
				deletePVC(memoryDumpPVC2)
			}
			if memoryDumpSmallPVC != nil {
				deletePVC(memoryDumpSmallPVC)
			}
		})

		DescribeTable("Should be able to get and remove memory dump", func(memoryDumpFunc memoryDumpFunction, removeMemoryDumpFunc removeMemoryDumpFunction) {
			By("Running memory dump")
			memoryDumpFunc(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Running remove memory dump")
			removeMemoryDumpFunc(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			// Verify the content is still on the pvc
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)
		},
			Entry("[test_id:8499]calling endpoint directly", memoryDumpVMSubresource, removeMemoryDumpVMSubresource),
			Entry("[test_id:8500]using virtctl", memoryDumpVirtctl, removeMemoryDumpVirtctl),
		)

		It("[test_id:8502]Run multiple memory dumps", func() {
			previousOutput := ""
			for i := 0; i < 3; i++ {
				By("Running memory dump number: " + strconv.Itoa(i))
				if i > 0 {
					memoryDumpVirtctl(vm.Name, vm.Namespace, "")
				} else {
					memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName)
				}

				waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
				verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
				previousOutput = verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, false)
			}

			By("Running remove memory dump")
			removeMemoryDumpVirtctl(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			// Verify the content is still on the pvc
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)
		})

		It("[test_id:8503]Run memory dump to a pvc, remove and run memory dump to different pvc", func() {
			By("Running memory dump to pvc: " + memoryDumpPVCName)
			memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Running remove memory dump to pvc: " + memoryDumpPVCName)
			removeMemoryDumpVirtctl(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			// Verify the content is still on the pvc
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)

			size, _ := resource.ParseQuantity("500Mi")
			memoryDumpPVC2 = createMemoryDumpPVC(memoryDumpPVCName2, sc, size)
			By("Running memory dump to other pvc: " + memoryDumpPVCName2)
			memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName2)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName2)
			verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName2)
			previousOutput = verifyMemoryDumpOutput(memoryDumpPVC2, previousOutput, false)

			By("Running remove memory dump to second pvc: " + memoryDumpPVCName2)
			removeMemoryDumpVirtctl(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName2)
			// Verify the content is still on the pvc
			verifyMemoryDumpOutput(memoryDumpPVC2, previousOutput, true)
		})

		It("[test_id:8506]Run memory dump, stop vm and remove memory dump", func() {
			By("Running memory dump")
			memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Stopping VM")
			vm = tests.StopVirtualMachine(vm)

			// verify the output is still the same even when vm is stopped
			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput = verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)

			By("Running remove memory dump")
			removeMemoryDumpVirtctl(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			// Verify the content is still on the pvc
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)
		})

		It("[test_id:8515]Run memory dump, stop vm start vm", func() {
			By("Running memory dump")
			memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Stopping VM")
			vm = tests.StopVirtualMachine(vm)
			By("Starting VM")
			vm = tests.StartVirtualMachine(vm)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			// verify memory dump didnt reappeared in the VMI
			verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)
		})

		It("[test_id:8501]Run memory dump with pvc too small should fail", func() {
			By("Trying to get memory dump with small pvc")
			size, _ := resource.ParseQuantity("200Mi")
			memoryDumpSmallPVC = createMemoryDumpPVC(memoryDumpSmallPVCName, sc, size)
			commandAndArgs := []string{virtctl.COMMAND_MEMORYDUMP, "get", vm.Name, fmt.Sprintf(virtCtlClaimName, memoryDumpSmallPVCName), virtCtlNamespace, vm.Namespace}
			memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Eventually(func() string {
				err := memorydumpCommand()
				return err.Error()
			}, 3*time.Second, 1*time.Second).Should(ContainSubstring("pvc size should be bigger then vm memory"))
		})
	})
})
