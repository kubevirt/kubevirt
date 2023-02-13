package mutator

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("test HyperConverged mutator", func() {
	s := scheme.Scheme
	_ = v1beta1.AddToScheme(s)

	codecFactory := serializer.NewCodecFactory(s)
	hcoV1beta1Codec := codecFactory.LegacyCodec(v1beta1.SchemeGroupVersion)

	Context("Check mutating webhook for create operation", func() {

		var (
			cr      *v1beta1.HyperConverged
			cli     client.Client
			mutator *HyperConvergedMutator
		)

		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(Succeed())
			cr = &v1beta1.HyperConverged{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.HyperConvergedName,
					Namespace: HcoValidNamespace,
				},
				Spec: v1beta1.HyperConvergedSpec{},
			}
			cli = commonTestUtils.InitClient(nil)
			mutator = initHCMutator(s, cli)
		})

		DescribeTable("check dict annotation on create", func(annotations map[string]string, expectedPatches *jsonpatch.JsonPatchOperation) {
			cr.Spec.DataImportCronTemplates = []v1beta1.DataImportCronTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "dictName",
						Annotations: annotations,
					},
				},
			}

			req := admission.Request{AdmissionRequest: newCreateRequest(cr, hcoV1beta1Codec)}

			res := mutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())

			if expectedPatches == nil {
				Expect(res.Patches).To(BeEmpty())
			} else {
				Expect(res.Patches).To(HaveLen(1))
				Expect(res.Patches[0]).To(Equal(*expectedPatches))
			}
		},
			Entry("no annotations", nil, &jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(annotationPathTemplate, 0),
				Value:     map[string]string{operands.CDIImmediateBindAnnotation: "true"},
			}),
			Entry("different annotations", map[string]string{"something/else": "value"}, &jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(dictAnnotationPathTemplate, 0),
				Value:     "true",
			}),
			Entry("annotation=true", map[string]string{operands.CDIImmediateBindAnnotation: "true"}, nil),
			Entry("annotation=false", map[string]string{operands.CDIImmediateBindAnnotation: "false"}, nil),
		)

		DescribeTable("check dict annotation on update", func(annotations map[string]string, expectedPatches *jsonpatch.JsonPatchOperation) {
			origCR := cr.DeepCopy()
			cr.Spec.DataImportCronTemplates = []v1beta1.DataImportCronTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "dictName",
						Annotations: annotations,
					},
				},
			}

			req := admission.Request{AdmissionRequest: newUpdateRequest(origCR, cr, hcoV1beta1Codec)}

			res := mutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())

			if expectedPatches == nil {
				Expect(res.Patches).To(BeEmpty())
			} else {
				Expect(res.Patches).To(HaveLen(1))
				Expect(res.Patches[0]).To(Equal(*expectedPatches))
			}
		},
			Entry("no annotations", nil, &jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(annotationPathTemplate, 0),
				Value:     map[string]string{operands.CDIImmediateBindAnnotation: "true"},
			}),
			Entry("different annotations", map[string]string{"something/else": "value"}, &jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(dictAnnotationPathTemplate, 0),
				Value:     "true",
			}),
			Entry("annotation=true", map[string]string{operands.CDIImmediateBindAnnotation: "true"}, nil),
			Entry("annotation=false", map[string]string{operands.CDIImmediateBindAnnotation: "false"}, nil),
		)

		It("should handle multiple DICTs", func() {
			cr.Spec.DataImportCronTemplates = []v1beta1.DataImportCronTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "no-annotation",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "different-annotation",
						Annotations: map[string]string{"something/else": "value"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "annotation-true",
						Annotations: map[string]string{operands.CDIImmediateBindAnnotation: "true"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "annotation-true",
						Annotations: map[string]string{operands.CDIImmediateBindAnnotation: "false"},
					},
				},
			}

			req := admission.Request{AdmissionRequest: newCreateRequest(cr, hcoV1beta1Codec)}

			res := mutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())

			Expect(res.Patches).To(HaveLen(2))
			Expect(res.Patches[0]).To(Equal(jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(annotationPathTemplate, 0),
				Value:     map[string]string{operands.CDIImmediateBindAnnotation: "true"},
			}))
			Expect(res.Patches[1]).To(Equal(jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(dictAnnotationPathTemplate, 1),
				Value:     "true",
			}))
		})
	})
})

func initHCMutator(s *runtime.Scheme, testClient client.Client) *HyperConvergedMutator {
	mutator := NewHyperConvergedMutator(testClient)

	decoder, err := admission.NewDecoder(s)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	ExpectWithOffset(1, mutator.InjectDecoder(decoder)).Should(Succeed())

	return mutator
}
