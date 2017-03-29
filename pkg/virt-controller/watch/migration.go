package watch

import (
	"fmt"
	kubeapi "k8s.io/client-go/pkg/api"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"k8s.io/client-go/kubernetes"
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

	setMigrationPhase := func(migration *v1.Migration, phase v1.MigrationPhase) error {

		if migration.Status.Phase == phase {
			return nil
		}

		logger := logging.DefaultLogger().Object(migration)

		// Copy migration for future modifications
		migrationCopy, err := copy(migration)
		if err != nil {
			logger.Error().Reason(err).Msg("could not copy migration object")
			queue.AddRateLimited(key)
			return nil
		}

		migrationCopy.Status.Phase = phase
		// TODO indicate why it was set to failed
		err = md.vmService.UpdateMigration(migrationCopy)
		if err != nil {
			logger.Error().Reason(err).Msgf("updating migration state failed: %v ", err)
			queue.AddRateLimited(key)
			return err
		}
		queue.Forget(key)
		return nil
	}

	setMigrationFailed := func(mig *v1.Migration) {
		setMigrationPhase(mig, v1.MigrationFailed)
	}

	obj, exists, err := store.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
		return
	}
	if !exists {
		queue.Forget(key)
		return
	}

	var migration *v1.Migration = obj.(*v1.Migration)
	logger := logging.DefaultLogger().Object(migration)

	vm, exists, err := md.vmService.FetchVM(migration.Spec.Selector.Name)
	if err != nil {
		logger.Error().Reason(err).Msgf("fetching the vm %s failed", migration.Spec.Selector.Name)
		queue.AddRateLimited(key)
		return
	}

	if !exists {
		logger.Info().Msgf("VM with name %s does not exist, marking migration as failed", migration.Spec.Selector.Name)
		setMigrationFailed(migration)
		return
	}

	switch migration.Status.Phase {
	case v1.MigrationUnknown:
		// Fetch vm which we want to migrate

		if vm.Status.Phase != v1.Running {
			logger.Error().Msgf("VM with name %s is in state %s, no migration possible. Marking migration as failed", vm.GetObjectMeta().GetName(), vm.Status.Phase)
			setMigrationFailed(migration)
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

		numOfPods, targetPod := investigateTargetPodSituation(migration, podList)

		if targetPod == nil {
			if numOfPods > 1 {
				logger.Error().Reason(err).Msg("another migration seems to be in progress, marking Migration as failed")
				// Another migration is currently going on
				setMigrationFailed(migration)
				return
			} else if numOfPods == 1 {
				// We need to start a migration target pod
				// TODO, this detection is not optimal, it can lead to strange situations
				err := md.vmService.CreateMigrationTargetPod(migration, vm)
				if err != nil {
					logger.Error().Reason(err).Msg("creating a migration target pod failed")
					queue.AddRateLimited(key)
					return
				}
			}
		} else {
			if targetPod.Status.Phase == k8sv1.PodFailed {
				setMigrationPhase(migration, v1.MigrationFailed)
				queue.Forget(key)
				return
			}
			// Unlikely to hit this case, but prevents erroring out
			// if we re-enter this loop
			logger.Info().Msgf("Migration appears to be set up, but was not set to %s.", v1.MigrationScheduled)
		}
		err = setMigrationPhase(migration, v1.MigrationScheduled)
		if err != nil {
			return
		}
	case v1.MigrationScheduled:
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			queue.AddRateLimited(key)
			return
		}

		_, targetPod := investigateTargetPodSituation(migration, podList)

		if targetPod == nil {
			setMigrationFailed(migration)
			queue.Forget(key)
			return
		}
		// Migration has been scheduled but no update on the status has been recorded
		err = setMigrationPhase(migration, v1.MigrationRunning)
		if err != nil {
			queue.Forget(key)
			return
		}
	case v1.MigrationRunning:
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			queue.AddRateLimited(key)
			return
		}
		_, targetPod := investigateTargetPodSituation(migration, podList)
		if targetPod == nil {
			setMigrationFailed(migration)
			queue.Forget(key)
			return
		}
		switch targetPod.Status.Phase {
		case k8sv1.PodRunning:
			break
		case k8sv1.PodSucceeded, k8sv1.PodFailed:
			setMigrationFailed(migration)
			return
		default:
			//Not requeuing, just not far enough along to proceed
			queue.Forget(key)
			return
		}

		if vm.Status.MigrationNodeName != targetPod.Spec.NodeName {
			vm.Status.Phase = v1.Migrating
			vm.Status.MigrationNodeName = targetPod.Spec.NodeName
			if err = md.updateVm(vm); err != nil {
				queue.AddRateLimited(key)
			}
		}

		// Let's check if the job already exists, it can already exist in case we could not update the VM object in a previous run
		migrationPod, exists, err := md.vmService.GetMigrationJob(migration)

		if err != nil {
			logger.Error().Reason(err).Msg("Checking for an existing migration job failed.")
			queue.AddRateLimited(key)
			return
		}

		if !exists {
			sourceNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.NodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("Fetching source node %s failed.", vm.Status.NodeName)
				queue.AddRateLimited(key)
				return
			}
			targetNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.MigrationNodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("Fetching target node %s failed.", vm.Status.MigrationNodeName)
				queue.AddRateLimited(key)
				return
			}

			if err := md.vmService.StartMigration(migration, vm, sourceNode, targetNode, targetPod); err != nil {
				logger.Error().Reason(err).Msg("Starting the migration job failed.")
				queue.AddRateLimited(key)
				return
			}
			queue.Forget(key)
			return
		}

		switch migrationPod.Status.Phase {
		case k8sv1.PodFailed:
			vm.Status.Phase = v1.Running
			vm.Status.MigrationNodeName = ""
			if err = md.updateVm(vm); err != nil {
				queue.AddRateLimited(key)
			} else {
				queue.Forget(key)
			}
			setMigrationPhase(migration, v1.MigrationFailed)
			return
		case k8sv1.PodSucceeded:
			vm.Status.NodeName = targetPod.Spec.NodeName
			vm.Status.MigrationNodeName = ""
			vm.Status.Phase = v1.Running
			if err = md.updateVm(vm); err != nil {
				queue.AddRateLimited(key)
			} else {
				queue.Forget(key)
			}
			setMigrationPhase(migration, v1.MigrationSucceeded)
			return
		}
	}
	queue.Forget(key)
	return
}
func (md *MigrationDispatch) updateVm(vmCopy *v1.VM) error {
	if _, err := md.vmService.PutVm(vmCopy); err != nil {
		logger := logging.DefaultLogger().Object(vmCopy)
		logger.V(3).Info().Msg("Enqueuing VM again.")
		return err
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
