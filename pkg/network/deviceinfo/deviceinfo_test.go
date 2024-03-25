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
package deviceinfo_test

import (
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
)

var _ = Describe("DeviceInfo", func() {

	const (
		deviceInfoPlugin    = "deviceinfo"
		nonDeviceInfoPlugin = "non_deviceinfo"
	)

	networkStatusWithMixedNetworks := `[
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
		    "interface": "pod6446d58d6df",
		    "mac": "8a:37:d9:e7:0f:18",
		    "dns": {}
	    },
	    {
		    "name": "default/nad2",
		    "interface": "pod2c26b46b68f",
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

	networkStatusWithPrimaryInterfaceOnly := `[
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

	bindingPlugins := map[string]v1.InterfaceBindingPlugin{
		deviceInfoPlugin:    {DownwardAPI: v1.DeviceInfo},
		nonDeviceInfoPlugin: {},
	}

	DescribeTable("should return an error",
		func(networkStatusAnnotationValue string) {
			networks := []v1.Network{*libvmi.MultusNetwork("foo", "default/nad1")}
			interfaces := []v1.Interface{newBindingPluginInterface("foo", deviceInfoPlugin)}
			_, err := deviceinfo.MapBindingPluginNetworkNameToDeviceInfo(networks, interfaces, networkStatusAnnotationValue, bindingPlugins)
			Expect(err).To(HaveOccurred())
		},
		Entry("when networkStatus annotation is empty", ""),
		Entry("when networkStatus annotation has invalid format", "invalid"),
	)

	DescribeTable("should prepare empty network device info annotation",
		func(networkList []v1.Network, interfaceList []v1.Interface, networkStatusAnnotationValue string) {
			Expect(deviceinfo.MapBindingPluginNetworkNameToDeviceInfo(
				networkList,
				interfaceList,
				networkStatusAnnotationValue,
				bindingPlugins,
			)).To(BeEmpty())
		},
		Entry("when there is no interface with device info binding plugin",
			[]v1.Network{
				*v1.DefaultPodNetwork(),
				*libvmi.MultusNetwork("foo", "default/nad1"),
			},
			[]v1.Interface{
				libvmi.InterfaceDeviceWithMasqueradeBinding(),
				newBindingPluginInterface("foo", nonDeviceInfoPlugin),
			},
			networkStatusWithMixedNetworks,
		),
		Entry("when the interface is not in the multus status",
			[]v1.Network{*libvmi.MultusNetwork("notfoo", "default/nad1")},
			[]v1.Interface{newBindingPluginInterface("notfoo", deviceInfoPlugin)},
			networkStatusWithPrimaryInterfaceOnly,
		),
	)

	It("should prepare network device info annotation with multiple networks", func() {
		networks := []v1.Network{
			*v1.DefaultPodNetwork(),
			*libvmi.MultusNetwork("boo", "default/nad1"),
			*libvmi.MultusNetwork("foo", "default/nad2"),
			*libvmi.MultusNetwork("doo", "default/nad3"),
		}
		interfaces := []v1.Interface{
			libvmi.InterfaceDeviceWithMasqueradeBinding(),
			libvmi.InterfaceDeviceWithBridgeBinding("boo"),
			newBindingPluginInterface("foo", deviceInfoPlugin),
			newBindingPluginInterface("doo", deviceInfoPlugin),
		}
		expectedMap := map[string]*networkv1.DeviceInfo{
			"foo": {Type: "pci", Version: "1.0.0", Pci: &networkv1.PciDevice{PciAddress: "0000:65:00.2"}}}
		Expect(deviceinfo.MapBindingPluginNetworkNameToDeviceInfo(networks, interfaces, networkStatusWithMixedNetworks, bindingPlugins)).
			To(Equal(expectedMap))
	})
})

func newBindingPluginInterface(name, bindingPlugin string) v1.Interface {
	return v1.Interface{
		Name:    name,
		Binding: &v1.PluginBinding{Name: bindingPlugin},
	}
}
