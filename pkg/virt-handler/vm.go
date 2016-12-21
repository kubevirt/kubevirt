package virthandler

import (
	"fmt"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-handler/libvirt"
)

func NewVMController(listWatcher cache.ListerWatcher, domainManager libvirt.DomainManager) (cache.Indexer, *cache.Controller) {
	return kubecli.NewInformer(listWatcher, &v1.VM{}, 0, kubecli.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) error {
			fmt.Printf("VM ADD\n")
			vm := obj.(*v1.VM)
			err := domainManager.SyncVM(vm)
			if err != nil {
				fmt.Println(err)
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
		DeleteFunc: func(obj interface{}) error {
			// stop and undefine
			// Let's reenque the delete request until we reach the end of the mothod or until
			// we detect that the VM does not exist anymore
			fmt.Printf("VM DELETE\n")
			vm, ok := obj.(*v1.VM)
			if !ok {
				vm = obj.(cache.DeletedFinalStateUnknown).Obj.(*v1.VM)
			}
			err := domainManager.KillVM(vm)
			if err != nil {
				fmt.Println(err)
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
		UpdateFunc: func(old interface{}, new interface{}) error {
			fmt.Printf("VM UPDATE\n")
			// TODO: at the moment kubecli.NewInformer guarantees that if old is already equal to new,
			//       in this case we don't need to sync if old is equal to new (but this might change)
			// TODO: Implement the spec update flow in LibvirtDomainManager.SyncVM
			vm := new.(*v1.VM)
			err := domainManager.SyncVM(vm)
			if err != nil {
				fmt.Println(err)
				return cache.ErrRequeue{Err: err}
			}
			return nil
		},
	})
}
