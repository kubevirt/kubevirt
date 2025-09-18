/*
 * This file is part of the kubevirt project
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

package libnet

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

func VerifyDynamicInterfaceChange(
	vmi *v1.VirtualMachineInstance,
	queueCount int32,
	timeout,
	pollInterval time.Duration,
) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	secondaryNetsByName := vmispec.IndexNetworkSpecByName(vmispec.FilterMultusNonDefaultNetworks(vmi.Spec.Networks))

	nonAbsentSecondaryIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		_, isSecondaryIface := secondaryNetsByName[iface.Name]
		return iface.State != v1.InterfaceStateAbsent && isSecondaryIface
	})
	ExpectWithOffset(1, nonAbsentSecondaryIfaces).NotTo(BeEmpty())

	EventuallyWithOffset(1, func() []v1.VirtualMachineInstanceNetworkInterface {
		return normalizeIfaceStatuses(vmiCurrentInterfaces(vmi.GetNamespace(), vmi.GetName()))
	}).
		WithTimeout(timeout).
		WithPolling(pollInterval).
		Should(
			ConsistOf(
				interfaceStatusFromInterfaces(queueCount, nonAbsentSecondaryIfaces),
			))

	vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return vmi
}

func WaitForSingleHotPlugIfaceOnVMISpec(vmi *v1.VirtualMachineInstance, ifaceName, nadName string) *v1.VirtualMachineInstance {
	EventuallyWithOffset(1, func() []v1.Network {
		var err error
		vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return vmi.Spec.Networks
	}, 30*time.Second).Should(
		ConsistOf(
			*v1.DefaultPodNetwork(),
			v1.Network{
				Name: ifaceName,
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
					NetworkName: nadName,
				}},
			},
		))
	return vmi
}

func vmiCurrentInterfaces(vmiNamespace, vmiName string) []v1.VirtualMachineInstanceNetworkInterface {
	vmi, err := kubevirt.Client().VirtualMachineInstance(vmiNamespace).Get(context.Background(), vmiName, metav1.GetOptions{})
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	return secondaryInterfaces(vmi)
}

func secondaryInterfaces(vmi *v1.VirtualMachineInstance) []v1.VirtualMachineInstanceNetworkInterface {
	indexedSecondaryNetworks := indexVMsSecondaryNetworks(vmi)

	var nonDefaultInterfaces []v1.VirtualMachineInstanceNetworkInterface
	for _, iface := range vmi.Status.Interfaces {
		if _, isNonDefaultPodNetwork := indexedSecondaryNetworks[iface.Name]; isNonDefaultPodNetwork {
			nonDefaultInterfaces = append(nonDefaultInterfaces, iface)
		}
	}
	return nonDefaultInterfaces
}

func indexVMsSecondaryNetworks(vmi *v1.VirtualMachineInstance) map[string]v1.Network {
	indexedSecondaryNetworks := map[string]v1.Network{}
	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil && !network.Multus.Default {
			indexedSecondaryNetworks[network.Name] = network
		}
	}
	return indexedSecondaryNetworks
}

func normalizeIfaceStatuses(status []v1.VirtualMachineInstanceNetworkInterface) []v1.VirtualMachineInstanceNetworkInterface {
	for i := range status {
		status[i].MAC = ""
		status[i].InterfaceName = ""
		status[i].IP = ""
		status[i].IPs = nil
	}
	return status
}

func interfaceStatusFromInterfaces(queueCount int32, ifaces []v1.Interface) []v1.VirtualMachineInstanceNetworkInterface {
	var ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	for _, iface := range ifaces {
		newIfaceStatus := v1.VirtualMachineInstanceNetworkInterface{
			Name: iface.Name,
			InfoSource: vmispec.NewInfoSource(
				vmispec.InfoSourceDomain, vmispec.InfoSourceGuestAgent, vmispec.InfoSourceMultusStatus),
			QueueCount:       queueCount,
			PodInterfaceName: namescheme.GenerateHashedInterfaceName(iface.Name),
		}

		if iface.SRIOV == nil {
			newIfaceStatus.LinkState = normalizeState(iface.State)
		}

		ifaceStatuses = append(ifaceStatuses, newIfaceStatus)
	}
	return ifaceStatuses
}

func normalizeState(state v1.InterfaceState) string {
	if state == "" {
		return "up"
	}
	return string(state)
}

func PatchVMWithNewInterface(vm *v1.VirtualMachine, newNetwork v1.Network, newIface v1.Interface) error {
	patchData, err := patch.New(
		patch.WithTest("/spec/template/spec/networks", vm.Spec.Template.Spec.Networks),
		patch.WithReplace("/spec/template/spec/networks", append(vm.Spec.Template.Spec.Networks, newNetwork)),
		patch.WithTest("/spec/template/spec/domain/devices/interfaces", vm.Spec.Template.Spec.Domain.Devices.Interfaces),
		patch.WithReplace("/spec/template/spec/domain/devices/interfaces", append(vm.Spec.Template.Spec.Domain.Devices.Interfaces, newIface)),
	).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = kubevirt.Client().VirtualMachine(vm.Namespace).Patch(
		context.Background(),
		vm.Name,
		types.JSONPatchType,
		patchData,
		metav1.PatchOptions{},
	)
	return err
}
