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
 * Copyright The KubeVirt Authors
 *
 */

package virtctl

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage][virtctl]Memory dump", decorators.SigStorage, func() {
	const (
		claimNameFlag   = "--claim-name"
		createClaimFlag = "--create-claim"
		outputFlag      = "--output"
		portForwardFlag = "--port-forward"
	)

	var (
		pvcName string
		vm      *v1.VirtualMachine
	)

	BeforeEach(func() {
		if _, exists := libstorage.GetRWOFileSystemStorageClass(); !exists {
			Fail("Fail no filesystem storage class available")
		}

		pvcName = "fs-pvc-" + rand.String(5)

		vm = libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		var err error
		vm, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
	})

	DescribeTable("Should be able to get and remove memory dump", func(create bool) {
		args := []string{
			claimNameFlag, pvcName,
		}
		if create {
			args = append(args, createClaimFlag)
		} else {
			libstorage.CreateFSPVC(pvcName, testsuite.GetTestNamespace(nil), "500Mi", nil)
		}

		Expect(runMemoryDumpGetCmd(vm.Name, args...)).To(Succeed())
		out := waitForMemoryDumpCompletion(vm.Name, pvcName, "", false)
		Expect(runMemoryDumpRemoveCmd(vm.Name)).To(Succeed())
		waitForMemoryDumpDeletion(vm.Name, pvcName, out, true)
	},
		Entry("[test_id:9034] when creating a PVC", true),
		Entry("with an existing PVC", false),
	)

	It("[test_id:9035]Run multiple memory dumps", func() {
		out := ""
		for i := range 3 {
			By(fmt.Sprintf("Running memory dump number: %d", i+1))
			if i > 0 {
				Expect(runMemoryDumpGetCmd(vm.Name)).To(Succeed())
			} else {
				Expect(runMemoryDumpGetCmd(vm.Name, claimNameFlag, pvcName, createClaimFlag)).To(Succeed())
			}
			out = waitForMemoryDumpCompletion(vm.Name, pvcName, out, false)
		}

		Expect(runMemoryDumpRemoveCmd(vm.Name)).To(Succeed())
		waitForMemoryDumpDeletion(vm.Name, pvcName, out, true)
	})

	It("[test_id:9036]Run memory dump to creates a pvc, remove and run memory dump to create a different pvc", func() {
		Expect(runMemoryDumpGetCmd(vm.Name, claimNameFlag, pvcName, createClaimFlag)).To(Succeed())
		out := waitForMemoryDumpCompletion(vm.Name, pvcName, "", false)
		Expect(runMemoryDumpRemoveCmd(vm.Name)).To(Succeed())
		out = waitForMemoryDumpDeletion(vm.Name, pvcName, out, true)

		pvcName2 := "fs-pvc-" + rand.String(5)
		Expect(runMemoryDumpGetCmd(vm.Name, claimNameFlag, pvcName2, createClaimFlag)).To(Succeed())
		out = waitForMemoryDumpCompletion(vm.Name, pvcName2, out, false)
		Expect(runMemoryDumpRemoveCmd(vm.Name)).To(Succeed())
		waitForMemoryDumpDeletion(vm.Name, pvcName2, out, true)
	})

	It("[test_id:9344]should create memory dump and download it", func() {
		output := filepath.Join(GinkgoT().TempDir(), "memorydump.gz")

		args := []string{
			claimNameFlag, pvcName,
			createClaimFlag,
			outputFlag, output,
		}
		if !checks.IsOpenShift() {
			args = append(args, portForwardFlag)
		}
		Expect(runMemoryDumpGetCmd(vm.Name, args...)).To(Succeed())

		vm, err := kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Status.MemoryDumpRequest.FileName).ToNot(BeNil())
		verifyMemoryDumpFile(output, *vm.Status.MemoryDumpRequest.FileName)
	})

	It("[test_id:9343]should download existing memory dump", func() {
		output := filepath.Join(GinkgoT().TempDir(), "memorydump.gz")

		Expect(runMemoryDumpGetCmd(vm.Name, claimNameFlag, pvcName, createClaimFlag)).To(Succeed())
		waitForMemoryDumpCompletion(vm.Name, pvcName, "", false)

		args := []string{
			outputFlag, output,
		}
		if !checks.IsOpenShift() {
			args = append(args, portForwardFlag)
		}
		Expect(runMemoryDumpDownloadCmd(vm.Name, args...)).To(Succeed())

		vm, err := kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Status.MemoryDumpRequest.FileName).ToNot(BeNil())
		verifyMemoryDumpFile(output, *vm.Status.MemoryDumpRequest.FileName)
	})
})

