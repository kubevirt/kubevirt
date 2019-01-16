package vm_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachine", func() {

	const unknownVM = "unknown-vm"
	const vmName = "testvm"
	var vmInterface *kubecli.MockVirtualMachineInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
	})

	Context("should patch VM", func() {
		It("with spec:running:true", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(2)
			vmInterface.EXPECT().Get(vm.Name, gomock.Any()).Return(vm, nil)
			runningVM := kubecli.NewMinimalVM("vnName")
			runningVM.Spec.Running = true
			vmInterface.EXPECT().Patch(vmName, gomock.Any(), gomock.Any()).Return(runningVM, nil)

			cmd := tests.NewVirtctlCommand("start", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("with spec:running:false", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = true

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(2)
			vmInterface.EXPECT().Get(vm.Name, gomock.Any()).Return(vm, nil)
			stoppedVM := kubecli.NewMinimalVM(vmName)
			stoppedVM.Spec.Running = false
			vmInterface.EXPECT().Patch(vmName, gomock.Any(), gomock.Any()).Return(stoppedVM, nil)

			cmd := tests.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("with spec:running:false when it's false already ", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = false

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface)
			vmInterface.EXPECT().Get(vm.Name, gomock.Any()).Return(vm, nil)

			cmd := tests.NewRepeatableVirtctlCommand("stop", vmName)
			Expect(cmd()).NotTo(BeNil())
		})

	})

	Context("with restart VM cmd", func() {
		It("should restart vm", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(2)
			vmInterface.EXPECT().Get(vm.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().Restart(vmName).Return(nil)

			cmd := tests.NewVirtctlCommand("restart", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("should return an error", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface)
			vmInterface.EXPECT().Get(vm.Name, gomock.Any()).Return(vm, errors.New(vmName))

			cmd := tests.NewVirtctlCommand("restart", vmName)
			Expect(cmd.Execute()).To(Equal(errors.New("Error fetching VirtualMachine: " + vmName)))
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
