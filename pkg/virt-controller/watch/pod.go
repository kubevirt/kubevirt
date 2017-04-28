package watch

import (
	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(kubeapi.PodRunning))
	labelSelector, err := labels.Parse(corev1.AppLabel + " in (virt-launcher)")
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewPodController(vmCache cache.Store, recorder record.EventRecorder, clientset *kubernetes.Clientset, restClient *rest.RESTClient, vmService services.VMService, migrationQueue workqueue.RateLimitingInterface) (cache.Store, *kubecli.Controller) {

	selector := scheduledVMPodSelector()
	lw := kubecli.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Pod{}, NewPodControllerDispatch(vmCache, restClient, vmService, clientset, migrationQueue))
}

func NewPodControllerDispatch(vmCache cache.Store, restClient *rest.RESTClient, vmService services.VMService, clientset *kubernetes.Clientset, migrationQueue workqueue.RateLimitingInterface) kubecli.ControllerDispatch {
	dispatch := podDispatch{
		vmCache:        vmCache,
		restClient:     restClient,
		vmService:      vmService,
		clientset:      clientset,
		migrationQueue: migrationQueue,
	}
	return &dispatch
}

type podDispatch struct {
	vmCache        cache.Store
	restClient     *rest.RESTClient
	vmService      services.VMService
	clientset      *kubernetes.Clientset
	migrationQueue workqueue.RateLimitingInterface
}

func (pd *podDispatch) Execute(podStore cache.Store, podQueue workqueue.RateLimitingInterface, key interface{}) {
	// Fetch the latest Vm state from cache
	obj, exists, err := podStore.GetByKey(key.(string))

	if err != nil {
		podQueue.AddRateLimited(key)
		return
	}

	if !exists {
		// Do nothing
		return
	}
	pod := obj.(*v1.Pod)

	vmObj, exists, err := pd.vmCache.GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[corev1.DomainLabel])
	if err != nil {
		podQueue.AddRateLimited(key)
		return
	}
	if !exists {
		// Do nothing, the pod will timeout.
		return
	}
	vm := vmObj.(*corev1.VM)
	if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[corev1.VMUIDLabel]) {
		// Obviously the pod of an outdated VM object, do nothing
		return
	}
	if vm.Status.Phase == corev1.Scheduling {
		// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
		pd.handleScheduling(podQueue, key, vm, pod)
	} else if _, isMigrationPod := pod.Labels[corev1.MigrationLabel]; vm.Status.Phase == corev1.Running && isMigrationPod {
		pd.handleMigration(podStore, podQueue, key, vm, pod)
	}
	return
}

func (pd *podDispatch) handleScheduling(podQueue workqueue.RateLimitingInterface, key interface{}, vm *corev1.VM, pod *v1.Pod) {
	// deep copy the VM to allow manipulations
	vmCopy := corev1.VM{}
	model.Copy(&vmCopy, vm)

	vmCopy.Status.Phase = corev1.Pending
	// FIXME we store this in the metadata since field selctors are currently not working for TPRs
	if vmCopy.GetObjectMeta().GetLabels() == nil {
		vmCopy.ObjectMeta.Labels = map[string]string{}
	}
	vmCopy.ObjectMeta.Labels[corev1.NodeNameLabel] = pod.Spec.NodeName
	vmCopy.Status.NodeName = pod.Spec.NodeName
	// Update the VM
	logger := logging.DefaultLogger()
	if _, err := pd.vmService.PutVm(&vmCopy); err != nil {
		logger.V(3).Info().Msg("Enqueuing VM again.")
		podQueue.AddRateLimited(key)
		return
	}
	logger.Info().Msgf("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
}

func (pd *podDispatch) handleMigration(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}, vm *corev1.VM, pod *v1.Pod) {
	pd.migrationQueue.Add(v1.NamespaceDefault + "/" + pod.Labels[corev1.MigrationLabel])
}
