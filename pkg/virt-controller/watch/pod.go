package watch

import (
	"fmt"

	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelectionQuery := fmt.Sprintf("status.phase=%s", string(kubeapi.PodRunning))
	fieldSelector := fields.ParseSelectorOrDie(fieldSelectionQuery)
	labelSelectorQuery := fmt.Sprintf("!%s, %s in (virt-launcher)", string(kubev1.MigrationLabel), kubev1.AppLabel)
	labelSelector, err := labels.Parse(labelSelectorQuery)
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewPodController(vmCache cache.Store, recorder record.EventRecorder, clientset *kubernetes.Clientset, restClient *rest.RESTClient, vmService services.VMService) (cache.Store, *kubecli.Controller) {

	selector := scheduledVMPodSelector()
	lw := kubecli.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &k8sv1.Pod{}, NewPodControllerDispatch(vmCache, restClient, vmService, clientset))
}

func NewPodControllerDispatch(vmCache cache.Store, restClient *rest.RESTClient, vmService services.VMService, clientset *kubernetes.Clientset) kubecli.ControllerDispatch {
	dispatch := podDispatch{
		vmCache:    vmCache,
		restClient: restClient,
		vmService:  vmService,
		clientset:  clientset,
	}
	return &dispatch
}

type podDispatch struct {
	vmCache    cache.Store
	restClient *rest.RESTClient
	vmService  services.VMService
	clientset  *kubernetes.Clientset
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
	pod := obj.(*k8sv1.Pod)

	vmObj, exists, err := pd.vmCache.GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[kubev1.DomainLabel])
	if err != nil {
		podQueue.AddRateLimited(key)
		return
	}
	if !exists {
		// Do nothing, the pod will timeout.
		return
	}
	vm := vmObj.(*kubev1.VM)
	if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[kubev1.VMUIDLabel]) {
		// Obviously the pod of an outdated VM object, do nothing
		return
	}
	if vm.Status.Phase == kubev1.Scheduling {
		// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
		pd.handleScheduling(podQueue, key, vm, pod)
	}
	return
}

func (pd *podDispatch) handleScheduling(podQueue workqueue.RateLimitingInterface, key interface{}, vm *kubev1.VM, pod *k8sv1.Pod) {
	// deep copy the VM to allow manipulations
	vmCopy := kubev1.VM{}
	model.Copy(&vmCopy, vm)

	vmCopy.Status.Phase = kubev1.Pending
	// FIXME we store this in the metadata since field selctors are currently not working for TPRs
	if vmCopy.GetObjectMeta().GetLabels() == nil {
		vmCopy.ObjectMeta.Labels = map[string]string{}
	}
	vmCopy.ObjectMeta.Labels[kubev1.NodeNameLabel] = pod.Spec.NodeName
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
