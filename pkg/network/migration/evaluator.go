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
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	k8scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

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
) (k8scorev1.ConditionStatus, string) {
	result, reason := shouldVMIBeMarkedForAutoMigration(
		vmi.Spec.Domain.Devices.Interfaces,
		vmi.Spec.Networks,
		vmi.Status.Interfaces,
		vmi.Namespace,
		pod,
		e.clusterConfigurer.LiveUpdateNADRefEnabled(),
	)

	switch result {
	case notRequired:
		return k8scorev1.ConditionUnknown, ""
	case immediateMigration:
		return k8scorev1.ConditionTrue, reason
	case pendingMigration:
		existingCondition := lookupMigrationRequiredCondition(vmi.Status.Conditions)
		if existingCondition != nil &&
			existingCondition.Status == k8scorev1.ConditionFalse &&
			e.timeProvider().Sub(existingCondition.LastTransitionTime.Time) > DynamicNetworkControllerGracePeriod {
			return k8scorev1.ConditionTrue, reason
		}

		return k8scorev1.ConditionFalse, ""
	}

	return k8scorev1.ConditionUnknown, ""
}

func shouldVMIBeMarkedForAutoMigration(
	ifaces []v1.Interface,
	nets []v1.Network,
	ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface,
	namespace string,
	pod *k8scorev1.Pod,
	isLiveUpdateNADRefEnabled bool,
) (migrationRequirementKind, string) {
	secondaryIfaces := vmispec.FilterInterfacesByNetworks(
		ifaces,
		vmispec.FilterMultusNonDefaultNetworks(nets),
	)

	ifaceStatusesByName := vmispec.IndexInterfaceStatusByName(ifaceStatuses, nil)
	netsByName := vmispec.IndexNetworkSpecByName(nets)

	for _, iface := range secondaryIfaces {
		ifaceStatus, ifaceStatusExists := ifaceStatusesByName[iface.Name]
		if iface.State != v1.InterfaceStateAbsent && !ifaceStatusExists {
			if iface.SRIOV != nil {
				return immediateMigration, "Live migration due to change in Interface"
			}

			return pendingMigration, "Live migration due to change in Interface"
		}

		if iface.State == v1.InterfaceStateAbsent &&
			ifaceStatusExists &&
			vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus) &&
			!vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain) {
			return pendingMigration, "Live migration due to change in Interface"
		}

		if isLiveUpdateNADRefEnabled {
			if iface.State == v1.InterfaceStateAbsent || !ifaceStatusExists {
				continue
			}
			net, netExists := netsByName[iface.Name]
			if !netExists || net.Multus == nil {
				continue
			}
			nadNameFromPod := nadNameFromPod(pod, ifaceStatus.PodInterfaceName)
			if !isNADNameEqual(net.Multus.NetworkName, nadNameFromPod, namespace) {
				return immediateMigration, "Live migration due to change in Network"
			}
		}
	}
	return notRequired, ""
}

func nadNameFromPod(pod *k8scorev1.Pod, ifaceName string) string {
	if pod == nil {
		return ""
	}
	annot, ok := pod.Annotations[networkv1.NetworkStatusAnnot]
	if !ok {
		return ""
	}
	var elements []networkv1.NetworkStatus
	if err := json.Unmarshal([]byte(annot), &elements); err != nil {
		return ""
	}
	for _, element := range elements {
		if element.Interface == ifaceName {
			return element.Name
		}
	}
	return ""
}

func isNADNameEqual(nameFromSpec, nameFromPod, vmiNamespace string) bool {
	if nameFromSpec == nameFromPod {
		return true
	}
	if !strings.Contains(nameFromSpec, "/") {
		nameFromSpec = fmt.Sprintf("%s/%s", vmiNamespace, nameFromSpec)
	}
	return nameFromSpec == nameFromPod
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
