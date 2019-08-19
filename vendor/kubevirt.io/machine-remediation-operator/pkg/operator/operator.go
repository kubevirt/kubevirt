package operator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
	"kubevirt.io/machine-remediation-operator/pkg/operator/components"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const machineRemediationOperatorFinalizer string = "foregroundDeleteMachineRemediationOperator"

var _ reconcile.Reconciler = &ReconcileMachineRemediationOperator{}

// ReconcileMachineRemediationOperator reconciles a MachineRemediationOperator object
type ReconcileMachineRemediationOperator struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client           client.Client
	namespace        string
	operatorVersion  string
	crdsManifestsDir string
}

// Add creates a new MachineRemediationOperator Controller and adds it to the Manager.
// The Manager will set fields on the Controller and start it when the Manager is started.
func Add(mgr manager.Manager, opts manager.Options) error {
	r, err := newReconciler(mgr, opts)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

func newReconciler(mgr manager.Manager, opts manager.Options) (reconcile.Reconciler, error) {
	return &ReconcileMachineRemediationOperator{
		client:           mgr.GetClient(),
		namespace:        opts.Namespace,
		operatorVersion:  os.Getenv(components.EnvVarOperatorVersion),
		crdsManifestsDir: "/data",
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("machine-remediation-operator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &mrv1.MachineRemediationOperator{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile monitors MachineRemediationOperator and bring all machine remediation components to desired state
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMachineRemediationOperator) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(4).Infof("Reconciling MachineRemediationOperator triggered by %s/%s\n", request.Namespace, request.Name)

	// Get MachineRemediation from request
	mro := &mrv1.MachineRemediationOperator{}
	err := r.client.Get(context.TODO(), request.NamespacedName, mro)
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

	// if MachineRemediationObject was deleted, remove all relevant componenets and remove finalizer
	if mro.DeletionTimestamp != nil {
		if err := r.deleteComponents(); err != nil {
			return reconcile.Result{}, err
		}

		mro.Finalizers = nil
		if err := r.client.Update(context.TODO(), mro); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	// add finalizer to prevent deletion of MachineRemediationOperator objet
	if !hasFinalizer(mro) {
		addFinalizer(mro)
		if err := r.client.Update(context.TODO(), mro); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := r.createOrUpdateComponents(mro); err != nil {
		glog.Errorf("Failed to create components: %v", err)
		if err := r.statusDegraded(mro, err.Error(), "Failed to create all components"); err != nil {
			glog.Errorf("Failed to update operator status: %v", err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	for _, component := range components.Components {
		ready, err := r.isDeploymentReady(component, consts.NamespaceOpenshiftMachineAPI)
		if err != nil {
			if err := r.statusProgressing(mro, err.Error(), fmt.Sprintf("Failed to get deployment %q", component)); err != nil {
				glog.Errorf("Failed to update operator status: %v", err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}

		if !ready {
			if err := r.statusProgressing(mro, "Deployment is not ready", fmt.Sprintf("Deployment %q is not ready", component)); err != nil {
				glog.Errorf("Failed to update operator status: %v", err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
	}

	if err := r.statusAvailable(mro); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateComponents(mro *mrv1.MachineRemediationOperator) error {
	for _, crd := range components.CRDS {
		glog.Infof("Creating or updating CRD %q", crd)
		if err := r.createOrUpdateCustomResourceDefinition(crd); err != nil {
			return err
		}
	}

	for _, component := range components.Components {
		glog.Infof("Creating objects for component %q", component)
		if err := r.createOrUpdateServiceAccount(component, consts.NamespaceOpenshiftMachineAPI); err != nil {
			return err
		}

		if err := r.createOrUpdateClusterRole(component); err != nil {
			return err
		}

		if err := r.createOrUpdateClusterRoleBinding(component, consts.NamespaceOpenshiftMachineAPI); err != nil {
			return err
		}

		deployData := &components.DeploymentData{
			Name:            component,
			Namespace:       consts.NamespaceOpenshiftMachineAPI,
			ImageRepository: mro.Spec.ImageRegistry,
			PullPolicy:      mro.Spec.ImagePullPolicy,
			OperatorVersion: r.operatorVersion,
			Verbosity:       "4",
		}

		if err := r.createOrUpdateDeployment(deployData); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileMachineRemediationOperator) deleteComponents() error {
	for _, component := range components.Components {
		glog.Infof("Deleting objets for component %q", component)
		if err := r.deleteDeployment(component, consts.NamespaceOpenshiftMachineAPI); err != nil {
			return err
		}

		if err := r.deleteClusterRoleBinding(component); err != nil {
			return err
		}

		if err := r.deleteClusterRole(component); err != nil {
			return err
		}

		if err := r.deleteServiceAccount(component, consts.NamespaceOpenshiftMachineAPI); err != nil {
			return err
		}
	}

	for _, crd := range components.CRDS {
		glog.Infof("Deleting CRD %q", crd)
		if err := r.deleteCustomResourceDefinition(crd); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileMachineRemediationOperator) statusAvailable(mro *mrv1.MachineRemediationOperator) error {
	now := time.Now()
	mro.Status.Conditions = []mrv1.MachineRemediationOperatorStatusCondition{
		{
			Type:               mrv1.OperatorAvailable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               mrv1.OperatorProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               mrv1.OperatorDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
	return r.client.Status().Update(context.TODO(), mro)
}

func (r *ReconcileMachineRemediationOperator) statusDegraded(mro *mrv1.MachineRemediationOperator, reason string, message string) error {
	now := time.Now()
	mro.Status.Conditions = []mrv1.MachineRemediationOperatorStatusCondition{
		{
			Type:               mrv1.OperatorAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               mrv1.OperatorProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               mrv1.OperatorDegraded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
	}
	return r.client.Status().Update(context.TODO(), mro)
}

func (r *ReconcileMachineRemediationOperator) statusProgressing(mro *mrv1.MachineRemediationOperator, reason string, message string) error {
	now := time.Now()
	mro.Status.Conditions = []mrv1.MachineRemediationOperatorStatusCondition{
		{
			Type:               mrv1.OperatorAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               mrv1.OperatorProgressing,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
		{
			Type:               mrv1.OperatorDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
	return r.client.Status().Update(context.TODO(), mro)
}

func addFinalizer(mro *mrv1.MachineRemediationOperator) {
	if !hasFinalizer(mro) {
		mro.Finalizers = append(mro.Finalizers, machineRemediationOperatorFinalizer)
	}
}

func hasFinalizer(mro *mrv1.MachineRemediationOperator) bool {
	for _, f := range mro.Finalizers {
		if f == machineRemediationOperatorFinalizer {
			return true
		}
	}
	return false
}
