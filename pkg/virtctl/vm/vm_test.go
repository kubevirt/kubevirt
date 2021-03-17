package vm_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachine", func() {

	const vmName = "testvm"
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	running := true
	notRunning := false
	runStrategyAlways := v1.RunStrategyAlways
	runStrategyManual := v1.RunStrategyManual
	runStrategyHalted := v1.RunStrategyHalted

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("start")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("stop")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("restart")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("migrate")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	Context("With missing input parameters", func() {
		It("should fail", func() {
			cmd := tests.NewRepeatableVirtctlCommand("rename")
			Expect(cmd()).NotTo(BeNil())
		})
	})

	Context("should patch VM", func() {
		It("with spec:running:true", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = &notRunning

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(vm.Name).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("start", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("with spec:running:false", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = &running

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(vm.Name).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("with spec:running:false when it's false already ", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = &notRunning

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(vm.Name).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		Context("Using RunStrategy", func() {
			It("with spec:runStrategy:running", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyHalted

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Start(vm.Name).Return(nil).Times(1)

				cmd := tests.NewVirtctlCommand("start", vmName)
				Expect(cmd.Execute()).To(BeNil())
			})

			It("with spec:runStrategy:halted", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyAlways

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Stop(vm.Name).Return(nil).Times(1)

				cmd := tests.NewVirtctlCommand("stop", vmName)
				Expect(cmd.Execute()).To(BeNil())
			})

			It("with spec:runStrategy:halted when it's false already ", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyHalted

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Stop(vm.Name).Return(nil).Times(1)

				cmd := tests.NewVirtctlCommand("stop", vmName)
				Expect(cmd.Execute()).To(BeNil())
			})
		})

	})

	Context("with migrate VM cmd", func() {
		It("should migrate vm", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Migrate(vm.Name).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("migrate", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})
	})

	Context("with restart VM cmd", func() {
		It("should restart vm", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Restart(vm.Name).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("restart", vmName)
			Expect(cmd.Execute()).To(BeNil())
		})

		Context("using RunStrategy", func() {
			It("should restart a vm with runStrategy:manual", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyManual

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Restart(vm.Name).Return(nil).Times(1)

				cmd := tests.NewVirtctlCommand("restart", vmName)
				Expect(cmd.Execute()).To(BeNil())
			})
		})

		It("should force restart vm", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().ForceRestart(vm.Name, 0).Return(nil).Times(1)

			cmd := tests.NewVirtctlCommand("restart", vmName, "--force", "--grace-period=0")
			Expect(cmd.Execute()).To(BeNil())
		})
	})

	Context("rename", func() {
		// All validations are done server-side
		It("should initiate rename api call", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(k8smetav1.NamespaceDefault).
				Return(vmInterface).
				Times(1)

			vmInterface.
				EXPECT().
				Rename(vm.Name, &v1.RenameOptions{NewName: vm.Name + "new"}).
				Return(nil).
				Times(1)

			cmd := tests.NewVirtctlCommand("rename", vm.Name, vm.Name+"new")
			Expect(cmd.Execute()).To(BeNil())
		})
	})

	Context("guest agent", func() {

		It("should return guest agent data", func() {
			vm := kubecli.NewMinimalVM(vmName)
			guestOSInfo := v1.VirtualMachineInstanceGuestAgentInfo{
				GAVersion: "3.1.0",
			}

			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)

			vmiInterface.EXPECT().GuestOsInfo(vm.Name).Return(guestOSInfo, nil).Times(1)

			cmd := tests.NewVirtctlCommand("guestosinfo", vm.Name)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("should return filesystem  data", func() {
			vm := kubecli.NewMinimalVM(vmName)
			fsList := v1.VirtualMachineInstanceFileSystemList{
				Items: []v1.VirtualMachineInstanceFileSystem{
					{
						DiskName: "TEST",
					},
				},
			}

			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)

			vmiInterface.EXPECT().FilesystemList(vm.Name).Return(fsList, nil).Times(1)

			cmd := tests.NewVirtctlCommand("fslist", vm.Name)
			Expect(cmd.Execute()).To(BeNil())
		})

		It("should return userlist  data", func() {
			vm := kubecli.NewMinimalVM(vmName)
			userList := v1.VirtualMachineInstanceGuestOSUserList{
				Items: []v1.VirtualMachineInstanceGuestOSUser{
					{
						UserName: "TEST",
					},
				},
			}

			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)

			vmiInterface.EXPECT().UserList(vm.Name).Return(userList, nil).Times(1)

			cmd := tests.NewVirtctlCommand("userlist", vm.Name)
			Expect(cmd.Execute()).To(BeNil())
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
