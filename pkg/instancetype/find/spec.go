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
	"fmt"
	"strings"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
)

type specFinder struct {
	instancetypeFinder        *instancetypeFinder
	clusterInstancetypeFinder *clusterInstancetypeFinder
	revisionFinder            *revisionFinder
}

func NewSpecFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *specFinder {
	return &specFinder{
		instancetypeFinder:        NewInstancetypeFinder(store, virtClient),
		clusterInstancetypeFinder: NewClusterInstancetypeFinder(clusterStore, virtClient),
		revisionFinder:            NewRevisionFinder(revisionStore, virtClient),
	}
}

const unexpectedKindFmt = "got unexpected kind in InstancetypeMatcher: %s"

func (f *specFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	revision, err := f.revisionFinder.Find(vm)
	if err != nil {
		return nil, err
	}
	if revision != nil {
		return compatibility.GetInstancetypeSpec(revision)
	}

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		instancetype, err := f.instancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &instancetype.Spec, nil

	case api.ClusterSingularResourceName, api.ClusterPluralResourceName, "":
		clusterInstancetype, err := f.clusterInstancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &clusterInstancetype.Spec, nil

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Instancetype.Kind)
	}
}
