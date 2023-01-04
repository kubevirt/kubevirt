package validator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
)

const (
	ResourceInvalidNamespace = "an-arbitrary-namespace"
	HcoValidNamespace        = "kubevirt-hyperconverged"
)

var (
	logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("hyperconverged-resource")
)

func TestValidatorWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator Webhooks Suite")
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

var _ = Describe("webhooks validator", func() {
	s := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		v1beta1.AddToScheme,
		cdiv1beta1.AddToScheme,
		kubevirtcorev1.AddToScheme,
		networkaddonsv1.AddToScheme,
		sspv1beta1.AddToScheme,
	} {
		Expect(f(s)).ToNot(HaveOccurred())
	}

	codecFactory := serializer.NewCodecFactory(s)
	v1beta1Codec := codecFactory.LegacyCodec(v1beta1.SchemeGroupVersion)

	cli := fake.NewClientBuilder().WithScheme(s).Build()
	wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

	decoder, err := admission.NewDecoder(s)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	err = wh.InjectDecoder(decoder)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	Context("Check create validation webhook", func() {
		var cr *v1beta1.HyperConverged
		var dryRun bool
		var ctx context.Context
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).ToNot(HaveOccurred())
			cr = commonTestUtils.NewHco()
			dryRun = false
			ctx = context.TODO()
		})

		It("should correctly handle a valid creation request", func() {
			req := newRequest(admissionv1.Create, cr, v1beta1Codec, false)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should correctly handle a valid dryrun creation request", func() {
			req := newRequest(admissionv1.Create, cr, v1beta1Codec, true)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should reject malformed creation requests", func() {
			req := newRequest(admissionv1.Create, cr, v1beta1Codec, false)
			req.OldObject = req.Object
			req.Object = runtime.RawExtension{}

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(400)))
			Expect(res.Result.Message).To(Equal("there is no content to decode"))

			req = newRequest(admissionv1.Create, cr, v1beta1Codec, false)
			req.Operation = "MALFORMED"

			res = wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(400)))
			Expect(res.Result.Message).To(Equal("unknown operation request \"MALFORMED\""))
		})

		It("should correctly handle operation errors", func() {
			cr.Namespace = ResourceInvalidNamespace
			req := newRequest(admissionv1.Create, cr, v1beta1Codec, false)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(403)))
			Expect(res.Result.Reason).To(BeEquivalentTo("invalid namespace for v1beta1.HyperConverged - please use the kubevirt-hyperconverged namespace"))
		})

		It("should accept creation of a resource with a valid namespace", func() {
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an arbitrary namespace", func() {
			cr.ObjectMeta.Namespace = ResourceInvalidNamespace
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).To(HaveOccurred())
		})

		It("should accept creation of a resource with a valid kv annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchKVAnnotationName: validKvAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid kv annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchKVAnnotationName: invalidKvAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).To(HaveOccurred())
		})

		It("should accept creation of a resource with a valid cdi annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: validCdiAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid cdi annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: invalidCdiAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).To(HaveOccurred())
		})

		It("should accept creation of a resource with a valid cna annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: validCnaAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an invalid cna annotation", func() {
			cr.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: invalidCnaAnnotation}
			err := wh.ValidateCreate(ctx, dryRun, cr)
			Expect(err).To(HaveOccurred())
		})

		Context("test permitted host devices validation", func() {
			It("should allow unique PCI Host Device", func() {
				cr.Spec.PermittedHostDevices = &v1beta1.PermittedHostDevices{
					PciHostDevices: []v1beta1.PciHostDevice{
						{
							PCIDeviceSelector: "111",
							ResourceName:      "name",
						},
						{
							PCIDeviceSelector: "222",
							ResourceName:      "name",
						},
						{
							PCIDeviceSelector: "333",
							ResourceName:      "name",
						},
					},
				}
				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow unique Mediate Host Device", func() {
				cr.Spec.PermittedHostDevices = &v1beta1.PermittedHostDevices{
					MediatedDevices: []v1beta1.MediatedHostDevice{
						{
							MDEVNameSelector: "111",
							ResourceName:     "name",
						},
						{
							MDEVNameSelector: "222",
							ResourceName:     "name",
						},
						{
							MDEVNameSelector: "333",
							ResourceName:     "name",
						},
					},
				}
				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Test DataImportCronTemplates", func() {
			var image1, image2, image3, image4 v1beta1.DataImportCronTemplate

			var dryRun bool
			var ctx context.Context

			BeforeEach(func() {
				dryRun = false
				ctx = context.TODO()

				image1 = v1beta1.DataImportCronTemplate{
					ObjectMeta: metav1.ObjectMeta{Name: "image1"},
					Spec: &cdiv1beta1.DataImportCronSpec{
						Schedule: "1 */12 * * *",
						Template: cdiv1beta1.DataVolume{
							Spec: cdiv1beta1.DataVolumeSpec{
								Source: &cdiv1beta1.DataVolumeSource{
									Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: pointer.String("docker://someregistry/image1")},
								},
							},
						},
						ManagedDataSource: "image1",
					},
				}

				image2 = v1beta1.DataImportCronTemplate{
					ObjectMeta: metav1.ObjectMeta{Name: "image2"},
					Spec: &cdiv1beta1.DataImportCronSpec{
						Schedule: "2 */12 * * *",
						Template: cdiv1beta1.DataVolume{
							Spec: cdiv1beta1.DataVolumeSpec{
								Source: &cdiv1beta1.DataVolumeSource{
									Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: pointer.String("docker://someregistry/image2")},
								},
							},
						},
						ManagedDataSource: "image2",
					},
				}

				image3 = v1beta1.DataImportCronTemplate{
					ObjectMeta: metav1.ObjectMeta{Name: "image3"},
					Spec: &cdiv1beta1.DataImportCronSpec{
						Schedule: "3 */12 * * *",
						Template: cdiv1beta1.DataVolume{
							Spec: cdiv1beta1.DataVolumeSpec{
								Source: &cdiv1beta1.DataVolumeSource{
									Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: pointer.String("docker://someregistry/image3")},
								},
							},
						},
						ManagedDataSource: "image3",
					},
				}

				image4 = v1beta1.DataImportCronTemplate{
					ObjectMeta: metav1.ObjectMeta{Name: "image4"},
					Spec: &cdiv1beta1.DataImportCronSpec{
						Schedule: "4 */12 * * *",
						Template: cdiv1beta1.DataVolume{
							Spec: cdiv1beta1.DataVolumeSpec{
								Source: &cdiv1beta1.DataVolumeSource{
									Registry: &cdiv1beta1.DataVolumeSourceRegistry{URL: pointer.String("docker://someregistry/image4")},
								},
							},
						},
						ManagedDataSource: "image4",
					},
				}

				cr.Spec.DataImportCronTemplates = []v1beta1.DataImportCronTemplate{image1, image2, image3, image4}
			})

			It("should allow setting the annotation to true", func() {
				cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "true"}
				cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "TRUE"}
				cr.Spec.DataImportCronTemplates[2].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "TrUe"}
				cr.Spec.DataImportCronTemplates[3].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "tRuE"}

				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow setting the annotation to false", func() {
				cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "false"}
				cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "FALSE"}
				cr.Spec.DataImportCronTemplates[2].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "FaLsE"}
				cr.Spec.DataImportCronTemplates[3].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "fAlSe"}

				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow setting no annotation", func() {
				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not allow empty annotation", func() {
				cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: ""}
				cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: ""}

				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).To(HaveOccurred())
			})

			It("should not allow unknown annotation values", func() {
				cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "wrong"}
				cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "mistake"}

				err := wh.ValidateCreate(ctx, dryRun, cr)
				Expect(err).To(HaveOccurred())
			})

			Context("Empty DICT spec", func() {
				It("don't allow if the annotation does not exist", func() {
					// empty annotation map
					cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{}
					cr.Spec.DataImportCronTemplates[0].Spec = nil
					// no annotation map
					cr.Spec.DataImportCronTemplates[1].Spec = nil

					err := wh.ValidateCreate(ctx, dryRun, cr)
					Expect(err).To(HaveOccurred())
				})

				It("don't allow if the annotation is true", func() {
					cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "True"}
					cr.Spec.DataImportCronTemplates[0].Spec = nil
					cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "true"}
					cr.Spec.DataImportCronTemplates[1].Spec = nil

					err := wh.ValidateCreate(ctx, dryRun, cr)
					Expect(err).To(HaveOccurred())
				})

				It("allow if the annotation is false", func() {
					cr.Spec.DataImportCronTemplates[0].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "False"}
					cr.Spec.DataImportCronTemplates[0].Spec = nil
					cr.Spec.DataImportCronTemplates[1].Annotations = map[string]string{util.DataImportCronEnabledAnnotation: "false"}
					cr.Spec.DataImportCronTemplates[1].Spec = nil

					err := wh.ValidateCreate(ctx, dryRun, cr)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("validate tlsSecurityProfiles", func() {
			var dryRun bool
			var ctx context.Context

			BeforeEach(func() {
				dryRun = false
				ctx = context.TODO()
			})

			updateTlsSecurityProfile := func(minTLSVersion openshiftconfigv1.TLSProtocolVersion, ciphers []string) error {
				cr.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							MinTLSVersion: minTLSVersion,
							Ciphers:       ciphers,
						},
					},
				}

				return wh.ValidateCreate(ctx, dryRun, cr)
			}

			DescribeTable("should succeed if has any of the HTTP/2-required ciphers",
				func(cipher string) {
					err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS12, []string{"DHE-RSA-AES256-GCM-SHA384", cipher, "DHE-RSA-CHACHA20-POLY1305"})
					Expect(err).ToNot(HaveOccurred())
				},
				Entry("ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES128-GCM-SHA256"),
				Entry("ECDHE-ECDSA-AES128-GCM-SHA256", "ECDHE-ECDSA-AES128-GCM-SHA256"),
			)

			It("should fail if does not have any of the HTTP/2-required ciphers", func() {
				err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS12, []string{"DHE-RSA-AES256-GCM-SHA384", "DHE-RSA-CHACHA20-POLY1305"})
				Expect(err).To(HaveOccurred())
			})

			It("should succeed if does not have any of the HTTP/2-required ciphers but TLS version >= 1.3", func() {
				err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS13, []string{"DHE-RSA-AES256-GCM-SHA384", "DHE-RSA-CHACHA20-POLY1305"})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("validate update validation webhook", func() {

		var hco *v1beta1.HyperConverged
		var dryRun bool
		var ctx context.Context

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			hco.Spec.Infra = v1beta1.HyperConvergedConfig{
				NodePlacement: newHyperConvergedConfig(),
			}
			hco.Spec.Workloads = v1beta1.HyperConvergedConfig{
				NodePlacement: newHyperConvergedConfig(),
			}
			dryRun = false
			ctx = context.TODO()
		})

		It("should correctly handle a valid update request", func() {
			req := newRequest(admissionv1.Update, hco, v1beta1Codec, false)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should correctly handle a valid dryrun update request", func() {
			req := newRequest(admissionv1.Update, hco, v1beta1Codec, true)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should reject malformed update requests", func() {
			req := newRequest(admissionv1.Update, hco, v1beta1Codec, false)
			req.Object = runtime.RawExtension{}

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(400)))
			Expect(res.Result.Message).To(Equal("there is no content to decode"))

			req = newRequest(admissionv1.Update, hco, v1beta1Codec, false)
			req.OldObject = runtime.RawExtension{}

			res = wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(400)))
			Expect(res.Result.Message).To(Equal("there is no content to decode"))
		})

		It("should return error if KV CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)

			kv := operands.NewKubeVirtWithNameOnly(hco)
			Expect(cli.Delete(ctx, kv)).ToNot(HaveOccurred())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("kubevirts.kubevirt.io"))
		})

		It("should return error if dry-run update of KV CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(kvUpdateFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should return error if CDI CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)
			cdi, err := operands.NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cdi)).ToNot(HaveOccurred())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err = wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("cdis.cdi.kubevirt.io"))
		})

		It("should return error if dry-run update of CDI CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(cdiUpdateFailure))
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should not return error if dry-run update of ALL CR passes", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(noFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error if NetworkAddons CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)
			cna, err := operands.NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cna)).ToNot(HaveOccurred())
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err = wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io"))
		})

		It("should return error if dry-run update of NetworkAddons CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(networkUpdateFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeNetworkError))
		})

		It("should return error if SSP CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)

			Expect(cli.Delete(ctx, operands.NewSSPWithNameOnly(hco))).ToNot(HaveOccurred())
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("ssps.ssp.kubevirt.io"))
		})

		It("should return error if dry-run update of SSP CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(sspUpdateFailure))
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeSspError))

		})

		It("should return error if dry-run update is timeout", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(initiateTimeout)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(context.DeadlineExceeded))
		})

		It("should not return error if nothing was changed", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(initiateTimeout)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)

			Expect(wh.ValidateUpdate(ctx, dryRun, newHco, hco)).ToNot(HaveOccurred())

		})

		Context("test permitted host devices update validation", func() {
			It("should allow unique PCI Host Device", func() {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				newHco.Spec.PermittedHostDevices = &v1beta1.PermittedHostDevices{
					PciHostDevices: []v1beta1.PciHostDevice{
						{
							PCIDeviceSelector: "111",
							ResourceName:      "name",
						},
						{
							PCIDeviceSelector: "222",
							ResourceName:      "name",
						},
						{
							PCIDeviceSelector: "333",
							ResourceName:      "name",
						},
					},
				}
				Expect(wh.ValidateUpdate(ctx, dryRun, newHco, hco)).ToNot(HaveOccurred())
			})

			It("should allow unique Mediate Host Device", func() {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				newHco.Spec.PermittedHostDevices = &v1beta1.PermittedHostDevices{
					MediatedDevices: []v1beta1.MediatedHostDevice{
						{
							MDEVNameSelector: "111",
							ResourceName:     "name",
						},
						{
							MDEVNameSelector: "222",
							ResourceName:     "name",
						},
						{
							MDEVNameSelector: "333",
							ResourceName:     "name",
						},
					},
				}
				Expect(wh.ValidateUpdate(ctx, dryRun, newHco, hco)).ToNot(HaveOccurred())
			})
		})

		Context("plain-k8s tests", func() {
			It("should return error in plain-k8s if KV CR is missing", func() {
				hco := &v1beta1.HyperConverged{}
				ctx := context.TODO()
				cli := getFakeClient(hco)
				kv, err := operands.NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cli.Delete(ctx, kv)).ToNot(HaveOccurred())
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, false, nil)

				newHco := commonTestUtils.NewHco()
				newHco.Spec.Infra = v1beta1.HyperConvergedConfig{
					NodePlacement: newHyperConvergedConfig(),
				}
				newHco.Spec.Workloads = v1beta1.HyperConvergedConfig{
					NodePlacement: newHyperConvergedConfig(),
				}

				err = wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("Check LiveMigrationConfiguration", func() {
			var hco *v1beta1.HyperConverged

			BeforeEach(func() {
				hco = commonTestUtils.NewHco()
			})

			It("should ignore if there is no change in live migration", func() {
				cli := getFakeClient(hco)

				// Deleting KV here, in order to make sure the that the webhook does not find differences,
				// and so it exits with no error before finding that KV is not there.
				// Later we'll check that there is no error from the webhook, and that will prove that
				// the comparison works.
				kv := operands.NewKubeVirtWithNameOnly(hco)
				Expect(cli.Delete(context.TODO(), kv)).ToNot(HaveOccurred())

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow updating of live migration", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the LiveMigrationConfig field
				newVal := int64(200)
				hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &newVal

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail if live migration is wrong", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the LiveMigrationConfig field
				wrongVal := "Wrong Value"
				newHco.Spec.LiveMigrationConfig.BandwidthPerMigration = &wrongVal

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to parse the LiveMigrationConfig.bandwidthPerMigration field"))
			})
		})

		Context("Check CertRotation", func() {
			var hco *v1beta1.HyperConverged

			BeforeEach(func() {
				hco = commonTestUtils.NewHco()
			})

			It("should ignore if there is no change in cert config", func() {
				cli := getFakeClient(hco)

				// Deleting KV here, in order to make sure the that the webhook does not find differences,
				// and so it exits with no error before finding that KV is not there.
				// Later we'll check that there is no error from the webhook, and that will prove that
				// the comparison works.
				kv := operands.NewKubeVirtWithNameOnly(hco)
				Expect(cli.Delete(context.TODO(), kv)).ToNot(HaveOccurred())

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow updating of cert config", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the CertConfig fields
				newHco.Spec.CertConfig.CA.Duration.Duration = hco.Spec.CertConfig.CA.Duration.Duration * 2
				newHco.Spec.CertConfig.CA.RenewBefore.Duration = hco.Spec.CertConfig.CA.RenewBefore.Duration * 2
				newHco.Spec.CertConfig.Server.Duration.Duration = hco.Spec.CertConfig.Server.Duration.Duration * 2
				newHco.Spec.CertConfig.Server.RenewBefore.Duration = hco.Spec.CertConfig.Server.RenewBefore.Duration * 2

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("should fail if cert config is wrong",
				func(newHco v1beta1.HyperConverged, errorMsg string) {
					cli := getFakeClient(hco)

					wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

					err := wh.ValidateUpdate(ctx, dryRun, &newHco, hco)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring(errorMsg))
				},
				Entry("certConfig.ca.duration is too short",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 8 * time.Minute},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig.ca.duration: value is too small"),
				Entry("certConfig.ca.renewBefore is too short",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 8 * time.Minute},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig.ca.renewBefore: value is too small"),
				Entry("certConfig.server.duration is too short",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 8 * time.Minute},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig.server.duration: value is too small"),
				Entry("certConfig.server.renewBefore is too short",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 8 * time.Minute},
								},
							},
						},
					},
					"spec.certConfig.server.renewBefore: value is too small"),
				Entry("ca: duration is smaller than renewBefore",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 23 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig.ca: duration is smaller than renewBefore"),
				Entry("server: duration is smaller than renewBefore",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 11 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig.server: duration is smaller than renewBefore"),
				Entry("ca.duration is smaller than server.duration",
					v1beta1.HyperConverged{
						ObjectMeta: metav1.ObjectMeta{
							Name:      util.HyperConvergedName,
							Namespace: HcoValidNamespace,
						},
						Spec: v1beta1.HyperConvergedSpec{
							CertConfig: v1beta1.HyperConvergedCertConfig{
								CA: v1beta1.CertRotateConfigCA{
									Duration:    &metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    &metav1.Duration{Duration: 96 * time.Hour},
									RenewBefore: &metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig: ca.duration is smaller than server.duration"),
			)

		})

		Context("validate tlsSecurityProfiles", func() {
			var hco *v1beta1.HyperConverged

			BeforeEach(func() {
				hco = commonTestUtils.NewHco()
			})

			updateTlsSecurityProfile := func(minTLSVersion openshiftconfigv1.TLSProtocolVersion, ciphers []string) error {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				newHco.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							MinTLSVersion: minTLSVersion,
							Ciphers:       ciphers,
						},
					},
				}

				return wh.ValidateUpdate(ctx, dryRun, newHco, hco)
			}

			DescribeTable("should succeed if has any of the HTTP/2-required ciphers",
				func(cipher string) {
					err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS12, []string{"DHE-RSA-AES256-GCM-SHA384", cipher, "DHE-RSA-CHACHA20-POLY1305"})
					Expect(err).ToNot(HaveOccurred())
				},
				Entry("ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES128-GCM-SHA256"),
				Entry("ECDHE-ECDSA-AES128-GCM-SHA256", "ECDHE-ECDSA-AES128-GCM-SHA256"),
			)

			It("should fail if does not have any of the HTTP/2-required ciphers", func() {
				err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS12, []string{"DHE-RSA-AES256-GCM-SHA384", "DHE-RSA-CHACHA20-POLY1305"})
				Expect(err).To(HaveOccurred())
			})

			It("should succeed if does not have any of the HTTP/2-required ciphers but TLS version >= 1.3", func() {
				err := updateTlsSecurityProfile(openshiftconfigv1.VersionTLS13, []string{"DHE-RSA-AES256-GCM-SHA384", "DHE-RSA-CHACHA20-POLY1305"})
				Expect(err).ToNot(HaveOccurred())
			})

		})

	})

	Context("validate delete validation webhook", func() {
		var hco *v1beta1.HyperConverged
		var dryRun bool
		var ctx context.Context

		BeforeEach(func() {
			hco = &v1beta1.HyperConverged{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.HyperConvergedName,
					Namespace: HcoValidNamespace,
				},
			}
			dryRun = false
			ctx = context.TODO()
		})

		It("should correctly handle a valid delete request", func() {
			req := newRequest(admissionv1.Delete, hco, v1beta1Codec, false)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should correctly handle a valid dryrun delete request", func() {
			req := newRequest(admissionv1.Delete, hco, v1beta1Codec, true)

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeTrue())
			Expect(res.Result.Code).To(Equal(int32(200)))
		})

		It("should reject a malformed delete request", func() {
			req := newRequest(admissionv1.Delete, hco, v1beta1Codec, false)
			req.OldObject = req.Object
			req.Object = runtime.RawExtension{}

			res := wh.Handle(ctx, req)
			Expect(res.Allowed).To(BeFalse())
			Expect(res.Result.Code).To(Equal(int32(400)))
			Expect(res.Result.Message).To(Equal("there is no content to decode"))
		})

		It("should validate deletion", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).ToNot(HaveOccurred())

			By("Validate that KV still exists, as it a dry-run deletion")
			kv := operands.NewKubeVirtWithNameOnly(hco)
			err = util.GetRuntimeObject(context.TODO(), cli, kv, logger)
			Expect(err).ToNot(HaveOccurred())

			By("Validate that CDI still exists, as it a dry-run deletion")
			cdi := operands.NewCDIWithNameOnly(hco)
			err = util.GetRuntimeObject(context.TODO(), cli, cdi, logger)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject if KV deletion fails", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "KubeVirt" {
						return ErrFakeKvError
					}
				}
				return nil
			})

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should reject if CDI deletion fails", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "CDI" {
						return ErrFakeCdiError
					}
				}
				return nil
			})

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should ignore if KV does not exist", func() {
			cli := getFakeClient(hco)
			ctx := context.TODO()

			kv := operands.NewKubeVirtWithNameOnly(hco)
			Expect(cli.Delete(ctx, kv)).ToNot(HaveOccurred())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject if getting KV failed for not-not-exists error", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			cli.InitiateGetErrors(func(key client.ObjectKey) error {
				if key.Name == "kubevirt-kubevirt-hyperconverged" {
					return ErrFakeKvError
				}
				return nil
			})

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should ignore if CDI does not exist", func() {
			cli := getFakeClient(hco)
			ctx := context.TODO()

			cdi := operands.NewCDIWithNameOnly(hco)
			Expect(cli.Delete(ctx, cdi)).ToNot(HaveOccurred())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject if getting CDI failed for not-not-exists error", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

			cli.InitiateGetErrors(func(key client.ObjectKey) error {
				if key.Name == "cdi-kubevirt-hyperconverged" {
					return ErrFakeCdiError
				}
				return nil
			})

			err := wh.ValidateDelete(ctx, dryRun, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})
	})

	Context("unsupported annotation", func() {
		var hco *v1beta1.HyperConverged
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).ToNot(HaveOccurred())
			hco = commonTestUtils.NewHco()
		})

		DescribeTable("should accept if annotation is valid",
			func(annotationName, annotation string) {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				dryRun := false
				ctx := context.TODO()

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				hco.Annotations = map[string]string{annotationName: annotation}

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("should accept if kv annotation is valid", common.JSONPatchKVAnnotationName, validKvAnnotation),
			Entry("should accept if cdi annotation is valid", common.JSONPatchCDIAnnotationName, validCdiAnnotation),
			Entry("should accept if cna annotation is valid", common.JSONPatchCNAOAnnotationName, validCnaAnnotation),
		)

		DescribeTable("should reject if annotation is invalid",
			func(annotationName, annotation string) {
				cli := getFakeClient(hco)
				cli.InitiateUpdateErrors(initiateTimeout)

				dryRun := false
				ctx := context.TODO()

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)
				newHco.Annotations = map[string]string{annotationName: annotation}

				err := wh.ValidateUpdate(ctx, dryRun, newHco, hco)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid jsonPatch in the %s", annotationName))
				fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)

			},
			Entry("should reject if kv annotation is invalid", common.JSONPatchKVAnnotationName, invalidKvAnnotation),
			Entry("should reject if cdi annotation is invalid", common.JSONPatchCDIAnnotationName, invalidCdiAnnotation),
			Entry("should reject if cna annotation is invalid", common.JSONPatchCNAOAnnotationName, invalidCnaAnnotation),
		)
	})

	Context("hcoTlsConfigCache", func() {
		var cr *v1beta1.HyperConverged
		var ctx context.Context

		intermediateTLSSecurityProfile := openshiftconfigv1.TLSSecurityProfile{
			Type:         openshiftconfigv1.TLSProfileIntermediateType,
			Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
		}
		initialTLSSecurityProfile := intermediateTLSSecurityProfile
		oldTLSSecurityProfile := openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileOldType,
			Old:  &openshiftconfigv1.OldTLSProfile{},
		}
		modernTLSSecurityProfile := openshiftconfigv1.TLSSecurityProfile{
			Type:   openshiftconfigv1.TLSProfileModernType,
			Modern: &openshiftconfigv1.ModernTLSProfile{},
		}

		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).ToNot(HaveOccurred())
			cr = commonTestUtils.NewHco()
			ctx = context.TODO()
			hcoTlsConfigCache = &initialTLSSecurityProfile
		})

		Context("create", func() {

			It("should update hcoTlsConfigCache creating a resource not in dry run mode", func() {
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
				cr.Spec.TLSSecurityProfile = &modernTLSSecurityProfile
				err := wh.ValidateCreate(ctx, false, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).To(Equal(&modernTLSSecurityProfile))
			})

			It("should not update hcoTlsConfigCache creating a resource in dry run mode", func() {
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
				cr.Spec.TLSSecurityProfile = &modernTLSSecurityProfile
				err := wh.ValidateCreate(ctx, true, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).ToNot(Equal(&modernTLSSecurityProfile))
			})

			It("should not update hcoTlsConfigCache if the create request is refused", func() {
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
				cr.Spec.TLSSecurityProfile = &modernTLSSecurityProfile
				cr.Namespace = ResourceInvalidNamespace
				err := wh.ValidateCreate(ctx, false, cr)
				Expect(err).To(HaveOccurred())
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
			})

		})

		Context("update", func() {

			It("should update hcoTlsConfigCache updating a resource not in dry run mode", func() {
				cli := getFakeClient(cr)
				cli.InitiateUpdateErrors(getUpdateError(noFailure))

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				newCr := &v1beta1.HyperConverged{}
				cr.DeepCopyInto(newCr)
				newCr.Spec.TLSSecurityProfile = &oldTLSSecurityProfile

				err = wh.ValidateUpdate(ctx, false, newCr, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).To(Equal(&oldTLSSecurityProfile))
			})

			It("should not update hcoTlsConfigCache updating a resource in dry run mode", func() {
				cli := getFakeClient(cr)
				cli.InitiateUpdateErrors(getUpdateError(noFailure))

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, &initialTLSSecurityProfile)

				newCr := &v1beta1.HyperConverged{}
				cr.DeepCopyInto(newCr)
				newCr.Spec.TLSSecurityProfile = &oldTLSSecurityProfile

				err = wh.ValidateUpdate(ctx, true, newCr, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
			})

			It("should not update hcoTlsConfigCache if the update request is refused", func() {
				cli := getFakeClient(cr)
				cli.InitiateUpdateErrors(getUpdateError(cdiUpdateFailure))

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, &initialTLSSecurityProfile)

				newCr := &v1beta1.HyperConverged{}
				cr.DeepCopyInto(newCr)
				newCr.Spec.TLSSecurityProfile = &oldTLSSecurityProfile

				err = wh.ValidateUpdate(ctx, false, newCr, cr)
				Expect(err).To(HaveOccurred())
				Expect(err).Should(Equal(ErrFakeCdiError))
				Expect(hcoTlsConfigCache).To(Equal(&initialTLSSecurityProfile))
			})

		})

		Context("delete", func() {

			It("should reset hcoTlsConfigCache deleting a resource not in dry run mode", func() {
				cli := getFakeClient(cr)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				hcoTlsConfigCache = &modernTLSSecurityProfile

				err = wh.ValidateDelete(ctx, false, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).To(BeNil())
			})

			It("should not update hcoTlsConfigCache deleting a resource in dry run mode", func() {
				cli := getFakeClient(cr)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				hcoTlsConfigCache = &modernTLSSecurityProfile

				err = wh.ValidateDelete(ctx, true, cr)
				Expect(err).ToNot(HaveOccurred())
				Expect(hcoTlsConfigCache).To(Equal(&modernTLSSecurityProfile))
			})

			It("should not update hcoTlsConfigCache if the delete request is refused", func() {
				cli := getFakeClient(cr)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, nil)

				hcoTlsConfigCache = &modernTLSSecurityProfile
				cli.InitiateDeleteErrors(func(obj client.Object) error {
					if unstructed, ok := obj.(runtime.Unstructured); ok {
						kind := unstructed.GetObjectKind()
						if kind.GroupVersionKind().Kind == "KubeVirt" {
							return ErrFakeKvError
						}
					}
					return nil
				})

				err = wh.ValidateDelete(ctx, false, cr)
				Expect(err).To(HaveOccurred())
				Expect(err).Should(Equal(ErrFakeKvError))
				Expect(hcoTlsConfigCache).To(Equal(&modernTLSSecurityProfile))
			})

		})

		Context("selectCipherSuitesAndMinTLSVersion", func() {
			const namespace = "kubevirt-hyperconverged"

			var apiServer *openshiftconfigv1.APIServer
			var cl *commonTestUtils.HcoTestClient

			BeforeEach(func() {
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)

				clusterVersion := &openshiftconfigv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Spec: openshiftconfigv1.ClusterVersionSpec{
						ClusterID: "clusterId",
					},
				}
				infrastructure := &openshiftconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: openshiftconfigv1.InfrastructureStatus{
						ControlPlaneTopology:   openshiftconfigv1.HighlyAvailableTopologyMode,
						InfrastructureTopology: openshiftconfigv1.HighlyAvailableTopologyMode,
						PlatformStatus: &openshiftconfigv1.PlatformStatus{
							Type: "mocked",
						},
					},
				}
				ingress := &openshiftconfigv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.IngressSpec{
						Domain: "domain",
					},
				}
				apiServer = &openshiftconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.APIServerSpec{},
				}
				namespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				}
				dns := &openshiftconfigv1.DNS{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.DNSSpec{
						BaseDomain: commonTestUtils.BaseDomain,
					},
				}

				resources := []runtime.Object{clusterVersion, infrastructure, ingress, apiServer, namespace, dns}
				cl = commonTestUtils.InitClient(resources)
			})

			DescribeTable("should consume ApiServer config if HCO one is not explicitly set",
				func(initApiTlsSecurityProfile, initHCOTlsSecurityProfile, midApiTlsSecurityProfile, midHCOTlsSecurityProfile, finApiTlsSecurityProfile, finHCOTlsSecurityProfile *openshiftconfigv1.TLSSecurityProfile, initExpected, midExpected, finExpected openshiftconfigv1.TLSProtocolVersion) {
					apiServer.Spec.TLSSecurityProfile = initApiTlsSecurityProfile
					err = cl.Update(context.TODO(), apiServer)
					Expect(err).ToNot(HaveOccurred())
					err = util.GetClusterInfo().Init(context.TODO(), cl, logger)
					Expect(err).ToNot(HaveOccurred())
					ci := util.GetClusterInfo()
					Expect(ci.IsOpenshift()).To(BeTrue())

					wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true, initHCOTlsSecurityProfile)

					_, minTypedTLSVersion := wh.selectCipherSuitesAndMinTLSVersion()
					Expect(minTypedTLSVersion).Should(Equal(initExpected))

					apiServer.Spec.TLSSecurityProfile = midApiTlsSecurityProfile
					err = cl.Update(context.TODO(), apiServer)
					hcoTlsConfigCache = midHCOTlsSecurityProfile
					err = util.GetClusterInfo().RefreshAPIServerCR(context.TODO(), cl)

					_, minTypedTLSVersion = wh.selectCipherSuitesAndMinTLSVersion()
					Expect(minTypedTLSVersion).Should(Equal(midExpected))

					apiServer.Spec.TLSSecurityProfile = finApiTlsSecurityProfile
					err = cl.Update(context.TODO(), apiServer)
					hcoTlsConfigCache = finHCOTlsSecurityProfile
					err = util.GetClusterInfo().RefreshAPIServerCR(context.TODO(), cl)
					Expect(err).ToNot(HaveOccurred())
					_, minTypedTLSVersion = wh.selectCipherSuitesAndMinTLSVersion()
					Expect(minTypedTLSVersion).Should(Equal(finExpected))
				},
				Entry("nil on APIServer, nil on HCO -> old on API server -> nil on API server",
					nil,
					nil,
					&oldTLSSecurityProfile,
					nil,
					nil,
					nil,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileOldType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
				),
				Entry("nil on APIServer, nil on HCO -> modern on HCO -> nil on HCO",
					nil,
					nil,
					nil,
					&modernTLSSecurityProfile,
					nil,
					nil,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileModernType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
				),
				Entry("old on APIServer, nil on HCO -> intermediate on HCO -> old on API server",
					&oldTLSSecurityProfile,
					nil,
					&oldTLSSecurityProfile,
					&intermediateTLSSecurityProfile,
					&oldTLSSecurityProfile,
					nil,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileOldType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileOldType].MinTLSVersion,
				),
				Entry("old on APIServer, modern on HCO -> intermediate on HCO -> modern on API server, intermediate on HCO",
					&oldTLSSecurityProfile,
					&modernTLSSecurityProfile,
					&oldTLSSecurityProfile,
					&intermediateTLSSecurityProfile,
					&modernTLSSecurityProfile,
					&intermediateTLSSecurityProfile,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileModernType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
					openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
				),
			)

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

