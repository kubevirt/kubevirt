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

package downwardapi_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"kubevirt.io/kubevirt/pkg/network/downwardapi"
)

const (
	netAMacAddress = "0c:42:a1:22:a3:52"
	netBMacAddress = "0c:42:a1:22:a3:53"
	netCMacAddress = "0c:42:a1:22:a3:54"
	netAName       = "netA"
	netBName       = "netB"
	netCName       = "netC"
	fooNetworkName = "foo"
	booNetworkName = "boo"
	deviceType1    = "type1"
	deviceType2    = "type2"
	deviceType3    = "type3"
	podAInterface  = "pod33219a16a42"
	podBInterface  = "pod034d3d19642"
	podCInterface  = "pod01783857d47"
)

var _ = Describe("Network info", func() {
	It("should create network info annotation value", func() {
		deviceInfoFoo := &networkv1.DeviceInfo{Type: "fooType"}

		networkStatusByNetworkName := map[string]networkv1.NetworkStatus{
			fooNetworkName: {Interface: "pod2c26b46b68f", DeviceInfo: deviceInfoFoo, Mtu: 1500},
			booNetworkName: {Interface: "pod6446d58d6df"},
		}

		expectedInterfaces := []downwardapi.Interface{
			{Network: fooNetworkName, DeviceInfo: deviceInfoFoo, Mtu: 1500},
			{Network: booNetworkName},
		}

		annotation := downwardapi.CreateNetworkInfoAnnotationValue(networkStatusByNetworkName)
		networkInfo := downwardapi.NetworkInfo{}
		err := json.Unmarshal([]byte(annotation), &networkInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkInfo.Interfaces).To(ConsistOf(expectedInterfaces))
	})

	It("should create an empty network info annotation value when there are no networks", func() {
		var networkStatusByNetworkName map[string]networkv1.NetworkStatus

		Expect(downwardapi.CreateNetworkInfoAnnotationValue(networkStatusByNetworkName)).To(Equal("{}"))
	})

	It("should produce a deterministic and output sorted by network name regardless of the map key order", func() {
		deviceInfo1 := &networkv1.DeviceInfo{Type: deviceType1}
		deviceInfo2 := &networkv1.DeviceInfo{Type: deviceType2}
		deviceInfo3 := &networkv1.DeviceInfo{Type: deviceType3}

		networkStatusByNetworkName1 := map[string]networkv1.NetworkStatus{
			netAName: {Interface: podAInterface, Mac: netAMacAddress, DeviceInfo: deviceInfo1, Mtu: 1500},
			netBName: {Interface: podBInterface, Mac: netBMacAddress, DeviceInfo: deviceInfo2, Mtu: 9000},
			netCName: {Interface: podCInterface, Mac: netCMacAddress, DeviceInfo: deviceInfo3},
		}

		networkStatusByNetworkName2 := map[string]networkv1.NetworkStatus{
			netCName: {Interface: podCInterface, Mac: netCMacAddress, DeviceInfo: deviceInfo3},
			netBName: {Interface: podBInterface, Mac: netBMacAddress, DeviceInfo: deviceInfo2, Mtu: 9000},
			netAName: {Interface: podAInterface, Mac: netAMacAddress, DeviceInfo: deviceInfo1, Mtu: 1500},
		}

		annotationValue1 := downwardapi.CreateNetworkInfoAnnotationValue(networkStatusByNetworkName1)
		annotationValue2 := downwardapi.CreateNetworkInfoAnnotationValue(networkStatusByNetworkName2)

		Expect(annotationValue1).To(Equal(annotationValue2))

		var actualNetworkInfo downwardapi.NetworkInfo
		Expect(json.Unmarshal([]byte(annotationValue2), &actualNetworkInfo)).To(Succeed())

		expectedNetworkInfo := downwardapi.NetworkInfo{
			Interfaces: []downwardapi.Interface{
				{Network: netAName, Mac: netAMacAddress, DeviceInfo: &networkv1.DeviceInfo{Type: deviceType1}, Mtu: 1500},
				{Network: netBName, Mac: netBMacAddress, DeviceInfo: &networkv1.DeviceInfo{Type: deviceType2}, Mtu: 9000},
				{Network: netCName, Mac: netCMacAddress, DeviceInfo: &networkv1.DeviceInfo{Type: deviceType3}},
			},
		}

		Expect(actualNetworkInfo).To(Equal(expectedNetworkInfo))
	})
})
