package webhooks

import (
	"context"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// ReconcileAPIServer reconciles APIServer to consume uptodate TLSSecurityProfile
type ReconcileAPIServer struct {
	client client.Client
	ci     hcoutil.ClusterInfo
}

var (
	logger = logf.Log.WithName("webhooks-controller")
)

// Implement reconcile.Reconciler so the controller can reconcile objects
var _ reconcile.Reconciler = &ReconcileAPIServer{}

func (r *ReconcileAPIServer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	err := r.ci.RefreshAPIServerCR(ctx, r.client)
	// TODO: once https://github.com/kubernetes-sigs/controller-runtime/issues/2032
	// is properly addressed, avoid polling on the APIServer CR
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
}

// RegisterReconciler creates a new HyperConverged Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo) error {
	if ci.IsOpenshift() {
		return add(mgr, newReconciler(mgr, ci))
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo) reconcile.Reconciler {
	r := &ReconcileAPIServer{
		client: mgr.GetClient(),
		ci:     ci,
	}
	return r
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	// Setup a new controller to reconcile APIServer
	logger.Info("Setting up APIServer controller")
	c, err := controller.New("hco-webhook-apiserver-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch APIServer and enqueue APIServer object key
	if err := c.Watch(&source.Kind{Type: &openshiftconfigv1.APIServer{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}
