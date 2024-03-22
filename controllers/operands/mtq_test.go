package operands

import (
	"context"
	"maps"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubevirt.io/controller-lifecycle-operator-sdk/api"
	mtqv1alpha1 "kubevirt.io/managed-tenant-quota/staging/src/kubevirt.io/managed-tenant-quota-api/pkg/apis/core/v1alpha1"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("MTQ tests", func() {
	var (
		hco *v1beta1.HyperConverged
		req *common.HcoRequest
		cl  client.Client

		testNodePlacement = api.NodePlacement{
			NodeSelector: map[string]string{
				"test": "testing",
			},
			Tolerations: []corev1.Toleration{{Key: "test", Operator: corev1.TolerationOpEqual, Value: "test", Effect: corev1.TaintEffectNoSchedule}},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchFields: []corev1.NodeSelectorRequirement{
									{
										Key:      "test",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"test"},
									},
								},
							},
						},
					},
				},
			},
		}
	)

	getClusterInfo := hcoutil.GetClusterInfo

	BeforeEach(func() {
		hco = commontestutils.NewHco()
		req = commontestutils.NewReq(hco)
		hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
			return &commontestutils.ClusterInfoMock{}
		}
	})

	AfterEach(func() {
		hcoutil.GetClusterInfo = getClusterInfo
	})

	Context("test NewMTQ", func() {
		It("should have all default fields", func() {
			mtq := NewMTQ(hco)

			Expect(mtq.Name).To(Equal("mtq-" + hco.Name))
			Expect(mtq.Namespace).To(BeEmpty())

			Expect(mtq.Spec.Infra.Tolerations).To(BeEmpty())
			Expect(mtq.Spec.Infra.Affinity).To(BeNil())
			Expect(mtq.Spec.Infra.NodeSelector).To(BeEmpty())

			Expect(mtq.Spec.Workloads.Tolerations).To(BeEmpty())
			Expect(mtq.Spec.Workloads.Affinity).To(BeNil())
			Expect(mtq.Spec.Workloads.NodeSelector).To(BeEmpty())

			Expect(mtq.Spec.PriorityClass).To(HaveValue(Equal(mtqv1alpha1.MTQPriorityClass(kvPriorityClass))))

			Expect(mtq.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
			Expect(mtq.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("24h0m0s"))

			Expect(mtq.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
			Expect(mtq.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("12h0m0s"))
		})

		It("should get node placement node placement configurations from the HyperConverged CR", func() {
			hco.Spec.Infra.NodePlacement = &testNodePlacement
			hco.Spec.Workloads.NodePlacement = &testNodePlacement

			mtq := NewMTQ(hco)

			Expect(mtq.Spec.Infra).To(Equal(testNodePlacement))
			Expect(mtq.Spec.Workloads).To(Equal(testNodePlacement))
		})

		It("should get node placement certification configurations from the HyperConverged CR", func() {

			hco.Spec.CertConfig = v1beta1.HyperConvergedCertConfig{
				CA: v1beta1.CertRotateConfigCA{
					Duration:    &metav1.Duration{Duration: time.Hour * 72},
					RenewBefore: &metav1.Duration{Duration: time.Hour * 56},
				},
				Server: v1beta1.CertRotateConfigServer{
					Duration:    &metav1.Duration{Duration: time.Hour * 36},
					RenewBefore: &metav1.Duration{Duration: time.Hour * 18},
				},
			}

			mtq := NewMTQ(hco)

			Expect(mtq.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("72h0m0s"))
			Expect(mtq.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("56h0m0s"))

			Expect(mtq.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(mtq.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("36h0m0s"))
			Expect(mtq.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("18h0m0s"))
		})
	})

	Context("check FG", func() {
		It("should not create MTQ if the FG is not set", func() {
			cl = commontestutils.InitClient([]client.Object{hco})

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundMTQs := &mtqv1alpha1.MTQList{}
			Expect(cl.List(context.Background(), foundMTQs)).To(Succeed())
			Expect(foundMTQs.Items).To(BeEmpty())
		})

		It("should delete MTQ if the FG is not set", func() {
			mtq := NewMTQ(hco)
			cl = commontestutils.InitClient([]client.Object{hco, mtq})

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal(mtq.Name))
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeTrue())

			foundMTQs := &mtqv1alpha1.MTQList{}
			Expect(cl.List(context.Background(), foundMTQs)).To(Succeed())
			Expect(foundMTQs.Items).To(BeEmpty())
		})

		It("should create MTQ if the FG is set", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			cl = commontestutils.InitClient([]client.Object{hco})

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal("mtq-kubevirt-hyperconverged"))
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundMTQ := &mtqv1alpha1.MTQ{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: res.Name}, foundMTQ)).To(Succeed())

			Expect(foundMTQ.Name).To(Equal("mtq-" + hco.Name))
			Expect(foundMTQ.Namespace).To(BeEmpty())

			// example of field set by the handler
			Expect(foundMTQ.Spec.PriorityClass).To(HaveValue(Equal(mtqv1alpha1.MTQPriorityClass(kvPriorityClass))))
		})

		It("should not create MTQ on a single node cluster, even if the FG is set", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			cl = commontestutils.InitClient([]client.Object{hco})

			hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
				return &commontestutils.ClusterInfoSNOMock{}
			}

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal("mtq-kubevirt-hyperconverged"))
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundMTQ := &mtqv1alpha1.MTQ{}
			err := cl.Get(context.Background(), client.ObjectKey{Name: res.Name}, foundMTQ)
			Expect(err).To(MatchError(errors.IsNotFound, "not found error"))
		})
	})

	Context("check update", func() {

		It("should update MTQ fields, if not matched to the requirements", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			mtq := NewMTQWithNameOnly(hco)
			mtq.Spec.Infra = testNodePlacement
			mtq.Spec.PriorityClass = ptr.To(mtqv1alpha1.MTQPriorityClass("wrongPC"))
			mtq.Spec.CertConfig = &mtqv1alpha1.MTQCertConfig{
				CA: &mtqv1alpha1.CertConfig{
					Duration:    &metav1.Duration{Duration: time.Hour * 72},
					RenewBefore: &metav1.Duration{Duration: time.Hour * 56},
				},
			}

			cl = commontestutils.InitClient([]client.Object{hco, mtq})
			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())
			Expect(res.Updated).To(BeTrue())

			foundMTQ := &mtqv1alpha1.MTQ{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: res.Name}, foundMTQ)).To(Succeed())
			Expect(foundMTQ.Spec.Infra.Affinity).To(BeNil())
			Expect(foundMTQ.Spec.Infra.NodeSelector).To(BeEmpty())
			Expect(foundMTQ.Spec.Infra.Tolerations).To(BeEmpty())

			Expect(foundMTQ.Spec.PriorityClass).To(HaveValue(Equal(mtqv1alpha1.MTQPriorityClass(kvPriorityClass))))
			Expect(foundMTQ.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(foundMTQ.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(foundMTQ.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
			Expect(foundMTQ.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("24h0m0s"))

			Expect(foundMTQ.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(foundMTQ.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(foundMTQ.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
			Expect(foundMTQ.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("12h0m0s"))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource := NewMTQ(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue
			outdatedResource.Labels[hcoutil.AppLabel] = expectedLabels[hcoutil.AppLabel]

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &mtqv1alpha1.MTQ{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default on label deletion without touching user added ones", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource := NewMTQ(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &mtqv1alpha1.MTQ{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})
	})

	Context("check cache", func() {
		It("should create new cache if it empty", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			handler := newMtqHandler(cl, commontestutils.GetScheme())
			op, ok := handler.(*conditionalHandler)
			Expect(ok).To(BeTrue())

			hooks, ok := op.operand.hooks.(*mtqHooks)
			Expect(ok).To(BeTrue())

			Expect(hooks.cache).To(BeNil())

			res := handler.ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			cache := hooks.cache
			Expect(cache).ToNot(BeNil())

			Expect(hooks.getFullCr(hco)).To(BeIdenticalTo(cache))

			By("recreate cache after reset")
			handler.reset()
			Expect(hooks.cache).To(BeNil())
			res = handler.ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			Expect(hooks.cache).ToNot(BeIdenticalTo(cache))
			mtq, _ := hooks.getFullCr(hco)
			Expect(mtq).ToNot(BeIdenticalTo(cache))
			Expect(mtq).To(BeIdenticalTo(hooks.cache))
		})
	})
})
