package descheduler

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"

	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
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

	getClusterInfo := hcoutil.GetClusterInfo

	Describe("Reconcile KubeDescheduler", func() {

		BeforeEach(func() {
			hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
				return commontestutils.ClusterInfoMock{}
			}
		})

		AfterEach(func() {
			hcoutil.GetClusterInfo = getClusterInfo
		})

		Context("KubeDescheduler CR", func() {

			externalClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = externalClusterInfo
			})

			It("Should refresh cached KubeDescheduler if the reconciliation is caused by a change there", func() {

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

				dns := &openshiftconfigv1.DNS{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.DNSSpec{
						BaseDomain: commontestutils.BaseDomain,
					},
				}

				ipv4network := &openshiftconfigv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: openshiftconfigv1.NetworkStatus{
						ClusterNetwork: []openshiftconfigv1.ClusterNetworkEntry{
							{
								CIDR: "10.128.0.0/14",
							},
						},
					},
				}

				apiServer := &openshiftconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.APIServerSpec{},
				}

				deschedulerCRD := &apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: hcoutil.DeschedulerCRDName,
					},
				}

				deschedulerNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: hcoutil.DeschedulerNamespace,
						Annotations: map[string]string{
							hcoutil.OpenshiftNodeSelectorAnn: "",
						},
					},
				}

				descheduler := &deschedulerv1.KubeDescheduler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      hcoutil.DeschedulerCRName,
						Namespace: hcoutil.DeschedulerNamespace,
					},
					Spec: deschedulerv1.KubeDeschedulerSpec{},
				}

				resources := []client.Object{deschedulerCRD, deschedulerNamespace, descheduler, clusterVersion, infrastructure, ingress, dns, ipv4network, apiServer}
				cl := commontestutils.InitClient(resources)

				logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("descheduler_controller_test")
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured()).To(BeTrue(), "default KubeDescheduler should not fit KubeVirt")

				r := &ReconcileDescheduler{
					client: cl,
				}

				// Reconcile to get all related objects under HCO's status
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				// Update KubeDescheduler CR
				descheduler.Spec.ProfileCustomizations = &deschedulerv1.ProfileCustomizations{
					DevEnableEvictionsInBackground: true,
				}
				Expect(cl.Update(context.TODO(), descheduler)).To(Succeed())
				Expect(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured()).To(BeTrue(), "should still return the cached value (initial value)")

				// Reconcile again to refresh KubeDescheduler CR in memory
				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				Expect(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured()).To(BeFalse(), "should return the up-to-date value")

				// Update again the KubeDescheduler CR
				descheduler.Spec.ProfileCustomizations = &deschedulerv1.ProfileCustomizations{
					DevEnableEvictionsInBackground: false,
				}
				Expect(cl.Update(context.TODO(), descheduler)).To(Succeed())
				Expect(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured()).To(BeFalse(), "should still return the cached value (previous value)")

				// Reconcile again to refresh KubeDescheduler CR in memory
				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				Expect(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured()).To(BeTrue(), "should return a different up-to-date value")

			})

		})

	})
})
