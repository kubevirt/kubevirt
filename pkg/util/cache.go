package util

import (
	"fmt"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
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

// TODO Namespace could be different, also store it somewhere in the domain, so that we can report deletes on handler startup properly
func NewVMReferenceFromName(name string) *v1.VM {
	vm := &v1.VM{
		ObjectMeta: kubev1.ObjectMeta{
			Name:      name,
			Namespace: api.NamespaceDefault,
			SelfLink:  fmt.Sprintf("/apis/%s/namespaces/%s/%s", v1.GroupVersion.String(), api.NamespaceDefault, name),
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: v1.GroupVersion.Group, Kind: "VM", Version: v1.GroupVersion.Version})
	return vm
}
