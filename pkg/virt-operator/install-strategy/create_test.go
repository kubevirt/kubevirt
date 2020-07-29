package installstrategy

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

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
		var pdbClient *fake.Clientset
		var cachedPodDisruptionBudget *v1beta1.PodDisruptionBudget
		var patched bool
		var shouldPatchFail bool
		var created bool
		var shouldCreateFail bool
		var ctrl *gomock.Controller
		var extClient *extclientfake.Clientset

		BeforeEach(func() {

			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			patched = false
			shouldPatchFail = false
			created = false
			shouldCreateFail = false

			pdbClient = fake.NewSimpleClientset()
			extClient = extclientfake.NewSimpleClientset()

			pdbClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				if shouldPatchFail {
					return true, nil, fmt.Errorf("Patch failed!")
				}
				patched = true
				return true, nil, nil
			})

			pdbClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				if shouldCreateFail {
					return true, nil, fmt.Errorf("Create failed!")
				}
				created = true
				return true, nil, nil
			})
			extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			stores = util.Stores{}
			mockPodDisruptionBudgetCacheStore = &MockStore{}
			stores.PodDisruptionBudgetCache = mockPodDisruptionBudgetCacheStore
			stores.CrdCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

			expectations = &util.Expectations{}
			expectations.PodDisruptionBudget = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudgets"))

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().PolicyV1beta1().Return(pdbClient.PolicyV1beta1()).AnyTimes()
			clientset.EXPECT().ExtensionsClient().Return(extClient).AnyTimes()
			kv = &v1.KubeVirt{}

			deployment, err = components.NewApiServerDeployment(Namespace, Registry, "", Version, corev1.PullIfNotPresent, "verbosity", map[string]string{})
			Expect(err).ToNot(HaveOccurred())

			cachedPodDisruptionBudget = components.NewPodDisruptionBudgetForDeployment(deployment)
		})

		AfterEach(func() {
			ctrl.Finish()
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

		It("should not roll out subresources on existing CRDs before controll-plane rollover", func() {
			crd := &extv1beta1.CustomResourceDefinition{
				ObjectMeta: v12.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: extv1beta1.CustomResourceDefinitionSpec{
					Subresources: &extv1beta1.CustomResourceSubresources{
						Scale: &extv1beta1.CustomResourceSubresourceScale{
							SpecReplicasPath: "blub",
						},
						Status: &extv1beta1.CustomResourceSubresourceStatus{},
					},
				},
			}
			targetStrategy := &InstallStrategy{
				crds: []*extv1beta1.CustomResourceDefinition{
					crd,
				},
			}

			crdWithoutSubresource := crd.DeepCopy()
			crdWithoutSubresource.Spec.Subresources = nil

			stores.CrdCache.Add(crdWithoutSubresource)
			extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)
				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())
				obj, err := json.Marshal(crdWithoutSubresource)
				Expect(err).To(BeNil())
				obj, err = patch.Apply(obj)
				Expect(err).To(BeNil())
				crd := &extv1beta1.CustomResourceDefinition{}
				Expect(json.Unmarshal(obj, crd)).To(Succeed())
				Expect(crd.Spec.Subresources.Status).To(BeNil())
				Expect(crd.Spec.Subresources.Scale).ToNot(BeNil())
				return true, crd, nil
			})

			Expect(createOrUpdateCrds(kv, targetStrategy, stores, clientset, expectations)).To(Succeed())
		})

		It("should not roll out subresources on existing CRDs after the controll-plane rollover", func() {
			crd := &extv1beta1.CustomResourceDefinition{
				ObjectMeta: v12.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: extv1beta1.CustomResourceDefinitionSpec{
					Subresources: &extv1beta1.CustomResourceSubresources{
						Scale: &extv1beta1.CustomResourceSubresourceScale{
							SpecReplicasPath: "blub",
						},
						Status: &extv1beta1.CustomResourceSubresourceStatus{},
					},
				},
			}
			targetStrategy := &InstallStrategy{
				crds: []*extv1beta1.CustomResourceDefinition{
					crd,
				},
			}

			crdWithoutSubresource := crd.DeepCopy()
			crdWithoutSubresource.Spec.Subresources = nil

			stores.CrdCache.Add(crdWithoutSubresource)
			extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)
				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())
				obj, err := json.Marshal(crdWithoutSubresource)
				Expect(err).To(BeNil())
				obj, err = patch.Apply(obj)
				Expect(err).To(BeNil())
				crd := &extv1beta1.CustomResourceDefinition{}
				Expect(json.Unmarshal(obj, crd)).To(Succeed())
				Expect(crd.Spec.Subresources.Status).ToNot(BeNil())
				Expect(crd.Spec.Subresources.Scale).ToNot(BeNil())
				return true, crd, nil
			})

			Expect(rolloutNonCompatibleCRDChanges(kv, targetStrategy, stores, clientset, expectations)).To(Succeed())
		})
	})

})
