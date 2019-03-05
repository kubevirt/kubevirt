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

package installstrategy

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var _ = Describe("Install Strategy", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {

	})

	AfterEach(func() {
	})

	namespace := "fake-namespace"
	imageTag := "v9.9.9"
	imageRegistry := "fake-registry"

	Context("should generate", func() {
		It("latest install strategy with lossless byte conversion.", func() {

			strategy, err := GenerateCurrentInstallStrategy(
				namespace,
				imageTag,
				imageRegistry,
				corev1.PullIfNotPresent,
				"2")
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
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.clusterRoles {
				var converted *rbacv1.ClusterRole
				for _, converted = range newStrategy.clusterRoles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.clusterRoleBindings {
				var converted *rbacv1.ClusterRoleBinding
				for _, converted = range newStrategy.clusterRoleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.roles {
				var converted *rbacv1.Role
				for _, converted = range newStrategy.roles {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.roleBindings {
				var converted *rbacv1.RoleBinding
				for _, converted = range newStrategy.roleBindings {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.crds {
				var converted *extv1beta1.CustomResourceDefinition
				for _, converted = range newStrategy.crds {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.services {
				var converted *corev1.Service
				for _, converted = range newStrategy.services {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.daemonSets {
				var converted *appsv1.DaemonSet
				for _, converted = range newStrategy.daemonSets {
					if original.Name == converted.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}

			for _, original := range strategy.deployments {
				var converted *appsv1.Deployment
				for _, converted = range newStrategy.deployments {
					if converted.Name == original.Name {
						break
					}
				}
				Expect(reflect.DeepEqual(original, converted)).To(Equal(true))
			}
		})
	})
})
