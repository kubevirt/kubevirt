package apply

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	_ "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply PDBs", func() {

	var ctrl *gomock.Controller
	var pdbClient *fake.Clientset
	var stores util.Stores
	var clientset *kubecli.MockKubevirtClient
	var expectations *util.Expectations
	var kv *v1.KubeVirt
	var deployment *v12.Deployment
	var mockPodDisruptionBudgetCacheStore *MockStore
	var requiredPDB *v1beta1.PodDisruptionBudget
	var r *Reconciler
	var mockGeneration int64
	var err error

	getCachedPDB := func() *v1beta1.PodDisruptionBudget {
		Expect(requiredPDB).ToNot(BeNil())

		cachedPDB := requiredPDB.DeepCopy()
		injectOperatorMetadata(kv, &cachedPDB.ObjectMeta, Version, Registry, Id, true)
		err := stores.PodDisruptionBudgetCache.Add(cachedPDB)
		Expect(err).To(BeNil())

		return cachedPDB
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
		pdbClient = fake.NewSimpleClientset()

		stores = util.Stores{}
		stores.PodDisruptionBudgetCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{}

		clientset = kubecli.NewMockKubevirtClient(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		clientset.EXPECT().PolicyV1beta1().Return(pdbClient.PolicyV1beta1()).AnyTimes()
		kv = &v1.KubeVirt{}

		r = &Reconciler{
			kv:             kv,
			targetStrategy: nil,
			stores:         stores,
			clientset:      clientset,
			expectations:   expectations,
		}

		deployment, err = components.NewApiServerDeployment(Namespace, Registry, "", Version, "", "", corev1.PullIfNotPresent, "verbosity", map[string]string{})
		Expect(err).ToNot(HaveOccurred())

		kv.Status.TargetKubeVirtRegistry = Registry
		kv.Status.TargetKubeVirtVersion = Version
		kv.Status.TargetDeploymentID = Id

		mockPodDisruptionBudgetCacheStore = &MockStore{}
		err = stores.PodDisruptionBudgetCache.Add(mockPodDisruptionBudgetCacheStore)

		mockGeneration = 123

		// Set required PDB
		requiredPDB = components.NewPodDisruptionBudgetForDeployment(deployment)
		Expect(requiredPDB).ToNot(BeNil())
		requiredPDB.Annotations = make(map[string]string)
		requiredPDB.SetGeneration(mockGeneration)
		SetGeneration(&kv.Status.Generations, requiredPDB)

	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Reconciliation", func() {
		It("should not patch PDB on sync when they are equal", func() {
			cachedPDB := getCachedPDB()
			Expect(cachedPDB).ToNot(BeNil())

			pdbClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				// Fail if patch occurred
				Expect(true).To(BeFalse())
				return true, nil, nil
			})

			Expect(r.syncPodDisruptionBudgetForDeployment(deployment)).To(BeNil())
		})

		It("should patch PDB on sync when it is not equal to the required PDB", func() {
			const versionAnnotation = v1.InstallStrategyVersionAnnotation
			originalVersion := Version
			modifiedVersion := Version + "_fake"
			patchedOccurred := false

			// Add modified PDB to cache
			cachedPDB := getCachedPDB()
			cachedPDB.ObjectMeta.Annotations[versionAnnotation] = modifiedVersion

			pdbClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				// Ensure that the PDB in cache is being patched to required state
				Expect(cachedPDB.ObjectMeta.Annotations[versionAnnotation]).To(Equal(modifiedVersion))
				a := action.(testing.PatchActionImpl)

				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				obj, err := json.Marshal(cachedPDB)
				Expect(err).To(BeNil())

				obj, err = patch.Apply(obj)
				Expect(err).To(BeNil())

				pdb := &v1beta1.PodDisruptionBudget{}
				Expect(json.Unmarshal(obj, pdb)).To(Succeed())
				Expect(pdb.ObjectMeta.Annotations[versionAnnotation]).To(Equal(originalVersion))

				patchedOccurred = true
				return true, pdb, nil
			})

			Expect(r.syncPodDisruptionBudgetForDeployment(deployment)).To(Succeed())

			// Fail if patch did not occur
			Expect(patchedOccurred).To(BeTrue())
		})
	})

})
