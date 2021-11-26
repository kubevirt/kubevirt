package network_test

import (
	"io/ioutil"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/network/cache"
	network "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("Hotplug Network Interfaces controller", func() {
	const (
		vmName = "tiny-winy-vm"
	)

	var (
		ctrl            *gomock.Controller
		ifaceController *network.ConcreteController
		tmpDir          string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		ifaceController = network.NewInterfaceController(cache.CacheCreator{}, nsNoopFactory)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
		ctrl.Finish()
	})

	Context("an existing virtual machine instance", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI(vmName)
		})

		DescribeTable("when the interface status features an hotplug request", func(vmiIfaceStatus []v1.VirtualMachineInstanceNetworkInterface, expectedVMIIfaceStatus []v1.VirtualMachineInstanceNetworkInterface) {
			const launcherPID = 1

			vmi.Status.Interfaces = vmiIfaceStatus
			Expect(ifaceController.HotplugIfaces(vmi, launcherPID)).To(Succeed())
			Expect(vmi.Status.Interfaces).To(Equal(expectedVMIIfaceStatus))
		},
			Entry(
				"pending doesn't change",
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:          "pepe",
					InterfaceName: "eth1",
				}},
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:          "pepe",
					InterfaceName: "eth1",
				}},
			),
			Entry(
				"attached to pod advances the state",
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:          "pepe",
					InterfaceName: "eth1",
				}},
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:          "pepe",
					InterfaceName: "eth1",
				}},
			),
		)

		DescribeTable("InterfacesToHotplug", func(vmi *v1.VirtualMachineInstance, expectedNetworksToHotplug ...v1.Network) {
			Expect(network.InterfacesToHotplug(vmi)).To(ConsistOf(expectedNetworksToHotplug))
		},
			Entry("VMI without networks in spec does not have anything to hotplug", libvmi.NewAlpine()),
			Entry(
				"VMI with networks in spec that are not represented in status identifies those are attachments to plug",
				libvmi.NewAlpine(
					libvmi.WithNetwork(&v1.Network{
						Name: "n1",
						NetworkSource: v1.NetworkSource{
							Multus: &v1.MultusNetwork{
								NetworkName: "nad1",
							},
						},
					}),
					libvmi.WithInterface(
						v1.Interface{
							Name:                   "n1",
							InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
						},
					),
				),
				v1.Network{
					Name: "n1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "nad1",
						},
					},
				},
			),
		)
	})

	const (
		guestIfaceName         = "eno123"
		sampleNetworkName      = "n1"
		sampleNetAttachDefName = "nad1"
	)

	DescribeTable("ReadyInterfacesToHotplug", func(vmi *v1.VirtualMachineInstance, expectedNetworksToHotplug ...v1.Network) {
		Expect(network.ReadyInterfacesToHotplug(vmi)).To(ConsistOf(expectedNetworksToHotplug))
	},
		Entry("VMI without networks in spec does not have anything to hotplug", libvmi.NewAlpine()),
		Entry(
			"VMI with networks in spec that are not represented in status does not have to hotplug anything",
			libvmi.NewAlpine(
				libvmi.WithNetwork(&v1.Network{
					Name: "n1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "nad1",
						},
					},
				}),
				libvmi.WithInterface(
					v1.Interface{
						Name:                   "n1",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
					},
				),
			),
		),
		Entry(
			"VMI with networks in spec that are readily available in the pod should hotplug an attachment",
			vmiWithAttachmentToPlug(sampleNetworkName, sampleNetAttachDefName, guestIfaceName),
			v1.Network{
				Name: sampleNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: sampleNetAttachDefName,
					},
				},
			},
		),
	)
})

func vmiWithAttachmentToPlug(networkName string, netAttachDefName string, guestIfaceName string) *v1.VirtualMachineInstance {
	vmi := libvmi.NewAlpine(
		libvmi.WithNetwork(&v1.Network{
			Name: networkName,
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{
					NetworkName: netAttachDefName,
				},
			},
		}),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   networkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			},
		),
	)
	vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
		{Name: networkName, InterfaceName: guestIfaceName, Ready: true},
	}
	return vmi
}
