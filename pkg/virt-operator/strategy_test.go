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

package virt_operator

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Strategy", func() {

	Context("deleteAllOldInstallStrategies", func() {

		var (
			ctrl       *gomock.Controller
			clientset  *kubecli.MockKubevirtClient
			kubeClient *fake.Clientset
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kubeClient = fake.NewSimpleClientset()
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should not panic when cache contains non-ConfigMap objects", func() {
			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			store.Add("not-a-configmap")
			store.Add(42)

			controller := &KubeVirtController{
				stores: util.Stores{
					InstallStrategyConfigMapCache: store,
				},
				clientset: clientset,
			}

			err := controller.deleteAllOldInstallStrategies("v1.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete configmaps with outdated versions", func() {
			oldConfigMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "old-strategy",
					Namespace: "kubevirt",
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation: "v0.9.0",
					},
				},
			}

			kubeClient = fake.NewSimpleClientset(oldConfigMap)
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			Expect(store.Add(oldConfigMap)).To(Succeed())

			controller := &KubeVirtController{
				stores: util.Stores{
					InstallStrategyConfigMapCache: store,
				},
				clientset: clientset,
			}

			err := controller.deleteAllOldInstallStrategies("v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().ConfigMaps("kubevirt").Get(context.Background(), "old-strategy", metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
		})

		It("should not delete configmaps with matching versions", func() {
			currentConfigMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "current-strategy",
					Namespace: "kubevirt",
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation: "v1.0.0",
					},
				},
			}

			kubeClient = fake.NewSimpleClientset(currentConfigMap)
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			Expect(store.Add(currentConfigMap)).To(Succeed())

			controller := &KubeVirtController{
				stores: util.Stores{
					InstallStrategyConfigMapCache: store,
				},
				clientset: clientset,
			}

			err := controller.deleteAllOldInstallStrategies("v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().ConfigMaps("kubevirt").Get(context.Background(), "current-strategy", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should skip configmaps without version annotation", func() {
			noAnnotationConfigMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-annotation",
					Namespace: "kubevirt",
				},
			}

			kubeClient = fake.NewSimpleClientset(noAnnotationConfigMap)
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			Expect(store.Add(noAnnotationConfigMap)).To(Succeed())

			controller := &KubeVirtController{
				stores: util.Stores{
					InstallStrategyConfigMapCache: store,
				},
				clientset: clientset,
			}

			err := controller.deleteAllOldInstallStrategies("v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().ConfigMaps("kubevirt").Get(context.Background(), "no-annotation", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle mixed cache contents without panic", func() {
			oldConfigMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "old-strategy",
					Namespace: "kubevirt",
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation: "v0.9.0",
					},
				},
			}
			currentConfigMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "current-strategy",
					Namespace: "kubevirt",
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation: "v1.0.0",
					},
				},
			}

			kubeClient = fake.NewSimpleClientset(oldConfigMap, currentConfigMap)
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			store.Add("unexpected-string")
			store.Add(oldConfigMap)
			store.Add(currentConfigMap)
			store.Add(42)

			controller := &KubeVirtController{
				stores: util.Stores{
					InstallStrategyConfigMapCache: store,
				},
				clientset: clientset,
			}

			err := controller.deleteAllOldInstallStrategies("v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().ConfigMaps("kubevirt").Get(context.Background(), "old-strategy", metav1.GetOptions{})
			Expect(err).To(HaveOccurred())

			_, err = kubeClient.CoreV1().ConfigMaps("kubevirt").Get(context.Background(), "current-strategy", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
