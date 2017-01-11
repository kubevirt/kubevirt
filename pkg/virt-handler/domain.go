package virthandler

import (
	"fmt"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-handler/libvirt"
)

/*
TODO: Define the exact scope of this controller.
For now it looks like we should use domain events to detect unexpected domain changes like crashes or vms going
into pause mode because of resource shortage or cut off connections to storage.
*/

func NewDomainController(lw cache.ListerWatcher) *cache.Controller {
	_, domainController := kubecli.NewInformer(lw, &libvirt.Domain{}, 0, kubecli.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) error {
			fmt.Printf("Domain ADDED: %s: %s\n", obj.(*libvirt.Domain).GetObjectMeta().GetName(), obj.(*libvirt.Domain).Status.Status)
			return nil
		},
		DeleteFunc: func(obj interface{}) error {
			fmt.Printf("Domain DELETED: %s: %s\n", obj.(*libvirt.Domain).GetObjectMeta().GetName(), obj.(*libvirt.Domain).Status.Status)
			return nil
		},
		UpdateFunc: func(old interface{}, new interface{}) error {
			fmt.Printf("Domain UPDATED: %s: %s\n", new.(*libvirt.Domain).GetObjectMeta().GetName(), new.(*libvirt.Domain).Status.Status)
			return nil
		},
	})
	return domainController
}
