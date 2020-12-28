package webhooks

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	sspopv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
	logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("hyperconverged-resource")
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
		networkaddons.AddToScheme,
		sspopv1.AddToScheme,
		vmimportv1beta1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	Context("Check validating webhook", func() {
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
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
		wh.Init(logger, cli, HcoValidNamespace, true)

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
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirt(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kvUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewCDI(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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
			c := getFakeClient(s, hco)
			cli := errorClient{c, cdiUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should not return error if dry-run update of ALL CR passes", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, noFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(BeNil())
		})

		It("should return error if NetworkAddons CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewNetworkAddons(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of NetworkAddons CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, networkUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeNetworkError))
		})

		It("should return error if KubeVirtCommonTemplateBundle CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirtCommonTemplateBundle(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of KubeVirtCommonTemplateBundle CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kubevirtCommonTemplateBundleUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeCommonTemplateBundleError))

		})

		It("should return error if KubeVirtNodeLabellerBundle CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirtNodeLabellerBundleForCR(hco, hco.Namespace))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of KubeVirtNodeLabellerBundle CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kubevirtNodeLabellerBundleUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeNodeLabellerBundleError))

		})

		It("should return error if KubeVirtTemplateValidator CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirtTemplateValidatorForCR(hco, hco.Namespace))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of KubeVirtTemplateValidator CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kubevirtTemplateValidatorUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeTemplateValidatorError))

		})

		It("should return error if NewKubeVirtMetricsAggregation CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirtMetricsAggregationForCR(hco, hco.Namespace))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of NewKubeVirtMetricsAggregation CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kubevirtMetricsAggregationUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeMetricsAggregationError))
		})

		It("should return error if VMImport CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewVMImportForCR(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

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

		It("should return error if dry-run update of VMImport CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, vmImportUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeVMImportError))
		})

		It("should return error if dry-run update is timeout", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, timeoutError}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(context.DeadlineExceeded))
		})

		// plain-k8s tests
		It("should not return error in plain-k8s if KubeVirtCommonTemplateBundle CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirtCommonTemplateBundle(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, false)

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
			Expect(err).To(BeNil())
		})

		It("should return error in plain-k8s if KV CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewKubeVirt(hco))).To(BeNil())
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, false)

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

		It("should return error in plain-k8s if dry-run update of VMImport CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, vmImportUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeVMImportError))
		})

		It("should not return error if dry-run update of NewKubeVirtMetricsAggregation CR returns error", func() {
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
			c := getFakeClient(s, hco)
			cli := errorClient{c, kubevirtMetricsAggregationUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, false)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(BeNil())
		})

	})

	Context("Check mutating webhook for namespace deletion", func() {
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
		})

		cr := &v1beta1.HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ResourceName,
				Namespace: HcoValidNamespace,
			},
			Spec: v1beta1.HyperConvergedSpec{},
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: HcoValidNamespace,
			},
		}

		It("should allow the delete of the namespace if Hyperconverged CR doesn't exist", func() {
			cli := fake.NewFakeClientWithScheme(s)
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			allowed, err := wh.HandleMutatingNsDelete(ns, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeTrue())
		})

		It("should not allow the delete of the namespace if Hyperconverged CR exists", func() {
			cli := fake.NewFakeClientWithScheme(s, cr)
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			allowed, err := wh.HandleMutatingNsDelete(ns, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})

		It("should ignore other namespaces even if Hyperconverged CR exists", func() {
			cli := fake.NewFakeClientWithScheme(s, cr)
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			other_ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ResourceInvalidNamespace,
				},
			}

			allowed, err := wh.HandleMutatingNsDelete(other_ns, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeTrue())
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

func getFakeClient(s *runtime.Scheme, hco *v1beta1.HyperConverged) client.Client {
	return fake.NewFakeClientWithScheme(
		s,
		hco,
		operands.NewKubeVirt(hco),
		operands.NewCDI(hco),
		operands.NewNetworkAddons(hco),
		operands.NewKubeVirtCommonTemplateBundle(hco),
		operands.NewKubeVirtNodeLabellerBundleForCR(hco, hco.Namespace),
		operands.NewKubeVirtTemplateValidatorForCR(hco, hco.Namespace),
		operands.NewKubeVirtMetricsAggregationForCR(hco, hco.Namespace),
		operands.NewVMImportForCR(hco))
}

type fakeFailure int

const (
	noFailure fakeFailure = iota
	kvUpdateFailure
	cdiUpdateFailure
	networkUpdateFailure
	kubevirtCommonTemplateBundleUpdateFailure
	kubevirtNodeLabellerBundleUpdateFailure
	kubevirtTemplateValidatorUpdateFailure
	kubevirtMetricsAggregationUpdateFailure
	vmImportUpdateFailure
	timeoutError
)

type errorClient struct {
	client.Client
	failure fakeFailure
}

var (
	ErrFakeKvError                   = errors.New("fake KubeVirt error")
	ErrFakeCdiError                  = errors.New("fake CDI error")
	ErrFakeNetworkError              = errors.New("fake Network error")
	ErrFakeCommonTemplateBundleError = errors.New("fake CommonTemplateBundle error")
	ErrFakeNodeLabellerBundleError   = errors.New("fake NodeLabellerBundle error")
	ErrFakeTemplateValidatorError    = errors.New("fake TemplateValidator error")
	ErrFakeMetricsAggregationError   = errors.New("fake MetricsAggregation error")
	ErrFakeVMImportError             = errors.New("fake VMImport error")
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
	case *networkaddonsv1.NetworkAddonsConfig:
		if ec.failure == networkUpdateFailure {
			return ErrFakeNetworkError
		}
	case *sspv1.KubevirtCommonTemplatesBundle:
		if ec.failure == kubevirtCommonTemplateBundleUpdateFailure {
			return ErrFakeCommonTemplateBundleError
		}
	case *sspv1.KubevirtNodeLabellerBundle:
		if ec.failure == kubevirtNodeLabellerBundleUpdateFailure {
			return ErrFakeNodeLabellerBundleError
		}
	case *sspv1.KubevirtTemplateValidator:
		if ec.failure == kubevirtTemplateValidatorUpdateFailure {
			return ErrFakeTemplateValidatorError
		}
	case *sspv1.KubevirtMetricsAggregation:
		if ec.failure == kubevirtMetricsAggregationUpdateFailure {
			return ErrFakeMetricsAggregationError
		}
	case *vmimportv1beta1.VMImportConfig:
		if ec.failure == vmImportUpdateFailure {
			return ErrFakeVMImportError
		}
	}

	if ec.failure == timeoutError {
		// timeout + 100 ms
		time.Sleep(updateDryRunTimeOut + time.Millisecond*100)
	}

	return ec.Client.Update(ctx, obj, opts...)
}
