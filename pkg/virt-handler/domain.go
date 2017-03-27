package virthandler

import (
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/tools/cache"

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
func NewDomainController(vmQueue workqueue.RateLimitingInterface, vmStore cache.Store, informer cache.SharedInformer) (cache.Store, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(kubecli.NewResourceEventHandlerFuncsForQorkqueue(queue))

	dispatch := DomainDispatch{
		vmQueue: vmQueue,
		vmStore: vmStore,
	}
	return kubecli.NewControllerFromInformer(informer.GetStore(), informer, queue, &dispatch)
}

type DomainDispatch struct {
	vmQueue workqueue.RateLimitingInterface
	vmStore cache.Store
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
		logging.DefaultLogger().Info().Object(domain).Msgf("Domain is in state %s", domain.Status.Status)
	}
	obj, vmExists, err := d.vmStore.GetByKey(key.(string))
	if err != nil {
		queue.AddRateLimited(key)
	} else if !vmExists || obj.(*v1.VM).GetObjectMeta().GetUID() != domain.GetObjectMeta().GetUID() {
		// The VM is not in the vm cache, or is a VM with a differend uuid, tell the VM controller to investigate it
		d.vmQueue.Add(key)
	}
	return
}
