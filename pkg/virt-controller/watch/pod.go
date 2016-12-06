package watch

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/rmohr/go-model"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/types"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	corev1 "kubevirt.io/core/pkg/api/v1"
	"kubevirt.io/core/pkg/kubecli"
)

func NewPodResourceEventHandler(logger log.Logger) (kubecli.ResourceEventHandler, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	return &podResourceEventHandler{logger: levels.New(logger).With("component", "PodWatcher"), restCli: restClient}, nil
}

type podResourceEventHandler struct {
	logger  levels.Levels
	restCli *rest.RESTClient
	VMCache cache.SharedInformer `inject:""`
}

func NewPodInformer(handler kubecli.ResourceEventHandler) (*cache.Controller, error) {
	restClient, err := kubecli.Get()
	if err != nil {
		return nil, err
	}
	selector := scheduledVMPodSelector()
	podCacheSource := kubecli.NewListWatchFromClient(restClient.Core().GetRESTClient(), "pods", kubeapi.NamespaceDefault, selector.FieldSelector, selector.LabelSelector)
	_, ctl := kubecli.NewInformer(podCacheSource, &v1.Pod{}, 0, handler)
	return ctl, nil
}

func NewPodCache() (cache.SharedInformer, error) {
	cli, err := kubecli.Get()
	if err != nil {
		return nil, err
	}
	//TODO, maybe we can combine the list watchers for the cache and the watcher?
	vmCacheSource := cache.NewListWatchFromClient(cli.Core().GetRESTClient(), "pods", kubeapi.NamespaceDefault, fields.Everything())
	informer := cache.NewSharedInformer(vmCacheSource, &v1.Pod{}, 0)
	return informer, nil
}

func processPod(p *podResourceEventHandler, pod *v1.Pod) error {
	defer kubecli.NewPanicCatcher(p.logger)()
	vmObj, exists, err := p.VMCache.GetStore().GetByKey(kubeapi.NamespaceDefault + "/" + pod.GetLabels()["kubevirt.io/domain"])
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

	logger := p.logger.With("object", "VM", "action", "setVMPending", "name", vm.GetObjectMeta().GetName(), "UUID", vm.GetObjectMeta().GetUID())
	if vm.GetObjectMeta().GetUID() != types.UID(pod.GetLabels()["kubevirt.io/vmUID"]) {
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
		vm.ObjectMeta.Labels["kubevirt.io/nodeName"] = pod.Spec.NodeName
		// Update the VM
		if err := p.restCli.Put().Resource("vms").Body(&vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
			logger.Error().Log("msg", err)
			if e, ok := err.(*errors.StatusError); ok {
				if e.Status().Reason == unversioned.StatusReasonNotFound {
					// VM does not exist anymore, we don't have to retry
					return nil
				}
				// TODO backoff policy here?
			}
			logger.Info().Log("msg", "Enqueing initialization of VM again.")
			return cache.ErrRequeue{Err: err}
		} else {
			logger.Info().Log("msg", "Succeeded.")
		}
	}
	return nil
}

func scheduledVMPodSelector() kubeapi.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(kubeapi.PodRunning))
	labelSelector, err := labels.Parse("kubevirt.io/app in (virt-launcher)")
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
