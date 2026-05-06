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

	workerv1 "kubevirt.io/api/worker/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	workerv1client "kubevirt.io/client-go/kubevirt/typed/worker/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
)

var _ = Describe(SIGSerial("worker pools", func() {
	var (
		virtClient   kubecli.KubevirtClient
		workerClient workerv1client.WorkerPoolInterface
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		wpClient, err := workerv1client.NewForConfig(virtClient.Config())
		Expect(err).ToNot(HaveOccurred())
		workerClient = wpClient.WorkerPools()
	})

	createWorkerPool := func(pool *workerv1.WorkerPool) *workerv1.WorkerPool {
		created, err := workerClient.Create(context.Background(), pool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return created
	}

	deleteWorkerPool := func(name string) {
		err := workerClient.Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			return
		}
	}

	Context("webhook validation", func() {
		BeforeEach(func() {
			config.EnableFeatureGate(featuregate.WorkerPools)
		})

		It("should reject pools with no image override", func() {
			pool := &workerv1.WorkerPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-img",
				},
				Spec: workerv1.WorkerPoolSpec{
					NodeSelector: map[string]string{"k": "v"},
					Selector:     workerv1.WorkerPoolSelector{DeviceNames: []string{"dev"}},
				},
			}
			_, err := workerClient.Create(context.Background(), pool, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("virtHandlerImage or virtLauncherImage"))
		})

		It("should reject pools with no selector criteria", func() {
			pool := &workerv1.WorkerPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-sel",
				},
				Spec: workerv1.WorkerPoolSpec{
					VirtHandlerImage: "img:latest",
					NodeSelector:     map[string]string{"k": "v"},
					Selector:         workerv1.WorkerPoolSelector{},
				},
			}
			_, err := workerClient.Create(context.Background(), pool, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("selector"))
		})
	})

	Context("pool DaemonSet lifecycle", func() {
		BeforeEach(func() {
			config.EnableFeatureGate(featuregate.WorkerPools)
		})

		It("should create and delete pool DaemonSets", func() {
			poolName := "func-test-pool"
			dsName := fmt.Sprintf("virt-handler-%s", poolName)

			pool := createWorkerPool(&workerv1.WorkerPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: poolName,
				},
				Spec: workerv1.WorkerPoolSpec{
					VirtLauncherImage: "registry.example.com/custom-launcher:test",
					NodeSelector:      map[string]string{"node-role.kubernetes.io/func-test-pool": "true"},
					Selector: workerv1.WorkerPoolSelector{
						DeviceNames: []string{"example.com/test-device"},
					},
				},
			})
			defer deleteWorkerPool(pool.Name)

			kv := libkubevirt.GetCurrentKv(virtClient)

			// Wait for the pool DaemonSet to be created
			Eventually(func() error {
				_, getErr := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return getErr
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			// Verify the pool DaemonSet has the correct labels
			ds, err := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(ds.Labels[workerv1.WorkerPoolLabel]).To(Equal(poolName))

			// Verify the pool DaemonSet has the correct nodeSelector
			Expect(ds.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("node-role.kubernetes.io/func-test-pool", "true"))

			// Delete the pool CR
			deleteWorkerPool(pool.Name)

			// Wait for the pool DaemonSet to be deleted
			Eventually(func() bool {
				_, err := virtClient.AppsV1().DaemonSets(kv.Namespace).Get(context.Background(), dsName, metav1.GetOptions{})
				return err != nil
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())
		})

		PIt("should apply anti-affinity to primary virt-handler", func() {
			pool := createWorkerPool(&workerv1.WorkerPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "anti-aff-test",
				},
				Spec: workerv1.WorkerPoolSpec{
					VirtLauncherImage: "registry.example.com/custom-launcher:test",
					NodeSelector:      map[string]string{"node-role.kubernetes.io/anti-aff-test": "true"},
					Selector: workerv1.WorkerPoolSelector{
						DeviceNames: []string{"example.com/test-device"},
					},
				},
			})
			defer deleteWorkerPool(pool.Name)

			kv := libkubevirt.GetCurrentKv(virtClient)

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
		PIt("should annotate virt-launcher pod with pool name for label-matched VMI", func() {
			// TODO: requires virt-controller WorkerPool informer RBAC and
			// anti-affinity steady-state support to avoid dual-handler conflicts
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
