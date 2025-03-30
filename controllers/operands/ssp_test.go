package operands

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const customNS = "custom-ns"

var _ = Describe("SSP Operands", func() {

	var (
		testFilesLocation = getTestFilesLocation() + "/dataImportCronTemplates"
	)
	Context("SSP", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &sspv1beta2.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).To(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			const cTNamespace = "nonDefault"
			hco.Spec.CommonTemplatesNamespace = ptr.To(cTNamespace)
			expectedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())
			existingResource := expectedResource.DeepCopy()

			existingResource.Spec.TemplateValidator.Replicas = ptr.To(defaultTemplateValidatorReplicas * 2) // non-default value

			req.HCOTriggered = false // mock a reconciliation triggered by a change in NewKubeVirtCommonTemplateBundle CR

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &sspv1beta2.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Spec).To(Equal(expectedResource.Spec))
			Expect(foundResource.Spec.CommonTemplates.Namespace).To(Equal(cTNamespace), "common-templates namespace should equal")

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &sspv1beta2.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default on label deletion without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &sspv1beta2.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should create ssp with deployVmConsoleProxy feature gate enabled", func() {
			hco := commontestutils.NewHco()
			hco.Spec.DeployVMConsoleProxy = ptr.To(true)

			expectedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())

			Expect(expectedResource.Spec.TokenGenerationService).ToNot(BeNil())
			Expect(expectedResource.Spec.TokenGenerationService.Enabled).To(BeTrue())
		})

		It("should create with deployCommonInstancetypes feature gate disabled", func() {
			hco := commontestutils.NewHco()

			expectedResource, _, err := NewSSP(hco)
			Expect(err).ToNot(HaveOccurred())

			Expect(expectedResource.Spec.FeatureGates.DeployCommonInstancetypes).To(HaveValue(BeFalse()))
		})

		Context("Node placement", func() {

			It("should add node placement if missing", func() {
				existingResource, _, err := NewSSP(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.TemplateValidator.Placement).To(BeNil())
				// TODO: replace BeEquivalentTo with BeEqual once SSP will consume kubevirt.io/controller-lifecycle-operator-sdk/api v0.2.4
				Expect(*foundResource.Spec.TemplateValidator.Placement).To(BeEquivalentTo(*hco.Spec.Infra.NodePlacement))
				Expect(req.Conditions).To(BeEmpty())
			})

			It("should remove node placement if missing in HCO CR", func() {

				hcoNodePlacement := commontestutils.NewHco()
				hcoNodePlacement.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hcoNodePlacement.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()
				existingResource, _, err := NewSSP(hcoNodePlacement, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.TemplateValidator.Placement).ToNot(BeNil())
				Expect(foundResource.Spec.TemplateValidator.Placement).To(BeNil())
				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify node placement according to HCO CR", func() {

				hco.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()
				existingResource, _, err := NewSSP(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())

				// now, modify HCO's node placement
				hco.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key12", Operator: "operator12", Value: "value12", Effect: "effect12", TolerationSeconds: ptr.To[int64](12),
				})
				hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

				hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: ptr.To[int64](34),
				})
				hco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "something entirely else"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.TemplateValidator.Placement.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(existingResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.TemplateValidator.Placement.NodeSelector).To(HaveKeyWithValue("key3", "value3"))

				Expect(foundResource.Spec.TemplateValidator.Placement.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(foundResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(3))
				Expect(foundResource.Spec.TemplateValidator.Placement.NodeSelector).To(HaveKeyWithValue("key3", "something entirely else"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite node placement if directly set on SSP CR", func() {
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewOtherNodePlacement()}
				existingResource, _, err := NewSSP(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in NewKubeVirtNodeLabellerBundle CR
				req.HCOTriggered = false

				// and modify TemplateValidator node placement
				existingResource.Spec.TemplateValidator.Placement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: ptr.To(int64(34)),
				})
				existingResource.Spec.TemplateValidator.Placement.NodeSelector["key3"] = "BADvalue3"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.TemplateValidator.Placement.NodeSelector).To(HaveKeyWithValue("key3", "BADvalue3"))

				Expect(foundResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.TemplateValidator.Placement.NodeSelector).To(HaveKeyWithValue("key3", "value3"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("jsonpath Annotation", func() {
			It("Should create SSP object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "replace",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				ssp, _, err := NewSSP(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(ssp).ToNot(BeNil())
				Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
				Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(5)))
			})

			It("Should fail to create SSP object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				_, _, err := NewSSP(hco)
				Expect(err).To(HaveOccurred())
			})

			It("Ensure func should create SSP object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "replace",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				expectedResource := NewSSPWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				ssp := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						ssp),
				).To(Succeed())

				Expect(ssp).ToNot(BeNil())
				Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
				Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(5)))
			})

			It("Ensure func should fail to create SSP object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				expectedResource := NewSSPWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				ssp := &sspv1beta2.SSP{}

				Expect(cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					ssp,
				)).To(MatchError(errors.IsNotFound, "not found error"))
			})

			It("Ensure func should update SSP object with changes from the annotation", func() {
				existsSsp, _, err := NewSSP(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "replace",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsSsp})

				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())
				Expect(res.UpgradeDone).To(BeFalse())

				ssp := &sspv1beta2.SSP{}

				expectedResource := NewSSPWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						ssp),
				).To(Succeed())

				Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
				Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(5)))
			})

			It("Ensure func should fail to update SSP object with wrong jsonPatch", func() {
				existsSsp, _, err := NewSSP(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchSSPAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/templateValidator/replicas",
						"value": 5
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsSsp})

				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				ssp := &sspv1beta2.SSP{}

				expectedResource := NewSSPWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						ssp),
				).To(Succeed())

				Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
				Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(defaultTemplateValidatorReplicas)))
			})
		})

		Context("Cache", func() {
			cl := commontestutils.InitClient([]client.Object{})
			handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*sspHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*sspHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*sspHooks).cache).To(BeIdenticalTo(cr))

					cdi1, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi1).ToNot(BeNil())
					Expect(cr).To(BeIdenticalTo(cdi1))
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*sspHooks).reset()
				Expect(handler.hooks.(*sspHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*sspHooks).cache).ToNot(BeNil())

				handler.hooks.(*sspHooks).reset()
				Expect(handler.hooks.(*sspHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*sspHooks).cache).ToNot(BeNil())

				Expect(crI).ToNot(BeIdenticalTo(crII))
				Expect(handler.hooks.(*sspHooks).cache).ToNot(BeIdenticalTo(crI))
				Expect(handler.hooks.(*sspHooks).cache).To(BeIdenticalTo(crII))
			})
		})

		Context("Test data import cron template", func() {
			dir := path.Join(os.TempDir(), fmt.Sprint(time.Now().UTC().Unix()))
			origFunc := getDataImportCronTemplatesFileLocation

			image1 := hcov1beta1.DataImportCronTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "image1"},
				Spec: &cdiv1beta1.DataImportCronSpec{
					Schedule: "1 */12 * * *",
					Template: cdiv1beta1.DataVolume{
						Spec: cdiv1beta1.DataVolumeSpec{
							Source: &cdiv1beta1.DataVolumeSource{
								Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To("docker://someregistry/image1")},
							},
						},
					},
					ManagedDataSource: "image1",
				},
			}

			statusImage1 := hcov1beta1.DataImportCronTemplateStatus{
				DataImportCronTemplate: image1,
				Status: hcov1beta1.DataImportCronStatus{
					CommonTemplate: true,
					Modified:       false,
				},
			}

			image2 := hcov1beta1.DataImportCronTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "image2"},
				Spec: &cdiv1beta1.DataImportCronSpec{
					Schedule: "2 */12 * * *",
					Template: cdiv1beta1.DataVolume{
						Spec: cdiv1beta1.DataVolumeSpec{
							Source: &cdiv1beta1.DataVolumeSource{
								Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To("docker://someregistry/image2")},
							},
						},
					},
					ManagedDataSource: "image2",
				},
			}

			statusImage2 := hcov1beta1.DataImportCronTemplateStatus{
				DataImportCronTemplate: image2,
				Status: hcov1beta1.DataImportCronStatus{
					CommonTemplate: true,
					Modified:       false,
				},
			}

			image3 := hcov1beta1.DataImportCronTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "image3"},
				Spec: &cdiv1beta1.DataImportCronSpec{
					Schedule: "3 */12 * * *",
					Template: cdiv1beta1.DataVolume{
						Spec: cdiv1beta1.DataVolumeSpec{
							Source: &cdiv1beta1.DataVolumeSource{
								Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To("docker://someregistry/image3")},
							},
						},
					},
					ManagedDataSource: "image3",
				},
			}

			statusImage3 := hcov1beta1.DataImportCronTemplateStatus{
				DataImportCronTemplate: image3,
				Status: hcov1beta1.DataImportCronStatus{
					CommonTemplate: false,
					Modified:       false,
				},
			}

			image4 := hcov1beta1.DataImportCronTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "image4"},
				Spec: &cdiv1beta1.DataImportCronSpec{
					Schedule: "4 */12 * * *",
					Template: cdiv1beta1.DataVolume{
						Spec: cdiv1beta1.DataVolumeSpec{
							Source: &cdiv1beta1.DataVolumeSource{
								Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To("docker://someregistry/image4")},
							},
						},
					},
					ManagedDataSource: "image4",
				},
			}

			statusImage4 := hcov1beta1.DataImportCronTemplateStatus{
				DataImportCronTemplate: image4,
				Status: hcov1beta1.DataImportCronStatus{
					CommonTemplate: false,
					Modified:       false,
				},
			}

			BeforeEach(func() {
				getDataImportCronTemplatesFileLocation = func() string {
					return dir
				}
			})

			AfterEach(func() {
				getDataImportCronTemplatesFileLocation = origFunc
			})

			It("should read the dataImportCronTemplates file", func() {

				By("directory does not exist - no error")
				Expect(readDataImportCronTemplatesFromFile()).To(Succeed())
				Expect(dataImportCronTemplateHardCodedMap).To(BeEmpty())

				By("file does not exist - no error")
				Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
				defer func() { _ = os.RemoveAll(dir) }()

				Expect(readDataImportCronTemplatesFromFile()).To(Succeed())
				Expect(dataImportCronTemplateHardCodedMap).To(BeEmpty())

				destFile := path.Join(dir, "dataImportCronTemplates.yaml")

				By("valid file exits")
				Expect(commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml"))).To(Succeed())
				defer os.Remove(destFile)
				Expect(readDataImportCronTemplatesFromFile()).To(Succeed())
				Expect(dataImportCronTemplateHardCodedMap).To(HaveLen(2))

				By("the file is wrong")
				Expect(commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "wrongDataImportCronTemplates.yaml"))).To(Succeed())
				defer os.Remove(destFile)
				Expect(readDataImportCronTemplatesFromFile()).To(HaveOccurred())
				Expect(dataImportCronTemplateHardCodedMap).To(BeEmpty())
			})

			Context("test getDataImportCronTemplates", func() {
				origList := dataImportCronTemplateHardCodedMap

				AfterEach(func() {
					dataImportCronTemplateHardCodedMap = origList
				})

				It("should not return the hard coded list dataImportCron FeatureGate is false", func() {
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(false)
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					list, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(list).To(HaveLen(2))
					Expect(list).To(ContainElements(statusImage3, statusImage4))

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{}
					list, err = getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(list).To(BeEmpty())
				})

				It("should return an empty list if both the hard-coded list and the list from HC are empty", func() {
					hcoWithEmptyList := commontestutils.NewHco()
					hcoWithEmptyList.Spec.EnableCommonBootImageImport = ptr.To(true)
					hcoWithEmptyList.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{}
					hcoWithNilList := commontestutils.NewHco()
					hcoWithNilList.Spec.EnableCommonBootImageImport = ptr.To(true)
					hcoWithNilList.Spec.DataImportCronTemplates = nil

					dataImportCronTemplateHardCodedMap = nil
					Expect(getDataImportCronTemplates(hcoWithNilList)).To(BeNil())
					Expect(getDataImportCronTemplates(hcoWithEmptyList)).To(BeNil())
					dataImportCronTemplateHardCodedMap = make(map[string]hcov1beta1.DataImportCronTemplate)
					Expect(getDataImportCronTemplates(hcoWithNilList)).To(BeNil())
					Expect(getDataImportCronTemplates(hcoWithEmptyList)).To(BeNil())
				})

				It("Should add the CR list to the hard-coded list", func() {
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(4))
					Expect(goldenImageList).To(HaveCap(4))
					Expect(goldenImageList).To(ContainElements(statusImage1, statusImage2, statusImage3, statusImage4))
				})

				It("Should not add a common DIC template if it marked as disabled", func() {
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					disabledImage1 := image1.DeepCopy()
					disableDict(disabledImage1)
					enabledImage2 := image2.DeepCopy()
					enableDict(enabledImage2)

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*disabledImage1, *enabledImage2, image3, image4}
					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(3))
					Expect(goldenImageList).To(HaveCap(4))

					statusImage2Enabled := statusImage2.DeepCopy()
					statusImage2Enabled.Status.Modified = true

					Expect(goldenImageList).To(ContainElements(*statusImage2Enabled, statusImage3, statusImage4))
				})

				It("should not add user DIC template if it is disabled", func() {
					dataImportCronTemplateHardCodedMap = nil
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					disabledUserImage := image1.DeepCopy()
					disableDict(disabledUserImage)
					enabledUserImage := image2.DeepCopy()
					enableDict(enabledUserImage)

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*disabledUserImage, *enabledUserImage}
					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(1))

					statusImageEnabled := hcov1beta1.DataImportCronTemplateStatus{
						DataImportCronTemplate: *enabledUserImage,
						Status:                 hcov1beta1.DataImportCronStatus{},
					}

					Expect(goldenImageList).To(ContainElements(statusImageEnabled))
				})

				It("Should reject if the CR list contain DIC templates with the same name, when there are also common DIC templates", func() {
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					image3Modified := image3.DeepCopy()
					image3Modified.Name = image4.Name

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*image3Modified, image4}
					_, err := getDataImportCronTemplates(hco)
					Expect(err).To(HaveOccurred())
				})

				It("Should reject if the CR list contain DIC templates with the same name", func() {
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					image3Modified := image3.DeepCopy()
					image3Modified.Name = image4.Name

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*image3Modified, image4}
					_, err := getDataImportCronTemplates(hco)
					Expect(err).To(HaveOccurred())
				})

				It("Should not add the CR list to the hard-coded list, if it's empty", func() {
					By("CR list is nil")
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.DataImportCronTemplates = nil
					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(2))
					Expect(goldenImageList).To(HaveCap(2))
					Expect(goldenImageList).To(ContainElements(statusImage1, statusImage2))

					By("CR list is empty")
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{}
					goldenImageList, err = getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(2))
					Expect(goldenImageList).To(ContainElements(statusImage1, statusImage2))
				})

				It("Should return only the CR list, if the hard-coded list is empty", func() {
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}

					By("when dataImportCronTemplateHardCodedList is nil")
					dataImportCronTemplateHardCodedMap = nil
					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(2))
					Expect(goldenImageList).To(HaveCap(2))
					Expect(goldenImageList).To(ContainElements(statusImage3, statusImage4))

					By("when dataImportCronTemplateHardCodedList is empty")
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{}
					goldenImageList, err = getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(2))
					Expect(goldenImageList).To(HaveCap(2))
					Expect(goldenImageList).To(ContainElements(statusImage3, statusImage4))
				})

				It("Should replace the common DICT registry field if the CR list includes it", func() {

					const (
						modifiedURL = "docker://someregistry/modified"
						anotherURL  = "docker://someregistry/anotherURL"
					)

					image1FromFile := image1.DeepCopy()
					image1FromFile.Spec.Template.Spec.Source = &cdiv1beta1.DataVolumeSource{
						Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To(modifiedURL)},
					}

					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: *image1FromFile,
						image2.Name: image2,
					}

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					modifiedImage1 := image1.DeepCopy()
					modifiedImage1.Spec.Template.Spec.Source = &cdiv1beta1.DataVolumeSource{
						Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: ptr.To(anotherURL)},
					}

					By("check that if the CR schedule is empty, HCO adds it from the common dict")
					modifiedImage1.Spec.Schedule = ""

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedImage1, image3, image4}

					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(4))
					Expect(goldenImageList).To(HaveCap(4))

					modifiedImage1.Spec.Schedule = image1.Spec.Schedule

					for _, dict := range goldenImageList {
						if dict.Name == "image1" {
							Expect(dict.Spec).To(Equal(modifiedImage1.Spec))
							Expect(dict.Status.Modified).To(BeTrue())
							Expect(dict.Status.CommonTemplate).To(BeTrue())
						} else if dict.Name == "image2" {
							Expect(dict.Status.Modified).To(BeFalse())
							Expect(dict.Status.CommonTemplate).To(BeTrue())
						}
					}
				})

				It("Should replace the common DICT spec field if the CR list includes it", func() {
					image1FromFile := image1.DeepCopy()

					storageFromFile := &cdiv1beta1.StorageSpec{
						VolumeName:       "volume-name",
						StorageClassName: ptr.To("testName"),
					}
					image1FromFile.Spec.Template.Spec.Storage = storageFromFile

					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: *image1FromFile,
						image2.Name: image2,
					}

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					modifiedImage1 := image1.DeepCopy()
					storageFromCr := &cdiv1beta1.StorageSpec{
						VolumeName: "another-class-name",

						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
						},
					}
					modifiedImage1.Spec.Template.Spec.Storage = storageFromCr.DeepCopy()
					modifiedImage1.Spec.Schedule = image1.Spec.Schedule

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedImage1, image3, image4}

					goldenImageList, err := getDataImportCronTemplates(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(goldenImageList).To(HaveLen(4))
					Expect(goldenImageList).To(HaveCap(4))

					for _, dict := range goldenImageList {
						if dict.Name == "image1" {
							Expect(dict.Spec.Template.Spec.Storage).To(BeEquivalentTo(storageFromCr))
							Expect(dict.Status.Modified).To(BeTrue())
							Expect(dict.Status.CommonTemplate).To(BeTrue())
						} else if dict.Name == "image2" {
							Expect(dict.Status.Modified).To(BeFalse())
							Expect(dict.Status.CommonTemplate).To(BeTrue())
						}
					}
				})
			})

			Context("test data import cron templates in NewSsp", func() {

				It("should return an empty list if there is no file and no list in the HyperConverged CR", func() {
					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(BeNil())
				})

				It("should return an the hard coded list if there is a file, but no list in the HyperConverged CR", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml"))).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(BeNil())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
				})

				It("should return a combined list if there is a file and a list in the HyperConverged CR", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(BeNil())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))

					var commonImages []hcov1beta1.DataImportCronTemplate
					for _, dict := range dataImportCronTemplateHardCodedMap {
						commonImages = append(commonImages, dict)
					}
					commonImages = append(commonImages, image3)
					commonImages = append(commonImages, image4)

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(hcoDictSliceToSSP(commonImages)))
				})

				It("Should not add a common DIC template if it marked as disabled", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					Expect(dataImportCronTemplateHardCodedMap).To(HaveLen(2))
					commonFedora := dataImportCronTemplateHardCodedMap["fedora-image-cron"]
					commonCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]

					fedoraDic := commonFedora.DeepCopy()
					disableDict(fedoraDic)

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*fedoraDic, image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(3))
					expected := hcoDictSliceToSSP([]hcov1beta1.DataImportCronTemplate{commonCentos8, image3, image4})
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(expected))
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(ContainElement(commonFedora))
				})

				It("Should reject if the CR list contain DIC template with the same name, and there are also common DIC templates", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					Expect(dataImportCronTemplateHardCodedMap).ToNot(BeEmpty())
					image3Modified := image3.DeepCopy()
					image3Modified.Name = image4.Name

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*image3Modified, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).To(HaveOccurred())
					Expect(ssp).To(BeNil())
				})

				It("Should reject if the CR list contain DIC template with the same name", func() {
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(false)

					Expect(dataImportCronTemplateHardCodedMap).To(BeEmpty())
					image3Modified := image3.DeepCopy()
					image3Modified.Name = image4.Name

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*image3Modified, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).To(HaveOccurred())
					Expect(ssp).To(BeNil())
				})

				It("should return a only the list from the HyperConverged CR, if the file is missing", func() {
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())
					Expect(dataImportCronTemplateHardCodedMap).To(BeEmpty())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(BeNil())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
					expected := hcoDictSliceToSSP([]hcov1beta1.DataImportCronTemplate{image3, image4})
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(expected))
				})

				It("should not return the common templates, if feature gate is false", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(false)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
					expected := hcoDictSliceToSSP([]hcov1beta1.DataImportCronTemplate{image3, image4})
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(expected))
				})

				It("should modify a common dic if it exist in the HyperConverged CR", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					Expect(dataImportCronTemplateHardCodedMap).To(HaveLen(2))
					commonFedora := dataImportCronTemplateHardCodedMap["fedora-image-cron"]
					commonCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]

					fedoraDic := commonFedora.DeepCopy()

					fedoraDic.Spec.RetentionPolicy = ptr.To(cdiv1beta1.DataImportCronRetainAll)
					fedoraDic.Spec.GarbageCollect = ptr.To(cdiv1beta1.DataImportCronGarbageCollectOutdated)
					fedoraDic.Spec.ImportsToKeep = ptr.To(int32(5))
					fedoraDic.Spec.Template.Spec.Source.Registry = &cdiv1beta1.DataVolumeSourceRegistry{
						URL: ptr.To("docker://not-the-same-image"),
					}
					fedoraDic.Spec.Template.Spec.Storage = &cdiv1beta1.StorageSpec{StorageClassName: ptr.To("someOtherStorageClass")}

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*fedoraDic, image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
					expected := hcoDictSliceToSSP([]hcov1beta1.DataImportCronTemplate{*fedoraDic, commonCentos8, image3, image4})
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(expected))
				})

				It("should add the cdi.kubevirt.io/storage.bind.immediate.requested annotation if missing", func() {
					err := os.Mkdir(dir, os.ModePerm)
					Expect(err).ToNot(HaveOccurred())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					err = commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplatesNoAnnotation.yaml"))
					Expect(err).ToNot(HaveOccurred())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)

					Expect(dataImportCronTemplateHardCodedMap).To(HaveLen(2))

					var customDicAnnotationFalse hcov1beta1.DataImportCronTemplate
					image3.DeepCopyInto(&customDicAnnotationFalse)
					customDicAnnotationFalse.Name = "custom-dict-annotation-false"
					customDicAnnotationFalse.Annotations = map[string]string{
						CDIImmediateBindAnnotation: "false",
					}

					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{customDicAnnotationFalse, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))

					annotationTrue := 0
					annotationFalse := 0
					for _, dict := range ssp.Spec.CommonTemplates.DataImportCronTemplates {
						Expect(dict.Annotations).ToNot(BeEmpty())
						if strings.HasSuffix(dict.Name, "-annotation-false") {
							Expect(dict.Annotations[CDIImmediateBindAnnotation]).To(Equal("false"))
							annotationFalse++
						} else {
							Expect(dict.Annotations[CDIImmediateBindAnnotation]).To(Equal("true"))
							annotationTrue++
						}
					}
					Expect(annotationTrue).To(Equal(2))
					Expect(annotationFalse).To(Equal(2))
				})

				It("should use custom namespace for common dicts, if defined in the hyperConverged CR", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml"))).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.CommonBootImageNamespace = ptr.To(customNS)
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(BeNil())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))

					for _, dict := range ssp.Spec.CommonTemplates.DataImportCronTemplates {
						Expect(dict.Namespace).To(Equal(customNS))
					}
				})

				It("only common dict should be with custom namespace", func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					defer func() { _ = os.RemoveAll(dir) }()
					destFile := path.Join(dir, "dataImportCronTemplates.yaml")

					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					defer os.Remove(destFile)
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())

					hco := commontestutils.NewHco()
					hco.Spec.EnableCommonBootImageImport = ptr.To(true)
					hco.Spec.CommonBootImageNamespace = ptr.To(customNS)
					hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
					ssp, _, err := NewSSP(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).ToNot(BeNil())
					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))

					var commonImages []hcov1beta1.DataImportCronTemplate
					for _, d := range dataImportCronTemplateHardCodedMap {
						dict := *d.DeepCopy()
						dict.ObjectMeta.Namespace = customNS

						commonImages = append(commonImages, dict)
					}
					commonImages = append(commonImages, image3)
					commonImages = append(commonImages, image4)

					Expect(ssp.Spec.CommonTemplates.DataImportCronTemplates).To(ContainElements(hcoDictSliceToSSP(commonImages)))
				})
			})

			Context("test applyDataImportSchedule", func() {
				It("should not set the schedule filed if missing from the status", func() {
					hco := commontestutils.NewHco()
					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}

					applyDataImportSchedule(hco)

					Expect(dataImportCronTemplateHardCodedMap[image1.Name].Spec.Schedule).To(Equal("1 */12 * * *"))
					Expect(dataImportCronTemplateHardCodedMap[image2.Name].Spec.Schedule).To(Equal("2 */12 * * *"))
				})

				It("should set the variable and the images, if the schedule is in the status field", func() {
					const schedule = "42 */1 * * *"
					hco := commontestutils.NewHco()
					hco.Status.DataImportSchedule = schedule

					dataImportCronTemplateHardCodedMap = map[string]hcov1beta1.DataImportCronTemplate{
						image1.Name: image1,
						image2.Name: image2,
					}

					applyDataImportSchedule(hco)
					for _, image := range dataImportCronTemplateHardCodedMap {
						Expect(image.Spec.Schedule).To(Equal(schedule))
					}
				})
			})

			Context("test data import cron templates in Status", func() {
				var destFile string
				BeforeEach(func() {
					Expect(os.Mkdir(dir, os.ModePerm)).To(Succeed())
					destFile = path.Join(dir, "dataImportCronTemplates.yaml")
					Expect(
						commontestutils.CopyFile(destFile, path.Join(testFilesLocation, "dataImportCronTemplates.yaml")),
					).To(Succeed())
					Expect(readDataImportCronTemplatesFromFile()).To(Succeed())
				})

				AfterEach(func() {
					_ = os.RemoveAll(dir)
					_ = os.Remove(destFile)
				})

				Context("on SSP create", func() {
					It("should create ssp with 2 common DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeTrue())
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 2 custom DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(false)
						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeFalse())
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 2 common and 2 custom DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 1 common and 2 custom DICTs, when one of the common is disabled", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]

						disabledCentos8 := sspCentos8.DeepCopy()
						disableDict(disabledCentos8)

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*disabledCentos8, image3, image4}
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(3))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(3))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Name).ToNot(Equal("centos8-image-cron"))
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 1 modified common DICT and 2 custom DICTs, when one of the common is modified", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]

						modifiedCentos8 := sspCentos8.DeepCopy()

						modifiedStorage := &cdiv1beta1.StorageSpec{
							StorageClassName: ptr.To("anotherStorageClassName"),
							VolumeName:       "volumeName",
						}

						modifiedCentos8.Spec.Template.Spec.Storage = modifiedStorage.DeepCopy()
						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedCentos8, image3, image4}
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range foundResource.Spec.CommonTemplates.DataImportCronTemplates {
							if dict.Name == "centos8-image-cron" {
								Expect(dict.Spec.Template.Spec.Storage).To(Equal(modifiedStorage))
							}
						}

						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}

							if dict.Name == "centos8-image-cron" {
								Expect(dict.Status.Modified).To(BeTrue())
							} else {
								Expect(dict.Status.Modified).To(BeFalse())
							}
						}
					})

					It("should create ssp with custom namespace", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						hco.Spec.CommonBootImageNamespace = ptr.To(customNS)
						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeTrue())
							Expect(dict.Status.Modified).To(BeFalse())
							Expect(dict.Namespace).To(Equal(customNS))
						}
					})

					It("modified or custom dicts should not be with custom namespace", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]

						modifiedCentos8 := sspCentos8.DeepCopy()

						modifiedStorage := &cdiv1beta1.StorageSpec{
							StorageClassName: ptr.To("anotherStorageClassName"),
							VolumeName:       "volumeName",
						}

						modifiedCentos8.ObjectMeta.Namespace = ""
						modifiedCentos8.Spec.Template.Spec.Storage = modifiedStorage.DeepCopy()
						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedCentos8, image3, image4}
						hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

						expectedResource, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeTrue())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundResource := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
								foundResource),
						).ToNot(HaveOccurred())
						Expect(foundResource.Name).To(Equal(expectedResource.Name))
						Expect(foundResource.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range foundResource.Spec.CommonTemplates.DataImportCronTemplates {
							if dict.Name == "centos8-image-cron" {
								Expect(dict.Spec.Template.Spec.Storage).To(Equal(modifiedStorage))
							}

							if dict.Name == "fedora-image-cron" {
								Expect(dict.Namespace).To(Equal(customNS))
							} else {
								Expect(dict.Namespace).To(Equal(""))
							}
						}

						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}

							if dict.Name == "centos8-image-cron" {
								Expect(dict.Status.Modified).To(BeTrue())
							} else {
								Expect(dict.Status.Modified).To(BeFalse())
							}
						}
					})
				})

				Context("on SSP update", func() {
					It("should create ssp with 2 common DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeFalse())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeTrue())
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 2 custom DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(false)

						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}

						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeFalse())
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 2 common and 2 custom DICTs", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)

						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{image3, image4}

						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 1 common and 2 custom DICTs, when one of the common is disabled", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)

						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())

						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]
						disabledCentos8 := sspCentos8.DeepCopy()
						disableDict(disabledCentos8)

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*disabledCentos8, image3, image4}

						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(3))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(3))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Name).ToNot(Equal("centos8-image-cron"))
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}
							Expect(dict.Status.Modified).To(BeFalse())
						}
					})

					It("should create ssp with 1 modified common DICT and 2 custom DICTs, when one of the common is modified", func() {
						const scName = "anotherStorageClassName"
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)

						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())

						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]
						modifiedCentos8 := sspCentos8.DeepCopy()
						modifiedCentos8.Spec.Template.Spec.Storage = &cdiv1beta1.StorageSpec{StorageClassName: ptr.To(scName)}

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedCentos8, image3, image4}

						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range foundSSP.Spec.CommonTemplates.DataImportCronTemplates {
							if dict.Name == "centos8-image-cron" {
								Expect(dict.Spec.Template.Spec.Storage.StorageClassName).To(HaveValue(Equal(scName)))
							}
						}

						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}

							if dict.Name == "centos8-image-cron" {
								Expect(dict.Status.Modified).To(BeTrue())
							} else {
								Expect(dict.Status.Modified).To(BeFalse())
							}
						}
					})

					It("should create ssp with 2 common DICTs, in a custom namespace", func() {
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)
						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())
						cl := commontestutils.InitClient([]client.Object{origSSP})

						hco.Spec.CommonBootImageNamespace = ptr.To(customNS)
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(2))
						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(2))
						for _, dict := range hco.Status.DataImportCronTemplates {
							Expect(dict.Status.CommonTemplate).To(BeTrue())
							Expect(dict.Status.Modified).To(BeFalse())
							Expect(dict.Namespace).To(Equal(customNS))
						}
					})

					It("only non modified common dict should use the custom namespace", func() {
						const scName = "anotherStorageClassName"
						hco.Spec.EnableCommonBootImageImport = ptr.To(true)

						origSSP, _, err := NewSSP(hco)
						Expect(err).ToNot(HaveOccurred())

						sspCentos8 := dataImportCronTemplateHardCodedMap["centos8-image-cron"]
						modifiedCentos8 := sspCentos8.DeepCopy()
						modifiedCentos8.Spec.Template.Spec.Storage = &cdiv1beta1.StorageSpec{StorageClassName: ptr.To(scName)}
						modifiedCentos8.ObjectMeta.Namespace = ""

						hco.Spec.DataImportCronTemplates = []hcov1beta1.DataImportCronTemplate{*modifiedCentos8, image3, image4}
						hco.Spec.CommonBootImageNamespace = ptr.To(customNS)

						cl := commontestutils.InitClient([]client.Object{origSSP})
						handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
						res := handler.ensure(req)
						Expect(res.Created).To(BeFalse())
						Expect(res.Updated).To(BeTrue())
						Expect(res.Overwritten).To(BeFalse())
						Expect(res.UpgradeDone).To(BeFalse())
						Expect(res.Err).ToNot(HaveOccurred())

						foundSSP := &sspv1beta2.SSP{}
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: origSSP.Name, Namespace: origSSP.Namespace},
								foundSSP),
						).ToNot(HaveOccurred())
						Expect(foundSSP.Name).To(Equal(origSSP.Name))
						Expect(foundSSP.Spec.CommonTemplates.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range foundSSP.Spec.CommonTemplates.DataImportCronTemplates {
							if dict.Name == "centos8-image-cron" {
								Expect(dict.Spec.Template.Spec.Storage.StorageClassName).To(HaveValue(Equal(scName)))
							}

							if dict.Name == "fedora-image-cron" {
								Expect(dict.Namespace).To(Equal(customNS))
							} else {
								Expect(dict.Namespace).To(Equal(""))
							}
						}

						Expect(hco.Status.DataImportCronTemplates).To(HaveLen(4))
						for _, dict := range hco.Status.DataImportCronTemplates {
							if dict.Name == image3.Name || dict.Name == image4.Name {
								Expect(dict.Status.CommonTemplate).To(BeFalse())
							} else {
								Expect(dict.Status.CommonTemplate).To(BeTrue())
							}

							if dict.Name == "centos8-image-cron" {
								Expect(dict.Status.Modified).To(BeTrue())
							} else {
								Expect(dict.Status.Modified).To(BeFalse())
							}
						}
					})
				})
			})

			Context("test isDataImportCronTemplateEnabled", func() {
				var image *hcov1beta1.DataImportCronTemplate

				BeforeEach(func() {
					image = image1.DeepCopy()
				})

				It("should be true if the annotation is missing", func() {
					image.Annotations = nil
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeTrue())
				})

				It("should be true if the annotation is missing", func() {
					image.Annotations = make(map[string]string)
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeTrue())
				})

				It("should be true if the annotation is set to 'true'", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: "true"}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeTrue())
				})

				It("should be true if the annotation is set to 'TRUE'", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: "TRUE"}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeTrue())
				})

				It("should be true if the annotation is set to 'TrUe'", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: "TrUe"}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeTrue())
				})

				It("should be false if the annotation is empty", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: ""}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeFalse())
				})

				It("should be false if the annotation is set to 'false'", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: "false"}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeFalse())
				})

				It("should be false if the annotation is set to 'something-else'", func() {
					image.Annotations = map[string]string{hcoutil.DataImportCronEnabledAnnotation: "something-else"}
					Expect(isDataImportCronTemplateEnabled(*image)).To(BeFalse())
				})
			})
		})

		Context("TLSSecurityProfile", func() {

			intermediateTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:         openshiftconfigv1.TLSProfileIntermediateType,
				Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
			}
			modernTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:   openshiftconfigv1.TLSProfileModernType,
				Modern: &openshiftconfigv1.ModernTLSProfile{},
			}

			It("should modify TLSSecurityProfile on SSP CR according to ApiServer or HCO CR", func() {
				existingResource, _, err := NewSSP(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(existingResource.Spec.TLSSecurityProfile).To(Equal(intermediateTLSSecurityProfile))

				// now, modify HCO's TLSSecurityProfile
				hco.Spec.TLSSecurityProfile = modernTLSSecurityProfile

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.TLSSecurityProfile).To(Equal(modernTLSSecurityProfile))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite TLSSecurityProfile if directly set on SSP CR", func() {
				hco.Spec.TLSSecurityProfile = intermediateTLSSecurityProfile
				existingResource, _, err := NewSSP(hco)
				Expect(err).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in CDI CR
				req.HCOTriggered = false

				// now, modify SSP TLSSecurityProfile
				existingResource.Spec.TLSSecurityProfile = modernTLSSecurityProfile

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newSspHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.TLSSecurityProfile).To(Equal(hco.Spec.TLSSecurityProfile))
				Expect(foundResource.Spec.TLSSecurityProfile).ToNot(Equal(existingResource.Spec.TLSSecurityProfile))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

	})
})

func enableDict(dict *hcov1beta1.DataImportCronTemplate) {
	if dict.Annotations == nil {
		dict.Annotations = make(map[string]string)
	}
	dict.Annotations[hcoutil.DataImportCronEnabledAnnotation] = "true"
}

func disableDict(dict *hcov1beta1.DataImportCronTemplate) {
	if dict.Annotations == nil {
		dict.Annotations = make(map[string]string)
	}
	dict.Annotations[hcoutil.DataImportCronEnabledAnnotation] = "false"
}
