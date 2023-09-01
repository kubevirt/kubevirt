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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
)

type PreferenceSpecOption func(*instancetypev1alpha2.VirtualMachinePreferenceSpec)
type PreferenceOption func(*instancetypev1alpha2.VirtualMachinePreference)
type ClusterPreferenceOption func(*instancetypev1alpha2.VirtualMachineClusterPreference)

func NewPreferenceSpec(opts ...PreferenceSpecOption) instancetypev1alpha2.VirtualMachinePreferenceSpec {
	spec := &instancetypev1alpha2.VirtualMachinePreferenceSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func NewPreference(opts ...PreferenceOption) *instancetypev1alpha2.VirtualMachinePreference {
	preference := basePreference(randPreferenceName())
	for _, f := range opts {
		f(preference)
	}
	return preference
}

func WithPreferenceSpec(spec instancetypev1alpha2.VirtualMachinePreferenceSpec) PreferenceOption {
	return func(preference *instancetypev1alpha2.VirtualMachinePreference) {
		preference.Spec = spec
	}
}

func NewClusterPreference(opts ...ClusterPreferenceOption) *instancetypev1alpha2.VirtualMachineClusterPreference {
	preference := baseClusterPreference(randPreferenceName())
	for _, f := range opts {
		f(preference)
	}
	return preference
}

func WithClusterPreferenceSpec(spec instancetypev1alpha2.VirtualMachinePreferenceSpec) ClusterPreferenceOption {
	return func(preference *instancetypev1alpha2.VirtualMachineClusterPreference) {
		preference.Spec = spec
	}
}

func basePreference(name string) *instancetypev1alpha2.VirtualMachinePreference {
	return &instancetypev1alpha2.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func baseClusterPreference(name string) *instancetypev1alpha2.VirtualMachineClusterPreference {
	return &instancetypev1alpha2.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func randPreferenceName() string {
	const randomPostfixLen = 5
	return "preference" + "-" + rand.String(randomPostfixLen)
}
