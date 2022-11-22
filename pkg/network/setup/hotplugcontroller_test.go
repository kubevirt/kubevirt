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

		XDescribeTable("when the interface status features an hotplug request", func(vmiIfaceStatus []v1.VirtualMachineInstanceNetworkInterface, expectedVMIIfaceStatus []v1.VirtualMachineInstanceNetworkInterface) {
			const launcherPID = 1

			vmi.Status.Interfaces = vmiIfaceStatus
			Expect(ifaceController.HotplugIfaces(vmi, launcherPID)).To(Succeed())
			Expect(vmi.Status.Interfaces).To(Equal(expectedVMIIfaceStatus))
		},
			Entry(
				"pending doesn't change",
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:             "pepe",
					InterfaceName:    "eth1",
					HotplugInterface: &v1.HotplugInterfaceStatus{},
				}},
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:             "pepe",
					InterfaceName:    "eth1",
					HotplugInterface: &v1.HotplugInterfaceStatus{},
				}},
			),
			Entry(
				"attached to pod advances the state",
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:             "pepe",
					InterfaceName:    "eth1",
					HotplugInterface: &v1.HotplugInterfaceStatus{},
				}},
				[]v1.VirtualMachineInstanceNetworkInterface{{
					Name:             "pepe",
					InterfaceName:    "eth1",
					HotplugInterface: &v1.HotplugInterfaceStatus{},
				}},
			),
		)
	})
})
