package virthandler

import (
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"net/http"
)

func NewVMController(lw cache.ListerWatcher, domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient, host string) (cache.Indexer, *kubecli.Controller) {
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

			// If we don't have the VM in the cache, it could be that it is currently migrating to us
			result := restClient.Get().Name(vm.GetObjectMeta().GetName()).Resource("vms").Namespace(kubeapi.NamespaceDefault).Do()
			if result.Error() == nil {
				// So the VM still seems to exist
				fetchedVM, err := result.Get()
				if err != nil {
					// Since there was no fetch error, this should have worked, let's back off
					queue.AddRateLimited(key)
					return true
				}
				if fetchedVM.(*v1.VM).Status.MigrationNodeName == host {
					// OK, this VM is migrating to us, don't interrupt it
					queue.Forget(key)
					return true
				}
			} else if result.Error().(*errors.StatusError).Status().Code != int32(http.StatusNotFound) {
				// Something went wrong, let's try again later
				queue.AddRateLimited(key)
				return true
			}
			// The VM is deleted on the cluster, let's go on with the deletion on the host
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

			// Only sync if the VM is not marked as migrating. Everything except shutting down the VM is not permitted when it is migrating.
			// TODO MigrationNodeName should be a pointer
			if vm.Status.MigrationNodeName == "" {
				err = domainManager.SyncVM(vm)
			} else {
				queue.Forget(key)
				return true
			}

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
