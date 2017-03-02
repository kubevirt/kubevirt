package virthandler

import (
	kubeapi "k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

func NewVMController(lw cache.ListerWatcher, domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient) (cache.Indexer, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return kubecli.NewController(lw, queue, &v1.VM{}, func(indexer cache.Indexer, queue workqueue.RateLimitingInterface) bool {
		key, quit := queue.Get()
		if quit {
			return false
		}
		defer queue.Done(key)
		// Fetch the latest Vm state from cache
		obj, exists, err := indexer.GetByKey(key.(string))

		if err != nil {
			queue.AddRateLimited(key)
			return true
		}

		// Retrieve the VM
		var vm *v1.VM
		if !exists {
			_, name, err := cache.SplitMetaNamespaceKey(key.(string))
			if err != nil {
				// TODO do something more smart here
				queue.AddRateLimited(key)
				return true
			}

			vm = v1.NewVMReferenceFromName(name)
		} else {
			vm = obj.(*v1.VM)
		}
		logging.DefaultLogger().V(3).Info().Object(vm).Msg("Processing VM update.")

		// Process the VM
		if !exists {
			// Since the VM was not in the cache, we delete it
			err = domainManager.KillVM(vm)
		} else {
			// Synchronize the VM state
			// TODO check if found VM has the same UID like the domain, if not, delete the Domain first
			err = domainManager.SyncVM(vm)

			// Update VM status to running
			if err == nil && vm.Status.Phase != v1.Running {
				obj, err = kubeapi.Scheme.Copy(vm)
				if err == nil {
					vm = obj.(*v1.VM)
					vm.Status.Phase = v1.Running
					err = restClient.Put().Resource("vms").Body(vm).
						Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
				}
			}
		}

		if err != nil {
			// Something went wrong, reenqueue the item with a delay
			logging.DefaultLogger().Error().Object(vm).Reason(err).Msg("Synchronizing the VM failed.")
			recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
			queue.AddRateLimited(key)
			return true
		}

		logging.DefaultLogger().V(3).Info().Object(vm).Msg("Synchronizing the VM succeeded.")
		queue.Forget(key)
		return true
	})
}
