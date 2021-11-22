package network_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/network"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Dynamic Interface Attachment", func() {
	var (
		ctrl         *gomock.Controller
		kubeClient   *fakek8sclient.Clientset
		vmInterface  *kubecli.MockVirtualMachineInterface
		vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		kubeClient = fakek8sclient.NewSimpleClientset()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	const (
		ifaceName   = "pluggediface1"
		networkName = "newnet"
		vmName      = "myvm1"
	)

	mockVMIAddInterfaceEndpoints := func(vmName string, networkName string, ifaceName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().AddInterface(vmName, gomock.Any()).DoAndReturn(func(arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.AddInterfaceOptions).NetworkName).To(Equal(networkName))
			Expect(arg1.(*v1.AddInterfaceOptions).InterfaceName).To(Equal(ifaceName))
			return nil
		})
	}

	mockVMAddInterfaceEndpoints := func(vmName string, networkName string, ifaceName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().AddInterface(vmName, gomock.Any()).DoAndReturn(func(arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.AddInterfaceOptions).NetworkName).To(Equal(networkName))
			Expect(arg1.(*v1.AddInterfaceOptions).InterfaceName).To(Equal(ifaceName))
			return nil
		})
	}

	mockVMIRemoveInterfaceEndpoints := func(vmName string, networkName string, ifaceName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().RemoveInterface(vmName, gomock.Any()).DoAndReturn(func(arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.RemoveInterfaceOptions).NetworkName).To(Equal(networkName))
			Expect(arg1.(*v1.RemoveInterfaceOptions).InterfaceName).To(Equal(ifaceName))
			return nil
		})
	}

	mockVMRemoveInterfaceEndpoints := func(vmName string, networkName string, ifaceName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().RemoveInterface(vmName, gomock.Any()).DoAndReturn(func(arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.RemoveInterfaceOptions).NetworkName).To(Equal(networkName))
			Expect(arg1.(*v1.RemoveInterfaceOptions).InterfaceName).To(Equal(ifaceName))
			return nil
		})
	}

	DescribeTable("should fail when required input parameters are missing", func(cmdType string, args ...string) {
		cmd := clientcmd.NewVirtctlCommand(append([]string{cmdType}, args...)...)
		Expect(cmd.Execute()).To(HaveOccurred())
	},
		Entry("missing the VM name as parameter for the `AddInterface` cmd", network.HotplugCmdName),
		Entry("missing the VM name as parameter for the `RemoveInterface` cmd", network.HotUnplugCmdName),
		Entry("missing all required flags for the `AddInterface` cmd", network.HotplugCmdName, vmName),
		Entry("missing all required flags for the `RemoveInterface` cmd", network.HotUnplugCmdName, vmName),
		Entry("missing the network name flag for the `AddInterface` cmd", network.HotplugCmdName, vmName, "--iface-name", ifaceName),
		Entry("missing the network name flag for the `RemoveInterface` cmd", network.HotUnplugCmdName, vmName, "--iface-name", ifaceName),
		Entry("missing the interface name flag for the `AddInterface` cmd", network.HotplugCmdName, vmName, "--network-name", networkName),
		Entry("missing the interface name flag for the `RemoveInterface` cmd", network.HotUnplugCmdName, vmName, "--network-name", networkName),
	)

	When("all the required input parameters are provided", func() {
		var requiredCmdArgs []string

		BeforeEach(func() {
			kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, errors.New("kubeClient command not mocked")
			})

			requiredCmdArgs = []string{"--network-name", networkName, "--iface-name", ifaceName}
		})

		isPersistent := func(flags []string) bool {
			for _, flag := range flags {
				if flag == "--persist" {
					return true
				}
			}
			return false
		}

		setupMocks := func(cmdType string, additionalFlags ...string) {
			if isPersistent(additionalFlags) {
				vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
				if cmdType == network.HotplugCmdName {
					mockVMAddInterfaceEndpoints(vmName, networkName, ifaceName)
				}
				if cmdType == network.HotUnplugCmdName {
					mockVMRemoveInterfaceEndpoints(vmName, networkName, ifaceName)
				}
			} else {
				vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
				if cmdType == network.HotplugCmdName {
					mockVMIAddInterfaceEndpoints(vmName, networkName, ifaceName)
				}
				if cmdType == network.HotUnplugCmdName {
					mockVMIRemoveInterfaceEndpoints(vmName, networkName, ifaceName)
				}
			}
		}

		DescribeTable("works", func(cmdType string, additionalFlags ...string) {
			setupMocks(cmdType, additionalFlags...)

			cmdArgs := append(requiredCmdArgs, additionalFlags...)
			cmd := clientcmd.NewVirtctlCommand(buildDynamicIfaceCmd(cmdType, vmName, cmdArgs...)...)
			Expect(cmd.Execute()).To(Succeed())
		},
			Entry("hot-plug an interface", network.HotplugCmdName),
			Entry("persistently hot-plug an interface", network.HotplugCmdName, "--persist"),
			Entry("remove an interface", network.HotUnplugCmdName),
			Entry("persistently remove an interface", network.HotUnplugCmdName, "--persist"),
		)
	})
})

func buildDynamicIfaceCmd(cmdType string, vmName string, requiredCmdArgs ...string) []string {
	if cmdType == network.HotplugCmdName {
		return buildHotplugIfaceCmd(vmName, requiredCmdArgs...)
	}
	if cmdType == network.HotUnplugCmdName {
		return buildHotUnplugIfaceCmd(vmName, requiredCmdArgs...)
	}
	return nil
}

func buildHotplugIfaceCmd(vmName string, requiredCmdArgs ...string) []string {
	return append([]string{network.HotplugCmdName, vmName}, requiredCmdArgs...)
}

func buildHotUnplugIfaceCmd(vmName string, requiredCmdArgs ...string) []string {
	return append([]string{network.HotUnplugCmdName, vmName}, requiredCmdArgs...)
}
