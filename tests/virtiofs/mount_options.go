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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] virtiofs mount options", decorators.SigStorage, decorators.VirtioFS, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		checks.FailTestIfNoFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
	})

	setupHostPathPVCWithContent := func(pvcBaseName, namespace string) (pvcName, pvHostPath string) {
		pvHostPath = filepath.Join(testsuite.HostPathBase, pvcBaseName)
		node := libstorage.CreateHostPathPv(pvcBaseName, namespace, pvHostPath)

		nodeSelector := map[string]string{k8sv1.LabelHostname: node}
		args := []string{fmt.Sprintf(
			`mkdir -p %s/data/sub && echo sub-content > %s/data/sub/visible_file && echo root-content > %s/hidden_root_file && chown -R 107:107 %s`,
			pvHostPath, pvHostPath, pvHostPath, pvHostPath,
		)}
		pod := libpod.RenderHostPathPod("virtiofs-mount-options-setup", pvHostPath, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)
		pod.Spec.NodeSelector = nodeSelector
		pod, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisPod(pod), 120).Should(BeInPhase(k8sv1.PodSucceeded))

		libstorage.CreateHostPathPVC(pvcBaseName, namespace, "1G")
		return fmt.Sprintf("disk-%s", pvcBaseName), pvHostPath
	}

	findVirtiofsDataVolumeMount := func(pod *k8sv1.Pod, volumeName string) k8sv1.VolumeMount {
		virtiofsContainerName := fmt.Sprintf("virtiofs-%s", volumeName)
		for _, container := range pod.Spec.Containers {
			if container.Name != virtiofsContainerName {
				continue
			}
			for _, mount := range container.VolumeMounts {
				if mount.Name == volumeName {
					return mount
				}
			}
		}
		Fail(fmt.Sprintf("virtiofs data volume mount for %s not found in pod %s", volumeName, pod.Name))
		return k8sv1.VolumeMount{}
	}

	waitForGuestAgent := func(vmi *v1.VirtualMachineInstance) {
		Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
	}

	Context("with subPath", func() {
		const pvcBaseName = "virtiofs-subpath"

		It("[Serial] should expose only the configured subdirectory to the guest", Serial, func() {
			pvcName, _ := setupHostPathPVCWithContent(pvcBaseName, testsuite.NamespaceTestDefault)
			defer func() {
				libstorage.DeletePVC(pvcBaseName, testsuite.NamespaceTestDefault)
				libstorage.DeletePV(pvcBaseName)
			}()

			virtiofsMountPath := fmt.Sprintf("/mnt/virtiofs_%s", pvcName)
			mountScript := fmt.Sprintf(`#!/bin/bash
mkdir %s
mount -t virtiofs %s %s
`, virtiofsMountPath, pvcName, virtiofsMountPath)

			vmi := libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountScript)),
				libvmi.WithFilesystemPVCVirtiofs(pvcName, v1.FilesystemVirtiofs{SubPath: "data/sub"}),
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)
			waitForGuestAgent(vmi)

			By("Verifying the virtiofs sidecar mounts only the configured subPath")
			virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			dataMount := findVirtiofsDataVolumeMount(virtLauncherPod, pvcName)
			Expect(dataMount.SubPath).To(Equal("data/sub"))
			Expect(dataMount.ReadOnly).To(BeFalse())

			By("Verifying the guest can access files from the subPath but not the volume root")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("test -f %s/visible_file && echo visible\n", virtiofsMountPath)},
				&expect.BExp{R: "visible"},
				&expect.BSnd{S: fmt.Sprintf("test -f %s/hidden_root_file && echo hidden || echo not_visible\n", virtiofsMountPath)},
				&expect.BExp{R: "not_visible"},
			}, 30*time.Second)).To(Succeed())
		})
	})

	Context("with readOnly", func() {
		const pvcBaseName = "virtiofs-readonly"

		It("[Serial] should prevent guest writes when readOnly is true", Serial, func() {
			pvcName, _ := setupHostPathPVCWithContent(pvcBaseName, testsuite.NamespaceTestDefault)
			defer func() {
				libstorage.DeletePVC(pvcBaseName, testsuite.NamespaceTestDefault)
				libstorage.DeletePV(pvcBaseName)
			}()

			virtiofsMountPath := fmt.Sprintf("/mnt/virtiofs_%s", pvcName)
			mountScript := fmt.Sprintf(`#!/bin/bash
mkdir %s
mount -t virtiofs %s %s
`, virtiofsMountPath, pvcName, virtiofsMountPath)

			vmi := libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountScript)),
				libvmi.WithFilesystemPVCVirtiofs(pvcName, v1.FilesystemVirtiofs{ReadOnly: true}),
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)
			waitForGuestAgent(vmi)

			By("Verifying the virtiofs sidecar mounts the source volume read-only")
			virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			dataMount := findVirtiofsDataVolumeMount(virtLauncherPod, pvcName)
			Expect(dataMount.ReadOnly).To(BeTrue())

			By("Verifying the guest cannot create new files on the read-only virtiofs mount")
			res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("touch %s/readonly_test 2>&1 || true\n", virtiofsMountPath)},
				&expect.BExp{R: ""},
			}, 30)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeEmpty())
			Expect(strings.ToLower(res[0].Output)).To(Or(
				ContainSubstring("read-only"),
				ContainSubstring("readonly"),
			))
		})
	})
})
