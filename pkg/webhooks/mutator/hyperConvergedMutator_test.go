package mutator

import (
	"context"
	"fmt"
	"os"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	"k8s.io/utils/pointer"

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
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
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
			cr.Spec.FeatureGates.Root = pointer.Bool(false)
			evictionStrategy := kubevirtcorev1.EvictionStrategyLiveMigrate
			cr.Spec.EvictionStrategy = &evictionStrategy
			cli = commontestutils.InitClient(nil)
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

		DescribeTable("Check nonRoot -> root FG transition", func(initialNonRoot *bool, initialRoot *bool, patches []jsonpatch.JsonPatchOperation) {
			cr.Spec.FeatureGates.NonRoot = initialNonRoot //nolint SA1019
			cr.Spec.FeatureGates.Root = initialRoot

			req := admission.Request{AdmissionRequest: newCreateRequest(cr, hcoV1beta1Codec)}

			res := mutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())

			Expect(res.Patches).To(Equal(patches))
		},
			Entry("should set only the default value for root if nothing is there",
				nil,
				nil,
				[]jsonpatch.JsonPatchOperation{jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/featureGates/root",
					Value:     false,
				}},
			),
			Entry("should set root=false if nonRoot was true",
				pointer.Bool(true),
				nil,
				[]jsonpatch.JsonPatchOperation{jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/featureGates/root",
					Value:     false,
				}},
			),
			Entry("should set root=true if nonRoot was false",
				pointer.Bool(false),
				nil,
				[]jsonpatch.JsonPatchOperation{jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      "/spec/featureGates/root",
					Value:     true,
				}},
			),
			Entry("should do nothing if both the values are already there (the CEL expression enforces the consistency) - 1",
				pointer.Bool(false),
				pointer.Bool(true),
				nil,
			),
			Entry("should do nothing if both the values are already there (the CEL expression enforces the consistency) - 2",
				pointer.Bool(true),
				pointer.Bool(false),
				nil,
			),
		)

		It("should handle multiple DICTs and nonRoot -> root FG transition at the same time", func() {
			cr.Spec.FeatureGates.NonRoot = pointer.Bool(false) //nolint SA1019
			cr.Spec.FeatureGates.Root = nil

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

			Expect(res.Patches).To(HaveLen(3))
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
			Expect(res.Patches[2]).To(Equal(jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/featureGates/root",
				Value:     true,
			}))
		})

		Context("Check defaults for cluster level EvictionStrategy", func() {

			getClusterInfo := hcoutil.GetClusterInfo

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			DescribeTable("check EvictionStrategy default", func(SNO bool, strategy *kubevirtcorev1.EvictionStrategy, patches []jsonpatch.JsonPatchOperation) {
				if SNO {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSNOMock{}
					}
				} else {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
				}

				cr.Spec.EvictionStrategy = strategy

				req := admission.Request{AdmissionRequest: newCreateRequest(cr, hcoV1beta1Codec)}

				res := mutator.Handle(context.TODO(), req)
				Expect(res.Allowed).To(BeTrue())

				Expect(res.Patches).To(Equal(patches))
			},
				Entry("should set EvictionStrategyNone if not set and on SNO",
					true,
					nil,
					[]jsonpatch.JsonPatchOperation{jsonpatch.JsonPatchOperation{
						Operation: "add",
						Path:      "/spec/evictionStrategy",
						Value:     kubevirtcorev1.EvictionStrategyNone,
					}},
				),
				Entry("should not override EvictionStrategy if set and on SNO - 1",
					true,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyNone),
					nil,
				),
				Entry("should not override EvictionStrategy if set and on SNO - 2",
					true,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyLiveMigrate),
					nil,
				),
				Entry("should not override EvictionStrategy if set and on SNO - 3",
					true,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyExternal),
					nil,
				),
				Entry("should set EvictionStrategyLiveMigrate if not set and not on SNO",
					false,
					nil,
					[]jsonpatch.JsonPatchOperation{jsonpatch.JsonPatchOperation{
						Operation: "add",
						Path:      "/spec/evictionStrategy",
						Value:     kubevirtcorev1.EvictionStrategyLiveMigrate,
					}},
				),
				Entry("should not override EvictionStrategy if set and not on SNO - 1",
					false,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyNone),
					nil,
				),
				Entry("should not override EvictionStrategy if set and not on SNO - 2",
					false,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyLiveMigrate),
					nil,
				),
				Entry("should not override EvictionStrategy if set and not on SNO - 3",
					false,
					pointerEvictionStrategy(kubevirtcorev1.EvictionStrategyExternal),
					nil,
				),
			)
		})

	})
})

func initHCMutator(s *runtime.Scheme, testClient client.Client) *HyperConvergedMutator {
	decoder := admission.NewDecoder(s)
	mutator := NewHyperConvergedMutator(testClient, decoder)

	return mutator
}

func pointerEvictionStrategy(strategy kubevirtcorev1.EvictionStrategy) *kubevirtcorev1.EvictionStrategy {
	str := strategy
	return &str
}
