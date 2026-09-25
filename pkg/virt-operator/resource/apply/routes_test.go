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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	routev1fake "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1/fake"
	"go.uber.org/mock/gomock"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	externalRouteAnnotation      = "haproxy.router.openshift.io/timeout"
	externalRouteAnnotationValue = "60m"
)

var _ = Describe("Apply Routes", func() {
	var (
		ctrl        *gomock.Controller
		virtClient  *kubecli.MockKubevirtClient
		routeClient *routev1fake.FakeRouteV1
		stores      util.Stores
		kv          *v1.KubeVirt
		reconciler  *Reconciler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		k8sClient := fake.NewSimpleClientset()
		routeClient = &routev1fake.FakeRouteV1{
			Fake: &k8sClient.Fake,
		}

		virtClient.EXPECT().
			RouteClient().
			Return(routeClient).
			AnyTimes()

		stores = util.Stores{
			RouteCache: cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		}

		kv = &v1.KubeVirt{}
		kv.Status.TargetKubeVirtRegistry = Registry
		kv.Status.TargetKubeVirtVersion = Version
		kv.Status.TargetDeploymentID = Id

		reconciler = &Reconciler{
			kv:         kv,
			stores:     stores,
			virtClient: virtClient,
		}
	})

	It("should preserve additional Route annotations during reconciliation", func() {
		cachedRoute := components.NewExportProxyRoute(Namespace)

		version, registry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(
			kv,
			&cachedRoute.ObjectMeta,
			version,
			registry,
			id,
			true,
		)

		cachedRoute.Spec.TLS.DestinationCACertificate = "old-ca"
		cachedRoute.Annotations[externalRouteAnnotation] = externalRouteAnnotationValue

		Expect(stores.RouteCache.Add(cachedRoute)).To(Succeed())

		var patchedRoute *routev1.Route

		routeClient.Fake.PrependReactor(
			"patch",
			"routes",
			func(action testing.Action) (bool, runtime.Object, error) {
				patchedRoute = applyRoutePatch(action, cachedRoute)
				return true, patchedRoute, nil
			},
		)

		desiredRoute := components.NewExportProxyRoute(Namespace)

		Expect(
			reconciler.syncRoute(
				desiredRoute,
				[]byte("new-ca"),
			),
		).To(Succeed())

		Expect(patchedRoute).ToNot(BeNil())

		Expect(
			patchedRoute.Spec.TLS.DestinationCACertificate,
		).To(Equal("new-ca"))

		Expect(
			patchedRoute.Annotations,
		).To(HaveKeyWithValue(
			externalRouteAnnotation,
			externalRouteAnnotationValue,
		))
	})

	It("should preserve additional Route annotations while operator-owned annotations win", func() {
		cachedRoute := components.NewExportProxyRoute(Namespace)

		version, registry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(
			kv,
			&cachedRoute.ObjectMeta,
			version,
			registry,
			id,
			true,
		)

		cachedRoute.Spec.TLS.DestinationCACertificate = "old-ca"

		// User-provided annotation: this must be preserved.
		cachedRoute.Annotations[externalRouteAnnotation] = externalRouteAnnotationValue

		// Simulate a user modifying an annotation owned by virt-operator.
		cachedRoute.Annotations[v1.InstallStrategyVersionAnnotation] = "overridden"

		Expect(stores.RouteCache.Add(cachedRoute)).To(Succeed())

		var patchedRoute *routev1.Route

		routeClient.Fake.PrependReactor(
			"patch",
			"routes",
			func(action testing.Action) (bool, runtime.Object, error) {
				patchedRoute = applyRoutePatch(action, cachedRoute)
				return true, patchedRoute, nil
			},
		)

		desiredRoute := components.NewExportProxyRoute(Namespace)

		Expect(
			reconciler.syncRoute(
				desiredRoute,
				[]byte("new-ca"),
			),
		).To(Succeed())

		Expect(patchedRoute).ToNot(BeNil())

		// The reconciliation must still apply the operator-owned spec change.
		Expect(
			patchedRoute.Spec.TLS.DestinationCACertificate,
		).To(Equal("new-ca"))

		// Unknown/user-provided annotation must survive reconciliation.
		Expect(
			patchedRoute.Annotations,
		).To(HaveKeyWithValue(
			externalRouteAnnotation,
			externalRouteAnnotationValue,
		))

		// Operator-owned annotation must be restored to the desired value.
		Expect(
			patchedRoute.Annotations,
		).To(HaveKeyWithValue(
			v1.InstallStrategyVersionAnnotation,
			version,
		))
	})
})

func applyRoutePatch(
	action testing.Action,
	cached *routev1.Route,
) *routev1.Route {
	a := action.(testing.PatchActionImpl)

	decodedPatch, err := jsonpatch.DecodePatch(a.Patch)
	Expect(err).ToNot(HaveOccurred())

	obj, err := json.Marshal(cached)
	Expect(err).ToNot(HaveOccurred())

	obj, err = decodedPatch.Apply(obj)
	Expect(err).ToNot(HaveOccurred())

	updated := &routev1.Route{}
	Expect(json.Unmarshal(obj, updated)).To(Succeed())

	return updated
}
