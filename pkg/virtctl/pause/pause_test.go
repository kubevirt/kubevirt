package pause_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Pausing", func() {

	const (
		COMMAND_PAUSE = "pause"
		vmName        = "testvm"
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
		It("should fail a pause", func() {
			cmd := testing.NewRepeatableVirtctlCommand(COMMAND_PAUSE)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	DescribeTable("should pause VMI", func(pauseOptions *v1.PauseOptions) {
		vmi := libvmi.New(libvmi.WithName(vmName))

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vmi.Name, pauseOptions).Return(nil).Times(1)

		args := []string{COMMAND_PAUSE, "vmi", vmName}
		if len(pauseOptions.DryRun) > 0 {
			args = append(args, "--dry-run")
		}
		Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should pause VM", func(pauseOptions *v1.PauseOptions) {
		vmi := libvmi.New(libvmi.WithName(vmName))
		vm := libvmi.NewVirtualMachine(vmi)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vm.Name, pauseOptions).Return(nil).Times(1)

		args := []string{COMMAND_PAUSE, "vm", vmName}
		if len(pauseOptions.DryRun) > 0 {
			args = append(args, "--dry-run")
		}
		Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)
})
