package apply

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
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
			virtClient:     clientset,
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
			virtClient:     clientset,
			expectations:   expectations,
		}

		Expect(r.rolloutNonCompatibleCRDChanges()).To(Succeed())
	})
	It("preserves the stored VMExport alpha definition through both rollout phases", func() {
		target, err := components.NewVirtualMachineExportCrd()
		Expect(err).ToNot(HaveOccurred())
		old := target.DeepCopy()
		beta := old.Spec.Versions[0].DeepCopy()
		beta.Storage = true
		alpha := beta.DeepCopy()
		alpha.Name = "v1alpha1"
		alpha.Storage = false
		old.Spec.Versions = []extv1.CustomResourceDefinitionVersion{*alpha, *beta}
		old.Status.StoredVersions = []string{"v1alpha1", "v1beta1"}
		originalSchema := alpha.Schema.DeepCopy()
		targetStrategy := loadTargetStrategy(target, config, stores)
		Expect(stores.OperatorCrdCache.Add(old)).To(Succeed())
		patches := 0
		extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (bool, runtime.Object, error) {
			a := action.(testing.PatchAction)
			patch, err := jsonpatch.DecodePatch(a.GetPatch())
			Expect(err).ToNot(HaveOccurred())
			data, err := json.Marshal(old)
			Expect(err).ToNot(HaveOccurred())
			data, err = patch.Apply(data)
			Expect(err).ToNot(HaveOccurred())
			updated := &extv1.CustomResourceDefinition{}
			Expect(json.Unmarshal(data, updated)).To(Succeed())
			names := []string{}
			for _, version := range updated.Spec.Versions {
				names = append(names, version.Name)
				Expect(version.Storage).To(Equal(version.Name == "v1"))
				if version.Name == "v1alpha1" {
					Expect(version.Served).To(BeFalse())
					Expect(version.Schema).To(Equal(originalSchema))
				} else {
					Expect(version.Served).To(BeTrue())
				}
			}
			for _, stored := range old.Status.StoredVersions {
				Expect(names).To(ContainElement(stored))
			}
			Expect(names).To(ConsistOf("v1alpha1", "v1beta1", "v1"))
			Expect(updated.Status).To(Equal(old.Status))
			patches++
			Expect(updated.Spec.Versions[1].Subresources.Status != nil).To(Equal(patches > 1))
			old = updated
			Expect(stores.OperatorCrdCache.Update(old)).To(Succeed())
			return true, updated, nil
		})
		r := &Reconciler{kv: kv, targetStrategy: targetStrategy, stores: stores, virtClient: clientset, expectations: expectations}
		Expect(r.createOrUpdateCrds()).To(Succeed())
		Expect(r.rolloutNonCompatibleCRDChanges()).To(Succeed())
		Expect(r.createOrUpdateCrds()).To(Succeed())
		Expect(r.rolloutNonCompatibleCRDChanges()).To(Succeed())
		Expect(patches).To(Equal(2))
		Expect(target.Spec.Versions).To(HaveLen(2))
	})

	DescribeTable("handles VMExport storage history",
		func(name string, stored []string, includeAlpha, expectAlpha, expectError bool) {
			target, err := components.NewVirtualMachineExportCrd()
			Expect(err).ToNot(HaveOccurred())
			target.Name = name
			cached := target.DeepCopy()
			cached.Status.StoredVersions = stored
			if includeAlpha {
				alpha := cached.Spec.Versions[0].DeepCopy()
				alpha.Name = "v1alpha1"
				cached.Spec.Versions = append(cached.Spec.Versions, *alpha)
			}
			original := cached.DeepCopy()
			calls := 0
			extClient.Fake.PrependReactor("patch", "customresourcedefinitions", func(action testing.Action) (bool, runtime.Object, error) {
				calls++
				patch, err := jsonpatch.DecodePatch(action.(testing.PatchAction).GetPatch())
				Expect(err).ToNot(HaveOccurred())
				data, err := json.Marshal(cached)
				Expect(err).ToNot(HaveOccurred())
				data, err = patch.Apply(data)
				Expect(err).ToNot(HaveOccurred())
				updated := &extv1.CustomResourceDefinition{}
				Expect(json.Unmarshal(data, updated)).To(Succeed())
				alphaCount := 0
				for _, version := range updated.Spec.Versions {
					if version.Name == "v1alpha1" {
						alphaCount++
					}
				}
				if expectAlpha {
					Expect(alphaCount).To(Equal(1))
				} else {
					Expect(alphaCount).To(BeZero())
				}
				return true, updated, nil
			})
			_, err = patchCRD(extClient, target, cached, nil)
			if expectError {
				Expect(err).To(MatchError(ContainSubstring("restore the version definition before retrying")))
				Expect(calls).To(BeZero())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(calls).To(Equal(1))
			}
			Expect(cached).To(Equal(original))
			Expect(target.Spec.Versions).To(HaveLen(2))
		},
		Entry("beta-only history", components.VIRTUALMACHINEEXPORT, []string{"v1beta1"}, true, false, false),
		Entry("unrelated CRD", "unrelated.example.com", []string{"v1alpha1"}, true, false, false),
		Entry("missing alpha definition", components.VIRTUALMACHINEEXPORT, []string{"v1alpha1"}, false, false, true),
	)

})
