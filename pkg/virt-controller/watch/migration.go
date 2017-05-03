package watch

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
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

func NewMigrationController(migrationService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient, clientset *kubernetes.Clientset) (cache.Store, *kubecli.Controller, *workqueue.RateLimitingInterface) {
	lw := cache.NewListWatchFromClient(restClient, "migrations", k8sv1.NamespaceDefault, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	store, controller := kubecli.NewController(lw, queue, &v1.Migration{}, NewMigrationControllerDispatch(migrationService, restClient, clientset))
	return store, controller, &queue
}

func NewMigrationControllerDispatch(vmService services.VMService, restClient *rest.RESTClient, clientset *kubernetes.Clientset) kubecli.ControllerDispatch {

	dispatch := MigrationDispatch{
		restClient: restClient,
		vmService:  vmService,
		clientset:  clientset,
	}
	return &dispatch
}

type MigrationDispatch struct {
	restClient *rest.RESTClient
	vmService  services.VMService
	clientset  *kubernetes.Clientset
}

func (md *MigrationDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {
	if err := md.execute(store, key.(string)); err != nil {
		logging.DefaultLogger().Info().Reason(err).Msgf("reenqueuing migration %v", key)
		queue.AddRateLimited(key)
	} else {
		logging.DefaultLogger().Info().V(4).Msgf("processed migration %v", key)
		queue.Forget(key)
	}
}

func (md *MigrationDispatch) execute(store cache.Store, key string) error {

	setMigrationPhase := func(migration *v1.Migration, phase v1.MigrationPhase) error {

		if migration.Status.Phase == phase {
			return nil
		}

		logger := logging.DefaultLogger().Object(migration)

		// Copy migration for future modifications
		migrationCopy, err := copy(migration)
		if err != nil {
			logger.Error().Reason(err).Msg("could not copy migration object")
			return err
		}

		migrationCopy.Status.Phase = phase
		// TODO indicate why it was set to failed
		err = md.vmService.UpdateMigration(migrationCopy)
		if err != nil {
			logger.Error().Reason(err).Msgf("updating migration state failed: %v ", err)
			return err
		}
		return nil
	}

	setMigrationFailed := func(mig *v1.Migration) error {
		return setMigrationPhase(mig, v1.MigrationFailed)
	}

	obj, exists, err := store.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	var migration *v1.Migration = obj.(*v1.Migration)
	logger := logging.DefaultLogger().Object(migration)

	vm, exists, err := md.vmService.FetchVM(migration.Spec.Selector.Name)
	if err != nil {
		logger.Error().Reason(err).Msgf("fetching the vm %s failed", migration.Spec.Selector.Name)
		return err
	}

	if !exists {
		logger.Info().Msgf("VM with name %s does not exist, marking migration as failed", migration.Spec.Selector.Name)
		if err = setMigrationFailed(migration); err != nil {
			return err
		}
		return nil
	}

	switch migration.Status.Phase {
	case v1.MigrationUnknown:
		if vm.Status.Phase != v1.Running {
			logger.Error().Msgf("VM with name %s is in state %s, no migration possible. Marking migration as failed", vm.GetObjectMeta().GetName(), vm.Status.Phase)
			if err = setMigrationFailed(migration); err != nil {
				return err
			}
			return nil
		}

		if err := mergeConstraints(migration, vm); err != nil {
			logger.Error().Reason(err).Msg("merging Migration and VM placement constraints failed.")
			return err
		}
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			return err
		}

		numOfPods, targetPod := investigateTargetPodSituation(migration, podList)

		if targetPod == nil {
			if numOfPods > 1 {
				logger.Error().Msg("another migration seems to be in progress, marking Migration as failed")
				// Another migration is currently going on
				if err = setMigrationFailed(migration); err != nil {
					return err
				}
				return nil
			} else if numOfPods == 1 {
				// We need to start a migration target pod
				// TODO, this detection is not optimal, it can lead to strange situations
				err := md.vmService.CreateMigrationTargetPod(migration, vm)
				if err != nil {
					logger.Error().Reason(err).Msg("creating a migration target pod failed")
					return err
				}
			}
		} else {
			if targetPod.Status.Phase == k8sv1.PodFailed {
				logger.Error().Msg("migration target pod is in failed state")
				if err = setMigrationFailed(migration); err != nil {
					return err
				}
				return nil
			}
			// Unlikely to hit this case, but prevents erroring out
			// if we re-enter this loop
			logger.Info().Msgf("migration appears to be set up, but was not set to %s", v1.MigrationScheduled)
		}
		err = setMigrationPhase(migration, v1.MigrationScheduled)
		if err != nil {
			return err
		}
		return nil
	case v1.MigrationScheduled:
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			return err
		}

		_, targetPod := investigateTargetPodSituation(migration, podList)

		if targetPod == nil {
			logger.Error().Msg("migration target pod does not exist or is an end state")
			if err = setMigrationFailed(migration); err != nil {
				return err
			}
			return nil
		}
		// Migration has been scheduled but no update on the status has been recorded
		err = setMigrationPhase(migration, v1.MigrationRunning)
		if err != nil {
			return err
		}
		return nil
	case v1.MigrationRunning:
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			return err
		}
		_, targetPod := investigateTargetPodSituation(migration, podList)
		if targetPod == nil {
			logger.Error().Msg("migration target pod does not exist or is in an end state")
			if err = setMigrationFailed(migration); err != nil {
				return err
			}
			return nil
		}
		switch targetPod.Status.Phase {
		case k8sv1.PodRunning:
			break
		case k8sv1.PodSucceeded, k8sv1.PodFailed:
			logger.Error().Msgf("migration target pod is in end state %s", targetPod.Status.Phase)
			if err = setMigrationFailed(migration); err != nil {
				return err
			}
			return nil
		default:
			//Not requeuing, just not far enough along to proceed
			logger.Info().V(3).Msg("target Pod not running yet")
			return nil
		}

		if vm.Status.MigrationNodeName != targetPod.Spec.NodeName {
			vm.Status.Phase = v1.Migrating
			vm.Status.MigrationNodeName = targetPod.Spec.NodeName
			if _, err = md.vmService.PutVm(vm); err != nil {
				logger.Error().Reason(err).Msgf("failed to update VM to state %s", v1.Migrating)
				return err
			}
		}

		// Let's check if the job already exists, it can already exist in case we could not update the VM object in a previous run
		migrationPod, exists, err := md.vmService.GetMigrationJob(migration)

		if err != nil {
			logger.Error().Reason(err).Msg("Checking for an existing migration job failed.")
			return err
		}

		if !exists {
			sourceNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.NodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("fetching source node %s failed", vm.Status.NodeName)
				return err
			}
			targetNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.MigrationNodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("fetching target node %s failed", vm.Status.MigrationNodeName)
				return err
			}

			if err := md.vmService.StartMigration(migration, vm, sourceNode, targetNode, targetPod); err != nil {
				logger.Error().Reason(err).Msg("Starting the migration job failed.")
				return err
			}
			return nil
		}

		// FIXME, the final state updates must come from virt-handler
		switch migrationPod.Status.Phase {
		case k8sv1.PodFailed:
			vm.Status.Phase = v1.Running
			vm.Status.MigrationNodeName = ""
			if _, err = md.vmService.PutVm(vm); err != nil {
				return err
			}
			if err = setMigrationFailed(migration); err != nil {
				return err
			}
		case k8sv1.PodSucceeded:
			vm.Status.NodeName = targetPod.Spec.NodeName
			vm.Status.MigrationNodeName = ""
			vm.Status.Phase = v1.Running
			if _, err = md.vmService.PutVm(vm); err != nil {
				logger.Error().Reason(err).Msg("updating the VM failed.")
				return err
			}
			if err = setMigrationPhase(migration, v1.MigrationSucceeded); err != nil {
				return err
			}
		}
	}
	return nil
}

func copy(migration *v1.Migration) (*v1.Migration, error) {
	obj, err := kubeapi.Scheme.Copy(migration)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Migration), nil
}

// Returns the number of  running pods and if a pod for exactly that migration is currently running
func investigateTargetPodSituation(migration *v1.Migration, podList *k8sv1.PodList) (int, *k8sv1.Pod) {
	var targetPod *k8sv1.Pod = nil
	for _, pod := range podList.Items {
		if pod.Labels[v1.MigrationUIDLabel] == string(migration.GetObjectMeta().GetUID()) {
			targetPod = &pod
			break
		}
	}
	return len(podList.Items), targetPod
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
