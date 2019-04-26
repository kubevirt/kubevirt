package watch

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdifake "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned/fake"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VirtualMachine", func() {

	Context("One valid VirtualMachine controller given", func() {

		var ctrl *gomock.Controller
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		var vmInterface *kubecli.MockVirtualMachineInterface
		var vmiSource *framework.FakeControllerSource
		var vmSource *framework.FakeControllerSource
		var vmiInformer cache.SharedIndexInformer
		var vmInformer cache.SharedIndexInformer
		var dataVolumeInformer cache.SharedIndexInformer
		var dataVolumeSource *framework.FakeControllerSource
		var stop chan struct{}
		var controller *VMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmiFeeder *testutils.VirtualMachineFeeder
		var dataVolumeFeeder *testutils.DataVolumeFeeder
		var cdiClient *cdifake.Clientset

		syncCaches := func(stop chan struct{}) {
			go vmiInformer.Run(stop)
			go vmInformer.Run(stop)
			go dataVolumeInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, vmInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

			dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMController(vmiInformer, vmInformer, dataVolumeInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
			dataVolumeFeeder = testutils.NewDataVolumeFeeder(mockQueue, dataVolumeSource)

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

			cdiClient = cdifake.NewSimpleClientset()
			virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			cdiClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

		})

		shouldExpectDataVolumeCreation := func(uid types.UID, idx *int) {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				*idx++
				dataVolume := update.GetObject().(*cdiv1.DataVolume)
				Expect(dataVolume.ObjectMeta.OwnerReferences[0].UID).To(Equal(uid))
				return true, update.GetObject(), nil
			})
		}

		shouldExpectDataVolumeDeletion := func(uid types.UID, idx *int) {
			cdiClient.Fake.PrependReactor("delete", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())

				*idx++
				return true, nil, nil
			})
		}

		shouldExpectDataVolumeUpdate := func(uid types.UID, name string) {
			cdiClient.Fake.PrependReactor("update", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				dataVolume := update.GetObject().(*cdiv1.DataVolume)

				Expect(dataVolume.Name).To(Equal(name))
				now := time.Now().UTC().Unix()
				deleteAfterStr, ok := dataVolume.Annotations[dataVolumeDeleteAfterTimestampAnno]
				Expect(ok).To(BeTrue())

				deleteAfterTimestamp, err := strconv.ParseInt(deleteAfterStr, 10, 64)
				Expect(err).To(BeNil())

				Expect(deleteAfterTimestamp > now).To(BeTrue())

				return true, update.GetObject(), nil
			})
		}

		addVirtualMachine := func(vm *v1.VirtualMachine) {
			syncCaches(stop)
			mockQueue.ExpectAdds(1)
			vmSource.Add(vm)
			mockQueue.Wait()
		}

		It("should create missing DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume.Namespace = "default"
			dataVolumeFeeder.Add(existingDataVolume)
			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, &createCount)
			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		It("should not delete failed DataVolume for VirtualMachineInstance until after timeout", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			// set the delete after timestamp way into the future
			existingDataVolume1.Annotations[dataVolumeDeleteAfterTimestampAnno] = strconv.FormatInt(time.Now().UTC().Unix()+60, 10)

			existingDataVolume2 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			deletionCount := 0

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			Expect(deletionCount).To(Equal(0))
			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
		})

		It("should delete failed DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed
			existingDataVolume1.Annotations[dataVolumeDeleteAfterTimestampAnno] = strconv.FormatInt(time.Now().UTC().Unix(), 10)

			existingDataVolume2 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			deletionCount := 0
			shouldExpectDataVolumeDeletion(vm.UID, &deletionCount)

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			Expect(deletionCount).To(Equal(1))
			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
		})

		It("should not delete failed DataVolume for VirtualMachineInstance unless deletion timestamp expires ", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			shouldExpectDataVolumeUpdate(vm.UID, existingDataVolume1.Name)

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
		})

		It("should handle failed DataVolume without Annotations", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed
			// explicitly delete the annotations field
			existingDataVolume1.Annotations = nil

			existingDataVolume2 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded
			existingDataVolume2.Annotations = nil

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			shouldExpectDataVolumeUpdate(vm.UID, existingDataVolume1.Name)

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
		})

		It("should only start VMI once DataVolumes are complete", func() {

			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should Not delete Datavolumes when VMI is stopped", func() {

			vm, vmi := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)

			dataVolumeFeeder.Add(existingDataVolume)
			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should create multiple DataVolumes for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, &createCount)
			controller.Execute()
			Expect(createCount).To(Equal(2))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})
		It("should create missing VirtualMachineInstance", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should update status to created if the vmi exists", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.Status.Phase = v1.Scheduled

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			// expect update status is called
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should update status to created and ready when vmi is running and running", func() {
			vm, vmi := DefaultVirtualMachine(true)
			markAsReady(vmi)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			// expect update status is called
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeTrue())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should have stable firmware UUIDs", func() {
			vm1, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi1 := controller.setupVMIFromVM(vm1)

			// intentionally use the same names
			vm2, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vmi2 := controller.setupVMIFromVM(vm2)
			Expect(vmi1.Spec.Domain.Firmware.UUID).To(Equal(vmi2.Spec.Domain.Firmware.UUID))

			// now we want different names
			vm3, _ := DefaultVirtualMachineWithNames(true, "testvm3", "testvmi3")
			vmi3 := controller.setupVMIFromVM(vm3)
			Expect(vmi1.Spec.Domain.Firmware.UUID).NotTo(Equal(vmi3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewRandom().String()
			vm1, _ := DefaultVirtualMachineWithNames(true, "testvm1", "testvmi1")
			vm1.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{UUID: types.UID(uid)}

			vmi1 := controller.setupVMIFromVM(vm1)
			Expect(string(vmi1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VirtualMachineInstance when stopped", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			// vmInterface.EXPECT().Update(gomock.Any()).Return(vm, nil)
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should not delete the VirtualMachineInstance again if it is already marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(false)
			vmi.DeletionTimestamp = now()

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()
		})

		It("should ignore non-matching VMIs", func() {
			vm, vmi := DefaultVirtualMachine(true)

			nonMatchingVMI := v1.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addVirtualMachine(vm)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			vmiSource.Add(nonMatchingVMI)

			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
			vmInterface.EXPECT().Update(gomock.Any()).Times(2).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VirtualMachineInstance", func() {
			vm, vmi := DefaultVirtualMachine(true)

			// Mark it as orphan deleted
			now := metav1.Now()
			vm.ObjectMeta.DeletionTimestamp = &now
			vm.ObjectMeta.Finalizers = []string{metav1.FinalizerOrphanDependents}

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().Get(vm.ObjectMeta.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().Update(gomock.Any()).Return(vm, nil)
			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().Update(gomock.Any()).Return(vm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			// vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				Expect(objVM.Status.Conditions).To(HaveLen(1))
				cond := objVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if deletion fails", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				Expect(objVM.Status.Conditions).To(HaveLen(1))
				cond := objVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})

		It("should back off if a sync error occurs", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				Expect(objVM.Status.Conditions).To(HaveLen(1))
				cond := objVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})
	})
})

func VirtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineSpec{
			Running: &started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := v1.NewMinimalVMI(vmiName)
	vmi.Status.Phase = v1.Running
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	t := true
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return vm, vmi
}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}
