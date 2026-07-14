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

package operator

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	exportProxyHPAName = components.VirtExportProxyHPAName
	exportProxyPDBName = components.VirtExportProxyName + "-pdb"
)

var _ = Describe("[sig-operator]virt-exportproxy HPA lifecycle", decorators.SigOperator, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := libkubevirt.GetCurrentKv(virtClient)
		testsuite.EnsureKubevirtReadyWithTimeout(kv, 120*time.Second)
		testsuite.WaitExportProxyReady()
	})

	// Serial: tests share the cluster-scoped virt-exportproxy-hpa; the reconcile test
	// patches MaxReplicas and must not race the install assertion under parallel E2E.
	Context("on a multi-node cluster", Serial, decorators.MultiReplica, func() {
		It("should install virt-exportproxy-hpa with a metrics profile annotation", func() {
			var hpa *autoscalingv2.HorizontalPodAutoscaler

			By("waiting for virt-exportproxy-hpa")
			Eventually(func(g Gomega) {
				var err error
				hpa, err = virtClient.AutoscalingV2().HorizontalPodAutoscalers(flags.KubeVirtInstallNamespace).Get(
					context.Background(), exportProxyHPAName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
			}, 60*time.Second, time.Second).Should(Succeed())

			By("checking HPA targets virt-exportproxy")
			Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(components.VirtExportProxyName))
			Expect(*hpa.Spec.MinReplicas).To(Equal(int32(2)))
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(20)))

			By("checking metrics profile annotation")
			profile, ok := hpa.Annotations[components.ExportProxyHPAMetricsProfileAnnotation]
			Expect(ok).To(BeTrue())
			Expect(profile).To(Or(
				Equal(string(components.ExportProxyHPAMetricsProfileResource)),
				Equal(string(components.ExportProxyHPAMetricsProfileCustomMetrics)),
			))

			By("checking export-proxy PDB exists with minAvailable 1")
			pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(
				context.Background(), exportProxyPDBName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pdb.Spec.MinAvailable.IntValue()).To(Equal(1))
		})

		It("should revert a manual HPA change", func() {
			hpa, err := virtClient.AutoscalingV2().HorizontalPodAutoscalers(flags.KubeVirtInstallNamespace).Get(
				context.Background(), exportProxyHPAName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			resource := hpa.DeepCopy()

			patchBytes, err := patch.New(
				patch.WithReplace("/spec/maxReplicas", int32(5)),
			).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())

			By("patching virt-exportproxy-hpa maxReplicas")
			_, err = virtClient.AutoscalingV2().HorizontalPodAutoscalers(flags.KubeVirtInstallNamespace).Patch(
				context.Background(), exportProxyHPAName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			var generation int64
			By("waiting for virt-operator to restore maxReplicas")
			Eventually(func(g Gomega) {
				current, err := virtClient.AutoscalingV2().HorizontalPodAutoscalers(flags.KubeVirtInstallNamespace).Get(
					context.Background(), exportProxyHPAName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(current.Spec.MaxReplicas).To(Equal(int32(20)))
				generation = current.GetGeneration()
			}, 120*time.Second, 5*time.Second).Should(Succeed())

			Eventually(func() int64 {
				currentKV := libkubevirt.GetCurrentKv(virtClient)
				return apply.GetExpectedGeneration(resource, currentKV.Status.Generations)
			}, 60*time.Second, 5*time.Second).Should(Equal(generation))
		})
	})

	Context("on a single-node cluster", decorators.SingleReplica, decorators.NoFlakeCheck, func() {
		It("should omit virt-exportproxy-hpa and export-proxy PDB protection", func() {
			By("checking virt-exportproxy-hpa is absent")
			Consistently(func(g Gomega) {
				_, err := virtClient.AutoscalingV2().HorizontalPodAutoscalers(flags.KubeVirtInstallNamespace).Get(
					context.Background(), exportProxyHPAName, metav1.GetOptions{})
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("checking virt-exportproxy-pdb is absent or has minAvailable 0")
			Consistently(func(g Gomega) {
				pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(
					context.Background(), exportProxyPDBName, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return
				}
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(pdb.Spec.MinAvailable.IntValue()).To(Equal(0))
			}, 30*time.Second, 2*time.Second).Should(Succeed())
		})
	})
})
