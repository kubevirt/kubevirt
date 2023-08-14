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
	goerrors "errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/client-go/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	commandMemoryDump                = "memory-dump"
	verifierPodName                  = "verifier"
	noPreviousOutput                 = ""
	noClaimName                      = ""
	memoryDumpSmallPVCName           = "fs-pvc-small"
	virtCtlClaimName                 = "--claim-name=%s"
	virtCtlCreate                    = "--create-claim"
	virtCtlOutputFile                = "--output=%s"
	virtCtlStorageClass              = "--storage-class=%s"
	virtCtlPortForward               = "--port-forward"
	virtCtlLocalPort                 = "--local-port=%s"
	waitMemoryDumpRequest            = "waiting on memory dump request in vm status"
	waitMemoryDumpPvcVolume          = "waiting on memory dump pvc in vm"
	waitMemoryDumpRequestRemove      = "waiting on memory dump request to be remove from vm status"
	waitMemoryDumpPvcVolumeRemove    = "waiting on memory dump pvc to be remove from vm volumes"
	waitMemoryDumpCompletion         = "waiting on memory dump completion in vm, phase: %s"
	waitMemoryDumpInProgress         = "waiting on memory dump in progress in vm, phase: %s"
	waitMemoryDumpAnnotation         = "waiting on memory dump annotation on pvc"
	waitVMIMemoryDumpPvcVolume       = "waiting memory dump not to be in vmi volumes list"
	waitVMIMemoryDumpPvcVolumeStatus = "waiting memory dump not to be in vmi volumeStatus list"
)

type memoryDumpFunction func(name, namespace, claimNames string)
type removeMemoryDumpFunction func(name, namespace string)

