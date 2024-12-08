package nodes

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// Mock TestRequest to simulate Reconcile() being called on an event for a watched resource
var (
	request = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nodes-controller",
			Namespace: commontestutils.Namespace,
		},
	}
)

var _ = Describe("NodesController", func() {

	getClusterInfo := hcoutil.GetClusterInfo

	Describe("Reconcile NodesController", func() {

		BeforeEach(func() {
			hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
				return commontestutils.ClusterInfoMock{}
			}
			_ = os.Setenv(hcoutil.OperatorNamespaceEnv, commontestutils.Namespace)
		})

		AfterEach(func() {
			hcoutil.GetClusterInfo = getClusterInfo
		})

		Context("Node Count Change", func() {
			It("Should update InfrastructureHighlyAvailable to true if there are two or more worker nodes", func() {
				hco := commontestutils.NewHco()
				numWorkerNodes := 3
				var nodesArray []client.Object
				for i := range numWorkerNodes {
					workerNode := &corev1.Node{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("worker%d", i),
							Labels: map[string]string{
								"node-role.kubernetes.io/worker": "",
							},
						},
					}
					nodesArray = append(nodesArray, workerNode)
				}

				resources := []client.Object{hco, nodesArray[0], nodesArray[1], nodesArray[2]}
				cl := commontestutils.InitClient(resources)
				logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("nodes_controller_test")
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())
				r := &ReconcileNodeCounter{
					client: cl,
				}

				// Reconcile to update HCO's status with the correct InfrastructureHighlyAvailable value
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())

				latestHCO := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: commontestutils.Name, Namespace: commontestutils.Namespace},
						latestHCO),
				).ToNot(HaveOccurred())

				Expect(latestHCO.Status.InfrastructureHighlyAvailable).To(BeTrue())
				Expect(res).To(Equal(reconcile.Result{}))
			})
			It("Should update InfrastructureHighlyAvailable to false if there is only one worker node", func() {
				hco := commontestutils.NewHco()
				workerNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				}
				resources := []client.Object{hco, workerNode}
				cl := commontestutils.InitClient(resources)
				logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("nodes_controller_test")
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())
				r := &ReconcileNodeCounter{
					client: cl,
				}

				// Reconcile to update HCO's status with the correct InfrastructureHighlyAvailable value
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())

				latestHCO := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: commontestutils.Name, Namespace: commontestutils.Namespace},
						latestHCO),
				).ToNot(HaveOccurred())

				Expect(hco.Status.InfrastructureHighlyAvailable).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))
			})
		})

	})
})
