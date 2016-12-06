package util

import (
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/core/pkg/api/v1"
	"kubevirt.io/core/pkg/kubecli"
)

func NewVMCache() (cache.SharedInformer, error) {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return nil, err
	}
	vmCacheSource := cache.NewListWatchFromClient(restClient, "vms", api.NamespaceDefault, fields.Everything())
	informer := cache.NewSharedInformer(vmCacheSource, &v1.VM{}, 0)
	return informer, nil
}
