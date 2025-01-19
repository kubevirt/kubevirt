package observability

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/observability/rules"
)

const testNamespace = "observability_test"

var logger = logf.Log.WithName("observability-controller")

var _ = Describe("Observability Controller", func() {
	var mgr manager.Manager

	BeforeEach(func() {
		err := os.Setenv("OPERATOR_NAMESPACE", testNamespace)
		Expect(err).ToNot(HaveOccurred())

		cl := commontestutils.InitClient([]client.Object{})

		mgr, err = commontestutils.NewManagerMock(&rest.Config{}, manager.Options{}, cl, logger)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should successfully setup the controller", func() {
		err := SetupWithManager(mgr, &appsv1.Deployment{})
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.ListAlerts()).To(Not(BeEmpty()))
	})

	It("Should successfully reconcile observability", func() {
		ownerDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-deployment",
			},
		}

		reconciler := NewReconciler(mgr, testNamespace, ownerDeployment)
		Expect(reconciler.owner.Name).To(Equal(ownerDeployment.Name))
		Expect(reconciler.namespace).To(Equal(testNamespace))
		Expect(reconciler.config).To(Equal(mgr.GetConfig()))
		Expect(reconciler.Client).To(Equal(mgr.GetClient()))
	})

	It("Should receive periodic events in reconciler events channel", func() {
		reconciler := NewReconciler(mgr, testNamespace, &appsv1.Deployment{})
		reconciler.startEventLoop()

		Eventually(reconciler.events).
			WithTimeout(5 * time.Second).
			WithPolling(100 * time.Millisecond).
			Should(Receive())
	})
})
