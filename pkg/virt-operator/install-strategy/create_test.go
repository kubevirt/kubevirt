package installstrategy

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

type MockStore struct {
	get interface{}
}

func (m *MockStore) Add(obj interface{}) error    { return nil }
func (m *MockStore) Update(obj interface{}) error { return nil }
func (m *MockStore) Delete(obj interface{}) error { return nil }
func (m *MockStore) List() []interface{}          { return nil }
func (m *MockStore) ListKeys() []string           { return nil }
func (m *MockStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	item = m.get
	if m.get != nil {
		exists = true
	}
	return
}
func (m *MockStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}
func (m *MockStore) Replace([]interface{}, string) error { return nil }
func (m *MockStore) Resync() error                       { return nil }

const (
	Namespace = "ns"
	Version   = "1.0"
	Registry  = "rep"
)

var _ = Describe("Create", func() {

	Context("method syncPodDisruptionBudgetForDeployment", func() {

		var deployment *appsv1.Deployment
		var err error
		var clientset *kubecli.MockKubevirtClient
		var kv *v1.KubeVirt
		var expectations *util.Expectations
		var stores util.Stores
		var mockPodDisruptionBudgetCacheStore *MockStore
		var kubeClient *fake.Clientset
		var cachedPodDisruptionBudget *v1beta1.PodDisruptionBudget
		var patched bool
		var created bool

		BeforeEach(func() {

			ctrl := gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			patched = false
			created = false

			kubeClient = fake.NewSimpleClientset()

			kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				patched = true
				return true, nil, nil
			})

			kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				created = true
				return true, nil, nil
			})

			stores = util.Stores{}
			mockPodDisruptionBudgetCacheStore = &MockStore{}
			stores.PodDisruptionBudgetCache = mockPodDisruptionBudgetCacheStore

			expectations = &util.Expectations{}
			expectations.PodDisruptionBudget = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudgets"))

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().PolicyV1beta1().Return(kubeClient.PolicyV1beta1()).AnyTimes()
			kv = &v1.KubeVirt{}

			deployment, err = components.NewApiServerDeployment(Namespace, Registry, Version, corev1.PullIfNotPresent, "verbosity")
			Expect(err).ToNot(HaveOccurred())

			cachedPodDisruptionBudget = components.NewPodDisruptionBudgetForDeployment(deployment)
		})

		AfterEach(func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("creation should not fail", func() {
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(created).To(BeTrue())
			Expect(patched).To(BeFalse())
		})

		It("patching should not fail", func() {
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(patched).To(BeTrue())
			Expect(created).To(BeFalse())
		})

		It("patching with same version should be skipped", func() {
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version

			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget
			injectOperatorMetadata(kv, &cachedPodDisruptionBudget.ObjectMeta, Version, Registry)

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
		})

	})

})
