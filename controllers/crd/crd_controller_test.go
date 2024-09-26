package crd

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
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
	deschedulerRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: hcoutil.DeschedulerCRDName,
		},
	}
	otherRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "other",
		},
	}
)

var _ = Describe("CRDController", func() {

	getClusterInfo := hcoutil.GetClusterInfo

	Describe("Reconcile KubeDescheduler", func() {

		Context("Descheduler CRD", func() {

			externalClusterInfo := hcoutil.GetClusterInfo

			logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("descheduler_controller_test")

			BeforeEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = externalClusterInfo
			})

			clusterObjects := []client.Object{
				&openshiftconfigv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Spec: openshiftconfigv1.ClusterVersionSpec{
						ClusterID: "clusterId",
					},
				},
				&openshiftconfigv1.Infrastructure{
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
				},
				&openshiftconfigv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.IngressSpec{
						Domain: "domain",
					},
				},
				&openshiftconfigv1.DNS{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.DNSSpec{
						BaseDomain: commontestutils.BaseDomain,
					},
				},
				&openshiftconfigv1.Network{
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
				},
				&openshiftconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.APIServerSpec{},
				},
			}

			It("Should trigger a restart of the operator if KubeDescheduler was not there and it appeared", func() {

				cl := commontestutils.InitClient(clusterObjects)
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "KubeDescheduler is not installed")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeFalse(), "KubeDescheduler is not installed")

				testCh := make(chan struct{}, 1)

				r := &ReconcileCRD{
					client:       cl,
					restartCh:    testCh,
					eventEmitter: commontestutils.NewEventEmitterMock(),
				}

				deschedulerCRD := &apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: hcoutil.DeschedulerCRDName,
					},
				}

				err := cl.Create(context.TODO(), deschedulerCRD)
				Expect(err).NotTo(HaveOccurred())
				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "When the operator started the KubeDescheduler wasn't available")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "KubeDescheduler is now installed")

				res, err := r.Reconcile(context.Background(), deschedulerRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))
				Eventually(testCh).Should(Receive())

			})

			It("Should not trigger a restart of the operator if KubeDescheduler was not there and another CRD appeared", func() {

				cl := commontestutils.InitClient(clusterObjects)
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "KubeDescheduler is not installed")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeFalse(), "KubeDescheduler is not installed")

				testCh := make(chan struct{}, 1)

				r := &ReconcileCRD{
					client:       cl,
					restartCh:    testCh,
					eventEmitter: commontestutils.NewEventEmitterMock(),
				}

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "When the operator started the KubeDescheduler wasn't available")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeFalse(), "KubeDescheduler is still not installed")

				res, err := r.Reconcile(context.Background(), otherRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))
				Consistently(testCh).Should(Not(Receive()))

			})

			It("Should not trigger a restart of the operator if KubeDescheduler was already there and its CRD got updated", func() {

				deschedulerCRD := &apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: hcoutil.DeschedulerCRDName,
					},
				}
				clusterObjects := append(clusterObjects, deschedulerCRD)

				cl := commontestutils.InitClient(clusterObjects)
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "KubeDescheduler is alredy installed")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "KubeDescheduler is already installed")

				testCh := make(chan struct{}, 1)

				r := &ReconcileCRD{
					client:       cl,
					restartCh:    testCh,
					eventEmitter: commontestutils.NewEventEmitterMock(),
				}

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "When the operator started the KubeDescheduler was already available")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "KubeDescheduler is already installed")

				res, err := r.Reconcile(context.Background(), deschedulerRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))
				Consistently(testCh).Should(Not(Receive()))

			})

			It("Should not trigger a restart of the operator if KubeDescheduler was already there and another CRD got updated", func() {

				deschedulerCRD := &apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: hcoutil.DeschedulerCRDName,
					},
				}
				clusterObjects := append(clusterObjects, deschedulerCRD)

				cl := commontestutils.InitClient(clusterObjects)
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "KubeDescheduler is alredy installed")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "KubeDescheduler is already installed")

				testCh := make(chan struct{}, 1)

				r := &ReconcileCRD{
					client:       cl,
					restartCh:    testCh,
					eventEmitter: commontestutils.NewEventEmitterMock(),
				}

				Expect(hcoutil.GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "When the operator started the KubeDescheduler was already available")
				Expect(hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "KubeDescheduler is already installed")

				res, err := r.Reconcile(context.Background(), otherRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))
				Consistently(testCh).Should(Not(Receive()))

			})

		})

	})
})
