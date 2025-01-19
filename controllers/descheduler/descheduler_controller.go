package descheduler

import (
	"context"

	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	operatorhandler "github.com/operator-framework/operator-lib/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var (
	log = logf.Log.WithName("controller_descheduler")
)

// RegisterReconciler creates a new Descheduler Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {

	r := &ReconcileDescheduler{
		client: mgr.GetClient(),
	}

	return r
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kubedescheduler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to KubeDeschedulers
	err = c.Watch(
		source.Kind(
			mgr.GetCache(), client.Object(&deschedulerv1.KubeDescheduler{}),
			&operatorhandler.InstrumentedEnqueueRequestForObject[client.Object]{},
			predicate.Or[client.Object](predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}),
		))
	if err != nil {
		return err
	}

	return nil
}

// ReconcileDescheduler reconciles a KubeDescheduler object
type ReconcileDescheduler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
}

// Reconcile refreshes KubeDesheduler view on ClusterInfo singleton
func (r *ReconcileDescheduler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {

	log.Info("Triggered by Descheduler CR, refreshing it")
	err := hcoutil.GetClusterInfo().RefreshDeschedulerCR(ctx, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}
	metrics.SetHCOMetricMisconfiguredDescheduler(hcoutil.GetClusterInfo().IsDeschedulerMisconfigured())

	return reconcile.Result{}, nil
}
