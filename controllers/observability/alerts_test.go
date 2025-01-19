package observability_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/observability"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/observability/rules"
)

const namespace = "observability_test"

var logger = logf.Log.WithName("observability-controller")

var _ = Describe("Reconcile Alerts", func() {
	var (
		reconciler *observability.Reconciler
		cl         client.Client
		promRules  *promv1.PrometheusRule
	)

	BeforeEach(func() {
		err := rules.SetupRules()
		Expect(err).ToNot(HaveOccurred())

		promRules, err = rules.BuildPrometheusRule(namespace, &metav1.OwnerReference{})
		Expect(err).ToNot(HaveOccurred())

		cl = commontestutils.InitClient([]client.Object{})
		mgr, err := commontestutils.NewManagerMock(&rest.Config{}, manager.Options{}, cl, logger)
		Expect(err).ToNot(HaveOccurred())

		reconciler = observability.NewReconciler(mgr, namespace, &appsv1.Deployment{})
	})

	It("Should create new PrometheusRules", func() {
		Expect(reconciler.ReconcileAlerts(context.TODO())).To(Succeed())

		var foundPromRules promv1.PrometheusRule
		err := cl.Get(context.TODO(), client.ObjectKeyFromObject(promRules), &foundPromRules)
		Expect(err).ToNot(HaveOccurred())

		Expect(foundPromRules.Spec).To(Equal(promRules.Spec))
	})

	It("Should update PrometheusRules with diffs", func() {
		diffPromRules := promRules.DeepCopy()
		diffPromRules.Spec.Groups[0].Rules[0].Expr = intstr.FromString("1")
		err := cl.Create(context.TODO(), diffPromRules)
		Expect(err).ToNot(HaveOccurred())

		Expect(reconciler.ReconcileAlerts(context.TODO())).To(Succeed())

		var foundPromRules promv1.PrometheusRule
		err = cl.Get(context.TODO(), client.ObjectKeyFromObject(promRules), &foundPromRules)
		Expect(err).ToNot(HaveOccurred())

		Expect(foundPromRules.Spec).To(Equal(promRules.Spec))
	})
})
