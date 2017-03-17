package watch

import (
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	kvirtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func migrationJobSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(v1.PodPending) +
			",status.phase!=" + string(v1.PodRunning) +
			",status.phase!=" + string(v1.PodUnknown))
	labelSelector, err := labels.Parse(kvirtv1.AppLabel + "=migration," + kvirtv1.DomainLabel + "," + kvirtv1.MigrationLabel)
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewJobController(vmService services.VMService, recorder record.EventRecorder, clientSet *kubernetes.Clientset, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {
	selector := migrationJobSelector()
	lw := kubecli.NewListWatchFromClient(clientSet.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	return NewJobControllerWithListWatch(vmService, recorder, lw, restClient)
}

func NewJobControllerWithListWatch(vmService services.VMService, _ record.EventRecorder, lw cache.ListerWatcher, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Pod{}, NewJobControllerFunction(vmService, restClient))
}

func NewJobControllerFunction(vmService services.VMService, restClient *rest.RESTClient) kubecli.ControllerFunc {
	return func(store cache.Store, queue workqueue.RateLimitingInterface) bool {
		key, quit := queue.Get()
		if quit {
			return false
		}
		defer queue.Done(key)

		// Fetch the latest Job state from cache
		obj, exists, err := store.GetByKey(key.(string))

		if err != nil {
			queue.AddRateLimited(key)
			return true
		}
		if exists {
			job := obj.(*v1.Pod)

			name := job.ObjectMeta.Labels[kvirtv1.DomainLabel]
			vm, vmExists, err := vmService.FetchVM(name)
			if err != nil {
				queue.AddRateLimited(key)
				return true
			}

			// TODO at the end, only virt-handler can decide for all migration types if a VM successfully migrated to it (think about p2p2 migrations)
			// For now we use a managed migration
			if vmExists && vm.Status.Phase == kvirtv1.Migrating {
				vm.Status.Phase = kvirtv1.Running
				if job.Status.Phase == v1.PodSucceeded {
					vm.ObjectMeta.Labels[kvirtv1.NodeNameLabel] = vm.Status.MigrationNodeName
					vm.Status.NodeName = vm.Status.MigrationNodeName
				}
				vm.Status.MigrationNodeName = ""
				_, err := putVm(vm, restClient, nil)
				if err != nil {
					queue.AddRateLimited(key)
					return true
				}
			}

			migration, migrationExists, err := vmService.FetchMigration(job.ObjectMeta.Labels[kvirtv1.MigrationLabel])
			if err != nil {
				queue.AddRateLimited(key)
				return true
			}

			if migrationExists {
				if migration.Status.Phase != kvirtv1.MigrationSucceeded && migration.Status.Phase != kvirtv1.MigrationFailed {
					if job.Status.Phase == v1.PodSucceeded {
						migration.Status.Phase = kvirtv1.MigrationSucceeded
					} else {
						migration.Status.Phase = kvirtv1.MigrationFailed
					}
					err := vmService.UpdateMigration(migration)
					if err != nil {
						queue.AddRateLimited(key)
						return true
					}
				}
			}
		}
		return true
	}
}
