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
 * Copyright The KubeVirt Authors
 *
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

type clusterPreferenceFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewClusterPreferenceFinder(store cache.Store, virtClient kubecli.KubevirtClient) *clusterPreferenceFinder {
	return &clusterPreferenceFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *clusterPreferenceFinder) FindPreference(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineClusterPreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	if f.store == nil {
		return f.virtClient.VirtualMachineClusterPreference().Get(
			context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(vm.Spec.Preference.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.VirtualMachineClusterPreference().Get(
			context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	}
	preference, ok := obj.(*v1beta1.VirtualMachineClusterPreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterPreference informer")
	}
	return preference, nil
}
