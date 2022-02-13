package softreboot_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/softreboot"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Soft rebooting", func() {

	const vmiName = "testvmi"
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
			cmd := tests.NewRepeatableVirtctlCommand(softreboot.COMMAND_SOFT_REBOOT)
			err := cmd()
			Expect(err).NotTo(BeNil())
		})
	})

	It("should soft reboot VMI", func() {
		vmi := api.NewMinimalVMI(vmiName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().SoftReboot(vmi.Name).Return(nil).Times(1)

		cmd := tests.NewVirtctlCommand(softreboot.COMMAND_SOFT_REBOOT, vmiName)
		Expect(cmd.Execute()).To(BeNil())
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
