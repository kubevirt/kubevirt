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
package metrics

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
)

type applyVMHandler interface {
	ApplyToVM(*virtv1.VirtualMachine) error
}

type instancetypeFindHandler interface {
	Find(*virtv1.VirtualMachine) (metav1.Object, error)
	FindFromVMI(*virtv1.VirtualMachineInstance) (metav1.Object, error)
}

type preferenceFindHandler interface {
	FindPreference(*virtv1.VirtualMachine) (metav1.Object, error)
	FindPreferenceFromVMI(*virtv1.VirtualMachineInstance) (metav1.Object, error)
}

type metricsHandler struct {
	instancetypeFindHandler
	preferenceFindHandler
	applyVMHandler
}

func New(
	instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, revisionStore cache.Store,
	virtClient kubecli.KubevirtClient,
) *metricsHandler {
	return &metricsHandler{
		instancetypeFindHandler: find.NewObjectFinder(instancetypeStore, clusterInstancetypeStore, revisionStore, virtClient),
		preferenceFindHandler:   preferenceFind.NewObjectFinder(preferenceStore, clusterPreferenceStore, revisionStore, virtClient),
		applyVMHandler: apply.NewVMApplier(
			find.NewSpecFinder(instancetypeStore, clusterInstancetypeStore, revisionStore, virtClient),
			preferenceFind.NewSpecFinder(preferenceStore, clusterInstancetypeStore, revisionStore, virtClient),
		),
	}
}

const (
	none                    = "<none>"
	other                   = "<other>"
	instancetypeVendorLabel = "instancetype.kubevirt.io/vendor"
)

var whitelistedVendors = map[string]bool{
	"kubevirt.io": true,
	"redhat.com":  true,
}

func whitelistName(obj metav1.Object) string {
	labels := obj.GetLabels()
	if labels == nil {
		return none
	}
	vendorName, ok := labels[instancetypeVendorLabel]
	if !ok {
		return none
	}
	if _, isWhitelisted := whitelistedVendors[vendorName]; isWhitelisted {
		return obj.GetName()
	}
	return other
}

func (m *metricsHandler) FetchNameFromVM(vm *virtv1.VirtualMachine) string {
	obj, err := m.Find(vm)
	if err != nil || obj == nil {
		return none
	}
	return whitelistName(obj)
}

func (m *metricsHandler) FetchPreferenceNameFromVM(vm *virtv1.VirtualMachine) string {
	obj, err := m.FindPreference(vm)
	if err != nil || obj == nil {
		return none
	}
	return whitelistName(obj)
}

func (m *metricsHandler) FetchNameFromVMI(vmi *virtv1.VirtualMachineInstance) string {
	obj, err := m.FindFromVMI(vmi)
	if err != nil || obj == nil {
		return none
	}
	return whitelistName(obj)
}

func (m *metricsHandler) FetchPreferenceNameFromVMI(vmi *virtv1.VirtualMachineInstance) string {
	obj, err := m.FindPreferenceFromVMI(vmi)
	if err != nil || obj == nil {
		return none
	}
	return whitelistName(obj)
}
