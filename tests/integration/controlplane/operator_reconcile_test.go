package controlplane_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

// Regression coverage: real API defaulting must not cause
// repeated writes when the operator reconciles an already converged resource.
var _ = Describe("Operator writes against a real API server", Label("integration"), Ordered, ContinueOnFailure, func() {
	var f *framework.Framework

	BeforeAll(func(ctx SpecContext) {
		f = framework.New()
		f.Start(context.Background())
	}, NodeTimeout(2*time.Minute))

	DescribeTable("should document no-op writes after convergence and drift repair",
		func(ctx SpecContext, setup operatorResourceSetup, introduceDrift bool) {
			ns := f.CreateNamespace(ctx)
			cfg := f.RESTConfig()
			writes := &operatorWriteRecorder{}
			cfg.Wrap(writes.wrap)
			operatorClient, err := kubernetes.NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			operatorAggregator, err := aggregator.NewForConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			observerAggregator, err := aggregator.NewForConfig(f.RESTConfig())
			Expect(err).NotTo(HaveOccurred())

			serviceFactory := informers.NewSharedInformerFactoryWithOptions(f.K8sClient(), 0, informers.WithNamespace(ns))
			serviceInformer := serviceFactory.Core().V1().Services().Informer()
			namespaceInformer := serviceFactory.Core().V1().Namespaces().Informer()
			apiServiceInformer := cache.NewSharedIndexInformer(
				cache.NewListWatchFromClient(
					observerAggregator.ApiregistrationV1().RESTClient(), "apiservices", metav1.NamespaceAll, fields.Everything(),
				),
				&apiregv1.APIService{}, 0, cache.Indexers{},
			)
			informerCtx, cancel := context.WithCancel(ctx)
			serviceFactory.Start(informerCtx.Done())
			var informerWG sync.WaitGroup
			informerWG.Go(func() { apiServiceInformer.RunWithContext(informerCtx) })
			DeferCleanup(func() {
				cancel()
				serviceFactory.Shutdown()
				informerWG.Wait()
			})
			syncCtx, stopSync := context.WithTimeout(ctx, 10*time.Second)
			defer stopSync()
			Expect(cache.WaitForCacheSync(
				syncCtx.Done(), serviceInformer.HasSynced, namespaceInformer.HasSynced, apiServiceInformer.HasSynced,
			)).To(BeTrue(), "operator informer caches did not sync within 10 seconds")

			stores := emptyOperatorStores()
			stores.ServiceCache = serviceInformer.GetStore()
			stores.APIServiceCache = apiServiceInformer.GetStore()
			stores.NamespaceCache = namespaceInformer.GetStore()
			expectations := &util.Expectations{
				Service:    controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Service")),
				APIService: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("APIService")),
			}
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
			DeferCleanup(queue.ShutDown)

			resource := setup(ctx, ns, f.K8sClient(), observerAggregator, stores)

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: ns},
				Spec: v1.KubeVirtSpec{Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						// This minimal strategy does not deploy the admission webhook.
						DisabledFeatureGates: []string{featuregate.ContainerPathVolumesGate},
					},
				}},
			}
			reconcile := func() {
				GinkgoHelper()
				r, err := apply.NewReconciler(
					kv,
					resource.strategy.deepCopy(), stores, util.OperatorConfig{}, f.VirtClient(), operatorClient,
					operatorAggregator.ApiregistrationV1().APIServices(), expectations,
					new(record.FakeRecorder), nil,
				)
				Expect(err).NotTo(HaveOccurred())
				complete, err := r.Sync(queue)
				Expect(err).NotTo(HaveOccurred())
				Expect(complete).To(BeTrue(), "each assertion requires a completed reconciliation")
			}

			// Synchronize on resourceVersion rather than sleeping: each pass must
			// see the server's defaulted response to the preceding write.
			observe := func() {
				rv := resource.get().GetResourceVersion()
				Eventually(ctx, func() string {
					obj, exists, err := resource.store.GetByKey(resource.key)
					Expect(err).NotTo(HaveOccurred())
					if !exists {
						return ""
					}
					return obj.(metav1.Object).GetResourceVersion()
				}, 10*time.Second, 10*time.Millisecond).Should(Equal(rv))
			}

			By("creating the resource through reconciliation and observing server defaults")
			reconcile()
			observe()
			if introduceDrift {
				By("repairing an operator-managed field changed by another client")
				resource.drift()
				observe()
				before := resource.get().GetResourceVersion()
				writes.reset()
				reconcile()
				Expect(writes.snapshot()).NotTo(BeEmpty(), "drift repair must send a write")
				Expect(resource.get().GetResourceVersion()).NotTo(Equal(before), "repair must change the stored object")
				resource.verifyRepair()
				observe()
			}

			// FIXME(lyarwood): each reconciliation incorrectly sends a no-op
			// PATCH after the resource has converged. Document that behavior for now;
			// change the write assertion to BeEmpty() when the bug is fixed.
			By("documenting one incorrect no-op PATCH per reconciliation after convergence")
			baselineRV := resource.get().GetResourceVersion()
			writes.reset()
			for range 3 {
				reconcile()
				observe()
				Expect(resource.get().GetResourceVersion()).To(Equal(baselineRV), "steady-state reconciliation must not change the object")
			}
			requests := writes.snapshot()
			Expect(requests).To(HaveLen(3), "currently one no-op PATCH per reconciliation")
			Expect(requests).To(HaveEach(HavePrefix("PATCH ")))
		},
		Entry("virt-api at steady state", serviceReconcileFixture, false, SpecTimeout(time.Minute)),
		Entry("virt-api after drift repair", serviceReconcileFixture, true, SpecTimeout(time.Minute)),
		Entry("APIService at steady state", apiServiceReconcileFixture, false, SpecTimeout(time.Minute)),
		Entry("APIService after drift repair", apiServiceReconcileFixture, true, SpecTimeout(time.Minute)),
	)
})

