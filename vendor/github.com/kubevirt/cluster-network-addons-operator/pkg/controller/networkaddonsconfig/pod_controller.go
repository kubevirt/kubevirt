package networkaddonsconfig

import (
	"log"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/controller/statusmanager"
)

var resyncPeriod = 5 * time.Minute

// newPodReconciler returns a new reconcile.Reconciler
func newPodReconciler(statusManager *statusmanager.StatusManager) *ReconcilePods {
	return &ReconcilePods{
		statusManager: statusManager,
	}
}

var _ reconcile.Reconciler = &ReconcilePods{}

// ReconcilePods watches for updates to specified resources and then updates its StatusManager
type ReconcilePods struct {
	statusManager *statusmanager.StatusManager
	resources     []types.NamespacedName
}

func (r *ReconcilePods) SetResources(resources []types.NamespacedName) {
	r.resources = resources
}

// Reconcile updates the NetworkAddonsConfig.Status to match the current state of the
// watched Deployments/DaemonSets
func (r *ReconcilePods) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	for _, name := range r.resources {
		if name.Namespace == request.Namespace && name.Name == request.Name {
			log.Printf("Reconciling update to %s/%s\n", request.Namespace, request.Name)
			r.statusManager.SetFromPods()
			return reconcile.Result{RequeueAfter: resyncPeriod}, nil
		}
	}
	return reconcile.Result{}, nil
}
