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
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var ctrl *gomock.Controller

	runStrategyAlways := v1.RunStrategyAlways
	runStrategyManual := v1.RunStrategyManual
	runStrategyHalted := v1.RunStrategyHalted

	startOpts := v1.StartOptions{Paused: false, DryRun: nil}
	stopOpts := v1.StopOptions{DryRun: nil}
	restartOpts := v1.RestartOptions{DryRun: nil}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
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
		It("should fail a restart", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("restart")
			Expect(cmd()).NotTo(Succeed())
		})
		It("should fail a migrate", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("migrate")
			Expect(cmd()).NotTo(Succeed())
		})
		It("should fail a migrate-cancel", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("migrate-cancel")
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

	Context("with migrate VM cmd", func() {
		DescribeTable("should migrate a vm according to options", func(migrateOptions *v1.MigrateOptions) {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Migrate(context.Background(), vm.Name, migrateOptions).Return(nil).Times(1)

			var cmd *cobra.Command
			if len(migrateOptions.DryRun) == 0 {
				cmd = clientcmd.NewVirtctlCommand("migrate", vmName)
			} else {
				cmd = clientcmd.NewVirtctlCommand("migrate", "--dry-run", vmName)
			}
			Expect(cmd.Execute()).To(Succeed())
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: []string{k8smetav1.DryRunAll}}),
		)
	})

	Context("with migrate-cancel VM cmd", func() {
		vm := kubecli.NewMinimalVM(vmName)
		migname := fmt.Sprintf("%s-%s", vm.Name, "migration") // "testvm-migration"
		mig := kubecli.NewMinimalMigration(migname)
		mig.Spec.VMIName = vm.Name // likely not required
		labelselector := fmt.Sprintf("%s==%s", v1.MigrationSelectorLabel, vm.Name)
		listoptions := k8smetav1.ListOptions{LabelSelector: labelselector}
		cmd := clientcmd.NewVirtctlCommand("migrate-cancel", vm.Name)
		It("should cancel the vm migration", func() {
			mig.Status.Phase = v1.MigrationRunning
			migList := v1.VirtualMachineInstanceMigrationList{
				Items: []v1.VirtualMachineInstanceMigration{
					*mig,
				},
			}

			kubecli.MockKubevirtClientInstance.EXPECT().
				VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).
				Return(migrationInterface).Times(2)

			migrationInterface.EXPECT().List(&listoptions).Return(&migList, nil).Times(1)
			migrationInterface.EXPECT().Delete(migname, &k8smetav1.DeleteOptions{}).Return(nil).Times(1)
			Expect(cmd.Execute()).To(Succeed())
		})
		It("Should fail if no active migration is found", func() {
			mig.Status.Phase = v1.MigrationSucceeded
			migList := v1.VirtualMachineInstanceMigrationList{
				Items: []v1.VirtualMachineInstanceMigration{
					*mig,
				},
			}

			kubecli.MockKubevirtClientInstance.EXPECT().
				VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).
				Return(migrationInterface).Times(1)

			migrationInterface.EXPECT().List(&listoptions).Return(&migList, nil).Times(1)

			res := cmd.Execute()
			Expect(res).To(HaveOccurred())
			errstr := fmt.Sprintf("Found no migration to cancel for %s", vm.Name)
			Expect(res.Error()).To(ContainSubstring(errstr))
		})
	})

	Context("with restart VM cmd", func() {
		It("should restart vm", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Restart(context.Background(), vm.Name, &restartOpts).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("restart", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		Context("using RunStrategy", func() {
			It("should restart a vm with runStrategy:manual", func() {
				vm := kubecli.NewMinimalVM(vmName)
				vm.Spec.RunStrategy = &runStrategyManual

				kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
				vmInterface.EXPECT().Restart(context.Background(), vm.Name, &restartOpts).Return(nil).Times(1)

				cmd := clientcmd.NewVirtctlCommand("restart", vmName)
				Expect(cmd.Execute()).To(Succeed())
			})
		})

		It("should force restart VM when --force=true and --grace-period is set", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			gracePeriod := int64(0)
			restartOptions := v1.RestartOptions{
				GracePeriodSeconds: &gracePeriod,
				DryRun:             nil,
			}
			vmInterface.EXPECT().ForceRestart(context.Background(), vm.Name, &restartOptions).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("restart", vmName, "--force", "--grace-period=0")
			Expect(cmd.Execute()).To(Succeed())
		})
		DescribeTable("should not force restart VM", func(force bool, gracePeriodSet bool) {
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

		It("should not restart VM", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(true)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Restart(context.Background(), vm.Name, &v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}).Return(nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("restart", vmName, "--dry-run")
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

			vmiInterface.EXPECT().GuestOsInfo(context.Background(), vm.Name).Return(guestOSInfo, nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("guestosinfo", vm.Name)
			Expect(cmd.Execute()).To(Succeed())
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

			vmiInterface.EXPECT().FilesystemList(context.Background(), vm.Name).Return(fsList, nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("fslist", vm.Name)
			Expect(cmd.Execute()).To(Succeed())
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

			vmiInterface.EXPECT().UserList(context.Background(), vm.Name).Return(userList, nil).Times(1)

			cmd := clientcmd.NewVirtctlCommand("userlist", vm.Name)
			Expect(cmd.Execute()).To(Succeed())
		})
	})

	Context("remove volume", func() {
		expectVMIEndpointRemoveVolumeError := func(vmiName, volumeName string) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)
			vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
				Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
				return fmt.Errorf("error removing")
			})
		}

		DescribeTable("removevolume should report error if call returns error according to option", func(isDryRun bool) {
			expectVMIEndpointRemoveVolumeError("testvmi", "testvolume")
			commandAndArgs := []string{"removevolume", "testvmi", "--volume-name=testvolume"}
			if isDryRun {
				commandAndArgs = append(commandAndArgs, "--dry-run")
			}
			cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			res := cmdAdd()
			Expect(res).To(HaveOccurred())
			Expect(res.Error()).To(ContainSubstring("error removing"))
		},
			Entry("with default", false),
			Entry("with dry-run arg", true),
		)
	})

})
