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

package migration

import (
	"slices"
	"time"

	k8scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const DynamicNetworkControllerGracePeriod = 15 * time.Second

type migrationRequirementKind int

const (
	notRequired migrationRequirementKind = iota
	immediateMigration
	pendingMigration
)

type timeProviderFunc func() metav1.Time

type Evaluator struct {
	timeProvider timeProviderFunc
}

func NewEvaluator() Evaluator {
	return NewEvaluatorWithTimeProvider(metav1.Now)
}

func NewEvaluatorWithTimeProvider(timeProvider timeProviderFunc) Evaluator {
	return Evaluator{timeProvider: timeProvider}
}

func (e Evaluator) Evaluate(vmi *v1.VirtualMachineInstance) k8scorev1.ConditionStatus {
	result := shouldVMIBeMarkedForAutoMigration(vmi)

	switch result {
	case notRequired:
		return k8scorev1.ConditionUnknown
	case immediateMigration:
		return k8scorev1.ConditionTrue
	case pendingMigration:
		existingCondition := lookupMigrationRequiredCondition(vmi.Status.Conditions)
		if existingCondition != nil &&
			existingCondition.Status == k8scorev1.ConditionFalse &&
			e.timeProvider().Sub(existingCondition.LastTransitionTime.Time) > DynamicNetworkControllerGracePeriod {
			return k8scorev1.ConditionTrue
		}

		return k8scorev1.ConditionFalse
	}

	return k8scorev1.ConditionUnknown
}

func shouldVMIBeMarkedForAutoMigration(vmi *v1.VirtualMachineInstance) migrationRequirementKind {
	ifaces := vmi.Spec.Domain.Devices.Interfaces
	nets := vmi.Spec.Networks
	ifaceStatuses := vmi.Status.Interfaces

	secondaryIfaces := vmispec.FilterInterfacesByNetworks(
		ifaces,
		vmispec.FilterMultusNonDefaultNetworks(nets),
	)

	ifaceStatusesByName := vmispec.IndexInterfaceStatusByName(ifaceStatuses, nil)

	for _, iface := range secondaryIfaces {
		ifaceStatus, ifaceStatusExists := ifaceStatusesByName[iface.Name]

		if result := shouldMigrateOnIfaceHotplug(iface, ifaceStatusExists); result != notRequired {
			return result
		}

		if result := shouldMigrateOnIfaceUnplug(iface, ifaceStatus, ifaceStatusExists); result != notRequired {
			return result
		}
	}
	return notRequired
}

func shouldMigrateOnIfaceHotplug(iface v1.Interface, ifaceStatusExists bool) migrationRequirementKind {
	if iface.State != v1.InterfaceStateAbsent && !ifaceStatusExists {
		if iface.SRIOV != nil {
			return immediateMigration
		}

		return pendingMigration
	}
	return notRequired
}

func shouldMigrateOnIfaceUnplug(
	iface v1.Interface,
	ifaceStatus v1.VirtualMachineInstanceNetworkInterface,
	ifaceStatusExists bool,
) migrationRequirementKind {
	if iface.State == v1.InterfaceStateAbsent &&
		ifaceStatusExists &&
		vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus) &&
		!vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain) {
		return pendingMigration
	}
	return notRequired
}

func lookupMigrationRequiredCondition(conditions []v1.VirtualMachineInstanceCondition) *v1.VirtualMachineInstanceCondition {
	idx := slices.IndexFunc(conditions, func(condition v1.VirtualMachineInstanceCondition) bool {
		return condition.Type == v1.VirtualMachineInstanceMigrationRequired
	})

	if idx == -1 {
		return nil
	}

	return &conditions[idx]
}
