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
	"crypto/tls"
	"encoding/json"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/mock/gomock"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Apply", func() {

	Context("Services", func() {

		It("should not patch if ClusterIp is empty during update", func() {
			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeClusterIP
			cachedService.Spec.ClusterIP = "10.10.10.10"

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeClusterIP
			service.Spec.ClusterIP = ""

			Expect(generateServicePatch(cachedService, service)).To(BeEmpty())
		})

		It("should replace if ClusterIp is not empty during update and ip changes", func() {
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

	Context("should reconcile configmap", func() {

		var clientset *kubecli.MockKubevirtClient
		var ctrl *gomock.Controller
		var coreclientset *fake.Clientset
		var expectations *util.Expectations
		var kv *v1.KubeVirt
		var stores util.Stores

		operatorNamespace := "opNamespace"
		queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		duration := &metav1.Duration{
			Duration: time.Hour,
		}

		createCrt := func() *tls.Certificate {
			caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour)

			encodedCert := cert.EncodeCertPEM(caKeyPair.Cert)
			encodedKey := cert.EncodePrivateKeyPEM(caKeyPair.Key)

			crt, err := tls.X509KeyPair(encodedCert, encodedKey)
			Expect(err).ToNot(HaveOccurred())
			leaf, err := cert.ParseCertsPEM(encodedCert)
			Expect(err).ToNot(HaveOccurred())
			crt.Leaf = leaf[0]

			return &crt
		}

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
			coreclientset = fake.NewSimpleClientset()

			coreclientset.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			stores = util.Stores{}
			stores.ConfigMapCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

			expectations = &util.Expectations{}

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().CoreV1().Return(coreclientset.CoreV1()).AnyTimes()

			kv = &v1.KubeVirt{}
		})

		It("should not patch ConfigMap on sync", func() {
			requiredCMs := components.NewCAConfigMaps(operatorNamespace)
			var requiredCM *corev1.ConfigMap
			for _, cm := range requiredCMs {
				if cm.Name == components.KubeVirtCASecretName {
					requiredCM = cm
				}
			}
			version, imageRegistry, id := getTargetVersionRegistryID(kv)
			injectOperatorMetadata(kv, &requiredCMs[0].ObjectMeta, version, imageRegistry, id, true)

			existingCM := requiredCM.DeepCopy()
			crt := createCrt()

			bundle, _, err := components.MergeCABundle(crt, []byte(cert.EncodeCertPEM(crt.Leaf)), time.Hour)
			Expect(err).ToNot(HaveOccurred())

			existingCM.Data = map[string]string{
				components.CABundleKey: string(bundle),
			}

			stores.ConfigMapCache.Add(existingCM)

			r := &Reconciler{
				kv:           kv,
				stores:       stores,
				clientset:    clientset,
				expectations: expectations,
			}

			_, err = r.createOrUpdateKubeVirtCAConfigMap(queue, crt, duration, requiredCM)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should patch ConfigMap on sync when not parsable", func() {
			notRSAParsableString := "something not parsable"
			requiredCMs := components.NewCAConfigMaps(operatorNamespace)
			var requiredCM *corev1.ConfigMap
			for _, cm := range requiredCMs {
				if cm.Name == components.KubeVirtCASecretName {
					requiredCM = cm
				}
			}
			version, imageRegistry, id := getTargetVersionRegistryID(kv)
			injectOperatorMetadata(kv, &requiredCM.ObjectMeta, version, imageRegistry, id, true)

			existingCM := requiredCM.DeepCopy()
			existingCM.Data = map[string]string{
				components.CABundleKey: notRSAParsableString,
			}
			stores.ConfigMapCache.Add(existingCM)

			r := &Reconciler{
				kv:           kv,
				stores:       stores,
				clientset:    clientset,
				expectations: expectations,
			}

			patched := false
			coreclientset.Fake.PrependReactor("patch", "configmaps", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)
				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				obj, err := json.Marshal(existingCM)
				Expect(err).ToNot(HaveOccurred())

				obj, err = patch.Apply(obj)
				Expect(err).ToNot(HaveOccurred())

				pr := &corev1.ConfigMap{}
				Expect(json.Unmarshal(obj, existingCM)).To(Succeed())
				Expect(existingCM.Data[components.CABundleKey]).ToNot(Equal(notRSAParsableString))

				patched = true
				return true, pr, nil
			})

			crt := createCrt()

			_, err := r.createOrUpdateKubeVirtCAConfigMap(queue, crt, duration, requiredCM)
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
		})

		It("should patch ConfigMap on sync when CA expired", func() {
			requiredCMs := components.NewCAConfigMaps(operatorNamespace)
			var requiredCM *corev1.ConfigMap
			for _, cm := range requiredCMs {
				if cm.Name == components.KubeVirtCASecretName {
					requiredCM = cm
				}
			}
			version, imageRegistry, id := getTargetVersionRegistryID(kv)
			injectOperatorMetadata(kv, &requiredCM.ObjectMeta, version, imageRegistry, id, true)

			existingCM := requiredCM.DeepCopy()
			crt := createCrt()

			bundle, _, err := components.MergeCABundle(crt, []byte(cert.EncodeCertPEM(crt.Leaf)), time.Hour)
			Expect(err).ToNot(HaveOccurred())

			existingCM.Data = map[string]string{
				components.CABundleKey: string(bundle),
			}
			stores.ConfigMapCache.Add(existingCM)

			r := &Reconciler{
				kv:           kv,
				stores:       stores,
				clientset:    clientset,
				expectations: expectations,
			}

			patched := false
			coreclientset.Fake.PrependReactor("patch", "configmaps", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				patched = true
				return true, &corev1.ConfigMap{}, nil
			})

			updatedCrt := createCrt()

			_, err = r.createOrUpdateKubeVirtCAConfigMap(queue, updatedCrt, duration, requiredCM)
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
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

			Expect(r.createOrUpdateServiceAccount(pr)).To(Succeed())
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
				Expect(err).ToNot(HaveOccurred())

				obj, err = patch.Apply(obj)
				Expect(err).ToNot(HaveOccurred())

				pr := &corev1.ServiceAccount{}
				Expect(json.Unmarshal(obj, pr)).To(Succeed())
				Expect(pr.ObjectMeta.Annotations).To(Equal(newAnnotation))

				return true, pr, nil
			})

			Expect(r.createOrUpdateServiceAccount(requiredPR)).To(Succeed())
			Expect(patched).To(BeTrue())
		})
	})

	Context("should handle service endpoint updates", func() {

		config := getConfig("fake-registry", "v9.9.9")

		DescribeTable("with either patch",
			func(cachedService *corev1.Service,
				targetService *corev1.Service,
				expectLabelsAnnotationsPatch bool,
				expectSpecPatch bool) {

				Expect(hasImmutableFieldChanged(targetService, cachedService)).To(BeFalse())
				ops, err := generateServicePatch(cachedService, targetService)
				Expect(err).ToNot(HaveOccurred())

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
					Expect(ops).To(BeEmpty())
				}
			},
			Entry("should do nothing if cached service has ClusterIP set and target does not (clusterIP is dynamically assigned when empty)",
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
			Entry("should update labels, annotations on update",
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
							"prometheus.kubevirt.io": "true",
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
							"prometheus.kubevirt.io": "true",
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
			Entry("no-op with identical specs",
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
			Entry("should patch spec when selectors differ",
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

		DescribeTable("complete replacement",
			func(cachedService *corev1.Service,
				targetService *corev1.Service) {

				shouldDeleteAndReplace := hasImmutableFieldChanged(targetService, cachedService)
				Expect(shouldDeleteAndReplace).To(BeTrue())
			},
			Entry("should delete and recreate service if of mixed 'type'.",
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
			Entry("should delete and recreate service if not of type ClusterIP.",
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
			Entry("should delete and recreate service if ClusterIP changes (clusterIP is not mutable)",
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
