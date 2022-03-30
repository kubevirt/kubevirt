package operands

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Namespace Operand", func() {
	Context("Namespace", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest
		customAnnotation := "customAnnotation"
		customValue := "customValue"

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should reconcile selected annotations to default", func() {
			existingResource := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: hco.Namespace,
					Annotations: map[string]string{
						hcoutil.OpenshiftNodeSelectorAnn: "",
					},
				},
			}
			existingResource.Annotations[hcoutil.OpenshiftNodeSelectorAnn] = customValue
			existingResource.Annotations[customAnnotation] = customValue

			req.HCOTriggered = false

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := newNamespaceHandler(cl, commonTestUtils.GetScheme())
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.Namespace{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Annotations[hcoutil.OpenshiftNodeSelectorAnn]).To(Not(BeIdenticalTo(customValue)))
			Expect(foundResource.Annotations[hcoutil.OpenshiftNodeSelectorAnn]).To(BeIdenticalTo(""))
			Expect(foundResource.Annotations[customAnnotation]).To(BeIdenticalTo(customValue))

		})

		It("should add 'openshift.io/node-selector' annotation if missing", func() {
			existingResource := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: hco.Namespace,
					Annotations: map[string]string{
						hcoutil.OpenshiftNodeSelectorAnn: "",
					},
				},
			}
			delete(existingResource.Annotations, hcoutil.OpenshiftNodeSelectorAnn)
			Expect(existingResource.Annotations).To(Not(HaveKey(hcoutil.OpenshiftNodeSelectorAnn)))

			req.HCOTriggered = true

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := newNamespaceHandler(cl, commonTestUtils.GetScheme())
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &corev1.Namespace{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Annotations).To(HaveKey(hcoutil.OpenshiftNodeSelectorAnn))
			Expect(foundResource.Annotations[hcoutil.OpenshiftNodeSelectorAnn]).To(BeIdenticalTo(""))

		})

	})

})
