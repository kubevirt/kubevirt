package virthandler

import (
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

/*
TODO: Define the exact scope of this controller.
For now it looks like we should use domain events to detect unexpected domain changes like crashes or vms going
into pause mode because of resource shortage or cut off connections to storage.
*/
func NewDomainController(vmQueue workqueue.RateLimitingInterface, vmStore cache.Store, informer cache.SharedInformer, restClient rest.RESTClient, recorder record.EventRecorder) (cache.Store, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForQorkqueue(queue))

	dispatch := DomainDispatch{
		vmQueue:    vmQueue,
		vmStore:    vmStore,
		recorder:   recorder,
		restClient: restClient,
	}
	return kubecli.NewControllerFromInformer(informer.GetStore(), informer, queue, &dispatch)
}

type DomainDispatch struct {
	vmQueue    workqueue.RateLimitingInterface
	vmStore    cache.Store
	recorder   record.EventRecorder
	restClient rest.RESTClient
}

func (d *DomainDispatch) Execute(indexer cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {
	obj, exists, err := indexer.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
		return
	}
	var domain *virtwrap.Domain
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			queue.AddRateLimited(key)
			return
		}
		domain = virtwrap.NewDomainReferenceFromName(name)
		logging.DefaultLogger().Info().Object(domain).Msgf("Domain deleted")
	} else {
		domain = obj.(*virtwrap.Domain)
		logging.DefaultLogger().Info().Object(domain).Msgf("Domain is in state %s reason %s", domain.Status.Status, domain.Status.Reason)
	}
	obj, vmExists, err := d.vmStore.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
	} else if !vmExists || obj.(*v1.VM).GetObjectMeta().GetUID() != domain.GetObjectMeta().GetUID() {
		// The VM is not in the vm cache, or is a VM with a differend uuid, tell the VM controller to investigate it
		d.vmQueue.Add(key)
	} else {
		err = d.setVmPhaseForStatusReason(domain, obj.(*v1.VM))
		if err != nil {
			queue.AddRateLimited(key)
		}
	}

	return
}

func (d *DomainDispatch) setVmPhaseForStatusReason(domain *virtwrap.Domain, vm *v1.VM) error {
	flag := false
	if domain.Status.Status == virtwrap.Shutoff {
		switch domain.Status.Reason {
		case virtwrap.ReasonCrashed:
			vm.Status.Phase = v1.Failed
			d.recorder.Event(vm, k8sv1.EventTypeWarning, v1.Stopped.String(), "The VM crashed.")
			flag = true
		case virtwrap.ReasonShutdown, virtwrap.ReasonDestroyed, virtwrap.ReasonSaved, virtwrap.ReasonFromSnapshot:
			vm.Status.Phase = v1.Succeeded
			d.recorder.Event(vm, k8sv1.EventTypeNormal, v1.Stopped.String(), "The VM was shut down.")
			flag = true
		}
	}

	if flag {
		logging.DefaultLogger().Info().Object(vm).Msgf("Changing VM phase to %s", vm.Status.Phase)
		return d.restClient.Put().Resource("vms").Body(vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
	}

	return nil
}
