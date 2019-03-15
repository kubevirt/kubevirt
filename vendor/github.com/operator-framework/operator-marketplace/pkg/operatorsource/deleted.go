package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDeletedReconciler returns a Reconciler that reconciles
// an OperatorSource that has been marked for deletion.
func NewDeletedReconciler(logger *log.Entry, datastore datastore.Writer, client client.Client) Reconciler {
	return &deletedReconciler{
		logger:    logger,
		datastore: datastore,
		client:    client,
	}
}

// deletedReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object that has been marked for deletion.
type deletedReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
	client    client.Client
}

// Reconcile reconciles an OperatorSource object that is marked for deletion.
// In the generic case, this is called when the OperatorSource has been marked
// for deletion. It removes all data related to this OperatorSource from the
// datastore, and it removes the OperatorSource finalizer from the object so
// that it can be cleaned up by the garbage collector.
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
func (r *deletedReconciler) Reconcile(ctx context.Context, in *marketplace.OperatorSource) (out *marketplace.OperatorSource, nextPhase *marketplace.Phase, err error) {
	out = in

	// Delete the operator source manifests.
	r.datastore.RemoveOperatorSource(out.UID)

	// Delete the owned CatalogSourceConfig
	err = r.deleteCreatedResources(ctx, in.Name, in.Namespace)
	if err != nil {
		// Something went wrong before we removed the finalizer, let's retry.
		nextPhase = phase.GetNextWithMessage(in.Status.CurrentPhase.Name, err.Error())
		return
	}

	// Remove the opsrc finalizer from the object.
	out.RemoveFinalizer()

	// Update the client. Since there is no phase shift, the transitioner
	// will not update it automatically like the normal phases.
	err = r.client.Update(context.TODO(), out)
	if err != nil {
		// An error happened on update. If it was transient, we will retry.
		// If not, and the finalizer was removed, then the delete will clean
		// the object up anyway. Let's set the next phase for a possible retry.
		nextPhase = phase.GetNextWithMessage(in.Status.CurrentPhase.Name, err.Error())
		return
	}

	r.logger.Info("Finalizer removed, now garbage collector will clean it up.")

	return
}

// Delete all resources owned by the operator source
func (r *deletedReconciler) deleteCreatedResources(ctx context.Context, name, namespace string) (err error) {
	labelMap := map[string]string{
		OpsrcOwnerNameLabel:      name,
		OpsrcOwnerNamespaceLabel: namespace,
	}
	labelSelector := labels.SelectorFromSet(labelMap)
	options := &client.ListOptions{LabelSelector: labelSelector}

	// Delete Catalog Source Configs
	catalogSourceConfigs := &marketplace.CatalogSourceConfigList{}
	err = r.client.List(ctx, options, catalogSourceConfigs)
	if err != nil {
		return err
	}

	for _, catalogSourceConfig := range catalogSourceConfigs.Items {
		r.logger.Infof("Removing catalogSourceConfig %s from namespace %s", catalogSourceConfig.Name, catalogSourceConfig.Namespace)
		err = r.client.Delete(ctx, &catalogSourceConfig)
		if err != nil {
			return err
		}
	}

	return
}
