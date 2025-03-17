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

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("utilitary funcs to identify attachments to hotplug", func() {
	Context("NetworksToHotplugWhosePodIfacesAreReady", func() {
		const (
			guestIfaceName = "eno123"
			nadName        = "nad1"
			networkName    = "n1"
		)
		DescribeTable("NetworksToHotplugWhosePodIfacesAreReady", func(vmi *v1.VirtualMachineInstance, networksToHotplug ...v1.Network) {
			Expect(vmispec.NetworksToHotplugWhosePodIfacesAreReady(vmi)).To(ConsistOf(networksToHotplug))
		},
			Entry("VMI with networks in spec, marked as ready in the status, but not yet available in the domain *is* subject to hotplug",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName)),
					libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name: networkName, InterfaceName: guestIfaceName, InfoSource: vmispec.InfoSourceMultusStatus,
					}))),
				),
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: nadName,
						},
					},
				},
			),
			Entry("VMI with networks in spec, marked as ready in the status, but already present in the domain *not* subject to hotplug",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName)),
					libvmistatus.WithStatus(
						libvmistatus.New(libvmistatus.WithInterfaceStatus(
							v1.VirtualMachineInstanceNetworkInterface{
								Name:          networkName,
								InterfaceName: guestIfaceName,
								InfoSource:    vmispec.NewInfoSource(vmispec.InfoSourceDomain, vmispec.InfoSourceMultusStatus),
							},
						)),
					),
				),
			),
		)
	})
})
