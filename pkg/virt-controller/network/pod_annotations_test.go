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
 * Copyright the KubeVirt Authors.
 *
 */

package network_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"
)

var _ = Describe("pod annotations", func() {

	Context("Generate pod network annotations", func() {
		const (
			deviceInfoPlugin    = "deviceinfo"
			nonDeviceInfoPlugin = "non_deviceinfo"
		)

		bindingPlugins := map[string]virtv1.InterfaceBindingPlugin{
			deviceInfoPlugin:    {DownwardAPI: virtv1.DeviceInfo},
			nonDeviceInfoPlugin: {},
		}

		const networkStatus = `[
      {
        "name": "default/no-device-info",
        "interface": "pod6446d58d6df",
        "mac": "8a:37:d9:e7:0f:18",
        "dns": {}
      },
      {
        "name": "default/with-device-info",
        "interface": "pod2c26b46b68f",
        "dns": {},
        "device-info": {
          "type": "pci",
          "version": "1.0.0",
          "pci": {
            "pci-address": "0000:65:00.2"
          }
        }
      },
      {
        "name": "default/sriov",
        "interface": "pod778c553efa0",
        "dns": {},
        "device-info": {
          "type": "pci",
          "version": "1.0.0",
          "pci": {
            "pci-address": "0000:65:00.3"
          }
        }
      },
      {
        "name": "default/br-net",
        "interface": "podeeea394806a",
        "mac": "6a:1f:28:23:58:40",
        "dns": {}
      }
  ]`

		It("should be empty when there are no networks", func() {
			networks := []virtv1.Network{}

			interfaces := []virtv1.Interface{}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(BeEmpty())
		})
		It("should be empty when there are no networks with binding plugin/SRIOV", func() {
			networks := []virtv1.Network{
				*libvmi.MultusNetwork("boo", "default/no-device-info"),
			}

			interfaces := []virtv1.Interface{
				newInterface("boo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(BeEmpty())
		})
		It("should be empty when there are networks with binding plugin but none with device-info", func() {
			networks := []virtv1.Network{
				*libvmi.MultusNetwork("boo", "default/no-device-info"),
			}

			interfaces := []virtv1.Interface{
				libvmi.InterfaceWithBindingPlugin(
					"boo", virtv1.PluginBinding{Name: nonDeviceInfoPlugin},
				),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(BeEmpty())
		})
		It("should have network-info entry when there is one binding plugin with device info", func() {
			networks := []virtv1.Network{
				*libvmi.MultusNetwork("foo", "default/with-device-info"),
			}

			interfaces := []virtv1.Interface{
				libvmi.InterfaceWithBindingPlugin(
					"foo", virtv1.PluginBinding{Name: deviceInfoPlugin},
				),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(Equal(map[string]string{
				downwardapi.NetworkInfoAnnot: `{"interfaces":[{"network":"foo","deviceInfo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}]}`,
			}))
		})
		It("should have network-info entry when there is one SR-IOV interface", func() {
			networks := []virtv1.Network{
				*libvmi.MultusNetwork("doo", "default/sriov"),
			}

			interfaces := []virtv1.Interface{
				libvmi.InterfaceDeviceWithSRIOVBinding("doo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(Equal(map[string]string{
				downwardapi.NetworkInfoAnnot: `{"interfaces":[{"network":"doo","deviceInfo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.3"}}}]}`,
			}))
		})
		It("should have network-info entry when there is SR-IOV interface and binding plugin interface with device-info", func() {
			networks := []virtv1.Network{
				*libvmi.MultusNetwork("foo", "default/with-device-info"),
				*libvmi.MultusNetwork("doo", "default/sriov"),
				*libvmi.MultusNetwork("boo", "default/no-device-info"),
				*libvmi.MultusNetwork("goo", "default/br-net"),
			}

			interfaces := []virtv1.Interface{
				libvmi.InterfaceWithBindingPlugin(
					"foo", virtv1.PluginBinding{Name: deviceInfoPlugin},
				),
				libvmi.InterfaceDeviceWithSRIOVBinding("doo"),
				libvmi.InterfaceWithBindingPlugin(
					"boo", virtv1.PluginBinding{Name: nonDeviceInfoPlugin},
				),
				libvmi.InterfaceDeviceWithBridgeBinding("goo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus, bindingPlugins)
			Expect(podAnnotationMap).To(HaveLen(1))
			Expect(podAnnotationMap).To(HaveKey(downwardapi.NetworkInfoAnnot))

			var actualNetInfo downwardapi.NetworkInfo
			Expect(json.Unmarshal([]byte(podAnnotationMap[downwardapi.NetworkInfoAnnot]), &actualNetInfo)).To(Succeed())
			Expect(actualNetInfo.Interfaces).To(ConsistOf([]downwardapi.Interface{
				{Network: "foo", DeviceInfo: &networkv1.DeviceInfo{Type: "pci", Version: "1.0.0",
					Pci: &networkv1.PciDevice{PciAddress: "0000:65:00.2"}}},
				{Network: "doo", DeviceInfo: &networkv1.DeviceInfo{Type: "pci", Version: "1.0.0",
					Pci: &networkv1.PciDevice{PciAddress: "0000:65:00.3"}}},
			}))
		})
	})
})

func newInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name: name,
	}
}
