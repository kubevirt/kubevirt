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
 */

package network

import (
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/find"
)

type preferenceFinder interface {
	FindPreference(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
}

type controller struct {
	finder preferenceFinder
}

func New(store, clusterStore, revisionStore cache.Store, clientSet kubecli.KubevirtClient) *controller {
	return &controller{
		finder: find.NewSpecFinder(store, clusterStore, revisionStore, clientSet),
	}
}

func (a *controller) ApplyInterfacePreferences(vm *v1.VirtualMachine, vmiSpec *v1.VirtualMachineInstanceSpec) error {
	if vm.Spec.Preference == nil || len(vmiSpec.Domain.Devices.Interfaces) == 0 {
		return nil
	}
	preferenceSpec, err := a.finder.FindPreference(vm)
	if err != nil {
		return err
	}
	if preferenceSpec != nil {
		apply.ApplyInterfacePreferences(preferenceSpec, vmiSpec)
	}
	return nil
}
