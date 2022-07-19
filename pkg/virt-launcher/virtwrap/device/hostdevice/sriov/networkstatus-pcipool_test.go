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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package sriov_test

import (
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

var _ = Describe("SRIOV PCI address pool", func() {
	type newPCIAddressPoolWithNetworkStatusParams struct {
		networkStatusAnnotationValue string
		sriovInterfaceList           []v1.Interface
		multusNetworkList            []v1.Network
	}
	type popWithNetworkStatusAnnotationParams struct {
		pciAddressPoolParams newPCIAddressPoolWithNetworkStatusParams
		requestedNetworkName string
		expectedPciAddress   string
	}
	Context("And network-status annotation is present on the vmi", func() {
		nonValidNetworkStatusAnnotation := ``
		validNetworkStatusAnnotationOneInterface := `
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
    "interface": "net1",
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
		validNetworkStatusAnnotationTwoSriovInterfaces := `
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
    "interface": "net1",
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
    "interface": "net2",
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
		validNetworkStatusAnnotationOneBridgeNetworkOneSriovInterfaces := `
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
    "interface": "net1",
    "mac": "8a:37:d9:e7:0f:18",
    "dns": {}
  },
  {
    "name": "default/sriov-network-vlan100",
    "interface": "net2",
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
		validNetworkStatusAnnotationTwoInterfacesAndOneWithNoPciData := `
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
    "interface": "net1",
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
    "interface": "net2",
    "dns": {},
    "device-info": {
      "type": "pci",
      "version": "1.0.0"
    }
  }
]`
		validNetworkStatusAnnotationTwoInterfacesAndOneWithNoDeviceInfoData := `
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
    "interface": "net1",
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
    "interface": "net2",
    "dns": {}
  }
]`
		emptyPciAddress := ""

		DescribeTable("should fail to create the pool", func(params newPCIAddressPoolWithNetworkStatusParams) {
			vmiAnnotations := map[string]string{networkv1.NetworkStatusAnnot: params.networkStatusAnnotationValue}
			pool, err := sriov.NewPCIAddressPoolWithNetworkStatus(params.sriovInterfaceList, params.multusNetworkList, vmiAnnotations)

			Expect(err).ToNot(Succeed())
			Expect(pool).To(BeNil())
		},
			Entry("when networkStatusAnnotation is invalid",
				newPCIAddressPoolWithNetworkStatusParams{
					networkStatusAnnotationValue: nonValidNetworkStatusAnnotation,
					sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo")},
					multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1")},
				}),
			Entry("when networkStatusAnnotation is valid but with no pci data on one of the SRIOV interfaces",
				newPCIAddressPoolWithNetworkStatusParams{
					networkStatusAnnotationValue: validNetworkStatusAnnotationTwoInterfacesAndOneWithNoPciData,
					sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo"), newSRIOVInterface("boo")},
					multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1"), newMultusNetwork("boo", "default/nad2")},
				}),
			Entry("when networkStatusAnnotation is valid but with no device-info data on one of the SRIOV interfaces",
				newPCIAddressPoolWithNetworkStatusParams{
					networkStatusAnnotationValue: validNetworkStatusAnnotationTwoInterfacesAndOneWithNoDeviceInfoData,
					sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo"), newSRIOVInterface("boo")},
					multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1"), newMultusNetwork("boo", "default/nad2")},
				}),
		)

		DescribeTable("should fail to pop a pci-address from the pool", func(params popWithNetworkStatusAnnotationParams) {
			vmiAnnotations := map[string]string{networkv1.NetworkStatusAnnot: params.pciAddressPoolParams.networkStatusAnnotationValue}
			pool, err := sriov.NewPCIAddressPoolWithNetworkStatus(params.pciAddressPoolParams.sriovInterfaceList, params.pciAddressPoolParams.multusNetworkList, vmiAnnotations)
			Expect(err).To(Succeed())

			address, err := pool.Pop(params.requestedNetworkName)
			Expect(err).ToNot(Succeed())
			Expect(address).To(Equal(params.expectedPciAddress))
		},
			Entry("when networkStatusAnnotation is valid but no SRIOV interfaces given",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationOneInterface,
						sriovInterfaceList:           []v1.Interface{},
						multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1")},
					},
					requestedNetworkName: "foo",
					expectedPciAddress:   emptyPciAddress,
				}),
			Entry("when networkStatusAnnotation is valid but no multus interfaces given",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationOneInterface,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{},
					},
					requestedNetworkName: "foo",
					expectedPciAddress:   emptyPciAddress,
				}),
			Entry("when networkStatusAnnotation is valid but the network name is not in the pool",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationOneInterface,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1")},
					},
					requestedNetworkName: "boo",
					expectedPciAddress:   emptyPciAddress,
				}),
		)

		DescribeTable("should succeed to pop a pci-address from the pool", func(params popWithNetworkStatusAnnotationParams) {
			vmiAnnotations := map[string]string{networkv1.NetworkStatusAnnot: params.pciAddressPoolParams.networkStatusAnnotationValue}
			pool, err := sriov.NewPCIAddressPoolWithNetworkStatus(params.pciAddressPoolParams.sriovInterfaceList, params.pciAddressPoolParams.multusNetworkList, vmiAnnotations)
			Expect(err).To(Succeed())

			address, err := pool.Pop(params.requestedNetworkName)
			Expect(err).To(Succeed())
			_, err = pool.Pop(params.requestedNetworkName)
			Expect(err).ToNot(Succeed(), "should return err when trying to pop an already popped interface")

			Expect(address).To(Equal(params.expectedPciAddress))
		},
			Entry("when given 1x SRIOV Interface, 1X Multus network, 1xNAD",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationOneInterface,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1")},
					},
					requestedNetworkName: "foo",
					expectedPciAddress:   "0000:04:02.5",
				}),
			Entry("when given 2x SRIOV Interface, 2X Multus network, 2xNAD (Request first network)",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationTwoSriovInterfaces,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("boo"), newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1"), newMultusNetwork("boo", "default/nad2")},
					},
					requestedNetworkName: "boo",
					expectedPciAddress:   "0000:04:02.2",
				}),
			Entry("when given 2x SRIOV Interface, 2X Multus network, 2xNAD (Request second network)",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationTwoSriovInterfaces,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("boo"), newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{newMultusNetwork("foo", "default/nad1"), newMultusNetwork("boo", "default/nad2")},
					},
					requestedNetworkName: "foo",
					expectedPciAddress:   "0000:04:02.5",
				}),
			Entry("when given 1x SRIOV Interface, 1X Bridge interface, 2X Multus network, 2xNAD",
				popWithNetworkStatusAnnotationParams{
					pciAddressPoolParams: newPCIAddressPoolWithNetworkStatusParams{
						networkStatusAnnotationValue: validNetworkStatusAnnotationOneBridgeNetworkOneSriovInterfaces,
						sriovInterfaceList:           []v1.Interface{newSRIOVInterface("foo")},
						multusNetworkList:            []v1.Network{newMultusNetwork("boo", "default/nad1"), newMultusNetwork("foo", "default/nad2")},
					},
					requestedNetworkName: "foo",
					expectedPciAddress:   "0000:65:00.2",
				}),
		)
	})
})
