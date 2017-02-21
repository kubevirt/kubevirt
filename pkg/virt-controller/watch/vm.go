package watch

import (
	"github.com/jeevatkm/go-model"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"strings"
)

func NewVMController(vmService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient) (cache.Indexer, *kubecli.Controller) {
	lw := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())

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
		logger := logging.DefaultLogger().Object(vm)

		if !exists {
			// Delete VM Pods
			err := vmService.DeleteVM(vm)
			if err != nil {
				logger.Error().Reason(err).Msg("Deleting VM target Pod failed.")
			}
			logger.Info().Msg("Deleting VM target Pod succeeded.")
			return true
		}

		if vm.Status.Phase == "" {
			scheduleVM(vm, vmService, queue, key, restClient)
			return true
		}
		return true
	})
}
func scheduleVM(vm *v1.VM, vmService services.VMService, queue workqueue.RateLimitingInterface, key interface{}, restClient *rest.RESTClient) {
	vmCopy := copyVM(vm)
	logger := logging.DefaultLogger().Object(&vmCopy)
	defaultGraphicsNetworking(vm)
	if err := vmService.StartVM(&vmCopy); err != nil {
		handleStartVMError(logger, err, vmService, vmCopy)
		queue.AddRateLimited(key)
		return
	}

	// Mark the VM as "initialized". After the created Pod above is scheduled by
	// kubernetes, virt-handler can take over.
	//Three cases where this can fail:
	// 1) VM spec got deleted
	// 2) VM  spec got updated by the user
	// 3) Technical difficulties, we can't reach the apiserver
	// For (1) we don't want to retry, the pods will time out and fail. For (2) another
	// object got enqueued already. It will fail above until the created pods time out.
	// For (3) we want to enqueue again. If we don't do that the created pods will time out and we will
	// not get any updates
	vmCopy.Status.Phase = v1.Scheduling
	if err := restClient.Put().Resource("vms").Body(&vmCopy).Name(vmCopy.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
		handleSchedulingError(logger, err, queue, key)

	} else {
		logger.Info().Msg("Handing over the VM to the scheduler succeeded.")
	}

}
func handleSchedulingError(logger *logging.FilteredLogger, err error, queue workqueue.RateLimitingInterface, key interface{}) {
	logger.Error().Reason(err).Msg("Updating the VM state to 'Scheduling' failed.")
	if e, ok := err.(*errors.StatusError); ok {
		if e.Status().Reason == metav1.StatusReasonNotFound ||
			e.Status().Reason == metav1.StatusReasonConflict {
			// Nothing to do for us, VM got either deleted in the meantime or a newer version is enqueued already
			return
		}
	}
	queue.AddRateLimited(key)
}
func handleStartVMError(logger *logging.FilteredLogger, err error, vmService services.VMService, vmCopy v1.VM) {
	logger.Error().Reason(err).Msg("Defining a target pod for the VM.")
	pl, err := vmService.GetRunningPods(&vmCopy)
	if err != nil {
		logger.Error().Reason(err).Msg("Getting all running Pods for the VM failed.")
		return
	}
	for _, p := range pl.Items {
		if p.GetObjectMeta().GetLabels()["kubevirt.io/vmUID"] == string(vmCopy.GetObjectMeta().GetUID()) {
			// Pod from incomplete initialization detected, cleaning up
			logger.Error().Msgf("Found orphan pod with name '%s' for VM.", p.GetName())
			err = vmService.DeleteVM(&vmCopy)
			if err != nil {
				logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
				break
			}
		} else {
			// TODO virt-api should make sure this does not happen. For now don't ask and clean up.
			// Pod from old VM object detected,
			logger.Error().Msgf("Found orphan pod with name '%s' for deleted VM.", p.GetName())
			err = vmService.DeleteVM(&vmCopy)
			if err != nil {
				logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
				break
			}
		}
	}

}
func copyVM(vm *v1.VM) v1.VM {

	vmCopy := v1.VM{}
	model.Copy(&vmCopy, vm)
	if vmCopy.Spec.Domain == nil {
		spec := v1.NewMinimalDomainSpec(vmCopy.GetObjectMeta().GetName())
		vmCopy.Spec.Domain = spec
	}
	vmCopy.Spec.Domain.UUID = string(vmCopy.GetObjectMeta().GetUID())
	vmCopy.Spec.Domain.Devices.Emulator = "/usr/local/bin/qemu-x86_64"
	vmCopy.Spec.Domain.Name = vmCopy.GetObjectMeta().GetName()
	return vmCopy
}
func defaultGraphicsNetworking(vm *v1.VM) {
	graphics := vm.Spec.Domain.Devices.Graphics
	for i, _ := range graphics {
		if strings.ToLower(graphics[i].Type) == "spice" {
			graphics[i].Port = int32(4000) + int32(i)
			graphics[i].Listen = v1.Listen{
				Address: "0.0.0.0",
				Type:    "address",
			}

		}
	}
}
