package vm_test

import (
	"context"
	"fmt"
	"os"

	"k8s.io/utils/pointer"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/clientcmd"

	"k8s.io/client-go/kubernetes/fake"

	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
)

const (
	vmName           = "testvm"
	invalidFormat    = "test-format"
	namespace        = "--namespace"
	nonExistingVM    = "non-existing-vm"
	testingNamespace = "testing-namespace"
	outputFormat     = "--output"
	fileInput        = "--file"
)

var _ = Describe("VirtualMachine", func() {

	var vm *v1.VirtualMachine
	var file *os.File
	var err error
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var expandSpecInterface *kubecli.MockExpandSpecInterface
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
		expandSpecInterface = kubecli.NewMockExpandSpecInterface(ctrl)
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
		It("should fail a expand", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("expand")
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

		It("should force restart vm", func() {
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

		It("should force delete vm", func() {
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

	Context("hotplug volume", func() {
		var (
			cdiClient  *cdifake.Clientset
			coreClient *fake.Clientset
		)
		createTestDataVolume := func() *v1beta1.DataVolume {
			return &v1beta1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "testvolume",
				},
			}
		}

		createTestPVC := func() *k8sv1.PersistentVolumeClaim {
			return &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "testvolume",
				},
			}
		}

		verifyVolumeSource := func(volumeOptions interface{}, useDv bool) {
			if useDv {
				Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.DataVolume).ToNot(BeNil())
				Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.PersistentVolumeClaim).To(BeNil())
			} else {
				Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.DataVolume).To(BeNil())
				Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			}
		}

		expectVMIEndpointAddVolume := func(vmiName, volumeName string, useDv bool) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)
			vmiInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
				Expect(arg1.(*v1.AddVolumeOptions).Name).To(Equal(volumeName))
				verifyVolumeSource(arg1, useDv)
				return nil
			})
		}

		expectVMIEndpointRemoveVolume := func(vmiName, volumeName string, useDv bool) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(k8smetav1.NamespaceDefault).
				Return(vmiInterface).
				Times(1)
			vmiInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
				Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
				return nil
			})
		}

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

		expectVMEndpointAddVolume := func(vmiName, volumeName string, useDv bool) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(k8smetav1.NamespaceDefault).
				Return(vmInterface).
				Times(1)
			vmInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
				Expect(arg1.(*v1.AddVolumeOptions).Name).To(Equal(volumeName))
				verifyVolumeSource(arg1, useDv)
				return nil
			})
		}

		expectVMEndpointRemoveVolume := func(vmiName, volumeName string, useDv bool) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(k8smetav1.NamespaceDefault).
				Return(vmInterface).
				Times(1)
			vmInterface.EXPECT().RemoveVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
				Expect(arg1.(*v1.RemoveVolumeOptions).Name).To(Equal(volumeName))
				return nil
			})
		}

		BeforeEach(func() {
			cdiClient = cdifake.NewSimpleClientset()
			coreClient = fake.NewSimpleClientset()
		})

		DescribeTable("should fail with missing required or invalid parameters", func(commandName, errorString string, args ...string) {
			commandAndArgs := []string{commandName}
			commandAndArgs = append(commandAndArgs, args...)
			cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			res := cmdAdd()
			Expect(res).To(HaveOccurred())
			Expect(res.Error()).To(ContainSubstring(errorString))
		},
			Entry("addvolume no args", "addvolume", "argument validation failed"),
			Entry("addvolume name, missing required volume-name", "addvolume", "required flag(s)", "testvmi"),
			Entry("addvolume name, invalid extra parameter", "addvolume", "unknown flag", "testvmi", "--volume-name=blah", "--invalid=test"),
			Entry("removevolume no args", "removevolume", "argument validation failed"),
			Entry("removevolume name, missing required volume-name", "removevolume", "required flag(s)", "testvmi"),
			Entry("removevolume name, invalid extra parameter", "removevolume", "unknown flag", "testvmi", "--volume-name=blah", "--invalid=test"),
		)
		DescribeTable("should fail addvolume when no source is found according to option", func(isDryRun bool) {
			kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
			kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
			commandAndArgs := []string{"addvolume", "testvmi", "--volume-name=testvolume"}
			if isDryRun {
				commandAndArgs = append(commandAndArgs, "--dry-run")
			}
			cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			res := cmdAdd()
			Expect(res).To(HaveOccurred())
			Expect(res.Error()).To(ContainSubstring("Volume testvolume is not a DataVolume or PersistentVolumeClaim"))
		},
			Entry("with default", false),
			Entry("with dry-run arg", true),
		)

		DescribeTable("should call correct endpoint", func(commandName, vmiName, volumeName string, useDv bool, expectFunc func(vmiName, volumeName string, useDv bool), args ...string) {
			if commandName == "addvolume" {
				kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
				if useDv {
					cdiClient.CdiV1beta1().DataVolumes(k8smetav1.NamespaceDefault).Create(context.Background(), createTestDataVolume(), k8smetav1.CreateOptions{})
				} else {
					kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
					coreClient.CoreV1().PersistentVolumeClaims(k8smetav1.NamespaceDefault).Create(context.Background(), createTestPVC(), k8smetav1.CreateOptions{})
				}
			}
			expectFunc(vmiName, volumeName, useDv)
			commandAndArgs := []string{commandName, vmiName, fmt.Sprintf("--volume-name=%s", volumeName)}
			commandAndArgs = append(commandAndArgs, args...)
			cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
			Expect(cmd.Execute()).To(Succeed())
		},
			Entry("addvolume dv, no persist should call VMI endpoint", "addvolume", "testvmi", "testvolume", true, expectVMIEndpointAddVolume),
			Entry("addvolume pvc, no persist should call VMI endpoint", "addvolume", "testvmi", "testvolume", false, expectVMIEndpointAddVolume),
			Entry("addvolume dv, with persist should call VM endpoint", "addvolume", "testvmi", "testvolume", true, expectVMEndpointAddVolume, "--persist"),
			Entry("addvolume pvc, with persist should call VM endpoint", "addvolume", "testvmi", "testvolume", false, expectVMEndpointAddVolume, "--persist"),
			Entry("removevolume dv, no persist should call VMI endpoint", "removevolume", "testvmi", "testvolume", true, expectVMIEndpointRemoveVolume),
			Entry("removevolume pvc, no persist should call VMI endpoint", "removevolume", "testvmi", "testvolume", false, expectVMIEndpointRemoveVolume),
			Entry("removevolume dv, with persist should call VM endpoint", "removevolume", "testvmi", "testvolume", true, expectVMEndpointRemoveVolume, "--persist"),
			Entry("removevolume pvc, with persist should call VM endpoint", "removevolume", "testvmi", "testvolume", false, expectVMEndpointRemoveVolume, "--persist"),

			Entry("addvolume dv, no persist with dry-run should call VMI endpoint", "addvolume", "testvmi", "testvolume", true, expectVMIEndpointAddVolume, "--dry-run"),
			Entry("addvolume pvc, no persist with dry-run should call VMI endpoint", "addvolume", "testvmi", "testvolume", false, expectVMIEndpointAddVolume, "--dry-run"),
			Entry("addvolume dv, with persist with dry-run should call VM endpoint", "addvolume", "testvmi", "testvolume", true, expectVMEndpointAddVolume, "--persist", "--dry-run"),
			Entry("addvolume pvc, with persist with dry-run should call VM endpoint", "addvolume", "testvmi", "testvolume", false, expectVMEndpointAddVolume, "--persist", "--dry-run"),
			Entry("removevolume dv, no persist with dry-run should call VMI endpoint", "removevolume", "testvmi", "testvolume", true, expectVMIEndpointRemoveVolume, "--dry-run"),
			Entry("removevolume pvc, no persist with dry-run should call VMI endpoint", "removevolume", "testvmi", "testvolume", false, expectVMIEndpointRemoveVolume, "--dry-run"),
			Entry("removevolume dv, with persist with dry-run should call VM endpoint", "removevolume", "testvmi", "testvolume", true, expectVMEndpointRemoveVolume, "--persist", "--dry-run"),
			Entry("removevolume pvc, with persist with dry-run should call VM endpoint", "removevolume", "testvmi", "testvolume", false, expectVMEndpointRemoveVolume, "--persist", "--dry-run"),
		)

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

	Context("Expand command", func() {
		BeforeEach(func() {
			vm = kubecli.NewMinimalVM(vmName)
		})

		It("should succeed when called on vm", func() {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), vmName)

			cmd := clientcmd.NewVirtctlCommand("expand", "--vm", vmName)
			Expect(cmd.Execute()).To(Succeed())
		})

		DescribeTable("should succeed when called with ", func(formatName string) {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), vmName)

			cmd := clientcmd.NewVirtctlCommand("expand", outputFormat, formatName, "--vm", vmName)
			Expect(cmd.Execute()).To(Succeed())
		},
			Entry("supported format json", "json"),
			Entry("supported format yaml", "yaml"),
		)

		It("should fail when called on non existing vm", func() {
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().GetWithExpandedSpec(context.Background(), nonExistingVM).Return(nil, fmt.Errorf("\"%s\" not found", nonExistingVM))

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", "--vm", nonExistingVM)
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error expanding VirtualMachine - non-existing-vm in namespace - default: \"non-existing-vm\" not found"))
		})

		It("should fail when called with non supported output format", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", outputFormat, invalidFormat, "--vm", vmName)
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error not supported output format defined: test-format"))
		})
	})

	Context("Expand command with file input", func() {
		const (
			vmSpec = `apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: testvm
spec:
  runStrategy: Always
  template:
    spec:
      domain:
        devices: {}
        machine:
          type: q35
        resources: {}
        volumes:
status:
`
			invalidYaml = `apiVersion: kubevirt.io/v1kind: VirtualMachine`
		)
		BeforeEach(func() {
			file, err = os.CreateTemp("", "file-*")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succeed when called with existing file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(vmSpec), 0777)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(vmSpec), vm)).To(Succeed())

			kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
			expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(vm, nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail when called with invalid vm defined in file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(vmSpec), 0777)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(vmSpec), vm)).To(Succeed())

			kubecli.MockKubevirtClientInstance.EXPECT().ExpandSpec(k8smetav1.NamespaceDefault).Return(expandSpecInterface).Times(1)
			expandSpecInterface.EXPECT().ForVirtualMachine(vm).Return(nil, fmt.Errorf("error expanding vm from file")).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error expanding VirtualMachine - testvm in namespace - default: error expanding vm from file"))
		})

		It("should fail when called with invalid yaml in file input", func() {
			Expect(os.WriteFile(file.Name(), []byte(invalidYaml), 0777)).To(Succeed())
			Expect(yaml.Unmarshal([]byte(invalidYaml), vm)).ToNot(Succeed())

			cmd := clientcmd.NewRepeatableVirtctlCommand("expand", fileInput, file.Name())
			err := cmd()

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError("error decoding VirtualMachine error converting YAML to JSON: yaml: mapping values are not allowed in this context"))
		})
	})
})
