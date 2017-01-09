package virthandler

import (
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	kubev1 "k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/libvirt"
)

func NewVMController(listWatcher cache.ListerWatcher, domainManager libvirt.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient) (cache.Indexer, *cache.Controller) {
	logger := logging.DefaultLogger()

	return kubecli.NewInformer(listWatcher, &v1.VM{}, 0, kubecli.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) error {
			logger.Info().Msg("VM ADD")
			vm := obj.(*v1.VM)
			err := domainManager.SyncVM(vm)
			if err != nil {
				goto is_error
			}

			if vm.Status.Phase != v1.Running {
				vm.Status.Phase = v1.Running
				err = restClient.Put().Resource("vms").Body(vm).
					Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
				if err != nil {
					goto is_error
				}
			}
		is_error:
			if err != nil {
				logger.Error().Msg(err)
				recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
		DeleteFunc: func(obj interface{}) error {
			// stop and undefine
			// Let's reenque the delete request until we reach the end of the mothod or until
			// we detect that the VM does not exist anymore
			logger.Info().Msg("VM DELETE")
			vm, ok := obj.(*v1.VM)
			if !ok {
				vm = obj.(cache.DeletedFinalStateUnknown).Obj.(*v1.VM)
			}
			err := domainManager.KillVM(vm)
			if err != nil {
				logger.Error().Msg(err)
				recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
		UpdateFunc: func(old interface{}, new interface{}) error {

			logger.Info().Msg("VM UPDATE")
			// TODO: at the moment kubecli.NewInformer guarantees that if old is already equal to new,
			//       in this case we don't need to sync if old is equal to new (but this might change)
			// TODO: Implement the spec update flow in LibvirtDomainManager.SyncVM
			vm := new.(*v1.VM)
			err := domainManager.SyncVM(vm)
			if err != nil {
				logger.Error().Msg(err)
				recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
	})
}