// Each table entry supplies the object-specific setup and operations; the
// reconciliation and request assertions are shared by both resource kinds.
type operatorResourceSetup func(context.Context, string, kubernetes.Interface, aggregator.Interface, util.Stores) operatorResourceFixture

type operatorResourceFixture struct {
	strategy     *operatorResourceStrategy
	store        cache.Store
	key          string
	get          func() metav1.Object
	drift        func()
	verifyRepair func()
}

func serviceReconcileFixture(
	ctx context.Context, ns string, client kubernetes.Interface, _ aggregator.Interface, stores util.Stores,
) operatorResourceFixture {
	GinkgoHelper()
	resource := operatorResourceFixture{strategy: &operatorResourceStrategy{}}
	desired := components.NewApiServerService(ns)
	expectedSelector := desired.DeepCopy().Spec.Selector
	resource.store, resource.key = stores.ServiceCache, ns+"/"+desired.Name
	resource.strategy.service = desired
	resource.get = func() metav1.Object {
		obj, err := client.CoreV1().Services(ns).Get(ctx, desired.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	}
	resource.drift = func() {
		payload, err := patch.New(patch.WithReplace("/spec/selector", map[string]string{"drift": "injected"})).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = client.CoreV1().Services(ns).Patch(ctx, desired.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())
	}
	resource.verifyRepair = func() {
		Expect(resource.get().(*corev1.Service).Spec.Selector).To(Equal(expectedSelector))
	}
	return resource
}

func apiServiceReconcileFixture(
	ctx context.Context, ns string, _ kubernetes.Interface, observerAggregator aggregator.Interface, stores util.Stores,
) operatorResourceFixture {
	GinkgoHelper()
	resource := operatorResourceFixture{strategy: &operatorResourceStrategy{}}
	desired := components.NewVirtAPIAPIServices(ns)[0]
	// APIServices are cluster-scoped; isolate this test from other cases.
	desired.Spec.Group = ns + ".example.com"
	desired.Name = desired.Spec.Version + "." + desired.Spec.Group
	resource.store, resource.key = stores.APIServiceCache, desired.Name
	resource.strategy.apiService = desired
	expectedPriority := desired.Spec.VersionPriority
	resource.get = func() metav1.Object {
		obj, err := observerAggregator.ApiregistrationV1().APIServices().Get(ctx, desired.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	}
	resource.drift = func() {
		payload, err := patch.New(patch.WithReplace("/spec/versionPriority", expectedPriority+1)).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = observerAggregator.ApiregistrationV1().APIServices().Patch(
			ctx, desired.Name, types.JSONPatchType, payload, metav1.PatchOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
	}
	resource.verifyRepair = func() {
		Expect(resource.get().(*apiregv1.APIService).Spec.VersionPriority).To(Equal(expectedPriority))
	}
	DeferCleanup(func(ctx SpecContext) {
		Expect(observerAggregator.ApiregistrationV1().APIServices().Delete(ctx, desired.Name, metav1.DeleteOptions{})).To(Succeed())
	})
	return resource
}

type operatorWriteRecorder struct {
	lock     sync.Mutex
	requests []string
}

func (r *operatorWriteRecorder) wrap(next http.RoundTripper) http.RoundTripper {
	return operatorRoundTripper(func(req *http.Request) (*http.Response, error) {
		isService := strings.HasPrefix(req.URL.Path, "/api/v1/namespaces/") && strings.Contains(req.URL.Path, "/services/")
		isAPIService := strings.HasPrefix(req.URL.Path, "/apis/apiregistration.k8s.io/v1/apiservices/")
		if (isService || isAPIService) && (req.Method == http.MethodPatch || req.Method == http.MethodPut) {
			var body []byte
			if req.Body != nil {
				var err error
				body, err = io.ReadAll(req.Body)
				req.Body.Close()
				req.Body = io.NopCloser(bytes.NewReader(body))
				if err != nil {
					return nil, fmt.Errorf("recording %s %s body: %w", req.Method, req.URL.Path, err)
				}
			}
			r.lock.Lock()
			r.requests = append(r.requests, fmt.Sprintf("%s %s %s", req.Method, req.URL.Path, body))
			r.lock.Unlock()
		}
		return next.RoundTrip(req)
	})
}

func (r *operatorWriteRecorder) reset() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.requests = nil
}

func (r *operatorWriteRecorder) snapshot() []string {
	r.lock.Lock()
	defer r.lock.Unlock()
	return append([]string(nil), r.requests...)
}

type operatorRoundTripper func(*http.Request) (*http.Response, error)

func (r operatorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

// Only the resource under test is in the install strategy. Other components
// have no desired objects or observed objects in this focused regression test.
type operatorResourceStrategy struct {
	install.Strategy
	service    *corev1.Service
	apiService *apiregv1.APIService
}

// Keep the fixture's desired state immutable, while allowing NewReconciler to
// apply customization to stable strategy objects for the duration of one pass.
func (s *operatorResourceStrategy) deepCopy() *operatorResourceStrategy {
	return &operatorResourceStrategy{
		service:    s.service.DeepCopy(),
		apiService: s.apiService.DeepCopy(),
	}
}

func (s *operatorResourceStrategy) Services() []*corev1.Service {
	if s.service == nil {
		return nil
	}
	return []*corev1.Service{s.service}
}

func (s *operatorResourceStrategy) APIServices() []*apiregv1.APIService {
	if s.apiService == nil {
		return nil
	}
	return []*apiregv1.APIService{s.apiService}
}

// Sync traverses cleanup caches even when the install strategy has no objects
// of those kinds, so unused caches must be empty stores rather than nil.
func emptyOperatorStores() util.Stores {
	newStore := func() cache.Store { return cache.NewStore(cache.MetaNamespaceKeyFunc) }
	return util.Stores{
		KubeVirtCache:                         newStore(),
		ServiceAccountCache:                   newStore(),
		ClusterRoleCache:                      newStore(),
		ClusterRoleBindingCache:               newStore(),
		RoleCache:                             newStore(),
		RoleBindingCache:                      newStore(),
		OperatorCrdCache:                      newStore(),
		ServiceCache:                          newStore(),
		DeploymentCache:                       newStore(),
		DaemonSetCache:                        newStore(),
		ValidationWebhookCache:                newStore(),
		MutatingWebhookCache:                  newStore(),
		APIServiceCache:                       newStore(),
		SCCCache:                              newStore(),
		RouteCache:                            newStore(),
		InstallStrategyConfigMapCache:         newStore(),
		InstallStrategyJobCache:               newStore(),
		InfrastructurePodCache:                newStore(),
		PodDisruptionBudgetCache:              newStore(),
		HorizontalPodAutoscalerCache:          newStore(),
		ServiceMonitorCache:                   newStore(),
		NamespaceCache:                        newStore(),
		PrometheusRuleCache:                   newStore(),
		SecretCache:                           newStore(),
		ConfigMapCache:                        newStore(),
		ValidatingAdmissionPolicyBindingCache: newStore(),
		ValidatingAdmissionPolicyCache:        newStore(),
		ClusterInstancetype:                   newStore(),
		ClusterPreference:                     newStore(),
		SynchronizationServiceCache:           newStore(),
		SynchronizationIngressCache:           newStore(),
		SynchronizationRouteCache:             newStore(),
	}
}
