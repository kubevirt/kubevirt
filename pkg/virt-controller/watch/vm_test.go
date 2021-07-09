package watch

import (
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	v1 "kubevirt.io/client-go/api/v1"
	virtv1 "kubevirt.io/client-go/api/v1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
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
		var pvcInformer cache.SharedIndexInformer
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
			pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMController(vmiInformer, vmInformer, dataVolumeInformer, pvcInformer, recorder, virtClient)
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

		shouldExpectDataVolumeCreation := func(uid types.UID, labels map[string]string, annotations map[string]string, idx *int) {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				*idx++
				dataVolume := update.GetObject().(*cdiv1.DataVolume)
				Expect(dataVolume.ObjectMeta.OwnerReferences[0].UID).To(Equal(uid))
				Expect(dataVolume.ObjectMeta.Labels).To(Equal(labels))
				Expect(dataVolume.ObjectMeta.Annotations).To(Equal(annotations))
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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"my": "label"},
					Annotations: map[string]string{"my": "annotation"},
					Name:        "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = v1.VirtualMachineStatusProvisioning
			addVirtualMachine(vm)

			existingDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume.Namespace = "default"
			dataVolumeFeeder.Add(existingDataVolume)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": "", "my": "label"}, map[string]string{"my": "annotation"}, &createCount)

			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		table.DescribeTable("should hotplug a vm", func(isRunning bool) {

			vm, vmi := DefaultVirtualMachine(isRunning)
			vm.Status.Created = true
			vm.Status.Ready = true
			vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
				{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
			}

			addVirtualMachine(vm)

			if isRunning {
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
				vmiInterface.EXPECT().AddVolume(vmi.ObjectMeta.Name, vm.Status.VolumeRequests[0].AddVolumeOptions)
			}

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Spec.Template.Spec.Volumes[0].Name).To(Equal("vol1"))
			}).Return(nil, nil)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change
				Expect(len(arg.(*v1.VirtualMachine).Status.VolumeRequests)).To(Equal(1))
			}).Return(nil, nil)

			controller.Execute()
		},

			table.Entry("that is running", true),
			table.Entry("that is not running", false),
		)

		table.DescribeTable("should unhotplug a vm", func(isRunning bool) {
			vm, vmi := DefaultVirtualMachine(isRunning)
			vm.Status.Created = true
			vm.Status.Ready = true
			vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
				{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "vol1",
					},
				},
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "vol1",
			})
			vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					},
				},
			})

			addVirtualMachine(vm)

			if isRunning {
				vmi.Spec.Volumes = vm.Spec.Template.Spec.Volumes
				vmi.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
				vmiInterface.EXPECT().RemoveVolume(vmi.ObjectMeta.Name, vm.Status.VolumeRequests[0].RemoveVolumeOptions)
			}

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachine).Spec.Template.Spec.Volumes)).To(Equal(0))
			}).Return(nil, nil)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change occured
				Expect(len(arg.(*v1.VirtualMachine).Status.VolumeRequests)).To(Equal(1))
			}).Return(nil, nil)

			controller.Execute()
		},

			table.Entry("that is running", true),
			table.Entry("that is not running", false),
		)

		table.DescribeTable("should clear VolumeRequests for added volumes that are satisfied", func(isRunning bool) {
			vm, vmi := DefaultVirtualMachine(isRunning)
			vm.Status.Created = true
			vm.Status.Ready = true
			vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
				{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				},
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "vol1",
			})
			vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					},
				},
			})

			addVirtualMachine(vm)

			if isRunning {
				vmi.Spec.Volumes = vm.Spec.Template.Spec.Volumes
				vmi.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
			}

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachine).Status.VolumeRequests)).To(Equal(0))
			}).Return(nil, nil)

			controller.Execute()
		},

			table.Entry("that is running", true),
			table.Entry("that is not running", false),
		)

		table.DescribeTable("should clear VolumeRequests for removed volumes that are satisfied", func(isRunning bool) {
			vm, vmi := DefaultVirtualMachine(isRunning)
			vm.Status.Created = true
			vm.Status.Ready = true
			vm.Status.VolumeRequests = []v1.VirtualMachineVolumeRequest{
				{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "vol1",
					},
				},
			}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{}
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{}

			addVirtualMachine(vm)

			if isRunning {
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
			}

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachine).Status.VolumeRequests)).To(Equal(0))
			}).Return(nil, nil)

			controller.Execute()
		},

			table.Entry("that is running", true),
			table.Entry("that is not running", false),
		)

		It("should not delete failed DataVolume for VirtualMachineInstance", func() {
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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
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

			deletionCount := 0
			shouldExpectDataVolumeDeletion(vm.UID, &deletionCount)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			Expect(deletionCount).To(Equal(0))
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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
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

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
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

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
		})

		It("should start VMI once DataVolumes are complete", func() {

			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should start VMI once DataVolumes are complete or WaitForFirstConsumer", func() {
			// WaitForFirstConsumer state can only be handled by VMI

			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.WaitForFirstConsumer
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
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
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)
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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = v1.VirtualMachineStatusProvisioning
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": ""}, map[string]string{}, &createCount)

			controller.Execute()
			Expect(createCount).To(Equal(2))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		Context("clone authorization tests", func() {
			dv1 := &v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: "ns1",
							Name:      "source-pvc",
						},
					},
				},
			}

			dv2 := &v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Name: "source-pvc",
						},
					},
				},
			}

			serviceAccountVol := &v1.Volume{
				Name: "sa",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{
						ServiceAccountName: "sa",
					},
				},
			}

			table.DescribeTable("create clone DataVolume for VirtualMachineInstance", func(dv *v1.DataVolumeTemplateSpec, saVol *v1.Volume, fail bool) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes,
					v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: dv.Name,
							},
						},
					},
				)

				if saVol != nil {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, *saVol)
				}

				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dv)

				vm.Status.PrintableStatus = v1.VirtualMachineStatusProvisioning
				addVirtualMachine(vm)

				createCount := 0
				shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": ""}, map[string]string{}, &createCount)
				if fail {
					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)
				}

				controller.cloneAuthFunc = func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
					if dv.Spec.Source.PVC.Namespace != "" {
						Expect(pvcNamespace).Should(Equal(dv.Spec.Source.PVC.Namespace))
					} else {
						Expect(pvcNamespace).Should(Equal(vm.Namespace))
					}

					Expect(pvcName).Should(Equal(dv.Spec.Source.PVC.Name))
					Expect(saNamespace).Should(Equal(vm.Namespace))

					if saVol != nil {
						Expect(saName).Should(Equal("sa"))
					} else {
						Expect(saName).Should(Equal("default"))
					}

					if fail {
						return false, "Authorization failed", nil
					}

					return true, "", nil
				}
				controller.Execute()
				if fail {
					Expect(createCount).To(Equal(0))
					testutils.ExpectEvent(recorder, UnauthorizedDataVolumeCreateReason)
				} else {
					Expect(createCount).To(Equal(1))
					testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
				}
			},
				table.Entry("with auth and source namespace defined", dv1, serviceAccountVol, false),
				table.Entry("with auth and no source namespace defined", dv2, serviceAccountVol, false),
				table.Entry("with auth and source namespace no serviceaccount defined", dv1, nil, false),
				table.Entry("with no auth and source namespace defined", dv1, serviceAccountVol, true),
			)
		})

		It("should create missing VirtualMachineInstance", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should ignore the name of a VirtualMachineInstance templates", func() {
			vm, vmi := DefaultVirtualMachineWithNames(true, "vmname", "vminame")

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("vmname"))
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
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

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should not delete the VirtualMachineInstance again if it is already marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(false)
			vmi.DeletionTimestamp = now()

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(2).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().Get(vm.ObjectMeta.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Return(vm, nil)
			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a DataVolume already exists and adopt it", func() {
			vm, _ := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dv1",
					Namespace: vm.Namespace,
				},
			})

			addVirtualMachine(vm)

			dv := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
			dv.Status.Phase = cdiv1.Succeeded

			orphanDV := dv.DeepCopy()
			orphanDV.ObjectMeta.OwnerReferences = nil
			dataVolumeInformer.GetStore().Add(orphanDV)

			cdiClient.Fake.PrependReactor("patch", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patch, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(patch.GetName()).To(Equal(dv.Name))
				Expect(patch.GetNamespace()).To(Equal(dv.Namespace))
				Expect(string(patch.GetPatch())).To(ContainSubstring(string(vm.UID)))
				Expect(string(patch.GetPatch())).To(ContainSubstring("ownerReferences"))
				return true, dv, nil
			})

			vmInterface.EXPECT().Get(vm.ObjectMeta.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Return(vm, nil)

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Return(vm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			// vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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

		table.DescribeTable("should add ready condition", func(setup func(vmi *v1.VirtualMachineInstance), status k8sv1.ConditionStatus) {
			vm, vmi := DefaultVirtualMachine(true)
			addVirtualMachine(vm)

			setup(vmi)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, v1.VirtualMachineReady)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(status))
			}).Return(vm, nil)

			controller.Execute()
		},
			table.Entry("True", markAsReady, k8sv1.ConditionTrue),
			table.Entry("False", markAsNonReady, k8sv1.ConditionFalse),
		)

		It("should add paused condition", func() {
			vm, vmi := DefaultVirtualMachine(true)
			addVirtualMachine(vm)

			markAsReady(vmi)
			vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			})
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, v1.VirtualMachinePaused)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(vm, nil)

			controller.Execute()
		})

		It("should remove paused condition", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Status.Conditions = append(vm.Status.Conditions, virtv1.VirtualMachineCondition{
				Type:   virtv1.VirtualMachinePaused,
				Status: k8sv1.ConditionTrue,
			})
			addVirtualMachine(vm)

			markAsReady(vmi)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, v1.VirtualMachinePaused)
				Expect(cond).To(BeNil())
			}).Return(vm, nil)

			controller.Execute()
		})

		It("should back off if a sync error occurs", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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

		It("should copy annotations from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"test": "test"}
			annotations := map[string]string{"test": "test"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*v1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubevirt ignitiondata annotation from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"kubevirt.io/ignitiondata": "test"}
			annotations := map[string]string{"kubevirt.io/ignitiondata": "test"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*v1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubernetes annotations from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}
			annotations := map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}

			vm.Status.PrintableStatus = v1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*v1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		Context("VM printableStatus", func() {

			It("Should set a Stopped status when running=false and VMI doesn't exist", func() {
				vm, _ := DefaultVirtualMachine(false)
				addVirtualMachine(vm)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopped))
				})

				controller.Execute()
			})

			table.DescribeTable("should set a Stopped status when VMI exists but stopped", func(phase v1.VirtualMachineInstancePhase, deletionTimestamp *metav1.Time) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase
				vmi.ObjectMeta.DeletionTimestamp = deletionTimestamp

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopped))
				})

				controller.Execute()
			},

				table.Entry("in Succeeded state", v1.Succeeded, nil),
				table.Entry("in Succeeded state with a deletionTimestamp", v1.Succeeded, &metav1.Time{Time: time.Now()}),
				table.Entry("in Failed state", v1.Failed, nil),
				table.Entry("in Failed state with a deletionTimestamp", v1.Failed, &metav1.Time{Time: time.Now()}),
			)

			It("Should set a Starting status when running=true and VMI doesn't exist", func() {
				vm, vmi := DefaultVirtualMachine(true)
				addVirtualMachine(vm)

				vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			})

			table.DescribeTable("Should set a Starting status when VMI is in a startup phase", func(phase v1.VirtualMachineInstancePhase) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			},

				table.Entry("VMI has no phase set", v1.VmPhaseUnset),
				table.Entry("VMI is in Pending phase", v1.Pending),
				table.Entry("VMI is in Scheduling phase", v1.Scheduling),
				table.Entry("VMI is in Scheduled phase", v1.Scheduled),
			)

			Context("VM with DataVolumes", func() {
				var vm *v1.VirtualMachine
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vm, vmi = DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv1",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv1",
							Namespace: vm.Namespace,
						},
					})
				})

				table.DescribeTable("Should set a Provisioning status when DataVolume doesn't exist", func(running bool) {
					vm.Spec.Running = &running
					addVirtualMachine(vm)

					createCount := 0
					shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": ""}, map[string]string{}, &createCount)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
					})

					controller.Execute()
					Expect(createCount).To(Equal(1))
				},

					table.Entry("running=true", true),
					table.Entry("running=false", false),
				)

				table.DescribeTable("Should set a Provisioning status when DataVolume exists but unready", func(dvPhase cdiv1.DataVolumePhase) {
					addVirtualMachine(vm)

					dv := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = dvPhase
					dataVolumeFeeder.Add(dv)

					if dvPhase == cdiv1.WaitForFirstConsumer {
						vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
					}
					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
					})

					controller.Execute()
				},

					table.Entry("DataVolume has no phase set", cdiv1.PhaseUnset),
					table.Entry("DataVolume is in Pending phase", cdiv1.Pending),
					table.Entry("DataVolume is in ImportScheduled phase", cdiv1.ImportScheduled),
					table.Entry("DataVolume is in ImportInProgress phase", cdiv1.ImportInProgress),
					table.Entry("DataVolume is in Failed phase", cdiv1.Failed),
					table.Entry("DataVolume is in WaitForFirstConsumer phase", cdiv1.WaitForFirstConsumer),
				)

				It("Should set a Provisioning status when one DataVolume is ready and another isn't", func() {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test2",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv2",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv2",
							Namespace: vm.Namespace,
						},
					})

					addVirtualMachine(vm)

					dv1 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[0], vm)
					dv1.Status.Phase = cdiv1.Succeeded
					dv2 := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[1], vm)
					dv2.Status.Phase = cdiv1.ImportInProgress

					dataVolumeFeeder.Add(dv1)
					dataVolumeFeeder.Add(dv2)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
					})

					controller.Execute()
				})
			})

			Context("VM with PersistentVolumeClaims", func() {
				var vm *v1.VirtualMachine

				BeforeEach(func() {
					vm, _ = DefaultVirtualMachine(false)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
						Name: "test1",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc1",
							},
						},
					})

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusProvisioning))
					})
				})

				It("Should set a Provisioning status when PersistentVolumeClaim doesn't exist", func() {
					controller.Execute()
				})

				table.DescribeTable("Should set a Provisioning status when PersistentVolumeClaim exists but unready", func(pvcPhase k8sv1.PersistentVolumeClaimPhase) {
					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pvc1",
							Namespace: vm.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: pvcPhase,
						},
					}
					pvcInformer.GetStore().Add(pvc)

					controller.Execute()
				},

					table.Entry("PersistentVolumeClaim is in Pending phase", k8sv1.ClaimPending),
					table.Entry("PersistentVolumeClaim is in Lost phase", k8sv1.ClaimLost),
				)

			})

			It("should set a Running status when VMI is running but not paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusRunning))
				})

				controller.Execute()
			})

			It("should set a Paused status when VMI is running but is paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running
				vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				})

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusPaused))
				})

				controller.Execute()
			})

			table.DescribeTable("should set a Stopping status when VMI has a deletion timestamp set", func(phase v1.VirtualMachineInstancePhase, condType v1.VirtualMachineInstanceConditionType) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				vmi.Status.Phase = phase

				if condType != "" {
					vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
						Type:   condType,
						Status: k8sv1.ConditionTrue,
					})
				}
				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusStopping))
				})

				controller.Execute()
			},

				table.Entry("when VMI is pending", v1.Pending, v1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is provisioning", v1.Pending, v1.VirtualMachineInstanceProvisioning),
				table.Entry("when VMI is scheduling", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is scheduled", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is running", v1.Running, v1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is paused", v1.Running, v1.VirtualMachineInstancePaused),
			)

			Context("should set a Terminating status when VM has a deletion timestamp set", func() {
				table.DescribeTable("when VMI exists", func(phase v1.VirtualMachineInstancePhase, condType v1.VirtualMachineInstanceConditionType) {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = phase

					if condType != "" {
						vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
							Type:   condType,
							Status: k8sv1.ConditionTrue,
						})
					}
					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					table.Entry("when VMI is pending", v1.Pending, v1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is provisioning", v1.Pending, v1.VirtualMachineInstanceProvisioning),
					table.Entry("when VMI is scheduling", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is scheduled", v1.Scheduling, v1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is running", v1.Running, v1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is paused", v1.Running, v1.VirtualMachineInstancePaused),
				)

				It("when VMI exists and has a deletion timestamp set", func() {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = v1.Running

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				})

				table.DescribeTable("when VMI does not exist", func(running bool) {
					vm, _ := DefaultVirtualMachine(running)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*v1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					table.Entry("with running: true", true),
					table.Entry("with running: false", false),
				)
			})

			It("should set a Migrating status when VMI is migrating", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Running
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					StartTimestamp: &metav1.Time{Time: time.Now()},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusMigrating))
				})

				controller.Execute()
			})

			It("should set an Unknown status when VMI is in unknown phase", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = v1.Unknown

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*v1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusUnknown))
				})

				controller.Execute()
			})
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
	vmi.GenerateName = "prettyrandom"
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
	virtcontroller.SetLatestApiVersionAnnotation(vmi)
	virtcontroller.SetLatestApiVersionAnnotation(vm)
	return vm, vmi
}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}
