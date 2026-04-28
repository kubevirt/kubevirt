/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package find

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

type clusterInstancetypeFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewClusterInstancetypeFinder(store cache.Store, virtClient kubecli.KubevirtClient) *clusterInstancetypeFinder {
	return &clusterInstancetypeFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *clusterInstancetypeFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineClusterInstancetype, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}
	if f.store == nil {
		return f.virtClient.VirtualMachineClusterInstancetype().Get(
			context.Background(), vm.Spec.Instancetype.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(vm.Spec.Instancetype.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.VirtualMachineClusterInstancetype().Get(
			context.Background(), vm.Spec.Instancetype.Name, metav1.GetOptions{})
	}
	instancetype, ok := obj.(*v1beta1.VirtualMachineClusterInstancetype)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterInstancetype informer")
	}
	return instancetype, nil
}
