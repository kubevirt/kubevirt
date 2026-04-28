/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package migration

import (
	"fmt"
	"slices"
	"strings"
	"time"

	k8scorev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"

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

type clusterConfigurer interface {
	LiveUpdateNADRefEnabled() bool
}

type Evaluator struct {
	timeProvider      timeProviderFunc
	clusterConfigurer clusterConfigurer
}

func NewEvaluator(clusterConfigurer clusterConfigurer) Evaluator {
	return NewEvaluatorWithTimeProvider(metav1.Now, clusterConfigurer)
}

func NewEvaluatorWithTimeProvider(timeProvider timeProviderFunc, clusterConfigurer clusterConfigurer) Evaluator {
	return Evaluator{
		timeProvider:      timeProvider,
		clusterConfigurer: clusterConfigurer,
	}
}

func (e Evaluator) Evaluate(vmi *v1.VirtualMachineInstance,
	pod *k8scorev1.Pod,
) k8scorev1.ConditionStatus {
	result := shouldVMIBeMarkedForAutoMigration(
		vmi,
		pod,
		e.clusterConfigurer.LiveUpdateNADRefEnabled(),
	)

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

func shouldVMIBeMarkedForAutoMigration(
	vmi *v1.VirtualMachineInstance,
	pod *k8scorev1.Pod,
	isLiveUpdateNADRefEnabled bool,
) migrationRequirementKind {
	ifaces := vmi.Spec.Domain.Devices.Interfaces
	nets := vmi.Spec.Networks
	ifaceStatuses := vmi.Status.Interfaces
	namespace := vmi.Namespace

	secondaryIfaces := vmispec.FilterInterfacesByNetworks(
		ifaces,
		vmispec.FilterMultusNonDefaultNetworks(nets),
	)

	ifaceStatusesByName := vmispec.IndexInterfaceStatusByName(ifaceStatuses, nil)
	netsByName := vmispec.IndexNetworkSpecByName(nets)
	netStatusByPodIfaceName := multus.NetworkStatusesByPodIfaceName(multus.NetworkStatusesFromPod(pod))

	for _, iface := range secondaryIfaces {
		ifaceStatus, ifaceStatusExists := ifaceStatusesByName[iface.Name]

		if result := shouldMigrateOnIfaceHotplug(iface, ifaceStatusExists); result != notRequired {
			return result
		}

		if result := shouldMigrateOnIfaceUnplug(iface, ifaceStatus, ifaceStatusExists); result != notRequired {
			return result
		}

		if !isLiveUpdateNADRefEnabled {
			continue
		}

		net := netsByName[iface.Name]

		podIfaceName := ifaceStatus.PodInterfaceName
		if podIfaceName == "" {
			continue
		}

		podNetStatus, exists := netStatusByPodIfaceName[podIfaceName]
		if !exists {
			continue
		}

		if !isNADNameEqual(net.Multus.NetworkName, podNetStatus.Name, namespace) {
			return immediateMigration
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

func isNADNameEqual(nameFromVMISpec, nameFromPodNetworkStatus, vmiNamespace string) bool {
	if nameFromVMISpec == nameFromPodNetworkStatus {
		return true
	}
	if !strings.Contains(nameFromVMISpec, "/") {
		vmiNADNameWithNamespace := fmt.Sprintf("%s/%s", vmiNamespace, nameFromVMISpec)
		return vmiNADNameWithNamespace == nameFromPodNetworkStatus
	}
	return false
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
