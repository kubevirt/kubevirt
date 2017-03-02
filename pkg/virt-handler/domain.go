package virthandler

import (
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

/*
TODO: Define the exact scope of this controller.
For now it looks like we should use domain events to detect unexpected domain changes like crashes or vms going
into pause mode because of resource shortage or cut off connections to storage.
*/

func RegisterDomainListener(informer cache.SharedInformer) error {
	return informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logging.DefaultLogger().Info().Object(obj.(*virtwrap.Domain)).Msgf("Added domain is in state %s", obj.(*virtwrap.Domain).Status.Status)
		},
		DeleteFunc: func(obj interface{}) {
			logging.DefaultLogger().Info().Object(obj.(*virtwrap.Domain)).Msgf("Deleted domain is in state %s", obj.(*virtwrap.Domain).Status.Status)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			logging.DefaultLogger().Info().Object(new.(*virtwrap.Domain)).Msgf("Updated domain is in state %s", new.(*virtwrap.Domain).Status.Status)

		},
	})
}
