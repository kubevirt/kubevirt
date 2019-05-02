package kwebui

import (
	"context"

    extenstionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubevirtv1alpha1 "github.com/kubevirt/web-ui-operator/pkg/apis/kubevirt/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kwebui")

// Add creates a new KWebUI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKWebUI{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kwebui-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KWebUI
	err = c.Watch(&source.Kind{Type: &kubevirtv1alpha1.KWebUI{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner KWebUI
	err = c.Watch(&source.Kind{Type: &extenstionsv1beta1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubevirtv1alpha1.KWebUI{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKWebUI{}

// ReconcileKWebUI reconciles a KWebUI object
type ReconcileKWebUI struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KWebUI object and makes changes based on the state read
// and what is in the KWebUI.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKWebUI) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// TODO: in case of error wait before reconciling again, see
	// following does not work: return reconcile.Result{RequeueAfter: RequeueDelay}, err
	// for reason, see: vendor/sigs.k8s.io/controller-runtime/pkg/internal/controller/controller.go

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KWebUI")

	// Fetch the KWebUI instance
	instance := &kubevirtv1alpha1.KWebUI{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			// TODO: use finalizer if the KWebUI CR is deleted
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	reqLogger.Info("Desired kubevirt-web-ui version: ", "instance.Spec.Version", instance.Spec.Version)

	// Fetch the kubevirt-web-ui Deployment
	deployment := &extenstionsv1beta1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "console", Namespace: request.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return freshProvision(r, request, instance)
		}
		reqLogger.Info("kubevirt-web-ui Deployment failed to be retrieved. Re-trying in a moment.", "error", err)
		updateStatus(r, request, PhaseOtherError, "Failed to retrieve kubevirt-web-ui Deployment object.")
		return reconcile.Result{}, err
	}

	// Deployment found
	return ReconcileExistingDeployment(r, request, instance, deployment)
}
