package operands

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	ttov1alpha1 "github.com/kubevirt/tekton-tasks-operator/api/v1alpha1"
)

var _ = Describe("TTO Operands", func() {
	Context("TTO", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewTTO(hco)

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newTtoHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &ttov1alpha1.TektonTasks{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewTTO(hco)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newTtoHandler(cl, commonTestUtils.GetScheme()))
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
			pipelinesNamespace := "nonDefault"
			hco.Spec.TektonPipelinesNamespace = &pipelinesNamespace
			expectedResource := NewTTO(hco)

			existingResource := expectedResource.DeepCopy()
			existingResource.Spec.FeatureGates.DeployTektonTaskResources = true

			req.HCOTriggered = false // mock a reconciliation triggered by a change in NewKubeVirtCommonTemplateBundle CR

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newTtoHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &ttov1alpha1.TektonTasks{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Spec).To(Equal(expectedResource.Spec))
			Expect(foundResource.Spec.Pipelines.Namespace).To(Equal(pipelinesNamespace), "pipelines namespace should equal")

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		Context("Cache", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := newTtoHandler(cl, commonTestUtils.GetScheme())

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*ttoHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*ttoHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*ttoHooks).cache).Should(BeIdenticalTo(cr))

					tto1, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(tto1).ToNot(BeNil())
					Expect(cr).Should(BeIdenticalTo(tto1))
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*ttoHooks).reset()
				Expect(handler.hooks.(*ttoHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*ttoHooks).cache).ToNot(BeNil())

				handler.hooks.(*ttoHooks).reset()
				Expect(handler.hooks.(*ttoHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*ttoHooks).cache).ToNot(BeNil())

				Expect(crI).ToNot(BeIdenticalTo(crII))
				Expect(handler.hooks.(*ttoHooks).cache).ToNot(BeIdenticalTo(crI))
				Expect(handler.hooks.(*ttoHooks).cache).To(BeIdenticalTo(crII))
			})
		})
	})
})
