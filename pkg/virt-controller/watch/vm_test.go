package watch

import (
	"fmt"

	"github.com/go-openapi/errors"
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

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"my": "label"},
					Annotations: map[string]string{"my": "annotation"},
					Name:        "dv1",
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
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": "", "my": "label"}, map[string]string{"my": "annotation"}, &createCount)
			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

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

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)

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
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": ""}, map[string]string{}, &createCount)
			controller.Execute()
			Expect(createCount).To(Equal(2))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		Context("clone authorization tests", func() {
			dv1 := &cdiv1.DataVolume{
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

			dv2 := &cdiv1.DataVolume{
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

			table.DescribeTable("create clone DataVolume for VirtualMachineInstance", func(dv *cdiv1.DataVolume, saVol *v1.Volume, fail bool) {
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
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Return(vm, nil)
			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

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

			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*v1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		Context("VM rename", func() {
			Context("source VM", func() {
				var vm *v1.VirtualMachine

				BeforeEach(func() {
					vm, _ = DefaultVirtualMachineWithNames(false, "test", "")
					virtcontroller.SetLatestApiVersionAnnotation(vm)
				})

				Context("a VM with the new name exists", func() {
					var newVM *v1.VirtualMachine

					BeforeEach(func() {
						newVM, _ = DefaultVirtualMachineWithNames(false, "newtest", "")
						vm.Status = v1.VirtualMachineStatus{
							StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
								{
									Action: v1.RenameRequest,
									Data: map[string]string{
										"newName": newVM.Name,
									},
								},
							},
						}
					})

					It("should remove the VM with the new name and create a copy of the source VM", func() {
						vmInterface.EXPECT().Delete(newVM.Name, gomock.Any())
						vmInterface.EXPECT().Create(gomock.Any()).
							Do(func(objs ...interface{}) {
								vm := objs[0].(*v1.VirtualMachine)

								Expect(vm.Name).To(Equal(newVM.Name))
								Expect(len(vm.Status.Conditions)).To(Equal(1))
								Expect(vm.Status.Conditions[0].Type).To(Equal(virtv1.RenameConditionType))
							})

						vmInterface.EXPECT().Delete(vm.Name, gomock.Any())

						addVirtualMachine(vm)
						controller.Execute()
					})
				})

				Context("a VM with the new name does not exist", func() {
					var newName string

					BeforeEach(func() {
						newName = "newtest"

						vmInterface.EXPECT().Delete(newName, gomock.Any()).Return(errors.NotFound("not found"))

						vm.Status = v1.VirtualMachineStatus{
							StateChangeRequests: []v1.VirtualMachineStateChangeRequest{
								{
									Action: v1.RenameRequest,
									Data: map[string]string{
										"newName": newName,
									},
								},
							},
						}
					})

					It("should create a new VM with the new name", func() {
						newVM := vm.DeepCopy()
						newVM.Name = newName

						vmInterface.EXPECT().
							Create(gomock.Any()).
							Do(func(objs ...interface{}) {
								vm := objs[0].(*v1.VirtualMachine)

								Expect(vm.Name).To(Equal(newName))
								Expect(len(vm.Status.Conditions)).To(Equal(1))
								Expect(vm.Status.Conditions[0].Type).To(Equal(virtv1.RenameConditionType))
							}).
							Return(newVM, nil)

						vmInterface.EXPECT().Delete(vm.Name, gomock.Any())

						addVirtualMachine(vm)
						controller.Execute()
					})
				})
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
