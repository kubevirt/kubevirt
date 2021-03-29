package apply

import (
	"bufio"
	"bytes"
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	marshalutil "kubevirt.io/kubevirt/tools/util"
)

var _ = Describe("Apply CRDs", func() {
	var clientset *kubecli.MockKubevirtClient
	var ctrl *gomock.Controller
	var extClient *extclientfake.Clientset
	var expectations *util.Expectations
	var kv *v1.KubeVirt
	var stores util.Stores

	config := getConfig("fake-registry", "v9.9.9")

	loadTargetStrategy := func(crd *extv1beta1.CustomResourceDefinition) *install.Strategy {
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)

		marshalutil.MarshallObject(crd, writer)
		writer.Flush()

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "kubevirt-install-strategy-",
				Namespace:    config.GetNamespace(),
				Labels: map[string]string{
					v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
					v1.InstallStrategyLabel: "",
				},
				Annotations: map[string]string{
					v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
					v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
					v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
				},
			},
			Data: map[string]string{
				"manifests": string(b.Bytes()),
			},
		}

		stores.InstallStrategyConfigMapCache.Add(configMap)
		targetStrategy, err := installstrategy.LoadInstallStrategyFromCache(stores, config)
		Expect(err).ToNot(HaveOccurred())

		return targetStrategy
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

		extClient = extclientfake.NewSimpleClientset()

		extClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		stores = util.Stores{}

		stores.CrdCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{}

		clientset = kubecli.NewMockKubevirtClient(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		clientset.EXPECT().ExtensionsClient().Return(extClient).AnyTimes()
		kv = &v1.KubeVirt{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should not roll out subresources on existing CRDs before control-plane rollover", func() {
		crd := &extv1beta1.CustomResourceDefinition{
			TypeMeta: v12.TypeMeta{
				APIVersion: extv1beta1.GroupName,
				Kind:       "CustomResourceDefinition",
			},
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
		targetStrategy := loadTargetStrategy(crd)

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

		r := &Reconciler{
			kv:             kv,
			targetStrategy: targetStrategy,
			stores:         stores,
			clientset:      clientset,
			expectations:   expectations,
		}

		Expect(r.createOrUpdateCrds()).To(Succeed())
	})

	It("should not roll out subresources on existing CRDs after the control-plane rollover", func() {
		crd := &extv1beta1.CustomResourceDefinition{
			TypeMeta: v12.TypeMeta{
				APIVersion: extv1beta1.GroupName,
				Kind:       "CustomResourceDefinition",
			},
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
		targetStrategy := loadTargetStrategy(crd)

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

		r := &Reconciler{
			kv:             kv,
			targetStrategy: targetStrategy,
			stores:         stores,
			clientset:      clientset,
			expectations:   expectations,
		}

		Expect(r.rolloutNonCompatibleCRDChanges()).To(Succeed())
	})
})
