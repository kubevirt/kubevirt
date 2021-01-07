package operands

import (
	"context"
	"fmt"
	"os"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	v1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	lifecycleapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSP Operands", func() {

	Context("SSP", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewSSP(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := newSspHandler(cl, commonTestUtils.GetScheme())
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1beta1.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewSSP(hco)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := newSspHandler(cl, commonTestUtils.GetScheme())
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			expectedResource := NewSSP(hco)
			existingResource := expectedResource.DeepCopy()
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			replicas := int32(defaultTemplateValidatorReplicas * 2) // non-default value
			existingResource.Spec.TemplateValidator.Replicas = &replicas
			existingResource.Spec.CommonTemplates.Namespace = "foobar"
			existingResource.Spec.NodeLabeller.Placement = &lifecycleapi.NodePlacement{
				NodeSelector: map[string]string{"foo": "bar"},
			}

			req.HCOTriggered = false // mock a reconciliation triggered by a change in NewKubeVirtCommonTemplateBundle CR

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := newSspHandler(cl, commonTestUtils.GetScheme())
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1beta1.SSP{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec).To(Equal(expectedResource.Spec))
		})

		Context("Node placement", func() {

			It("should add node placement if missing", func() {
				existingResource := NewSSP(hco, commonTestUtils.Namespace)

				hco.Spec.Workloads.NodePlacement = commonTestUtils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commonTestUtils.NewOtherNodePlacement()

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &sspv1beta1.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.NodeLabeller.Placement).To(BeZero())
				Expect(existingResource.Spec.TemplateValidator.Placement).To(BeZero())
				Expect(*foundResource.Spec.NodeLabeller.Placement).To(Equal(*hco.Spec.Workloads.NodePlacement))
				Expect(*foundResource.Spec.TemplateValidator.Placement).To(Equal(*hco.Spec.Infra.NodePlacement))
				Expect(req.Conditions).To(BeEmpty())
			})

			It("should remove node placement if missing in HCO CR", func() {

				hcoNodePlacement := commonTestUtils.NewHco()
				hcoNodePlacement.Spec.Workloads.NodePlacement = commonTestUtils.NewNodePlacement()
				hcoNodePlacement.Spec.Infra.NodePlacement = commonTestUtils.NewOtherNodePlacement()
				existingResource := NewSSP(hcoNodePlacement, commonTestUtils.Namespace)

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &sspv1beta1.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.NodeLabeller.Placement).ToNot(BeZero())
				Expect(existingResource.Spec.TemplateValidator.Placement).ToNot(BeZero())
				Expect(foundResource.Spec.NodeLabeller.Placement).To(BeZero())
				Expect(foundResource.Spec.TemplateValidator.Placement).To(BeZero())
				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify node placement according to HCO CR", func() {

				hco.Spec.Workloads.NodePlacement = commonTestUtils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commonTestUtils.NewOtherNodePlacement()
				existingResource := NewSSP(hco, commonTestUtils.Namespace)

				// now, modify HCO's node placement
				seconds12 := int64(12)
				hco.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key12", Operator: "operator12", Value: "value12", Effect: "effect12", TolerationSeconds: &seconds12,
				})
				hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

				seconds34 := int64(34)
				hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: &seconds34,
				})
				hco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "something entirely else"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &sspv1beta1.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.NodeLabeller.Placement.Affinity.NodeAffinity).ToNot(BeZero())
				Expect(existingResource.Spec.NodeLabeller.Placement.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.NodeLabeller.Placement.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(existingResource.Spec.TemplateValidator.Placement.Affinity.NodeAffinity).ToNot(BeZero())
				Expect(existingResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.TemplateValidator.Placement.NodeSelector["key3"]).Should(Equal("value3"))

				Expect(foundResource.Spec.NodeLabeller.Placement.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(foundResource.Spec.NodeLabeller.Placement.Tolerations).To(HaveLen(3))
				Expect(foundResource.Spec.NodeLabeller.Placement.NodeSelector["key1"]).Should(Equal("something else"))
				Expect(foundResource.Spec.TemplateValidator.Placement.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(foundResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(3))
				Expect(foundResource.Spec.TemplateValidator.Placement.NodeSelector["key3"]).Should(Equal("something entirely else"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite node placement if directly set on SSP CR", func() {
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewOtherNodePlacement()}
				existingResource := NewSSP(hco, commonTestUtils.Namespace)

				// mock a reconciliation triggered by a change in NewKubeVirtNodeLabellerBundle CR
				req.HCOTriggered = false

				// now, modify NodeLabeller node placement
				seconds12 := int64(12)
				existingResource.Spec.NodeLabeller.Placement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key12", Operator: "operator12", Value: "value12", Effect: "effect12", TolerationSeconds: &seconds12,
				})
				existingResource.Spec.NodeLabeller.Placement.NodeSelector["key1"] = "BADvalue1"

				// and modify TemplateValidator node placement
				seconds34 := int64(34)
				existingResource.Spec.TemplateValidator.Placement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: &seconds34,
				})
				existingResource.Spec.TemplateValidator.Placement.NodeSelector["key3"] = "BADvalue3"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).To(BeNil())

				foundResource := &sspv1beta1.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.NodeLabeller.Placement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.NodeLabeller.Placement.NodeSelector["key1"]).Should(Equal("BADvalue1"))
				Expect(existingResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.TemplateValidator.Placement.NodeSelector["key3"]).Should(Equal("BADvalue3"))

				Expect(foundResource.Spec.NodeLabeller.Placement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.NodeLabeller.Placement.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(foundResource.Spec.TemplateValidator.Placement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.TemplateValidator.Placement.NodeSelector["key3"]).Should(Equal("value3"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("SSP Upgrade", func() {

			It("shouldn't remove old CRDs if upgrade isn't done", func() {
				oldCrds := oldSSPCrdsAsObjects()
				cl := commonTestUtils.InitClient(oldCrds)

				// Simulate ongoing upgrade
				req.SetUpgradeMode(true)

				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)

				Expect(res.Created).To(BeTrue())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCrds := apiextensionsv1.CustomResourceDefinitionList{}
				Expect(cl.List(context.TODO(), &foundCrds)).To(BeNil())
				Expect(foundCrds.Items).To(HaveLen(len(oldCrds)))
			})

			It("should remove old CRDs if general upgrade is done", func() {
				oldCrds := oldSSPCrdsAsObjects()
				cl := commonTestUtils.InitClient(oldCrds)

				// Simulate no upgrade
				req.SetUpgradeMode(false)

				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)

				Expect(res.Created).To(BeTrue())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCrds := apiextensionsv1.CustomResourceDefinitionList{}
				Expect(cl.List(context.TODO(), &foundCrds)).To(BeNil())
				Expect(foundCrds.Items).To(BeEmpty())
			})

			It("should remove old CRDs if SSP upgrade is done", func() {
				existingResource := NewSSP(hco, commonTestUtils.Namespace)
				existingResource.Status.Conditions = []conditionsv1.Condition{
					{
						Type:   conditionsv1.ConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   conditionsv1.ConditionDegraded,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   conditionsv1.ConditionProgressing,
						Status: corev1.ConditionFalse,
					},
				}

				// Set the expected SSP version that indicates upgrade complete.
				// Note: the value doesn't really matter, even when we move beyond 2.6
				const expectedSSPVersion = "2.6"
				os.Setenv(hcoutil.SspVersionEnvV, expectedSSPVersion)
				existingResource.Status.ObservedVersion = expectedSSPVersion

				oldCrds := oldSSPCrdsAsObjects()
				objects := append(oldCrds, existingResource)
				cl := commonTestUtils.InitClient(objects)

				// Simulate ongoing upgrade
				req.SetUpgradeMode(true)

				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)

				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeTrue())
				Expect(res.Err).To(BeNil())

				foundCrds := apiextensionsv1.CustomResourceDefinitionList{}
				Expect(cl.List(context.TODO(), &foundCrds)).To(BeNil())
				Expect(foundCrds.Items).To(BeEmpty())
			})

			It("should remove old related objects if upgrade is done", func() {
				// Simulate no upgrade
				req.SetUpgradeMode(false)

				// Initialize RelatedObjects with a bunch of objects
				// including old SSP ones.
				for _, objRef := range oldSSPRelatedObjects() {
					v1.SetObjectReference(&hco.Status.RelatedObjects, objRef)
				}
				for _, objRef := range otherRelatedObjects() {
					v1.SetObjectReference(&hco.Status.RelatedObjects, objRef)
				}

				cl := commonTestUtils.InitClient(nil)
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)

				Expect(res.Created).To(BeTrue())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				Expect(hco.Status.RelatedObjects).To(HaveLen(len(otherRelatedObjects())))
				for _, objRef := range oldSSPRelatedObjects() {
					Expect(hco.Status.RelatedObjects).ToNot(ContainElement(objRef))
				}
			})

			It("should retry removing old related objects when they fail to be removed from the status", func() {
				// Simulate no upgrade
				req.SetUpgradeMode(false)

				// Initialize RelatedObjects with a bunch of objects
				// including old SSP ones.
				for _, objRef := range oldSSPRelatedObjects() {
					v1.SetObjectReference(&hco.Status.RelatedObjects, objRef)
				}
				for _, objRef := range otherRelatedObjects() {
					v1.SetObjectReference(&hco.Status.RelatedObjects, objRef)
				}

				cl := commonTestUtils.InitClient(nil)
				handler := newSspHandler(cl, commonTestUtils.GetScheme())
				res := handler.ensure(req)

				Expect(res.Created).To(BeTrue())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				// Now simulate "status update failure",
				// i.e. related objects aren't removed.
				for _, objRef := range oldSSPRelatedObjects() {
					v1.SetObjectReference(&hco.Status.RelatedObjects, objRef)
				}

				// Simulate another reconciliation cycle
				res = handler.ensure(req)

				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				// len+1 because the (new) SSP object is now added to RelatedObjects
				Expect(hco.Status.RelatedObjects).To(HaveLen(len(otherRelatedObjects()) + 1))
				for _, objRef := range oldSSPRelatedObjects() {
					Expect(hco.Status.RelatedObjects).ToNot(ContainElement(objRef))
				}
			})
		})
	})
})

