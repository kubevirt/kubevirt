package installstrategy

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			deployment, err = components.NewApiServerDeployment(Namespace, Registry, "", Version, "", "", corev1.PullIfNotPresent, "verbosity", map[string]string{})
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
			injectOperatorMetadata(kv, &cachedPodDisruptionBudget.ObjectMeta, Version, Registry, Id, true)

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

	Context("Services", func() {

		It("should patch if ClusterIp == \"\" during update", func() {

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "somenamespace",
				},
				Spec: v1.KubeVirtSpec{
					ImageRegistry: "someregistery",
					ImageTag:      "v1",
				},
			}

			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeClusterIP
			cachedService.Spec.ClusterIP = "10.10.10.10"

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeClusterIP
			service.Spec.ClusterIP = ""

			ops, deleteAndReplace, err := generateServicePatch(kv, cachedService, service)
			Expect(err).To(BeNil())
			Expect(deleteAndReplace).To(BeFalse())
			Expect(ops).ToNot(Equal(""))
		})

		It("should replace if ClusterIp != \"\" during update and ip changes", func() {

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "somenamespace",
				},
				Spec: v1.KubeVirtSpec{
					ImageRegistry: "someregistery",
					ImageTag:      "v1",
				},
			}

			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeClusterIP
			cachedService.Spec.ClusterIP = "10.10.10.10"

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeClusterIP
			service.Spec.ClusterIP = "10.10.10.11"

			_, deleteAndReplace, err := generateServicePatch(kv, cachedService, service)
			Expect(err).To(BeNil())
			Expect(deleteAndReplace).To(BeTrue())
		})

		It("should replace if not a ClusterIP service", func() {

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "somenamespace",
				},
				Spec: v1.KubeVirtSpec{
					ImageRegistry: "someregistery",
					ImageTag:      "v1",
				},
			}

			cachedService := &corev1.Service{}
			cachedService.Spec.Type = corev1.ServiceTypeNodePort

			service := &corev1.Service{}
			service.Spec.Type = corev1.ServiceTypeNodePort

			_, deleteAndReplace, err := generateServicePatch(kv, cachedService, service)
			Expect(err).To(BeNil())
			Expect(deleteAndReplace).To(BeTrue())
		})
	})

	Context("Product Names and Versions", func() {
		table.DescribeTable("label validation", func(testVector string, expectedResult bool) {
			Expect(isValidLabel(testVector)).To(Equal(expectedResult))
		},
			table.Entry("should allow 1 character strings", "a", true),
			table.Entry("should allow 2 character strings", "aa", true),
			table.Entry("should allow 3 character strings", "aaa", true),
			table.Entry("should allow 63 character strings", strings.Repeat("a", 63), true),
			table.Entry("should reject 64 character strings", strings.Repeat("a", 64), false),
			table.Entry("should reject strings that begin with .", ".a", false),
			table.Entry("should reject strings that end with .", "a.", false),
			table.Entry("should reject strings that contain junk characters", `a\a`, false),
			table.Entry("should allow strings that contain dots", "a.a", true),
			table.Entry("should allow strings that contain dashes", "a-a", true),
			table.Entry("should allow strings that contain underscores", "a_a", true),
			table.Entry("should allow empty strings", "", true),
		)
	})

	Context("Injecting Metadata", func() {

		It("should set expected values", func() {

			kv := &v1.KubeVirt{}
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version
			kv.Status.TargetDeploymentID = Id

			deployment := appsv1.Deployment{}
			injectOperatorMetadata(kv, &deployment.ObjectMeta, "fakeversion", "fakeregistry", "fakeid", false)

			// NOTE we are purposfully not using the defined constant values
			// in types.go here. This test is explicitly verifying that those
			// values in types.go that we depend on for virt-operator updates
			// do not change. This is meant to preserve backwards and forwards
			// compatibility

			managedBy, ok := deployment.Labels["app.kubernetes.io/managed-by"]

			Expect(ok).To(BeTrue())
			Expect(managedBy).To(Equal("kubevirt-operator"))

			version, ok := deployment.Annotations["kubevirt.io/install-strategy-version"]
			Expect(ok).To(BeTrue())
			Expect(version).To(Equal("fakeversion"))

			registry, ok := deployment.Annotations["kubevirt.io/install-strategy-registry"]
			Expect(ok).To(BeTrue())
			Expect(registry).To(Equal("fakeregistry"))

			id, ok := deployment.Annotations["kubevirt.io/install-strategy-identifier"]
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal("fakeid"))
		})
	})

	Context("on calling injectPlacementMetadata", func() {
		var componentConfig *v1.ComponentConfig
		var nodePlacement *v1.NodePlacement
		var podSpec *corev1.PodSpec
		var toleration corev1.Toleration
		var toleration2 corev1.Toleration
		var affinity *corev1.Affinity
		var affinity2 *corev1.Affinity

		BeforeEach(func() {
			componentConfig = &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{},
			}
			nodePlacement = componentConfig.NodePlacement
			podSpec = &corev1.PodSpec{}

			toleration = corev1.Toleration{
				Key:      "test-taint",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}
			toleration2 = corev1.Toleration{
				Key:      "test-taint2",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}

			affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									corev1.NodeSelectorRequirement{
										Key:      "required",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						corev1.PreferredSchedulingTerm{
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									corev1.NodeSelectorRequirement{
										Key:      "preferred",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
				},
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"required": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						corev1.WeightedPodAffinityTerm{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"preferred": "term"},
								},
							},
						},
					},
				},
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"anti-required": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						corev1.WeightedPodAffinityTerm{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"anti-preferred": "term"},
								},
							},
						},
					},
				},
			}

			affinity2 = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									corev1.NodeSelectorRequirement{
										Key:      "required2",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						corev1.PreferredSchedulingTerm{
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									corev1.NodeSelectorRequirement{
										Key:      "preferred2",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
				},
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"required2": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						corev1.WeightedPodAffinityTerm{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"preferred2": "term"},
								},
							},
						},
					},
				},
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"anti-required2": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						corev1.WeightedPodAffinityTerm{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"anti-preferred2": "term"},
								},
							},
						},
					},
				},
			}

		})

		// Node Selectors
		It("should succeed if nodePlacement is nil", func() {
			// if componentConfig is nil
			injectPlacementMetadata(nil, podSpec)
			// if componentConfig.NodePlacement is nil
			componentConfig.NodePlacement = nil
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(0))
		})

		It("should succeed if podSpec is nil", func() {
			orig := componentConfig.DeepCopy()
			injectPlacementMetadata(componentConfig, nil)
			Expect(reflect.DeepEqual(orig, componentConfig)).To(BeTrue())
		})

		It("should copy NodeSelectors when podSpec is empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
		})

		It("should merge NodeSelectors when podSpec is not empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["existing"] = "value"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector["existing"]).To(Equal("value"))
		})

		It("should favor podSpec if NodeSelectors collide", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
		})

		It("should preserve NodeSelectors if nodePlacement has none", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
		})

		// tolerations
		It("should copy tolerations when podSpec is empty", func() {
			toleration := corev1.Toleration{
				Key:      "test-taint",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should preserve tolerations when nodePlacement is empty", func() {
			podSpec.Tolerations = []corev1.Toleration{toleration}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should merge tolerations when both are defined", func() {
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			podSpec.Tolerations = []corev1.Toleration{toleration2}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(2))
		})

		It("It should copy NodePlacement if podSpec Affinity is empty", func() {
			nodePlacement.Affinity = affinity
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should copy NodePlacement if Node, Pod and Anti affinities are empty", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = &corev1.Affinity{}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should merge NodePlacement and podSpec affinity terms", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = affinity2
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(2))
			Expect(len(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeFalse())
		})

		It("It should copy Required NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(1))
		})

		It("It should copy Preferred NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Required PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Preferred PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Required PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Preferred PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})
	})
})
