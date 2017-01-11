package watch

import (
	"github.com/jeevatkm/go-model"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	corev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
)

func NewPodResourceEventHandler() (kubecli.ResourceEventHandler, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	return &podResourceEventHandler{restCli: restClient}, nil
}

type podResourceEventHandler struct {
	restCli *rest.RESTClient
	VMCache cache.SharedInformer `inject:""`
}

func NewPodInformer(handler kubecli.ResourceEventHandler) (*cache.Controller, error) {
	restClient, err := kubecli.Get()
	if err != nil {
		return nil, err
	}
	selector := scheduledVMPodSelector()
	podCacheSource := kubecli.NewListWatchFromClient(restClient.Core().RESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	_, ctl := kubecli.NewInformer(podCacheSource, &v1.Pod{}, 0, handler)
	return ctl, nil
}

func NewPodCache() (cache.SharedInformer, error) {
	cli, err := kubecli.Get()
	if err != nil {
		return nil, err
	}
	//TODO, maybe we can combine the list watchers for the cache and the watcher?
	vmCacheSource := cache.NewListWatchFromClient(cli.Core().RESTClient(), "pods", kubeapi.NamespaceDefault, fields.Everything())
	informer := cache.NewSharedInformer(vmCacheSource, &v1.Pod{}, 0)
	return informer, nil
}

func processPod(p *podResourceEventHandler, pod *v1.Pod) error {
	defer kubecli.NewPanicCatcher()()
	vmObj, exists, err := p.VMCache.GetStore().GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()[corev1.DomainLabel])
	if err != nil {
		// TODO handle this smarter, for now just try again
		return cache.ErrRequeue{Err: err}
	}
	if !exists {
		// Do nothing, the pod will timeout.
		return nil
	}
	// deep copy the VM to allow manipulations
	vm := corev1.VM{}
	model.Copy(&vm, vmObj)

	logger := logging.DefaultLogger().Object(&vm)
	if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()[corev1.UIDLabel]) {
		// Obviously the pod of an outdated VM object, do nothing
		return nil
	}
	// This is basically a hack, so that virt-handler can completely focus on the VM object and does not have to care about pods
	if vm.Status.Phase == corev1.Scheduling {
		vm.Status.Phase = corev1.Pending
		// FIXME we store this in the metadata since field selctors are currently not working for TPRs
		if vm.GetObjectMeta().GetLabels() == nil {
			vm.ObjectMeta.Labels = map[string]string{}
		}
		vm.ObjectMeta.Labels[corev1.NodeNameLabel] = pod.Spec.NodeName
		// Update the VM
		if err := p.restCli.Put().Resource("vms").Body(&vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
			logger.Error().Msgf("Setting the VM to pending failed with: %s", err)
			if e, ok := err.(*errors.StatusError); ok {
				if e.Status().Reason == metav1.StatusReasonNotFound {
					// VM does not exist anymore, we don't have to retry
					return nil
				}
				// TODO backoff policy here?
			}
			logger.V(3).Info().Msg("Enqueuing VM initialization again.")
			return cache.ErrRequeue{Err: err}
		} else {
			logger.Info().Msgf("VM successfully scheduled to %s.", vm.Status.NodeName)
		}
	}
	return nil
}

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(kubeapi.PodRunning))
	labelSelector, err := labels.Parse(corev1.AppLabel + " in (virt-launcher)")
	if err != nil {
		panic(err)
	}
	return kubeapi.ListOptions{FieldSelector: fieldSelector, LabelSelector: labelSelector}
}

func (p *podResourceEventHandler) OnAdd(obj interface{}) error {
	return processPod(p, obj.(*v1.Pod))
}

func (p *podResourceEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	return processPod(p, newObj.(*v1.Pod))
}

func (p *podResourceEventHandler) OnDelete(obj interface{}) error {
	// virt-controller does nothing in this case, it is up to virt-handler or the pod timeout itself to clean up
	return nil
}
