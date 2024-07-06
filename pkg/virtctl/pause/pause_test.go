package pause_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/api"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/pause"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Pausing", func() {

	const vmName = "testvm"
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
		It("should fail a pause", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(pause.COMMAND_PAUSE)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
		It("should fail an unpause", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(pause.COMMAND_UNPAUSE)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	DescribeTable("should pause VMI", func(pauseOptions *v1.PauseOptions) {

		vmi := api.NewMinimalVMI(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vmi.Name, pauseOptions).Return(nil).Times(1)

		var command *cobra.Command
		if len(pauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_PAUSE, "vmi", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_PAUSE, "--dry-run", "vmi", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should unpause VMI", func(unpauseOptions *v1.UnpauseOptions) {

		vmi := api.NewMinimalVMI(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Unpause(context.Background(), vmi.Name, unpauseOptions).Return(nil).Times(1)

		var command *cobra.Command
		if len(unpauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_UNPAUSE, "vmi", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_UNPAUSE, "--dry-run", "vmi", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.UnpauseOptions{}),
		Entry("with dry-run option", &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should pause VM", func(pauseOptions *v1.PauseOptions) {

		vmi := api.NewMinimalVMI(vmName)
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vm.Name, pauseOptions).Return(nil).Times(1)

		var command *cobra.Command
		if len(pauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_PAUSE, "vm", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_PAUSE, "--dry-run", "vm", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should unpause VM", func(unpauseOptions *v1.UnpauseOptions) {

		vmi := api.NewMinimalVMI(vmName)
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Unpause(context.Background(), vm.Name, unpauseOptions).Return(nil).Times(1)
		var command *cobra.Command
		if len(unpauseOptions.DryRun) == 0 {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_UNPAUSE, "vm", vmName)
		} else {
			command = clientcmd.NewVirtctlCommand(pause.COMMAND_UNPAUSE, "--dry-run", "vm", vmName)
		}
		Expect(command.Execute()).To(Succeed())
	},
		Entry("", &v1.UnpauseOptions{}),
		Entry("with dry-run option", &v1.UnpauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)
})
