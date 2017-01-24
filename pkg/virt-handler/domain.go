package virthandler

import (
	"fmt"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

/*
TODO: Define the exact scope of this controller.
For now it looks like we should use domain events to detect unexpected domain changes like crashes or vms going
into pause mode because of resource shortage or cut off connections to storage.
*/

func NewDomainController(lw cache.ListerWatcher) *cache.Controller {
	_, domainController := cache.NewInformer(lw, &virtwrap.Domain{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Printf("Domain ADDED: %s: %s\n", obj.(*virtwrap.Domain).GetObjectMeta().GetName(), obj.(*virtwrap.Domain).Status.Status)
			logging.DefaultLogger().Info().Object(obj.(*virtwrap.Domain)).Msg("Domain added.")
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("Domain DELETED: %s: %s\n", obj.(*virtwrap.Domain).GetObjectMeta().GetName(), obj.(*virtwrap.Domain).Status.Status)
			logging.DefaultLogger().Info().Object(obj.(*virtwrap.Domain)).Msg("Domain deleted.")
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			logging.DefaultLogger().Info().Object(new.(*virtwrap.Domain)).Msg("Domain updated.")

		},
	})
	return domainController
}
