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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package install

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	//"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Install Strategy", func() {

	namespace := "fake-namespace"

	getConfig := func(registry, version string) *util.KubeVirtDeploymentConfig {
		return util.GetTargetConfigFromKV(&v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: v1.KubeVirtSpec{
				ImageRegistry: registry,
				ImageTag:      version,
			},
		})
	}

	config := getConfig("fake-registry", "v9.9.9")

	Context("monitoring detection", func() {
		DescribeTable("should", func(expectedNS string, objects ...runtime.Object) {
			client := fake.NewSimpleClientset(objects...)
			ns, err := getMonitorNamespace(client.CoreV1(), config)
			Expect(ns).To(Equal(expectedNS))
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("match first entry if namespace and SA exist",
				"openshift-monitoring",
				newSA("openshift-monitoring", "prometheus-k8s"),
				newNS("openshift-monitoring"),
			),
			Entry("should match second namespace if SA for the first namespace does not exist",
				"monitoring",
				newSA("monitoring", "prometheus-k8s"),
				newNS("openshift-monitoring"),
				newNS("monitoring"),
			),
			Entry("should match first namespace if SA for both namespaces exist",
				"openshift-monitoring",
				newSA("openshift-monitoring", "prometheus-k8s"),
				newSA("monitoring", "prometheus-k8s"),
				newNS("openshift-monitoring"),
				newNS("monitoring"),
			),
			Entry("succeed fail if SA does not exist in both namespaces",
				"",
				newNS("openshift-monitoring"),
				newNS("monitoring"),
			),
		)
	})

	Context("should generate", func() {
		It("install strategy convertable back to objects", func() {
			strategy, err := GenerateCurrentInstallStrategy(config, "openshift-monitoring", namespace)
			Expect(err).NotTo(HaveOccurred())

			data := string(dumpInstallStrategyToBytes(strategy))

			entries := strings.Split(data, "---")

			for _, entry := range entries {
				entry := strings.TrimSpace(entry)
				if entry == "" {
					continue
				}
				var obj metav1.TypeMeta
				err := yaml.Unmarshal([]byte(entry), &obj)
				Expect(err).NotTo(HaveOccurred())
			}

		})
		It("latest install strategy with lossless byte conversion.", func() {
			strategy, err := GenerateCurrentInstallStrategy(config, "openshift-monitoring", namespace)
			Expect(err).ToNot(HaveOccurred())

			strategyStr := string(dumpInstallStrategyToBytes(strategy))

			newStrategy, err := loadInstallStrategyFromBytes(strategyStr)
			Expect(err).ToNot(HaveOccurred())

			for _, original := range strategy.serviceAccounts {
				var converted *corev1.ServiceAccount
				for _, converted = range newStrategy.serviceAccounts {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.clusterRoles {
				var converted *rbacv1.ClusterRole
				for _, converted = range newStrategy.clusterRoles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.clusterRoleBindings {
				var converted *rbacv1.ClusterRoleBinding
				for _, converted = range newStrategy.clusterRoleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.roles {
				var converted *rbacv1.Role
				for _, converted = range newStrategy.roles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.roleBindings {
				var converted *rbacv1.RoleBinding
				for _, converted = range newStrategy.roleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.crds {
				var converted *extv1.CustomResourceDefinition
				for _, converted = range newStrategy.crds {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.services {
				var converted *corev1.Service
				for _, converted = range newStrategy.services {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.daemonSets {
				var converted *appsv1.DaemonSet
				for _, converted = range newStrategy.daemonSets {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.deployments {
				var converted *appsv1.Deployment
				for _, converted = range newStrategy.deployments {
					if converted.Name == original.Name {
						break
					}
				}
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.configMaps {
				var converted *corev1.ConfigMap
				for _, converted = range newStrategy.configMaps {
					if converted.Name == original.Name {
						break
					}
				}
				//delete ManagedByLabel labels from original config map.
				//dumpInstallStrategyToBytes function deletes it, and then
				//original and converted configmaps are not the same
				delete(original.Labels, v1.ManagedByLabel)
				Expect(equality.Semantic.DeepEqual(original, converted)).To(BeTrue())
			}
		})
	})

	Context("should match", func() {
		It("the most recent install strategy.", func() {
			var configMaps []*corev1.ConfigMap

			configMaps = append(configMaps, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test1",
					CreationTimestamp: metav1.Time{},
				},
			})
			configMaps = append(configMaps, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test2",
					CreationTimestamp: metav1.Now(),
				},
			})
			configMaps = append(configMaps, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test3",
					CreationTimestamp: metav1.Time{},
				},
			})

			configMap := mostRecentConfigMap(configMaps)
			Expect(configMap.Name).To(Equal("test2"))
		})
	})

	Context("should load", func() {
		It("a plaintext install strategy.", func() {
			// for backwards compatibility
			stores := util.Stores{}
			stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)
			strategy, err := GenerateCurrentInstallStrategy(config, "openshift-monitoring", namespace)
			Expect(err).ToNot(HaveOccurred())
			data := string(dumpInstallStrategyToBytes(strategy))

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "plaintext-install-strategy",
					Namespace:    config.GetNamespace(),
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
						v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
						v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
					},
				},
				Data: map[string]string{
					"manifests": data,
				},
			}
			stores.InstallStrategyConfigMapCache.Add(configMap)
			_, err = LoadInstallStrategyFromCache(stores, config)
			Expect(err).ToNot(HaveOccurred())
		})
		It("a gzip+base64 encoded install strategy.", func() {
			stores := util.Stores{}
			stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)
			configMap, err := NewInstallStrategyConfigMap(config, "openshift-monitoring", namespace)
			Expect(err).ToNot(HaveOccurred())
			stores.InstallStrategyConfigMapCache.Add(configMap)
			_, err = LoadInstallStrategyFromCache(stores, config)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func newSA(namespace string, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
}

func newNS(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}
