package mutator

import (
	"context"
	"errors"
	"os"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
)

const (
	ResourceInvalidNamespace = "an-arbitrary-hcoNamespace"
	HcoValidNamespace        = "kubevirt-hyperconverged"
)

var (
	ErrFakeHcoError = errors.New("fake HyperConverged error")
)

func TestMutatorWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mutator Webhooks Suite")
}

var _ = Describe("webhooks mutator", func() {
	s := scheme.Scheme

	for _, f := range []func(*runtime.Scheme) error{
		v1beta1.AddToScheme,
		cdiv1beta1.AddToScheme,
		kubevirtcorev1.AddToScheme,
		networkaddonsv1.AddToScheme,
		sspv1beta1.AddToScheme,
		corev1.AddToScheme,
	} {
		Expect(f(s)).To(Succeed())
	}

	codecFactory := serializer.NewCodecFactory(s)
	corev1Codec := codecFactory.LegacyCodec(corev1.SchemeGroupVersion)

	Context("Check mutating webhook for hcoNamespace deletion", func() {
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(Succeed())
		})

		cr := &v1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.HyperConvergedName,
				Namespace: HcoValidNamespace,
			},
			Spec: v1beta1.HyperConvergedSpec{},
		}

		var ns runtime.Object = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: HcoValidNamespace,
			},
		}

		It("should allow the delete of the hcoNamespace if Hyperconverged CR doesn't exist", func() {
			cli := commonTestUtils.InitClient(nil)
			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: newRequest(admissionv1.Delete, ns, corev1Codec)}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())
		})

		It("should not allow the delete of the hcoNamespace if Hyperconverged CR exists", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{cr})
			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: newRequest(admissionv1.Delete, ns, corev1Codec)}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeFalse())
		})

		It("should not allow when the request is not valid", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{cr})
			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{Operation: admissionv1.Delete}}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeFalse())
		})

		It("should not allow the delete of the hcoNamespace if failed to get Hyperconverged CR", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{cr})

			cli.InitiateGetErrors(func(key client.ObjectKey) error {
				if key.Name == util.HyperConvergedName {
					return ErrFakeHcoError
				}
				return nil
			})

			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: newRequest(admissionv1.Delete, ns, corev1Codec)}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeFalse())
		})

		It("should ignore other namespaces even if Hyperconverged CR exists", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{cr})
			otherNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ResourceInvalidNamespace,
				},
			}

			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: newRequest(admissionv1.Delete, otherNs, corev1Codec)}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())
		})

		It("should allow other operations", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{cr})
			nsMutator := initMutator(s, cli)
			req := admission.Request{AdmissionRequest: newRequest(admissionv1.Update, ns, corev1Codec)}

			res := nsMutator.Handle(context.TODO(), req)
			Expect(res.Allowed).To(BeTrue())
		})
	})

})

func initMutator(s *runtime.Scheme, testClient client.Client) *NsMutator {
	nsMutator := NewNsMutator(testClient, HcoValidNamespace)

	decoder, err := admission.NewDecoder(s)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	ExpectWithOffset(1, nsMutator.InjectDecoder(decoder)).Should(Succeed())

	return nsMutator
}

func newRequest(operation admissionv1.Operation, object runtime.Object, encoder runtime.Encoder) admissionv1.AdmissionRequest {
	return admissionv1.AdmissionRequest{
		Operation: operation,
		OldObject: runtime.RawExtension{
			Raw:    []byte(runtime.EncodeOrDie(encoder, object)),
			Object: object,
		},
	}
}
