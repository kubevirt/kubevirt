package pause_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
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
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("pause")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	It("should pause VMI", func() {
		vmi := v1.NewMinimalVMI(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Pause(vmi.Name).Return(nil).Times(1)

		cmd := tests.NewVirtctlCommand("pause", "vmi", vmName)
		Expect(cmd.Execute()).To(BeNil())
	})

	It("should unpause VMI", func() {
		vmi := v1.NewMinimalVMI(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Unpause(vmi.Name).Return(nil).Times(1)

		cmd := tests.NewVirtctlCommand("unpause", "vmi", vmName)
		Expect(cmd.Execute()).To(BeNil())
	})

	It("should pause VM", func() {
		vmi := v1.NewMinimalVMI(vmName)
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Pause(vm.Name).Return(nil).Times(1)

		cmd := tests.NewVirtctlCommand("pause", "vm", vmName)
		Expect(cmd.Execute()).To(BeNil())
	})

	It("should unpause VM", func() {
		vmi := v1.NewMinimalVMI(vmName)
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Unpause(vm.Name).Return(nil).Times(1)

		cmd := tests.NewVirtctlCommand("unpause", "vm", vmName)
		Expect(cmd.Execute()).To(BeNil())
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
