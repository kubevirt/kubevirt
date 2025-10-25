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

package deviceinfo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
)

var _ = Describe("MacAddress", func() {
	const deviceInfoPlugin = "deviceinfo"

	networkStatusWithMixedNetworks := []networkv1.NetworkStatus{
		{
			Name:      "kindnet",
			Interface: "eth0",
			IPs:       []string{"10.244.1.9"},
			Mac:       "3a:7e:42:fa:37:c6",
			Default:   true,
			DNS:       networkv1.DNS{},
		},
		{
			Name:      "default/nad1",
			Interface: "pod6446d58d6df",
			Mac:       "8a:37:d9:e7:0f:18",
			DNS:       networkv1.DNS{},
		},
		{
			Name:      "default/nad2",
			Interface: "pod2c26b46b68f",
			DNS:       networkv1.DNS{},
			DeviceInfo: &networkv1.DeviceInfo{
				Type:    "pci",
				Version: "1.0.0",
				Pci:     &networkv1.PciDevice{PciAddress: "0000:65:00.2"},
			},
		},
		{
			Name:      "default/nad4",
			Interface: "podb260539511e",
			DNS:       networkv1.DNS{},
			Mac:       "8a:2f:36:e7:22:19",
			DeviceInfo: &networkv1.DeviceInfo{
				Type:    "pci",
				Version: "1.0.0",
				Pci:     &networkv1.PciDevice{PciAddress: "0000:65:00.3"},
			},
		},
	}

	networkStatusWithPrimaryInterfaceOnly := []networkv1.NetworkStatus{
		{
			Name:      "kindnet",
			Interface: "eth0",
			IPs:       []string{"10.244.2.131"},
			Mac:       "82:cf:7c:98:43:7e",
			Default:   true,
			DNS:       networkv1.DNS{},
		},
	}

	DescribeTable("should return an empty map",
		func(networkStatuses []networkv1.NetworkStatus) {
			networks := []v1.Network{*libvmi.MultusNetwork("foo", "default/nad1")}
			interfaces := []v1.Interface{newBindingPluginInterface("foo", deviceInfoPlugin)}

			Expect(deviceinfo.MapNetworkNameToDeviceMacAddress(networks, interfaces, networkStatuses)).To(BeEmpty())
		},
		Entry("when networkStatus list is nil", nil),
		Entry("when the interface is not in the multus status", networkStatusWithPrimaryInterfaceOnly),
	)

	It("should return network device info mapping with multiple networks", func() {
		networks := []v1.Network{
			*v1.DefaultPodNetwork(),
			*libvmi.MultusNetwork("boo", "default/nad1"),
			*libvmi.MultusNetwork("foo", "default/nad2"),
			*libvmi.MultusNetwork("doo", "default/nad3"),
			*libvmi.MultusNetwork("woo", "default/nad4"),
		}
		interfaces := []v1.Interface{
			libvmi.InterfaceDeviceWithMasqueradeBinding(),
			libvmi.InterfaceDeviceWithBridgeBinding("boo"),
			newBindingPluginInterface("foo", deviceInfoPlugin),
			newBindingPluginInterface("doo", deviceInfoPlugin),
			newBindingPluginInterface("woo", deviceInfoPlugin),
		}
		expectedMap := map[string]string{
			"default": "3a:7e:42:fa:37:c6",
			"boo":     "8a:37:d9:e7:0f:18",
			"woo":     "8a:2f:36:e7:22:19",
		}
		Expect(deviceinfo.MapNetworkNameToDeviceMacAddress(networks, interfaces, networkStatusWithMixedNetworks)).To(Equal(expectedMap))
	})
})
