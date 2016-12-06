package watch

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/rmohr/go-model"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/core/pkg/api/v1"
	"kubevirt.io/core/pkg/kubecli"
	"kubevirt.io/core/pkg/virt-controller/services"
)

type vmResourceEventHandler struct {
	VMService services.VMService `inject:""`
	logger    levels.Levels
	restCli   *rest.RESTClient
}

func NewVMResourceEventHandler(logger log.Logger) (kubecli.ResourceEventHandler, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	return &vmResourceEventHandler{logger: levels.New(logger).With("component", "VMWatcher"), restCli: restClient}, nil
}

func NewVMInformer(handler kubecli.ResourceEventHandler) (*cache.Controller, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	vmCacheSource := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())
	_, ctl := kubecli.NewInformer(vmCacheSource, &v1.VM{}, 0, handler)
	return ctl, nil
}

func NewVMCache() (cache.SharedInformer, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	vmCacheSource := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())
	informer := cache.NewSharedInformer(vmCacheSource, &v1.VM{}, 0)
	return informer, nil
}

func processVM(v *vmResourceEventHandler, obj *v1.VM) error {
	defer kubecli.NewPanicCatcher(v.logger)()
	//TODO: Field selectors are not yet working for TPRs
	if obj.Status.Phase == "" {
		vm := v1.VM{}
		// Deep copy the object, so that we can savely manipulate it
		model.Copy(&vm, obj)
		vmName := vm.GetObjectMeta().GetName()
		logger := v.logger.With("object", "VM", "action", "createVMPod", "name", vmName, "UUID", vm.GetObjectMeta().GetUID())
		// Create a pod for the specified VM
		//Three cases where this can fail:
		// 1) VM pods exist from old definition // 2) VM pods exist from previous start attempt and updating the VM definition failed
		//    below
		// 3) Technical difficulties, we can't reach the apiserver
		// For case (1) this loop is not responsible. virt-handler or another loop is
		// responsible.
		// For case (2) we want to delete the VM first and then start over again.

		// TODO move defaulting to virt-api
		if vm.Spec.Domain == nil {
			spec := v1.NewMinimalVM(vm.GetObjectMeta().GetName())
			vm.Spec.Domain = spec
		}
		vm.Spec.Domain.UUID = string(vm.GetObjectMeta().GetUID())
		vm.Spec.Domain.Devices.Emulator = "/usr/local/bin/qemu-x86_64"

		// TODO get rid of these service calls
		if err := v.VMService.StartVM(&vm); err != nil {
			logger.Error().Log("msg", err)
			pl, err := v.VMService.GetRunningPods(&vm)
			if err != nil {
				// TODO detect if communication error and backoff
				logger.Error().Log("msg", err)
				return cache.ErrRequeue{Err: err}
			}
			for _, p := range pl.Items {
				if p.GetObjectMeta().GetLabels()["kubevirt.io/vmUID"] == string(vm.GetObjectMeta().GetUID()) {
					// Pod from incomplete initialization detected, cleaning up
					logger.Error().Log("msg", "Found orphan pod of current VM spec.", "pod", p.GetName())
					err = v.VMService.DeleteVM(&vm)
					if err != nil {
						// TODO detect if communication error and do backoff
						logger.Crit().Log("msg", err)
						return cache.ErrRequeue{Err: err}
					}
				} else {
					// TODO virt-api should make sure this does not happen. For now don't ask and clean up.
					// Pod from old VM object detected,
					logger.Error().Log("msg", "Found orphan pod of old VM spec.", "pod", p.GetName())
					err = v.VMService.DeleteVM(&vm)
					if err != nil {
						// TODO detect if communication error and backoff
						logger.Crit().Log("msg", err)
						return cache.ErrRequeue{Err: err}
					}
				}
			}
			return cache.ErrRequeue{Err: err}
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
		vm.Status.Phase = v1.Scheduling
		if err := v.restCli.Put().Resource("vms").Body(&vm).Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
			logger.Error().Log("msg", err)
			if e, ok := err.(*errors.StatusError); ok {
				if e.Status().Reason == unversioned.StatusReasonNotFound ||
					e.Status().Reason == unversioned.StatusReasonConflict {
					// Nothing to do for us, VM got either deleted in the meantime or a newer version is enqueued already
					return nil
				}
			}
			// TODO backoff policy here
			return cache.ErrRequeue{Err: err}
		}
		logger.Info().Log("msg", "Succeeded.")
	}
	return nil
}

func (v *vmResourceEventHandler) OnAdd(obj interface{}) error {
	return processVM(v, obj.(*v1.VM))
}

func (v *vmResourceEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	return processVM(v, newObj.(*v1.VM))
}

func (v *vmResourceEventHandler) OnDelete(obj interface{}) error {
	vm := obj.(*v1.VM)
	logger := v.logger.With("object", "VM", "action", "deleteVMPods", "name", vm.GetObjectMeta().GetName(), "UUID", vm.GetObjectMeta().GetUID())
	// TODO make sure the grace period is big enough that virt-handler can stop the VM the libvirt way
	// TODO maybe add a SIGTERM delay to virt-launcher in combination with a grace periode on the delete?
	err := v.VMService.DeleteVM(obj.(*v1.VM))
	if err != nil {
		logger.Error().Log("msg", err)
		return cache.ErrRequeue{Err: err}
	}
	logger.Info().Log("msg", "Succeeded.")
	return nil
}
