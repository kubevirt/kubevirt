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
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"

	//"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Install Strategy", func() {
	log.Log.SetIOWriter(GinkgoWriter)

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

	Context("should generate", func() {
		It("install strategy convertable back to objects", func() {
			strategy, err := GenerateCurrentInstallStrategy(config, true, namespace)
			Expect(err).NotTo(HaveOccurred())

			b, err := dumpInstallStrategyToBytes(strategy)
			Expect(err).NotTo(HaveOccurred())
			data := string(b)

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
			strategy, err := GenerateCurrentInstallStrategy(config, true, namespace)
			Expect(err).ToNot(HaveOccurred())

			b, err := dumpInstallStrategyToBytes(strategy)
			Expect(err).NotTo(HaveOccurred())
			strategyStr := string(b)

			newStrategy, err := loadInstallStrategyFromBytes(strategyStr)
			Expect(err).ToNot(HaveOccurred())

			for _, original := range strategy.serviceAccounts {
				var converted *corev1.ServiceAccount
				for _, converted = range newStrategy.serviceAccounts {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.clusterRoles {
				var converted *rbacv1.ClusterRole
				for _, converted = range newStrategy.clusterRoles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.clusterRoleBindings {
				var converted *rbacv1.ClusterRoleBinding
				for _, converted = range newStrategy.clusterRoleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.roles {
				var converted *rbacv1.Role
				for _, converted = range newStrategy.roles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.roleBindings {
				var converted *rbacv1.RoleBinding
				for _, converted = range newStrategy.roleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.crds {
				var converted *extv1beta1.CustomResourceDefinition
				for _, converted = range newStrategy.crds {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.services {
				var converted *corev1.Service
				for _, converted = range newStrategy.services {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.daemonSets {
				var converted *appsv1.DaemonSet
				for _, converted = range newStrategy.daemonSets {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
			}

			for _, original := range strategy.deployments {
				var converted *appsv1.Deployment
				for _, converted = range newStrategy.deployments {
					if converted.Name == original.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
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
				Expect(reflect.DeepEqual(original, converted)).To(BeTrue())
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
})
