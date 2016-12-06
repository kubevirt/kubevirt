package watch

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/rmohr/go-model"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/core/pkg/api/v1"
	"kubevirt.io/core/pkg/kubecli"
	"kubevirt.io/core/pkg/virt-controller/services"
)

type VMWatcher interface {
	Watch() (chan struct{}, error)
}

type vmWatcher struct {
	VMService services.VMService `inject:""`
	logger    levels.Levels
}

func NewVMWatcher(logger log.Logger) VMWatcher {
	return &vmWatcher{logger: levels.New(logger).With("component", "VMWatcher")}
}

func (v *vmWatcher) Watch() (chan struct{}, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}

	stop := make(chan struct{})
	vmSource := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceDefault, fields.Everything())
	queue := cache.NewFIFO(cache.MetaNamespaceKeyFunc)
	cache.NewReflector(vmSource, &v1.VM{}, queue, 0).RunUntil(stop)
	// TODO catch precond panics
	go func() {
		for {
			dto := cache.Pop(queue).(*v1.VM)
			//TODO: Field selectors are not yet working for TPRs
			if dto.Status.Phase == "" {
				vm := v1.VM{}
				model.Copy(&vm, dto)
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
				if err := v.VMService.StartVM(&vm); err != nil {
					logger.Error().Log("msg", err)
					if pl, err := v.VMService.GetRunningPods(&vm); err == nil {
						for _, p := range pl.Items {
							if p.GetObjectMeta().GetLabels()["vmUID"] == string(vm.GetObjectMeta().GetUID()) {
								// Pod from incomplete initialization detected, cleaning up
								logger.Error().Log("msg", "Found orphan pod of current VM spec.", "pod", p.GetName())
								err = v.VMService.DeleteVM(&vm)
								if err != nil {
									// TODO detect if communication error and do backoff
									logger.Crit().Log("msg", err)
								}
							} else {
								// TODO admission plugin should make sure this does not happen. For now don't ask and clean up.
								// Pod from old VM object detected,
								logger.Error().Log("msg", "Found orphan pod of old VM spec.", "pod", p.GetName())
								err = v.VMService.DeleteVM(&vm)
								if err != nil {
									// TODO detect if communication error and backoff
									logger.Crit().Log("msg", err)
								}
							}
						}
						// Enqueue the initialization task again
						logger.Info().Log("msg", "Enqueing initialization of VM again.")
						queue.AddIfNotPresent(dto)
					} else {
						logger.Crit().Log("msg", "Checking for running pods for VM failed.")
						// TODO detect if communication error and backoff
					}
					continue
				}
				// Mark the VM as "initialized". After the created Pod above is scheduled by
				// kubernetes, virt-handler can take over.
				//Three cases where this can fail:
				// 1) VM spec got deleted
				// 2) VM  spec got updated by the user
				// 3) Technical difficulties, we can't reach the apiserver
				// For (1) we don't want to retry, the pods will time out and fail. For (2) another
				// object got enqueued already. It will fail above until the created pods time out.
				// For (3) we want to enqueue again. If not the created pods will time out and we will
				// not get any updates
				dto.Status.Phase = v1.Pending
				if err := restClient.Put().Resource("vms").Body(dto).Suffix(dto.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error(); err != nil {
					logger.Error().Log("msg", err)
					if e, ok := err.(*errors.StatusError); ok {
						if e.Status().Reason == unversioned.StatusReasonNotFound ||
							e.Status().Reason == unversioned.StatusReasonConflict {
							continue
						}
					}
					logger.Info().Log("msg", "Enqueing initialization of VM again.")
					// TODO backoff policy here
					queue.AddIfNotPresent(dto)
					fmt.Println(err)
				}
				logger.Info().Log("msg", "Initialized.")
			}
		}
	}()

	return stop, nil
}
