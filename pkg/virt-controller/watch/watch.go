package watch

import (
	"github.com/rmohr/go-model"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/kubecli"
	"kubevirt/core/pkg/virt-controller/services"
	"time"
)

type VMWatcher interface {
	Watch() (chan struct{}, error)
}

type vmWatcher struct {
	VMService services.VMService `inject:""`
}

func NewVMWatcher() VMWatcher {
	return &vmWatcher{}
}

func (v *vmWatcher) Watch() (chan struct{}, error) {
	restClient, err := kubecli.GetRESTCLient()
	if err != nil {
		return nil, err
	}

	source := cache.NewListWatchFromClient(restClient, "vms", kubeapi.NamespaceAll, fields.Everything())

	addHandler := func(obj interface{}) {
		vm := api.VM{}
		model.Copy(&vm, obj)
		v.VMService.StartVM(&vm)
	}

	deleteHandler := func(obj interface{}) {
		vm := api.VM{}
		model.Copy(&vm, obj)
		v.VMService.DeleteVM(&vm)
	}

	_, c := cache.NewInformer(
		source,
		&v1.VM{},
		30*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addHandler,
			DeleteFunc: deleteHandler,
		})

	stop := make(chan struct{})
	go func() {
		c.Run(stop)
	}()
	return stop, nil
}
