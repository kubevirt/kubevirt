package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewConfiguringReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Configuring" phase.
func NewConfiguringReconciler(logger *log.Entry, datastore datastore.Writer, client client.Client) Reconciler {
	return &configuringReconciler{
		logger:    logger,
		datastore: datastore,
		client:    client,
		builder:   &CatalogSourceConfigBuilder{},
	}
}

// configuringReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Configuring" phase.
type configuringReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
	client    client.Client
	builder   *CatalogSourceConfigBuilder
}

// Reconcile reconciles an OperatorSource object that is in "Configuring" phase.
// It ensures that a corresponding CatalogSourceConfig object exists.
//
// in represents the original OperatorSource object received from the sdk
// and before reconciliation has started.
//
// out represents the OperatorSource object after reconciliation has completed
// and could be different from the original. The OperatorSource object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given OperatorSource
// object. If nil is returned, it implies that no phase transition is expected.
//
// Upon success, it returns "Succeeded" as the next and final desired phase.
// On error, the function returns "Failed" as the next desied phase
// and Message is set to appropriate error message.
//
// If the corresponding CatalogSourceConfig object already exists
// then no further action is taken.
func (r *configuringReconciler) Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Configuring {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	manifests := r.datastore.GetPackageIDsByOperatorSource(in.GetUID())

	cscCreate := new(CatalogSourceConfigBuilder).WithTypeMeta().
		WithNamespacedName(in.Namespace, in.Name).
		WithLabels(in.GetLabels()).
		WithSpec(in.Namespace, manifests, in.Spec.DisplayName, in.Spec.Publisher).
		WithOwnerLabel(in).
		CatalogSourceConfig()

	err = r.client.Create(ctx, cscCreate)
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		r.logger.Errorf("Unexpected error while creating CatalogSourceConfig: %s", err.Error())
		nextPhase = phase.GetNextWithMessage(phase.Configuring, err.Error())

		return
	}

	if err == nil {
		nextPhase = phase.GetNext(phase.Succeeded)
		r.logger.Info("CatalogSourceConfig object has been created successfully")

		return
	}

	// If we are here, the given CatalogSourceConfig object already exists.
	cscNamespacedName := types.NamespacedName{Name: in.Name, Namespace: in.Namespace}
	cscExisting := marketplace.CatalogSourceConfig{}
	err = r.client.Get(ctx, cscNamespacedName, &cscExisting)
	if err != nil {
		r.logger.Errorf("Unexpected error while getting CatalogSourceConfig: %s", err.Error())
		nextPhase = phase.GetNextWithMessage(phase.Configuring, err.Error())

		return
	}

	cscExisting.EnsureGVK()

	builder := CatalogSourceConfigBuilder{object: cscExisting}
	cscUpdate := builder.WithSpec(in.Namespace, manifests, in.Spec.DisplayName, in.Spec.Publisher).
		WithLabels(in.GetLabels()).
		WithOwnerLabel(in).
		CatalogSourceConfig()

	// Drop the status to force a CatalogSourceConfig update. This is to account
	// for the the scenario where a Quay namespace has changed without
	// app-registry repositories being added or removed but with existing
	// repositories being updated.
	cscUpdate.Status = marketplace.CatalogSourceConfigStatus{}

	err = r.client.Update(ctx, cscUpdate)
	if err != nil {
		r.logger.Errorf("Unexpected error while updating CatalogSourceConfig: %s", err.Error())
		nextPhase = phase.GetNextWithMessage(phase.Configuring, err.Error())

		return
	}

	r.logger.Info("CatalogSourceConfig object has been updated successfully")

	nextPhase = phase.GetNext(phase.Succeeded)
	return
}
