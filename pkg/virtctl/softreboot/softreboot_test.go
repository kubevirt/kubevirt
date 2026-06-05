package softreboot_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virtctl/softreboot"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Soft rebooting", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := testing.NewRepeatableVirtctlCommand(softreboot.COMMAND_SOFT_REBOOT)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	It("should soft reboot VMI", func() {
		vmi := libvmi.New()

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().SoftReboot(context.Background(), vmi.Name).Return(nil).Times(1)

		cmd := testing.NewRepeatableVirtctlCommand(softreboot.COMMAND_SOFT_REBOOT, vmi.Name)
		Expect(cmd()).To(Succeed())
	})
})
