package installstrategy

import (
	"fmt"

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
	Id        = "42"
)

var _ = Describe("Create", func() {

	Context("on calling syncPodDisruptionBudgetForDeployment", func() {

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
		var shouldPatchFail bool
		var created bool
		var shouldCreateFail bool

		BeforeEach(func() {

			ctrl := gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			patched = false
			shouldPatchFail = false
			created = false
			shouldCreateFail = false

			kubeClient = fake.NewSimpleClientset()

			kubeClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				if shouldPatchFail {
					return true, nil, fmt.Errorf("Patch failed!")
				}
				patched = true
				return true, nil, nil
			})

			kubeClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				if shouldCreateFail {
					return true, nil, fmt.Errorf("Create failed!")
				}
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

			deployment, err = components.NewApiServerDeployment(Namespace, Registry, "", Version, corev1.PullIfNotPresent, "verbosity", map[string]string{})
			Expect(err).ToNot(HaveOccurred())

			cachedPodDisruptionBudget = components.NewPodDisruptionBudgetForDeployment(deployment)
		})

		It("should not fail creation", func() {
			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(created).To(BeTrue())
			Expect(patched).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not fail patching", func() {
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(patched).To(BeTrue())
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should skip patching of same version", func() {
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version
			kv.Status.TargetDeploymentID = Id

			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget
			injectOperatorMetadata(kv, &cachedPodDisruptionBudget.ObjectMeta, Version, Registry, Id)

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return create error", func() {
			shouldCreateFail = true

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(err).To(HaveOccurred())
			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
		})

		It("should return patch error", func() {
			shouldPatchFail = true
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget

			err = syncPodDisruptionBudgetForDeployment(deployment, clientset, kv, expectations, stores)

			Expect(err).To(HaveOccurred())
			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
		})

	})

})
