package vm_test

import (
	"context"
	"fmt"

	"k8s.io/utils/pointer"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("VirtualMachine", func() {

	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	runStrategyAlways := v1.RunStrategyAlways
	runStrategyHalted := v1.RunStrategyHalted

	startOpts := v1.StartOptions{Paused: false, DryRun: nil}
	stopOpts := v1.StopOptions{DryRun: nil}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail a start", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("start")
			Expect(cmd()).NotTo(Succeed())
		})
		It("should fail a stop", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("stop")
			Expect(cmd()).NotTo(Succeed())
		})
	})

	Context("should patch VM", func() {
		It("with spec:running:true", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(false)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &startOpts).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("start", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("with spec:running:false", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(true)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(context.Background(), vm.Name, &stopOpts).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("with spec:running:false when it's false already ", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(false)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(context.Background(), vm.Name, &stopOpts).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		Context("Using RunStrategy", func() {
			It("with spec:runStrategy:running", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyHalted

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Start(context.Background(), vm.Name, &startOpts).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("start", vmName)
				Expect(cmd.Execute()).To(Succeed())
			})

			It("with spec:runStrategy:halted", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyAlways

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Stop(context.Background(), vm.Name, &stopOpts).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("stop", vmName)
				Expect(cmd.Execute()).To(Succeed())
			})

			It("with spec:runStrategy:halted when it's false already ", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyHalted

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Stop(context.Background(), vm.Name, &stopOpts).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("stop", vmName)
				Expect(cmd.Execute()).To(Succeed())
			})
		})
		Context("With --paused flag", func() {
			It("should start paused if --paused true", func() {
				vm := kubecli.NewMinimalVM(vmName)

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{Paused: true, DryRun: nil}).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("start", vmName, "--paused")
				Expect(cmd.Execute()).To(Succeed())
			})
			It("should start if --paused false", func() {
				vm := kubecli.NewMinimalVM(vmName)

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Start(context.Background(), vm.Name, &startOpts).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("start", vmName, "--paused=false")
				Expect(cmd.Execute()).To(Succeed())
			})
		})

	})

	Context("with stop VM cmd", func() {
		It("should stop VM", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(context.Background(), vm.Name, &stopOpts).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("stop", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("should force stop VM when --force=true and --grace-period is set", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			gracePeriod := int64(0)
			stopOptions := v1.StopOptions{
				GracePeriod: &gracePeriod,
				DryRun:      nil,
			}
			vmInterface.EXPECT().ForceStop(context.Background(), vm.Name, &stopOptions).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("stop", vmName, "--force", "--grace-period=0")
			Expect(cmd.Execute()).To(Succeed())
		})

		DescribeTable("should not force stop VM", func(force bool, gracePeriodSet bool) {
			var cmd *cobra.Command

			if force {
				cmd = clientcmd.NewVirtctlCommand("stop", vmName, "--force")
			} else {
				cmd = clientcmd.NewVirtctlCommand("stop", vmName, "--grace-period=0")
			}

			err := fmt.Errorf("Must both use --force=true and set --grace-period.")
			Expect(cmd.Execute()).To(MatchError(err))
		},
			Entry("when --force=true and --grace-period is not set", true, false),
			Entry("when --force=false and --grace-period is set", false, true),
		)
	})

	Context("with dry-run parameter", func() {

		It("should not start VM", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(false)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{DryRun: []string{k8smetav1.DryRunAll}}).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("start", vmName, "--dry-run")
			Expect(cmd.Execute()).To(Succeed())
		})

		It("should not stop VM", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(true)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Stop(context.Background(), vm.Name, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("stop", vmName, "--dry-run")
			Expect(cmd.Execute()).To(Succeed())
		})

	})

	Context("guest agent", func() {

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

			vmiInterface.EXPECT().UserList(context.Background(), vm.Name).Return(userList, nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("userlist", vm.Name)
			Expect(cmd.Execute()).To(Succeed())
		})
	})

})
