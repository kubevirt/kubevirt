package hyperconverged

import (
	"context"

	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	kubevirt "kubevirt.io/kubevirt/pkg/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_hyperconverged")

// Add creates a new HyperConverged Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHyperConverged{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("hyperconverged-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HyperConverged
	err = c.Watch(&source.Kind{Type: &hcov1alpha1.HyperConverged{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner HyperConverged
	err = c.Watch(&source.Kind{Type: &kubevirt.KubeVirt{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHyperConverged{}

// ReconcileHyperConverged reconciles a HyperConverged object
type ReconcileHyperConverged struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a HyperConverged object and makes changes based on the state read
// and what is in the HyperConverged.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHyperConverged) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling HyperConverged operator")

	// Fetch the HyperConverged instance
	instance := &hcov1alpha1.HyperConverged{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new KubeVirt object
	virtCR := newKubeVirtForCR(instance)

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, virtCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this KubeVirt CR already exists
	found := &kubevirt.KubeVirt{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: virtCR.Name, Namespace: virtCR.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new KubeVirt CR")
		err = r.client.Create(context.TODO(), virtCR)
		if err != nil {
			return reconcile.Result{}, err
		}

		// KubeVirt CR created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// KubeVirt CR already exists - don't requeue
	reqLogger.Info("Skip reconcile: KubeVirt CR already exists")
	return reconcile.Result{}, nil
}

// newKubeVirtForCR returns a KubeVirt CR
func newKubeVirtForCR(cr *hcov1alpha1.HyperConverged) *kubevirt.KubeVirt {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &kubevirt.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: kubevirt.KubeVirtSpec{
			ImagePullPolicy: "IfNotPresent",
		},
	}
}
