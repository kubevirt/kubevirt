package observability

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/alertmanager"
)

var (
	log         = logf.Log.WithName("controller_observability")
	periodicity = 1 * time.Hour
)

type Reconciler struct {
	config *rest.Config
	events chan event.GenericEvent

	amApi *alertmanager.Api
}

func (r *Reconciler) Reconcile(_ context.Context, _ ctrl.Request) (ctrl.Result, error) {
	log.Info("Reconciling Observability")

	if err := r.ensurePodDisruptionBudgetAtLimitIsSilenced(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func NewReconciler(config *rest.Config) *Reconciler {
	return &Reconciler{
		config: config,
		events: make(chan event.GenericEvent, 1),
	}
}

func SetupWithManager(mgr ctrl.Manager) error {
	log.Info("Setting up controller")

	r := NewReconciler(mgr.GetConfig())
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
