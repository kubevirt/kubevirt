package kubecli

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	snapshotapi "kubevirt.io/api/snapshot/v1beta1"
)

func (k *kubevirtClient) ExpandSpec(namespace string) ExpandSpecInterface {
	return &expandSpec{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "expand-vm-spec",
	}
}

type expandSpec struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (e *expandSpec) ForVirtualMachine(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	uri := fmt.Sprintf("/apis/"+v1.SubresourceGroupName+"/%s/namespaces/%s/%s", v1.ApiStorageVersion, e.namespace, e.resource)
	expandedVm := &v1.VirtualMachine{}
	err := e.restClient.Put().
		AbsPath(uri).
		Body(vm).
		Do(context.Background()).
		Into(expandedVm)

	expandedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return expandedVm, err
}

func (e *expandSpec) ForSnapshotVirtualMachine(snapshotVM *snapshotapi.VirtualMachine) (*snapshotapi.VirtualMachine, error) {
	regularVM := &v1.VirtualMachine{
		ObjectMeta: snapshotVM.ObjectMeta,
		Spec:       snapshotVM.Spec,
	}

	expandedVM, err := e.ForVirtualMachine(regularVM)
	if err != nil {
		return nil, err
	}

	expandedSnapshotVM := &snapshotapi.VirtualMachine{
		ObjectMeta: snapshotVM.ObjectMeta,
		Spec:       expandedVM.Spec,
		Status:     snapshotVM.Status,
	}

	return expandedSnapshotVM, nil
}