func getFakeClient(hco *v1beta1.HyperConverged) *commonTestUtils.HcoTestClient {
	kv, err := operands.NewKubeVirt(hco)
	Expect(err).ToNot(HaveOccurred())

	cdi, err := operands.NewCDI(hco)
	Expect(err).ToNot(HaveOccurred())

	cna, err := operands.NewNetworkAddons(hco)
	Expect(err).ToNot(HaveOccurred())

	ssp, _, err := operands.NewSSP(hco)
	Expect(err).ToNot(HaveOccurred())

	return commonTestUtils.InitClient([]runtime.Object{hco, kv, cdi, cna, ssp})
}

type fakeFailure int

const (
	noFailure fakeFailure = iota
	kvUpdateFailure
	cdiUpdateFailure
	networkUpdateFailure
	sspUpdateFailure
)

var (
	ErrFakeKvError      = errors.New("fake KubeVirt error")
	ErrFakeCdiError     = errors.New("fake CDI error")
	ErrFakeNetworkError = errors.New("fake Network error")
	ErrFakeSspError     = errors.New("fake SSP error")
)

func getUpdateError(failure fakeFailure) commonTestUtils.FakeWriteErrorGenerator {
	switch failure {
	case kvUpdateFailure:
		return func(obj client.Object) error {
			if _, ok := obj.(*kubevirtcorev1.KubeVirt); ok {
				return ErrFakeKvError
			}
			return nil
		}

	case cdiUpdateFailure:
		return func(obj client.Object) error {
			if _, ok := obj.(*cdiv1beta1.CDI); ok {
				return ErrFakeCdiError
			}
			return nil
		}

	case networkUpdateFailure:
		return func(obj client.Object) error {
			if _, ok := obj.(*networkaddonsv1.NetworkAddonsConfig); ok {
				return ErrFakeNetworkError
			}
			return nil
		}

	case sspUpdateFailure:
		return func(obj client.Object) error {
			if _, ok := obj.(*sspv1beta1.SSP); ok {
				return ErrFakeSspError
			}
			return nil
		}
	default:
		return nil
	}
}

