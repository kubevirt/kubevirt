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

package sriov_test

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/sriov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("SRIOV", func() {

	const (
		booHashedIfaceName  = "pod6446d58d6df"
		fooHashedIfaceName  = "pod2c26b46b68f"
		fooOrdinalIfaceName = "net1"
		booOrdinalIfaceName = "net2"
	)
	networkStatusWithOneSRIOVNetworkFmt := `
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
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:04:02.5"
    }
  }
}
]`
	networkStatusWithTwoSRIOVNetworksFmt := `
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
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:04:02.5"
    }
  }
},
{
  "name": "default/nad2",
  "interface": "%s",
  "dns": {},
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:04:02.2"
    }
  }
}
]`
	networkStatusWithOneBridgeOneSRIOVNetworksFmt := `
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
  "name": "default/bridge-network",
  "interface": "%s",
  "mac": "8a:37:d9:e7:0f:18",
  "dns": {}
},
{
  "name": "default/sriov-network-vlan100",
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
	networkStatusWithTwoSRIOVNetworksButOneWithNoPCIDataFmt := `
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
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:04:02.5"
    }
  }
},
{
  "name": "default/nad1",
  "interface": "%s",
  "dns": {},
  "device-info": {
    "type": "pci",
    "version": "1.0.0"
  }
}
]`
	networkStatusWithTwoSRIOVNetworksButOneWithNoDeviceInfoDataFmt := `
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
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:04:02.5"
    }
  }
},
{
  "name": "default/nad1",
  "interface": "%s",
  "dns": {}
}
]`
	DescribeTable("should fail to prepare network pci map on the pod network-pci-map anotation",
		func(networkList []virtv1.Network, interfaceList []virtv1.Interface, networkStatusAnnotationValue string) {
			networkPCIAnnotationValue := sriov.CreateNetworkPCIAnnotationValue(networkList, interfaceList, networkStatusAnnotationValue)
			Expect(networkPCIAnnotationValue).To(Equal("{}"))
		},
		Entry("when networkStatusAnnotation is valid but with no pci data on one of the SRIOV interfaces",
			[]virtv1.Network{
				newMultusNetwork("foo", "default/nad1"),
				newMultusNetwork("boo", "default/nad2"),
			},
			[]virtv1.Interface{
				newSRIOVInterface("foo"),
				newSRIOVInterface("boo"),
			},
			fmt.Sprintf(networkStatusWithTwoSRIOVNetworksButOneWithNoPCIDataFmt, fooHashedIfaceName, booHashedIfaceName),
		),
		Entry("when networkStatusAnnotation is valid but with no device-info data on one of the SRIOV interfaces",
			[]virtv1.Network{
				newMultusNetwork("foo", "default/nad1"),
				newMultusNetwork("boo", "default/nad2"),
			},
			[]virtv1.Interface{
				newSRIOVInterface("foo"),
				newSRIOVInterface("boo"),
			},
			fmt.Sprintf(networkStatusWithTwoSRIOVNetworksButOneWithNoDeviceInfoDataFmt, fooHashedIfaceName, booHashedIfaceName),
		),
	)

	DescribeTable("should succeed to prepare network pci map on pod's network-pci-map",
		func(networkList []virtv1.Network, interfaceList []virtv1.Interface, networkStatusAnnotationValue, expectedPciMapString string) {
			Expect(sriov.CreateNetworkPCIAnnotationValue(networkList, interfaceList, networkStatusAnnotationValue)).To(Equal(expectedPciMapString))
		},
		Entry("when given Interfaces{1X masquarade(primary),1X SRIOV}; Networks{1X masquarade(primary),1X Multus} 1xNAD",
			[]virtv1.Network{newMasqueradeDefaultNetwork(), newMultusNetwork("foo", "default/nad1")},
			[]virtv1.Interface{newMasqueradePrimaryInterface(), newSRIOVInterface("foo")},
			fmt.Sprintf(networkStatusWithOneSRIOVNetworkFmt, fooHashedIfaceName),
			`{"foo":"0000:04:02.5"}`,
		),
		Entry("when given Interfaces{1X masquarade(primary),2X SRIOV}, Networks{1X masquarade(primary),2X Multus}, 2xNAD",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("foo", "default/nad1"),
				newMultusNetwork("boo", "default/nad2"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newSRIOVInterface("boo"), newSRIOVInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithTwoSRIOVNetworksFmt, fooHashedIfaceName, booHashedIfaceName),
			`{"boo":"0000:04:02.2","foo":"0000:04:02.5"}`,
		),
		Entry("when given Interfaces{1X masquarade(primary),1X SRIOV, 1X Bridge}  Networks{1X masquarade(primary),2X Multus}, 2xNAD",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("boo", "default/nad1"),
				newMultusNetwork("foo", "default/nad2"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newBridgeInterface("boo"), newSRIOVInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithOneBridgeOneSRIOVNetworksFmt, booHashedIfaceName, fooHashedIfaceName),
			`{"foo":"0000:65:00.2"}`,
		),
		Entry("given 1 primary masquerade, 1 SR-IOV interfaces and pod network status with ordinal names",
			[]virtv1.Network{newMasqueradeDefaultNetwork(), newMultusNetwork("foo", "default/nad1")},
			[]virtv1.Interface{newMasqueradePrimaryInterface(), newSRIOVInterface("foo")},
			fmt.Sprintf(networkStatusWithOneSRIOVNetworkFmt, fooOrdinalIfaceName),
			`{"foo":"0000:04:02.5"}`,
		),
		Entry("given 1 primary masquerade, 2 SR-IOV interfaces and pod network status with ordinal names",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("foo", "default/nad1"),
				newMultusNetwork("boo", "default/nad2"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newSRIOVInterface("boo"), newSRIOVInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithTwoSRIOVNetworksFmt, fooOrdinalIfaceName, booOrdinalIfaceName),
			`{"boo":"0000:04:02.2","foo":"0000:04:02.5"}`,
		),
		Entry("given 1 primary masquerade, 1 SR-IOV, 1 bridge interfaces and pod network status with ordinal names",
			[]virtv1.Network{
				newMasqueradeDefaultNetwork(),
				newMultusNetwork("boo", "default/nad1"),
				newMultusNetwork("foo", "default/nad2"),
			},
			[]virtv1.Interface{
				newMasqueradePrimaryInterface(),
				newBridgeInterface("boo"), newSRIOVInterface("foo"),
			},
			fmt.Sprintf(networkStatusWithOneBridgeOneSRIOVNetworksFmt, fooOrdinalIfaceName, booOrdinalIfaceName),
			`{"foo":"0000:65:00.2"}`,
		),
		Entry("when pod's networkStatus Annotation does not exist",
			[]virtv1.Network{newMultusNetwork("foo", "default/nad1")},
			[]virtv1.Interface{newSRIOVInterface("foo")},
			"",
			`{}`,
		),
		Entry("when networkStatusAnnotation is valid but one SR-IOV entry is missing",
			[]virtv1.Network{
				newMultusNetwork("foo", "default/nad1"),
				newMultusNetwork("boo", "default/nad2"),
			},
			[]virtv1.Interface{
				newSRIOVInterface("foo"),
				newSRIOVInterface("boo"),
			},
			fmt.Sprintf(networkStatusWithOneSRIOVNetworkFmt, fooHashedIfaceName),
			`{"foo":"0000:04:02.5"}`,
		),
	)
})

func newSRIOVInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:                   name,
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{SRIOV: &virtv1.InterfaceSRIOV{}},
	}
}

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
