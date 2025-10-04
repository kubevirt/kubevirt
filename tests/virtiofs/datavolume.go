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

package virtiofs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
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
		checks.SkipTestIfNoFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
	})

	Context("VirtIO-FS with multiple PVCs", func() {
		pvc1 := "pvc-1" + uuid.NewString()[:6]
		pvc2 := "pvc-2" + uuid.NewString()[:6]
		createPVC := func(namespace, name string) {
			sc, foundSC := libstorage.GetAvailableRWFileSystemStorageClass()
			Expect(foundSC).To(BeTrue(), "Unable to get a FileSystem Storage Class")
			pvc := libstorage.NewPVC(name, "1Gi", sc, libstorage.WithStorageProfile())
			_, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
		}

		It("[Serial] should be successfully started and accessible", Serial, func() {
			createPVC(testsuite.NamespaceTestDefault, pvc1)
			createPVC(testsuite.NamespaceTestDefault, pvc2)

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

			vmi = libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountVirtiofsCommands)),
				libvmi.WithFilesystemPVC(pvc1),
				libvmi.WithFilesystemPVC(pvc2),
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
			)

			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By(checkingVMInstanceConsoleOut)
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvc1)
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))

			virtioFsFileTestCmd = fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvc2)
			pod, err = libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podVirtioFsFileExist, err = exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
		})
	})

	Context("VirtIO-FS with an empty PVC", func() {
		var (
			pvc            = "empty-pvc1"
			originalConfig v1.KubeVirtConfiguration
		)

		BeforeEach(func() {
			originalConfig = *libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
		})

		AfterEach(func() {
			kvconfig.UpdateKubeVirtConfigValueAndWait(originalConfig)
		})

		createHostPathPV := func(pvc, namespace string) {
			pvhostpath := filepath.Join(testsuite.HostPathBase, pvc)
			node := libstorage.CreateHostPathPv(pvc, namespace, pvhostpath)

			// We change the owner to qemu regardless of virtiofsd's privileges,
			// because the root user will be able to access the directory anyway
			nodeSelector := map[string]string{k8sv1.LabelHostname: node}
			args := []string{fmt.Sprintf(`chown 107 %s`, pvhostpath)}
			pod := libpod.RenderHostPathPod("tmp-change-owner-job", pvhostpath, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)
			pod.Spec.NodeSelector = nodeSelector
			pod, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisPod(pod), 120).Should(BeInPhase(k8sv1.PodSucceeded))
		}

		It("[Serial] should be successfully started and virtiofs could be accessed", Serial, func() {
			createHostPathPV(pvc, testsuite.NamespaceTestDefault)
			libstorage.CreateHostPathPVC(pvc, testsuite.NamespaceTestDefault, "1G")
			defer func() {
				libstorage.DeletePVC(pvc, testsuite.NamespaceTestDefault)
				libstorage.DeletePV(pvc)
			}()

			resources := v1.ResourceRequirementsWithoutClaims{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("2m"),
					k8sv1.ResourceMemory: resource.MustParse("14M"),
				},
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("101m"),
					k8sv1.ResourceMemory: resource.MustParse("81M"),
				},
			}
			config := originalConfig.DeepCopy()
			config.SupportContainerResources = []v1.SupportContainerResources{
				{
					Type:      v1.VirtioFS,
					Resources: resources,
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

			pvcName := fmt.Sprintf("disk-%s", pvc)
			virtiofsMountPath := fmt.Sprintf("/mnt/virtiofs_%s", pvcName)
			virtiofsTestFile := fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath)
			mountVirtiofsCommands := fmt.Sprintf(`#!/bin/bash
                                   mkdir %s
                                   mount -t virtiofs %s %s
                                   touch %s
                           `, virtiofsMountPath, pvcName, virtiofsMountPath, virtiofsTestFile)

			vmi = libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountVirtiofsCommands)),
				libvmi.WithFilesystemPVC(pvcName),
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By(checkingVMInstanceConsoleOut)
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", pvcName)
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
			By("Finding virt-launcher pod")
			virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			foundContainer := false
			virtiofsContainerName := fmt.Sprintf("virtiofs-%s", pvcName)
			for _, container := range virtlauncherPod.Spec.Containers {
				if container.Name == virtiofsContainerName {
					foundContainer = true
					Expect(container.Resources.Requests.Cpu().Value()).To(Equal(resources.Requests.Cpu().Value()))
					Expect(container.Resources.Requests.Memory().Value()).To(Equal(resources.Requests.Memory().Value()))
					Expect(container.Resources.Limits.Cpu().Value()).To(Equal(resources.Limits.Cpu().Value()))
					Expect(container.Resources.Limits.Memory().Value()).To(Equal(resources.Limits.Memory().Value()))
				}
			}
			Expect(foundContainer).To(BeTrue())
		})
	})

	Context("Run a VMI with VirtIO-FS and a datavolume", func() {
		var sc string

		BeforeEach(func() {
			if !libstorage.HasCDI() {
				Fail("Fail DataVolume tests when CDI is not present")
			}

			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Fail("Fail test when Filesystem storage is not present")
			}
		})

		It("[Serial] should be successfully started and virtiofs could be accessed", Serial, func() {
			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
				),
				libdv.WithNamespace(testsuite.NamespaceTestDefault),
			)

			dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestDefault).Create(context.Background(), dataVolume, metav1.CreateOptions{})
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

			vmi = libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountVirtiofsCommands)),
				libvmi.WithFilesystemDV(dataVolume.Name),
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
			)

			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
			// import and start
			By("Waiting until the DataVolume is ready")
			libstorage.EventuallyDV(dataVolume, 500, HaveSucceeded())
			By("Waiting until the VirtualMachineInstance starts")
			vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, libwait.WithTimeout(500))
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
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podVirtioFsFileExist, err := exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", virtioFsFileTestCmd},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
		})
	})
})
