package watch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	appsv1 "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/api"

	virtv1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/flavor"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var (
	vmUID types.UID = "vm-uid"
	t               = true
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
		var crInformer cache.SharedIndexInformer
		var flavorMethods *testutils.MockFlavorMethods
		var stop chan struct{}
		var controller *VMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmiFeeder *testutils.VirtualMachineFeeder
		var dataVolumeFeeder *testutils.DataVolumeFeeder
		var cdiClient *cdifake.Clientset
		var k8sClient *k8sfake.Clientset
		var virtClient *kubecli.MockKubevirtClient

		syncCaches := func(stop chan struct{}) {
			go vmiInformer.Run(stop)
			go vmInformer.Run(stop)
			go dataVolumeInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, vmInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			generatedInterface := fake.NewSimpleClientset()

			dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
			vmInformer, vmSource = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
			pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			crInformer, _ = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
				"vm": func(obj interface{}) ([]string, error) {
					cr := obj.(*appsv1.ControllerRevision)
					for _, ref := range cr.OwnerReferences {
						if ref.Kind == "VirtualMachine" {
							return []string{string(ref.UID)}, nil
						}
					}
					return nil, nil
				},
			})

			flavorMethods = testutils.NewMockFlavorMethods()

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			controller = NewVMController(vmiInformer,
				vmInformer,
				dataVolumeInformer,
				pvcInformer,
				crInformer,
				flavorMethods,
				recorder,
				virtClient)

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue

			vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
			dataVolumeFeeder = testutils.NewDataVolumeFeeder(mockQueue, dataVolumeSource)

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().GeneratedKubeVirtClient().Return(generatedInterface).AnyTimes()

			cdiClient = cdifake.NewSimpleClientset()
			virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			cdiClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		})

		shouldExpectVMIFinalizerRemoval := func(vmi *virtv1.VirtualMachineInstance) {
			patch := `[{ "op": "test", "path": "/metadata/finalizers", "value": ["kubevirt.io/virtualMachineControllerFinalize"] }, { "op": "replace", "path": "/metadata/finalizers", "value": [] }]`

			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
		}

		shouldExpectDataVolumeCreationPriorityClass := func(uid types.UID, labels map[string]string, annotations map[string]string, priorityClassName string, idx *int) {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				*idx++
				dataVolume := update.GetObject().(*cdiv1.DataVolume)
				Expect(dataVolume.ObjectMeta.OwnerReferences[0].UID).To(Equal(uid))
				Expect(dataVolume.ObjectMeta.Labels).To(Equal(labels))
				Expect(dataVolume.ObjectMeta.Annotations).To(Equal(annotations))
				Expect(dataVolume.Spec.PriorityClassName).To(Equal(priorityClassName))
				return true, update.GetObject(), nil
			})
		}

		shouldExpectDataVolumeCreation := func(uid types.UID, labels map[string]string, annotations map[string]string, idx *int) {
			shouldExpectDataVolumeCreationPriorityClass(uid, labels, annotations, "", idx)
		}

		shouldExpectDataVolumeDeletion := func(uid types.UID, idx *int) {
			cdiClient.Fake.PrependReactor("delete", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())

				*idx++
				return true, nil, nil
			})
		}

		expectControllerRevisionList := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("list", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				list, _ := action.(testing.ListAction)
				if strings.Contains(list.GetListRestrictions().Labels.String(), string(vmUID)) {
					return true, &appsv1.ControllerRevisionList{Items: []appsv1.ControllerRevision{*vmRevision}}, nil
				}
				return true, &appsv1.ControllerRevisionList{}, nil
			})
		}

		expectControllerRevisionDelete := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())
				Expect(deleted.GetNamespace()).To(Equal(vmRevision.Namespace))
				Expect(deleted.GetName()).To(Equal(vmRevision.Name))
				return true, nil, nil
			})
		}

		expectControllerRevisionCreation := func(vmRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				createObj := created.GetObject().(*appsv1.ControllerRevision)
				Expect(createObj).To(Equal(vmRevision))

				return true, created.GetObject(), nil
			})
		}

		patchVMRevision := func(vm *virtv1.VirtualMachine) runtime.RawExtension {
			vmBytes, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			var raw map[string]interface{}
			err = json.Unmarshal(vmBytes, &raw)
			Expect(err).ToNot(HaveOccurred())

			objCopy := make(map[string]interface{})
			spec := raw["spec"].(map[string]interface{})
			objCopy["spec"] = spec
			patch, err := json.Marshal(objCopy)
			Expect(err).ToNot(HaveOccurred())
			return runtime.RawExtension{Raw: patch}
		}

		createVMRevision := func(vm *virtv1.VirtualMachine) *appsv1.ControllerRevision {
			return &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getVMRevisionName(vm.UID, vm.Generation),
					Namespace: vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
						Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
						Name:               vm.ObjectMeta.Name,
						UID:                vm.ObjectMeta.UID,
						Controller:         &t,
						BlockOwnerDeletion: &t,
					}},
				},
				Data:     patchVMRevision(vm),
				Revision: vm.Generation,
			}
		}

		addVirtualMachine := func(vm *virtv1.VirtualMachine) {
			syncCaches(stop)
			mockQueue.ExpectAdds(1)
			vmSource.Add(vm)
			mockQueue.Wait()
		}

		It("should create missing DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"my": "label"},
					Annotations: map[string]string{"my": "annotation"},
					Name:        "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStopped
			addVirtualMachine(vm)

			existingDataVolume, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume.Namespace = "default"
			dataVolumeFeeder.Add(existingDataVolume)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID), "my": "label"}, map[string]string{"my": "annotation"}, &createCount)

			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		table.DescribeTable("should hotplug a vm", func(isRunning bool) {

			vm, vmi := DefaultVirtualMachine(isRunning)
			vm.Status.Created = true
			vm.Status.Ready = true
			vm.Status.VolumeRequests = []virtv1.VirtualMachineVolumeRequest{
				{
					AddVolumeOptions: &virtv1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &virtv1.Disk{},
						VolumeSource: &virtv1.HotplugVolumeSource{},
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
				Expect(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes[0].Name).To(Equal("vol1"))
			}).Return(nil, nil)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change
				Expect(len(arg.(*virtv1.VirtualMachine).Status.VolumeRequests)).To(Equal(1))
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
			vm.Status.VolumeRequests = []virtv1.VirtualMachineVolumeRequest{
				{
					RemoveVolumeOptions: &virtv1.RemoveVolumeOptions{
						Name: "vol1",
					},
				},
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, virtv1.Disk{
				Name: "vol1",
			})
			vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name: "vol1",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					}},
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
				Expect(len(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes)).To(Equal(0))
			}).Return(nil, nil)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change occured
				Expect(len(arg.(*virtv1.VirtualMachine).Status.VolumeRequests)).To(Equal(1))
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
			vm.Status.VolumeRequests = []virtv1.VirtualMachineVolumeRequest{
				{
					AddVolumeOptions: &virtv1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &virtv1.Disk{},
						VolumeSource: &virtv1.HotplugVolumeSource{},
					},
				},
			}
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, virtv1.Disk{
				Name: "vol1",
			})
			vm.Spec.Template.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name: "vol1",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testpvcdiskclaim",
					}},
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
				Expect(len(arg.(*virtv1.VirtualMachine).Status.VolumeRequests)).To(Equal(0))
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
			vm.Status.VolumeRequests = []virtv1.VirtualMachineVolumeRequest{
				{
					RemoveVolumeOptions: &virtv1.RemoveVolumeOptions{
						Name: "vol1",
					},
				},
			}
			vm.Spec.Template.Spec.Volumes = []virtv1.Volume{}
			vm.Spec.Template.Spec.Domain.Devices.Disks = []virtv1.Disk{}

			addVirtualMachine(vm)

			if isRunning {
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
			}

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*virtv1.VirtualMachine).Status.VolumeRequests)).To(Equal(0))
			}).Return(nil, nil)

			controller.Execute()
		},

			table.Entry("that is running", true),
			table.Entry("that is not running", false),
		)

		It("should not delete failed DataVolume for VirtualMachineInstance", func() {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[1], vm)
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
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[1], vm)
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
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})
			addVirtualMachine(vm)

			existingDataVolume1, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed
			// explicitly delete the annotations field
			existingDataVolume1.Annotations = nil

			existingDataVolume2, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[1], vm)
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
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should start VMI once DataVolumes are complete or WaitForFirstConsumer", func() {
			// WaitForFirstConsumer state can only be handled by VMI

			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.WaitForFirstConsumer
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should Not delete Datavolumes when VMI is stopped", func() {

			vm, vmi := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})

			existingDataVolume, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)

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
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv2",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			})
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
			})

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStopped
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, &createCount)

			controller.Execute()
			Expect(createCount).To(Equal(2))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		table.DescribeTable("should properly set priority class", func(dvPriorityClass, vmPriorityClass, expectedPriorityClass string) {
			vm, _ := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					PriorityClassName: dvPriorityClass,
				},
			})
			vm.Spec.Template.Spec.PriorityClassName = vmPriorityClass
			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStopped
			addVirtualMachine(vm)

			createCount := 0
			shouldExpectDataVolumeCreationPriorityClass(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, expectedPriorityClass, &createCount)

			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		},
			table.Entry("when dv priorityclass is not defined and VM priorityclass is defined", "", "vmpriority", "vmpriority"),
			table.Entry("when dv priorityclass is defined and VM priorityclass is defined", "dvpriority", "vmpriority", "dvpriority"),
			table.Entry("when dv priorityclass is defined and VM priorityclass is not defined", "dvpriority", "", "dvpriority"),
			table.Entry("when dv priorityclass is not defined and VM priorityclass is not defined", "", "", ""),
		)

		Context("crashloop backoff tests", func() {

			It("should track start failures when VMIs fail without hitting running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "123"
				vmi.Status.Phase = virtv1.Failed

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.ConsecutiveFailCount).To(Equal(1))
				}).Return(nil, nil)

				shouldExpectVMIFinalizerRemoval(vmi)

				controller.Execute()

				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			})

			It("should track a new start failures when a new VMI fails without hitting running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = virtv1.Failed

				oldRetry := time.Now().Add(-300 * time.Second)
				vm.Status.StartFailure = &virtv1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.RetryAfterTimestamp).ToNot(Equal(oldRetry))
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.LastFailedVMIUID).To(Equal(vmi.UID))
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.ConsecutiveFailCount).To(Equal(2))
				}).Return(nil, nil)

				shouldExpectVMIFinalizerRemoval(vmi)

				controller.Execute()

				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			})

			It("should clear start failures when VMI hits running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = virtv1.Running
				vmi.Status.PhaseTransitionTimestamps = []virtv1.VirtualMachineInstancePhaseTransitionTimestamp{
					{
						Phase:                    virtv1.Running,
						PhaseTransitionTimestamp: metav1.Now(),
					},
				}

				oldRetry := time.Now().Add(-300 * time.Second)
				vm.Status.StartFailure = &virtv1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).To(BeNil())
				}).Return(nil, nil)

				controller.Execute()

			})

			table.DescribeTable("should clear existing start failures when runStrategy is halted or manual", func(runStrategy virtv1.VirtualMachineRunStrategy) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "456"
				vmi.Status.Phase = virtv1.Failed
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy

				oldRetry := time.Now().Add(300 * time.Second)
				vm.Status.StartFailure = &virtv1.VirtualMachineStartFailure{
					LastFailedVMIUID:     "123",
					ConsecutiveFailCount: 1,
					RetryAfterTimestamp: &metav1.Time{
						Time: oldRetry,
					},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(arg interface{}) {
					if runStrategy == virtv1.RunStrategyHalted || runStrategy == virtv1.RunStrategyManual {
						Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).To(BeNil())
					} else {
						Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).ToNot(BeNil())

					}
				}).Return(nil, nil)

				//	if runStrategy != v1.RunStrategyManual {
				shouldExpectVMIFinalizerRemoval(vmi)
				//	}

				controller.Execute()

				if runStrategy != virtv1.RunStrategyManual {
					testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
				}
			},

				table.Entry("runStrategyHalted", virtv1.RunStrategyHalted),
				table.Entry("always", virtv1.RunStrategyAlways),
				table.Entry("manual", virtv1.RunStrategyManual),
				table.Entry("rerunOnFailure", virtv1.RunStrategyRerunOnFailure),
			)

			table.DescribeTable("should calculated expected backoff delay", func(failCount, minExpectedDelay int, maxExpectedDelay int) {

				for i := 0; i < 1000; i++ {
					delay := calculateStartBackoffTime(failCount, defaultMaxCrashLoopBackoffDelaySeconds)

					if delay > maxExpectedDelay {
						Expect(fmt.Errorf("delay: %d: failCount %d should not result in a delay greater than %d", delay, failCount, maxExpectedDelay)).To(BeNil())
					} else if delay < minExpectedDelay {
						Expect(fmt.Errorf("delay: %d: failCount %d should not result in a delay less than than %d", delay, failCount, minExpectedDelay)).To(BeNil())

					}
				}
			},

				table.Entry("failCount 0", 0, 10, 15),
				table.Entry("failCount 1", 1, 10, 15),
				table.Entry("failCount 2", 2, 40, 60),
				table.Entry("failCount 3", 3, 90, 135),
				table.Entry("failCount 4", 4, 160, 240),
				table.Entry("failCount 5", 5, 250, 300),
				table.Entry("failCount 6", 6, 300, 300),
			)

			table.DescribeTable("has start failure backoff expired", func(vmFunc func() *virtv1.VirtualMachine, expected int64) {
				vm := vmFunc()
				seconds := startFailureBackoffTimeLeft(vm)

				// since the tests all run in parallel, it's difficult to
				// do precise timing. We use a tolerance of 2 seconds to account
				// for some delays in test execution and make sure the calculation
				// falls within the ballpark of what we expect.
				const tolerance = 2
				Expect(seconds).To(BeNumerically("~", expected, tolerance))
				Expect(seconds).To(BeNumerically(">=", 0))
			},

				table.Entry("no vm start failures",
					func() *virtv1.VirtualMachine {
						return &virtv1.VirtualMachine{}
					},
					int64(0)),
				table.Entry("vm failure waiting 300 seconds",
					func() *virtv1.VirtualMachine {
						return &virtv1.VirtualMachine{
							Status: virtv1.VirtualMachineStatus{
								StartFailure: &virtv1.VirtualMachineStartFailure{
									RetryAfterTimestamp: &metav1.Time{
										Time: time.Now().Add(300 * time.Second),
									},
								},
							},
						}
					},
					int64(300)),
				table.Entry("vm failure 300 seconds past retry time",
					func() *virtv1.VirtualMachine {
						return &virtv1.VirtualMachine{
							Status: virtv1.VirtualMachineStatus{
								StartFailure: &virtv1.VirtualMachineStartFailure{
									RetryAfterTimestamp: &metav1.Time{
										Time: time.Now().Add(-300 * time.Second),
									},
								},
							},
						}
					},
					int64(0)),
			)
		})

		Context("clone authorization tests", func() {
			dv1 := &virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: "ns1",
							Name:      "source-pvc",
						},
					},
				},
			}

			dv2 := &virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv2",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Name: "source-pvc",
						},
					},
				},
			}

			ds := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns2",
					Name:      "source-ref",
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: "ns1",
							Name:      "source-pvc",
						},
					},
				},
			}

			dv3 := &virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv3",
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Namespace: &ds.Namespace,
						Name:      ds.Name,
					},
				},
			}

			serviceAccountVol := &virtv1.Volume{
				Name: "sa",
				VolumeSource: virtv1.VolumeSource{
					ServiceAccount: &virtv1.ServiceAccountVolumeSource{
						ServiceAccountName: "sa",
					},
				},
			}

			table.DescribeTable("create clone DataVolume for VirtualMachineInstance", func(dv *virtv1.DataVolumeTemplateSpec, saVol *virtv1.Volume, ds *cdiv1.DataSource, fail bool) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes,
					virtv1.Volume{
						Name: "test1",
						VolumeSource: virtv1.VolumeSource{
							DataVolume: &virtv1.DataVolumeSource{
								Name: dv.Name,
							},
						},
					},
				)

				if saVol != nil {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, *saVol)
				}

				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dv)

				vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStopped
				addVirtualMachine(vm)

				createCount := 0
				shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID)}, map[string]string{}, &createCount)
				if fail {
					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Return(vm, nil)
				}

				if ds != nil {
					cdiClient.PrependReactor("get", "datasources", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						ga := action.(testing.GetAction)
						Expect(ga.GetNamespace()).To(Equal(ds.Namespace))
						Expect(ga.GetName()).To(Equal(ds.Name))
						return true, ds, nil
					})

					cdiClient.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						ca := action.(testing.CreateAction)
						dv := ca.GetObject().(*cdiv1.DataVolume)
						Expect(dv.Spec.SourceRef).To(BeNil())
						Expect(dv.Spec.Source).ToNot(BeNil())
						Expect(dv.Spec.Source.PVC).ToNot(BeNil())
						Expect(dv.Spec.Source.PVC.Namespace).To(Equal(ds.Spec.Source.PVC.Namespace))
						Expect(dv.Spec.Source.PVC.Name).To(Equal(ds.Spec.Source.PVC.Name))
						return false, ds, nil
					})
				}

				controller.cloneAuthFunc = func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
					if dv.Spec.Source != nil {
						if dv.Spec.Source.PVC.Namespace != "" {
							Expect(pvcNamespace).Should(Equal(dv.Spec.Source.PVC.Namespace))
						} else {
							Expect(pvcNamespace).Should(Equal(vm.Namespace))
						}

						Expect(pvcName).Should(Equal(dv.Spec.Source.PVC.Name))
					} else {
						Expect(pvcNamespace).Should(Equal(ds.Spec.Source.PVC.Namespace))
						Expect(pvcName).Should(Equal(ds.Spec.Source.PVC.Name))
					}

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
				table.Entry("with auth and source namespace defined", dv1, serviceAccountVol, nil, false),
				table.Entry("with auth and no source namespace defined", dv2, serviceAccountVol, nil, false),
				table.Entry("with auth and source namespace no serviceaccount defined", dv1, nil, nil, false),
				table.Entry("with no auth and source namespace defined", dv1, serviceAccountVol, nil, true),
				table.Entry("with auth, datasource and source namespace defined", dv3, serviceAccountVol, ds, false),
			)
		})

		It("should create VMI with vmRevision", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Generation = 1

			addVirtualMachine(vm)

			vmRevision := createVMRevision(vm)
			expectControllerRevisionCreation(vmRevision)
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete older vmRevision and create VMI with new one", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Generation = 1
			oldVMRevision := createVMRevision(vm)

			vm.Generation = 2
			addVirtualMachine(vm)
			vmRevision := createVMRevision(vm)

			expectControllerRevisionList(oldVMRevision)
			expectControllerRevisionDelete(oldVMRevision)
			expectControllerRevisionCreation(vmRevision)
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should create missing VirtualMachineInstance", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should ignore the name of a VirtualMachineInstance templates", func() {
			vm, vmi := DefaultVirtualMachineWithNames(true, "vmname", "vminame")

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("vmname"))
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should update status to created if the vmi exists", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.Status.Phase = virtv1.Scheduled

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
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
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeTrue())
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

			nonMatchingVMI := api.NewMinimalVMI("testvmi1")
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
			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any(), &metav1.PatchOptions{})

			controller.Execute()
		})

		It("should detect that a DataVolume already exists and adopt it", func() {
			vm, _ := DefaultVirtualMachine(false)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})

			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dv1",
					Namespace: vm.Namespace,
				},
			})

			addVirtualMachine(vm)

			dv, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
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

			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, fmt.Errorf("some random failure"))

			// We should see the failed condition, replicas should stay at 0
			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
				Expect(cond).To(Not(BeNil()))
				Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(ContainSubstring("some random failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if deletion fails", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("some random failure"))

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
				Expect(cond).To(Not(BeNil()))
				Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(ContainSubstring("some random failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})

		table.DescribeTable("should add ready condition when VMI exists", func(setup func(vmi *virtv1.VirtualMachineInstance), status k8sv1.ConditionStatus) {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, virtv1.VirtualMachineReady)
			addVirtualMachine(vm)

			setup(vmi)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachineReady)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(status))
			}).Return(vm, nil)

			controller.Execute()
		},
			table.Entry("VMI Ready condition is True", markAsReady, k8sv1.ConditionTrue),
			table.Entry("VMI Ready condition is False", markAsNonReady, k8sv1.ConditionFalse),
			table.Entry("VMI Ready condition doesn't exist", unmarkReady, k8sv1.ConditionFalse),
		)

		It("should sync VMI conditions", func() {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, virtv1.VirtualMachineReady)

			cm := virtcontroller.NewVirtualMachineInstanceConditionManager()
			cmVM := virtcontroller.NewVirtualMachineConditionManager()

			addCondList := []virtv1.VirtualMachineInstanceConditionType{
				virtv1.VirtualMachineInstanceProvisioning,
				virtv1.VirtualMachineInstanceSynchronized,
				virtv1.VirtualMachineInstancePaused,
			}

			removeCondList := []virtv1.VirtualMachineInstanceConditionType{
				virtv1.VirtualMachineInstanceAgentConnected,
				virtv1.VirtualMachineInstanceAccessCredentialsSynchronized,
				virtv1.VirtualMachineInstanceUnsupportedAgent,
			}

			updateCondList := []virtv1.VirtualMachineInstanceConditionType{
				virtv1.VirtualMachineInstanceIsMigratable,
			}

			now := metav1.Now()
			for _, condName := range addCondList {
				cm.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
					Type:               condName,
					Status:             k8score.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			for _, condName := range updateCondList {
				// Set to true on VMI
				cm.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
					Type:               condName,
					Status:             k8score.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})

				// Set to false on VM, expect sync to update it to true
				cmVM.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
					Type:               virtv1.VirtualMachineConditionType(condName),
					Status:             k8score.ConditionFalse,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			for _, condName := range removeCondList {
				cmVM.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
					Type:               virtv1.VirtualMachineConditionType(condName),
					Status:             k8score.ConditionTrue,
					Reason:             "fakereason",
					Message:            "fakemsg",
					LastProbeTime:      now,
					LastTransitionTime: now,
				})
			}

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				// these conditions should be added
				for _, condName := range addCondList {
					cond := cmVM.GetCondition(objVM, virtv1.VirtualMachineConditionType(condName))
					Expect(cond).ToNot(BeNil())
					Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				}
				// these conditions shouldn't exist anymore
				for _, condName := range removeCondList {
					cond := cmVM.GetCondition(objVM, virtv1.VirtualMachineConditionType(condName))
					Expect(cond).To(BeNil())
				}
				// these conditsion should be updated
				for _, condName := range updateCondList {
					cond := cmVM.GetCondition(objVM, virtv1.VirtualMachineConditionType(condName))
					Expect(cond).ToNot(BeNil())
					Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				}
			}).Return(vm, nil)

			controller.Execute()
		})

		It("should add ready condition when VMI doesn't exists", func() {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, virtv1.VirtualMachineReady)
			addVirtualMachine(vm)

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachineReady)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
			}).Return(vm, nil)

			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)

			controller.Execute()
		})

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
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachinePaused)
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
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachinePaused)
				Expect(cond).To(BeNil())
			}).Return(vm, nil)

			controller.Execute()
		})

		It("should back off if a sync error occurs", func() {
			vm, vmi := DefaultVirtualMachine(false)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("some random failure"))

			vmInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
				Expect(cond).To(Not(BeNil()))
				Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(ContainSubstring("some random failure"))
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

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubevirt ignitiondata annotation from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"kubevirt.io/ignitiondata": "test"}
			annotations := map[string]string{"kubevirt.io/ignitiondata": "test"}

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubernetes annotations from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}
			annotations := map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(gomock.Any()).Do(func(obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		Context("VM printableStatus", func() {

			It("Should set a Stopped status when running=false and VMI doesn't exist", func() {
				vm, _ := DefaultVirtualMachine(false)
				addVirtualMachine(vm)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopped))
				})

				controller.Execute()
			})

			table.DescribeTable("should set a Stopped status when VMI exists but stopped", func(phase virtv1.VirtualMachineInstancePhase, deletionTimestamp *metav1.Time) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase
				vmi.Status.PhaseTransitionTimestamps = []virtv1.VirtualMachineInstancePhaseTransitionTimestamp{
					{
						Phase:                    virtv1.Running,
						PhaseTransitionTimestamp: metav1.Now(),
					},
				}
				vmi.ObjectMeta.DeletionTimestamp = deletionTimestamp

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopped))
				})

				shouldExpectVMIFinalizerRemoval(vmi)

				controller.Execute()
			},

				table.Entry("in Succeeded state", virtv1.Succeeded, nil),
				table.Entry("in Succeeded state with a deletionTimestamp", virtv1.Succeeded, &metav1.Time{Time: time.Now()}),
				table.Entry("in Failed state", virtv1.Failed, nil),
				table.Entry("in Failed state with a deletionTimestamp", virtv1.Failed, &metav1.Time{Time: time.Now()}),
			)

			It("Should set a Starting status when running=true and VMI doesn't exist", func() {
				vm, vmi := DefaultVirtualMachine(true)
				addVirtualMachine(vm)

				vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			})

			table.DescribeTable("Should set a Starting status when VMI is in a startup phase", func(phase virtv1.VirtualMachineInstancePhase) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			},

				table.Entry("VMI has no phase set", virtv1.VmPhaseUnset),
				table.Entry("VMI is in Pending phase", virtv1.Pending),
				table.Entry("VMI is in Scheduling phase", virtv1.Scheduling),
				table.Entry("VMI is in Scheduled phase", virtv1.Scheduled),
			)

			table.DescribeTable("Should set a CrashLoop status when VMI is deleted and VM is in crash loop backoff", func(status virtv1.VirtualMachineStatus, runStrategy virtv1.VirtualMachineRunStrategy, hasVMI bool, expectCrashloop bool) {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy
				vm.Status = status

				addVirtualMachine(vm)
				if hasVMI {
					vmi.Status.Phase = virtv1.Running
					vmiFeeder.Add(vmi)
				}

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					if expectCrashloop {
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusCrashLoopBackOff))
					} else {
						Expect(objVM.Status.PrintableStatus).ToNot(Equal(virtv1.VirtualMachineStatusCrashLoopBackOff))
					}
				})

				controller.Execute()
			},

				table.Entry("vm with runStrategy always and crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyAlways,
					false,
					true),
				table.Entry("vm with runStrategy rerun on failure and crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyRerunOnFailure,
					false,
					true),
				table.Entry("vm with runStrategy halt should not report crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyHalted,
					false,
					false),
				table.Entry("vm with runStrategy manual should not report crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyManual,
					false,
					false),
				table.Entry("vm with runStrategy always and VMI still exists should not report crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyAlways,
					true,
					false),
			)
			Context("VM with DataVolumes", func() {
				var vm *virtv1.VirtualMachine
				var vmi *virtv1.VirtualMachineInstance

				BeforeEach(func() {
					vm, vmi = DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
						Name: "test1",
						VolumeSource: virtv1.VolumeSource{
							DataVolume: &virtv1.DataVolumeSource{
								Name: "dv1",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv1",
							Namespace: vm.Namespace,
						},
					})
				})

				table.DescribeTable("Should set a Stopped/WaitingForVolumeBinding status when DataVolume exists but not bound", func(running bool, status virtv1.VirtualMachinePrintableStatus) {
					vm.Spec.Running = &running
					addVirtualMachine(vm)

					dv, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
					dataVolumeFeeder.Add(dv)

					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      dv.Name,
							Namespace: dv.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: k8score.ClaimPending,
						},
					}
					pvcInformer.GetStore().Add(&pvc)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(status))
					})

					controller.Execute()
				},

					table.Entry("Started VM", true, virtv1.VirtualMachineStatusWaitingForVolumeBinding),
					table.Entry("Stopped VM", false, virtv1.VirtualMachineStatusStopped),
				)

				table.DescribeTable("Should set a Provisioning status when DataVolume bound but not ready",
					func(dvPhase cdiv1.DataVolumePhase) {
						addVirtualMachine(vm)

						dv, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
						dv.Status.Phase = dvPhase
						dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
							Type:   cdiv1.DataVolumeBound,
							Status: k8score.ConditionTrue,
						})
						dataVolumeFeeder.Add(dv)

						if dvPhase == cdiv1.WaitForFirstConsumer {
							vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
						}
						vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
							objVM := obj.(*virtv1.VirtualMachine)
							Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusProvisioning))
						})

						controller.Execute()
					},

					table.Entry("DataVolume is in ImportScheduled phase", cdiv1.ImportScheduled),
					table.Entry("DataVolume is in ImportInProgress phase", cdiv1.ImportInProgress),
					table.Entry("DataVolume is in WaitForFirstConsumer phase", cdiv1.WaitForFirstConsumer),
				)

				table.DescribeTable("Should set a DataVolumeError status when DataVolume reports an error", func(dvFunc func(*cdiv1.DataVolume)) {
					addVirtualMachine(vm)

					dv, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
					dvFunc(dv)
					dataVolumeFeeder.Add(dv)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusDataVolumeError))
					})

					controller.Execute()
				},

					table.Entry(
						"DataVolume is in Failed phase",
						func(dv *cdiv1.DataVolume) {
							dv.Status.Phase = cdiv1.Failed
						},
					),
					table.Entry(
						"DataVolume Running condition is in error",
						func(dv *cdiv1.DataVolume) {
							dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
								Type:   cdiv1.DataVolumeRunning,
								Status: k8sv1.ConditionFalse,
								Reason: "Error",
							})
						},
					),
				)

				It("Should clear a DataVolumeError status when the DataVolume error is gone", func() {
					vm.Status.PrintableStatus = virtv1.VirtualMachineStatusDataVolumeError
					addVirtualMachine(vm)

					dv, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = cdiv1.CloneInProgress
					dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})
					dataVolumeFeeder.Add(dv)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusProvisioning))
					})

					controller.Execute()
				})

				It("Should set a Provisioning status when one DataVolume is ready and another isn't", func() {
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
						Name: "test2",
						VolumeSource: virtv1.VolumeSource{
							DataVolume: &virtv1.DataVolumeSource{
								Name: "dv2",
							},
						},
					})

					vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, virtv1.DataVolumeTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dv2",
							Namespace: vm.Namespace,
						},
					})

					addVirtualMachine(vm)

					dv1, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[0], vm)
					dv1.Status.Phase = cdiv1.Succeeded
					dv1.Status.Conditions = append(dv1.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})
					dv2, _ := createDataVolumeManifest(virtClient, &vm.Spec.DataVolumeTemplates[1], vm)
					dv2.Status.Phase = cdiv1.ImportInProgress
					dv2.Status.Conditions = append(dv2.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})

					dataVolumeFeeder.Add(dv1)
					dataVolumeFeeder.Add(dv2)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusProvisioning))
					})

					controller.Execute()
				})
			})

			Context("VM with PersistentVolumeClaims", func() {
				var vm *virtv1.VirtualMachine
				var vmi *virtv1.VirtualMachineInstance

				BeforeEach(func() {
					vm, vmi = DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
						Name: "test1",
						VolumeSource: virtv1.VolumeSource{
							PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc1",
							}},
						},
					})

					vmiInterface.EXPECT().Create(gomock.Any()).Times(1).Return(vmi, nil)

					addVirtualMachine(vm)
				})

				table.DescribeTable("Should set a WaitingForVolumeBinding status when PersistentVolumeClaim exists but unbound", func(pvcPhase k8sv1.PersistentVolumeClaimPhase) {
					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pvc1",
							Namespace: vm.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: pvcPhase,
						},
					}
					pvcInformer.GetStore().Add(&pvc)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusWaitingForVolumeBinding))
					})

					controller.Execute()
				},

					table.Entry("PersistentVolumeClaim is in Pending phase", k8sv1.ClaimPending),
					table.Entry("PersistentVolumeClaim is in Lost phase", k8sv1.ClaimLost),
				)

			})

			It("should set a Running status when VMI is running but not paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = virtv1.Running

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusRunning))
				})

				controller.Execute()
			})

			It("should set a Paused status when VMI is running but is paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = virtv1.Running
				vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				})

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusPaused))
				})

				controller.Execute()
			})

			table.DescribeTable("should set a Stopping status when VMI has a deletion timestamp set", func(phase virtv1.VirtualMachineInstancePhase, condType virtv1.VirtualMachineInstanceConditionType) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				vmi.Status.Phase = phase

				if condType != "" {
					vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
						Type:   condType,
						Status: k8sv1.ConditionTrue,
					})
				}
				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopping))
				})

				controller.Execute()
			},

				table.Entry("when VMI is pending", virtv1.Pending, virtv1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is provisioning", virtv1.Pending, virtv1.VirtualMachineInstanceProvisioning),
				table.Entry("when VMI is scheduling", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is scheduled", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is running", virtv1.Running, virtv1.VirtualMachineInstanceConditionType("")),
				table.Entry("when VMI is paused", virtv1.Running, virtv1.VirtualMachineInstancePaused),
			)

			Context("should set a Terminating status when VM has a deletion timestamp set", func() {
				table.DescribeTable("when VMI exists", func(phase virtv1.VirtualMachineInstancePhase, condType virtv1.VirtualMachineInstanceConditionType) {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = phase

					if condType != "" {
						vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
							Type:   condType,
							Status: k8sv1.ConditionTrue,
						})
					}
					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					table.Entry("when VMI is pending", virtv1.Pending, virtv1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is provisioning", virtv1.Pending, virtv1.VirtualMachineInstanceProvisioning),
					table.Entry("when VMI is scheduling", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is scheduled", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is running", virtv1.Running, virtv1.VirtualMachineInstanceConditionType("")),
					table.Entry("when VMI is paused", virtv1.Running, virtv1.VirtualMachineInstancePaused),
				)

				It("when VMI exists and has a deletion timestamp set", func() {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = virtv1.Running

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				})

				table.DescribeTable("when VMI does not exist", func(running bool) {
					vm, _ := DefaultVirtualMachine(running)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					table.Entry("with running: true", true),
					table.Entry("with running: false", false),
				)
			})

			It("should set a Migrating status when VMI is migrating", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = virtv1.Running
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					StartTimestamp: &metav1.Time{Time: time.Now()},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusMigrating))
				})

				controller.Execute()
			})

			It("should set an Unknown status when VMI is in unknown phase", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = virtv1.Unknown

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusUnknown))
				})

				controller.Execute()
			})

			table.DescribeTable("should set a failure status in accordance to VMI condition",
				func(status virtv1.VirtualMachinePrintableStatus, cond virtv1.VirtualMachineInstanceCondition) {

					vm, vmi := DefaultVirtualMachine(true)
					vmi.Status.Phase = virtv1.Scheduling
					vmi.Status.Conditions = append(vmi.Status.Conditions, cond)

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(status))
					})

					controller.Execute()
				},

				table.Entry("FailedUnschedulable", virtv1.VirtualMachineStatusUnschedulable,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
						Status: k8sv1.ConditionFalse,
						Reason: k8sv1.PodReasonUnschedulable,
					},
				),
				table.Entry("FailedPvcNotFound", virtv1.VirtualMachineStatusPvcNotFound,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: FailedPvcNotFoundReason,
					},
				),
				table.Entry("FailedDataVolumeNotFound", virtv1.VirtualMachineStatusDataVolumeNotFound,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: FailedDataVolumeNotFoundReason,
					},
				),
			)

			table.DescribeTable("should set an ImagePullBackOff/ErrPullImage statuses according to VMI Synchronized condition", func(reason string) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.Status.Phase = virtv1.Scheduling
				vmi.Status.Conditions = []virtv1.VirtualMachineInstanceCondition{
					{
						Type:   virtv1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: reason,
					},
				}

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachinePrintableStatus(reason)))
				})

				controller.Execute()
			},
				table.Entry("Reason: ErrImagePull", ErrImagePullReason),
				table.Entry("Reason: ImagePullBackOff", ImagePullBackOffReason),
			)
		})

		Context("Flavor", func() {
			BeforeEach(func() {
				flavorMethods.FindFlavorFunc = func(_ *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
					return &flavorv1alpha1.VirtualMachineFlavorProfile{
						CPU: &virtv1.CPU{
							Sockets: 2,
							Cores:   1,
							Threads: 1,
						},
					}, nil
				}
			})

			It("should apply flavor", func() {
				const flavorCpus = uint32(4)
				flavorMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) flavor.Conflicts {
					vmiSpec.Domain.CPU = &virtv1.CPU{Sockets: flavorCpus}
					return nil
				}

				vm, vmi := DefaultVirtualMachine(true)
				vm.Spec.Flavor = &virtv1.FlavorMatcher{
					Name: "test-flavor",
				}

				vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{Sockets: 2}

				addVirtualMachine(vm)

				vmiInterface.EXPECT().Create(gomock.Any()).Times(1).Do(func(arg interface{}) {
					vmiArg := arg.(*virtv1.VirtualMachineInstance)
					Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(flavorCpus))
				}).Return(vmi, nil)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1)

				controller.Execute()

				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			})

			It("should fail if flavor does not exist", func() {
				const errorMessage = "flavor not found"
				flavorMethods.FindFlavorFunc = func(_ *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
					return nil, fmt.Errorf(errorMessage)
				}

				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Flavor = &virtv1.FlavorMatcher{
					Name: "test-flavor",
				}

				addVirtualMachine(vm)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
					Expect(cond.Reason).To(Equal("FailedCreate"))
					Expect(cond.Message).To(ContainSubstring(errorMessage))
				}).Return(vm, nil)

				controller.Execute()

				testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
			})

			It("should fail applying flavor", func() {
				flavorMethods.ApplyToVmiFunc = func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, _ *virtv1.VirtualMachineInstanceSpec) flavor.Conflicts {
					return flavor.Conflicts{k8sfield.NewPath("spec", "template", "test", "path")}
				}

				vm, _ := DefaultVirtualMachine(true)
				vm.Spec.Flavor = &virtv1.FlavorMatcher{
					Name: "test-flavor",
				}

				addVirtualMachine(vm)

				vmInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
					Expect(cond).To(Not(BeNil()))
					Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
					Expect(cond.Reason).To(Equal("FailedCreate"))
					Expect(cond.Message).To(ContainSubstring("VMI conflicts with flavor"))
					Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
				}).Return(vm, nil)

				controller.Execute()

				testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
			})
		})
	})
})

func VirtualMachineFromVMI(name string, vmi *virtv1.VirtualMachineInstance, started bool) *virtv1.VirtualMachine {
	vm := &virtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: vmUID},
		Spec: virtv1.VirtualMachineSpec{
			Running: &started,
			Template: &virtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
		Status: virtv1.VirtualMachineStatus{
			Conditions: []virtv1.VirtualMachineCondition{
				{
					Type:   virtv1.VirtualMachineReady,
					Status: k8sv1.ConditionFalse,
					Reason: "VMINotExists",
				},
			},
		},
	}
	return vm
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) {
	vmi := api.NewMinimalVMI(vmiName)
	vmi.GenerateName = "prettyrandom"
	vmi.Status.Phase = virtv1.Running
	vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineControllerFinalizer)
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	virtcontroller.SetLatestApiVersionAnnotation(vmi)
	virtcontroller.SetLatestApiVersionAnnotation(vm)
	return vm, vmi
}

func DefaultVirtualMachine(started bool) (*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}
