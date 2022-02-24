package status

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Status", func() {
	var ctrl *gomock.Controller
	var kvInterface *kubecli.MockKubeVirtInterface
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmirsInterface *kubecli.MockReplicaSetInterface
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmirsInterface = kubecli.NewMockReplicaSetInterface(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		virtClient.EXPECT().KubeVirt(gomock.Any()).Return(kvInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(gomock.Any()).Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().ReplicaSet(gomock.Any()).Return(vmirsInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
	})
	Context("starting with the assumption of /status being present", func() {

		Context("for PUT operations", func() {
			It("should continuously use the /status subresource if no errors occur", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(2)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				kvInterface.EXPECT().Update(kv).Return(kv, nil).Times(2)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update but keep in mind that objects may have disappeared", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				kvInterface.EXPECT().Update(kv).Return(kv, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update but keep in mind that the subresource may get enabled directly afterwards", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				newKV := kv.DeepCopy()
				newKV.Status.Phase = v1.KubeVirtPhaseDeleted
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				kvInterface.EXPECT().Update(kv).Return(newKV, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})

			It("should stick with /status if an arbitrary error occurs", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, fmt.Errorf("I am not 404")).Times(1)
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})
		})

		Context("for PATCH operations", func() {
			It("should continuously use the /status subresource if no errors occur", func() {
				updater := NewVMStatusUpdater(virtClient)
				patchOptions := &v12.PatchOptions{}
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, nil).Times(2)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update", func() {
				updater := NewVMStatusUpdater(virtClient)
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				newVM := vm.DeepCopy()
				newVM.SetResourceVersion("2")
				patchOptions := &v12.PatchOptions{}
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(newVM, nil).Times(2)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update but keep in mind that objects may have disappeared", func() {
				updater := NewVMStatusUpdater(virtClient)
				patchOptions := &v12.PatchOptions{}
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(nil, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(nil, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})

			It("should fall back on a 404 error on the /status subresource to an ordinary update but keep in mind that the subresource may get enabled directly afterwards", func() {
				updater := NewVMStatusUpdater(virtClient)
				patchOptions := &v12.PatchOptions{}
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(nil, errors.NewNotFound(schema.GroupResource{}, "something")).Times(1)
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})

			It("should stick with /status if an arbitrary error occurs", func() {
				updater := NewVMStatusUpdater(virtClient)
				patchOptions := &v12.PatchOptions{}
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, fmt.Errorf("I am not a 404 error")).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(vm, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})
		})

	})

	Context("starting with the assumption that /status is not present", func() {

		Context("for PUT operations", func() {
			It("should stick with a normal update if the  status did change", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				updater.updater.subresource = false
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().Update(kv).Return(kv, nil).Times(2)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})

			It("should stick with a normal update if we get a 404 error", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				updater.updater.subresource = false
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().Update(kv).Return(kv, errors.NewNotFound(schema.GroupResource{}, "something")).Times(2)
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
			})

			It("should stick with a normal update if we get an arbitrary error", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				updater.updater.subresource = false
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				kvInterface.EXPECT().Update(kv).Return(kv, fmt.Errorf("I am not 404")).Times(2)
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
				Expect(updater.UpdateStatus(kv)).ToNot(Succeed())
			})

			It("should fall back to /status if the status did not change and stick to it", func() {
				updater := NewKubeVirtStatusUpdater(virtClient)
				updater.updater.subresource = false
				kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
				oldKV := kv.DeepCopy()
				oldKV.Status.Phase = v1.KubeVirtPhaseDeploying
				kvInterface.EXPECT().Update(kv).Return(oldKV, nil).Times(1)
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
				kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(1)
				Expect(updater.UpdateStatus(kv)).To(Succeed())
			})
		})
		Context("for PATCH operations", func() {
			It("should stick with a normal update if the resource version did change", func() {
				updater := NewVMStatusUpdater(virtClient)
				updater.updater.subresource = false
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				newVM := vm.DeepCopy()
				newVM.SetResourceVersion("2")
				patchOptions := &v12.PatchOptions{}
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(newVM, nil).Times(2)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})

			It("should stick with a normal update if we get a 404 error", func() {
				updater := NewVMStatusUpdater(virtClient)
				updater.updater.subresource = false
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				newVM := vm.DeepCopy()
				newVM.SetResourceVersion("2")
				patchOptions := &v12.PatchOptions{}
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(nil, errors.NewNotFound(schema.GroupResource{}, "something")).Times(2)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
			})

			It("should stick with a normal update if we get an arbitrary error", func() {
				updater := NewVMStatusUpdater(virtClient)
				updater.updater.subresource = false
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				newVM := vm.DeepCopy()
				newVM.SetResourceVersion("2")
				patchOptions := &v12.PatchOptions{}
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(nil, fmt.Errorf("I am an arbitrary error")).Times(2)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).ToNot(Succeed())
			})

			It("should fall back to /status if the status did not change and stick to it", func() {
				updater := NewVMStatusUpdater(virtClient)
				updater.updater.subresource = false
				vm := &v1.VirtualMachine{ObjectMeta: v12.ObjectMeta{Name: "test", ResourceVersion: "1"}, Status: v1.VirtualMachineStatus{Ready: true}}
				newVM := vm.DeepCopy()
				patchOptions := &v12.PatchOptions{}
				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(newVM, nil).Times(1)
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(newVM, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
				vmInterface.EXPECT().PatchStatus(vm.Name, types.JSONPatchType, []byte("test"), patchOptions).Return(newVM, nil).Times(1)
				Expect(updater.PatchStatus(vm, types.JSONPatchType, []byte("test"), patchOptions)).To(Succeed())
			})
		})
	})

	Context("the generic updater", func() {
		It("should work for /status based updates for all types needed", func() {

			By("checking the KubeVirt resource")
			kvUpdater := NewKubeVirtStatusUpdater(virtClient)
			kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
			kvInterface.EXPECT().UpdateStatus(kv).Return(kv, nil).Times(2)
			Expect(kvUpdater.UpdateStatus(kv)).To(Succeed())
			Expect(kvUpdater.UpdateStatus(kv)).To(Succeed())

			By("checking the VirtualMachine resource")
			vmUpdater := NewVMStatusUpdater(virtClient)
			vm := &v1.VirtualMachine{Status: v1.VirtualMachineStatus{Ready: true}}
			vmInterface.EXPECT().UpdateStatus(vm).Return(vm, nil).Times(1)
			Expect(vmUpdater.UpdateStatus(vm)).To(Succeed())

			By("checking the VirtualMachineInstanceReplicaSet resource")
			vmirsUpdater := NewVMIRSStatusUpdater(virtClient)
			vmirs := &v1.VirtualMachineInstanceReplicaSet{Status: v1.VirtualMachineInstanceReplicaSetStatus{Replicas: 2}}
			vmirsInterface.EXPECT().UpdateStatus(vmirs).Return(vmirs, nil).Times(1)
			Expect(vmirsUpdater.UpdateStatus(vmirs)).To(Succeed())

			By("checking the VirtualMachineInstanceMigration resource")
			migrationUpdater := NewMigrationStatusUpdater(virtClient)
			migration := &v1.VirtualMachineInstanceMigration{Status: v1.VirtualMachineInstanceMigrationStatus{Phase: v1.MigrationPhaseUnset}}
			migrationInterface.EXPECT().UpdateStatus(migration).Return(migration, nil).Times(1)
			Expect(migrationUpdater.UpdateStatus(migration)).To(Succeed())
		})

		It("should work for resource where /status is not enabled", func() {

			By("checking the KubeVirt resource")
			kvUpdater := NewKubeVirtStatusUpdater(virtClient)
			kvUpdater.updater.subresource = false
			kv := &v1.KubeVirt{Status: v1.KubeVirtStatus{Phase: v1.KubeVirtPhaseDeployed}}
			kvInterface.EXPECT().Update(kv).Return(kv, nil).Times(2)
			Expect(kvUpdater.UpdateStatus(kv)).To(Succeed())
			Expect(kvUpdater.UpdateStatus(kv)).To(Succeed())

			By("checking the VirtualMachine resource")
			vmUpdater := NewVMStatusUpdater(virtClient)
			vmUpdater.updater.subresource = false
			vm := &v1.VirtualMachine{Status: v1.VirtualMachineStatus{Ready: true}}
			vmInterface.EXPECT().Update(vm).Return(vm, nil).Times(1)
			Expect(vmUpdater.UpdateStatus(vm)).To(Succeed())

			By("checking the VirtualMachineInstanceReplicaSet resource")
			vmirsUpdater := NewVMIRSStatusUpdater(virtClient)
			vmirsUpdater.updater.subresource = false
			vmirs := &v1.VirtualMachineInstanceReplicaSet{Status: v1.VirtualMachineInstanceReplicaSetStatus{Replicas: 2}}
			vmirsInterface.EXPECT().Update(vmirs).Return(vmirs, nil).Times(1)
			Expect(vmirsUpdater.UpdateStatus(vmirs)).To(Succeed())

			By("checking the VirtualMachineInstanceMigration resource")
			migrationUpdater := NewMigrationStatusUpdater(virtClient)
			migrationUpdater.updater.subresource = false
			migration := &v1.VirtualMachineInstanceMigration{Status: v1.VirtualMachineInstanceMigrationStatus{Phase: v1.MigrationPhaseUnset}}
			migrationInterface.EXPECT().Update(migration).Return(migration, nil).Times(1)
			Expect(migrationUpdater.UpdateStatus(migration)).To(Succeed())
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
