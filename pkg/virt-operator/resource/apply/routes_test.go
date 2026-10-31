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
	routeTimeoutAnnotation = "haproxy.router.openshift.io/timeout"
	routeTimeoutValue      = "10m"
	exportProxyCABundle    = "test-ca"
)

var _ = Describe("Apply Routes", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var routeClient *routev1fake.FakeRouteV1
	var stores util.Stores
	var kv *v1.KubeVirt
	var r *Reconciler

	existingExportProxyRoute := func(withTimeout bool) *routev1.Route {
		route := components.NewExportProxyRoute(Namespace)
		if !withTimeout {
			delete(route.Annotations, routeTimeoutAnnotation)
		}
		version, imageRegistry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(kv, &route.ObjectMeta, version, imageRegistry, id, true)
		route.Spec.TLS.DestinationCACertificate = exportProxyCABundle
		return route
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		k8sClient := fake.NewSimpleClientset()
		routeClient = &routev1fake.FakeRouteV1{
			Fake: &k8sClient.Fake,
		}
		virtClient.EXPECT().RouteClient().Return(routeClient).AnyTimes()

		stores = util.Stores{}
		stores.RouteCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		kv = &v1.KubeVirt{}
		kv.Status.TargetKubeVirtRegistry = Registry
		kv.Status.TargetKubeVirtVersion = Version
		kv.Status.TargetDeploymentID = Id

		r = &Reconciler{
			kv:         kv,
			stores:     stores,
			virtClient: virtClient,
		}
	})

	It("should patch an existing Route missing the timeout annotation on reconcile", func() {
		cached := existingExportProxyRoute(false)
		Expect(cached.Annotations).ToNot(HaveKey(routeTimeoutAnnotation))
		Expect(stores.RouteCache.Add(cached)).To(Succeed())

		patched := false
		routeClient.Fake.PrependReactor("patch", "routes", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(testing.PatchActionImpl)
			patch, err := jsonpatch.DecodePatch(a.Patch)
			Expect(err).ToNot(HaveOccurred())

			obj, err := json.Marshal(cached)
			Expect(err).ToNot(HaveOccurred())
			obj, err = patch.Apply(obj)
			Expect(err).ToNot(HaveOccurred())

			updated := &routev1.Route{}
			Expect(json.Unmarshal(obj, updated)).To(Succeed())
			Expect(updated.Annotations).To(HaveKeyWithValue(routeTimeoutAnnotation, routeTimeoutValue))

			patched = true
			return true, updated, nil
		})

		Expect(r.syncRoute(components.NewExportProxyRoute(Namespace), []byte(exportProxyCABundle))).To(Succeed())
		Expect(patched).To(BeTrue())
	})

	It("should not patch the export proxy Route when the timeout annotation is already set", func() {
		cached := existingExportProxyRoute(true)
		Expect(stores.RouteCache.Add(cached)).To(Succeed())

		routeClient.Fake.PrependReactor("patch", "routes", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			Fail("should not patch an up-to-date route")
			return true, nil, nil
		})

		Expect(r.syncRoute(components.NewExportProxyRoute(Namespace), []byte(exportProxyCABundle))).To(Succeed())
	})
})
