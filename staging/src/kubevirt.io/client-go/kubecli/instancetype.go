package kubecli

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
)

func (k *kubevirt) ExpandSpec() *ExpandSpec {
	return &ExpandSpec{
		restClient: k.restClient,
	}
}

type ExpandSpec struct {
	restClient *rest.RESTClient
}

func (e *ExpandSpec) ForVirtualMachine(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	uri := fmt.Sprintf("/apis/"+v1.SubresourceGroupName+"/%s/expand-spec", v1.ApiStorageVersion)
	expandedVm := &v1.VirtualMachine{}
	err := e.restClient.Put().
		AbsPath(uri).
		Body(vm).
		Do(context.Background()).
		Into(expandedVm)

	expandedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return expandedVm, err
}
