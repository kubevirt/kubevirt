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
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
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

	Context("should calculate", func() {

		table.DescribeTable("update path based on semver", func(target string, current string, expected bool) {
			takeUpdatePath := shouldTakeUpdatePath(target, current)

			Expect(takeUpdatePath).To(Equal(expected))
		},
			table.Entry("with increasing semver", "v0.15.0", "v0.14.0", true),
			table.Entry("with decreasing semver", "v0.14.0", "v0.15.0", false),
			table.Entry("with identical semver", "v0.15.0", "v0.15.0", false),
			table.Entry("with invalid semver", "devel", "v0.14.0", true),
			table.Entry("with increasing semver no prefix", "0.15.0", "0.14.0", true),
			table.Entry("with decreasing semver no prefix", "0.14.0", "0.15.0", false),
			table.Entry("with identical semver no prefix", "0.15.0", "0.15.0", false),
			table.Entry("with invalid semver no prefix", "devel", "0.14.0", true),
			table.Entry("with no current no prefix", "devel", "", false),
		)
	})

	Context("should handle service endpoint updates", func() {
		table.DescribeTable("with either patch or complete replacement",
			func(cachedService *corev1.Service,
				targetService *corev1.Service,
				expectLabelsAnnotationsPatch bool,
				expectSpecPatch bool,
				expectDelete bool) {

				kv := &v1.KubeVirt{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-install",
						Namespace: "default",
					},
					Spec: v1.KubeVirtSpec{
						ImageTag:      imageTag,
						ImageRegistry: imageRegistry,
					},
					Status: v1.KubeVirtStatus{
						TargetKubeVirtVersion:  imageTag,
						TargetKubeVirtRegistry: imageRegistry,
					},
				}

				ops, shouldDeleteAndReplace, err := generateServicePatch(kv, cachedService, targetService)
				Expect(err).To(BeNil())
				Expect(shouldDeleteAndReplace).To(Equal(expectDelete))

				hasSubstring := func(ops []string, substring string) bool {
					for _, op := range ops {
						if strings.Contains(op, substring) {
							return true
						}
					}
					return false
				}

				if expectLabelsAnnotationsPatch {
					Expect(hasSubstring(ops, "/metadata/labels")).To(BeTrue())
					Expect(hasSubstring(ops, "/metadata/annotations")).To(BeTrue())
				}

				if expectSpecPatch {
					Expect(hasSubstring(ops, "/spec")).To(BeTrue())
				}

				if !expectSpecPatch && !expectLabelsAnnotationsPatch {
					Expect(len(ops)).To(Equal(0))
				}
			},
			table.Entry("should delete and recreate service if of mixed 'type'.",
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				},
				false, false, true),
			table.Entry("should delete and recreate service if not of type ClusterIP.",
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				},
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				},
				false, false, true),
			table.Entry("should delete and recreate service if ClusterIP changes (clusterIP is not mutable)",
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						ClusterIP: "2.2.2.2",
						Type:      corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.1.1.1",
						Type:      corev1.ServiceTypeClusterIP,
					},
				},
				false, false, true),
			table.Entry("should do nothing if cached service has ClusterIP set and target does not (clusterIP is dynamically assigned when empty)",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "2.2.2.2",
						Type:      corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				false, false, false),
			table.Entry("should update labels, annotations on update",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  "oldversion",
							v1.InstallStrategyRegistryAnnotation: "oldversion",
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"prometheus.kubevirt.io": "",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "old",
								Port: 444,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8444,
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"prometheus.kubevirt.io": "",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "old",
								Port: 444,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8444,
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				true, false, false),
			table.Entry("no-op with identical specs",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							v1.AppLabel: "virt-api",
						},
						Ports: []corev1.ServicePort{
							{
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8443,
								},
								Protocol: corev1.ProtocolTCP,
							},
							{
								Name: "metrics",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.String,
									StrVal: "metrics",
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							v1.AppLabel: "virt-api",
						},
						Ports: []corev1.ServicePort{
							{
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8443,
								},
								Protocol: corev1.ProtocolTCP,
							},
							{
								Name: "metrics",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.String,
									StrVal: "metrics",
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				false, false, false),
			table.Entry("should patch spec when selectors differ",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  "old",
							v1.InstallStrategyRegistryAnnotation: "old",
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							v1.AppLabel: "virt-api",
						},
						Ports: []corev1.ServicePort{
							{
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8443,
								},
								Protocol: corev1.ProtocolTCP,
							},
							{
								Name: "metrics",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.String,
									StrVal: "metrics",
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:  imageTag,
							v1.InstallStrategyRegistryAnnotation: imageRegistry,
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"somenew-selector": "val",
						},
						Ports: []corev1.ServicePort{
							{
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8443,
								},
								Protocol: corev1.ProtocolTCP,
							},
							{
								Name: "metrics",
								Port: 443,
								TargetPort: intstr.IntOrString{
									Type:   intstr.String,
									StrVal: "metrics",
								},
								Protocol: corev1.ProtocolTCP,
							},
						},
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				true, true, false),
		)
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
