package webhooks

import (
	"context"
	"errors"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	ResourceName             = "kubevirt-hyperconverged"
	ResourceInvalidNamespace = "an-arbitrary-namespace"
	HcoValidNamespace        = "kubevirt-hyperconverged"
)

var (
	logger = logf.Log.WithName("hyperconverged-resource")
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhooks Suite")
}

var _ = Describe("webhooks handler", func() {
	s := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		v1beta1.AddToScheme,
		cdiv1beta1.AddToScheme,
		kubevirtv1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	Context("Check validating webhook", func() {
		BeforeEach(func() {
			os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)
		})

		cr := &v1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ResourceName,
				Namespace: HcoValidNamespace,
			},
			Spec: v1beta1.HyperConvergedSpec{},
		}

		cli := fake.NewFakeClientWithScheme(s)
		wh := &WebhookHandler{}
		wh.Init(logger, cli)

		It("should accept creation of a resource with a valid namespace", func() {
			err := wh.ValidateCreate(cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an arbitrary namespace", func() {
			cr.ObjectMeta.Namespace = ResourceInvalidNamespace
			err := wh.ValidateCreate(cr)
			Expect(err).To(HaveOccurred())
		})

		// TODO: add tests for update validation with existing workload
	})

	Context("validate update webhook", func() {

		It("should return error if KV CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			// replace the real client with a mock
			cli := fake.NewFakeClientWithScheme(s, hco, operands.NewCDI(hco))
			wh := &WebhookHandler{}
			wh.Init(logger, cli)

			newHco := &v1beta1.HyperConverged{
				Spec: v1beta1.HyperConvergedSpec{
					Infra: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
					Workloads: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
				},
			}

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("should return error if dry-run update of KV CR returns error", func() {
			hco := &v1beta1.HyperConverged{
				Spec: v1beta1.HyperConvergedSpec{
					Infra: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
					Workloads: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
				},
			}
			// replace the real client with a mock
			c := fake.NewFakeClientWithScheme(s, hco, operands.NewKubeVirt(hco), operands.NewCDI(hco))
			cli := errorClient{c, kvUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should return error if CDI CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			// replace the real client with a mock
			cli := fake.NewFakeClientWithScheme(s, hco, operands.NewKubeVirt(hco))
			wh := &WebhookHandler{}
			wh.Init(logger, cli)

			newHco := &v1beta1.HyperConverged{
				Spec: v1beta1.HyperConvergedSpec{
					Infra: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
					Workloads: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
				},
			}

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("should return error if dry-run update of CDI CR returns error", func() {
			hco := &v1beta1.HyperConverged{
				Spec: v1beta1.HyperConvergedSpec{
					Infra: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
					Workloads: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
				},
			}
			// replace the real client with a mock
			c := fake.NewFakeClientWithScheme(s, hco, operands.NewKubeVirt(hco), operands.NewCDI(hco))
			cli := errorClient{c, cdiUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should not return error if dry-run update of CDI CR passes", func() {
			hco := &v1beta1.HyperConverged{
				Spec: v1beta1.HyperConvergedSpec{
					Infra: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
					Workloads: v1beta1.HyperConvergedConfig{
						NodePlacement: newHyperConvergedConfig(),
					},
				},
			}
			// replace the real client with a mock
			c := fake.NewFakeClientWithScheme(s, hco, operands.NewKubeVirt(hco), operands.NewCDI(hco))
			cli := errorClient{c, noFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(BeNil())
		})

	})
})

func newHyperConvergedConfig() *sdkapi.NodePlacement {
	seconds1, seconds2 := int64(1), int64(2)
	return &sdkapi.NodePlacement{
		NodeSelector: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
								{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
							},
							MatchFields: []corev1.NodeSelectorRequirement{
								{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
								{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: &seconds1},
			{Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: &seconds2},
		},
	}
}

type fakeFailure int

const (
	noFailure fakeFailure = iota
	kvUpdateFailure
	cdiUpdateFailure
)

type errorClient struct {
	client.Client
	failure fakeFailure
}

var (
	ErrFakeKvError  = errors.New("fake KubeVirt error")
	ErrFakeCdiError = errors.New("fake CDI error")
)

func (ec errorClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	switch obj.(type) {
	case *kubevirtv1.KubeVirt:
		if ec.failure == kvUpdateFailure {
			return ErrFakeKvError
		}
	case *cdiv1beta1.CDI:
		if ec.failure == cdiUpdateFailure {
			return ErrFakeCdiError
		}
	}

	return ec.Client.Update(ctx, obj, opts...)
}
