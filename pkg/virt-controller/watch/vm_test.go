package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	k8score "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/api"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	kvpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
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
		var crSource *framework.FakeControllerSource
		var podInformer cache.SharedIndexInformer
		var instancetypeMethods *testutils.MockInstancetypeMethods
		var stop chan struct{}
		var controller *VMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmiFeeder *testutils.VirtualMachineFeeder
		var dataVolumeFeeder *testutils.DataVolumeFeeder
		var cdiClient *cdifake.Clientset
		var k8sClient *k8sfake.Clientset
		var virtClient *kubecli.MockKubevirtClient
		var config *virtconfig.ClusterConfig
		var kvInformer cache.SharedIndexInformer

		syncCaches := func(stop chan struct{}) {
			go vmiInformer.Run(stop)
			go vmInformer.Run(stop)
			go dataVolumeInformer.Run(stop)
			go crInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, vmInformer.HasSynced)).To(BeTrue())
		}

		asInt64Ptr := func(i int64) *int64 {
			return &i
		}

		asStrPtr := func(s string) *string {
			return &s
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			generatedInterface := fake.NewSimpleClientset()

			dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
			vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
			vmInformer, vmSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
			pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
			ns1 := &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			}
			ns2 := &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			}
			Expect(namespaceInformer.GetStore().Add(ns1)).To(Succeed())
			Expect(namespaceInformer.GetStore().Add(ns2)).To(Succeed())
			crInformer, crSource = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
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
			podInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})

			instancetypeMethods = testutils.NewMockInstancetypeMethods()

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			config, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

			controller, _ = NewVMController(vmiInformer,
				vmInformer,
				dataVolumeInformer,
				dataSourceInformer,
				namespaceInformer.GetStore(),
				pvcInformer,
				crInformer,
				podInformer,
				instancetypeMethods,
				recorder,
				virtClient,
				config)

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
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
			virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()
		})

		shouldExpectGracePeriodPatched := func(expectedGracePeriod int64, vmi *virtv1.VirtualMachineInstance) {
			patch := fmt.Sprintf(`{"spec":{"terminationGracePeriodSeconds": %d }}`, expectedGracePeriod)
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.MergePatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
		}

		shouldExpectVMIFinalizerRemoval := func(vmi *virtv1.VirtualMachineInstance) {
			patch := fmt.Sprintf(`[{ "op": "test", "path": "/metadata/finalizers", "value": ["%s"] }, { "op": "replace", "path": "/metadata/finalizers", "value": [] }]`, virtv1.VirtualMachineControllerFinalizer)

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
		}

		shouldExpectVMFinalizerAddition := func(vm *virtv1.VirtualMachine) {
			patch := fmt.Sprintf(`[{ "op": "test", "path": "/metadata/finalizers", "value": null }, { "op": "replace", "path": "/metadata/finalizers", "value": ["%s"] }]`, virtv1.VirtualMachineControllerFinalizer)

			vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vm, nil)
		}

		shouldExpectVMFinalizerRemoval := func(vm *virtv1.VirtualMachine) {
			patch := fmt.Sprintf(`[{ "op": "test", "path": "/metadata/finalizers", "value": ["%s"] }, { "op": "replace", "path": "/metadata/finalizers", "value": [] }]`, virtv1.VirtualMachineControllerFinalizer)

			vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vm, nil)
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

		shouldFailDataVolumeCreationNoResourceFound := func() {
			cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("the server could not find the requested resource (post datavolumes.cdi.kubevirt.io)")
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

				return createObj == vmRevision, created.GetObject(), nil
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

		createVMRevision := func(vm *virtv1.VirtualMachine, prefix string) *appsv1.ControllerRevision {
			return &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getVMRevisionName(vm.UID, vm.Generation, prefix),
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

		modifyVirtualMachine := func(vm *virtv1.VirtualMachine) {
			mockQueue.ExpectAdds(1)
			vmSource.Modify(vm)
			mockQueue.Wait()
		}

		It("should update conditions when failed creating DataVolume for virtualMachineInstance", func() {
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
			})
			addVirtualMachine(vm)

			shouldFailDataVolumeCreationNoResourceFound()

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
				Expect(cond).To(Not(BeNil()))
				Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(ContainSubstring("Error encountered while creating DataVolumes: failed to create DataVolume"))
				Expect(cond.Message).To(ContainSubstring("the server could not find the requested resource (post datavolumes.cdi.kubevirt.io)"))
			}).Return(vm, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, FailedDataVolumeCreateReason)
		})

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

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume.Namespace = "default"
			dataVolumeFeeder.Add(existingDataVolume)

			createCount := 0
			shouldExpectDataVolumeCreation(vm.UID, map[string]string{"kubevirt.io/created-by": string(vm.UID), "my": "label"}, map[string]string{"my": "annotation"}, &createCount)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusProvisioning))
			})

			controller.Execute()
			Expect(createCount).To(Equal(1))
			testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
		})

		DescribeTable("should hotplug a vm", func(isRunning bool) {

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
				vmiInterface.EXPECT().AddVolume(context.Background(), vmi.ObjectMeta.Name, vm.Status.VolumeRequests[0].AddVolumeOptions)
			}

			vmInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes[0].Name).To(Equal("vol1"))
			}).Return(vm, nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change
				Expect(arg.(*virtv1.VirtualMachine).Status.VolumeRequests).To(HaveLen(1))
			}).Return(vm, nil)

			controller.Execute()
		},

			Entry("that is running", true),
			Entry("that is not running", false),
		)

		DescribeTable("should unhotplug a vm", func(isRunning bool) {
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
				vmiInterface.EXPECT().RemoveVolume(context.Background(), vmi.ObjectMeta.Name, vm.Status.VolumeRequests[0].RemoveVolumeOptions)
			}

			vmInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes).To(BeEmpty())
			}).Return(vm, nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				// vol request shouldn't be cleared until update status observes the new volume change occured
				Expect(arg.(*virtv1.VirtualMachine).Status.VolumeRequests).To(HaveLen(1))
			}).Return(vm, nil)

			controller.Execute()
		},

			Entry("that is running", true),
			Entry("that is not running", false),
		)

		DescribeTable("should clear VolumeRequests for added volumes that are satisfied", func(isRunning bool) {
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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.VolumeRequests).To(BeEmpty())
			}).Return(nil, nil)

			controller.Execute()
		},

			Entry("that is running", true),
			Entry("that is not running", false),
		)

		DescribeTable("should clear VolumeRequests for removed volumes that are satisfied", func(isRunning bool) {
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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.VolumeRequests).To(BeEmpty())
			}).Return(nil, nil)

			controller.Execute()
		},

			Entry("that is running", true),
			Entry("that is not running", false),
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

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			deletionCount := 0
			shouldExpectDataVolumeDeletion(vm.UID, &deletionCount)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

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

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

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

			existingDataVolume1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			existingDataVolume1.Namespace = "default"
			existingDataVolume1.Status.Phase = cdiv1.Failed
			// explicitly delete the annotations field
			existingDataVolume1.Annotations = nil

			existingDataVolume2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
			existingDataVolume2.Namespace = "default"
			existingDataVolume2.Status.Phase = cdiv1.Succeeded
			existingDataVolume2.Annotations = nil

			dataVolumeFeeder.Add(existingDataVolume1)
			dataVolumeFeeder.Add(existingDataVolume2)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

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

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should start VMI once DataVolumes (not templates) are complete", func() {

			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: "test1",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "dv1",
					},
				},
			})
			dvt := virtv1.DataVolumeTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dv1",
				},
			}

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, dvt, vm)

			existingDataVolume.OwnerReferences = nil
			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)

			// expect creation called
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
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

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.WaitForFirstConsumer
			addVirtualMachine(vm)
			dataVolumeFeeder.Add(existingDataVolume)
			// expect creation called
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)
			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
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

			existingDataVolume, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)

			existingDataVolume.Namespace = "default"
			existingDataVolume.Status.Phase = cdiv1.Succeeded
			addVirtualMachine(vm)

			dataVolumeFeeder.Add(existingDataVolume)
			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)
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

		DescribeTable("should properly set priority class", func(dvPriorityClass, vmPriorityClass, expectedPriorityClass string) {
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
			Entry("when dv priorityclass is not defined and VM priorityclass is defined", "", "vmpriority", "vmpriority"),
			Entry("when dv priorityclass is defined and VM priorityclass is defined", "dvpriority", "vmpriority", "dvpriority"),
			Entry("when dv priorityclass is defined and VM priorityclass is not defined", "dvpriority", "", "dvpriority"),
			Entry("when dv priorityclass is not defined and VM priorityclass is not defined", "", "", ""),
		)

		Context("crashloop backoff tests", func() {

			It("should track start failures when VMIs fail without hitting running state", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.UID = "123"
				vmi.Status.Phase = virtv1.Failed

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
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

				vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.RetryAfterTimestamp).ToNot(BeNil())
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure.RetryAfterTimestamp.Time).ToNot(Equal(oldRetry))
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

				vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).To(BeNil())
				}).Return(nil, nil)

				controller.Execute()

			})

			DescribeTable("should clear existing start failures when runStrategy is halted or manual", func(runStrategy virtv1.VirtualMachineRunStrategy) {
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

				vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
					if runStrategy == virtv1.RunStrategyHalted || runStrategy == virtv1.RunStrategyManual {
						Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).To(BeNil())
					} else {
						Expect(arg.(*virtv1.VirtualMachine).Status.StartFailure).ToNot(BeNil())

					}
				}).Return(nil, nil)

				if runStrategy == virtv1.RunStrategyRerunOnFailure {
					vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).Do(
						func(ctx context.Context, name string, patchType types.PatchType, body []byte, opts *metav1.PatchOptions) {
							Expect(string(body)).To(ContainSubstring(`"action":"Start"`))
						}).Return(vm, nil).Times(2)
				}

				shouldExpectVMIFinalizerRemoval(vmi)

				controller.Execute()

				if runStrategy != virtv1.RunStrategyManual && runStrategy != virtv1.RunStrategyOnce {
					testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
				}
			},

				Entry("runStrategyHalted", virtv1.RunStrategyHalted),
				Entry("always", virtv1.RunStrategyAlways),
				Entry("manual", virtv1.RunStrategyManual),
				Entry("rerunOnFailure", virtv1.RunStrategyRerunOnFailure),
				Entry("once", virtv1.RunStrategyOnce),
			)

			DescribeTable("should calculated expected backoff delay", func(failCount, minExpectedDelay int, maxExpectedDelay int) {

				for i := 0; i < 1000; i++ {
					delay := calculateStartBackoffTime(failCount, defaultMaxCrashLoopBackoffDelaySeconds)

					// check that minExpectedDelay <= delay <= maxExpectedDelay
					Expect(delay).To(And(BeNumerically(">=", minExpectedDelay), BeNumerically("<=", maxExpectedDelay)))
				}
			},

				Entry("failCount 0", 0, 10, 15),
				Entry("failCount 1", 1, 10, 15),
				Entry("failCount 2", 2, 40, 60),
				Entry("failCount 3", 3, 90, 135),
				Entry("failCount 4", 4, 160, 240),
				Entry("failCount 5", 5, 250, 300),
				Entry("failCount 6", 6, 300, 300),
			)

			DescribeTable("has start failure backoff expired", func(vmFunc func() *virtv1.VirtualMachine, expected int64) {
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

				Entry("no vm start failures",
					func() *virtv1.VirtualMachine {
						return &virtv1.VirtualMachine{}
					},
					int64(0)),
				Entry("vm failure waiting 300 seconds",
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
				Entry("vm failure 300 seconds past retry time",
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

			DescribeTable("create clone DataVolume for VirtualMachineInstance", func(dv *virtv1.DataVolumeTemplateSpec, saVol *virtv1.Volume, ds *cdiv1.DataSource, fail, sourcePVC bool) {
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
					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)
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

				// Add source PVC to cache
				if sourcePVC {
					pvc := k8sv1.PersistentVolumeClaim{}
					if dv.Spec.Source != nil {
						pvc.Name = dv.Spec.Source.PVC.Name
						pvc.Namespace = dv.Spec.Source.PVC.Namespace
					} else {
						pvc.Name = ds.Spec.Source.PVC.Name
						pvc.Namespace = ds.Spec.Source.PVC.Namespace
					}
					if pvc.Namespace == "" {
						pvc.Namespace = vm.Namespace
					}
					Expect(pvcInformer.GetStore().Add(&pvc)).To(Succeed())
				}

				controller.cloneAuthFunc = func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error) {
					k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						return true, &authorizationv1.SubjectAccessReview{
							Status: authorizationv1.SubjectAccessReviewStatus{
								Allowed: true,
							},
						}, nil
					})
					response, err := dv.AuthorizeSA(requestNamespace, requestName, proxy, saNamespace, saName)
					Expect(err).ToNot(HaveOccurred())
					if dv.Spec.Source != nil {
						if dv.Spec.Source.PVC.Namespace != "" {
							Expect(response.Handler.SourceNamespace).Should(Equal(dv.Spec.Source.PVC.Namespace))
						} else {
							Expect(response.Handler.SourceNamespace).Should(Equal(vm.Namespace))
						}

						Expect(response.Handler.SourceName).Should(Equal(dv.Spec.Source.PVC.Name))
					} else {
						Expect(response.Handler.SourceNamespace).Should(Equal(ds.Spec.Source.PVC.Namespace))
						Expect(response.Handler.SourceName).Should(Equal(ds.Spec.Source.PVC.Name))
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
					if !sourcePVC {
						testutils.ExpectEvent(recorder, SourcePVCNotAvailabe)
					}
					Expect(createCount).To(Equal(1))
					testutils.ExpectEvent(recorder, SuccessfulDataVolumeCreateReason)
				}
			},
				Entry("with auth, source PVC and source namespace defined", dv1, serviceAccountVol, nil, false, true),
				Entry("with auth and no source namespace defined", dv2, serviceAccountVol, nil, false, true),
				Entry("with auth and source namespace no serviceaccount defined", dv1, nil, nil, false, true),
				Entry("with no auth and source namespace defined", dv1, serviceAccountVol, nil, true, true),
				Entry("with auth, source PVC, datasource and source namespace defined", dv3, serviceAccountVol, ds, false, true),
				Entry("with auth, datasource and source namespace but no source PVC", dv3, serviceAccountVol, ds, false, false),
			)
		})

		It("should create VMI with vmRevision", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Generation = 1

			addVirtualMachine(vm)

			vmRevisionStart := createVMRevision(vm, revisionPrefixStart)
			vmRevisionLastSeen := createVMRevision(vm, revisionPrefixLastSeen)
			expectControllerRevisionCreation(vmRevisionStart)
			expectControllerRevisionCreation(vmRevisionLastSeen)
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.VirtualMachineRevisionName).To(Equal(vmRevisionStart.Name))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete older vmRevision and create VMI with new one", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Generation = 1
			oldVMRevision := createVMRevision(vm, revisionPrefixStart)

			vm.Generation = 2
			addVirtualMachine(vm)
			vmRevision := createVMRevision(vm, revisionPrefixStart)

			expectControllerRevisionList(oldVMRevision)
			expectControllerRevisionDelete(oldVMRevision)
			expectControllerRevisionCreation(vmRevision)
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.VirtualMachineRevisionName).To(Equal(vmRevision.Name))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		Context("VM generation tests", func() {

			DescribeTable("should add the generation annotation onto the VMI", func(startingAnnotations map[string]string, endAnnotations map[string]string) {
				_, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = startingAnnotations

				annotations := endAnnotations
				setGenerationAnnotationOnVmi(6, vmi)
				Expect(vmi.ObjectMeta.Annotations).To(Equal(annotations))
			},
				Entry("with previous annotations", map[string]string{"test": "test"}, map[string]string{"test": "test", virtv1.VirtualMachineGenerationAnnotation: "6"}),
				Entry("without previous annotations", map[string]string{}, map[string]string{virtv1.VirtualMachineGenerationAnnotation: "6"}),
			)

			DescribeTable("should add generation annotation during VMI creation", func(runStrategy virtv1.VirtualMachineRunStrategy) {
				vm, vmi := DefaultVirtualMachine(true)

				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy
				vm.Generation = 3
				addVirtualMachine(vm)

				annotations := map[string]string{virtv1.VirtualMachineGenerationAnnotation: "3"}
				vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
					Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
				}).Return(vmi, nil)

				// expect update status is called
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
					Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
				}).Return(nil, nil)

				if runStrategy == virtv1.RunStrategyRerunOnFailure {
					vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).Do(
						func(ctx context.Context, name string, patchType types.PatchType, body []byte, opts *metav1.PatchOptions) {
							Expect(string(body)).To(ContainSubstring(`"action":"Start"`))
						}).Return(vm, nil).Times(1)
				}

				controller.Execute()

				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			},

				Entry("with run strategy Always", virtv1.RunStrategyAlways),
				Entry("with run strategy Once", virtv1.RunStrategyOnce),
				Entry("with run strategy RerunOnFailure", virtv1.RunStrategyRerunOnFailure),
			)

			It("should patch the generation annotation onto the vmi", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = map[string]string{}
				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				patch := `[{ "op": "test", "path": "/metadata/annotations", "value": {} }, { "op": "replace", "path": "/metadata/annotations", "value": {"kubevirt.io/vm-generation":"4"} }]`
				vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)

				err := controller.patchVmGenerationAnnotationOnVmi(4, vmi)
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("should get the generation annotation from the vmi", func(annotations map[string]string, desiredGeneration *string, desiredErr error) {
				_, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = annotations

				gen, err := getGenerationAnnotation(vmi)
				if desiredGeneration == nil {
					Expect(gen).To(BeNil())
				} else {
					Expect(gen).To(Equal(desiredGeneration))
				}
				if desiredErr == nil {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(Equal(desiredErr))
				}
			},
				Entry("with only one entry in the annotations", map[string]string{virtv1.VirtualMachineGenerationAnnotation: "6"}, asStrPtr("6"), nil),
				Entry("with multiple entries in the annotations", map[string]string{"test": "test", virtv1.VirtualMachineGenerationAnnotation: "5"}, asStrPtr("5"), nil),
				Entry("with no generation annotation existing", map[string]string{"test": "testing"}, nil, nil),
				Entry("with empty annotations map", map[string]string{}, nil, nil),
			)

			DescribeTable("should parse generation from vm controller revision name", func(name string, desiredGeneration *int64) {

				gen := parseGeneration(name, log.DefaultLogger())
				if desiredGeneration == nil {
					Expect(gen).To(BeNil())
				} else {
					Expect(gen).To(Equal(desiredGeneration))
				}
			},
				Entry("with standard name", getVMRevisionName("9160e5de-2540-476a-86d9-af0081aee68a", 3, revisionPrefixStart), asInt64Ptr(3)),
				Entry("with one dash in name", getVMRevisionName("abcdef", 5, revisionPrefixStart), asInt64Ptr(5)),
				Entry("with no dash in name", "12345", nil),
				Entry("with ill formatted generation", "123-456-2b3b", nil),
			)

			Context("conditionally bump generation tests", func() {
				// Needed for the default values in each Entry(..)
				vm, vmi := DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = DefaultVirtualMachine(true)
				})

				DescribeTable("should conditionally bump the generation annotation on the vmi", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmSpec virtv1.VirtualMachineSpec, newVMSpec virtv1.VirtualMachineSpec, vmGeneration int64, desiredErr error, expectPatch bool) {
					// Spec and generation for the vmRevision and 'old' objects
					vmi.ObjectMeta.Annotations = initialAnnotations
					vm.Generation = 1
					vm.Spec = revisionVmSpec

					crName, err := controller.createVMRevision(vm, revisionPrefixStart)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), crName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmi.Status.VirtualMachineRevisionName = crName

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					// This is the 'updated' details on the vm
					vm.Generation = vmGeneration
					vm.Spec = newVMSpec

					if expectPatch {
						oldAnnotations, err := json.Marshal(initialAnnotations)
						Expect(err).ToNot(HaveOccurred())
						newAnnotations, err := json.Marshal(desiredAnnotations)
						Expect(err).ToNot(HaveOccurred())
						var ops []string
						ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s }`, string(oldAnnotations)))
						ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s }`, string(newAnnotations)))

						vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte("["+strings.Join(ops, ", ")+"]"), &metav1.PatchOptions{}).Return(vmi, nil)
					} else {
						// Should not be called
						vmiInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
					}

					err = controller.conditionallyBumpGenerationAnnotationOnVmi(vm, vmi)
					if desiredErr == nil {
						Expect(err).ToNot(HaveOccurred())
					} else {
						Expect(err).To(Equal(desiredErr))
					}
				},
					Entry(
						"with generation and template staying the same",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vm.ObjectMeta.Name,
									Labels: vm.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vm.ObjectMeta.Name,
									Labels: vm.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(2),
						nil,
						false, // Expect no patch
					),
					Entry(
						"with generation increasing and a change in template",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 3,
										},
									},
								},
							},
						},
						int64(3),
						nil,
						false, // No patch because template has changed
					),
					Entry(
						"with generation increasing and no change in template",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "3"},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(3),
						nil,
						true, // Patch since there is no change and we can bump
					),
					Entry(
						"with generation increasing, no change in template, and run strategy changing",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "7"},
						virtv1.VirtualMachineSpec{
							RunStrategy: func(rs virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachineRunStrategy { return &rs }(virtv1.RunStrategyAlways),
							Running:     func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							RunStrategy: func(rs virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachineRunStrategy { return &rs }(virtv1.RunStrategyRerunOnFailure),
							Running:     func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4,
										},
									},
								},
							},
						},
						int64(7),
						nil,
						true, // Patch since only template matters, not run strategy
					),
				)
			})

			DescribeTable("should sync the generation info", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmGeneration int64, vmGeneration int64, desiredErr error, expectPatch bool, desiredObservedGeneration int64, desiredDesiredGeneration int64) {
				vm, vmi := DefaultVirtualMachine(true)
				vmi.ObjectMeta.Annotations = initialAnnotations
				vm.Generation = revisionVmGeneration

				crName, err := controller.createVMRevision(vm, revisionPrefixStart)
				Expect(err).ToNot(HaveOccurred())

				vmi.Status.VirtualMachineRevisionName = crName

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vm.Generation = vmGeneration

				if expectPatch {
					var ops []string
					oldAnnotations, err := json.Marshal(initialAnnotations)
					Expect(err).ToNot(HaveOccurred())
					newAnnotations, err := json.Marshal(desiredAnnotations)
					Expect(err).ToNot(HaveOccurred())
					ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s }`, string(oldAnnotations)))
					ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s }`, string(newAnnotations)))

					vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte("["+strings.Join(ops, ", ")+"]"), &metav1.PatchOptions{}).Times(1).Return(vmi, nil)
				} else {
					// Should not be called
					vmiInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				}

				err = controller.syncGenerationInfo(vm, vmi, log.DefaultLogger())
				if desiredErr == nil {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(Equal(desiredErr))
				}

				Expect(vm.Status.ObservedGeneration).To(Equal(desiredObservedGeneration))
				Expect(vm.Status.DesiredGeneration).To(Equal(desiredDesiredGeneration))
			},
				Entry(
					"with annotation existing - generation updates",
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
					int64(2),
					int64(3),
					nil,
					false,
					int64(2),
					int64(3),
				),
				Entry(
					"with annotation existing - generation does not change",
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
					int64(2),
					int64(2),
					nil,
					false,
					int64(2),
					int64(2),
				),
				Entry(
					// In this case the annotation should be back filled from the revision
					"with annotation existing - ill formatted generation annotation",
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2b3c"},
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "3"},
					int64(3),
					int64(3),
					nil,
					true,
					int64(3),
					int64(3),
				),
				Entry(
					"with annotation not existing - generation updates and patches vmi",
					map[string]string{},
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "3"},
					int64(3),
					int64(4),
					nil,
					true,
					int64(3),
					int64(4),
				),
				Entry(
					"with annotation not existing - generation does not update and patches vmi",
					map[string]string{},
					map[string]string{virtv1.VirtualMachineGenerationAnnotation: "7"},
					int64(7),
					int64(7),
					nil,
					true,
					int64(7),
					int64(7),
				),
			)

			Context("generation tests with Execute()", func() {
				// Needed for the default values in each Entry(..)
				vm, vmi := DefaultVirtualMachine(true)

				BeforeEach(func() {
					// Reset every time
					vm, vmi = DefaultVirtualMachine(true)
				})

				DescribeTable("should update annotations and sync during Execute()", func(initialAnnotations map[string]string, desiredAnnotations map[string]string, revisionVmSpec virtv1.VirtualMachineSpec, newVMSpec virtv1.VirtualMachineSpec, revisionVmGeneration int64, vmGeneration int64, desiredErr error, expectPatch bool, desiredObservedGeneration int64, desiredDesiredGeneration int64) {
					vmi.ObjectMeta.Annotations = initialAnnotations
					vm.Generation = revisionVmGeneration
					vm.Spec = revisionVmSpec

					crName, err := controller.createVMRevision(vm, revisionPrefixStart)
					Expect(err).ToNot(HaveOccurred())

					vmi.Status.VirtualMachineRevisionName = crName

					vm.Generation = vmGeneration
					vm.Spec = newVMSpec
					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					if expectPatch {
						var ops []string
						oldAnnotations, err := json.Marshal(initialAnnotations)
						Expect(err).ToNot(HaveOccurred())
						newAnnotations, err := json.Marshal(desiredAnnotations)
						Expect(err).ToNot(HaveOccurred())
						ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s }`, string(oldAnnotations)))
						ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s }`, string(newAnnotations)))

						vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte("["+strings.Join(ops, ", ")+"]"), &metav1.PatchOptions{}).Times(1).Return(vmi, nil)
					} else {
						// Should not be called
						vmiInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
					}

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
						Expect(arg.(*virtv1.VirtualMachine).Status.ObservedGeneration).To(Equal(desiredObservedGeneration))
						Expect(arg.(*virtv1.VirtualMachine).Status.DesiredGeneration).To(Equal(desiredDesiredGeneration))
					}).Return(nil, nil)

					controller.Execute()
				},
					Entry(
						// Expect no patch on vmi annotations, and vm status to be correct
						"with annotation existing, new changes in VM spec",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 4, // changed
										},
									},
								},
							},
						},
						int64(2),
						int64(3),
						nil,
						false,
						int64(2),
						int64(3),
					),
					Entry(
						// Expect a patch on vmi annotations, and vm status to be correct
						"with annotation existing, no new changes in VM spec",
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "2"},
						map[string]string{virtv1.VirtualMachineGenerationAnnotation: "3"},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						virtv1.VirtualMachineSpec{
							Running: func(b bool) *bool { return &b }(true),
							Template: &virtv1.VirtualMachineInstanceTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:   vmi.ObjectMeta.Name,
									Labels: vmi.ObjectMeta.Labels,
								},
								Spec: virtv1.VirtualMachineInstanceSpec{
									Domain: v1.DomainSpec{
										CPU: &virtv1.CPU{
											Cores: 2,
										},
									},
								},
							},
						},
						int64(2),
						int64(3),
						nil,
						true,
						int64(3),
						int64(3),
					),
				)
			})
		})

		DescribeTable("should create missing VirtualMachineInstance", func(runStrategy virtv1.VirtualMachineRunStrategy) {
			vm, vmi := DefaultVirtualMachine(true)

			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvmi"))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			if runStrategy == virtv1.RunStrategyRerunOnFailure {
				vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).Do(
					func(ctx context.Context, name string, patchType types.PatchType, body []byte, opts *metav1.PatchOptions) {
						Expect(string(body)).To(ContainSubstring(`"action":"Start"`))
					}).Return(vm, nil).Times(1)
			}

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		},

			Entry("with run strategy Always", virtv1.RunStrategyAlways),
			Entry("with run strategy Once", virtv1.RunStrategyOnce),
			Entry("with run strategy RerunOnFailure", virtv1.RunStrategyRerunOnFailure),
		)

		It("should ignore the name of a VirtualMachineInstance templates", func() {
			vm, vmi := DefaultVirtualMachineWithNames(true, "vmname", "vminame")

			addVirtualMachine(vm)

			// expect creation called
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("vmname"))
				Expect(arg.(*virtv1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vmi, nil)

			// expect update status is called
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
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
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
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
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
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
			uid := uuid.NewString()
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
			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should add controller finalizer if VirtualMachine does not have it", func() {
			vm, _ := DefaultVirtualMachine(false)
			vm.Finalizers = nil

			addVirtualMachine(vm)

			shouldExpectVMFinalizerAddition(vm)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()
		})

		It("should add controller finalizer only once", func() {
			//DefaultVirtualMachine already set finalizer
			vm, _ := DefaultVirtualMachine(false)
			Expect(vm.Finalizers).To(HaveLen(1))
			Expect(vm.Finalizers[0]).To(BeEquivalentTo(virtv1.VirtualMachineControllerFinalizer))
			addVirtualMachine(vm)

			//Expect only update status, not Patch on vmInterface
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()
		})

		It("should delete VirtualMachineInstance when VirtualMachine marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.DeletionTimestamp = now()
			vm.DeletionGracePeriodSeconds = kvpointer.P(v1.DefaultGracePeriodSeconds)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Return(nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)
			shouldExpectGracePeriodPatched(v1.DefaultGracePeriodSeconds, vmi)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should remove controller finalizer once VirtualMachineInstance is gone", func() {
			//DefaultVirtualMachine already set finalizer
			vm, _ := DefaultVirtualMachine(true)
			vm.DeletionTimestamp = now()

			addVirtualMachine(vm)

			shouldExpectVMFinalizerRemoval(vm)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()
		})

		DescribeTable("should not delete VirtualMachineInstance when vmi failed", func(runStrategy virtv1.VirtualMachineRunStrategy) {
			vm, vmi := DefaultVirtualMachine(true)

			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			vmi.Status.Phase = virtv1.Failed

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			shouldExpectVMIFinalizerRemoval(vmi)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()

		},

			Entry("with run strategy Once", virtv1.RunStrategyOnce),
			Entry("with run strategy Manual", virtv1.RunStrategyManual),
		)

		It("should not delete the VirtualMachineInstance again if it is already marked for deletion", func() {
			vm, vmi := DefaultVirtualMachine(false)
			vmi.DeletionTimestamp = now()

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

			controller.Execute()
		})

		It("should ignore non-matching VMIs", func() {
			vm, vmi := DefaultVirtualMachine(true)

			nonMatchingVMI := api.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addVirtualMachine(vm)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			vmiSource.Add(nonMatchingVMI)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(2).Return(vm, nil).AnyTimes()

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().Get(context.Background(), vm.ObjectMeta.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Return(vm, nil)
			vmiInterface.EXPECT().Patch(context.Background(), vmi.ObjectMeta.Name, gomock.Any(), gomock.Any(), &metav1.PatchOptions{})

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

			dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
			dv.Status.Phase = cdiv1.Succeeded

			orphanDV := dv.DeepCopy()
			orphanDV.ObjectMeta.OwnerReferences = nil
			Expect(dataVolumeInformer.GetStore().Add(orphanDV)).To(Succeed())

			cdiClient.Fake.PrependReactor("patch", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patch, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(patch.GetName()).To(Equal(dv.Name))
				Expect(patch.GetNamespace()).To(Equal(dv.Namespace))
				Expect(string(patch.GetPatch())).To(ContainSubstring(string(vm.UID)))
				Expect(string(patch.GetPatch())).To(ContainSubstring("ownerReferences"))
				return true, dv, nil
			})

			vmInterface.EXPECT().Get(context.Background(), vm.ObjectMeta.Name, gomock.Any()).Return(vm, nil)
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Return(vm, nil)

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Return(vm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			vm, vmi := DefaultVirtualMachine(true)

			addVirtualMachine(vm)
			// vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, fmt.Errorf("some random failure"))

			// We should see the failed condition, replicas should stay at 0
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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

			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("some random failure"))

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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

		DescribeTable("should add ready condition when VMI exists", func(setup func(vmi *virtv1.VirtualMachineInstance), status k8sv1.ConditionStatus) {
			vm, vmi := DefaultVirtualMachine(true)
			virtcontroller.NewVirtualMachineConditionManager().RemoveCondition(vm, virtv1.VirtualMachineReady)
			addVirtualMachine(vm)

			setup(vmi)
			vmiFeeder.Add(vmi)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachineReady)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(status))
			}).Return(vm, nil)

			controller.Execute()
		},
			Entry("VMI Ready condition is True", markAsReady, k8sv1.ConditionTrue),
			Entry("VMI Ready condition is False", markAsNonReady, k8sv1.ConditionFalse),
			Entry("VMI Ready condition doesn't exist", unmarkReady, k8sv1.ConditionFalse),
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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
				objVM := obj.(*virtv1.VirtualMachine)
				cond := virtcontroller.NewVirtualMachineConditionManager().
					GetCondition(objVM, virtv1.VirtualMachineReady)
				Expect(cond).ToNot(BeNil())
				Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
			}).Return(vm, nil)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)

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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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

			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("some random failure"))

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
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
			annotations := map[string]string{"test": "test", virtv1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubevirt ignitiondata annotation from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"kubevirt.io/ignitiondata": "test"}
			annotations := map[string]string{"kubevirt.io/ignitiondata": "test", virtv1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should copy kubernetes annotations from spec.template to vmi", func() {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true"}
			annotations := map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "true", virtv1.VirtualMachineGenerationAnnotation: "0"}

			vm.Status.PrintableStatus = virtv1.VirtualMachineStatusStarting
			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, obj interface{}) {
				Expect(obj.(*virtv1.VirtualMachineInstance).ObjectMeta.Annotations).To(Equal(annotations))
			}).Return(vmi, nil)

			controller.Execute()
		})

		Context("VM memory dump", func() {
			const (
				testPVCName    = "testPVC"
				targetFileName = "memory.dump"
			)

			shouldExpectVMIVolumesAddPatched := func(vmi *virtv1.VirtualMachineInstance) {
				test := `{ "op": "test", "path": "/spec/volumes", "value": null}`
				update := `{ "op": "add", "path": "/spec/volumes", "value": [{"name":"testPVC","memoryDump":{"claimName":"testPVC","hotpluggable":true}}]}`
				patch := fmt.Sprintf("[%s, %s]", test, update)

				vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
			}

			shouldExpectVMIVolumesRemovePatched := func(vmi *virtv1.VirtualMachineInstance) {
				test := `{ "op": "test", "path": "/spec/volumes", "value": [{"name":"testPVC","memoryDump":{"claimName":"testPVC","hotpluggable":true}}]}`
				update := `{ "op": "replace", "path": "/spec/volumes", "value": []}`
				patch := fmt.Sprintf("[%s, %s]", test, update)
				fmt.Println(patch)

				vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
			}

			applyVMIMemoryDumpVol := func(spec *virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstanceSpec {
				newVolume := virtv1.Volume{
					Name: testPVCName,
					VolumeSource: virtv1.VolumeSource{
						MemoryDump: &virtv1.MemoryDumpVolumeSource{
							PersistentVolumeClaimVolumeSource: virtv1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
									ClaimName: testPVCName,
								},
								Hotpluggable: true,
							},
						},
					},
				}

				spec.Volumes = append(spec.Volumes, newVolume)

				return spec
			}

			expectPVCAnnotationUpdate := func(expectedAnnotation string, pvcAnnotationUpdated chan bool) {
				virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
				k8sClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
					update, ok := action.(testing.UpdateAction)
					Expect(ok).To(BeTrue())

					pvc, ok := update.GetObject().(*k8sv1.PersistentVolumeClaim)
					Expect(ok).To(BeTrue())
					Expect(pvc.Name).To(Equal(testPVCName))
					Expect(pvc.Annotations[virtv1.PVCMemoryDumpAnnotation]).To(Equal(expectedAnnotation))
					pvcAnnotationUpdated <- true

					return true, nil, nil
				})
			}

			It("should add memory dump volume and update vmi volumes", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpAssociating,
				}

				addVirtualMachine(vm)

				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				shouldExpectVMIVolumesAddPatched(vmi)

				vmInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes[0].Name).To(Equal(testPVCName))
				}).Return(vm, nil)
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

				controller.Execute()
			})

			It("should update memory dump phase to InProgress when memory dump in vm volumes", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpAssociating,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)
				vmi.Spec = vm.Spec.Template.Spec
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				// when the memory dump volume is in the vm volume list we should change status to in progress
				updatedMemoryDump := &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpInProgress,
				}

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
				}).Return(nil, nil)

				controller.Execute()
			})

			It("should change status to unmounting when memory dump timestamp updated", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpInProgress,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)
				vmi.Spec = vm.Spec.Template.Spec
				now := metav1.Now()
				vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
					{
						Name:  testPVCName,
						Phase: virtv1.MemoryDumpVolumeCompleted,
						MemoryDumpVolume: &virtv1.DomainMemoryDumpInfo{
							StartTimestamp: &now,
							EndTimestamp:   &now,
							ClaimName:      testPVCName,
							TargetFileName: targetFileName,
						},
					},
				}
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				updatedMemoryDump := &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName:      testPVCName,
					Phase:          virtv1.MemoryDumpUnmounting,
					EndTimestamp:   &now,
					StartTimestamp: &now,
					FileName:       &vmi.Status.VolumeStatus[0].MemoryDumpVolume.TargetFileName,
				}

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
				}).Return(nil, nil)

				controller.Execute()
			})

			It("should update status to failed when memory dump failed", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpInProgress,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)
				vmi.Spec = vm.Spec.Template.Spec
				now := metav1.Now()
				vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
					{
						Name:    testPVCName,
						Phase:   virtv1.MemoryDumpVolumeFailed,
						Message: "Memory dump failed",
						MemoryDumpVolume: &virtv1.DomainMemoryDumpInfo{
							ClaimName:    testPVCName,
							EndTimestamp: &now,
						},
					},
				}
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				updatedMemoryDump := &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        virtv1.MemoryDumpFailed,
					Message:      vmi.Status.VolumeStatus[0].Message,
					EndTimestamp: &now,
				}

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
				}).Return(nil, nil)

				controller.Execute()
			})

			DescribeTable("should remove memory dump volume from vmi volumes and update pvc annotation", func(phase virtv1.MemoryDumpPhase, expectedAnnotation string) {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     phase,
				}
				if phase != virtv1.MemoryDumpFailed {
					fileName := targetFileName
					vm.Status.MemoryDumpRequest.FileName = &fileName
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)
				vmi.Spec = vm.Spec.Template.Spec
				vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
					{
						Name: testPVCName,
						MemoryDumpVolume: &virtv1.DomainMemoryDumpInfo{
							ClaimName: testPVCName,
						},
					},
				}
				markAsReady(vmi)
				vmiFeeder.Add(vmi)
				pvc := k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testPVCName,
						Namespace: vm.Namespace,
					},
				}
				Expect(pvcInformer.GetStore().Add(&pvc)).To(Succeed())

				pvcAnnotationUpdated := make(chan bool, 1)
				defer close(pvcAnnotationUpdated)
				expectPVCAnnotationUpdate(expectedAnnotation, pvcAnnotationUpdated)
				shouldExpectVMIVolumesRemovePatched(vmi)
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

				controller.Execute()
				Eventually(func() bool {
					select {
					case updated := <-pvcAnnotationUpdated:
						return updated
					default:
					}
					return false
				}, 10*time.Second, 2).Should(BeTrue(), "failed, pvc annotation wasn't updated")
			},
				Entry("when phase is Unmounting", virtv1.MemoryDumpUnmounting, targetFileName),
				Entry("when phase is Failed", virtv1.MemoryDumpFailed, "Memory dump failed"),
			)

			It("should update memory dump to complete once memory dump volume unmounted", func() {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Status.Created = true
				vm.Status.Ready = true
				now := metav1.Now()
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        virtv1.MemoryDumpUnmounting,
					EndTimestamp: &now,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				// in case the volume is not in vmi volume status we should update status to completed
				updatedMemoryDump := &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName:    testPVCName,
					Phase:        virtv1.MemoryDumpCompleted,
					EndTimestamp: &now,
				}

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
				}).Return(nil, nil)

				controller.Execute()
			})

			It("should remove memory dump volume from vm volumes list when status is Dissociating", func() {
				// No need to add vmi - can do this action even if vm not running
				vm, _ := DefaultVirtualMachine(false)
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpDissociating,
				}

				vm.Spec.Template.Spec = *applyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
				addVirtualMachine(vm)

				vmInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Spec.Template.Spec.Volumes).To(BeEmpty())
				}).Return(vm, nil)
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Return(vm, nil)

				controller.Execute()
			})

			It("should dissociate memory dump request when status is Dissociating and not in vm volumes", func() {
				// No need to add vmi - can do this action even if vm not running
				vm, _ := DefaultVirtualMachine(false)
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     virtv1.MemoryDumpDissociating,
				}

				addVirtualMachine(vm)

				// in case the volume is not in vm volumes we should remove memory dump request
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachine).Status.MemoryDumpRequest).To(BeNil())
				}).Return(nil, nil)

				controller.Execute()
			})

			DescribeTable("should not setup vmi with memory dump if memory dump", func(phase virtv1.MemoryDumpPhase) {
				vm, _ := DefaultVirtualMachine(true)
				vm.Status.MemoryDumpRequest = &virtv1.VirtualMachineMemoryDumpRequest{
					ClaimName: testPVCName,
					Phase:     phase,
				}

				vmi := controller.setupVMIFromVM(vm)
				Expect(vmi.Spec.Volumes).To(BeEmpty())

			},
				Entry("in phase Unmounting", virtv1.MemoryDumpUnmounting),
				Entry("in phase Completed", virtv1.MemoryDumpCompleted),
				Entry("in phase Dissociating", virtv1.MemoryDumpDissociating),
			)

		})

		Context("VM printableStatus", func() {

			It("Should set a Stopped status when running=false and VMI doesn't exist", func() {
				vm, _ := DefaultVirtualMachine(false)
				addVirtualMachine(vm)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopped))
				})

				controller.Execute()
			})

			DescribeTable("should set a Stopped status when VMI exists but stopped", func(phase virtv1.VirtualMachineInstancePhase, deletionTimestamp *metav1.Time) {
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

				vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).AnyTimes()
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopped))
				})

				shouldExpectVMIFinalizerRemoval(vmi)

				controller.Execute()
			},

				Entry("in Succeeded state", virtv1.Succeeded, nil),
				Entry("in Succeeded state with a deletionTimestamp", virtv1.Succeeded, &metav1.Time{Time: time.Now()}),
				Entry("in Failed state", virtv1.Failed, nil),
				Entry("in Failed state with a deletionTimestamp", virtv1.Failed, &metav1.Time{Time: time.Now()}),
			)

			It("Should set a Starting status when running=true and VMI doesn't exist", func() {
				vm, vmi := DefaultVirtualMachine(true)
				addVirtualMachine(vm)

				vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			})

			DescribeTable("Should set a Starting status when VMI is in a startup phase", func(phase virtv1.VirtualMachineInstancePhase) {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = phase

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStarting))
				})

				controller.Execute()
			},

				Entry("VMI has no phase set", virtv1.VmPhaseUnset),
				Entry("VMI is in Pending phase", virtv1.Pending),
				Entry("VMI is in Scheduling phase", virtv1.Scheduling),
				Entry("VMI is in Scheduled phase", virtv1.Scheduled),
			)

			DescribeTable("Should set a CrashLoop status when VMI is deleted and VM is in crash loop backoff", func(status virtv1.VirtualMachineStatus, runStrategy virtv1.VirtualMachineRunStrategy, hasVMI bool, expectCrashloop bool) {
				vm, vmi := DefaultVirtualMachine(true)
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategy
				vm.Status = status

				addVirtualMachine(vm)
				if hasVMI {
					vmi.Status.Phase = virtv1.Running
					vmiFeeder.Add(vmi)
				}

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					if expectCrashloop {
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusCrashLoopBackOff))
					} else {
						Expect(objVM.Status.PrintableStatus).ToNot(Equal(virtv1.VirtualMachineStatusCrashLoopBackOff))
					}
				})

				if runStrategy == virtv1.RunStrategyRerunOnFailure {
					vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.MergePatchType, gomock.Any(), gomock.Any()).Do(
						func(ctx context.Context, name string, patchType types.PatchType, body []byte, opts *metav1.PatchOptions) {
							Expect(string(body)).To(ContainSubstring(`"action":"Start"`))
						}).Return(vm, nil).Times(1)
				}

				controller.Execute()
			},

				Entry("vm with runStrategy always and crash loop",
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
				Entry("vm with runStrategy rerun on failure and crash loop",
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
				Entry("vm with runStrategy halt should not report crash loop",
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
				Entry("vm with runStrategy manual should not report crash loop",
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
				Entry("vm with runStrategy once should not report crash loop",
					virtv1.VirtualMachineStatus{
						StartFailure: &virtv1.VirtualMachineStartFailure{
							ConsecutiveFailCount: 1,
							RetryAfterTimestamp: &metav1.Time{
								Time: time.Now().Add(300 * time.Second),
							},
						},
					},
					virtv1.RunStrategyOnce,
					true,
					false),
				Entry("vm with runStrategy always and VMI still exists should not report crash loop",
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

				DescribeTable("Should set a appropriate status when DataVolume exists but not bound", func(running bool, phase cdiv1.DataVolumePhase, status virtv1.VirtualMachinePrintableStatus) {
					vm.Spec.Running = &running
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = phase
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
					Expect(pvcInformer.GetStore().Add(&pvc)).To(Succeed())

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(status))
					})

					if running {
						vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
					}

					controller.Execute()
				},
					Entry("Started VM PendingPopulation", true, cdiv1.PendingPopulation, virtv1.VirtualMachineStatusWaitingForVolumeBinding),
					Entry("Started VM WFFC", true, cdiv1.WaitForFirstConsumer, virtv1.VirtualMachineStatusWaitingForVolumeBinding),
					Entry("Stopped VM PendingPopulation", false, cdiv1.PendingPopulation, virtv1.VirtualMachineStatusStopped),
					Entry("Stopped VM", false, cdiv1.WaitForFirstConsumer, virtv1.VirtualMachineStatusStopped),
				)

				DescribeTable("Should set a Provisioning status when DataVolume bound but not ready",
					func(dvPhase cdiv1.DataVolumePhase) {
						addVirtualMachine(vm)

						dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
						dv.Status.Phase = dvPhase
						dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
							Type:   cdiv1.DataVolumeBound,
							Status: k8score.ConditionTrue,
						})
						dataVolumeFeeder.Add(dv)

						if dvPhase == cdiv1.WaitForFirstConsumer {
							vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
						}
						vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
							objVM := obj.(*virtv1.VirtualMachine)
							Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusProvisioning))
						})

						controller.Execute()
					},

					Entry("DataVolume is in ImportScheduled phase", cdiv1.ImportScheduled),
					Entry("DataVolume is in ImportInProgress phase", cdiv1.ImportInProgress),
					Entry("DataVolume is in WaitForFirstConsumer phase", cdiv1.WaitForFirstConsumer),
				)

				DescribeTable("Should set a DataVolumeError status when DataVolume reports an error", func(dvFunc func(*cdiv1.DataVolume)) {
					addVirtualMachine(vm)

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dvFunc(dv)
					dataVolumeFeeder.Add(dv)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusDataVolumeError))
					})

					controller.Execute()
				},

					Entry(
						"DataVolume is in Failed phase",
						func(dv *cdiv1.DataVolume) {
							dv.Status.Phase = cdiv1.Failed
						},
					),
					Entry(
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

					dv, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv.Status.Phase = cdiv1.CloneInProgress
					dv.Status.Conditions = append(dv.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})
					dataVolumeFeeder.Add(dv)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
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

					dv1, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[0], vm)
					dv1.Status.Phase = cdiv1.Succeeded
					dv1.Status.Conditions = append(dv1.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})
					dv2, _ := watchutil.CreateDataVolumeManifest(virtClient, vm.Spec.DataVolumeTemplates[1], vm)
					dv2.Status.Phase = cdiv1.ImportInProgress
					dv2.Status.Conditions = append(dv2.Status.Conditions, cdiv1.DataVolumeCondition{
						Type:   cdiv1.DataVolumeBound,
						Status: k8score.ConditionTrue,
					})

					dataVolumeFeeder.Add(dv1)
					dataVolumeFeeder.Add(dv2)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
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

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Return(vmi, nil)

					addVirtualMachine(vm)
				})

				DescribeTable("Should set a WaitingForVolumeBinding status when PersistentVolumeClaim exists but unbound", func(pvcPhase k8sv1.PersistentVolumeClaimPhase) {
					pvc := k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pvc1",
							Namespace: vm.Namespace,
						},
						Status: k8sv1.PersistentVolumeClaimStatus{
							Phase: pvcPhase,
						},
					}
					Expect(pvcInformer.GetStore().Add(&pvc)).To(Succeed())

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusWaitingForVolumeBinding))
					})

					controller.Execute()
				},

					Entry("PersistentVolumeClaim is in Pending phase", k8sv1.ClaimPending),
					Entry("PersistentVolumeClaim is in Lost phase", k8sv1.ClaimLost),
				)

			})

			It("should set a Running status when VMI is running but not paused", func() {
				vm, vmi := DefaultVirtualMachine(true)

				vmi.Status.Phase = virtv1.Running

				addVirtualMachine(vm)
				vmiFeeder.Add(vmi)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
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

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusPaused))
				})

				controller.Execute()
			})

			DescribeTable("should set a Stopping status when VMI has a deletion timestamp set", func(phase virtv1.VirtualMachineInstancePhase, condType virtv1.VirtualMachineInstanceConditionType) {
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

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusStopping))
				})

				controller.Execute()
			},

				Entry("when VMI is pending", virtv1.Pending, virtv1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is provisioning", virtv1.Pending, virtv1.VirtualMachineInstanceProvisioning),
				Entry("when VMI is scheduling", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is scheduled", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is running", virtv1.Running, virtv1.VirtualMachineInstanceConditionType("")),
				Entry("when VMI is paused", virtv1.Running, virtv1.VirtualMachineInstancePaused),
			)

			Context("should set a Terminating status when VM has a deletion timestamp set", func() {
				DescribeTable("when VMI exists", func(phase virtv1.VirtualMachineInstancePhase, condType virtv1.VirtualMachineInstanceConditionType) {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vm.DeletionGracePeriodSeconds = kvpointer.P(v1.DefaultGracePeriodSeconds)
					vmi.Status.Phase = phase

					if condType != "" {
						vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
							Type:   condType,
							Status: k8sv1.ConditionTrue,
						})
					}
					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					shouldExpectGracePeriodPatched(v1.DefaultGracePeriodSeconds, vmi)
					vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).AnyTimes()
					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					Entry("when VMI is pending", virtv1.Pending, virtv1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is provisioning", virtv1.Pending, virtv1.VirtualMachineInstanceProvisioning),
					Entry("when VMI is scheduling", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is scheduled", virtv1.Scheduling, virtv1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is running", virtv1.Running, virtv1.VirtualMachineInstanceConditionType("")),
					Entry("when VMI is paused", virtv1.Running, virtv1.VirtualMachineInstancePaused),
				)

				It("when VMI exists and has a deletion timestamp set", func() {
					vm, vmi := DefaultVirtualMachine(true)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					vmi.Status.Phase = virtv1.Running

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				})

				DescribeTable("when VMI does not exist", func(running bool) {
					vm, _ := DefaultVirtualMachine(running)

					vm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					addVirtualMachine(vm)

					shouldExpectVMFinalizerRemoval(vm)
					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusTerminating))
					})

					controller.Execute()
				},

					Entry("with running: true", true),
					Entry("with running: false", false),
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

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
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

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachineStatusUnknown))
				})

				controller.Execute()
			})

			DescribeTable("should set a failure status in accordance to VMI condition",
				func(status virtv1.VirtualMachinePrintableStatus, cond virtv1.VirtualMachineInstanceCondition) {

					vm, vmi := DefaultVirtualMachine(true)
					vmi.Status.Phase = virtv1.Scheduling
					vmi.Status.Conditions = append(vmi.Status.Conditions, cond)

					addVirtualMachine(vm)
					vmiFeeder.Add(vmi)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						Expect(objVM.Status.PrintableStatus).To(Equal(status))
					})

					controller.Execute()
				},

				Entry("FailedUnschedulable", virtv1.VirtualMachineStatusUnschedulable,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
						Status: k8sv1.ConditionFalse,
						Reason: k8sv1.PodReasonUnschedulable,
					},
				),
				Entry("FailedPvcNotFound", virtv1.VirtualMachineStatusPvcNotFound,
					virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceSynchronized,
						Status: k8sv1.ConditionFalse,
						Reason: FailedPvcNotFoundReason,
					},
				),
			)

			DescribeTable("should set an ImagePullBackOff/ErrPullImage statuses according to VMI Synchronized condition", func(reason string) {
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

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
					objVM := obj.(*virtv1.VirtualMachine)
					Expect(objVM.Status.PrintableStatus).To(Equal(virtv1.VirtualMachinePrintableStatus(reason)))
				})

				controller.Execute()
			},
				Entry("Reason: ErrImagePull", ErrImagePullReason),
				Entry("Reason: ImagePullBackOff", ImagePullBackOffReason),
			)
		})

		Context("Instancetype and Preferences", func() {

			const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
			const resourceGeneration int64 = 1

			var (
				vm  *virtv1.VirtualMachine
				vmi *virtv1.VirtualMachineInstance

				fakeInstancetypeClients       instancetypeclientset.InstancetypeV1beta1Interface
				fakeInstancetypeClient        instancetypeclientset.VirtualMachineInstancetypeInterface
				fakeClusterInstancetypeClient instancetypeclientset.VirtualMachineClusterInstancetypeInterface
				fakePreferenceClient          instancetypeclientset.VirtualMachinePreferenceInterface
				fakeClusterPreferenceClient   instancetypeclientset.VirtualMachineClusterPreferenceInterface

				instancetypeInformerStore        cache.Store
				clusterInstancetypeInformerStore cache.Store
				preferenceInformerStore          cache.Store
				clusterPreferenceInformerStore   cache.Store
				controllerrevisionInformerStore  cache.Store
			)

			BeforeEach(func() {
				vm, vmi = DefaultVirtualMachine(true)

				// We need to clear the domainSpec here to ensure the instancetype doesn't conflict
				vm.Spec.Template.Spec.Domain = v1.DomainSpec{}

				fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1beta1()

				fakeInstancetypeClient = fakeInstancetypeClients.VirtualMachineInstancetypes(metav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachineInstancetype(gomock.Any()).Return(fakeInstancetypeClient).AnyTimes()

				fakeClusterInstancetypeClient = fakeInstancetypeClients.VirtualMachineClusterInstancetypes()
				virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeClusterInstancetypeClient).AnyTimes()

				fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(metav1.NamespaceDefault)
				virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()

				fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
				virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

				k8sClient = k8sfake.NewSimpleClientset()
				virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

				instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
				instancetypeInformerStore = instancetypeInformer.GetStore()

				clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
				clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

				preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
				preferenceInformerStore = preferenceInformer.GetStore()

				clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
				clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

				controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
				controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

				controller.instancetypeMethods = &instancetype.InstancetypeMethods{
					InstancetypeStore:        instancetypeInformerStore,
					ClusterInstancetypeStore: clusterInstancetypeInformerStore,
					PreferenceStore:          preferenceInformerStore,
					ClusterPreferenceStore:   clusterPreferenceInformerStore,
					ControllerRevisionStore:  controllerrevisionInformerStore,
					Clientset:                virtClient,
				}
			})

			Context("instancetype", func() {
				var (
					instancetypeObj        *instancetypev1beta1.VirtualMachineInstancetype
					clusterInstancetypeObj *instancetypev1beta1.VirtualMachineClusterInstancetype
				)

				BeforeEach(func() {
					instancetypeSpec := instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128M"),
						},
					}
					instancetypeObj = &instancetypev1beta1.VirtualMachineInstancetype{
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachineInstancetype",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:       "instancetype",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypeSpec,
					}
					_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), instancetypeObj, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = instancetypeInformerStore.Add(instancetypeObj)
					Expect(err).NotTo(HaveOccurred())

					clusterInstancetypeObj = &instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "clusterInstancetype",
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypeSpec,
					}
					_, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetypeObj, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = clusterInstancetypeInformerStore.Add(clusterInstancetypeObj)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should apply VirtualMachineInstancetype to VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					addVirtualMachine(vm)

					expectedRevisionName := instancetype.GetRevisionName(vm.Name, instancetypeObj.Name, instancetypeObj.UID, instancetypeObj.Generation)
					expectedRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
					Expect(err).ToNot(HaveOccurred())
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(expectedRevision, nil)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					revisionInstancetype, ok := revision.Data.Object.(*instancetypev1beta1.VirtualMachineInstancetype)
					Expect(ok).To(BeTrue(), "Expected Instancetype in ControllerRevision")

					Expect(revisionInstancetype.Spec).To(Equal(instancetypeObj.Spec))
				})

				DescribeTable("should apply VirtualMachineInstancetype from ControllerRevision to VirtualMachineInstance", func(getRevisionData func() []byte) {
					instancetypeRevision := &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name: "crName",
						},
						Data: runtime.RawExtension{
							Raw: getRevisionData(),
						},
					}

					instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         instancetypeObj.Name,
						Kind:         instancetypeapi.SingularResourceName,
						RevisionName: instancetypeRevision.Name,
					}

					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

				},
					Entry("using v1alpha1 and VirtualMachineInstancetypeSpecRevision with APIVersion", func() []byte {
						v1alpha1instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: instancetypeObj.Spec.CPU.Guest,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: instancetypeObj.Spec.Memory.Guest,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1instancetypeSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1 and VirtualMachineInstancetypeSpecRevision without APIVersion", func() []byte {
						v1alpha1instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
							CPU: instancetypev1alpha1.CPUInstancetype{
								Guest: instancetypeObj.Spec.CPU.Guest,
							},
							Memory: instancetypev1alpha1.MemoryInstancetype{
								Guest: instancetypeObj.Spec.Memory.Guest,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1instancetypeSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
							APIVersion: "",
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1", func() []byte {
						v1alpha1instancetype := &instancetypev1alpha1.VirtualMachineInstancetype{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
								Kind:       "VirtualMachineInstancetype",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: instancetypeObj.Name,
							},
							Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha1.CPUInstancetype{
									Guest: instancetypeObj.Spec.CPU.Guest,
								},
								Memory: instancetypev1alpha1.MemoryInstancetype{
									Guest: instancetypeObj.Spec.Memory.Guest,
								},
							},
						}
						instancetypeBytes, err := json.Marshal(v1alpha1instancetype)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
					Entry("using v1alpha2", func() []byte {
						v1alpha2instancetype := &instancetypev1alpha2.VirtualMachineInstancetype{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
								Kind:       "VirtualMachineInstancetype",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: instancetypeObj.Name,
							},
							Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha2.CPUInstancetype{
									Guest: instancetypeObj.Spec.CPU.Guest,
								},
								Memory: instancetypev1alpha2.MemoryInstancetype{
									Guest: instancetypeObj.Spec.Memory.Guest,
								},
							},
						}
						instancetypeBytes, err := json.Marshal(v1alpha2instancetype)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
					Entry("using v1beta1", func() []byte {
						instancetypeBytes, err := json.Marshal(instancetypeObj)
						Expect(err).ToNot(HaveOccurred())

						return instancetypeBytes
					}),
				)

				It("should apply VirtualMachineInstancetype to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, instancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					// We expect a request to add in the missing instancetype revisionName
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeRevision, nil)
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(instancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.InstancetypeAnnotation, instancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeRevision.Name))

				})

				It("should apply VirtualMachineClusterInstancetype to VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: clusterInstancetypeObj.Name,
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					addVirtualMachine(vm)

					expectedRevisionName := instancetype.GetRevisionName(vm.Name, clusterInstancetypeObj.Name, clusterInstancetypeObj.UID, clusterInstancetypeObj.Generation)
					expectedRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(expectedRevision, nil)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					revisionClusterInstancetype, ok := revision.Data.Object.(*instancetypev1beta1.VirtualMachineClusterInstancetype)
					Expect(ok).To(BeTrue(), "Expected ClusterInstancetype in ControllerRevision")

					Expect(revisionClusterInstancetype.Spec).To(Equal(clusterInstancetypeObj.Spec))
				})

				It("should apply VirtualMachineClusterInstancetype from ControllerRevision to VirtualMachineInstance", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name:         clusterInstancetypeObj.Name,
						Kind:         instancetypeapi.ClusterSingularResourceName,
						RevisionName: instancetypeRevision.Name,
					}

					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

				})

				It("should apply VirtualMachineClusterInstancetype to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetypeObj)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					// We expect a request to add in the missing instancetype revisionName
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeRevision, nil)
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: clusterInstancetypeObj.Name,
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
						Expect(*vmiArg.Spec.Domain.Memory.Guest).To(Equal(clusterInstancetypeObj.Spec.Memory.Guest))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeRevision.Name))

				})

				It("should reject request if an invalid InstancetypeMatcher Kind is provided", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: "foobar",
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
						Expect(cond.Message).To(ContainSubstring("got unexpected kind in InstancetypeMatcher"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachineInstancetype cannot be found", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: "foobar",
						Kind: instancetypeapi.SingularResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachineClusterInstancetype cannot be found", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: "foobar",
						Kind: instancetypeapi.ClusterSingularResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should fail if the VirtualMachineInstancetype conflicts with the VirtualMachineInstance", func() {

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: uint32(1),
						Cores:   uint32(4),
						Threads: uint32(1),
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
						Expect(cond.Message).To(ContainSubstring("Error encountered while storing Instancetype ControllerRevisions: VM field conflicts with selected Instancetype"))
						Expect(cond.Message).To(ContainSubstring("spec.template.spec.domain.cpu"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
				})

				It("should reject if an existing ControllerRevision is found with unexpected VirtualMachineInstancetypeSpec data", func() {
					unexpectedInstancetype := instancetypeObj.DeepCopy()
					unexpectedInstancetype.Spec.CPU.Guest = 15

					instancetypeRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Instancetype = &v1.InstancetypeMatcher{
						Name: instancetypeObj.Name,
						Kind: instancetypeapi.SingularResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
						Expect(cond.Message).To(ContainSubstring("found existing ControllerRevision with unexpected data"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})
			})

			Context("preference", func() {
				var (
					preference        *instancetypev1beta1.VirtualMachinePreference
					clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference
				)

				BeforeEach(func() {
					preferenceSpec := instancetypev1beta1.VirtualMachinePreferenceSpec{
						Firmware: &instancetypev1beta1.FirmwarePreferences{
							PreferredUseEfi: pointer.Bool(true),
						},
						Devices: &instancetypev1beta1.DevicePreferences{
							PreferredDiskBus:        virtv1.DiskBusVirtio,
							PreferredInterfaceModel: "virtio",
							PreferredInputBus:       virtv1.InputBusUSB,
							PreferredInputType:      virtv1.InputTypeTablet,
						},
					}
					preference = &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "preference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						TypeMeta: metav1.TypeMeta{
							APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
							Kind:       "VirtualMachinePreference",
						},
						Spec: preferenceSpec,
					}
					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = preferenceInformerStore.Add(preference)
					Expect(err).NotTo(HaveOccurred())

					clusterPreference = &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "clusterPreference",
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: preferenceSpec,
					}
					_, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					err = clusterPreferenceInformerStore.Add(clusterPreference)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should apply VirtualMachinePreference to VirtualMachineInstance", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					expectedPreferenceRevisionName := instancetype.GetRevisionName(vm.Name, preference.Name, preference.UID, preference.Generation)
					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*instancetypev1beta1.VirtualMachinePreference)
					Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
					Expect(preferenceRevisionObj.Spec).To(Equal(preference.Spec))
				})

				DescribeTable("should apply VirtualMachinePreference from ControllerRevision to VirtualMachineInstance", func(getRevisionData func() []byte) {
					preferenceRevision := &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name: "crName",
						},
						Data: runtime.RawExtension{
							Raw: getRevisionData(),
						},
					}

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name:         preference.Name,
						Kind:         instancetypeapi.SingularPreferenceResourceName,
						RevisionName: preferenceRevision.Name,
					}

					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

				},
					Entry("using v1alpha1 and VirtualMachinePreferenceSpecRevision with APIVersion", func() []byte {
						v1alpha1preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
							Firmware: &instancetypev1alpha1.FirmwarePreferences{
								PreferredUseEfi: pointer.Bool(true),
							},
							Devices: &instancetypev1alpha1.DevicePreferences{
								PreferredDiskBus:        virtv1.DiskBusVirtio,
								PreferredInterfaceModel: "virtio",
								PreferredInputBus:       virtv1.InputBusUSB,
								PreferredInputType:      virtv1.InputTypeTablet,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1preferenceSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
							APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1 and VirtualMachinePreferenceSpecRevision without APIVersion", func() []byte {
						v1alpha1preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
							Firmware: &instancetypev1alpha1.FirmwarePreferences{
								PreferredUseEfi: pointer.Bool(true),
							},
							Devices: &instancetypev1alpha1.DevicePreferences{
								PreferredDiskBus:        virtv1.DiskBusVirtio,
								PreferredInterfaceModel: "virtio",
								PreferredInputBus:       virtv1.InputBusUSB,
								PreferredInputType:      virtv1.InputTypeTablet,
							},
						}

						specBytes, err := json.Marshal(&v1alpha1preferenceSpec)
						Expect(err).ToNot(HaveOccurred())

						specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
							APIVersion: "",
							Spec:       specBytes,
						}
						specRevisionBytes, err := json.Marshal(specRevision)
						Expect(err).ToNot(HaveOccurred())

						return specRevisionBytes
					}),
					Entry("using v1alpha1", func() []byte {
						v1alpha1preference := &instancetypev1alpha1.VirtualMachinePreference{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
								Kind:       "VirtualMachinePreference",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: preference.Name,
							},
							Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
								Firmware: &instancetypev1alpha1.FirmwarePreferences{
									PreferredUseEfi: pointer.Bool(true),
								},
								Devices: &instancetypev1alpha1.DevicePreferences{
									PreferredDiskBus:        virtv1.DiskBusVirtio,
									PreferredInterfaceModel: "virtio",
									PreferredInputBus:       virtv1.InputBusUSB,
									PreferredInputType:      virtv1.InputTypeTablet,
								},
							},
						}
						preferenceBytes, err := json.Marshal(v1alpha1preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
					Entry("using v1alpha2", func() []byte {
						v1alpha2preference := &instancetypev1alpha2.VirtualMachinePreference{
							TypeMeta: metav1.TypeMeta{
								APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
								Kind:       "VirtualMachinePreference",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: preference.Name,
							},
							Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
								Firmware: &instancetypev1alpha2.FirmwarePreferences{
									PreferredUseEfi: pointer.Bool(true),
								},
								Devices: &instancetypev1alpha2.DevicePreferences{
									PreferredDiskBus:        virtv1.DiskBusVirtio,
									PreferredInterfaceModel: "virtio",
									PreferredInputBus:       virtv1.InputBusUSB,
									PreferredInputType:      virtv1.InputTypeTablet,
								},
							},
						}
						preferenceBytes, err := json.Marshal(v1alpha2preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
					Entry("using v1beta1", func() []byte {
						preferenceBytes, err := json.Marshal(preference)
						Expect(err).ToNot(HaveOccurred())

						return preferenceBytes
					}),
				)

				It("should apply VirtualMachinePreference to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					// We expect a request to add in the missing preference revisionName
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, preferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceRevision.Name))

				})

				It("should apply VirtualMachineClusterPreference to VirtualMachineInstance", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: clusterPreference.Name,
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					expectedPreferenceRevisionName := instancetype.GetRevisionName(vm.Name, clusterPreference.Name, clusterPreference.UID, clusterPreference.Generation)
					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))

					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*instancetypev1beta1.VirtualMachineClusterPreference)
					Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
					Expect(preferenceRevisionObj.Spec).To(Equal(clusterPreference.Spec))
				})

				It("should apply VirtualMachineClusterPreference from ControllerRevision to VirtualMachineInstance", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name:         clusterPreference.Name,
						Kind:         instancetypeapi.ClusterSingularPreferenceResourceName,
						RevisionName: preferenceRevision.Name,
					}

					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

				})

				It("should apply VirtualMachineClusterPreference to VirtualMachineInstance if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
					preferenceRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					// We expect a request to add in the missing preference revisionName
					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, preferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: clusterPreference.Name,
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())

						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.InstancetypeAnnotation))
						Expect(vmiArg.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, clusterPreference.Name))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.ClusterInstancetypeAnnotation))
						Expect(vmiArg.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()

					Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceRevision.Name))

				})

				It("should reject the request if an invalid PreferenceMatcher Kind is provided", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: "foobar",
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
						Expect(cond.Message).To(ContainSubstring("got unexpected kind in PreferenceMatcher"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachinePreference cannot be found", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: "foobar",
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject the request if a VirtualMachineClusterPreference cannot be found", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: "foobar",
						Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should reject if an existing ControllerRevision is found with unexpected VirtualMachinePreferenceSpec data", func() {
					unexpectedPreference := preference.DeepCopy()
					unexpectedPreference.Spec.Firmware = &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios: pointer.Bool(true),
					}

					preferenceRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj interface{}) {
						objVM := obj.(*virtv1.VirtualMachine)
						cond := virtcontroller.NewVirtualMachineConditionManager().GetCondition(objVM, virtv1.VirtualMachineFailure)
						Expect(cond).To(Not(BeNil()))
						Expect(cond.Type).To(Equal(virtv1.VirtualMachineFailure))
						Expect(cond.Reason).To(Equal("FailedCreate"))
						Expect(cond.Message).To(ContainSubstring("found existing ControllerRevision with unexpected data"))
					}).Return(vm, nil)

					controller.Execute()

					testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)

				})

				It("should apply preferences to default network interface", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.Interfaces = []virtv1.Interface{}
					vm.Spec.Template.Spec.Networks = []virtv1.Network{}

					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Devices.Interfaces[0].Model).To(Equal(preference.Spec.Devices.PreferredInterfaceModel))
						Expect(vmiArg.Spec.Networks).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})

				It("should apply preferredAutoattachPodInterface and skip adding default network interface", func() {

					autoattachPodInterfacePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "autoattachPodInterfacePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachPodInterface: pointer.Bool(false),
							},
						},
					}

					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachPodInterfacePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachPodInterfacePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.Interfaces = []virtv1.Interface{}
					vm.Spec.Template.Spec.Networks = []virtv1.Network{}

					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachPodInterfacePreference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(*vmiArg.Spec.Domain.Devices.AutoattachPodInterface).To(BeFalse())
						Expect(vmiArg.Spec.Domain.Devices.Interfaces).To(BeEmpty())
						Expect(vmiArg.Spec.Networks).To(BeEmpty())
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})

				It("should apply preferences to default volume disk", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					presentVolumeName := "present-vol"
					missingVolumeName := "missing-vol"
					vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{
						v1.Disk{
							Name: presentVolumeName,
							DiskDevice: v1.DiskDevice{
								Disk: &v1.DiskTarget{
									Bus: v1.DiskBusSATA,
								},
							},
						},
					}
					vm.Spec.Template.Spec.Volumes = []v1.Volume{
						v1.Volume{
							Name: presentVolumeName,
						},
						v1.Volume{
							Name: missingVolumeName,
						},
					}

					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Devices.Disks).To(HaveLen(2))
						Expect(vmiArg.Spec.Domain.Devices.Disks[0].Name).To(Equal(presentVolumeName))
						// Assert that the preference hasn't overwritten anything defined by the user
						Expect(vmiArg.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(v1.DiskBusSATA))
						Expect(vmiArg.Spec.Domain.Devices.Disks[1].Name).To(Equal(missingVolumeName))
						// Assert that it has however been applied to the newly introduced disk
						Expect(vmiArg.Spec.Domain.Devices.Disks[1].Disk.Bus).To(Equal(preference.Spec.Devices.PreferredDiskBus))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})

				It("should apply preferences to AutoattachInputDevice attached input device", func() {

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: preference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = pointer.Bool(true)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, preference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Devices.Inputs).To(HaveLen(1))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Type).To(Equal(preference.Spec.Devices.PreferredInputType))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(preference.Spec.Devices.PreferredInputBus))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})

				It("should apply preferences to preferredAutoattachInputDevice attached input device", func() {

					autoattachInputDevicePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "autoattachInputDevicePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachInputDevice: pointer.Bool(true),
								PreferredInputBus:              virtv1.InputBusVirtio,
								PreferredInputType:             virtv1.InputTypeTablet,
							},
						},
					}
					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachInputDevicePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachInputDevicePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachInputDevicePreference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					addVirtualMachine(vm)

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(vmiArg.Spec.Domain.Devices.Inputs).To(HaveLen(1))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Type).To(Equal(autoattachInputDevicePreference.Spec.Devices.PreferredInputType))
						Expect(vmiArg.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(autoattachInputDevicePreference.Spec.Devices.PreferredInputBus))
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})

				It("should apply preferredAutoattachInputDevice and skip adding default input device", func() {

					autoattachInputDevicePreference := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "preferredAutoattachInputDevicePreference",
							Namespace:  vm.Namespace,
							UID:        resourceUID,
							Generation: resourceGeneration,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							Devices: &instancetypev1beta1.DevicePreferences{
								PreferredAutoattachInputDevice: pointer.Bool(false),
							},
						},
					}

					_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), autoattachInputDevicePreference, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					vm.Spec.Preference = &v1.PreferenceMatcher{
						Name: autoattachInputDevicePreference.Name,
						Kind: instancetypeapi.SingularPreferenceResourceName,
					}

					addVirtualMachine(vm)

					expectedPreferenceRevision, err := instancetype.CreateControllerRevision(vm, autoattachInputDevicePreference)
					Expect(err).ToNot(HaveOccurred())

					expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, expectedPreferenceRevision)
					Expect(err).ToNot(HaveOccurred())

					vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
						vmiArg := arg.(*virtv1.VirtualMachineInstance)
						Expect(*vmiArg.Spec.Domain.Devices.AutoattachInputDevice).To(BeFalse())
						Expect(vmiArg.Spec.Domain.Devices.Inputs).To(BeEmpty())
					}).Return(vmi, nil)

					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

					controller.Execute()
				})
			})
		})

		DescribeTable("should add the default network interface",
			func(iface string) {
				vm, vmi := DefaultVirtualMachine(true)

				expectedIface := "bridge"
				switch iface {
				case "masquerade":
					expectedIface = "masquerade"
				case "slirp":
					expectedIface = "slirp"
				}

				permit := true
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							NetworkConfiguration: &v1.NetworkConfiguration{
								NetworkInterface:     expectedIface,
								PermitSlirpInterface: &permit,
							},
						},
					},
				})

				addVirtualMachine(vm)

				vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
					vmiArg := arg.(*virtv1.VirtualMachineInstance)
					switch expectedIface {
					case "bridge":
						Expect(vmiArg.Spec.Domain.Devices.Interfaces[0].Bridge).NotTo(BeNil())
					case "masquerade":
						Expect(vmiArg.Spec.Domain.Devices.Interfaces[0].Masquerade).NotTo(BeNil())
					case "slirp":
						Expect(vmiArg.Spec.Domain.Devices.Interfaces[0].Slirp).NotTo(BeNil())
					}
					Expect(vmiArg.Spec.Networks).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
				}).Return(vmi, nil)

				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

				controller.Execute()

			},
			Entry("as bridge", "bridge"),
			Entry("as masquerade", "masquerade"),
			Entry("as slirp", "slirp"),
		)

		DescribeTable("should not add the default interfaces if", func(interfaces []v1.Interface, networks []v1.Network) {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Domain.Devices.Interfaces = append([]v1.Interface{}, interfaces...)
			vm.Spec.Template.Spec.Networks = append([]v1.Network{}, networks...)

			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
				vmiArg := arg.(*virtv1.VirtualMachineInstance)
				Expect(vmiArg.Spec.Domain.Devices.Interfaces).To(Equal(interfaces))
				Expect(vmiArg.Spec.Networks).To(Equal(networks))
			}).Return(vmi, nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

			controller.Execute()

		},
			Entry("interfaces and networks are non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{{Name: "b"}}),
			Entry("interfaces is non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{}),
			Entry("networks is non-empty", []v1.Interface{}, []v1.Network{{Name: "b"}}),
		)

		It("should add a missing volume disk", func() {
			vm, vmi := DefaultVirtualMachine(true)
			presentVolumeName := "present-vol"
			missingVolumeName := "missing-vol"
			vm.Spec.Template.Spec.Domain.Devices.Disks = []v1.Disk{
				v1.Disk{
					Name: presentVolumeName,
				},
			}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{
				v1.Volume{
					Name: presentVolumeName,
				},
				v1.Volume{
					Name: missingVolumeName,
				},
			}

			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
				vmiArg := arg.(*virtv1.VirtualMachineInstance)
				Expect(vmiArg.Spec.Domain.Devices.Disks).To(HaveLen(2))
				Expect(vmiArg.Spec.Domain.Devices.Disks[0].Name).To(Equal(presentVolumeName))
				Expect(vmiArg.Spec.Domain.Devices.Disks[1].Name).To(Equal(missingVolumeName))
			}).Return(vmi, nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

			controller.Execute()

		})

		DescribeTable("AutoattachInputDevice should ", func(autoAttach *bool, existingInputDevices []v1.Input, expectedInputDevice *v1.Input) {
			vm, vmi := DefaultVirtualMachine(true)
			vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = autoAttach
			vm.Spec.Template.Spec.Domain.Devices.Inputs = existingInputDevices

			addVirtualMachine(vm)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(1).Do(func(ctx context.Context, arg interface{}) {
				vmiArg := arg.(*virtv1.VirtualMachineInstance)

				if expectedInputDevice != nil {
					Expect(vmiArg.Spec.Domain.Devices.Inputs).To(HaveLen(1))
					Expect(vmiArg.Spec.Domain.Devices.Inputs[0]).To(Equal(*expectedInputDevice))
				} else {
					Expect(vmiArg.Spec.Domain.Devices.Inputs).To(BeEmpty())
				}

			}).Return(vmi, nil)

			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Times(1)

			controller.Execute()

		},
			Entry("add default input device when enabled in VirtualMachine", pointer.Bool(true), []v1.Input{}, &v1.Input{Name: "default-0"}),
			Entry("not add default input device when disabled by VirtualMachine", pointer.Bool(false), []v1.Input{}, nil),
			Entry("not add default input device by default", nil, []v1.Input{}, nil),
			Entry("not add default input device when devices already present in VirtualMachine", pointer.Bool(true), []v1.Input{{Name: "existing-0"}}, &v1.Input{Name: "existing-0"}),
		)

		Context("Live update features", func() {
			const maxSocketsFromSpec uint32 = 24
			const maxSocketsFromConfig uint32 = 48
			maxGuestFromSpec := resource.MustParse("128Mi")
			maxGuestFromConfig := resource.MustParse("256Mi")

			Context("CPU", func() {
				It("should honour the maximum CPU sockets from VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{MaxSockets: maxSocketsFromSpec}

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				It("should prefer maximum CPU sockets from VM spec rather than from cluster config", func() {
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{MaxSockets: maxSocketsFromSpec}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{
									MaxCpuSockets: kvpointer.P(maxSocketsFromConfig),
								},
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromSpec))
				})

				It("should use maximum sockets configured in cluster config when its not set in VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{
									MaxCpuSockets: kvpointer.P(maxSocketsFromConfig),
								},
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(maxSocketsFromConfig))
				})

				It("should calculate max sockets to be 4x times the configured sockets when no max sockets defined", func() {
					const cpuSockets uint32 = 4
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{
						Sockets: cpuSockets,
					}

					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(cpuSockets * 4))
				})

				It("should calculate max sockets to be 4x times the default sockets when default CPU topology used", func() {
					const defaultSockets uint32 = 1
					vm, _ := DefaultVirtualMachine(true)

					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(defaultSockets * 4))
				})
			})

			Context("Memory", func() {
				It("should honour the max guest memory from VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				It("should prefer maxGuest from VM spec rather than from cluster config", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{
									MaxGuest: &maxGuestFromConfig,
								},
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromSpec))
				})

				It("should use maxGuest configured in cluster config when its not set in VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{
									MaxGuest: &maxGuestFromConfig,
								},
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(*vmi.Spec.Domain.Memory.MaxGuest).To(Equal(maxGuestFromConfig))
				})

				It("should opt-out from memory live-update if liveUpdateFeatures is disabled in the VM spec", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("0")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.Memory.MaxGuest).To(BeNil())
				})

				It("should calculate maxGuest to be `MaxHotplugRatio` times the configured guest memory when no maxGuest is defined", func() {
					vm, _ := DefaultVirtualMachine(true)
					guestMemory := resource.MustParse("64Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}
					testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
						Spec: v1.KubeVirtSpec{
							Configuration: v1.KubeVirtConfiguration{
								VMRolloutStrategy: &v1.VMRolloutStrategy{
									LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
								},
								DeveloperConfiguration: &v1.DeveloperConfiguration{
									FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
								},
							},
						},
					})

					vmi := controller.setupVMIFromVM(vm)
					Expect(vmi.Spec.Domain.Memory.MaxGuest.Value()).To(Equal(guestMemory.Value() * int64(config.GetMaxHotplugRatio())))
				})

				It("should patch VMI when memory hotplug is requested", func() {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("128Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &newMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("64Mi")
					vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}

					memReqBuffer := resource.MustParse("100Mi")
					memoryRequest := guestMemory.DeepCopy()
					memoryRequest.Add(memReqBuffer)
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = memoryRequest

					vmi.Status.Memory = &virtv1.MemoryStatus{
						GuestAtBoot:    &guestMemory,
						GuestCurrent:   &guestMemory,
						GuestRequested: &guestMemory,
					}

					vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Do(func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) {
						originalVMIBytes, err := json.Marshal(vmi)
						Expect(err).ToNot(HaveOccurred())
						patchBytes := patch.([]byte)

						patchJSON, err := jsonpatch.DecodePatch(patchBytes)
						Expect(err).ToNot(HaveOccurred())
						newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
						Expect(err).ToNot(HaveOccurred())

						var newVMI *virtv1.VirtualMachineInstance
						err = json.Unmarshal(newVMIBytes, &newVMI)
						Expect(err).ToNot(HaveOccurred())

						// this tests when memory request is not equal guest memory
						expectedMemReq := vm.Spec.Template.Spec.Domain.Memory.Guest.DeepCopy()
						expectedMemReq.Add(memReqBuffer)

						Expect(newVMI.Spec.Domain.Memory.Guest.Value()).To(Equal(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()))
						Expect(newVMI.Spec.Domain.Resources.Requests.Memory().Value()).To(Equal(expectedMemReq.Value()))

					})

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should not patch VMI if memory hotplug is already in progress", func() {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("128Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &newMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("64Mi")
					vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &virtv1.MemoryStatus{
						GuestAtBoot:    &guestMemory,
						GuestCurrent:   &guestMemory,
						GuestRequested: &guestMemory,
					}

					condition := virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceMemoryChange,
						Status: k8sv1.ConditionTrue,
					}
					virtcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &condition)

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).To(HaveOccurred())
				})

				It("should not patch VMI if a migration is in progress", func() {
					vm, _ := DefaultVirtualMachine(true)
					newMemory := resource.MustParse("128Mi")
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &newMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					guestMemory := resource.MustParse("64Mi")
					vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &virtv1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}
					migrationStart := metav1.Now()
					vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
						StartTimestamp: &migrationStart,
					}

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).To(HaveOccurred())
				})

				It("should not patch VMI if guest memory did not change", func() {
					guestMemory := resource.MustParse("64Mi")
					vm, _ := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
						Guest:    &guestMemory,
						MaxGuest: &maxGuestFromSpec,
					}

					vmi := api.NewMinimalVMI(vm.Name)
					vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &guestMemory}
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = guestMemory
					vmi.Status.Memory = &virtv1.MemoryStatus{
						GuestAtBoot:  &guestMemory,
						GuestCurrent: &guestMemory,
					}

					err := controller.handleMemoryHotplugRequest(vm, vmi)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("CPU topology", func() {
			When("isn't set in VMI template", func() {
				It("Set default CPU topology in VMI status", func() {
					vm, vmi := DefaultVirtualMachine(true)
					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
						Expect(arg.(*virtv1.VirtualMachineInstance).Status.CurrentCPUTopology).To(Not(BeNil()))
					}).Return(vmi, nil)

					// expect update status is called
					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
						Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
						Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
					}).Return(nil, nil)

					controller.Execute()

					testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
				})
			})
			When("set in VMI template", func() {
				It("copy CPU topology to VMI status", func() {
					const (
						numOfSockets uint32 = 8
						numOfCores   uint32 = 8
						numOfThreads uint32 = 8
					)
					vm, vmi := DefaultVirtualMachine(true)
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						Sockets: numOfSockets,
						Cores:   numOfCores,
						Threads: numOfThreads,
					}
					addVirtualMachine(vm)

					vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
						currentCPUTopology := arg.(*virtv1.VirtualMachineInstance).Status.CurrentCPUTopology
						Expect(currentCPUTopology).To(Not(BeNil()))
						Expect(currentCPUTopology.Sockets).To(Equal(numOfSockets))
						Expect(currentCPUTopology.Cores).To(Equal(numOfCores))
						Expect(currentCPUTopology.Threads).To(Equal(numOfThreads))
					}).Return(vmi, nil)

					// expect update status is called
					vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
						Expect(arg.(*virtv1.VirtualMachine).Status.Created).To(BeFalse())
						Expect(arg.(*virtv1.VirtualMachine).Status.Ready).To(BeFalse())
					}).Return(nil, nil)

					controller.Execute()

					testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
				})
			})
		})

		Context("The RestartRequired condition", Serial, func() {
			var vm *virtv1.VirtualMachine
			var vmi *virtv1.VirtualMachineInstance
			var kv *virtv1.KubeVirt
			var crList appsv1.ControllerRevisionList
			var crListLock sync.Mutex
			var restartRequired = map[types.UID]bool{}

			expectVMUpdate := func() {
				vmInterface.EXPECT().Update(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, arg interface{}) (interface{}, error) {
					return arg, nil
				})
			}

			expectVMStatusUpdate := func() {
				vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, arg interface{}) (interface{}, error) {
					vmObj := arg.(*virtv1.VirtualMachine)
					for _, cond := range vmObj.Status.Conditions {
						if cond.Type == virtv1.VirtualMachineRestartRequired {
							restartRequired[vmObj.ObjectMeta.UID] = true
						}
					}

					return arg, nil
				}).AnyTimes()
			}

			expectVMICreation := func() {
				vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil).AnyTimes()
			}

			expectVMIPatch := func() {
				vmiInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(vmi, nil).AnyTimes()
			}

			expectControllerRevisionList := func() {
				k8sClient.Fake.PrependReactor("list", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					crListLock.Lock()
					defer crListLock.Unlock()
					return true, crList.DeepCopy(), nil
				})
			}

			expectControllerRevisionDelete := func() {
				k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					deleted, ok := action.(testing.DeleteAction)
					Expect(ok).To(BeTrue())

					crListLock.Lock()
					defer crListLock.Unlock()
					for i, obj := range crList.Items {
						if obj.Name == deleted.GetName() && obj.Namespace == deleted.GetNamespace() {
							crList.Items = append(crList.Items[:i], crList.Items[i+1:]...)
							return true, nil, nil
						}
					}
					return true, nil, fmt.Errorf("not found")
				})
			}

			expectControllerRevisionCreation := func() {
				k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					created, ok := action.(testing.CreateAction)
					Expect(ok).To(BeTrue())

					createObj, ok := created.GetObject().(*appsv1.ControllerRevision)
					Expect(ok).To(BeTrue())

					crListLock.Lock()
					defer crListLock.Unlock()
					crList.Items = append(crList.Items, *createObj)
					crSource.Add(createObj)

					return true, created.GetObject(), nil
				})
			}

			crListFor := func(uid string) []appsv1.ControllerRevision {
				var res []appsv1.ControllerRevision
				crListLock.Lock()
				defer crListLock.Unlock()
				for _, cr := range crList.Items {
					if strings.Contains(cr.Name, uid) {
						res = append(res, cr)
					}
				}

				return res
			}

			BeforeEach(func() {
				k8sClient.Fake.ClearActions()
				crList = appsv1.ControllerRevisionList{}
				vm, vmi = DefaultVirtualMachine(true)
				vm.ObjectMeta.UID = types.UID(uuid.NewString())
				vmi.ObjectMeta.UID = vm.ObjectMeta.UID
				vm.Generation = 1
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					Cores: 2,
				}
				guest := resource.MustParse("128Mi")
				vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
					Guest: &guest,
				}
				kv = &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{},
							VMRolloutStrategy: &v1.VMRolloutStrategy{
								LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{},
							},
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
							},
						},
					},
				}
			})

			AfterEach(func() {
				k8sClient.Fake.ClearActions()
			})

			It("should appear when changing a non-live-updatable field", func() {
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

				By("Creating a VM with hostname a")
				vm.Spec.Template.Spec.Hostname = "a"
				addVirtualMachine(vm)

				By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
				vmi = controller.setupVMIFromVM(vm)
				expectVMICreation()
				expectVMStatusUpdate()
				expectControllerRevisionList()
				expectControllerRevisionCreation()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeFalse())
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				By("Bumping the VM sockets above the cluster maximum")
				vm.Spec.Template.Spec.Hostname = "b"
				vm.Generation = 2
				modifyVirtualMachine(vm)

				By("Executing the controller again expecting the RestartRequired condition to appear")
				expectControllerRevisionDelete()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeTrue())
			})

			It("should appear when VM doesn't specify maxSockets and sockets go above cluster-wide maxSockets", func() {
				var maxSockets uint32 = 8

				By("Setting a cluster-wide CPU maxSockets value")
				kv.Spec.Configuration.LiveUpdateConfiguration.MaxCpuSockets = kvpointer.P(maxSockets)
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

				By("Creating a VM with CPU sockets set to the cluster maxiumum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 8
				addVirtualMachine(vm)

				By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
				vmi = controller.setupVMIFromVM(vm)
				expectVMICreation()
				expectVMStatusUpdate()
				expectControllerRevisionList()
				expectControllerRevisionCreation()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeFalse())
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				By("Bumping the VM sockets above the cluster maximum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 10
				vm.Generation = 2
				modifyVirtualMachine(vm)

				By("Executing the controller again expecting the RestartRequired condition to appear")
				expectVMUpdate()
				expectControllerRevisionDelete()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeTrue())
			})

			It("should appear when VM doesn't specify maxGuest and guest memory goes above cluster-wide maxGuest", func() {
				var maxGuest = resource.MustParse("256Mi")

				By("Setting a cluster-wide CPU maxGuest value")
				kv.Spec.Configuration.LiveUpdateConfiguration.MaxGuest = &maxGuest
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

				By("Creating a VM with guest memory set to the cluster maximum")
				vm.Spec.Template.Spec.Domain.Memory.Guest = &maxGuest
				addVirtualMachine(vm)

				By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
				vmi = controller.setupVMIFromVM(vm)
				expectVMICreation()
				expectVMStatusUpdate()
				expectControllerRevisionList()
				expectControllerRevisionCreation()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeFalse())
				markAsReady(vmi)
				vmi.Status.Memory = &virtv1.MemoryStatus{
					GuestAtBoot:  &maxGuest,
					GuestCurrent: &maxGuest,
				}
				vmiFeeder.Add(vmi)

				By("Bumping the VM guest memory above the cluster maximum")
				bigGuest := resource.MustParse("257Mi")
				vm.Spec.Template.Spec.Domain.Memory.Guest = &bigGuest
				vm.Generation = 2
				modifyVirtualMachine(vm)

				By("Executing the controller again expecting the RestartRequired condition to appear")
				expectVMUpdate()
				expectControllerRevisionDelete()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(restartRequired[vm.UID]).Should(BeTrue())
			})

			DescribeTable("when changing a live-updatable field", func(fgs []string, strat *virtv1.VMRolloutStrategy, expectCond bool) {
				kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = fgs
				kv.Spec.Configuration.VMRolloutStrategy = strat
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

				By("Creating a VM with CPU sockets set to the cluster maximum")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 2
				addVirtualMachine(vm)

				By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
				vmi = controller.setupVMIFromVM(vm)
				expectVMICreation()
				expectVMStatusUpdate()
				expectControllerRevisionList()
				expectControllerRevisionCreation()
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(func() {
					Expect(crListFor(string(vm.ObjectMeta.UID))).To(HaveLen(2))
				})
				Eventually(restartRequired[vm.UID]).Should(BeFalse())
				markAsReady(vmi)
				vmiFeeder.Add(vmi)

				By("Bumping the VM sockets to a reasonable value")
				vm.Spec.Template.Spec.Domain.CPU.Sockets = 4
				vm.Generation = 2
				modifyVirtualMachine(vm)

				By("Executing the controller again expecting the RestartRequired condition to appear")
				expectControllerRevisionDelete()
				if !expectCond {
					expectVMIPatch()
				}
				controller.Execute()
				Expect(cache.WaitForCacheSync(stop, crInformer.HasSynced)).To(BeTrue())
				Eventually(func() {
					Expect(crListFor(string(vm.ObjectMeta.UID))).To(HaveLen(2))
				})
				Eventually(restartRequired[vm.UID]).Should(Equal(expectCond))
			},
				Entry("should appear if the feature gate is not set",
					[]string{},
					&virtv1.VMRolloutStrategy{LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{}},
					true),
				Entry("should appear if the VM rollout strategy is not set",
					[]string{virtconfig.VMLiveUpdateFeaturesGate},
					nil,
					true),
				Entry("should appear if the VM rollout strategy is set to Stage",
					[]string{virtconfig.VMLiveUpdateFeaturesGate},
					&virtv1.VMRolloutStrategy{Stage: &virtv1.RolloutStrategyStage{}},
					true),
				Entry("should not appear if both the VM rollout strategy and feature gate are set",
					[]string{virtconfig.VMLiveUpdateFeaturesGate},
					&virtv1.VMRolloutStrategy{LiveUpdate: &virtv1.RolloutStrategyLiveUpdate{}},
					false),
			)
		})
	})
	Context("syncConditions", func() {
		var vm *virtv1.VirtualMachine
		var vmi *virtv1.VirtualMachineInstance

		BeforeEach(func() {
			vm, vmi = DefaultVirtualMachineWithNames(false, "test", "test")
		})

		It("should set ready to false when VMI is nil", func() {
			syncConditions(vm, nil, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(virtv1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should set ready to false when VMI doesn't have a ready condition", func() {
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(virtv1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should set ready to false when VMI has a false ready condition", func() {
			vmi.Status.Conditions = []virtv1.VirtualMachineInstanceCondition{{
				Type:   virtv1.VirtualMachineInstanceReady,
				Status: k8sv1.ConditionFalse,
			}}
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(1))
			Expect(vm.Status.Conditions[0].Type).To(Equal(virtv1.VirtualMachineReady))
			Expect(vm.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should sync appropriate conditions and ignore others", func() {
			fromCondList := []virtv1.VirtualMachineConditionType{
				virtv1.VirtualMachineReady, virtv1.VirtualMachineFailure, virtv1.VirtualMachinePaused,
				virtv1.VirtualMachineInitialized, virtv1.VirtualMachineRestartRequired,
			}
			toCondList := []virtv1.VirtualMachineConditionType{
				virtv1.VirtualMachineReady, virtv1.VirtualMachinePaused,
			}
			vmi.Status.Conditions = []virtv1.VirtualMachineInstanceCondition{}
			for _, cond := range fromCondList {
				vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceConditionType(cond),
					Status: k8sv1.ConditionTrue,
				})
			}
			syncConditions(vm, vmi, nil)
			Expect(vm.Status.Conditions).To(HaveLen(len(toCondList)))
			for _, cond := range vm.Status.Conditions {
				Expect(toCondList).To(ContainElements(cond.Type))
			}
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
	vm.Finalizers = append(vm.Finalizers, virtv1.VirtualMachineControllerFinalizer)
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

func markVmAsReady(vm *virtv1.VirtualMachine) {
	virtcontroller.NewVirtualMachineConditionManager().UpdateCondition(vm, &virtv1.VirtualMachineCondition{Type: virtv1.VirtualMachineReady, Status: k8sv1.ConditionTrue})
}
