package reset_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/clientcmd"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virtctl/reset"
)

var _ = Describe("Resetting", func() {
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
			cmd := clientcmd.NewRepeatableVirtctlCommand(reset.COMMAND_RESET)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	It("should reset VMI", func() {
		vmi := libvmi.New()

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Reset(context.Background(), vmi.Name).Return(nil).Times(1)

		cmd := clientcmd.NewVirtctlCommand(reset.COMMAND_RESET, vmi.Name)
		Expect(cmd.Execute()).To(Succeed())
	})
})
