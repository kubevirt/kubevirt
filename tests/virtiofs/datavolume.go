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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virtiofs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	checkingVMInstanceConsoleOut = "Checking that the VirtualMachineInstance console has expected output"
)

var _ = Describe("[sig-storage] virtiofs", decorators.SigStorage, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var vmi *virtv1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		vmi = nil
	})

	Context("VirtIO-FS with multiple PVCs", func() {
		pvc1 := "pvc-1"
		pvc2 := "pvc-2"
		createPVC := func(name string) {
			sc, _ := libstorage.GetRWXFileSystemStorageClass()
			pvc := libstorage.NewPVC(name, "1Gi", sc)
			_, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespacePrivileged).Create(context.Background(), pvc, metav1.CreateOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
		}

		BeforeEach(func() {
			checks.SkipTestIfNoFeatureGate(virtconfig.VirtIOFSGate)
			createPVC(pvc1)
			createPVC(pvc2)
		})

		AfterEach(func() {
			libstorage.DeletePVC(pvc1, testsuite.NamespacePrivileged)
			libstorage.DeletePVC(pvc2, testsuite.NamespacePrivileged)
		})

		DescribeTable("should be successfully started and accessible", func(option1, option2 libvmi.Option) {

			virtiofsMountPath := func(pvcName string) string { return fmt.Sprintf("/mnt/virtiofs_%s", pvcName) }
			virtiofsTestFile := func(virtiofsMountPath string) string { return fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath) }
			mountVirtiofsCommands := fmt.Sprintf(`#!/bin/bash
                                   mkdir %s
                                   mount -t virtiofs %s %s
                                   touch %s

								   mkdir %s
                                   mount -t virtiofs %s %s
                                   touch %s
                           `, virtiofsMountPath(pvc1), pvc1, virtiofsMountPath(pvc1), virtiofsTestFile(virtiofsMountPath(pvc1)),
				virtiofsMountPath(pvc2), pvc2, virtiofsMountPath(pvc2), virtiofsTestFile(virtiofsMountPath(pvc2)))

			vmi = libvmi.NewFedora(
				libvmi.WithCloudInitNoCloudUserData(mountVirtiofsCommands, true),
				libvmi.WithFilesystemPVC(pvc1),
				libvmi.WithFilesystemPVC(pvc2),
				libvmi.WithNamespace(testsuite.NamespacePrivileged),
				option1, option2,
			)

			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By(checkingVMInstanceConsoleOut)
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvc1)
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{tests.BinBash, "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))

			virtioFsFileTestCmd = fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvc2)
			pod = tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			podVirtioFsFileExist, err = exec.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{tests.BinBash, "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
		},
			Entry("", func(instance *virtv1.VirtualMachineInstance) {}, func(instance *virtv1.VirtualMachineInstance) {}),
			Entry("with passt enabled", libvmi.WithPasstInterfaceWithPort(), libvmi.WithNetwork(v1.DefaultPodNetwork())),
		)
	})

	Context("VirtIO-FS with an empty PVC", func() {

		var pvc = "empty-pvc1"

		BeforeEach(func() {
			checks.SkipTestIfNoFeatureGate(virtconfig.VirtIOFSGate)
			libstorage.CreateHostPathPv(pvc, testsuite.NamespacePrivileged, filepath.Join(testsuite.HostPathBase, pvc))
			libstorage.CreateHostPathPVC(pvc, testsuite.NamespacePrivileged, "1G")
		})

		AfterEach(func() {
			libstorage.DeletePVC(pvc, testsuite.NamespacePrivileged)
			libstorage.DeletePV(pvc)
		})

		It("should be successfully started and virtiofs could be accessed", func() {
			pvcName := fmt.Sprintf("disk-%s", pvc)
			virtiofsMountPath := fmt.Sprintf("/mnt/virtiofs_%s", pvcName)
			virtiofsTestFile := fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath)
			mountVirtiofsCommands := fmt.Sprintf(`#!/bin/bash
                                   mkdir %s
                                   mount -t virtiofs %s %s
                                   touch %s
                           `, virtiofsMountPath, pvcName, virtiofsMountPath, virtiofsTestFile)

			vmi = libvmi.NewFedora(
				libvmi.WithCloudInitNoCloudUserData(mountVirtiofsCommands, true),
				libvmi.WithFilesystemPVC(pvcName),
				libvmi.WithNamespace(testsuite.NamespacePrivileged),
			)
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By(checkingVMInstanceConsoleOut)
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvcName)
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{tests.BinBash, "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
		})
	})

	Context("Run a VMI with VirtIO-FS and a datavolume", func() {
		var dataVolume *cdiv1.DataVolume
		BeforeEach(func() {
			checks.SkipTestIfNoFeatureGate(virtconfig.VirtIOFSGate)
			if !libstorage.HasCDI() {
				Skip("Skip DataVolume tests when CDI is not present")
			}

			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}

			dataVolume = libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
			)
		})

		AfterEach(func() {
			libstorage.DeleteDataVolume(&dataVolume)
		})

		It("should be successfully started and virtiofs could be accessed", func() {
			dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespacePrivileged).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Waiting until the DataVolume is ready")
			if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(libstorage.Config.StorageRWOFileSystem) {
				Eventually(ThisDV(dataVolume), 30).Should(WaitForFirstConsumer())
			}

			virtiofsMountPath := fmt.Sprintf("/mnt/virtiofs_%s", dataVolume.Name)
			virtiofsTestFile := fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath)
			mountVirtiofsCommands := fmt.Sprintf(`#!/bin/bash
                                       mkdir %s
                                       mount -t virtiofs %s %s
                                       touch %s
                               `, virtiofsMountPath, dataVolume.Name, virtiofsMountPath, virtiofsTestFile)

			vmi = libvmi.NewFedora(
				libvmi.WithCloudInitNoCloudUserData(mountVirtiofsCommands, true),
				libvmi.WithFilesystemDV(dataVolume.Name),
				libvmi.WithNamespace(testsuite.NamespacePrivileged),
			)
			// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
			// import and start
			vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By(checkingVMInstanceConsoleOut)
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			By("Checking that virtio-fs is mounted")
			listVirtioFSDisk := fmt.Sprintf("ls -l %s/*disk* | wc -l\n", virtiofsMountPath)
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: listVirtioFSDisk},
				&expect.BExp{R: console.RetValue("1")},
			}, 30*time.Second)).To(Succeed(), "Should be able to access the mounted virtiofs file")

			virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", dataVolume.Name)
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{tests.BinBash, "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
		})
	})
})
