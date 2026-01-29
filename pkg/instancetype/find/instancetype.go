/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */
package find

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
)

type instancetypeFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewInstancetypeFinder(store cache.Store, virtClient kubecli.KubevirtClient) *instancetypeFinder {
	return &instancetypeFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *instancetypeFinder) Find(vm *virtv1.VirtualMachine) (*v1.VirtualMachineInstancetype, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Instancetype.Name,
	}

	// Helper function to convert v1beta1 to v1
	convertFromV1beta1 := func(v1beta1Obj *v1beta1.VirtualMachineInstancetype) (*v1.VirtualMachineInstancetype, error) {
		v1Obj := &v1.VirtualMachineInstancetype{}
		if err := compatibility.Convert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype(v1beta1Obj, v1Obj, nil); err != nil {
			return nil, err
		}
		return v1Obj, nil
	}

	if f.store == nil {
		v1beta1Obj, err := f.virtClient.VirtualMachineInstancetype(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return convertFromV1beta1(v1beta1Obj)
	}

	obj, exists, err := f.store.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		v1beta1Obj, err := f.virtClient.VirtualMachineInstancetype(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return convertFromV1beta1(v1beta1Obj)
	}
	instancetype, ok := obj.(*v1beta1.VirtualMachineInstancetype)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineInstancetype informer")
	}
	return convertFromV1beta1(instancetype)
}
