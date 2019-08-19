package machineremediation

import (
	"context"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var _ reconcile.Reconciler = &ReconcileMachineRemediation{}

// ReconcileMachineRemediation reconciles a MachineRemediation object
type ReconcileMachineRemediation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	remediator Remediator
	namespace  string
}

// AddWithRemediator creates a new MachineRemediation Controller with remediator and adds it to the Manager.
// The Manager will set fields on the Controller and start it when the Manager is started.
func AddWithRemediator(mgr manager.Manager, remediator Remediator, opts manager.Options) error {
	r, err := newReconciler(mgr, remediator, opts)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

func newReconciler(mgr manager.Manager, remediator Remediator, opts manager.Options) (reconcile.Reconciler, error) {
	return &ReconcileMachineRemediation{
		client:     mgr.GetClient(),
		remediator: remediator,
		namespace:  opts.Namespace,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("machineremediation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &mrv1.MachineRemediation{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile monitors MachineRemediation and apply the remediation strategy in the case when the
// MachineRemediation was created
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMachineRemediation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(4).Infof("Reconciling MachineRemediation triggered by %s/%s\n", request.Namespace, request.Name)

	// Get MachineRemediation from request
	mr := &mrv1.MachineRemediation{}
	err := r.client.Get(context.TODO(), request.NamespacedName, mr)
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

	// we do not want to do anything on delete objects
	if mr.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	if mr.Status.State == "" {
		mrCopy := mr.DeepCopy()
		mrCopy.Status = mrv1.MachineRemediationStatus{
			State:     mrv1.RemediationStateStarted,
			Reason:    "Machine remediation started",
			StartTime: &metav1.Time{Time: time.Now()},
		}
		if err := r.client.Status().Update(context.TODO(), mrCopy); err != nil {
			glog.Errorf("failed to update MR %q status: %v", mr.Name, err)
			return reconcile.Result{}, err
		}
	}

	switch mr.Spec.Type {
	case mrv1.RemediationTypeReboot:
		glog.V(4).Infof("Run remediation reboot acion for MachineRemediation %s", mr.Name)
		if err := r.remediator.Reboot(context.TODO(), mr); err != nil {
			glog.Errorf("Remediation reboot acion for MachineRemediation %s failed with error: %v", mr.Name, err)
			return reconcile.Result{}, err
		}
	case mrv1.RemediationTypeRecreate:
		glog.V(4).Infof("Run remediation recreate acion for MachineRemediation %s", mr.Name)
		if err := r.remediator.Recreate(context.TODO(), mr); err != nil {
			glog.Errorf("Remediation recreate acion for MachineRemediation %s failed with error: %v", mr.Name, err)
			return reconcile.Result{}, err
		}
	}

	switch mr.Status.State {
	// we want to stop reconcile the object once it reaches Succeeded or Failed state
	case mrv1.RemediationStateFailed, mrv1.RemediationStateSucceeded:
		return reconcile.Result{}, nil
	// for all other cases we want to reconcile object in ten seconds, to give time for the object update
	default:
		return reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}
}
