package watch

import (
	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
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
)

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(kubeapi.PodRunning))
	labelSelector, err := labels.Parse(corev1.AppLabel + " in (virt-launcher)")
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func NewPodController(vmCache cache.Store, recorder record.EventRecorder, clientset *kubernetes.Clientset, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {

	selector := scheduledVMPodSelector()
	lw := kubecli.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	return NewPodControllerWithListWatch(vmCache, recorder, lw, restClient)
}

func NewPodControllerWithListWatch(vmCache cache.Store, _ record.EventRecorder, lw cache.ListerWatcher, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.Pod{}, func(store cache.Store, queue workqueue.RateLimitingInterface) bool {
		key, quit := queue.Get()
		if quit {
			return false
		}
		defer queue.Done(key)

		// Fetch the latest Vm state from cache
		obj, exists, err := store.GetByKey(key.(string))

		if err != nil {
			queue.AddRateLimited(key)
			return true
		}

		if !exists {
			// Do nothing
			return true
		}
		pod := obj.(*v1.Pod)

		vmObj, exists, err := vmCache.GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[corev1.DomainLabel])
		if err != nil {
			queue.AddRateLimited(key)
			return true
		}
		if !exists {
			// Do nothing, the pod will timeout.
			return true
		}
		vm := vmObj.(*corev1.VM)
		if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[corev1.UIDLabel]) {
			// Obviously the pod of an outdated VM object, do nothing
			return true
		}
		// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
		if vm.Status.Phase == corev1.Scheduling {
			// deep copy the VM to allow manipulations
			schedulePod(vm, pod, restClient, queue, key)
		}
		return true
	})
}
func schedulePod(vm *corev1.VM, pod *v1.Pod, restClient *rest.RESTClient, queue workqueue.RateLimitingInterface, key interface{}) {
	vmCopy := copyVMFromPod(vm, pod)
	logger := logging.DefaultLogger().Object(vm)
	if err := restClient.Put().Resource("vms").Body(&vmCopy).Name(vmCopy.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
		handlePodSchedulingError(logger, err, queue, key)

	} else {
		logger.Info().Msgf("VM successfully scheduled to %s.", vmCopy.Status.NodeName)
	}
}
func handlePodSchedulingError(logger *logging.FilteredLogger, err error, queue workqueue.RateLimitingInterface, key interface{}) {
	logger.Error().Reason(err).Msg("Setting the VM to pending failed.")
	if e, ok := err.(*errors.StatusError); ok {
		if e.Status().Reason == metav1.StatusReasonNotFound {
			// VM does not exist anymore, we don't have to retry
			return
		}
	}
	logger.V(3).Info().Msg("Enqueuing VM initialization again.")
	queue.AddRateLimited(key)
}

func copyVMFromPod(vm *corev1.VM, pod *v1.Pod) corev1.VM {
	vmCopy := corev1.VM{}
	model.Copy(&vmCopy, vm)
	vmCopy.Status.Phase = corev1.Pending
	if vmCopy.GetObjectMeta().GetLabels() == nil {
		vmCopy.ObjectMeta.Labels = map[string]string{}
	}
	vmCopy.ObjectMeta.Labels[corev1.NodeNameLabel] = pod.Spec.NodeName
	vmCopy.Status.NodeName = pod.Spec.NodeName
	return vmCopy
}