var _ = SIGDescribe("Memory dump", func() {
	var (
		err                error
		virtClient         kubecli.KubevirtClient
		memoryDumpPVCName  string
		memoryDumpPVCName2 string
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		memoryDumpPVCName = "fs-pvc" + rand.String(5)
		memoryDumpPVCName2 = "fs-pvc2" + rand.String(5)
	})

	createVirtualMachine := func(running bool, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
		By("Creating VirtualMachine")
		vm := tests.NewRandomVirtualMachine(template, running)
		newVM, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm)
		Expect(err).ToNot(HaveOccurred())
		return newVM
	}

	createAndStartVM := func() *v1.VirtualMachine {
		template := libvmi.NewCirros()
		vm := createVirtualMachine(true, template)
		Eventually(func() bool {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false
			}
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
			return virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
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

	verifyMemoryDumpNotOnVMI := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
		}, 90*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	waitAndVerifyMemoryDumpCompletion := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), memoryDumpPVC, metav1.GetOptions{})
			if err != nil {
				return err
			}
			Expect(pvc.GetAnnotations()).ToNot(BeNil())
			Expect(pvc.Annotations[v1.PVCMemoryDumpAnnotation]).To(Equal(*updatedVM.Status.MemoryDumpRequest.FileName))

			return nil
		}, 90*time.Second, 2*time.Second).ShouldNot(HaveOccurred())

	}

	waitAndVerifyMemoryDumpDissociation := func(vm *v1.VirtualMachine, memoryDumpPVC string) {
		Eventually(func() error {
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
		}, 90*time.Second, 2*time.Second).ShouldNot(HaveOccurred())

	}

	verifyMemoryDumpOutput := func(memoryDumpPVC *k8sv1.PersistentVolumeClaim, previousOutput string, shouldEqual bool) string {
		executorPod := tests.CreateExecutorPodWithPVC(verifierPodName, memoryDumpPVC)
		lsOutput, err := exec.ExecuteCommandOnPod(
			virtClient,
			executorPod,
			executorPod.Spec.Containers[0].Name,
			[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s", libstorage.DefaultPvcMountPath)},
		)
		lsOutput = strings.TrimSpace(lsOutput)
		log.Log.Infof("%s", lsOutput)
		Expect(err).ToNot(HaveOccurred())
		wcOutput, err := exec.ExecuteCommandOnPod(
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

			return virtClient.VirtualMachine(namespace).MemoryDump(context.Background(), vmName, memoryDumpRequest)
		}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	memoryDumpVirtctl := func(name, namespace, claimName string) {
		By("Invoking virtctl memory dump")
		commandAndArgs := []string{commandMemoryDump, "get", name, virtCtlNamespace, namespace}
		if claimName != "" {
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
		}
		memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return memorydumpCommand()
		}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	memoryDumpVirtctlCreatePVC := func(name, namespace, claimName string) {
		By("Invoking virtctl memory dump with create flag")
		commandAndArgs := []string{commandMemoryDump, "get", name, virtCtlNamespace, namespace}
		commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
		commandAndArgs = append(commandAndArgs, virtCtlCreate)
		memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			err := memorydumpCommand()
			if err != nil {
				_, getErr := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, metav1.GetOptions{})
				if getErr == nil {
					// already created the pvc can't call the memory dump command with
					// create-claim flag again
					By("Error memory dump command after claim created")
					memoryDumpVirtctl(name, namespace, claimName)
					return nil
				}
			}
			return err
		}, 20*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	memoryDumpVirtctlCreatePVCWithStorgeClass := func(name, namespace, claimName, storageClass string) {
		By("Invoking virtctl memory dump with create flag")
		commandAndArgs := []string{commandMemoryDump, "get", name, virtCtlNamespace, namespace}
		commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
		commandAndArgs = append(commandAndArgs, virtCtlCreate)
		commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlStorageClass, storageClass))
		memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			err := memorydumpCommand()
			if err != nil {
				_, getErr := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, metav1.GetOptions{})
				if getErr == nil {
					// already created the pvc can't call the memory dump command with
					// create-claim flag again
					By("Error memory dump command after claim created")
					memoryDumpVirtctl(name, namespace, claimName)
					return nil
				}
			}
			return err
		}, 20*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	removeMemoryDumpVMSubresource := func(vmName, namespace string) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).RemoveMemoryDump(context.Background(), vmName)
		}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	removeMemoryDumpVirtctl := func(name, namespace string) {
		By("Invoking virtctl remove memory dump")
		commandAndArgs := []string{commandMemoryDump, "remove", name, virtCtlNamespace, namespace}
		removeMemorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		Eventually(func() error {
			return removeMemorydumpCommand()
		}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	createMemoryDumpAndVerify := func(vm *v1.VirtualMachine, pvcName, previousOutput string, memoryDumpFunc memoryDumpFunction) string {
		By("Running memory dump")
		memoryDumpFunc(vm.Name, vm.Namespace, pvcName)

		waitAndVerifyMemoryDumpCompletion(vm, pvcName)
		verifyMemoryDumpNotOnVMI(vm, pvcName)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return verifyMemoryDumpOutput(pvc, previousOutput, false)
	}

	removeMemoryDumpAndVerify := func(vm *v1.VirtualMachine, pvcName, previousOutput string, removeMemoryDumpFunc removeMemoryDumpFunction) {
		By("Running remove memory dump")
		removeMemoryDumpFunc(vm.Name, vm.Namespace)
		waitAndVerifyMemoryDumpDissociation(vm, pvcName)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		// Verify the content is still on the pvc
		verifyMemoryDumpOutput(pvc, previousOutput, true)
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

			memoryDumpPVC = libstorage.CreateFSPVC(memoryDumpPVCName, testsuite.GetTestNamespace(vm), "500Mi", nil)
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
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpFunc)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpFunc)
		},
			Entry("[test_id:8499]calling endpoint directly", memoryDumpVMSubresource, removeMemoryDumpVMSubresource),
			Entry("[test_id:8500]using virtctl", memoryDumpVirtctl, removeMemoryDumpVirtctl),
		)

		It("[test_id:8502]Run multiple memory dumps", func() {
			previousOutput := ""
			for i := 0; i < 3; i++ {
				By("Running memory dump number: " + strconv.Itoa(i))
				if i > 0 {
					// Running memory dump to the same pvc doesnt require claim name
					memoryDumpVirtctl(vm.Name, vm.Namespace, noClaimName)
				} else {
					memoryDumpVirtctl(vm.Name, vm.Namespace, memoryDumpPVCName)
				}
				waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
				verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
				previousOutput = verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, false)
			}

			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)
		})

		It("[test_id:8503]Run memory dump to a pvc, remove and run memory dump to different pvc", func() {
			By("Running memory dump to pvc: " + memoryDumpPVCName)
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpVirtctl)

			By("Running remove memory dump to pvc: " + memoryDumpPVCName)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)

			memoryDumpPVC2 = libstorage.CreateFSPVC(memoryDumpPVCName2, testsuite.GetTestNamespace(vm), "500Mi", nil)
			By("Running memory dump to other pvc: " + memoryDumpPVCName2)
			previousOutput = createMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, memoryDumpVirtctl)

			By("Running remove memory dump to second pvc: " + memoryDumpPVCName2)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, removeMemoryDumpVirtctl)
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
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)
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
			memoryDumpSmallPVC = libstorage.CreateFSPVC(memoryDumpSmallPVCName, testsuite.GetTestNamespace(vm), "200Mi", nil)
			commandAndArgs := []string{commandMemoryDump, "get", vm.Name, fmt.Sprintf(virtCtlClaimName, memoryDumpSmallPVCName), virtCtlNamespace, vm.Namespace}
			memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Eventually(func() string {
				err := memorydumpCommand()
				return err.Error()
			}, 10*time.Second, 2*time.Second).Should(ContainSubstring("should be bigger then"))
		})
	})

	Context("Memory dump with creating a PVC", func() {
		var (
			vm *v1.VirtualMachine
			sc string
		)
		const (
			numPVs = 1
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip no filesystem storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)

			vm = createAndStartVM()
		})

		AfterEach(func() {
			if vm != nil {
				deleteVirtualMachine(vm)
			}
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), memoryDumpPVCName, metav1.GetOptions{})
			if err == nil && pvc != nil {
				deletePVC(pvc)
			}

			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), memoryDumpPVCName2, metav1.GetOptions{})
			if err == nil && pvc != nil {
				deletePVC(pvc)
			}
		})

		It("[test_id:9034]Should be able to get and remove memory dump", func() {
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpVirtctlCreatePVC)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)
		})

		It("[test_id:9035]Run multiple memory dumps", func() {
			previousOutput := ""
			for i := 0; i < 3; i++ {
				By("Running memory dump number: " + strconv.Itoa(i))
				if i > 0 {
					// Running memory dump to the same pvc doesnt require claim name
					memoryDumpVirtctl(vm.Name, vm.Namespace, noClaimName)
				} else {
					memoryDumpVirtctlCreatePVC(vm.Name, vm.Namespace, memoryDumpPVCName)
				}
				waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
				verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), memoryDumpPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				previousOutput = verifyMemoryDumpOutput(pvc, previousOutput, false)
			}

			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)
		})

		It("[test_id:9036]Run memory dump to creates a pvc, remove and run memory dump to create a different pvc", func() {
			By("Running memory dump to pvc: " + memoryDumpPVCName)
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpVirtctlCreatePVC)

			By("Running remove memory dump to pvc: " + memoryDumpPVCName)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVirtctl)

			By("Running memory dump to other pvc: " + memoryDumpPVCName2)
			previousOutput = createMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, memoryDumpVirtctlCreatePVC)

			By("Running remove memory dump to second pvc: " + memoryDumpPVCName2)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, removeMemoryDumpVirtctl)
		})

		It("[test_id:9341]Should be able to remove memory dump while memory dump is stuck", func() {
			By("create pvc with a non-existing storage-class")
			memoryDumpVirtctlCreatePVCWithStorgeClass(vm.Name, vm.Namespace, memoryDumpPVCName, "no-exist")
			By("Wait memory dump in progress")
			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}
				if updatedVM.Status.MemoryDumpRequest == nil || updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpInProgress {
					return fmt.Errorf(fmt.Sprintf(waitMemoryDumpInProgress, updatedVM.Status.MemoryDumpRequest.Phase))
				}

				return nil
			}, 90*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
			By("Running remove memory dump")
			removeMemoryDumpVirtctl(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), memoryDumpPVCName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if pvc.Annotations != nil {
				Expect(pvc.Annotations[v1.PVCMemoryDumpAnnotation]).To(BeNil())
			}
		})
	})

	Context("Memory dump with download", func() {
		var (
			vm         *v1.VirtualMachine
			outputFile string
		)
		const (
			numPVs        = 1
			tlsSecretName = "test-tls"
			defaultOutput = "/tmp/memorydump-%s.dump.gz"
		)

		memoryDumpVirtctlDownload := func(name, namespace, outputFile string) {
			By("Invoking virtctl memory dump download")
			commandAndArgs := []string{commandMemoryDump, "download", name, virtCtlNamespace, namespace}
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlOutputFile, outputFile))
			if !checks.IsOpenShift() {
				commandAndArgs = append(commandAndArgs, virtCtlPortForward)
			}
			memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Eventually(func() error {
				return memorydumpCommand()
			}, 20*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
		}

		memoryDumpVirtctlGetWithDownload := func(name, namespace, claimName, outputFile string) {
			By("Invoking virtctl memory dump")
			commandAndArgs := []string{commandMemoryDump, "get", name, virtCtlNamespace, namespace}
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlOutputFile, outputFile))
			memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Eventually(func() error {
				return memorydumpCommand()
			}, 20*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
		}

		memoryDumpVirtctlCreateWithDownload := func(name, namespace, claimName, outputFile string) {
			By("Invoking virtctl memory dump with create flag")
			commandAndArgs := []string{commandMemoryDump, "get", name, virtCtlNamespace, namespace}
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlClaimName, claimName))
			commandAndArgs = append(commandAndArgs, virtCtlCreate)
			commandAndArgs = append(commandAndArgs, fmt.Sprintf(virtCtlOutputFile, outputFile))
			if !checks.IsOpenShift() {
				targetPort := fmt.Sprintf("%d", 37548+rand.Intn(6000))
				commandAndArgs = append(commandAndArgs, virtCtlPortForward, fmt.Sprintf(virtCtlLocalPort, targetPort))
			}
			memorydumpCommand := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Eventually(func() error {
				err := memorydumpCommand()
				if err != nil {
					_, getErr := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, metav1.GetOptions{})
					if getErr == nil {
						// already created the pvc can't call the memory dump command with
						// create-claim flag again
						By("Error memory dump command after claim created")
						memoryDumpVirtctlGetWithDownload(name, namespace, claimName, outputFile)
						return nil
					}
				}
				return err
			}, 20*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
		}

		BeforeEach(func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip no filesystem storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)

			vm = createAndStartVM()
			outputFile = fmt.Sprintf(defaultOutput, rand.String(12))
		})

		AfterEach(func() {
			if vm != nil {
				deleteVirtualMachine(vm)
			}
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), memoryDumpPVCName, metav1.GetOptions{})
			if err == nil && pvc != nil {
				deletePVC(pvc)
			}
			if err := os.Remove(outputFile); err != nil && !goerrors.Is(err, os.ErrNotExist) {
				Fail(err.Error())
			}
		})

		It("[test_id:9344]should create memory dump and download it", func() {
			memoryDumpVirtctlCreateWithDownload(vm.Name, vm.Namespace, memoryDumpPVCName, outputFile)
			//Check the outputFile was created
			_, err = os.Stat(outputFile)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:9343]should download existing memory dump", func() {
			memoryDumpVirtctlCreatePVC(vm.Name, vm.Namespace, memoryDumpPVCName)
			memoryDumpVirtctlDownload(vm.Name, vm.Namespace, outputFile)
			//Check the outputFile was created
			_, err = os.Stat(outputFile)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
