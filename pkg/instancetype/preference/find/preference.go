/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package find

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

type preferenceFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewPreferenceFinder(store cache.Store, virtClient kubecli.KubevirtClient) *preferenceFinder {
	return &preferenceFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *preferenceFinder) FindPreference(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Preference.Name,
	}
	if f.store == nil {
		return f.virtClient.VirtualMachinePreference(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.VirtualMachinePreference(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}
	preference, ok := obj.(*v1beta1.VirtualMachinePreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachinePreference informer")
	}
	return preference, nil
}
