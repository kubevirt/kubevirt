package apply

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	jsonpatch "github.com/evanphx/json-patch"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply CRDs", func() {
	var clientset *kubecli.MockKubevirtClient
	var ctrl *gomock.Controller
	var extClient *extclientfake.Clientset
	var expectations *util.Expectations
	var kv *v1.KubeVirt
	var stores util.Stores

	config := getConfig("fake-registry", "v9.9.9")

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

		extClient = extclientfake.NewSimpleClientset()

		extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		stores = util.Stores{}
		stores.OperatorCrdCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{}

		clientset = kubecli.NewMockKubevirtClient(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		clientset.EXPECT().ExtensionsClient().Return(extClient).AnyTimes()
		kv = &v1.KubeVirt{}
	})

	It("should not roll out subresources on existing CRDs before control-plane rollover", func() {
		crd := &extv1.CustomResourceDefinition{
			TypeMeta: v12.TypeMeta{
				APIVersion: extv1.SchemeGroupVersion.String(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: v12.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: extv1.CustomResourceDefinitionSpec{
				Versions: []extv1.CustomResourceDefinitionVersion{
					{
						Subresources: &extv1.CustomResourceSubresources{
							Scale: &extv1.CustomResourceSubresourceScale{
								SpecReplicasPath: "blub",
							},
							Status: &extv1.CustomResourceSubresourceStatus{},
						},
					},
				},
			},
		}
		targetStrategy := loadTargetStrategy(crd, config, stores)

		crdWithoutSubresource := crd.DeepCopy()
		crdWithoutSubresource.Spec.Versions[0].Subresources = nil

		stores.OperatorCrdCache.Add(crdWithoutSubresource)
		extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(testing.PatchActionImpl)
			patch, err := jsonpatch.DecodePatch(a.Patch)
			Expect(err).ToNot(HaveOccurred())
			obj, err := json.Marshal(crdWithoutSubresource)
			Expect(err).ToNot(HaveOccurred())
			obj, err = patch.Apply(obj)
			Expect(err).ToNot(HaveOccurred())
			crd := &extv1.CustomResourceDefinition{}
			Expect(json.Unmarshal(obj, crd)).To(Succeed())
			Expect(crd.Spec.Versions[0].Subresources.Status).To(BeNil())
			Expect(crd.Spec.Versions[0].Subresources.Scale).ToNot(BeNil())
			return true, crd, nil
		})

		r := &Reconciler{
			kv:             kv,
			targetStrategy: targetStrategy,
			stores:         stores,
			virtClientset:  clientset,
			expectations:   expectations,
		}

		Expect(r.createOrUpdateCrds()).To(Succeed())
	})

	It("should not roll out subresources on existing CRDs after the control-plane rollover", func() {
		crd := &extv1.CustomResourceDefinition{
			TypeMeta: v12.TypeMeta{
				APIVersion: extv1.SchemeGroupVersion.String(),
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: v12.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: extv1.CustomResourceDefinitionSpec{
				Versions: []extv1.CustomResourceDefinitionVersion{
					{
						Subresources: &extv1.CustomResourceSubresources{
							Scale: &extv1.CustomResourceSubresourceScale{
								SpecReplicasPath: "blub",
							},
							Status: &extv1.CustomResourceSubresourceStatus{},
						},
					},
				},
			},
		}
		targetStrategy := loadTargetStrategy(crd, config, stores)

		crdWithoutSubresource := crd.DeepCopy()
		crdWithoutSubresource.Spec.Versions[0].Subresources = nil

		stores.OperatorCrdCache.Add(crdWithoutSubresource)
		extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			a := action.(testing.PatchActionImpl)
			patch, err := jsonpatch.DecodePatch(a.Patch)
			Expect(err).ToNot(HaveOccurred())
			obj, err := json.Marshal(crdWithoutSubresource)
			Expect(err).ToNot(HaveOccurred())
			obj, err = patch.Apply(obj)
			Expect(err).ToNot(HaveOccurred())
			crd := &extv1.CustomResourceDefinition{}
			Expect(json.Unmarshal(obj, crd)).To(Succeed())
			Expect(crd.Spec.Versions[0].Subresources.Status).ToNot(BeNil())
			Expect(crd.Spec.Versions[0].Subresources.Scale).ToNot(BeNil())
			return true, crd, nil
		})

		r := &Reconciler{
			kv:             kv,
			targetStrategy: targetStrategy,
			stores:         stores,
			virtClientset:  clientset,
			expectations:   expectations,
		}

		Expect(r.rolloutNonCompatibleCRDChanges()).To(Succeed())
	})
})
