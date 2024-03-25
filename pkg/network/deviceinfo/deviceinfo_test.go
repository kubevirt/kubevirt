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
* Copyright 2022 Red Hat, Inc.
*
 */
package deviceinfo_test

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DeviceInfo", func() {

	const (
		booHashedIfaceName  = "pod6446d58d6df"
		fooHashedIfaceName  = "pod2c26b46b68f"
		fooOrdinalIfaceName = "net1"
		booOrdinalIfaceName = "net2"
	)

	networkStatusWithMixedNetworksFmt := `
	[
	{
	"name": "kindnet",
	"interface": "eth0",
	"ips": [
	  "10.244.1.9"
	],
	"mac": "3a:7e:42:fa:37:c6",
	"default": true,
	"dns": {}
	},
	{
	"name": "default/nad1",
	"interface": "%s",
	"mac": "8a:37:d9:e7:0f:18",
	"dns": {}
	},
	{
	"name": "default/nad2",
	"interface": "%s",
	"dns": {},
	"device-info": {
	  "type": "pci",
	  "version": "1.0.0",
	  "pci": {
	    "pci-address": "0000:65:00.2"
	  }
	}
	}
	]`
	networkStatusWithNoDeviceInfoFmt := `
[
{
 "name": "kindnet",
 "interface": "eth0",
 "ips": [
   "10.244.2.131"
 ],
 "mac": "82:cf:7c:98:43:7e",
 "default": true,
 "dns": {}
},
{
 "name": "default/nad1",
 "interface": "%s",
 "dns": {},
}
]`
	networkStatusWithPrimaryInterfaceOnlyFmt := `
[
{
 "name": "kindnet",
 "interface": "eth0",
 "ips": [
   "10.244.2.131"
 ],
 "mac": "82:cf:7c:98:43:7e",
 "default": true,
 "dns": {}
}
]`
	DescribeTable("should prepare empty network device info annotation",
		func(networkList []virtv1.Network, interfaceList []virtv1.Interface, networkStatusAnnotationValue string) {
			networkDeviceInfoAnnotationValue := deviceinfo.CreateNetworkDeviceInfoAnnotationValue(networkList, interfaceList, networkStatusAnnotationValue)
			Expect(networkDeviceInfoAnnotationValue).To(Equal(""))
		},
		Entry("when networkStatusAnnotation deson't contain the interface",
			[]virtv1.Network{
				newMultusNetwork("foo", "default/nad1"),
			},
			[]virtv1.Interface{
				newBindingPluginInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithPrimaryInterfaceOnlyFmt),
		),
		Entry("when networkStatusAnnotation has no interfaces with device info",
			[]virtv1.Network{
				newMultusNetwork("foo", "default/nad1"),
			},
			[]virtv1.Interface{
				newBindingPluginInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithNoDeviceInfoFmt, fooHashedIfaceName, booHashedIfaceName),
		),
		Entry("when pod's networkStatus Annotation does not exist",
			[]virtv1.Network{newMultusNetwork("foo", "default/nad1")},
			[]virtv1.Interface{newBindingPluginInterface("foo")},
			"",
		),
	)

	DescribeTable("should prepare non empty network device info annotation",
		func(networkList []virtv1.Network, interfaceList []virtv1.Interface, networkStatusAnnotationValue, expectedAnnotation string) {
			Expect(deviceinfo.CreateNetworkDeviceInfoAnnotationValue(networkList, interfaceList, networkStatusAnnotationValue)).To(Equal(expectedAnnotation))
		},

		Entry("when given Interfaces{1X primary- no device info, 1X no device info, 1X with device info, 1X not in multus status}",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("boo", "default/nad1"),
				newMultusNetwork("foo", "default/nad2"),
				newMultusNetwork("doo", "default/nad3"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newBridgeInterface("boo"), newBindingPluginInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithMixedNetworksFmt, booHashedIfaceName, fooHashedIfaceName),
			`{"foo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}`,
		),
		Entry("when given Interfaces{1X primary- no device info, 1X no device info, 1X with device info, 1X not in multus status with ordinal names",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("boo", "default/nad1"),
				newMultusNetwork("foo", "default/nad2"),
				newMultusNetwork("doo", "default/nad3"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newBridgeInterface("boo"), newBindingPluginInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithMixedNetworksFmt, fooOrdinalIfaceName, booOrdinalIfaceName),
			`{"foo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}`,
		),
	)
})

func newBridgeInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:                   name,
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{Bridge: &virtv1.InterfaceBridge{}},
	}
}

func newMasqueradePrimaryInterface() virtv1.Interface {
	return virtv1.Interface{
		Name:                   "testmasquerade",
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{Masquerade: &virtv1.InterfaceMasquerade{}},
	}
}

func newMasqueradeDefaultNetwork() virtv1.Network {
	return virtv1.Network{
		Name: "testmasquerade",
		NetworkSource: virtv1.NetworkSource{
			Pod: &virtv1.PodNetwork{},
		},
	}
}

func newBindingPluginInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:    name,
		Binding: &virtv1.PluginBinding{},
	}
}

func newMultusNetwork(name, networkName string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Multus: &virtv1.MultusNetwork{
				NetworkName: networkName,
			},
		},
	}
}
