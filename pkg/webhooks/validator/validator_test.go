package validator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
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

var _ = Describe("webhooks validator", func() {
	s := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		v1beta1.AddToScheme,
		cdiv1beta1.AddToScheme,
		kubevirtcorev1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	Context("Check create validation webhook", func() {
		var cr *v1beta1.HyperConverged
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
			cr = commonTestUtils.NewHco()
		})

		cli := fake.NewClientBuilder().WithScheme(s).Build()
		wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

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
				err := wh.ValidateCreate(cr)
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
				err := wh.ValidateCreate(cr)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("validate update validation webhook", func() {

		var hco *v1beta1.HyperConverged

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			hco.Spec.Infra = v1beta1.HyperConvergedConfig{
				NodePlacement: newHyperConvergedConfig(),
			}
			hco.Spec.Workloads = v1beta1.HyperConvergedConfig{
				NodePlacement: newHyperConvergedConfig(),
			}
		})

		It("should return error if KV CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)

			kv := operands.NewKubeVirtWithNameOnly(hco)
			Expect(cli.Delete(ctx, kv)).ToNot(HaveOccurred())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("kubevirts.kubevirt.io"))
		})

		It("should return error if dry-run update of KV CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(kvUpdateFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should return error if CDI CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)
			cdi, err := operands.NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cdi)).To(BeNil())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err = wh.ValidateUpdate(newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("cdis.cdi.kubevirt.io"))
		})

		It("should return error if dry-run update of CDI CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(cdiUpdateFailure))
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should not return error if dry-run update of ALL CR passes", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(noFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(BeNil())
		})

		It("should return error if NetworkAddons CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)
			cna, err := operands.NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(cli.Delete(ctx, cna)).To(BeNil())
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err = wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io"))
		})

		It("should return error if dry-run update of NetworkAddons CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(networkUpdateFailure))

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeNetworkError))
		})

		It("should return error if SSP CR is missing", func() {
			ctx := context.TODO()
			cli := getFakeClient(hco)
			Expect(cli.Delete(ctx, operands.NewSSP(hco))).To(BeNil())
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// just do some change to force update
			newHco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "value3"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			Expect(err.Error()).Should(ContainSubstring("ssps.ssp.kubevirt.io"))
		})

		It("should return error if dry-run update of SSP CR returns error", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(getUpdateError(sspUpdateFailure))
			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).NotTo(BeNil())
			Expect(err).Should(Equal(ErrFakeSspError))

		})

		It("should return error if dry-run update is timeout", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(initiateTimeout)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)
			// change something in workloads to trigger dry-run update
			newHco.Spec.Workloads.NodePlacement.NodeSelector["a change"] = "Something else"

			err := wh.ValidateUpdate(newHco, hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(context.DeadlineExceeded))
		})

		It("should not return error if nothing was changed", func() {
			cli := getFakeClient(hco)
			cli.InitiateUpdateErrors(initiateTimeout)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			newHco := &v1beta1.HyperConverged{}
			hco.DeepCopyInto(newHco)

			Expect(wh.ValidateUpdate(newHco, hco)).ToNot(HaveOccurred())

		})

		Context("test permitted host devices update validation", func() {
			It("should allow unique PCI Host Device", func() {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

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
				Expect(wh.ValidateUpdate(newHco, hco)).ToNot(HaveOccurred())
			})

			It("should allow unique Mediate Host Device", func() {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

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
				Expect(wh.ValidateUpdate(newHco, hco)).ToNot(HaveOccurred())
			})
		})

		Context("plain-k8s tests", func() {
			It("should return error in plain-k8s if KV CR is missing", func() {
				hco := &v1beta1.HyperConverged{}
				ctx := context.TODO()
				cli := getFakeClient(hco)
				kv, err := operands.NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cli.Delete(ctx, kv)).To(BeNil())
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, false)

				newHco := commonTestUtils.NewHco()
				newHco.Spec.Infra = v1beta1.HyperConvergedConfig{
					NodePlacement: newHyperConvergedConfig(),
				}
				newHco.Spec.Workloads = v1beta1.HyperConvergedConfig{
					NodePlacement: newHyperConvergedConfig(),
				}

				err = wh.ValidateUpdate(newHco, hco)
				Expect(err).NotTo(BeNil())
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

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow updating of live migration", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the LiveMigrationConfig field
				newVal := int64(200)
				hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &newVal

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail if live migration is wrong", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the LiveMigrationConfig field
				wrongVal := "Wrong Value"
				newHco.Spec.LiveMigrationConfig.BandwidthPerMigration = &wrongVal

				err := wh.ValidateUpdate(newHco, hco)
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

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should allow updating of cert config", func() {
				cli := getFakeClient(hco)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

				newHco := &v1beta1.HyperConverged{}
				hco.DeepCopyInto(newHco)

				// change something in the CertConfig fields
				newHco.Spec.CertConfig.CA.Duration.Duration = hco.Spec.CertConfig.CA.Duration.Duration * 2
				newHco.Spec.CertConfig.CA.RenewBefore.Duration = hco.Spec.CertConfig.CA.RenewBefore.Duration * 2
				newHco.Spec.CertConfig.Server.Duration.Duration = hco.Spec.CertConfig.Server.Duration.Duration * 2
				newHco.Spec.CertConfig.Server.RenewBefore.Duration = hco.Spec.CertConfig.Server.RenewBefore.Duration * 2

				err := wh.ValidateUpdate(newHco, hco)
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("should fail if cert config is wrong",
				func(newHco v1beta1.HyperConverged, errorMsg string) {
					cli := getFakeClient(hco)

					wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

					err := wh.ValidateUpdate(&newHco, hco)
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
									Duration:    metav1.Duration{Duration: 8 * time.Minute},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
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
									Duration:    metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 8 * time.Minute},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
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
									Duration:    metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 8 * time.Minute},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
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
									Duration:    metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 8 * time.Minute},
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
									Duration:    metav1.Duration{Duration: 23 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 24 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
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
									Duration:    metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 11 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
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
									Duration:    metav1.Duration{Duration: 48 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
								},
								Server: v1beta1.CertRotateConfigServer{
									Duration:    metav1.Duration{Duration: 96 * time.Hour},
									RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
								},
							},
						},
					},
					"spec.certConfig: ca.duration is smaller than server.duration"),
			)

		})

	})

	Context("validate delete validation webhook", func() {
		var hco *v1beta1.HyperConverged

		BeforeEach(func() {
			hco = &v1beta1.HyperConverged{
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.HyperConvergedName,
					Namespace: HcoValidNamespace,
				},
			}
		})

		It("should validate deletion", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			err := wh.ValidateDelete(hco)
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

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "KubeVirt" {
						return ErrFakeKvError
					}
				}
				return nil
			})

			err := wh.ValidateDelete(hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should reject if CDI deletion fails", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "CDI" {
						return ErrFakeCdiError
					}
				}
				return nil
			})

			err := wh.ValidateDelete(hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})

		It("should ignore if KV does not exist", func() {
			cli := getFakeClient(hco)
			ctx := context.TODO()

			kv := operands.NewKubeVirtWithNameOnly(hco)
			Expect(cli.Delete(ctx, kv)).To(BeNil())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			err := wh.ValidateDelete(hco)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject if getting KV failed for not-not-exists error", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			cli.InitiateGetErrors(func(key client.ObjectKey) error {
				if key.Name == "kubevirt-kubevirt-hyperconverged" {
					return ErrFakeKvError
				}
				return nil
			})

			err := wh.ValidateDelete(hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeKvError))
		})

		It("should ignore if CDI does not exist", func() {
			cli := getFakeClient(hco)
			ctx := context.TODO()

			cdi := operands.NewCDIWithNameOnly(hco)
			Expect(cli.Delete(ctx, cdi)).To(BeNil())

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			err := wh.ValidateDelete(hco)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject if getting CDI failed for not-not-exists error", func() {
			cli := getFakeClient(hco)

			wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

			cli.InitiateGetErrors(func(key client.ObjectKey) error {
				if key.Name == "cdi-kubevirt-hyperconverged" {
					return ErrFakeCdiError
				}
				return nil
			})

			err := wh.ValidateDelete(hco)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(ErrFakeCdiError))
		})
	})

	Context("unsupported annotation", func() {
		var hco *v1beta1.HyperConverged
		BeforeEach(func() {
			Expect(os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)).To(BeNil())
			hco = commonTestUtils.NewHco()
		})

		DescribeTable("should accept if annotation is valid",
			func(annotationName, annotation string) {
				cli := getFakeClient(hco)
				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

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
				cli := getFakeClient(hco)
				cli.InitiateUpdateErrors(initiateTimeout)

				wh := NewWebhookHandler(logger, cli, HcoValidNamespace, true)

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

	return commonTestUtils.InitClient([]runtime.Object{hco, kv, cdi, cna, operands.NewSSP(hco)})
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
