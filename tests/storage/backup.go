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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/velero"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Backup", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Velero backup hooks injection", Serial, func() {
		It("should dynamically sync hooks annotations based on KubeVirt CR annotation", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)))

			By("Creating VMI without skip-backup-hooks annotation")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 300*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))

			By("Verifying launcher pod has Velero backup hooks annotations")
			pod := getPodByVMI(vmi)
			Expect(pod.Annotations).To(HaveKey(velero.PreBackupHookContainerAnnotation))

			kv := libkubevirt.GetCurrentKv(virtClient)
			originalKvAnnotations := kv.Annotations
			if originalKvAnnotations == nil {
				originalKvAnnotations = make(map[string]string)
			}
			_, hadSkipAnnotation := originalKvAnnotations[velero.SkipHooksAnnotation]

			By("Adding skip-backup-hooks annotation to KubeVirt CR")
			patchData := fmt.Appendf(nil, `{"metadata":{"annotations":{%q:"true"}}}`, velero.SkipHooksAnnotation)
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying launcher pod Velero annotations are removed")
			Eventually(func() bool {
				pod = getPodByVMI(vmi)
				_, hasPreHook := pod.Annotations[velero.PreBackupHookContainerAnnotation]
				_, hasPostHook := pod.Annotations[velero.PostBackupHookContainerAnnotation]
				return !hasPreHook && !hasPostHook
			}, 60*time.Second, 1*time.Second).Should(BeTrue(), "Velero hook annotations should be removed from launcher pod when KubeVirt CR annotation is set")

			By("Restoring KubeVirt CR annotations to original state")
			if hadSkipAnnotation {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:%q}}}`, velero.SkipHooksAnnotation, originalKvAnnotations[velero.SkipHooksAnnotation])
			} else {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:null}}}`, velero.SkipHooksAnnotation)
			}
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying launcher pod Velero annotations are added back")
			Eventually(func() bool {
				pod = getPodByVMI(vmi)
				return pod.Annotations[velero.PreBackupHookContainerAnnotation] == "compute" &&
					pod.Annotations[velero.PostBackupHookContainerAnnotation] == "compute"
			}, 60*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("VMI annotation should take precedence over KubeVirt CR annotation", func() {
			By("Getting KubeVirt CR and setting skip annotation to true")
			kv := libkubevirt.GetCurrentKv(virtClient)
			originalKvAnnotations := kv.Annotations
			if originalKvAnnotations == nil {
				originalKvAnnotations = make(map[string]string)
			}
			_, hadSkipAnnotation := originalKvAnnotations[velero.SkipHooksAnnotation]

			patchData := fmt.Appendf(nil, `{"metadata":{"annotations":{%q:"true"}}}`, velero.SkipHooksAnnotation)
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating VMI with skip-backup-hooks=false annotation (opposite of KubeVirt CR)")
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithAnnotation(velero.SkipHooksAnnotation, "false"))

			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 300*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))

			By("Verifying launcher pod has Velero annotations (VMI annotation takes precedence)")
			pod := getPodByVMI(vmi)
			Expect(pod.Annotations).To(HaveKey(velero.PreBackupHookContainerAnnotation), "VMI annotation should override KubeVirt CR annotation")
			Expect(pod.Annotations).To(HaveKey(velero.PostBackupHookContainerAnnotation))
			Expect(pod.Annotations[velero.PreBackupHookContainerAnnotation]).To(Equal("compute"))

			By("Restoring KubeVirt CR annotations to original state")
			if hadSkipAnnotation {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:%q}}}`, velero.SkipHooksAnnotation, originalKvAnnotations[velero.SkipHooksAnnotation])
			} else {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:null}}}`, velero.SkipHooksAnnotation)
			}
			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
}))

func getPodByVMI(vmi *v1.VirtualMachineInstance) *corev1.Pod {
	pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pod).ToNot(BeNil())
	return pod
}
