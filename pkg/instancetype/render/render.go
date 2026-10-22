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
package render

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type specFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceSpecFinder interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type renderer struct {
	instancetypeFinder specFinder
	preferenceFinder   preferenceSpecFinder
}

// New renders VM templates and VMIs using instancetype and preference specs
// resolved by the supplied finders. Live resources and offline specs share
// the same application flow.
func New(instancetypeFinder specFinder, preferenceFinder preferenceSpecFinder) *renderer {
	return &renderer{instancetypeFinder: instancetypeFinder, preferenceFinder: preferenceFinder}
}
