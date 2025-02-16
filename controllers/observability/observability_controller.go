package observability

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/alertmanager"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/observability/rules"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var (
	log         = logf.Log.WithName("controller_observability")
	periodicity = 1 * time.Hour
)

type Reconciler struct {
	client.Client

	namespace string
	config    *rest.Config
	events    chan event.GenericEvent
	owner     *metav1.OwnerReference

	amApi *alertmanager.Api
}

func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	log.Info("Reconciling Observability")

	var errors []error

	if err := r.ensurePodDisruptionBudgetAtLimitIsSilenced(); err != nil {
		errors = append(errors, err)
	}

	if err := r.ReconcileAlerts(ctx); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		err := fmt.Errorf("reconciliation failed: %v", errors)
		log.Error(err, "Reconciliation failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func NewReconciler(mgr ctrl.Manager, namespace string, ownerDeployment *appsv1.Deployment) *Reconciler {
	return &Reconciler{
		Client:    mgr.GetClient(),
		namespace: namespace,
		config:    mgr.GetConfig(),
		events:    make(chan event.GenericEvent, 1),
		owner:     buildOwnerReference(ownerDeployment),
	}
}

func SetupWithManager(mgr ctrl.Manager, ownerDeployment *appsv1.Deployment) error {
	log.Info("Setting up controller")

	namespace := util.GetOperatorNamespaceFromEnv()

	err := rules.SetupRules()
	if err != nil {
		return fmt.Errorf("failed to setup Prometheus rules: %v", err)
	}

	r := NewReconciler(mgr, namespace, ownerDeployment)
	r.startEventLoop()

	return ctrl.NewControllerManagedBy(mgr).
		Named("observability").
		WatchesRawSource(source.Channel(
			r.events,
			&handler.EnqueueRequestForObject{},
		)).
		Complete(r)
}

func (r *Reconciler) startEventLoop() {
	ticker := time.NewTicker(periodicity)

	go func() {
		r.events <- event.GenericEvent{
			Object: &metav1.PartialObjectMetadata{},
		}

		for range ticker.C {
			r.events <- event.GenericEvent{
				Object: &metav1.PartialObjectMetadata{},
			}
		}
	}()
}

func buildOwnerReference(ownerDeployment *appsv1.Deployment) *metav1.OwnerReference {
	if ownerDeployment == nil {
		return nil
	}

	return &metav1.OwnerReference{
		APIVersion:         appsv1.SchemeGroupVersion.String(),
		Kind:               "Deployment",
		Name:               ownerDeployment.GetName(),
		UID:                ownerDeployment.GetUID(),
		BlockOwnerDeletion: ptr.To(false),
		Controller:         ptr.To(false),
	}
}
