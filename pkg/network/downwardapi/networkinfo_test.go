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

		expectedInterfaces := []downwardapi.Interface{
			{Network: "foo", DeviceInfo: deviceInfoFoo},
			{Network: "boo"},
		}

		annotation := downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap)
		networkInfo := downwardapi.NetworkInfo{}
		err := json.Unmarshal([]byte(annotation), &networkInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(networkInfo.Interfaces).To(ConsistOf(expectedInterfaces))
	})
	It("should create an empty network info annotation value when there are no networks", func() {
		networkDeviceInfoMap := map[string]*networkv1.DeviceInfo{}

		Expect(downwardapi.CreateNetworkInfoAnnotationValue(networkDeviceInfoMap)).To(Equal("{}"))
	})
})