func oldSSPCrds() []*apiextensionsv1.CustomResourceDefinition {
	names := []string{
		"kubevirtcommontemplatesbundles.ssp.kubevirt.io",
		"kubevirtmetricsaggregations.ssp.kubevirt.io",
		"kubevirtnodelabellerbundles.ssp.kubevirt.io",
		"kubevirttemplatevalidators.ssp.kubevirt.io",
		"kubevirtcommontemplatesbundles.kubevirt.io",
		"kubevirtmetricsaggregations.kubevirt.io",
		"kubevirtnodelabellerbundles.kubevirt.io",
		"kubevirttemplatevalidators.kubevirt.io",
	}

	crds := make([]*apiextensionsv1.CustomResourceDefinition, 0, len(names))
	for _, name := range names {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		crds = append(crds, crd)
	}

	return crds
}

func oldSSPCrdsAsObjects() []runtime.Object {
	crds := oldSSPCrds()
	objs := make([]runtime.Object, 0, len(crds))
	for _, crd := range crds {
		objs = append(objs, crd)
	}

	return objs
}

func oldSSPRelatedObjects() []corev1.ObjectReference {
	return []corev1.ObjectReference{
		{
			APIVersion: "ssp.kubevirt.io/v1",
			Kind:       "KubevirtCommonTemplatesBundle",
			Name:       "common-templates-kubevirt-hyperconverged",
			Namespace:  "openshift",
		},
		{
			APIVersion: "ssp.kubevirt.io/v1",
			Kind:       "KubevirtNodeLabellerBundle",
			Name:       "node-labeller-kubevirt-hyperconverged",
			Namespace:  "kubevirt-hyperconverged",
		},
		{
			APIVersion: "ssp.kubevirt.io/v1",
			Kind:       "KubevirtTemplateValidator",
			Name:       "template-validator-kubevirt-hyperconverged",
			Namespace:  "kubevirt-hyperconverged",
		},
		{
			APIVersion: "ssp.kubevirt.io/v1",
			Kind:       "KubevirtMetricsAggregation",
			Name:       "metrics-aggregation-kubevirt-hyperconverged",
			Namespace:  "kubevirt-hyperconverged",
		},
	}
}

func otherRelatedObjects() []corev1.ObjectReference {
	return []corev1.ObjectReference{
		{
			APIVersion: "kubevirt.io/v1alpha3",
			Kind:       "Kubevirt",
			Name:       "kubevirt-kubevirt-hyperconverged",
			Namespace:  "openshift",
		},
		{
			APIVersion: "cdi.kubevirt.io/v1beta1",
			Kind:       "CDI",
			Name:       "cdi-kubevirt-hyperconverged",
			Namespace:  "kubevirt-hyperconverged",
		},
	}
}
