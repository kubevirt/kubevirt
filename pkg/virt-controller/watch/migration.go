package watch

import (
	"fmt"
	"github.com/jeevatkm/go-model"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func NewMigrationController(migrationService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {
	lw := cache.NewListWatchFromClient(restClient, "migrations", kubeapi.NamespaceDefault, fields.Everything())
	return NewMigrationControllerWithListWatch(migrationService, recorder, lw, restClient)
}

func NewMigrationControllerWithListWatch(migrationService services.VMService, _ record.EventRecorder, lw cache.ListerWatcher, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Migration{}, func(store cache.Store, queue workqueue.RateLimitingInterface) bool {
		key, quit := queue.Get()
		if quit {
			return false
		}
		defer queue.Done(key)

		// Fetch the latest Migration state from cache
		obj, exists, err := store.GetByKey(key.(string))

		if err != nil {
			queue.AddRateLimited(key)
			return true
		}
		if exists {
			var migration *v1.Migration = obj.(*v1.Migration)
			if migration.Status.Phase == v1.MigrationUnknown {
				migrationCopy := copyMigration(migration)
				logger := logging.DefaultLogger().Object(&migrationCopy)
				if err := StartMigrationTargetPod(migrationService, &migrationCopy); err != nil {
					handleStartMigrationError(logger, err, migrationService, migrationCopy)
				}
			}
		} else {
			cleanupOldMigration(key, queue, migrationService)
		}
		return true
	})
}
func cleanupOldMigration(key interface{}, queue workqueue.RateLimitingInterface, migrationService services.VMService) {
	var migration *v1.Migration
	_, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		// TODO do something more smart here
		queue.AddRateLimited(key)
	} else {
		migration = v1.NewMigrationReferenceFromName(name)
		err = migrationService.DeleteMigration(migration)
		logger := logging.DefaultLogger().Object(migration)

		if err != nil {
			logger.Error().Reason(err).Msg("Deleting VM target Pod failed.")
		}
		logger.Info().Msg("Deleting VM target Pod succeeded.")
	}
}

func handleStartMigrationError(logger *logging.FilteredLogger, err error, migrationService services.VMService, migrationCopy v1.Migration) {
	logger.Error().Reason(err).Msg("Defining a target pod for the Migration.")
	pl, err := migrationService.GetRunningMigrationPods(&migrationCopy)
	if err != nil {
		logger.Error().Reason(err).Msg("Getting running Pod for the Migration failed.")
		return
	}
	for _, p := range pl.Items {
		if p.GetObjectMeta().GetLabels()["kubevirt.io/vmUID"] == string(migrationCopy.GetObjectMeta().GetUID()) {
			// Pod from incomplete initialization detected, cleaning up
			logger.Error().Msgf("Found orphan pod with name '%s' for Migration.", p.GetName())
			err = migrationService.DeleteMigration(&migrationCopy)
			if err != nil {
				logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for Migration failed.", p.GetName())
				break
			}
		} else {
			// TODO virt-api should make sure this does not happen. For now don't ask and clean up.
			// Pod from old VM object detected,
			logger.Error().Msgf("Found orphan pod with name '%s' for deleted VM.", p.GetName())

			err = migrationService.DeleteMigration(&migrationCopy)
			if err != nil {
				logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for Migration failed.", p.GetName())
				break
			}
		}
	}

}
func copyMigration(migration *v1.Migration) v1.Migration {
	migrationCopy := v1.Migration{}
	model.Copy(&migrationCopy, migration)
	return migrationCopy
}

func StartMigrationTargetPod(v services.VMService, migration *v1.Migration) error {
	precond.MustNotBeNil(migration)
	precond.MustNotBeEmpty(migration.ObjectMeta.Name)
	precond.MustNotBeEmpty(string(migration.ObjectMeta.UID))

	vm, err := v.FetchVM(migration.Spec.MigratingVMName)
	if err != nil {
		migration.Status.Phase = v1.MigrationFailed
		err2 := v.UpdateMigration(migration)
		if err2 != nil {
			return err2
		}
		// Report the error with the migration in the controller log
		return err
	}

	podList, err := v.GetRunningVMPods(vm)
	if err != nil {
		return err
	}

	if len(podList.Items) < 1 {
		return middleware.NewResourceConflictError(fmt.Sprintf("VM %s Pod does not exist", vm.GetObjectMeta().GetName()))
	}

	// If there are more than one pod in other states than Succeeded or Failed we can't go on
	if len(podList.Items) > 1 {
		return middleware.NewResourceConflictError(fmt.Sprintf("VM %s Pod is already migrating", vm.GetObjectMeta().GetName()))
	}

	//TODO:  detect collisions
	for k, v := range migration.Spec.DestinationNodeSelector {
		vm.Spec.NodeSelector[k] = v
	}

	err = v.SetupMigration(migration, vm)

	// Report the result of the `Create` call
	return err
}
