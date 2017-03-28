package watch

import (
	"fmt"
	kubeapi "k8s.io/client-go/pkg/api"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func NewMigrationController(migrationService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {
	lw := cache.NewListWatchFromClient(restClient, "migrations", k8sv1.NamespaceDefault, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Migration{}, NewMigrationControllerDispatch(migrationService))
}

func NewMigrationControllerDispatch(vmService services.VMService) kubecli.ControllerDispatch {

	dispatch := MigrationDispatch{
		vmService: vmService,
	}
	return &dispatch
}

type MigrationDispatch struct {
	vmService services.VMService
}

func (md *MigrationDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {

	setMigrationFailed := func(migrationCopy *v1.Migration) {

		logger := logging.DefaultLogger().Object(migrationCopy)
		migrationCopy.Status.Phase = v1.MigrationFailed
		// TODO indicate why it was set to failed
		err := md.vmService.UpdateMigration(migrationCopy)
		if err != nil {
			logger.Error().Reason(err).Msgf("updating migration state failed: %v ", err)
			queue.AddRateLimited(key)
			return
		}
		queue.Forget(key)
	}

	// Fetch the latest Migration state from cache
	// error is always nil
	obj, exists, _ := store.GetByKey(key.(string))

	if exists {
		var migration *v1.Migration = obj.(*v1.Migration)
		logger := logging.DefaultLogger().Object(migration)

		if migration.Status.Phase == v1.MigrationUnknown {
			// Copy migration for future modifications
			migrationCopy, err := copy(migration)
			if err != nil {
				logger.Error().Reason(err).Msg("could not copy migration object")
				queue.AddRateLimited(key)
				return
			}
			// Fetch vm which we want to migrate
			vm, exists, err := md.vmService.FetchVM(migration.Spec.Selector.Name)
			if err != nil {
				logger.Error().Reason(err).Msgf("fetching the vm %s failed", migration.Spec.Selector.Name)
				queue.AddRateLimited(key)
				return
			}
			if !exists {
				logger.Info().Msgf("VM with name %s does not exist, marking migration as failed", migration.Spec.Selector.Name)
				setMigrationFailed(migrationCopy)
				return
			}
			if vm.Status.Phase != v1.Running {
				logger.Error().Msgf("VM with name %s is in state %s, no migration possible. Marking migration as failed", vm.GetObjectMeta().GetName(), vm.Status.Phase)
				setMigrationFailed(migrationCopy)
				return
			}

			if err := mergeConstraints(migration, vm); err != nil {
				logger.Error().Reason(err).Msg("merging Migration and VM placement constraints failed.")
				queue.AddRateLimited(key)
				return
			}
			podList, err := md.vmService.GetRunningVMPods(vm)
			if err != nil {
				logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
				queue.AddRateLimited(key)
				return
			}

			numOfPods, migrationPodExists := investigateTargetPodSituation(migration, podList)
			if numOfPods > 1 && !migrationPodExists {
				logger.Error().Reason(err).Msg("another migration seems to be in progress, marking Migration as failed")
				// Another migration is currently going on
				setMigrationFailed(migrationCopy)
				return
			} else if numOfPods == 1 && !migrationPodExists {
				// We need to start a migration target pod
				// TODO, this detection is not optimal, it can lead to strange situations
				err := md.vmService.SetupMigration(migration, vm)
				if err != nil {
					logger.Error().Reason(err).Msg("creating a migration target node failed")
					queue.AddRateLimited(key)
					return
				}
			}
			logger.Error().Msg("another migration seems to be in progress, marking Migration as failed")
			migrationCopy.Status.Phase = v1.MigrationInProgress
			// TODO indicate when this has happened
			err = md.vmService.UpdateMigration(migrationCopy)
			if err != nil {
				logger.Error().Reason(err).Msgf("updating migration state failed : %v ", err)
				queue.AddRateLimited(key)
				return
			}
		}
	}

	queue.Forget(key)
	return
}

func copy(migration *v1.Migration) (*v1.Migration, error) {
	obj, err := kubeapi.Scheme.Copy(migration)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Migration), nil
}

// Returns the number of  running pods and if a pod for exactly that migration is currently running
func investigateTargetPodSituation(migration *v1.Migration, podList *k8sv1.PodList) (int, bool) {
	podExists := false
	for _, pod := range podList.Items {
		if pod.Labels[v1.MigrationUIDLabel] == string(migration.GetObjectMeta().GetUID()) {
			podExists = true
		}
	}
	return len(podList.Items), podExists
}

func mergeConstraints(migration *v1.Migration, vm *v1.VM) error {

	merged := map[string]string{}
	for k, v := range vm.Spec.NodeSelector {
		merged[k] = v
	}
	conflicts := []string{}
	for k, v := range migration.Spec.NodeSelector {
		val, exists := vm.Spec.NodeSelector[k]
		if exists && val != v {
			conflicts = append(conflicts, k)
		} else {
			merged[k] = v
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("Conflicting node selectors: %v", conflicts)
	}
	vm.Spec.NodeSelector = merged
	return nil
}
