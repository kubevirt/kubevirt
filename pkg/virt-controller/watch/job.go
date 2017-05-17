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
	"kubevirt.io/kubevirt/pkg/logging"
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

func NewJobController(vmService services.VMService, recorder record.EventRecorder, clientSet *kubernetes.Clientset, restClient *rest.RESTClient, migrationQueue workqueue.RateLimitingInterface) (cache.Store, *kubecli.Controller) {
	selector := migrationJobSelector()
	lw := kubecli.NewListWatchFromClient(clientSet.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Pod{}, NewJobControllerDispatch(vmService, restClient, migrationQueue))
}

func NewJobControllerDispatch(vmService services.VMService, restClient *rest.RESTClient, migrationQueue workqueue.RateLimitingInterface) kubecli.ControllerDispatch {
	dispatch := JobDispatch{
		restClient:     restClient,
		vmService:      vmService,
		migrationQueue: migrationQueue,
	}
	var vmd kubecli.ControllerDispatch = &dispatch
	return vmd
}

type JobDispatch struct {
	restClient     *rest.RESTClient
	vmService      services.VMService
	migrationQueue workqueue.RateLimitingInterface
}

func (jd *JobDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {
	obj, exists, err := store.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
		return
	}
	if exists {
		job := obj.(*v1.Pod)

		//TODO Use the namespace from the Job and stop referencing the migration object
		migrationLabel := job.ObjectMeta.Labels[kvirtv1.MigrationLabel]
		migration, migrationExists, err := jd.vmService.FetchMigration(migrationLabel)
		if err != nil {
			queue.AddRateLimited(key)
			return
		}
		if !migrationExists {
			//REstart where the Migration has gone away.
			queue.Forget(key)
			return
		}
		migrationKey, err := cache.MetaNamespaceKeyFunc(migration)
		if err == nil {
			jd.migrationQueue.Add(migrationKey)
		} else {
			logger := logging.DefaultLogger().Object(migration)
			logger.Error().Reason(err).Msgf("Updating migration queue with %s failed.", migrationLabel)
			queue.AddRateLimited(key)
			return
		}
	}
}
