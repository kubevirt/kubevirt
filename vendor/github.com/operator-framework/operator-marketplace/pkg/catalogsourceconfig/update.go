package catalogsourceconfig

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewUpdateReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object that needs to be updated
func NewUpdateReconciler(log *logrus.Entry, client client.Client, cache Cache, targetChanged bool) Reconciler {
	return &updateReconciler{
		cache:         cache,
		client:        client,
		log:           log,
		targetChanged: targetChanged,
	}
}

// updateReconciler is an implementation of Reconciler interface that
// reconciles an CatalogSourceConfig object that needs to be updated.
type updateReconciler struct {
	cache         Cache
	client        client.Client
	log           *logrus.Entry
	targetChanged bool
}

// Reconcile reconciles an CatalogSourceConfig object that needs to be updated.
// It returns "Configuring" as the next desired phase.
func (r *updateReconciler) Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error) {
	out = in.DeepCopy()

	// The TargetNamespace of the CatalogSourceConfig object has changed
	if r.targetChanged {
		// Best case attempt at deleting the objects in the old TargetNamespace
		// If the csc is not cached we don't want to fail because there are
		// cases where we won't be able to find the objects.
		r.deleteObjects(in)
	}

	// Remove it from the cache so that it does not get picked up during
	// the "Configuring" phase
	r.cache.Evict(in)

	// Drop existing Status field so that reconciliation can start anew.
	out.Status = marketplace.CatalogSourceConfigStatus{}
	nextPhase = phase.GetNext(phase.Configuring)

	r.log.Info("Spec has changed, scheduling for configuring")

	return
}

// deleteObjects attempts to delete the CatalogSource in the old TargetNamespace.
// This is just an attempt, because if we are not aware of the the old namespace
// we will not be able to succeed.
func (r *updateReconciler) deleteObjects(in *marketplace.CatalogSourceConfig) {
	cachedCSCSpec, found := r.cache.Get(in)
	// This generally won't happen as it is because the cached Spec has changed
	// when we are in the update reconciler.
	// However, in the case that the cache is invalidated because of a restart,
	// we could simply be unsure if the targetNamespace changed. Ideally,  we
	// we should be using something like a direct reference to the old object
	// instead of an in memory cache, since when the service restarts we will be
	// unable to find the old object if the targetNamespace changed while we were
	// down. For now, we will just ignore it.
	if !found {
		r.log.Debugf("Cache invalid for csc %s, unable to validate if the targetNamespace changed.", in.Name)
		return
	}

	// Delete the CatalogSource
	name := in.Name
	catalogSource := new(CatalogSourceBuilder).
		WithMeta(name, cachedCSCSpec.TargetNamespace).
		CatalogSource()
	if err := r.client.Delete(context.TODO(), catalogSource); err != nil {
		r.log.Errorf("Error %v deleting CatalogSource %s/%s", err, cachedCSCSpec.TargetNamespace, name)
	}
	r.log.Infof("Deleted CatalogSource %s/%s", cachedCSCSpec.TargetNamespace, name)

	return
}
