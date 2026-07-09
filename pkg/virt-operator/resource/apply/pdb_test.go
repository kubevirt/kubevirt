package apply

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	v12 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	_ "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply PDBs", func() {

	var ctrl *gomock.Controller
	var k8sClient *fake.Clientset
	var stores util.Stores
	var virtClient *kubecli.MockKubevirtClient
	var expectations *util.Expectations
	var kv *v1.KubeVirt
	var deployment *v12.Deployment
	var requiredPDB *policyv1.PodDisruptionBudget
	var r *Reconciler
	var mockGeneration int64

	getCachedPDB := func() *policyv1.PodDisruptionBudget {
		Expect(requiredPDB).ToNot(BeNil())

		cachedPDB := requiredPDB.DeepCopy()
		injectOperatorMetadata(kv, &cachedPDB.ObjectMeta, Version, Registry, Id, true)
		err := stores.PodDisruptionBudgetCache.Add(cachedPDB)
		Expect(err).ToNot(HaveOccurred())

		return cachedPDB
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
		k8sClient = fake.NewSimpleClientset()

		stores = util.Stores{}
		stores.PodDisruptionBudgetCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		expectations = &util.Expectations{
			PodDisruptionBudget: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudgets")),
		}

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		kv = &v1.KubeVirt{}

		r = &Reconciler{
			kv:             kv,
			kvKey:          Namespace + "/test",
			targetStrategy: nil,
			stores:         stores,
			virtClient:     virtClient,
			k8sClient:      k8sClient,
			expectations:   expectations,
		}

		virtApiConfig := &util.KubeVirtDeploymentConfig{
			Registry:        Registry,
			KubeVirtVersion: Version,
			Namespace:       Namespace,
		}
		deployment = components.NewApiServerDeployment(virtApiConfig, "", "", "")

		kv.Status.TargetKubeVirtRegistry = Registry
		kv.Status.TargetKubeVirtVersion = Version
		kv.Status.TargetDeploymentID = Id

		mockGeneration = 123

		// Set required PDB
		requiredPDB = components.NewPodDisruptionBudgetForDeployment(deployment)
		Expect(requiredPDB).ToNot(BeNil())
		requiredPDB.Annotations = make(map[string]string)
		requiredPDB.SetGeneration(mockGeneration)
		SetGeneration(&kv.Status.Generations, requiredPDB)

	})

	Context("Reconciliation", func() {
		It("should not patch PDB on sync when they are equal", func() {
			cachedPDB := getCachedPDB()
			Expect(cachedPDB).ToNot(BeNil())

			k8sClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				// Fail if patch occurred
				Expect(true).To(BeFalse())
				return true, nil, nil
			})

			Expect(r.syncPodDisruptionBudgetForDeployment(deployment)).To(Succeed())
		})

		It("should patch PDB on sync when it is not equal to the required PDB", func() {
			const versionAnnotation = v1.InstallStrategyVersionAnnotation
			originalVersion := Version
			modifiedVersion := Version + "_fake"
			patchedOccurred := false

			// Add modified PDB to cache
			cachedPDB := getCachedPDB()
			cachedPDB.ObjectMeta.Annotations[versionAnnotation] = modifiedVersion

			k8sClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				// Ensure that the PDB in cache is being patched to required state
				Expect(cachedPDB.ObjectMeta.Annotations[versionAnnotation]).To(Equal(modifiedVersion))
				a := action.(testing.PatchActionImpl)

				patch, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				obj, err := json.Marshal(cachedPDB)
				Expect(err).ToNot(HaveOccurred())

				obj, err = patch.Apply(obj)
				Expect(err).ToNot(HaveOccurred())

				pdb := &policyv1.PodDisruptionBudget{}
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

	Context("export-proxy PDB", func() {
		It("should keep minAvailable at 1 when more than one replica is running", func() {
			createFakeNodes(k8sClient, 2, 0)
			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")
			scaledReplicas := int32(5)
			exportProxy.Spec.Replicas = &scaledReplicas

			pdb := components.NewExportProxyPodDisruptionBudget(exportProxy)
			Expect(pdb.Spec.MinAvailable.IntValue()).To(Equal(1))
			Expect(pdb.Name).To(Equal("virt-exportproxy-pdb"))

			k8sClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(testing.CreateActionImpl)
				createdPDB := createAction.Object.(*policyv1.PodDisruptionBudget)
				Expect(createdPDB.Spec.MinAvailable.IntValue()).To(Equal(1))
				return true, createdPDB, nil
			})

			Expect(r.syncExportProxyPodDisruptionBudget(exportProxy, nil)).To(Succeed())
		})

		It("should delete the PDB when operator desired replica count does not allow it", func() {
			createFakeNodes(k8sClient, 1, 0)
			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")

			cachedPDB := components.NewExportProxyPodDisruptionBudget(exportProxy)
			injectOperatorMetadata(kv, &cachedPDB.ObjectMeta, Version, Registry, Id, true)
			Expect(stores.PodDisruptionBudgetCache.Add(cachedPDB)).To(Succeed())

			deleted := false
			k8sClient.Fake.PrependReactor("delete", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				Expect(action.(testing.DeleteActionImpl).Name).To(Equal("virt-exportproxy-pdb"))
				deleted = true
				return true, nil, nil
			})

			Expect(r.syncExportProxyPodDisruptionBudget(exportProxy, nil)).To(Succeed())
			Expect(deleted).To(BeTrue())
		})

		It("should patch stale minAvailable down to 1", func() {
			createFakeNodes(k8sClient, 2, 0)
			exportProxyConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
				Namespace:       Namespace,
			}
			exportProxy := components.NewExportProxyDeployment(exportProxyConfig, "", "", "")
			scaledReplicas := int32(5)
			exportProxy.Spec.Replicas = &scaledReplicas

			staleMinAvailable := intstr.FromInt(4)
			cachedPDB := components.NewExportProxyPodDisruptionBudget(exportProxy)
			cachedPDB.Spec.MinAvailable = &staleMinAvailable
			cachedPDB.Annotations = make(map[string]string)
			cachedPDB.SetGeneration(mockGeneration)
			SetGeneration(&kv.Status.Generations, cachedPDB)
			injectOperatorMetadata(kv, &cachedPDB.ObjectMeta, Version, Registry, Id, true)
			Expect(stores.PodDisruptionBudgetCache.Add(cachedPDB)).To(Succeed())

			patched := false
			k8sClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				a := action.(testing.PatchActionImpl)

				patchOps, err := jsonpatch.DecodePatch(a.Patch)
				Expect(err).ToNot(HaveOccurred())

				obj, err := json.Marshal(cachedPDB)
				Expect(err).ToNot(HaveOccurred())

				obj, err = patchOps.Apply(obj)
				Expect(err).ToNot(HaveOccurred())

				pdb := &policyv1.PodDisruptionBudget{}
				Expect(json.Unmarshal(obj, pdb)).To(Succeed())
				Expect(pdb.Spec.MinAvailable.IntValue()).To(Equal(1))

				patched = true
				return true, pdb, nil
			})

			Expect(r.syncExportProxyPodDisruptionBudget(exportProxy, nil)).To(Succeed())
			Expect(patched).To(BeTrue())
		})
	})
})
