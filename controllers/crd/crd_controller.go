package crd

import (
	"context"

	operatorhandler "github.com/operator-framework/operator-lib/handler"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var (
	log = logf.Log.WithName("controller_crd")
)

// RegisterReconciler creates a new Descheduler Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager, restartCh chan<- struct{}) error {
	return add(mgr, newReconciler(mgr, restartCh))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, restartCh chan<- struct{}) reconcile.Reconciler {

	r := &ReconcileCRD{
		client:       mgr.GetClient(),
		restartCh:    restartCh,
		eventEmitter: hcoutil.GetEventEmitter(),
	}

	return r
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {

	// Create a new controller
	c, err := controller.New("crd-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to selected (by name) CRDs
	// look only at descheduler CRD for now
	err = c.Watch(
		source.Kind(
			mgr.GetCache(), client.Object(&apiextensionsv1.CustomResourceDefinition{}),
			&operatorhandler.InstrumentedEnqueueRequestForObject[client.Object]{},
			predicate.NewPredicateFuncs(func(object client.Object) bool {
				switch object.GetName() {
				case hcoutil.DeschedulerCRDName:
					return true
				}
				return false
			}),
		))
	if err != nil {
		return err
	}

	return nil
}

// ReconcileCRD reconciles a CRD object
type ReconcileCRD struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	eventEmitter hcoutil.EventEmitter
	restartCh    chan<- struct{}
}

// operatorRestart triggers a restart of the operator:
// the controller-runtime caching client can only handle kinds
// that were already defined when the client cache got initialized.
// If a new relevant kind got deployed at runtime,
// the operator should restart to be able to read it.
// See: https://github.com/kubernetes-sigs/controller-runtime/issues/2456
func (r *ReconcileCRD) operatorRestart() {
	r.restartCh <- struct{}{}
}

// Reconcile refreshes KubeDesheduler view on ClusterInfo singleton
func (r *ReconcileCRD) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	log.Info("Triggered by a CRD")
	if !hcoutil.GetClusterInfo().IsDeschedulerAvailable() {
		if hcoutil.GetClusterInfo().IsDeschedulerCRDDeployed(ctx, r.client) {
			log.Info("KubeDescheduler CRD got deployed, restarting the operator to reconfigure the operator for the new kind")
			r.eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "KubeDescheduler CRD got deployed, restarting the operator to reconfigure the operator for the new kind", "Restarting the operator to be able to read KubeDescheduler CRs ")
			r.operatorRestart()
		}
	}

	return reconcile.Result{}, nil
}
