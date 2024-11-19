package reset_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/reset"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Resetting", func() {
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(reset.COMMAND_RESET)
			Expect(cmd()).To(MatchError(ContainSubstring("received 0")))
		})
	})

	It("should reset VMI", func() {
		vmiName := "resetable-vmi"

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Reset(context.Background(), vmiName).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand(reset.COMMAND_RESET, vmiName)
		Expect(cmd()).To(Succeed())
	})
})
