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
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	verifierPodName                  = "verifier"
	noPreviousOutput                 = ""
	noClaimName                      = ""
	waitMemoryDumpRequest            = "waiting on memory dump request in vm status"
	waitMemoryDumpPvcVolume          = "waiting on memory dump pvc in vm"
	waitMemoryDumpRequestRemove      = "waiting on memory dump request to be remove from vm status"
	waitMemoryDumpPvcVolumeRemove    = "waiting on memory dump pvc to be remove from vm volumes"
	waitMemoryDumpCompletion         = "waiting on memory dump completion in vm, phase: %s"
	waitMemoryDumpInProgress         = "waiting on memory dump in progress in vm, phase: %s"
	waitVMIMemoryDumpPvcVolume       = "waiting memory dump not to be in vmi volumes list"
	waitVMIMemoryDumpPvcVolumeStatus = "waiting memory dump not to be in vmi volumeStatus list"
	memoryDumpPVCSize                = "500Mi"
)

type memoryDumpFunction func(name, namespace, claimNames string)
type removeMemoryDumpFunction func(name, namespace string)

var _ = Describe(SIG("Memory dump", func() {
	var (
		virtClient         kubecli.KubevirtClient
		memoryDumpPVCName  string
		memoryDumpPVCName2 string
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		memoryDumpPVCName = "fs-pvc" + rand.String(5)
		memoryDumpPVCName2 = "fs-pvc2" + rand.String(5)
	})

	createAndStartVM := func() *v1.VirtualMachine {
		By("Creating VirtualMachine")
		vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false
			}
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vm.Status.Ready && vmi.Status.Phase == v1.Running
		}, 180*time.Second, time.Second).Should(BeTrue())

		return vm
	}

	waitDeleted := func(deleteFunc func() error) {
		Eventually(func() error {
			return deleteFunc()
		}, 180*time.Second, time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
	}

	deleteVirtualMachine := func(vm *v1.VirtualMachine) {
		waitDeleted(func() error {
			return virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
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
			updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
		executorPod := createExecutorPodWithPVC(verifierPodName, memoryDumpPVC)
		lsOutput, err := exec.ExecuteCommandOnPod(
			executorPod,
			executorPod.Spec.Containers[0].Name,
			[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s", libstorage.DefaultPvcMountPath)},
		)
		lsOutput = strings.TrimSpace(lsOutput)
		log.Log.Infof("%s", lsOutput)
		Expect(err).ToNot(HaveOccurred())
		wcOutput, err := exec.ExecuteCommandOnPod(
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

	removeMemoryDumpVMSubresource := func(vmName, namespace string) {
		Eventually(func() error {
			return virtClient.VirtualMachine(namespace).RemoveMemoryDump(context.Background(), vmName)
		}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
	}

	createMemoryDumpAndVerify := func(vm *v1.VirtualMachine, pvcName, previousOutput string, memoryDumpFunc memoryDumpFunction) string {
		By("Running memory dump")
		memoryDumpFunc(vm.Name, vm.Namespace, pvcName)

		waitAndVerifyMemoryDumpCompletion(vm, pvcName)
		verifyMemoryDumpNotOnVMI(vm, pvcName)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespaceTestDefault).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return verifyMemoryDumpOutput(pvc, previousOutput, false)
	}

	removeMemoryDumpAndVerify := func(vm *v1.VirtualMachine, pvcName, previousOutput string, removeMemoryDumpFunc removeMemoryDumpFunction) {
		By("Running remove memory dump")
		removeMemoryDumpFunc(vm.Name, vm.Namespace)
		waitAndVerifyMemoryDumpDissociation(vm, pvcName)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespaceTestDefault).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		// Verify the content is still on the pvc
		verifyMemoryDumpOutput(pvc, previousOutput, true)
	}

	Context("Memory dump with existing PVC", func() {
		var (
			vm             *v1.VirtualMachine
			memoryDumpPVC  *k8sv1.PersistentVolumeClaim
			memoryDumpPVC2 *k8sv1.PersistentVolumeClaim
			sc             string
		)
		const (
			numPVs = 2
		)

		BeforeEach(func() {
			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Fail("Fail no filesystem storage class available")
			}
			libstorage.CheckNoProvisionerStorageClassPVs(sc, numPVs)

			vm = createAndStartVM()

			memoryDumpPVC = libstorage.CreateFSPVC(memoryDumpPVCName, testsuite.GetTestNamespace(vm), memoryDumpPVCSize, nil)
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
		})

		It("[test_id:8499]Should be able to get and remove memory dump calling endpoint directly", func() {
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpVMSubresource)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVMSubresource)
		})

		It("[test_id:8502]Run multiple memory dumps", decorators.StorageCritical, func() {
			previousOutput := ""
			for i := 0; i < 3; i++ {
				By("Running memory dump number: " + strconv.Itoa(i))
				if i > 0 {
					// Running memory dump to the same pvc doesnt require claim name
					memoryDumpVMSubresource(vm.Name, vm.Namespace, noClaimName)
				} else {
					memoryDumpVMSubresource(vm.Name, vm.Namespace, memoryDumpPVCName)
				}
				waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
				verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
				previousOutput = verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, false)
			}

			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVMSubresource)
		})

		It("[test_id:8503]Run memory dump to a pvc, remove and run memory dump to different pvc", func() {
			By("Running memory dump to pvc: " + memoryDumpPVCName)
			previousOutput := createMemoryDumpAndVerify(vm, memoryDumpPVCName, noPreviousOutput, memoryDumpVMSubresource)

			By("Running remove memory dump to pvc: " + memoryDumpPVCName)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVMSubresource)

			memoryDumpPVC2 = libstorage.CreateFSPVC(memoryDumpPVCName2, testsuite.GetTestNamespace(vm), memoryDumpPVCSize, nil)
			By("Running memory dump to other pvc: " + memoryDumpPVCName2)
			previousOutput = createMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, memoryDumpVMSubresource)

			By("Running remove memory dump to second pvc: " + memoryDumpPVCName2)
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName2, previousOutput, removeMemoryDumpVMSubresource)
		})

		It("[test_id:8506]Run memory dump, stop vm and remove memory dump", func() {
			By("Running memory dump")
			memoryDumpVMSubresource(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Stopping VM")
			vm = libvmops.StopVirtualMachine(vm)

			// verify the output is still the same even when vm is stopped
			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput = verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)

			By("Running remove memory dump")
			removeMemoryDumpAndVerify(vm, memoryDumpPVCName, previousOutput, removeMemoryDumpVMSubresource)
		})

		It("[test_id:8515]Run memory dump, stop vm start vm", func() {
			By("Running memory dump")
			memoryDumpVMSubresource(vm.Name, vm.Namespace, memoryDumpPVCName)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			previousOutput := verifyMemoryDumpOutput(memoryDumpPVC, "", false)

			By("Stopping VM")
			vm = libvmops.StopVirtualMachine(vm)
			By("Starting VM")
			vm = libvmops.StartVirtualMachine(vm)

			waitAndVerifyMemoryDumpCompletion(vm, memoryDumpPVCName)
			// verify memory dump didnt reappeared in the VMI
			verifyMemoryDumpNotOnVMI(vm, memoryDumpPVCName)
			verifyMemoryDumpOutput(memoryDumpPVC, previousOutput, true)
		})

		It("[test_id:8501]Run memory dump with pvc too small should fail", func() {
			By("Trying to get memory dump with small pvc")
			memoryDumpPVC2 = libstorage.CreateFSPVC(memoryDumpPVCName2, testsuite.GetTestNamespace(vm), "200Mi", nil)
			Eventually(func() error {
				return virtClient.VirtualMachine(vm.Namespace).MemoryDump(context.Background(), vm.Name, &v1.VirtualMachineMemoryDumpRequest{
					ClaimName: memoryDumpPVC2.Name,
				})
			}, 10*time.Second, 2*time.Second).Should(MatchError(ContainSubstring("should be bigger then")))
		})

		It("[test_id:9341]Should be able to remove memory dump while memory dump is stuck", func() {
			By("create pvc with a non-existing storage-class")
			memoryDumpPVC2 = libstorage.NewPVC(memoryDumpPVCName2, memoryDumpPVCSize, "no-exist")
			memoryDumpPVC2.Namespace = vm.Namespace
			memoryDumpPVC2, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), memoryDumpPVC2, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			memoryDumpVMSubresource(vm.Name, vm.Namespace, memoryDumpPVC2.Name)

			By("Wait memory dump in progress")
			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if updatedVM.Status.MemoryDumpRequest == nil || updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpInProgress {
					return fmt.Errorf(fmt.Sprintf(waitMemoryDumpInProgress, updatedVM.Status.MemoryDumpRequest.Phase))
				}

				return nil
			}, 90*time.Second, 2*time.Second).ShouldNot(HaveOccurred())

			By("Running remove memory dump")
			removeMemoryDumpVMSubresource(vm.Name, vm.Namespace)
			waitAndVerifyMemoryDumpDissociation(vm, memoryDumpPVCName)
			memoryDumpPVC2, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), memoryDumpPVC2.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if memoryDumpPVC2.Annotations != nil {
				Expect(memoryDumpPVC2.Annotations[v1.PVCMemoryDumpAnnotation]).To(BeNil())
			}
		})
	})
}))

// createExecutorPodWithPVC creates a Pod with the passed in PVC mounted under /pvc. You can then use the executor utilities to
// run commands against the PVC through this Pod.
func createExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	pod := libstorage.RenderPodWithPVC(podName, []string{"/bin/bash", "-c", "touch /tmp/startup; while true; do echo hello; sleep 2; done"}, nil, pvc)
	pod.Spec.Containers[0].ReadinessProbe = &k8sv1.Probe{
		ProbeHandler: k8sv1.ProbeHandler{
			Exec: &k8sv1.ExecAction{
				Command: []string{"/bin/cat", "/tmp/startup"},
			},
		},
	}
	return runPodAndExpectPhase(pod, k8sv1.PodRunning)
}
