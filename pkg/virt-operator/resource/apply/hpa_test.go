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

package apply

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply HPAs", func() {
	var (
		ctrl           *gomock.Controller
		k8sClient      *fake.Clientset
		stores         util.Stores
		virtClient     *kubecli.MockKubevirtClient
		expectations   *util.Expectations
		kv             *v1.KubeVirt
		r              *Reconciler
		mockGeneration int64
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
		k8sClient = fake.NewSimpleClientset()

		stores = util.Stores{}
		stores.HorizontalPodAutoscalerCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{
			HorizontalPodAutoscaler: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("HorizontalPodAutoscalers")),
		}

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		kv = &v1.KubeVirt{}

		r = &Reconciler{
			kv:           kv,
			kvKey:        Namespace + "/test",
			stores:       stores,
			virtClient:   virtClient,
			k8sClient:    k8sClient,
			expectations: expectations,
		}

		kv.Status.TargetKubeVirtRegistry = Registry
		kv.Status.TargetKubeVirtVersion = Version
		kv.Status.TargetDeploymentID = Id

		mockGeneration = 123
	})

	Context("export-proxy HPA", func() {
		It("should create an HPA with min 2 and max 20 replicas", func() {
			createFakeNodes(k8sClient, 2, 0)

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			k8sClient.Fake.PrependReactor("create", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(testing.CreateActionImpl)
				createdHPA := createAction.Object.(*autoscalingv2.HorizontalPodAutoscaler)
				Expect(createdHPA.Name).To(Equal(components.VirtExportProxyHPAName))
				Expect(*createdHPA.Spec.MinReplicas).To(Equal(int32(2)))
				Expect(createdHPA.Spec.MaxReplicas).To(Equal(int32(20)))
				Expect(createdHPA.Spec.ScaleTargetRef.Name).To(Equal(components.VirtExportProxyName))
				Expect(createdHPA.Spec.Metrics).To(HaveLen(1))
				Expect(createdHPA.Spec.Metrics[0].Type).To(Equal(autoscalingv2.PodsMetricSourceType))
				Expect(createdHPA.Spec.Metrics[0].Pods).NotTo(BeNil())
				Expect(createdHPA.Spec.Metrics[0].Pods.Metric.Name).To(Equal(components.ExportProxyActiveTransfersMetricName))
				Expect(createdHPA.Spec.Metrics[0].Pods.Target.Type).To(Equal(autoscalingv2.AverageValueMetricType))
				Expect(createdHPA.Spec.Metrics[0].Pods.Target.AverageValue.String()).To(Equal("50"))
				return true, createdHPA, nil
			})

			Expect(r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)).To(Succeed())
		})

		It("should not patch HPA on sync when it is up-to-date", func() {
			createFakeNodes(k8sClient, 2, 0)

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			cachedHPA := components.NewExportProxyHorizontalPodAutoscaler(exportProxy)
			injectOperatorMetadata(kv, &cachedHPA.ObjectMeta, Version, Registry, Id, true)
			cachedHPA.SetGeneration(mockGeneration)
			SetGeneration(&kv.Status.Generations, cachedHPA)
			Expect(stores.HorizontalPodAutoscalerCache.Add(cachedHPA)).To(Succeed())
			Expect(GetExpectedGeneration(cachedHPA, kv.Status.Generations)).To(Equal(mockGeneration))

			k8sClient.Fake.PrependReactor("create", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "create should not be called")
				return true, nil, nil
			})
			k8sClient.Fake.PrependReactor("patch", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "patch should not be called")
				return true, nil, nil
			})

			Expect(r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)).To(Succeed())
		})

		It("should patch stale maxReplicas", func() {
			createFakeNodes(k8sClient, 2, 0)

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			cachedHPA := components.NewExportProxyHorizontalPodAutoscaler(exportProxy)
			cachedHPA.Spec.MaxReplicas = 10
			cachedHPA.Annotations = make(map[string]string)
			cachedHPA.SetGeneration(mockGeneration)
			SetGeneration(&kv.Status.Generations, cachedHPA)
			injectOperatorMetadata(kv, &cachedHPA.ObjectMeta, Version, Registry, Id, true)
			Expect(stores.HorizontalPodAutoscalerCache.Add(cachedHPA)).To(Succeed())

			patched := false
			k8sClient.Fake.PrependReactor("patch", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)

				patchOps, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				obj, err := json.Marshal(cachedHPA)
				Expect(err).ToNot(HaveOccurred())

				obj, err = patchOps.Apply(obj)
				Expect(err).ToNot(HaveOccurred())

				hpa := &autoscalingv2.HorizontalPodAutoscaler{}
				Expect(json.Unmarshal(obj, hpa)).To(Succeed())
				Expect(hpa.Spec.MaxReplicas).To(Equal(int32(20)))

				patched = true
				return true, hpa, nil
			})

			Expect(r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)).To(Succeed())
			Expect(patched).To(BeTrue())
		})

		It("should return an error when node listing fails", func() {
			k8sClient.Fake.PrependReactor("list", "nodes", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, fmt.Errorf("node list unavailable")
			})

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			k8sClient.Fake.PrependReactor("create", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "create should not be called")
				return true, nil, nil
			})
			k8sClient.Fake.PrependReactor("patch", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "patch should not be called")
				return true, nil, nil
			})
			k8sClient.Fake.PrependReactor("delete", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "delete should not be called")
				return true, nil, nil
			})

			err := r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)
			Expect(err).To(MatchError(ContainSubstring("node list unavailable")))
		})

		It("should not create an HPA when virt-api would run at most one replica", func() {
			createFakeNodes(k8sClient, 1, 0)

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			k8sClient.Fake.PrependReactor("create", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(true).To(BeFalse(), "create should not be called")
				return true, nil, nil
			})

			Expect(r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)).To(Succeed())
		})

		It("should delete the HPA when virt-api would run at most one replica", func() {
			createFakeNodes(k8sClient, 1, 0)

			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			cachedHPA := components.NewExportProxyHorizontalPodAutoscaler(exportProxy)
			injectOperatorMetadata(kv, &cachedHPA.ObjectMeta, Version, Registry, Id, true)
			Expect(stores.HorizontalPodAutoscalerCache.Add(cachedHPA)).To(Succeed())

			deleted := false
			k8sClient.Fake.PrependReactor("delete", "horizontalpodautoscalers", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(action.(testing.DeleteActionImpl).Name).To(Equal(components.VirtExportProxyHPAName))
				deleted = true
				return true, nil, nil
			})

			Expect(r.syncExportProxyHorizontalPodAutoscaler(exportProxy, nil)).To(Succeed())
			Expect(deleted).To(BeTrue())
			Expect(expectations.HorizontalPodAutoscaler.SatisfiedExpectations(r.kvKey)).To(BeFalse())
		})
	})
})