func initiateTimeout(_ client.Object) error {
	time.Sleep(updateDryRunTimeOut + time.Millisecond*100)
	return nil
}

func newRequest(operation admissionv1.Operation, cr *v1beta1.HyperConverged, encoder runtime.Encoder, dryrun bool) admission.Request {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			DryRun:    pointer.Bool(dryrun),
			Operation: operation,
			Resource: metav1.GroupVersionResource{
				Group:    v1beta1.SchemeGroupVersion.Group,
				Version:  v1beta1.SchemeGroupVersion.Version,
				Resource: "testresource",
			},
			UID: "test-uid",
		},
	}

	switch operation {
	case admissionv1.Create:
		req.Object = runtime.RawExtension{
			Raw:    []byte(runtime.EncodeOrDie(encoder, cr)),
			Object: cr,
		}
	case admissionv1.Update:
		req.Object = runtime.RawExtension{
			Raw:    []byte(runtime.EncodeOrDie(encoder, cr)),
			Object: cr,
		}
		req.OldObject = runtime.RawExtension{
			Raw:    []byte(runtime.EncodeOrDie(encoder, cr)),
			Object: cr,
		}
	case admissionv1.Delete:
		req.OldObject = runtime.RawExtension{
			Raw:    []byte(runtime.EncodeOrDie(encoder, cr)),
			Object: cr,
		}
	default:
		req.Object = runtime.RawExtension{}
		req.OldObject = runtime.RawExtension{}
	}

	return req
}
