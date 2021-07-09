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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package apply

import (
	"encoding/json"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"github.com/golang/mock/gomock"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Apply", func() {

	Context("Services", func() {

		It("should patch if ClusterIp == \"\" during update", func() {
			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeClusterIP
			cachedService.Spec.ClusterIP = "10.10.10.10"

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeClusterIP
			service.Spec.ClusterIP = ""

			ops, err := generateServicePatch(cachedService, service)
			Expect(err).To(BeNil())
			Expect(ops).ToNot(Equal(""))
		})

		It("should replace if ClusterIp != \"\" during update and ip changes", func() {

			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeClusterIP
			cachedService.Spec.ClusterIP = "10.10.10.10"

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeClusterIP
			service.Spec.ClusterIP = "10.10.10.11"

			deleteAndReplace := hasImmutableFieldChanged(service, cachedService)
			Expect(deleteAndReplace).To(BeTrue())
		})

		It("should replace if not a ClusterIP service", func() {
			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeNodePort

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeNodePort

			deleteAndReplace := hasImmutableFieldChanged(service, cachedService)
			Expect(deleteAndReplace).To(BeTrue())
		})
	})

	Context("should reconcile service account", func() {

		newServiceAccount := func() *corev1.ServiceAccount {
			return &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
					Name:      "name",
				},
			}
		}

		var clientset *kubecli.MockKubevirtClient
		var ctrl *gomock.Controller
		var coreclientset *fake.Clientset
		var expectations *util.Expectations
		var kv *v1.KubeVirt
		var stores util.Stores

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
			coreclientset = fake.NewSimpleClientset()

			coreclientset.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			stores = util.Stores{}
			stores.ServiceAccountCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

			expectations = &util.Expectations{}

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().CoreV1().Return(coreclientset.CoreV1()).AnyTimes()

			kv = &v1.KubeVirt{}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should not patch ServiceAccount on sync when they are equal", func() {

			pr := newServiceAccount()

			version, imageRegistry, id := getTargetVersionRegistryID(kv)
			injectOperatorMetadata(kv, &pr.ObjectMeta, version, imageRegistry, id, true)

			stores.ServiceAccountCache.Add(pr)

			r := &Reconciler{
				kv:           kv,
				stores:       stores,
				clientset:    clientset,
				expectations: expectations,
			}

			Expect(r.createOrUpdateServiceAccount(pr)).To(BeNil())
		})

		It("should patch ServiceAccount on sync when they are not equal", func() {
			pr := newServiceAccount()
			version, imageRegistry, id := getTargetVersionRegistryID(kv)
			injectOperatorMetadata(kv, &pr.ObjectMeta, version, imageRegistry, id, true)

			stores.ServiceAccountCache.Add(pr)

			r := &Reconciler{
				kv:           kv,
				stores:       stores,
				clientset:    clientset,
				expectations: expectations,
			}

			requiredPR := pr.DeepCopy()
			newAnnotation := map[string]string{
				"something": "new",
			}
			requiredPR.ObjectMeta.Annotations = newAnnotation

			patched := false
			coreclientset.Fake.PrependReactor("patch", "serviceaccounts", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)
				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				patched = true

				obj, err := json.Marshal(pr)
				Expect(err).To(BeNil())

				obj, err = patch.Apply(obj)
				Expect(err).To(BeNil())

				pr := &corev1.ServiceAccount{}
				Expect(json.Unmarshal(obj, pr)).To(Succeed())
				Expect(pr.ObjectMeta.Annotations).To(Equal(newAnnotation))

				return true, pr, nil
			})

			Expect(r.createOrUpdateServiceAccount(requiredPR)).To(BeNil())
			Expect(patched).To(BeTrue())
		})
	})

	Context("should handle service endpoint updates", func() {

		config := getConfig("fake-registry", "v9.9.9")

		table.DescribeTable("with either patch",
			func(cachedService *corev1.Service,
				targetService *corev1.Service,
				expectLabelsAnnotationsPatch bool,
				expectSpecPatch bool) {

				// kv := &v1.KubeVirt{
				// 	ObjectMeta: metav1.ObjectMeta{
				// 		Name:       "test-install",
				// 		Namespace:  "default",
				// 		Generation: int64(1),
				// 	},
				// 	Spec: v1.KubeVirtSpec{
				// 		ImageTag:      config.GetKubeVirtVersion(),
				// 		ImageRegistry: config.GetImageRegistry(),
				// 	},
				// }
				// config.SetTargetDeploymentConfig(kv)

				Expect(hasImmutableFieldChanged(targetService, cachedService)).To(BeFalse())
				ops, err := generateServicePatch(cachedService, targetService)
				Expect(err).To(BeNil())

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
			table.Entry("should do nothing if cached service has ClusterIP set and target does not (clusterIP is dynamically assigned when empty)",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
							v1.KubeVirtGenerationAnnotation:        "1",
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
							v1.KubeVirtGenerationAnnotation:        "1",
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				false, false),
			table.Entry("should update labels, annotations on update",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:    "oldversion",
							v1.InstallStrategyRegistryAnnotation:   "oldversion",
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
				true, false),
			table.Entry("no-op with identical specs",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
							v1.KubeVirtGenerationAnnotation:        "1",
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
							v1.KubeVirtGenerationAnnotation:        "1",
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
				false, false),
			table.Entry("should patch spec when selectors differ",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.InstallStrategyVersionAnnotation:    "old",
							v1.InstallStrategyRegistryAnnotation:   "old",
							v1.InstallStrategyIdentifierAnnotation: "old",
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
							v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
							v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
							v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
						},
						Labels: map[string]string{
							v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
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
				true, true),
		)

		table.DescribeTable("complete replacement",
			func(cachedService *corev1.Service,
				targetService *corev1.Service) {

				shouldDeleteAndReplace := hasImmutableFieldChanged(targetService, cachedService)
				Expect(shouldDeleteAndReplace).To(BeTrue())
			},
			table.Entry("should delete and recreate service if of mixed 'type'.",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.KubeVirtGenerationAnnotation: "1",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.KubeVirtGenerationAnnotation: "1",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				}),
			table.Entry("should delete and recreate service if not of type ClusterIP.",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.KubeVirtGenerationAnnotation: "1",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.KubeVirtGenerationAnnotation: "1",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
					},
				}),
			table.Entry("should delete and recreate service if ClusterIP changes (clusterIP is not mutable)",
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							v1.KubeVirtGenerationAnnotation: "1",
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
							v1.KubeVirtGenerationAnnotation: "1",
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.1.1.1",
						Type:      corev1.ServiceTypeClusterIP,
					},
				}),
		)
	})
})