func runMemoryDumpGetCmd(name string, args ...string) error {
	_args := append([]string{
		"memory-dump", "get", name,
		"--namespace", testsuite.GetTestNamespace(nil),
	}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func runMemoryDumpDownloadCmd(name string, args ...string) error {
	_args := append([]string{
		"memory-dump", "download", name,
		"--namespace", testsuite.GetTestNamespace(nil),
	}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}
func runMemoryDumpRemoveCmd(name string, args ...string) error {
	_args := append([]string{
		"memory-dump", "remove", name,
		"--namespace", testsuite.GetTestNamespace(nil),
	}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func waitForMemoryDumpCompletion(vmName, pvcName, previousOut string, shouldEqual bool) string {
	virtClient := kubevirt.Client()

	var pvc *k8sv1.PersistentVolumeClaim
	Eventually(func(g Gomega) bool {
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		if vm.Status.MemoryDumpRequest == nil {
			return false
		}
		if vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
			return false
		}

		found := false
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.Name == pvcName {
				found = true
				break
			}
		}
		if !found {
			return false
		}

		pvc, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), pvcName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(pvc.Annotations).To(HaveKeyWithValue(v1.PVCMemoryDumpAnnotation, *vm.Status.MemoryDumpRequest.FileName))

		return true
	}, 90*time.Second, 2*time.Second).Should(BeTrue())

	return verifyMemoryDumpPVC(pvc, previousOut, shouldEqual)
}

func waitForMemoryDumpDeletion(vmName, pvcName, previousOut string, shouldEqual bool) string {
	virtClient := kubevirt.Client()

	Eventually(func(g Gomega) bool {
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		if vm.Status.MemoryDumpRequest != nil {
			return false
		}

		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.Name == pvcName {
				return false
			}
		}

		return true
	}, 90*time.Second, 2*time.Second).Should(BeTrue())

	// Expect PVC to still exist
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), pvcName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	return verifyMemoryDumpPVC(pvc, previousOut, shouldEqual)
}

func verifyMemoryDumpPVC(pvc *k8sv1.PersistentVolumeClaim, previousOut string, shouldEqual bool) string {
	virtClient := kubevirt.Client()

	pod := libstorage.RenderPodWithPVC(
		"pod-"+rand.String(5),
		[]string{"/bin/bash", "-c", "touch /tmp/startup; while true; do echo hello; sleep 2; done"},
		nil, pvc,
	)
	pod.Spec.Containers[0].ReadinessProbe = &k8sv1.Probe{
		ProbeHandler: k8sv1.ProbeHandler{
			Exec: &k8sv1.ExecAction{
				Command: []string{"/bin/cat", "/tmp/startup"},
			},
		},
	}

	pod, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 120*time.Second, 1*time.Second).Should(HaveConditionTrue(k8sv1.PodReady))

	lsOut, err := exec.ExecuteCommandOnPod(
		pod, pod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s", libstorage.DefaultPvcMountPath)},
	)
	lsOut = strings.TrimSpace(lsOut)
	Expect(err).ToNot(HaveOccurred())
	Expect(lsOut).To(ContainSubstring("memory.dump"))

	wcOut, err := exec.ExecuteCommandOnPod(
		pod, pod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s | wc -l", libstorage.DefaultPvcMountPath)},
	)
	wcOut = strings.TrimSpace(wcOut)
	Expect(err).ToNot(HaveOccurred())

	// If length is not 1 then length has to be 2 and second entry has to be 'lost+found'
	if wcOut != "1" {
		Expect(wcOut).To(Equal("2"))
		Expect(lsOut).To(ContainSubstring("lost+found"))
	}

	if shouldEqual {
		Expect(lsOut).To(Equal(previousOut))
	} else {
		Expect(lsOut).ToNot(Equal(previousOut))
	}

	err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())

	return lsOut
}

func verifyMemoryDumpFile(dumpFilePath, dumpName string) {
	extractPath := filepath.Join(GinkgoT().TempDir(), "extracted")

	dumpFile, err := os.Open(dumpFilePath)
	Expect(err).ToNot(HaveOccurred())
	defer dumpFile.Close()
	gzReader, err := gzip.NewReader(dumpFile)
	Expect(err).ToNot(HaveOccurred())
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		Expect(err).ToNot(HaveOccurred())
		switch header.Typeflag {
		case tar.TypeDir:
			Expect(os.MkdirAll(filepath.Join(extractPath, header.Name), 0750)).To(Succeed())
		case tar.TypeReg:
			extractedFile, err := os.Create(filepath.Join(extractPath, header.Name))
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(extractedFile, tarReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(extractedFile.Close()).To(Succeed())
		default:
			Fail("unknown tar header type")
		}
	}

	stat, err := os.Stat(filepath.Join(extractPath, dumpName))
	Expect(err).ToNot(HaveOccurred())
	Expect(stat.Size()).To(BeNumerically(">", 0))
}
