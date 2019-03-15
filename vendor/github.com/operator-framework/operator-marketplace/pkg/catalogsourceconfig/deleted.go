package catalogsourceconfig

import (
	"context"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDeletedReconciler returns a Reconciler that reconciles
// a CatalogSourceConfig that has been marked for deletion.
func NewDeletedReconciler(logger *log.Entry, cache Cache, client client.Client) Reconciler {
	return &deletedReconciler{
		logger: logger,
		cache:  cache,
		client: client,
	}
}

// deletedReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object that has been marked for deletion.
type deletedReconciler struct {
	logger *log.Entry
	cache  Cache
	client client.Client
}

// Reconcile reconciles a CatalogSourceConfig object that is marked for deletion.
// In the generic case, this is called when the CatalogSourceConfig has been marked
// for deletion. It removes all data related to this CatalogSourceConfig from the
// datastore, and it removes the CatalogSourceConfig finalizer from the object so
// that it can be cleaned up by the garbage collector.
//
// in represents the original CatalogSourceConfig object received from the sdk
// and before reconciliation has started.
//
// out represents the CatalogSourceConfig object after reconciliation has completed
// and could be different from the original. The CatalogSourceConfig object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given CatalogSourceConfig
// object. If nil is returned, it implies that no phase transition is expected.
func (r *deletedReconciler) Reconcile(ctx context.Context, in *marketplace.CatalogSourceConfig) (out *marketplace.CatalogSourceConfig, nextPhase *marketplace.Phase, err error) {
	out = in

	// Evict the catalogsourceconfig data from the cache.
	r.cache.Evict(out)

	// Delete all created resources
	err = r.deleteCreatedResources(ctx, in.Name, in.Namespace, in.Spec.TargetNamespace)
	if err != nil {
		// Something went wrong before we removed the finalizer, let's retry.
		nextPhase = phase.GetNextWithMessage(in.Status.CurrentPhase.Name, err.Error())
		return
	}

	// Remove the csc finalizer from the object.
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

// Delete all resources owned by the catalog source config
func (r *deletedReconciler) deleteCreatedResources(ctx context.Context, name, namespace, targetNamespace string) error {
	allErrors := []error{}
	labelMap := map[string]string{
		CscOwnerNameLabel:      name,
		CscOwnerNamespaceLabel: namespace,
	}
	labelSelector := labels.SelectorFromSet(labelMap)
	catalogSourceOptions := &client.ListOptions{LabelSelector: labelSelector}
	catalogSourceOptions.InNamespace(targetNamespace)
	namespacedResourceOptions := &client.ListOptions{LabelSelector: labelSelector}
	namespacedResourceOptions.InNamespace(namespace)

	// Delete Catalog Sources
	catalogSources := &olm.CatalogSourceList{}
	err := r.client.List(ctx, catalogSourceOptions, catalogSources)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, catalogSource := range catalogSources.Items {
		r.logger.Infof("Removing catalogSource %s from namespace %s", catalogSource.Name, catalogSource.Namespace)
		err := r.client.Delete(ctx, &catalogSource)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Delete Services
	services := &core.ServiceList{}
	err = r.client.List(ctx, namespacedResourceOptions, services)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, service := range services.Items {
		r.logger.Infof("Removing service %s from namespace %s", service.Name, service.Namespace)
		err := r.client.Delete(ctx, &service)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Delete Deployments
	deployments := &apps.DeploymentList{}
	err = r.client.List(ctx, namespacedResourceOptions, deployments)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, deployment := range deployments.Items {
		r.logger.Infof("Removing deployment %s from namespace %s", deployment.Name, deployment.Namespace)
		err := r.client.Delete(ctx, &deployment)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Delete Role Bindings
	roleBindings := &rbac.RoleBindingList{}
	err = r.client.List(ctx, namespacedResourceOptions, roleBindings)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, roleBinding := range roleBindings.Items {
		r.logger.Infof("Removing roleBinding %s from namespace %s", roleBinding.Name, roleBinding.Namespace)
		err := r.client.Delete(ctx, &roleBinding)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Delete Roles
	roles := &rbac.RoleList{}
	err = r.client.List(ctx, namespacedResourceOptions, roles)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, role := range roles.Items {
		r.logger.Infof("Removing role %s from namespace %s", role.Name, role.Namespace)
		err := r.client.Delete(ctx, &role)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	// Delete Service Accounts
	serviceAccounts := &core.ServiceAccountList{}
	err = r.client.List(ctx, namespacedResourceOptions, serviceAccounts)
	if err != nil {
		allErrors = append(allErrors, err)
	}

	for _, serviceAccount := range serviceAccounts.Items {
		r.logger.Infof("Removing serviceAccount %s from namespace %s", serviceAccount.Name, serviceAccount.Namespace)
		err := r.client.Delete(ctx, &serviceAccount)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return utilerrors.NewAggregate(allErrors)
}
