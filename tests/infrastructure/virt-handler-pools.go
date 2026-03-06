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

package infrastructure

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIGSerial("virt-handler pools", func() {
	var (
		virtClient       kubecli.KubevirtClient
		originalKubeVirt *v1.KubeVirt
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKubeVirt = libkubevirt.GetCurrentKv(virtClient)
	})

	AfterEach(func() {
		By("Restoring the original KubeVirt configuration")
		kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		kv.Spec.Configuration = originalKubeVirt.Spec.Configuration
		kv.Spec.VirtHandlerPools = originalKubeVirt.Spec.VirtHandlerPools
		_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		testsuite.EnsureKubevirtReady()
	})

	enableFeatureGate := func() {
		kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
		if kvConfig.DeveloperConfiguration == nil {
			kvConfig.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		}
		kvConfig.DeveloperConfiguration.FeatureGates = append(kvConfig.DeveloperConfiguration.FeatureGates, featuregate.VirtHandlerPools)
		kvconfig.UpdateKubeVirtConfigValueAndWait(*kvConfig)
	}

	Context("pool lifecycle", func() {
		It("should create and delete pool DaemonSets when pools are added and removed", func() {
			enableFeatureGate()

			poolName := "test-pool"
			dsName := "virt-handler-" + poolName

			By("Adding a virt-handler pool to the KubeVirt CR")
			kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = []v1.VirtHandlerPoolConfig{
				{
					Name:              poolName,
					VirtHandlerImage:  "registry.example.com/virt-handler:pool-test",
					VirtLauncherImage: "registry.example.com/virt-launcher:pool-test",
					NodeSelector:      map[string]string{"pool-test-label": "true"},
					Selector: v1.VirtHandlerPoolSelector{
						DeviceNames: []string{"example.com/test-device"},
					},
				},
			}
			kv, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the pool DaemonSet to be created")
			Eventually(func() error {
				_, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return err
			}, 120*time.Second, 2*time.Second).Should(Succeed(), "pool DaemonSet %s should be created", dsName)

			By("Verifying the pool DaemonSet has the correct labels and selectors")
			ds, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), dsName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(ds.Labels).To(HaveKeyWithValue(v1.HandlerPoolLabel, poolName))
			Expect(ds.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("pool-test-label", "true"))

			By("Verifying the primary virt-handler DaemonSet has anti-affinity for pool nodes")
			primaryDS, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(primaryDS.Spec.Template.Spec.Affinity).ToNot(BeNil())
			Expect(primaryDS.Spec.Template.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(primaryDS.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
			terms := primaryDS.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			found := false
			for _, term := range terms {
				for _, expr := range term.MatchExpressions {
					if expr.Key == "pool-test-label" && expr.Operator == k8sv1.NodeSelectorOpNotIn {
						found = true
					}
				}
			}
			Expect(found).To(BeTrue(), "primary virt-handler should have anti-affinity for pool nodeSelector keys")

			By("Removing the pool from the KubeVirt CR")
			kv, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = nil
			_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the pool DaemonSet to be deleted")
			Eventually(func() bool {
				_, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 120*time.Second, 2*time.Second).Should(BeTrue(), "pool DaemonSet %s should be deleted", dsName)
		})
	})

	Context("webhook validation", func() {
		It("should reject pools when feature gate is not enabled", func() {
			kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = []v1.VirtHandlerPoolConfig{
				{
					Name:              "should-fail",
					VirtHandlerImage:  "registry.example.com/virt-handler:test",
					NodeSelector:      map[string]string{"k": "v"},
					Selector: v1.VirtHandlerPoolSelector{
						DeviceNames: []string{"example.com/device"},
					},
				},
			}
			_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).To(HaveOccurred(), "should reject pools when VirtHandlerPools feature gate is not enabled")
		})

		It("should reject pools with duplicate names", func() {
			enableFeatureGate()

			kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = []v1.VirtHandlerPoolConfig{
				{
					Name:              "dup",
					VirtHandlerImage:  "registry.example.com/virt-handler:test",
					NodeSelector:      map[string]string{"k": "v"},
					Selector: v1.VirtHandlerPoolSelector{
						DeviceNames: []string{"example.com/device"},
					},
				},
				{
					Name:              "dup",
					VirtLauncherImage: "registry.example.com/virt-launcher:test",
					NodeSelector:      map[string]string{"k2": "v2"},
					Selector: v1.VirtHandlerPoolSelector{
						DeviceNames: []string{"example.com/device2"},
					},
				},
			}
			_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).To(HaveOccurred(), "should reject pools with duplicate names")
		})

		It("should reject pools with no image override", func() {
			enableFeatureGate()

			kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = []v1.VirtHandlerPoolConfig{
				{
					Name:         "no-image",
					NodeSelector: map[string]string{"k": "v"},
					Selector: v1.VirtHandlerPoolSelector{
						DeviceNames: []string{"example.com/device"},
					},
				},
			}
			_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).To(HaveOccurred(), "should reject pools with no image override")
		})

		It("should reject pools with no selector criteria", func() {
			enableFeatureGate()

			kv, err := virtClient.KubeVirt(originalKubeVirt.Namespace).Get(context.Background(), originalKubeVirt.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.VirtHandlerPools = []v1.VirtHandlerPoolConfig{
				{
					Name:              "no-selector",
					VirtHandlerImage:  "registry.example.com/virt-handler:test",
					NodeSelector:      map[string]string{"k": "v"},
					Selector:          v1.VirtHandlerPoolSelector{},
				},
			}
			_, err = virtClient.KubeVirt(originalKubeVirt.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).To(HaveOccurred(), "should reject pools with no selector criteria")
		})
	})
}))
