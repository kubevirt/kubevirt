package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"

	"kubevirt.io/kubevirt/pkg/virt-controller/network"
)

var _ = Describe("pod annotations", func() {
	const (
		booHashedIfaceName = "pod6446d58d6df"
		fooHashedIfaceName = "pod2c26b46b68f"
	)

	networkStatus := `
[
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
}
]`

	Context("Generate pod network annotations", func() {
		It("should have empty device info annotation when there are no networks", func() {
			networks := []virtv1.Network{}

			interfaces := []virtv1.Interface{}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus)
			Expect(podAnnotationMap).To(HaveLen(1))
			Expect(podAnnotationMap).To(HaveKeyWithValue(deviceinfo.NetworkDeviceInfoMapAnnot, ""))
		})
		It("should have empty device info annotation when there are no networks with device-info", func() {
			networks := []virtv1.Network{
				newMultusNetwork("boo", "default/no-device-info"),
			}

			interfaces := []virtv1.Interface{
				newInterface("boo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus)
			Expect(podAnnotationMap).To(HaveLen(1))
			Expect(podAnnotationMap).To(HaveKeyWithValue(deviceinfo.NetworkDeviceInfoMapAnnot, ""))
		})
		It("should have network-device-info-map entry when there is one non SRIOV interface with device info", func() {
			networks := []virtv1.Network{
				newMultusNetwork("foo", "default/with-device-info"),
			}

			interfaces := []virtv1.Interface{
				newInterface("foo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus)
			Expect(podAnnotationMap).To(HaveLen(1))
			Expect(podAnnotationMap).To(HaveKeyWithValue(deviceinfo.NetworkDeviceInfoMapAnnot, `{"foo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}`))
		})
		It("should have both network-pci-map, network-device-info-map entries when there is SRIOV interface with device info", func() {
			networks := []virtv1.Network{
				newMultusNetwork("foo", "default/with-device-info"),
			}

			interfaces := []virtv1.Interface{
				newSRIOVInterface("foo"),
			}
			podAnnotationMap := network.GeneratePodAnnotations(networks, interfaces, networkStatus)
			Expect(podAnnotationMap).To(HaveLen(2))
			Expect(podAnnotationMap).To(HaveKeyWithValue(deviceinfo.NetworkDeviceInfoMapAnnot, `{"foo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}`))
			Expect(podAnnotationMap).To(HaveKeyWithValue(deviceinfo.NetworkPCIMapAnnot, `{"foo":"0000:65:00.2"}`))
		})
	})
})

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

func newInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name: name,
	}
}

func newSRIOVInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:                   name,
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{SRIOV: &virtv1.InterfaceSRIOV{}},
	}
}
