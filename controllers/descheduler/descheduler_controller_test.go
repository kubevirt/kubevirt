package descheduler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// Mock TestRequest to simulate Reconcile() being called on an event for a watched resource
var (
	request = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      hcoutil.DeschedulerCRName,
			Namespace: hcoutil.DeschedulerNamespace,
		},
	}
)

var _ = Describe("DeschedulerController", func() {

	Describe("Reconcile KubeDescheduler", func() {

		BeforeEach(func() {
			getClusterInfo := hcoutil.GetClusterInfo
			hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
				return commontestutils.ClusterInfoMock{}
			}

			origLog := log
			log = GinkgoLogr

			DeferCleanup(func() {
				hcoutil.GetClusterInfo = getClusterInfo
				log = origLog
			})
		})

		Context("KubeDescheduler CR", func() {
			DescribeTable("should set the kubevirt_hco_misconfigured_descheduler metric", func(resources []client.Object, metricValueValidator gomegatypes.GomegaMatcher) {
				cl := commontestutils.InitClient(resources)

				r := &ReconcileDescheduler{
					client: cl,
				}

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				misconfiguredDeschedulerMetrix, err := metrics.GetHCOMetrictMisconfiguredDescheduler()
				Expect(err).ToNot(HaveOccurred())
				Expect(misconfiguredDeschedulerMetrix).To(metricValueValidator)
			},
				Entry("should set the metric to false if the KubeDescheduler CR is not found", nil, BeFalse()),
				Entry("should set the metric to true for the default KubeDescheduler, at it not fit KubeVirt",
					[]client.Object{
						&deschedulerv1.KubeDescheduler{
							ObjectMeta: metav1.ObjectMeta{
								Name:      hcoutil.DeschedulerCRName,
								Namespace: hcoutil.DeschedulerNamespace,
							},
							Spec: deschedulerv1.KubeDeschedulerSpec{},
						},
					},
					BeTrue(),
				),
				Entry("should set the metric to true for the KubeDescheduler with unexpected configuration",
					[]client.Object{
						&deschedulerv1.KubeDescheduler{
							ObjectMeta: metav1.ObjectMeta{
								Name:      hcoutil.DeschedulerCRName,
								Namespace: hcoutil.DeschedulerNamespace,
							},
							Spec: deschedulerv1.KubeDeschedulerSpec{
								ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
									DevEnableEvictionsInBackground: true,
								},
							},
						},
					},
					BeTrue(),
				),
				Entry("should set the metric to false for the KubeDescheduler with a valid configuration",
					[]client.Object{
						&deschedulerv1.KubeDescheduler{
							ObjectMeta: metav1.ObjectMeta{
								Name:      hcoutil.DeschedulerCRName,
								Namespace: hcoutil.DeschedulerNamespace,
							},
							Spec: deschedulerv1.KubeDeschedulerSpec{
								Profiles: []deschedulerv1.DeschedulerProfile{
									deschedulerv1.RelieveAndMigrate,
								},
								ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
									DevDeviationThresholds:      &deschedulerv1.AsymmetricLowDeviationThreshold,
									DevEnableSoftTainter:        true,
									DevActualUtilizationProfile: deschedulerv1.PrometheusCPUCombinedProfile,
								},
							},
						},
					},
					BeFalse(),
				),
			)
		})
	})
})
