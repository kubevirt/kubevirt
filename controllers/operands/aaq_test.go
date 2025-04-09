package operands

import (
	"context"
	"maps"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	aaqv1alpha1 "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core/v1alpha1"
	"kubevirt.io/controller-lifecycle-operator-sdk/api"
)

var _ = Describe("AAQ tests", func() {
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

	Context("test NewAAQ", func() {
		It("should have all default fields", func() {
			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())

			Expect(aaq.Name).To(Equal("aaq-" + hco.Name))
			Expect(aaq.Namespace).To(BeEmpty())

			Expect(aaq.Spec.Infra.Tolerations).To(BeEmpty())
			Expect(aaq.Spec.Infra.Affinity).To(BeNil())
			Expect(aaq.Spec.Infra.NodeSelector).To(BeEmpty())

			Expect(aaq.Spec.Workloads.Tolerations).To(BeEmpty())
			Expect(aaq.Spec.Workloads.Affinity).To(BeNil())
			Expect(aaq.Spec.Workloads.NodeSelector).To(BeEmpty())

			Expect(aaq.Spec.PriorityClass).To(HaveValue(Equal(aaqv1alpha1.AAQPriorityClass(kvPriorityClass))))

			Expect(aaq.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
			Expect(aaq.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("24h0m0s"))

			Expect(aaq.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
			Expect(aaq.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("12h0m0s"))

			Expect(aaq.Spec.NamespaceSelector).To(BeNil())
			Expect(aaq.Spec.Configuration.VmiCalculatorConfiguration.ConfigName).To(Equal(aaqv1alpha1.DedicatedVirtualResources))
			Expect(aaq.Spec.Configuration.AllowApplicationAwareClusterResourceQuota).To(BeFalse())
		})

		It("should have namespaceSelector", func() {
			labels := map[string]string{"name": "value"}

			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			}

			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(aaq.Spec.NamespaceSelector).ToNot(BeNil())
			Expect(aaq.Spec.NamespaceSelector.MatchLabels).To(Equal(labels))
		})

		It("should have ConfigName", func() {
			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{
				VmiCalcConfigName: ptr.To(aaqv1alpha1.VmiPodUsage),
			}

			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(aaq.Spec.Configuration.VmiCalculatorConfiguration.ConfigName).To(Equal(aaqv1alpha1.VmiPodUsage))
		})

		It("should have ConfigName", func() {
			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{
				AllowApplicationAwareClusterResourceQuota: true,
			}

			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(aaq.Spec.Configuration.AllowApplicationAwareClusterResourceQuota).To(BeTrue())
		})

		It("should get node placement configurations from the HyperConverged CR", func() {
			hco.Spec.Infra.NodePlacement = &testNodePlacement
			hco.Spec.Workloads.NodePlacement = &testNodePlacement

			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())

			Expect(aaq.Spec.Infra).To(Equal(testNodePlacement))
			Expect(aaq.Spec.Workloads).To(Equal(testNodePlacement))
		})

		It("should get certification configurations from the HyperConverged CR", func() {

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

			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())

			Expect(aaq.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("72h0m0s"))
			Expect(aaq.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("56h0m0s"))

			Expect(aaq.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(aaq.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("36h0m0s"))
			Expect(aaq.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("18h0m0s"))
		})
	})

	Context("check FG", func() {
		It("should not create AAQ if the FG is not set", func() {
			cl = commontestutils.InitClient([]client.Object{hco})

			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundAAQs := &aaqv1alpha1.AAQList{}
			Expect(cl.List(context.Background(), foundAAQs)).To(Succeed())
			Expect(foundAAQs.Items).To(BeEmpty())
		})

		It("should delete AAQ if the enableApplicationAwareQuota FG is not set", func() {
			aaq, err := NewAAQ(hco)
			Expect(err).ToNot(HaveOccurred())
			cl = commontestutils.InitClient([]client.Object{hco, aaq})

			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal(aaq.Name))
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeTrue())

			foundAAQs := &aaqv1alpha1.AAQList{}
			Expect(cl.List(context.Background(), foundAAQs)).To(Succeed())
			Expect(foundAAQs.Items).To(BeEmpty())
		})

		It("should create AAQ if the enableApplicationAwareQuota FG is true", func() {
			hco.Spec.EnableApplicationAwareQuota = ptr.To(true)
			cl = commontestutils.InitClient([]client.Object{hco})

			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal("aaq-kubevirt-hyperconverged"))
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundAAQ := &aaqv1alpha1.AAQ{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: res.Name}, foundAAQ)).To(Succeed())

			Expect(foundAAQ.Name).To(Equal("aaq-" + hco.Name))
			Expect(foundAAQ.Namespace).To(BeEmpty())

			// example of field set by the handler
			Expect(foundAAQ.Spec.PriorityClass).To(HaveValue(Equal(aaqv1alpha1.AAQPriorityClass(kvPriorityClass))))
		})
	})

	Context("check update", func() {

		It("should update AAQ fields, if not matched to the requirements", func() {
			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{}
			hco.Spec.EnableApplicationAwareQuota = ptr.To(true)
			aaq := NewAAQWithNameOnly(hco)
			aaq.Spec.Infra = testNodePlacement
			aaq.Spec.PriorityClass = ptr.To[aaqv1alpha1.AAQPriorityClass]("wrongPC")
			aaq.Spec.CertConfig = &aaqv1alpha1.AAQCertConfig{
				CA: &aaqv1alpha1.CertConfig{
					Duration:    &metav1.Duration{Duration: time.Hour * 72},
					RenewBefore: &metav1.Duration{Duration: time.Hour * 56},
				},
			}
			aaq.Spec.Configuration.VmiCalculatorConfiguration.ConfigName = aaqv1alpha1.IgnoreVmiCalculator
			aaq.Spec.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "value"},
			}

			cl = commontestutils.InitClient([]client.Object{hco, aaq})
			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())
			Expect(res.Updated).To(BeTrue())

			foundAAQ := &aaqv1alpha1.AAQ{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: res.Name}, foundAAQ)).To(Succeed())

			Expect(foundAAQ.Spec.NamespaceSelector).To(BeNil())
			Expect(foundAAQ.Spec.Configuration.VmiCalculatorConfiguration.ConfigName).To(Equal(aaqv1alpha1.DedicatedVirtualResources))

			Expect(foundAAQ.Spec.Infra.Affinity).To(BeNil())
			Expect(foundAAQ.Spec.Infra.NodeSelector).To(BeEmpty())
			Expect(foundAAQ.Spec.Infra.Tolerations).To(BeEmpty())

			Expect(foundAAQ.Spec.PriorityClass).To(HaveValue(Equal(aaqv1alpha1.AAQPriorityClass(kvPriorityClass))))
			Expect(foundAAQ.Spec.CertConfig.CA).ToNot(BeNil())
			Expect(foundAAQ.Spec.CertConfig.CA.Duration).ToNot(BeNil())
			Expect(foundAAQ.Spec.CertConfig.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
			Expect(foundAAQ.Spec.CertConfig.CA.RenewBefore.Duration.String()).To(Equal("24h0m0s"))

			Expect(foundAAQ.Spec.CertConfig.Server).ToNot(BeNil())
			Expect(foundAAQ.Spec.CertConfig.Server.Duration).ToNot(BeNil())
			Expect(foundAAQ.Spec.CertConfig.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
			Expect(foundAAQ.Spec.CertConfig.Server.RenewBefore.Duration.String()).To(Equal("12h0m0s"))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{}
			hco.Spec.EnableApplicationAwareQuota = ptr.To(true)
			outdatedResource := NewAAQWithNameOnly(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &aaqv1alpha1.AAQ{}
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
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			hco.Spec.ApplicationAwareConfig = &v1beta1.ApplicationAwareConfigurations{}
			hco.Spec.EnableApplicationAwareQuota = ptr.To(true)
			outdatedResource := NewAAQWithNameOnly(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newAAQHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &aaqv1alpha1.AAQ{}
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
			hco.Spec.EnableApplicationAwareQuota = ptr.To(true)
			handler := newAAQHandler(cl, commontestutils.GetScheme())
			op, ok := handler.(*conditionalHandler)
			Expect(ok).To(BeTrue())

			hooks, ok := op.operand.hooks.(*aaqHooks)
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
