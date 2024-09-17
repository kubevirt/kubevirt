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
 * Copyright 2024 Red Hat, Inc.
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

type SpecFinder struct {
	preferenceFinder        *PreferenceFinder
	clusterPreferenceFinder *ClusterPreferenceFinder
	revisionFinder          *RevisionFinder
}

func NewSpecFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *SpecFinder {
	return &SpecFinder{
		preferenceFinder:        NewPreferenceFinder(store, virtClient),
		clusterPreferenceFinder: NewClusterPreferenceFinder(clusterStore, virtClient),
		revisionFinder:          NewRevisionFinder(revisionStore, virtClient),
	}
}

const unexpectedKindFmt = "got unexpected kind in PreferenceMatcher: %s"

func (f *SpecFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if vm.Spec.Preference.RevisionName != "" {
		revision, err := f.revisionFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return compatibility.GetPreferenceSpec(revision)
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case api.SingularPreferenceResourceName, api.PluralPreferenceResourceName:
		preference, err := f.preferenceFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &preference.Spec, nil

	case api.ClusterSingularPreferenceResourceName, api.ClusterPluralPreferenceResourceName, "":
		clusterPreference, err := f.clusterPreferenceFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &clusterPreference.Spec, nil

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Preference.Kind)
	}
}
