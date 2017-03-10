package watch

import (
	kubeapi "k8s.io/client-go/pkg/api"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
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
	fieldSelector := fields.Everything()
	labelSelector, err := labels.Parse(kvirtv1.DomainLabel)
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewJobController(vmService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {
	selector := migrationJobSelector()
	lw := kubecli.NewListWatchFromClient(restClient, "jobs", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	return NewJobControllerWithListWatch(vmService, recorder, lw, restClient)
}

func NewJobControllerWithListWatch(vmService services.VMService, _ record.EventRecorder, lw cache.ListerWatcher, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &batchv1.Job{}, func(store cache.Store, queue workqueue.RateLimitingInterface) bool {
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
			var job *batchv1.Job = obj.(*batchv1.Job)

			if job.Status.Succeeded < 1 {
				//Job did not succeed, do not update the vm
				return true
			}

			name := job.ObjectMeta.Labels["vmname"]
			vm, err := vmService.FetchVM(name)
			if err != nil {
				//TODO proper error handling
				queue.AddRateLimited(key)
				return true
			}
			vm.Status.Phase = kvirtv1.Running
			putVm(vm, restClient, nil)
		}
		return true
	})
}
