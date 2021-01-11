package webhooks

import (
	"context"
	"errors"
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"os"
	"testing"
	"time"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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

const (
	validKvAnnotation = `[
					{
						"op": "add",
						"path": "/spec/configuration/cpuRequest",
						"value": "12m"
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration",
						"value": {"featureGates": ["fg1"]}
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration/featureGates/-",
						"value": "fg2"
					}
			]`
	validCdiAnnotation = `[
				{
					"op": "add",
					"path": "/spec/config/featureGates/-",
					"value": "fg1"
				},
				{
					"op": "add",
					"path": "/spec/config/filesystemOverhead",
					"value": {"global": "50", "storageClass": {"AAA": "75", "BBB": "25"}}
				}
			]`
	validCnaAnnotation = `[
					{
						"op": "add",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					},
					{
						"op": "add",
						"path": "/spec/imagePullPolicy",
						"value": "Always"
					}
			]`
	invalidKvAnnotation  = `[{"op": "wrongOp", "path": "/spec/configuration/cpuRequest", "value": "12m"}]`
	invalidCdiAnnotation = `[{"op": "wrongOp", "path": "/spec/config/featureGates/-", "value": "fg1"}]`
	invalidCnaAnnotation = `[{"op": "wrongOp", "path": "/spec/kubeMacPool", "value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }}]`
)

var _ = Describe("webhooks handler", func() {
	s := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		v1beta1.AddToScheme,
		cdiv1beta1.AddToScheme,
		kubevirtv1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
		vmimportv1beta1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	Context("Check create validation webhook", func() {
		var cr *v1beta1.HyperConverged
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
			cr = &v1beta1.HyperConverged{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ResourceName,
					Namespace: HcoValidNamespace,
				},
				Spec: v1beta1.HyperConvergedSpec{},
			}
		})

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

		It("should accept creation of a resource with a valid kv annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchKVAnnotationName: validKvAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid kv annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchKVAnnotationName: invalidKvAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).To(HaveOccurred())
		})

		It("should accept creation of a resource with a valid cdi annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: validCdiAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid cdi annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: invalidCdiAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).To(HaveOccurred())
		})

		It("should accept creation of a resource with a valid cna annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: validCnaAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid cna annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: invalidCnaAnnotation}
			err := wh.ValidateCreate(cr)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("validate update validation webhook", func() {

		It("should return error if KV CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			kv, err := operands.NewKubeVirt(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, kv)).To(BeNil())
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

			err = wh.ValidateUpdate(newHco, hco)
			Expect(err).To(HaveOccurred())
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
			cdi, err := operands.NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cdi)).To(BeNil())
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

			err = wh.ValidateUpdate(newHco, hco)
			Expect(err).To(HaveOccurred())
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
			cna, err := operands.NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cna)).To(BeNil())
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

			err = wh.ValidateUpdate(newHco, hco)
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

		It("should return error if SSP CR is missing", func() {
			hco := &v1beta1.HyperConverged{}
			ctx := context.TODO()
			cli := getFakeClient(s, hco)
			Expect(cli.Delete(ctx, operands.NewSSP(hco))).To(BeNil())
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

		It("should return error if dry-run update of SSP CR returns error", func() {
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
			cli := errorClient{c, sspUpdateFailure}
			wh := &WebhookHandler{}
			wh.Init(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeSspError))

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

		Context("plain-k8s tests", func() {
			It("should return error in plain-k8s if KV CR is missing", func() {
				hco := &v1beta1.HyperConverged{}
				ctx := context.TODO()
				cli := getFakeClient(s, hco)
				kv, err := operands.NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cli.Delete(ctx, kv)).To(BeNil())
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

				err = wh.ValidateUpdate(newHco, hco)
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
		})
	})

	Context("unsupported annotation", func() {
		var hco *v1beta1.HyperConverged
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
			hco = &v1beta1.HyperConverged{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ResourceName,
					Namespace: HcoValidNamespace,
				},
				Spec: v1beta1.HyperConvergedSpec{},
			}
		})

		DescribeTable("should accept if annotation is valid",
			func(annotationName, annotation string) {
				cli := getFakeClient(s, hco)
				wh := &WebhookHandler{}
				wh.Init(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				hco.Annotations = map[string]string{annotationName: annotation}

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("should accept if kv annotation is valid", common.JSONPatchKVAnnotationName, validKvAnnotation),
			Entry("should accept if cdi annotation is valid", common.JSONPatchCDIAnnotationName, validCdiAnnotation),
			Entry("should accept if cna annotation is valid", common.JSONPatchCNAOAnnotationName, validCnaAnnotation),
		)

		DescribeTable("should reject if annotation is invalid",
			func(annotationName, annotation string) {
				c := getFakeClient(s, hco)
				cli := errorClient{c, timeoutError}
				wh := &WebhookHandler{}
				wh.Init(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				newHco.Annotations = map[string]string{annotationName: annotation}

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid jsonPatch in the %s", annotationName))
				fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)

			},
			Entry("should reject if kv annotation is invalid", common.JSONPatchKVAnnotationName, invalidKvAnnotation),
			Entry("should reject if cdi annotation is invalid", common.JSONPatchCDIAnnotationName, invalidCdiAnnotation),
			Entry("should reject if cna annotation is invalid", common.JSONPatchCNAOAnnotationName, invalidCnaAnnotation),
		)
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

			otherNs := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ResourceInvalidNamespace,
				},
			}

			allowed, err := wh.HandleMutatingNsDelete(otherNs, false)
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
	kv, err := operands.NewKubeVirt(hco)
	Expect(err).ToNot(HaveOccurred())

	cdi, err := operands.NewCDI(hco)
	Expect(err).ToNot(HaveOccurred())

	cna, err := operands.NewNetworkAddons(hco)
	Expect(err).ToNot(HaveOccurred())

	return fake.NewFakeClientWithScheme(
		s,
		hco,
		kv,
		cdi,
		cna,
		operands.NewSSP(hco),
		operands.NewVMImportForCR(hco))
}

type fakeFailure int

const (
	noFailure fakeFailure = iota
	kvUpdateFailure
	cdiUpdateFailure
	networkUpdateFailure
	sspUpdateFailure
	vmImportUpdateFailure
	timeoutError
)

type errorClient struct {
	client.Client
	failure fakeFailure
}

var (
	ErrFakeKvError       = errors.New("fake KubeVirt error")
	ErrFakeCdiError      = errors.New("fake CDI error")
	ErrFakeNetworkError  = errors.New("fake Network error")
	ErrFakeSspError      = errors.New("fake SSP error")
	ErrFakeVMImportError = errors.New("fake VMImport error")
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
	case *sspv1beta1.SSP:
		if ec.failure == sspUpdateFailure {
			return ErrFakeSspError
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
