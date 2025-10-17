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

var _ = Describe("Network info", func() {
	It("should create network info annotation value", func() {
		deviceInfoFoo := &networkv1.DeviceInfo{Type: "fooType"}
		networkDeviceInfoMap := map[string]*networkv1.DeviceInfo{"foo": deviceInfoFoo, "boo": nil}
		networkDeviceMacMap := map[string]string{"foo": "", "boo": ""}

		expectedInterfaces := []downwardapi.Interface{
			{Network: "foo", DeviceInfo: deviceInfoFoo},
			{Network: "boo"},
		}

		annotation := downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap, networkDeviceMacMap)
		networkInfo := downwardapi.NetworkInfo{}
		err := json.Unmarshal([]byte(annotation), &networkInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkInfo.Interfaces).To(ConsistOf(expectedInterfaces))
	})

	It("should create an empty network info annotation value when there are no networks", func() {
		networkDeviceInfoMap := map[string]*networkv1.DeviceInfo{}
		networkDeviceMacMap := map[string]string{}

		Expect(downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap, networkDeviceMacMap)).To(Equal("{}"))
	})

	It("should produce a deterministic and output sorted by network name regardless of the map key order", func() {
		deviceInfo1 := &networkv1.DeviceInfo{Type: "type1"}
		deviceInfo2 := &networkv1.DeviceInfo{Type: "type2"}
		deviceInfo3 := &networkv1.DeviceInfo{Type: "type3"}

		deviceInfoByNetName1 := map[string]*networkv1.DeviceInfo{
			"netA": deviceInfo1,
			"netB": deviceInfo2,
			"netC": deviceInfo3,
		}

		deviceInfoByNetName2 := map[string]*networkv1.DeviceInfo{
			"netC": deviceInfo3,
			"netB": deviceInfo2,
			"netA": deviceInfo1,
		}

		emptyDeviceMacByNetName := map[string]string{}

		annotationValue1 := downwardapi.CreateNetworkInfoAnnotationValue(deviceInfoByNetName1, emptyDeviceMacByNetName)
		annotationValue2 := downwardapi.CreateNetworkInfoAnnotationValue(deviceInfoByNetName2, emptyDeviceMacByNetName)
		Expect(annotationValue1).To(Equal(annotationValue2))

		var actualNetworkInfo downwardapi.NetworkInfo
		Expect(json.Unmarshal([]byte(annotationValue2), &actualNetworkInfo)).To(Succeed())

		expectedNetworkInfo := downwardapi.NetworkInfo{
			Interfaces: []downwardapi.Interface{
				{Network: "netA", DeviceInfo: &networkv1.DeviceInfo{Type: "type1"}},
				{Network: "netB", DeviceInfo: &networkv1.DeviceInfo{Type: "type2"}},
				{Network: "netC", DeviceInfo: &networkv1.DeviceInfo{Type: "type3"}},
			},
		}

		Expect(actualNetworkInfo).To(Equal(expectedNetworkInfo))
	})

	It("should add mac addresses if not empty", func() {
		// mac address + full deviceinfo
		// mac address + empty deviceinfo
		// another case with either of both
		deviceInfoByNetName := map[string]*networkv1.DeviceInfo{
			"netA": &networkv1.DeviceInfo{Type: "type1"},
			"netB": &networkv1.DeviceInfo{},
			"netC": nil,
		}
		deviceMacByNetName := map[string]string{
			"netA": "0c:42:a1:22:a3:52",
			"netC": "0c:42:a1:22:a3:54",
			"netB": "0c:42:a1:22:a3:53",
		}

		expectedNetworkInfo := downwardapi.NetworkInfo{
			Interfaces: []downwardapi.Interface{
				{Network: "netA", DeviceInfo: &networkv1.DeviceInfo{Type: "type1"}, Mac: "0c:42:a1:22:a3:52"},
				{Network: "netB", DeviceInfo: &networkv1.DeviceInfo{}, Mac: "0c:42:a1:22:a3:53"},
				{Network: "netC", Mac: "0c:42:a1:22:a3:54"},
			},
		}

		annotationValue := downwardapi.CreateNetworkInfoAnnotationValue(deviceInfoByNetName, deviceMacByNetName)

		var actualNetworkInfo downwardapi.NetworkInfo
		Expect(json.Unmarshal([]byte(annotationValue), &actualNetworkInfo)).To(Succeed())

		Expect(actualNetworkInfo).To(Equal(expectedNetworkInfo))
	})

	It("should add mac addresses if not empty even without network deviceinfo", func() {
		// mac address + full deviceinfo
		// mac address + empty deviceinfo
		// another case with either of both
		deviceInfoByNetName := map[string]*networkv1.DeviceInfo{}
		deviceMacByNetName := map[string]string{
			"netA": "0c:42:a1:22:a3:52",
			"netC": "0c:42:a1:22:a3:54",
			"netB": "0c:42:a1:22:a3:53",
		}

		expectedNetworkInfo := downwardapi.NetworkInfo{
			Interfaces: []downwardapi.Interface{
				{Network: "netA", Mac: "0c:42:a1:22:a3:52"},
				{Network: "netB", Mac: "0c:42:a1:22:a3:53"},
				{Network: "netC", Mac: "0c:42:a1:22:a3:54"},
			},
		}

		annotationValue := downwardapi.CreateNetworkInfoAnnotationValue(deviceInfoByNetName, deviceMacByNetName)

		var actualNetworkInfo downwardapi.NetworkInfo
		Expect(json.Unmarshal([]byte(annotationValue), &actualNetworkInfo)).To(Succeed())

		Expect(actualNetworkInfo).To(Equal(expectedNetworkInfo))
	})
})
