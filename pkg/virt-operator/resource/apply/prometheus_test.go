package apply

import (
	"encoding/json"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	jsonpatch "github.com/evanphx/json-patch"

	promclientfake "kubevirt.io/client-go/generated/prometheus-operator/clientset/versioned/fake"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply Prometheus", func() {
	var clientset *kubecli.MockKubevirtClient
	var ctrl *gomock.Controller
	var extClient *extclientfake.Clientset
	var promClient *promclientfake.Clientset
	var expectations *util.Expectations
	var kv *v1.KubeVirt
	var stores util.Stores

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

		extClient = extclientfake.NewSimpleClientset()

		extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		stores = util.Stores{}
		stores.ServiceMonitorCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.PrometheusRuleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{}

		clientset = kubecli.NewMockKubevirtClient(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()

		promClient = promclientfake.NewSimpleClientset()
		clientset.EXPECT().PrometheusClient().Return(promClient).AnyTimes()

		kv = &v1.KubeVirt{}
	})

	It("should not patch ServiceMonitor on sync when they are equal", func() {

		sm := components.NewServiceMonitorCR("namespace", "mNamespace", true)

		version, imageRegistry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(kv, &sm.ObjectMeta, version, imageRegistry, id, true)

		stores.ServiceMonitorCache.Add(sm)

		r := &Reconciler{
			kv:           kv,
			stores:       stores,
			clientset:    clientset,
			expectations: expectations,
		}

		Expect(r.createOrUpdateServiceMonitor(sm)).To(Succeed())
	})

	It("should patch ServiceMonitor on sync when they are equal", func() {
		sm := components.NewServiceMonitorCR("namespace", "mNamespace", true)

		version, imageRegistry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(kv, &sm.ObjectMeta, version, imageRegistry, id, true)

		stores.ServiceMonitorCache.Add(sm)

		r := &Reconciler{
			kv:           kv,
			stores:       stores,
			clientset:    clientset,
			expectations: expectations,
		}

		requiredSM := sm.DeepCopy()
		updatedEndpoints := []promv1.Endpoint{
			{
				Port: "metrics-update",
			},
		}
		requiredSM.Spec.Endpoints = updatedEndpoints

		patched := false
		promClient.Fake.PrependReactor("patch", "servicemonitors", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(testing.PatchActionImpl)
			patch, err := jsonpatch.DecodePatch(a.Patch)
			Expect(err).ToNot(HaveOccurred())

			patched = true

			obj, err := json.Marshal(sm)
			Expect(err).ToNot(HaveOccurred())

			obj, err = patch.Apply(obj)
			Expect(err).ToNot(HaveOccurred())

			sm := &promv1.ServiceMonitor{}
			Expect(json.Unmarshal(obj, sm)).To(Succeed())
			Expect(sm.Spec.Endpoints).To(Equal(updatedEndpoints))

			return true, sm, nil
		})

		Expect(r.createOrUpdateServiceMonitor(requiredSM)).To(Succeed())
		Expect(patched).To(BeTrue())
	})

	It("should not patch PrometheusRules on sync when they are equal", func() {

		pr := components.NewPrometheusRuleCR("namespace")

		version, imageRegistry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(kv, &pr.ObjectMeta, version, imageRegistry, id, true)

		stores.PrometheusRuleCache.Add(pr)

		r := &Reconciler{
			kv:           kv,
			stores:       stores,
			clientset:    clientset,
			expectations: expectations,
		}

		Expect(r.createOrUpdatePrometheusRule(pr)).To(Succeed())
	})

	It("should patch PrometheusRules on sync when they are equal", func() {

		pr := components.NewPrometheusRuleCR("namespace")

		version, imageRegistry, id := getTargetVersionRegistryID(kv)
		injectOperatorMetadata(kv, &pr.ObjectMeta, version, imageRegistry, id, true)

		stores.PrometheusRuleCache.Add(pr)

		r := &Reconciler{
			kv:           kv,
			stores:       stores,
			clientset:    clientset,
			expectations: expectations,
		}

		requiredPR := pr.DeepCopy()
		updatedGroups := []promv1.RuleGroup{
			{
				Name: "Updated",
			},
		}
		requiredPR.Spec.Groups = updatedGroups

		patched := false
		promClient.Fake.PrependReactor("patch", "prometheusrules", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(testing.PatchActionImpl)
			patch, err := jsonpatch.DecodePatch(a.Patch)
			Expect(err).ToNot(HaveOccurred())

			patched = true

			obj, err := json.Marshal(pr)
			Expect(err).ToNot(HaveOccurred())

			obj, err = patch.Apply(obj)
			Expect(err).ToNot(HaveOccurred())

			pr := &promv1.PrometheusRule{}
			Expect(json.Unmarshal(obj, pr)).To(Succeed())
			Expect(pr.Spec.Groups).To(Equal(updatedGroups))
			Expect(pr.Spec.Groups).To(HaveLen(1))

			return true, pr, nil
		})

		Expect(r.createOrUpdatePrometheusRule(requiredPR)).To(Succeed())
		Expect(patched).To(BeTrue())
	})
})
