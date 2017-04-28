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

func NewVMController(vmService services.VMService, recorder record.EventRecorder, restClient *rest.RESTClient) (cache.Store, *kubecli.Controller) {
	lw := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())
	dispatch := NewVMControllerDispatch(restClient, vmService)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, informer := cache.NewIndexerInformer(lw, &v1.VM{}, 0, kubecli.NewResourceEventHandlerFuncsForWorkqueue(queue), cache.Indexers{})
	return kubecli.NewControllerFromInformer(indexer, informer, queue, dispatch)

}

func NewVMControllerDispatch(restClient *rest.RESTClient, vmService services.VMService) kubecli.ControllerDispatch {
	dispatch := VMDispatch{
		restClient: restClient,
		vmService:  vmService,
	}
	var vmd kubecli.ControllerDispatch = &dispatch
	return vmd
}

type VMDispatch struct {
	restClient *rest.RESTClient
	vmService  services.VMService
}

func (vmd *VMDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {

	// Fetch the latest Vm state from cache
	obj, exists, err := store.GetByKey(key.(string))

	if err != nil {
		queue.AddRateLimited(key)
		return
	}

	// Retrieve the VM
	var vm *v1.VM
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			// TODO do something more smart here
			queue.AddRateLimited(key)
			return
		}
		vm = v1.NewVMReferenceFromName(name)
	} else {
		vm = obj.(*v1.VM)
	}
	logger := logging.DefaultLogger().Object(vm)

	if !exists {
		// Delete VM Pods
		err := vmd.vmService.DeleteVMPod(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("Deleting VM target Pod failed.")
		}
		logger.Info().Msg("Deleting VM target Pod succeeded.")
	} else if vm.Status.Phase == v1.VmPhaseUnset {
		// Schedule the VM
		vmCopy := v1.VM{}

		// Deep copy the object, so that we can safely manipulate it
		model.Copy(&vmCopy, vm)
		logger := logging.DefaultLogger().Object(&vmCopy)

		// Create a pod for the specified VM
		//Three cases where this can fail:
		// 1) VM pods exist from old definition // 2) VM pods exist from previous start attempt and updating the VM definition failed
		//    below
		// 3) Technical difficulties, we can't reach the apiserver
		// For case (1) this loop is not responsible. virt-handler or another loop is
		// responsible.
		// For case (2) we want to delete the VM first and then start over again.

		// TODO move defaulting to virt-api
		if vmCopy.Spec.Domain == nil {
			spec := v1.NewMinimalDomainSpec(vmCopy.GetObjectMeta().GetName())
			vmCopy.Spec.Domain = spec
		}
		vmCopy.Spec.Domain.UUID = string(vmCopy.GetObjectMeta().GetUID())
		vmCopy.Spec.Domain.Devices.Emulator = "/usr/local/bin/qemu-x86_64"
		vmCopy.Spec.Domain.Name = vmCopy.GetObjectMeta().GetName()

		// TODO when we move this to virt-api, we have to block that they are set on POST or changed on PUT
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

		// TODO get rid of these service calls
		if err := vmd.vmService.StartVMPod(&vmCopy); err != nil {
			logger.Error().Reason(err).Msg("Defining a target pod for the VM.")
			pl, err := vmd.vmService.GetRunningVMPods(&vmCopy)
			if err != nil {
				logger.Error().Reason(err).Msg("Getting all running Pods for the VM failed.")
				queue.AddRateLimited(key)
				return
			}
			for _, p := range pl.Items {
				if p.GetObjectMeta().GetLabels()["kubevirt.io/vmUID"] == string(vmCopy.GetObjectMeta().GetUID()) {
					// Pod from incomplete initialization detected, cleaning up
					logger.Error().Msgf("Found orphan pod with name '%s' for VM.", p.GetName())
					err = vmd.vmService.DeleteVMPod(&vmCopy)
					if err != nil {
						logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
						queue.AddRateLimited(key)
						return
					}
				} else {
					// TODO virt-api should make sure this does not happen. For now don't ask and clean up.
					// Pod from old VM object detected,
					logger.Error().Msgf("Found orphan pod with name '%s' for deleted VM.", p.GetName())
					err = vmd.vmService.DeleteVMPod(&vmCopy)
					if err != nil {
						logger.Critical().Reason(err).Msgf("Deleting orphaned pod with name '%s' for VM failed.", p.GetName())
						queue.AddRateLimited(key)
						return
					}
				}
			}
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
		if err := vmd.restClient.Put().Resource("vms").Body(&vmCopy).Name(vmCopy.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
			logger.Error().Reason(err).Msg("Updating the VM state to 'Scheduling' failed.")
			if e, ok := err.(*errors.StatusError); ok {
				if e.Status().Reason == metav1.StatusReasonNotFound ||
					e.Status().Reason == metav1.StatusReasonConflict {
					// Nothing to do for us, VM got either deleted in the meantime or a newer version is enqueued already
					return
				}
			}
			queue.AddRateLimited(key)
			return
		}
		logger.Info().Msg("Handing over the VM to the scheduler succeeded.")
	}
	return
}
