package unpause_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Unpausing", func() {

	const (
		COMMAND_UNPAUSE = "unpause"
		vmName          = "testvm"
	)

	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail an unpause", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(COMMAND_UNPAUSE)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	DescribeTable("should unpause VMI", func(unpauseOptions *v1.UnpauseOptions) {
		vmi := libvmi.New(
			libvmi.WithNamespace(k8smetav1.NamespaceDefault),
			libvmi.WithName(vmName),
		)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Unpause(context.Background(), vmi.Name, unpauseOptions).Return(nil).Times(1)

		var command *cobra.Command
		if len(unpauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(COMMAND_UNPAUSE, "vmi", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(COMMAND_UNPAUSE, "--dry-run", "vmi", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.UnpauseOptions{}),
		Entry("with dry-run option", &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should unpause VM", func(unpauseOptions *v1.UnpauseOptions) {
		vmi := libvmi.New(
			libvmi.WithNamespace(k8smetav1.NamespaceDefault),
			libvmi.WithName(vmName),
		)
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyHalted))

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Unpause(context.Background(), vm.Name, unpauseOptions).Return(nil).Times(1)
		var command *cobra.Command
		if len(unpauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(COMMAND_UNPAUSE, "vm", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(COMMAND_UNPAUSE, "--dry-run", "vm", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.UnpauseOptions{}),
		Entry("with dry-run option", &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)
})
