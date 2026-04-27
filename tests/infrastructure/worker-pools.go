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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIGSerial("worker pools", func() {
	var (
		virtClient kubecli.KubevirtClient
		originalKv *v1.KubeVirt
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv = libkubevirt.GetCurrentKv(virtClient)
	})

	AfterEach(func() {
		// Restore original KubeVirt config
		patchSet := patch.New(
			patch.WithReplace("/spec/workerPools", nil),
		)
		patchBytes, err := patchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		// Ignore errors during cleanup - the field might already be empty
		_, _ = virtClient.KubeVirt(originalKv.Namespace).Patch(
			context.Background(), originalKv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		config.UpdateKubeVirtConfigValueAndWait(originalKv.Spec.Configuration)
	})

	Context("webhook validation", func() {
		It("should reject pools when the feature gate is not enabled", func() {
			kv := libkubevirt.GetCurrentKv(virtClient)

			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:             "test",
						VirtHandlerImage: "img:latest",
						NodeSelector:     map[string]string{"test-key": "test-val"},
						Selector:         v1.WorkerPoolSelector{DeviceNames: []string{"test-dev"}},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("feature gate"))
		})

		It("should reject pools with duplicate names", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)
			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:             "dup",
						VirtHandlerImage: "img:latest",
						NodeSelector:     map[string]string{"k": "v"},
						Selector:         v1.WorkerPoolSelector{DeviceNames: []string{"dev"}},
					},
					{
						Name:              "dup",
						VirtLauncherImage: "img:latest",
						NodeSelector:      map[string]string{"k2": "v2"},
						Selector:          v1.WorkerPoolSelector{DeviceNames: []string{"dev2"}},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate"))
		})

		It("should reject pools with no image override", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)
			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:         "no-img",
						NodeSelector: map[string]string{"k": "v"},
						Selector:     v1.WorkerPoolSelector{DeviceNames: []string{"dev"}},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("virtHandlerImage or virtLauncherImage"))
		})

		It("should reject pools with no selector criteria", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)
			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:             "no-sel",
						VirtHandlerImage: "img:latest",
						NodeSelector:     map[string]string{"k": "v"},
						Selector:         v1.WorkerPoolSelector{},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("selector"))
		})
	})

	Context("pool DaemonSet lifecycle", func() {
		It("should create and delete pool DaemonSets", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)

			poolName := "func-test-pool"
			dsName := fmt.Sprintf("virt-handler-%s", poolName)
			poolConfig := []v1.WorkerPoolConfig{
				{
					Name:              poolName,
					VirtLauncherImage: "registry.example.com/custom-launcher:test",
					NodeSelector:      map[string]string{"node-role.kubernetes.io/func-test-pool": "true"},
					Selector: v1.WorkerPoolSelector{
						DeviceNames: []string{"example.com/test-device"},
					},
				},
			}

			// Add the pool
			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", poolConfig),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the pool DaemonSet to be created
			Eventually(func() error {
				_, getErr := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return getErr
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			// Verify the pool DaemonSet has the correct labels
			ds, err := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(ds.Labels[v1.WorkerPoolLabel]).To(Equal(poolName))

			// Verify the pool DaemonSet has the correct nodeSelector
			Expect(ds.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("node-role.kubernetes.io/func-test-pool", "true"))

			// Remove the pool
			patchSet = patch.New(
				patch.WithReplace("/spec/workerPools", nil),
			)
			patchBytes, err = patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the pool DaemonSet to be deleted
			Eventually(func() bool {
				_, err := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return err != nil
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("should apply anti-affinity to primary virt-handler", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)

			poolConfig := []v1.WorkerPoolConfig{
				{
					Name:              "anti-aff-test",
					VirtLauncherImage: "registry.example.com/custom-launcher:test",
					NodeSelector:      map[string]string{"node-role.kubernetes.io/anti-aff-test": "true"},
					Selector: v1.WorkerPoolSelector{
						DeviceNames: []string{"example.com/test-device"},
					},
				},
			}

			// Add the pool
			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", poolConfig),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Wait for the primary virt-handler to have anti-affinity
			Eventually(func() bool {
				ds, err := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				if err != nil {
					return false
				}
				return hasAntiAffinityForKey(ds, "node-role.kubernetes.io/anti-aff-test")
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())
		})
	})

	Context("VMI pool matching", func() {
		It("should annotate virt-launcher pod with pool name for label-matched VMI", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)
			poolName := "label-match-test"

			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:              poolName,
						VirtLauncherImage: kv.Status.TargetDeploymentConfig,
						NodeSelector:      map[string]string{"kubernetes.io/os": "linux"},
						Selector: v1.WorkerPoolSelector{
							VMLabels: &v1.WorkerPoolVMLabels{
								MatchLabels: map[string]string{"test-pool": "true"},
							},
						},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmifact.NewGuestless(
				libvmi.WithLabel("test-pool", "true"),
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Annotations).To(HaveKeyWithValue(v1.WorkerPoolLabel, poolName))
		})
	})

	Context("pool removal blocking", func() {
		It("should reject pool removal while matched VMIs are running", func() {
			config.EnableFeatureGate(featuregate.WorkerPools)

			kv := libkubevirt.GetCurrentKv(virtClient)
			poolName := "removal-block-test"

			patchSet := patch.New(
				patch.WithAdd("/spec/workerPools", []v1.WorkerPoolConfig{
					{
						Name:              poolName,
						VirtLauncherImage: kv.Status.TargetDeploymentConfig,
						NodeSelector:      map[string]string{"kubernetes.io/os": "linux"},
						Selector: v1.WorkerPoolSelector{
							VMLabels: &v1.WorkerPoolVMLabels{
								MatchLabels: map[string]string{"removal-test": "true"},
							},
						},
					},
				}),
			)
			patchBytes, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmifact.NewGuestless(
				libvmi.WithLabel("removal-test", "true"),
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

			// Attempt to remove the pool while VMI is running
			patchSet = patch.New(
				patch.WithReplace("/spec/workerPools", nil),
			)
			patchBytes, err = patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot remove pool"))

			// Delete the VMI
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, getErr := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				return getErr
			}, 2*time.Minute, 5*time.Second).ShouldNot(Succeed())

			// Now pool removal should succeed
			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
}))

func hasAntiAffinityForKey(ds *appsv1.DaemonSet, key string) bool {
	affinity := ds.Spec.Template.Spec.Affinity
	if affinity == nil || affinity.NodeAffinity == nil {
		return false
	}
	req := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if req == nil {
		return false
	}
	for _, term := range req.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == key {
				return true
			}
		}
	}
	return false
}
