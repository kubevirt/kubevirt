package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	k8ssnapshotfake "kubevirt.io/client-go/externalsnapshotter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	testNamespace = "default"
	vmSnapshotUID = "snapshot-uid"
	contentUID    = "content-uid"
	diskName      = "disk1"
)

var (
	vmUID             types.UID        = "vm-uid"
	vmAPIGroup                         = "kubevirt.io"
	storageClassName                   = "rook-ceph-block"
	noFailureDeadline *metav1.Duration = &metav1.Duration{Duration: 0}
	instancetypeUID   types.UID        = "instancetype-uid"
	preferenceUID     types.UID        = "preference-uid"
)

var _ = Describe("Snapshot controlleer", func() {
	var (
		timeStamp                = metav1.Time{Time: time.Now().Truncate(time.Second)}
		vmName                   = "testvm"
		vmRevisionName           = "testvm-revision"
		vmSnapshotName           = "test-snapshot"
		retain                   = snapshotv1.VirtualMachineSnapshotContentRetain
		volumeSnapshotClassName  = "csi-rbdplugin-snapclass"
		volumeSnapshotClassName2 = "csi-rbdplugin-snapclass-alt"
	)

	timeFunc := func() *metav1.Time {
		return &timeStamp
	}

	createVMSnapshot := func() *snapshotv1.VirtualMachineSnapshot {
		return createVirtualMachineSnapshot(testNamespace, vmSnapshotName, vmName)
	}

	createVMSnapshotSuccess := func() *snapshotv1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse:   pointer.P(true),
			SourceUID:    &vmUID,
			CreationTime: timeFunc(),
			Conditions: []snapshotv1.Condition{
				newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
				newReadyCondition(corev1.ConditionTrue, "Ready"),
			},
			Phase: snapshotv1.Succeeded,
		}

		return vms
	}

	createVMSnapshotInProgress := func() *snapshotv1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse: pointer.P(false),
			SourceUID:  &vmUID,
			Phase:      snapshotv1.InProgress,
		}

		return vms
	}

	createVM := func() *v1.VirtualMachine {
		return createVirtualMachine(testNamespace, vmName)
	}

	createLockedVM := func() *v1.VirtualMachine {
		vm := createVM()
		vm.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
		vm.Status.SnapshotInProgress = &[]string{vmSnapshotName}[0]
		return vm
	}

	createVMI := func(vm *v1.VirtualMachine) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vm.Namespace,
				Name:      vm.Name,
			},
			Spec: vm.Spec.Template.Spec,
		}
	}

	getVMRevisionData := func(vm *v1.VirtualMachine) runtime.RawExtension {
		vmCpy := vm.DeepCopy()
		vmCpy.ResourceVersion = "1"
		vmCpy.Status = v1.VirtualMachineStatus{}
		data, err := json.Marshal(vmCpy)
		Expect(err).ToNot(HaveOccurred())
		return runtime.RawExtension{Raw: data}
	}

	createVMRevision := func(vm *v1.VirtualMachine) *appsv1.ControllerRevision {
		return &appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmRevisionName,
				Namespace: vm.Namespace,
			},
			Data:     getVMRevisionData(vm),
			Revision: 1,
		}
	}

	createPersistentVolumeClaims := func() []corev1.PersistentVolumeClaim {
		return createPVCsForVM(createLockedVM())
	}

	createVMSnapshotContent := func() *snapshotv1.VirtualMachineSnapshotContent {
		vmSnapshot := createVMSnapshotInProgress()
		vm := createLockedVM()
		pvcs := createPersistentVolumeClaims()
		return createVirtualMachineSnapshotContent(vmSnapshot, vm, pvcs)
	}

	createErrorVMSnapshotContent := func(errorMessage string) *snapshotv1.VirtualMachineSnapshotContent {
		content := createVMSnapshotContent()
		content.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
			ReadyToUse: pointer.P(false),
		}
		content.Status.Error = &snapshotv1.Error{
			Time:    timeFunc(),
			Message: &errorMessage,
		}
		return content
	}

	createReadyVMSnapshotContent := func() *snapshotv1.VirtualMachineSnapshotContent {
		content := createVMSnapshotContent()
		content.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
			ReadyToUse:   pointer.P(true),
			CreationTime: timeFunc(),
		}
		return content
	}

	createVMSnapshotErrored := func() *snapshotv1.VirtualMachineSnapshot {
		vms := createVMSnapshotInProgress()
		errorMessage := "VolumeSnapshot vol1 error: error"
		content := createErrorVMSnapshotContent(errorMessage)
		vms.Status.VirtualMachineSnapshotContentName = &content.Name
		vms.Status.ReadyToUse = pointer.P(false)
		vms.Status.Conditions = []snapshotv1.Condition{
			newProgressingCondition(corev1.ConditionFalse, "In error state"),
			newReadyCondition(corev1.ConditionFalse, "Not ready"),
		}
		vms.Status.Error = content.Status.Error
		return vms
	}

	createStorageProfile := func() *cdiv1.StorageProfile {
		return &cdiv1.StorageProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name: storageClassName,
			},
			Status: cdiv1.StorageProfileStatus{
				SnapshotClass: pointer.P(volumeSnapshotClassName2),
			},
		}
	}

	createStorageClass := func() *storagev1.StorageClass {
		return &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: storageClassName,
			},
			Provisioner: "rook-ceph.rbd.csi.ceph.com",
		}
	}

	createVolumeSnapshots := func(content *snapshotv1.VirtualMachineSnapshotContent) []vsv1.VolumeSnapshot {
		var volumeSnapshots []vsv1.VolumeSnapshot
		for _, vb := range content.Spec.VolumeBackups {
			if vb.VolumeSnapshotName == nil {
				continue
			}
			vs := vsv1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *vb.VolumeSnapshotName,
					Namespace: content.Namespace,
				},
				Spec: vsv1.VolumeSnapshotSpec{
					Source: vsv1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &vb.PersistentVolumeClaim.Name,
					},
				},
				Status: &vsv1.VolumeSnapshotStatus{
					ReadyToUse: pointer.P(false),
				},
			}
			volumeSnapshots = append(volumeSnapshots, vs)
		}
		return volumeSnapshots
	}

	createVolumeSnapshotClasses := func() []vsv1.VolumeSnapshotClass {
		return []vsv1.VolumeSnapshotClass{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: volumeSnapshotClassName,
					Annotations: map[string]string{
						"snapshot.storage.kubernetes.io/is-default-class": "",
					},
				},
				Driver: "rook-ceph.rbd.csi.ceph.com",
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: volumeSnapshotClassName2,
				},
				Driver: "rook-ceph.rbd.csi.ceph.com",
			},
		}
	}

	Context("One valid Snapshot controller given", func() {
		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVirtualMachineInterface
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		var vmSnapshotSource *framework.FakeControllerSource
		var vmSnapshotInformer cache.SharedIndexInformer
		var vmSnapshotContentSource *framework.FakeControllerSource
		var vmSnapshotContentInformer cache.SharedIndexInformer
		var vmInformer cache.SharedIndexInformer
		var vmSource *framework.FakeControllerSource
		var vmiInformer cache.SharedIndexInformer
		var vmiSource *framework.FakeControllerSource
		var podInformer cache.SharedIndexInformer
		var podSource *framework.FakeControllerSource
		var volumeSnapshotInformer cache.SharedIndexInformer
		var volumeSnapshotSource *framework.FakeControllerSource
		var volumeSnapshotClassInformer cache.SharedIndexInformer
		var volumeSnapshotClassSource *framework.FakeControllerSource
		var storageClassInformer cache.SharedIndexInformer
		var storageClassSource *framework.FakeControllerSource
		var storageProfileInformer cache.SharedIndexInformer
		var storageProfileSource *framework.FakeControllerSource
		var pvcInformer cache.SharedIndexInformer
		var pvcSource *framework.FakeControllerSource
		var crdInformer cache.SharedIndexInformer
		var crdSource *framework.FakeControllerSource
		var dvInformer cache.SharedIndexInformer
		var crInformer cache.SharedIndexInformer
		var crSource *framework.FakeControllerSource
		var dvSource *framework.FakeControllerSource
		var stop chan struct{}
		var controller *VMSnapshotController
		var recorder *record.FakeRecorder
		var mockVMSnapshotQueue *testutils.MockWorkQueue[string]
		var mockVMSnapshotContentQueue *testutils.MockWorkQueue[string]
		var mockCRDQueue *testutils.MockWorkQueue[string]
		var mockVMQueue *testutils.MockWorkQueue[string]

		var vmSnapshotClient *kubevirtfake.Clientset
		var k8sSnapshotClient *k8ssnapshotfake.Clientset
		var k8sClient *k8sfake.Clientset
		var virtClient *kubecli.MockKubevirtClient

		syncCaches := func(stop chan struct{}) {
			go vmSnapshotInformer.Run(stop)
			go vmSnapshotContentInformer.Run(stop)
			go vmInformer.Run(stop)
			go storageClassInformer.Run(stop)
			go storageProfileInformer.Run(stop)
			go pvcInformer.Run(stop)
			go crdInformer.Run(stop)
			go vmiInformer.Run(stop)
			go podInformer.Run(stop)
			go dvInformer.Run(stop)
			go crInformer.Run(stop)
			Expect(cache.WaitForCacheSync(
				stop,
				vmSnapshotInformer.HasSynced,
				vmSnapshotContentInformer.HasSynced,
				vmInformer.HasSynced,
				storageClassInformer.HasSynced,
				storageProfileInformer.HasSynced,
				pvcInformer.HasSynced,
				crdInformer.HasSynced,
				vmiInformer.HasSynced,
				podInformer.HasSynced,
				dvInformer.HasSynced,
				crInformer.HasSynced,
			)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

			vmSnapshotInformer, vmSnapshotSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineSnapshot{}, virtcontroller.GetVirtualMachineSnapshotInformerIndexers())
			vmSnapshotContentInformer, vmSnapshotContentSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineSnapshotContent{}, virtcontroller.GetVirtualMachineSnapshotContentInformerIndexers())
			crInformer, crSource = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, virtcontroller.GetControllerRevisionInformerIndexers())
			vmInformer, vmSource = testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
			vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&v1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
			podInformer, podSource = testutils.NewFakeInformerFor(&corev1.Pod{})
			volumeSnapshotInformer, volumeSnapshotSource = testutils.NewFakeInformerFor(&vsv1.VolumeSnapshot{})
			volumeSnapshotClassInformer, volumeSnapshotClassSource = testutils.NewFakeInformerFor(&vsv1.VolumeSnapshotClass{})
			storageClassInformer, storageClassSource = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			storageProfileInformer, storageProfileSource = testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
			pvcInformer, pvcSource = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			crdInformer, crdSource = testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
			dvInformer, dvSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			controller = &VMSnapshotController{
				Client:                    virtClient,
				VMSnapshotInformer:        vmSnapshotInformer,
				VMSnapshotContentInformer: vmSnapshotContentInformer,
				VMInformer:                vmInformer,
				VMIInformer:               vmiInformer,
				PodInformer:               podInformer,
				StorageClassInformer:      storageClassInformer,
				StorageProfileInformer:    storageProfileInformer,
				PVCInformer:               pvcInformer,
				CRDInformer:               crdInformer,
				DVInformer:                dvInformer,
				CRInformer:                crInformer,
				Recorder:                  recorder,
				ResyncPeriod:              60 * time.Second,
			}
			controller.Init()

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockVMSnapshotQueue = testutils.NewMockWorkQueue(controller.vmSnapshotQueue)
			controller.vmSnapshotQueue = mockVMSnapshotQueue

			mockVMSnapshotContentQueue = testutils.NewMockWorkQueue(controller.vmSnapshotContentQueue)
			controller.vmSnapshotContentQueue = mockVMSnapshotContentQueue

			mockCRDQueue = testutils.NewMockWorkQueue(controller.crdQueue)
			controller.crdQueue = mockCRDQueue

			mockVMQueue = testutils.NewMockWorkQueue(controller.vmQueue)
			controller.vmQueue = mockVMQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface).AnyTimes()

			vmSnapshotClient = kubevirtfake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineSnapshot(testNamespace).
				Return(vmSnapshotClient.SnapshotV1beta1().VirtualMachineSnapshots(testNamespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshotContent(testNamespace).
				Return(vmSnapshotClient.SnapshotV1beta1().VirtualMachineSnapshotContents(testNamespace)).AnyTimes()

			k8sSnapshotClient = k8ssnapshotfake.NewSimpleClientset()
			virtClient.EXPECT().KubernetesSnapshotClient().Return(k8sSnapshotClient).AnyTimes()

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().StorageV1().Return(k8sClient.StorageV1()).AnyTimes()
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

			k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			k8sSnapshotClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			vmSnapshotClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			currentTime = timeFunc
		})

		addVirtualMachineSnapshot := func(s *snapshotv1.VirtualMachineSnapshot) {
			syncCaches(stop)
			mockVMSnapshotQueue.ExpectAdds(1)
			vmSnapshotSource.Add(s)
			mockVMSnapshotQueue.Wait()
		}

		addVirtualMachineSnapshotContent := func(s *snapshotv1.VirtualMachineSnapshotContent) {
			syncCaches(stop)
			mockVMSnapshotContentQueue.ExpectAdds(1)
			vmSnapshotContentSource.Add(s)
			mockVMSnapshotContentQueue.Wait()
		}

		addVolumeSnapshot := func(s *vsv1.VolumeSnapshot) {
			syncCaches(stop)
			mockVMSnapshotContentQueue.ExpectAdds(1)
			volumeSnapshotSource.Add(s)
			mockVMSnapshotContentQueue.Wait()
		}

		addCRD := func(crd *extv1.CustomResourceDefinition) {
			syncCaches(stop)
			mockCRDQueue.ExpectAdds(1)
			crdSource.Add(crd)
			mockCRDQueue.Wait()
		}

		addVolumeSnapshotClass := func(volumeSnapshotClasses ...vsv1.VolumeSnapshotClass) {
			syncCaches(stop)
			for i := range volumeSnapshotClasses {
				mockVMQueue.ExpectAdds(1)
				volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				mockVMQueue.Wait()
			}
		}

		Context("with VolumeSnapshot and VolumeSnapshotContent informers", func() {
			BeforeEach(func() {
				stopCh := make(chan struct{})
				volumeSnapshotInformer.AddEventHandler(controller.eventHandlerMap[volumeSnapshotCRD])
				volumeSnapshotClassInformer.AddEventHandler(controller.eventHandlerMap[volumeSnapshotClassCRD])
				controller.dynamicInformerMap[volumeSnapshotCRD].stopCh = stopCh
				controller.dynamicInformerMap[volumeSnapshotCRD].informer = volumeSnapshotInformer
				controller.dynamicInformerMap[volumeSnapshotClassCRD].stopCh = stopCh
				controller.dynamicInformerMap[volumeSnapshotClassCRD].informer = volumeSnapshotClassInformer
				go volumeSnapshotInformer.Run(stopCh)
				go volumeSnapshotClassInformer.Run(stopCh)
				Expect(cache.WaitForCacheSync(
					stopCh,
					volumeSnapshotInformer.HasSynced,
					volumeSnapshotClassInformer.HasSynced,
				)).To(BeTrue())

				pvcs := createPersistentVolumeClaims()
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}
			})

			It("should initialize VirtualMachineSnapshot status", func() {
				vmSnapshot := createVMSnapshot()
				vm := createVM()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, vmSnapshot, updatedSnapshot)
				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot2)
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should initialize VirtualMachineSnapshot status (no VM)", func() {
				vmSnapshot := createVMSnapshot()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, vmSnapshot, updatedSnapshot)
				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Source does not exist"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot2)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should initialize VirtualMachineSnapshot status (Progressing)", func() {
				vmSnapshot := createVMSnapshot()
				vm := createLockedVM()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, vmSnapshot, updatedSnapshot)
				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				vmSource.Add(vm)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot2)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should unlock source VirtualMachine", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vm := createLockedVM()
				updatedVM := vm.DeepCopy()
				updatedVM.Finalizers = []string{}
				updatedVM.ResourceVersion = "1"
				vmSource.Add(vm)

				patchBytes, err := patch.GenerateTestReplacePatch("/metadata/finalizers", []string{"snapshot.kubevirt.io/snapshot-source-protection"}, []string{})
				Expect(err).ToNot(HaveOccurred())
				vmInterface.EXPECT().Patch(context.Background(), updatedVM.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}).Return(updatedVM, nil).Times(1)

				statusUpdate := updatedVM.DeepCopy()
				statusUpdate.Status.SnapshotInProgress = nil
				vmInterface.EXPECT().UpdateStatus(context.Background(), statusUpdate, metav1.UpdateOptions{}).Return(statusUpdate, nil).Times(1)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("shouldn't unlock if snapshot deleting before completed and content not deleted", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vm := createLockedVM()
				vmSource.Add(vm)
				content := createVMSnapshotContent()
				vmSnapshotContentSource.Add(content)
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Phase = snapshotv1.Deleting
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "VM snapshot is deleting"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				addVirtualMachineSnapshot(vmSnapshot)
				contentDeletes := expectVMSnapshotContentDelete(vmSnapshotClient, content.Name)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*contentDeletes).To(Equal(1))
			})

			It("should finish unlock source VirtualMachine", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vm := createLockedVM()
				vm.Finalizers = []string{}
				statusUpdate := vm.DeepCopy()
				statusUpdate.ResourceVersion = "1"
				statusUpdate.Status.SnapshotInProgress = nil
				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(context.Background(), statusUpdate, metav1.UpdateOptions{}).Return(statusUpdate, nil).Times(1)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("cleanup when VirtualMachineSnapshot is deleted", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vmSnapshot.DeletionTimestamp = timeFunc()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
					IncludedVolumes: []string{diskName},
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Finalizers = []string{}

				content := createVMSnapshotContent()

				vmSnapshotContentSource.Add(content)
				contentDeletes := expectVMSnapshotContentDelete(vmSnapshotClient, content.Name)
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, updatedSnapshot, updatedSnapshot2)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*contentDeletes).To(Equal(1))
			})

			It("should not delete content when VirtualMachineSnapshot is deleted, content ready and retain policy", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vmSnapshot.Spec.DeletionPolicy = &retain
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
					IncludedVolumes: []string{diskName},
				}

				content := createReadyVMSnapshotContent()

				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &content.Name

				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Finalizers = []string{}

				vmSnapshotContentSource.Add(content)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, updatedSnapshot, updatedSnapshot2)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should delete content when VirtualMachineSnapshot is deleted, content not ready even if retain policy", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vmSnapshot.Spec.DeletionPolicy = &retain
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
					IncludedVolumes: []string{diskName},
				}

				updatedSnapshot2 := updatedSnapshot.DeepCopy()
				updatedSnapshot2.Finalizers = []string{}

				content := createVMSnapshotContent()

				vmSnapshotContentSource.Add(content)
				contentDeletes := expectVMSnapshotContentDelete(vmSnapshotClient, content.Name)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				patchCalls := expectVMSnapshotPatch(vmSnapshotClient, updatedSnapshot, updatedSnapshot2)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*patchCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*contentDeletes).To(Equal(1))
			})

			DescribeTable("should partial lock source if VM patch fails", func(runstrategy v1.VirtualMachineRunStrategy) {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.RunStrategy = pointer.P(runstrategy)
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

				vmInterface.EXPECT().UpdateStatus(context.Background(), vmUpdate, metav1.UpdateOptions{}).Return(vmUpdate, nil).Times(1)
				vmInterface.EXPECT().Patch(context.Background(), vmUpdate.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).Return(nil, fmt.Errorf("error")).Times(1)

				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			},
				Entry("With runstrategy halted", v1.RunStrategyHalted),
				Entry("With runstrategy always", v1.RunStrategyAlways),
				Entry("With runstrategy manual", v1.RunStrategyManual),
				Entry("With runstrategy rerunOnFailure", v1.RunStrategyRerunOnFailure),
			)

			It("should complete lock source after partial lock", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				// Update of snapshotinprogress succeeded already
				vm.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSource.Add(vm)

				patchBytes, err := patch.GenerateTestReplacePatch("/metadata/finalizers", nil, []string{"snapshot.kubevirt.io/snapshot-source-protection"})
				Expect(err).ToNot(HaveOccurred())
				vmInterface.EXPECT().Patch(context.Background(), vmUpdate.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = nil
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			DescribeTable("should complete lock source", func(vmiExists bool) {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

				vmSource.Add(vm)
				if vmiExists {
					vmi := createVMI(vm)
					vmiSource.Add(vmi)
				}
				vmInterface.EXPECT().UpdateStatus(context.Background(), vmUpdate, metav1.UpdateOptions{}).Return(vmUpdate, nil).Times(1)

				vmUpdate2 := vmUpdate.DeepCopy()
				vmUpdate2.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
				patchBytes, err := patch.GenerateTestReplacePatch("/metadata/finalizers", nil, []string{"snapshot.kubevirt.io/snapshot-source-protection"})
				Expect(err).ToNot(HaveOccurred())
				vmInterface.EXPECT().Patch(context.Background(), vmUpdate2.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}).Return(vmUpdate2, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = nil
				if vmiExists {
					updatedSnapshot.Status.Indications = []snapshotv1.Indication{
						snapshotv1.VMSnapshotNoGuestAgentIndication,
						snapshotv1.VMSnapshotOnlineSnapshotIndication,
					}
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("when vm running", true),
				Entry("when vm not running", false),
			)

			DescribeTable("should not lock source if volume PVCs", func(createPVCs, boundPVCs bool, expectedReason string) {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmSource.Add(vm)

				pvcs := createPersistentVolumeClaims()
				for i := range pvcs {
					if createPVCs {
						if !boundPVCs {
							pvcs[i].Status.Phase = corev1.ClaimPending
						} else {
							dv := &cdiv1.DataVolume{
								ObjectMeta: metav1.ObjectMeta{
									Name:      pvcs[i].Name,
									Namespace: pvcs[i].Namespace,
								},
								Status: cdiv1.DataVolumeStatus{
									Phase: cdiv1.WaitForFirstConsumer,
								},
							}
							dvSource.Add(dv)
							pvcs[i].OwnerReferences = []metav1.OwnerReference{
								*metav1.NewControllerRef(dv, schema.GroupVersionKind{
									Group:   cdiv1.SchemeGroupVersion.Group,
									Version: cdiv1.SchemeGroupVersion.Version,
									Kind:    "DataVolume",
								}),
							}
						}
						pvcSource.Add(&pvcs[i])
					} else {
						pvcSource.Delete(&pvcs[i])
					}
				}

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				expectedReason = fmt.Sprintf("Source not locked source default/testvm %s: alpine-dv", expectedReason)
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, expectedReason),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = nil
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("doesnt exist", false, false, "volume doesnt exist"),
				Entry("are not bound", true, false, "volume not bound"),
				Entry("are not populated", true, true, "volume not populated"),
			)

			It("should not lock source if pods using PVCs when vm not running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()

				pods := createPodsUsingPVCs(vm)
				podSource.Add(&pods[0])
				vmSource.Add(vm)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked source is offline but 1 pods using PVCs [alpine-dv]"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = nil
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should (partial) lock source if pods using PVCs if VM is running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
				vmSource.Add(vm)
				vmiSource.Add(createVMI(vm))
				pods := createPodsUsingPVCs(vm)
				podSource.Add(&pods[0])

				vmStatusUpdate := vm.DeepCopy()
				vmStatusUpdate.ResourceVersion = "1"
				vmStatusUpdate.Status.SnapshotInProgress = &vmSnapshotName
				vmInterface.EXPECT().UpdateStatus(context.Background(), vmStatusUpdate, metav1.UpdateOptions{}).Return(vmStatusUpdate, nil).Times(1)
				vmInterface.EXPECT().Patch(context.Background(), vmStatusUpdate.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).Return(nil, fmt.Errorf("error")).Times(1)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should not lock source if another snapshot in progress", func() {
				n := "otherSnapshot"
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Status.SnapshotInProgress = &n
				vmSource.Add(vm)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.Phase = snapshotv1.InProgress
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, fmt.Sprintf("Source not locked snapshot %q in progress", n)),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = nil
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should create VirtualMachineSnapshotContent", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				storageClass := createStorageClass()
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				vmSnapshotContent := createVMSnapshotContent()

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				createCalls := expectVMSnapshotContentCreate(vmSnapshotClient, vmSnapshotContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*createCalls).To(Equal(1))
			})

			It("create VirtualMachineSnapshotContent online snapshot", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
				vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				}

				vmRevision := createVMRevision(vm)
				crSource.Add(vmRevision)
				vmi := createVMI(vm)
				vmi.Status.VirtualMachineRevisionName = vmRevisionName
				vmiSource.Add(vmi)

				vm.ObjectMeta.Annotations = map[string]string{}
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "disk2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						}},
					},
				})
				// the content source will have the a combination of the vm revision, the vmi and the vm volumes
				vm.ObjectMeta.Generation = 2
				pvcs := createPersistentVolumeClaims()

				expectedContent := createVirtualMachineSnapshotContent(vmSnapshot, vm, pvcs)
				vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
				vmSource.Add(vm)
				storageClass := createStorageClass()
				storageClassSource.Add(storageClass)
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				createCalls := expectVMSnapshotContentCreate(vmSnapshotClient, expectedContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotNoGuestAgentIndication,
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*createCalls).To(Equal(1))
			})

			It("should create VirtualMachineSnapshotContent with memory dump", func() {
				storageClass := createStorageClass()
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]

				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				vm = updateVMWithMemoryDump(vm)
				pvcs := createPersistentVolumeClaims()
				md := memoryDumpPVC()
				pvcSource.Add(&md)
				pvcs = append(pvcs, md)
				vmSnapshotContent := createVirtualMachineSnapshotContent(vmSnapshot, vm, pvcs)

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				createCalls := expectVMSnapshotContentCreate(vmSnapshotClient, vmSnapshotContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: pointer.P(false),
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*createCalls).To(Equal(1))
			})

			It("should update VirtualMachineSnapshotStatus", func() {
				vmSnapshotContent := createReadyVMSnapshotContent()

				vmSnapshot := createVMSnapshotInProgress()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SourceUID = &vmUID
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.CreationTime = timeFunc()
				updatedSnapshot.Status.ReadyToUse = pointer.P(true)
				updatedSnapshot.Status.Phase = snapshotv1.Succeeded
				updatedSnapshot.Status.Indications = nil
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Ready"),
				}
				updatedSnapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
					IncludedVolumes: []string{diskName},
				}

				vm := createLockedVM()

				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update included and excluded volume in VirtualMachineSnapshotStatus", func() {
				vmSnapshotContent := createReadyVMSnapshotContent()
				vm := createLockedVM()
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "disk2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						}},
					},
				})
				// Add volume to vm in content that is not in the volumebackup list
				// this is a simulation for a volume that is not included in the snapshot
				vmSnapshotContent.Spec.Source.VirtualMachine.Spec = *vm.Spec.DeepCopy()

				vmSnapshot := createVMSnapshotInProgress()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SourceUID = &vmUID
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.CreationTime = timeFunc()
				updatedSnapshot.Status.ReadyToUse = pointer.P(true)
				updatedSnapshot.Status.Phase = snapshotv1.Succeeded
				updatedSnapshot.Status.Indications = nil
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Ready"),
				}
				updatedSnapshot.Status.SnapshotVolumes = &snapshotv1.SnapshotVolumesLists{
					IncludedVolumes: []string{diskName},
					ExcludedVolumes: []string{"disk2"},
				}

				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update VirtualMachineSnapshot error when VirtualMachineSnapshotContent error", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				errorMessage := "VolumeSnapshot vol1 error: error"
				vmSnapshotContent := createErrorVMSnapshotContent(errorMessage)

				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.ReadyToUse = pointer.P(false)
				updatedSnapshot.Status.Indications = nil
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "In error state"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Error = vmSnapshotContent.Status.Error

				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update VirtualMachineSnapshot deadline exceeded after error phase", func() {
				vmSnapshot := createVMSnapshotErrored()
				negativeDeadline, _ := time.ParseDuration("-1m")
				vmSnapshot.Spec.FailureDeadline = &metav1.Duration{Duration: negativeDeadline}
				vm := createLockedVM()
				errorMessage := "VolumeSnapshot vol1 error: error"
				vmSnapshotContent := createErrorVMSnapshotContent(errorMessage)

				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.Phase = snapshotv1.Failed
				updatedSnapshot.Status.Conditions = append(updatedSnapshot.Status.Conditions,
					newFailureCondition(corev1.ConditionTrue, vmSnapshotDeadlineExceededError))

				contentDeletes := expectVMSnapshotContentDelete(vmSnapshotClient, vmSnapshotContent.Name)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*contentDeletes).To(Equal(1))
			})

			It("should remove error if vmsnapshotcontent not in error anymore", func() {
				vmSnapshot := createVMSnapshotErrored()
				vm := createLockedVM()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}
				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.Phase = snapshotv1.InProgress
				updatedSnapshot.Status.Error = nil
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}

				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("shouldn't timeout if VirtualMachineSnapshot succeeded", func() {
				vmSnapshot := createVMSnapshotSuccess()
				negativeDeadline, _ := time.ParseDuration("-1m")
				vmSnapshot.Spec.FailureDeadline = &metav1.Duration{Duration: negativeDeadline}
				vm := createVM()

				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)

				controller.processVMSnapshotWorkItem()
			})

			It("should timeout if VirtualMachineSnapshot passed failure deadline", func() {
				vmSnapshot := createVMSnapshotInProgress()
				negativeDeadline, _ := time.ParseDuration("-1m")
				vmSnapshot.Spec.FailureDeadline = &metav1.Duration{Duration: negativeDeadline}
				vm := createLockedVM()
				vmSnapshotContent := createVMSnapshotContent()

				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.Phase = snapshotv1.Failed
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newFailureCondition(corev1.ConditionTrue, vmSnapshotDeadlineExceededError),
					newProgressingCondition(corev1.ConditionFalse, "Operation failed"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}

				contentDeletes := expectVMSnapshotContentDelete(vmSnapshotClient, vmSnapshotContent.Name)
				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*contentDeletes).To(Equal(1))
			})

			It("should create VolumeSnapshot", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)

				snapshotCreates := expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*snapshotCreates).To(Equal(1))
			})

			It("should create VolumeSnapshot with multiple VolumeSnapshotClasses", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)

				snapshotCreates := expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClasses[0].Name, vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				addVolumeSnapshotClass(volumeSnapshotClasses...)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*snapshotCreates).To(Equal(1))
			})

			It("should create VolumeSnapshot with multiple VolumeSnapshotClasses and storageprofile", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				storageProfile := createStorageProfile()
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				storageProfileSource.Add(storageProfile)

				snapshotCreates := expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClassName2, vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				addVolumeSnapshotClass(volumeSnapshotClasses...)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*snapshotCreates).To(Equal(1))
			})

			It("should create VolumeSnapshot with online snapshot no guest agent", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)

				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				storageClassSource.Add(storageClass)

				snapshotCreates := expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*snapshotCreates).To(Equal(1))
			})

			It("should set content error if failed freeze vm", func() {
				storageClass := createStorageClass()
				storageClassSource.Add(storageClass)

				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotSource.Add(vmSnapshot)

				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vmSnapshotContentSource.Add(vmSnapshotContent)

				vm := createLockedVM()
				vmSource.Add(vm)

				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				errorMessage := "Failed freezing vm"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
					Error: &snapshotv1.Error{
						Time:    timeFunc(),
						Message: &errorMessage,
					},
				}

				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				addVolumeSnapshotClass(volumeSnapshotClass)

				vmiInterface.EXPECT().Freeze(context.Background(), vm.Name, 0*time.Second).Return(fmt.Errorf(errorMessage)).Times(1)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)

				controller.processVMSnapshotContentWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should set QuiesceFailed indication if failed freeze vm", func() {
				vm := createLockedVM()
				vmSource.Add(vm)
				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				errorMessage := "Failed freezing vm"
				vmSnapshotContent := createErrorVMSnapshotContent(errorMessage)
				vmSnapshotContentSource.Add(vmSnapshotContent)

				vmSnapshot := createVMSnapshotInProgress()
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.ReadyToUse = pointer.P(false)
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotGuestAgentIndication,
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotQuiesceFailedIndication,
				}
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "In error state"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Error = vmSnapshotContent.Status.Error

				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should not unset QuiesceFailed indication if freeze succeeded afterwards", func() {
				vm := createLockedVM()
				vmSource.Add(vm)
				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				vmSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotGuestAgentIndication,
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotQuiesceFailedIndication,
				}
				addVirtualMachineSnapshot(vmSnapshot)

				updateStatusCalls := expectVMSnapshotUpdateStatus(vmSnapshotClient, vmSnapshot)

				controller.processVMSnapshotWorkItem()
				Expect(*updateStatusCalls).To(Equal(0))
			})

			It("should freeze vm with online snapshot and guest agent", func() {
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vm := createLockedVM()
				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)

				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				storageClassSource.Add(storageClass)

				vmiInterface.EXPECT().Freeze(context.Background(), vm.Name, 0*time.Second).Return(nil).Times(1)
				snapshotCreates := expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*snapshotCreates).To(Equal(1))
			})

			DescribeTable("should update VirtualMachineSnapshotContent", func(readyToUse bool) {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &readyToUse,
					CreationTime: timeFunc(),
				}

				vmSnapshotSource.Add(vmSnapshot)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					volumeSnapshots[i].Status.ReadyToUse = &readyToUse
					volumeSnapshots[i].Status.CreationTime = timeFunc()
					addVolumeSnapshot(&volumeSnapshots[i])

					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
						ReadyToUse:         volumeSnapshots[i].Status.ReadyToUse,
						CreationTime:       volumeSnapshots[i].Status.CreationTime,
						Error:              translateError(volumeSnapshots[i].Status.Error),
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				controller.processVMSnapshotContentWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("not ready", false),
				Entry("ready", true),
			)

			It("should update VirtualMachineSnapshotContent no snapshots", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				for i := range vmSnapshotContent.Spec.VolumeBackups {
					vmSnapshotContent.Spec.VolumeBackups[i].VolumeSnapshotName = nil
				}

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   pointer.P(true),
					CreationTime: timeFunc(),
				}

				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			DescribeTable("should update VirtualMachineSnapshotContent on error", func(rtu bool, ct *metav1.Time) {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &rtu,
					CreationTime: ct,
				}

				vmSnapshotSource.Add(vmSnapshot)
				vmSnapshotContentSource.Add(vmSnapshotContent)

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					message := "bad error"
					volumeSnapshots[i].Status.ReadyToUse = &rtu
					volumeSnapshots[i].Status.CreationTime = ct
					volumeSnapshots[i].Status.Error = &vsv1.VolumeSnapshotError{
						Message: &message,
						Time:    timeFunc(),
					}
					addVolumeSnapshot(&volumeSnapshots[i])

					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
						ReadyToUse:         volumeSnapshots[i].Status.ReadyToUse,
						CreationTime:       volumeSnapshots[i].Status.CreationTime,
						Error:              translateError(volumeSnapshots[i].Status.Error),
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)

					if i == 0 && !rtu {
						errorMessage := fmt.Sprintf("VolumeSnapshot %s error: %s", vss.VolumeSnapshotName, *vss.Error.Message)
						updatedContent.Status.Error = &snapshotv1.Error{
							Time:    timeFunc(),
							Message: &errorMessage,
						}
					}
				}
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)

				controller.processVMSnapshotContentWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("not ready", false, nil),
				Entry("ready", true, timeFunc()),
			)

			It("should update VirtualMachineSnapshotContent when VolumeSnapshot deleted", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createReadyVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status.ReadyToUse = pointer.P(false)
				updatedContent.Status.CreationTime = nil

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				var volumeSnapshotNames []string
				for i := range volumeSnapshots {
					volumeSnapshotNames = append(volumeSnapshotNames, volumeSnapshots[i].Name)
				}
				errorMessage := fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(volumeSnapshotNames, ","))
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   pointer.P(false),
					CreationTime: timeFunc(),
					Error: &snapshotv1.Error{
						Message: &errorMessage,
						Time:    timeFunc(),
					},
				}

				vmSnapshotSource.Add(vmSnapshot)
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "VolumeSnapshotMissing")
				Expect(*updateStatusCalls).To(Equal(1))
			})

			DescribeTable("should unfreeze vm with online snapshot and guest agent", func(ct *metav1.Time, r bool) {
				storageClass := createStorageClass()
				storageClassSource.Add(storageClass)
				volumeSnapshotClass := createVolumeSnapshotClasses()[0]

				vm := createLockedVM()
				vmSource.Add(vm)
				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}
				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					volumeSnapshots[i].Status.ReadyToUse = &r
					volumeSnapshots[i].Status.CreationTime = ct
					volumeSnapshotSource.Add(&volumeSnapshots[i])
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					vmSnapshotContent.Status.VolumeSnapshotStatus = append(vmSnapshotContent.Status.VolumeSnapshotStatus, vss)
				}

				vmSnapshotContentSource.Add(vmSnapshotContent)

				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)

				updatedContent := createVMSnapshotContent()
				updatedContent.UID = contentUID
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: ct,
					ReadyToUse:   &r,
				}
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
						ReadyToUse:         volumeSnapshots[i].Status.ReadyToUse,
						CreationTime:       volumeSnapshots[i].Status.CreationTime,
						Error:              translateError(volumeSnapshots[i].Status.Error),
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				if ct != nil {
					vmiInterface.EXPECT().Unfreeze(context.Background(), vm.Name).Return(nil).Times(1)
				}
				updateStatusCalls := expectVMSnapshotContentUpdateStatus(vmSnapshotClient, updatedContent)
				vmSnapshotSource.Add(vmSnapshot)
				addVolumeSnapshotClass(volumeSnapshotClass)
				controller.processVMSnapshotContentWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("not created and not ready", nil, false),
				Entry("created and not ready", timeFunc(), false),
				Entry("created and ready", timeFunc(), true),
			)

			DescribeTable("should attempt to unfreeze vm and remove content finalizer if vmsnapshot deleting", func(unfreezeError error) {
				vm := createLockedVM()
				vmSource.Add(vm)
				vmi := createVMI(vm)
				agentCondition := v1.VirtualMachineInstanceCondition{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        corev1.ConditionTrue,
				}
				vmi.Status.Conditions = append(vmi.Status.Conditions, agentCondition)
				vmiSource.Add(vmi)

				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: pointer.P(false),
				}

				vmSnapshotContentSource.Add(vmSnapshotContent)

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Finalizers = []string{}

				vmiInterface.EXPECT().Unfreeze(context.Background(), vm.Name).Return(unfreezeError).Times(1)
				patchCalls := expectVMSnapshotContentPatch(vmSnapshotClient, vmSnapshotContent, updatedContent)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotContentWorkItem()
				Expect(*patchCalls).To(Equal(1))
			},
				Entry("UnFreeze success", nil),
				Entry("UnFreeze error", fmt.Errorf("error")),
			)

			DescribeTable("should delete informer", func(crdName string) {
				crd := &extv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name:              crdName,
						DeletionTimestamp: timeFunc(),
					},
					Spec: extv1.CustomResourceDefinitionSpec{
						Versions: []extv1.CustomResourceDefinitionVersion{
							{
								Name:   "v1",
								Served: true,
							},
						},
					},
				}

				addCRD(crd)
				controller.processCRDWorkItem()
				Expect(controller.dynamicInformerMap[crdName].stopCh).Should(BeNil())
				Expect(controller.dynamicInformerMap[crdName].informer).Should(BeNil())
			},
				Entry("for VolumeSnapshot", volumeSnapshotCRD),
				Entry("for VolumeSnapshotClass", volumeSnapshotClassCRD),
			)

			It("should update volume snapshot status for each volume", func() {
				vm := createVM()
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "disk2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						}},
					},
				})

				updateCalled := false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(2))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[1].Enabled).To(BeFalse())
						updateCalled = true
					})

				vmSource.Add(vm)
				syncCaches(stop)
				mockVMQueue.Add(fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()

				Expect(updateCalled).To(BeTrue())
			})

			It("should set volume snapshot status to true for each supported volume type", func() {
				vm := createVM()
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "disk2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						}},
					},
				})

				pvc1 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &storageClassName,
					},
				}
				dv1 := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpine-dv",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: &storageClassName,
						},
					},
				}
				pvcSource.Add(&pvc1)
				dvSource.Add(&dv1)
				storageClassSource.Add(createStorageClass())
				vmSource.Add(vm)
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}
				mockVMQueue.Add(fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))
				syncCaches(stop)

				updateCalled := false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(2))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeTrue())
						Expect(vm.Status.VolumeSnapshotStatuses[1].Enabled).To(BeTrue())
						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()

				Expect(updateCalled).To(BeTrue())
			})

			It("should set volume snapshot status to false for unsupported volume types", func() {
				localStorageClassName := "local"

				vm := createVM()
				vm.Spec.Template.Spec.Volumes = []v1.Volume{
					{
						Name: "disk1",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc-unsnapshottable-storage-class",
							}},
						},
					},
					{
						Name: "disk2",
						VolumeSource: v1.VolumeSource{
							Ephemeral: &v1.EphemeralVolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc-ephemeral-pvc",
								},
							},
						},
					},
					{
						Name: "disk3",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv-with-pvc-unsnapshottable-storage-class",
							},
						},
					},
					{
						Name: "disk4",
						VolumeSource: v1.VolumeSource{
							HostDisk: &v1.HostDisk{
								Path:   "/some/path",
								Type:   "test-type",
								Shared: nil,
							},
						},
					},
					{
						Name: "disk5",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv-without-pvc-and-storageclass",
							},
						},
					},
					{
						Name: "disk6",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "non-existent-pvc",
							}},
						},
					},
					{
						Name: "disk7",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc-without-storage-class",
							}},
						},
					},
					{
						Name: "disk8",
						VolumeSource: v1.VolumeSource{
							DataVolume: &v1.DataVolumeSource{
								Name: "dv-with-pvc-without-storage-class",
							},
						},
					},
				}

				pvc1 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc-unsnapshottable-storage-class",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &localStorageClassName,
					},
				}
				pvc2 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc-ephemeral-pvc",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
						StorageClassName: &storageClassName,
					},
				}
				pvc3 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dv-with-pvc-unsnapshottable-storage-class",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &localStorageClassName,
					},
				}
				pvc7 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-without-storage-class",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: nil,
					},
				}
				pvc8 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dv-with-pvc-without-storage-class",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: nil,
					},
				}

				pvcSource.Add(&pvc1)
				pvcSource.Add(&pvc2)
				pvcSource.Add(&pvc3)
				pvcSource.Add(&pvc7)
				pvcSource.Add(&pvc8)

				emptyString := ""
				dv3 := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dv-with-pvc-unsnapshottable-storage-class",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: &emptyString,
						},
					},
				}
				dv5 := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dv-without-pvc-and-storageclass",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: &localStorageClassName,
						},
					},
				}
				dv8 := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dv-with-pvc-without-storage-class",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: nil,
						},
					},
				}
				dvSource.Add(&dv3)
				dvSource.Add(&dv5)
				dvSource.Add(&dv8)

				localStorageClass := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: localStorageClassName,
					},
					Provisioner: "kubernetes.io/no-provisioner",
				}
				storageClassSource.Add(createStorageClass())
				storageClassSource.Add(localStorageClass)
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}

				vmSource.Add(vm)
				syncCaches(stop)
				mockVMQueue.Add(fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))

				updateCalled := false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(8))

						Expect(vm.Status.VolumeSnapshotStatuses[0].Name).To(Equal("disk1"))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[0].Reason).
							To(Equal("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [local] [disk1]"))

						Expect(vm.Status.VolumeSnapshotStatuses[1].Name).To(Equal("disk2"))
						Expect(vm.Status.VolumeSnapshotStatuses[1].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[1].Reason).
							To(Equal("Snapshot is not supported for this volumeSource type [disk2]"))

						Expect(vm.Status.VolumeSnapshotStatuses[2].Name).To(Equal("disk3"))
						Expect(vm.Status.VolumeSnapshotStatuses[2].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[2].Reason).
							To(Equal("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [local] [disk3]"))

						Expect(vm.Status.VolumeSnapshotStatuses[3].Name).To(Equal("disk4"))
						Expect(vm.Status.VolumeSnapshotStatuses[3].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[3].Reason).
							To(Equal("Snapshot is not supported for this volumeSource type [disk4]"))

						Expect(vm.Status.VolumeSnapshotStatuses[4].Name).To(Equal("disk5"))
						Expect(vm.Status.VolumeSnapshotStatuses[4].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[4].Reason).
							To(Equal("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [local] [disk5]"))

						Expect(vm.Status.VolumeSnapshotStatuses[5].Name).To(Equal("disk6"))
						Expect(vm.Status.VolumeSnapshotStatuses[5].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[5].Reason).
							To(Equal("PVC not found"))

						Expect(vm.Status.VolumeSnapshotStatuses[6].Name).To(Equal("disk7"))
						Expect(vm.Status.VolumeSnapshotStatuses[6].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[6].Reason).
							To(Equal("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [] [disk7]"))

						Expect(vm.Status.VolumeSnapshotStatuses[7].Name).To(Equal("disk8"))
						Expect(vm.Status.VolumeSnapshotStatuses[7].Enabled).To(BeFalse())
						Expect(vm.Status.VolumeSnapshotStatuses[7].Reason).
							To(Equal("No VolumeSnapshotClass: Volume snapshots are not configured for this StorageClass [] [disk8]"))

						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()

				Expect(updateCalled).To(BeTrue())
			})

			It("should process volume snapshot status when a new snapshot class is added", func() {
				vm := createVM()

				pvc := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpine-dv",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &storageClassName,
					},
				}
				dv := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpine-dv",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: &storageClassName,
						},
					},
				}
				pvcSource.Add(&pvc)
				dvSource.Add(&dv)
				storageClassSource.Add(createStorageClass())
				vmSource.Add(vm)

				mockVMQueue.Add(fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))
				syncCaches(stop)

				updateCalled := false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(1))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeFalse())
						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()
				Expect(updateCalled).To(BeTrue())

				volumeSnapshotClasses := createVolumeSnapshotClasses()
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}
				updateCalled = false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(1))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeTrue())
						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()
				Expect(updateCalled).To(BeTrue())
			})

			It("should should use storage class from template when DV does not have it", func() {
				vm := createVM()
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "disk2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						}},
					},
				})

				pvc1 := corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc",
						Namespace: testNamespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &storageClassName,
					},
				}
				emptyString := ""
				dv1 := cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alpine-dv",
						Namespace: testNamespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								Kind: "VirtualMachine",
								Name: vm.Name,
							},
						},
					},
					Spec: cdiv1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: &emptyString,
						},
					},
				}
				pvcSource.Add(&pvc1)
				dvSource.Add(&dv1)
				storageClassSource.Add(createStorageClass())
				vmSource.Add(vm)
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}
				mockVMQueue.Add(fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))
				syncCaches(stop)

				updateCalled := false
				vmInterface.EXPECT().
					UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						vm := obj.(*v1.VirtualMachine)

						Expect(vm.Status.VolumeSnapshotStatuses).To(HaveLen(2))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeTrue())
						Expect(vm.Status.VolumeSnapshotStatuses[1].Enabled).To(BeTrue())
						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()

				Expect(updateCalled).To(BeTrue())
			})
			Describe("it should snapshot vm with instancetypes and preferences", func() {
				var (
					vm                *v1.VirtualMachine
					vmSnapshot        *snapshotv1.VirtualMachineSnapshot
					instancetypeObj   *instancetypev1beta1.VirtualMachineInstancetype
					preferenceObj     *instancetypev1beta1.VirtualMachinePreference
					instancetypeCR    *appsv1.ControllerRevision
					preferenceCR      *appsv1.ControllerRevision
					err               error
					updateStatusCalls *int
				)

				BeforeEach(func() {
					vmSnapshot = createVMSnapshotInProgress()
					vm = createLockedVM()

					expectedSnapshotUpdate := vmSnapshot.DeepCopy()
					expectedSnapshotUpdate.ResourceVersion = "2"
					expectedSnapshotUpdate.Status = &snapshotv1.VirtualMachineSnapshotStatus{
						SourceUID:  &vmUID,
						ReadyToUse: pointer.P(false),
						Phase:      snapshotv1.InProgress,
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
							newReadyCondition(corev1.ConditionFalse, "Not ready"),
						},
					}

					updateStatusCalls = expectVMSnapshotUpdateStatus(vmSnapshotClient, expectedSnapshotUpdate)

					instancetypeObj = createInstancetype()
					instancetypeCR, err = revision.CreateControllerRevision(vm, instancetypeObj)
					Expect(err).ToNot(HaveOccurred())
					crSource.Add(instancetypeCR)

					preferenceObj = createPreference()
					preferenceCR, err = revision.CreateControllerRevision(vm, preferenceObj)
					Expect(err).ToNot(HaveOccurred())
					crSource.Add(preferenceCR)
				})

				DescribeTable("capture ControllerRevision and reference from VirtualMachineSnapshotContent",
					func(getInstancetypeMatcher func() *v1.InstancetypeMatcher, getPreferenceMatcher func() *v1.PreferenceMatcher, getExpectedCR func() *appsv1.ControllerRevision) {
						vm.Spec.Instancetype = getInstancetypeMatcher()
						vm.Spec.Preference = getPreferenceMatcher()

						// Expect the clone of the instancetype CR to be created and owned by the VMSnapshot
						crCreates := expectControllerRevisionCreate(k8sClient, getExpectedCR())

						// Expect the clone of the CR to be referenced from within VMSnapshotContent
						expectedSnapshotContentVM := vm.DeepCopy()
						if expectedSnapshotContentVM.Spec.Instancetype != nil {
							expectedSnapshotContentVM.Spec.Instancetype.RevisionName = strings.Replace(vm.Spec.Instancetype.RevisionName, vm.Name, vmSnapshotName, 1)
						}
						if expectedSnapshotContentVM.Spec.Preference != nil {
							expectedSnapshotContentVM.Spec.Preference.RevisionName = strings.Replace(vm.Spec.Preference.RevisionName, vm.Name, vmSnapshotName, 1)
						}
						expectedVMSnapshotContent := createVirtualMachineSnapshotContent(vmSnapshot, expectedSnapshotContentVM, nil)
						createCalls := expectVMSnapshotContentCreate(vmSnapshotClient, expectedVMSnapshotContent)

						vmSource.Add(vm)
						vmSnapshotSource.Add(vmSnapshot)

						addVirtualMachineSnapshot(vmSnapshot)
						controller.processVMSnapshotWorkItem()
						testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
						Expect(*updateStatusCalls).To(Equal(1))
						Expect(*createCalls).To(Equal(1))
						Expect(*crCreates).To(Equal(1))
					},
					Entry("for a referenced instancetype",
						func() *v1.InstancetypeMatcher {
							return &v1.InstancetypeMatcher{
								Name:         instancetypeObj.Name,
								Kind:         instancetypeapi.SingularResourceName,
								RevisionName: instancetypeCR.Name,
							}
						},
						func() *v1.PreferenceMatcher { return nil },
						func() *appsv1.ControllerRevision {
							return createInstancetypeVirtualMachineSnapshotCR(vm, vmSnapshot, instancetypeObj)
						},
					),
					Entry("for a referenced preference",
						func() *v1.InstancetypeMatcher { return nil },
						func() *v1.PreferenceMatcher {
							return &v1.PreferenceMatcher{
								Name:         preferenceObj.Name,
								Kind:         instancetypeapi.SingularPreferenceResourceName,
								RevisionName: preferenceCR.Name,
							}
						},
						func() *appsv1.ControllerRevision {
							return createInstancetypeVirtualMachineSnapshotCR(vm, vmSnapshot, preferenceObj)
						},
					),
				)
			})
		})

		Context("without VolumeSnapshot and VolumeSnapshotClass informers", func() {
			BeforeEach(func() {
				controller.dynamicInformerMap[volumeSnapshotCRD].informerFunc = func(kubecli.KubevirtClient, time.Duration) cache.SharedIndexInformer {
					return volumeSnapshotInformer
				}

				controller.dynamicInformerMap[volumeSnapshotClassCRD].informerFunc = func(kubecli.KubevirtClient, time.Duration) cache.SharedIndexInformer {
					return volumeSnapshotClassInformer
				}
			})

			DescribeTable("should create informer", func(crdName string) {
				crd := &extv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: crdName,
					},
					Spec: extv1.CustomResourceDefinitionSpec{
						Versions: []extv1.CustomResourceDefinitionVersion{
							{
								Name:   "v1",
								Served: true,
							},
						},
					},
				}

				addCRD(crd)
				controller.processCRDWorkItem()
				Expect(controller.dynamicInformerMap[crdName].stopCh).ShouldNot(BeNil())
				Expect(controller.dynamicInformerMap[crdName].informer).ShouldNot(BeNil())
			},
				Entry("for VolumeSnapshot", volumeSnapshotCRD),
				Entry("for VolumeSnapshotClass", volumeSnapshotClassCRD),
			)
		})
	})
})

func applyPatch(patch []byte, orig, patched interface{}) error {
	originalBytes, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	patchJSON, err := jsonpatch.DecodePatch(patch)
	if err != nil {
		return err
	}
	patchedBytes, err := patchJSON.Apply(originalBytes)
	if err != nil {
		return err
	}

	fmt.Fprintf(GinkgoWriter, "Patched object: %s\n", string(patchedBytes))

	if err = json.Unmarshal(patchedBytes, patched); err != nil {
		return err
	}
	return nil
}

func expectVMSnapshotPatch(client *kubevirtfake.Clientset, orig, newObj *snapshotv1.VirtualMachineSnapshot) *int {
	calls := 0
	client.Fake.PrependReactor("patch", "virtualmachinesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		patch, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())

		patched := &snapshotv1.VirtualMachineSnapshot{}
		if err := applyPatch(patch.GetPatch(), orig, patched); err != nil {
			return false, nil, err
		}

		Expect(patched).To(Equal(newObj))

		calls++

		return true, patched, nil
	})
	return &calls
}

func expectVMSnapshotContentPatch(client *kubevirtfake.Clientset, orig, newObj *snapshotv1.VirtualMachineSnapshotContent) *int {
	calls := 0
	client.Fake.PrependReactor("patch", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		patch, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())

		patched := &snapshotv1.VirtualMachineSnapshotContent{}
		if err := applyPatch(patch.GetPatch(), orig, patched); err != nil {
			return false, nil, err
		}

		Expect(patched).To(Equal(newObj))

		calls++

		return reflect.DeepEqual(newObj, patched), patched, nil
	})
	return &calls
}

func expectVMSnapshotUpdateStatus(client *kubevirtfake.Clientset, vmSnapshot *snapshotv1.VirtualMachineSnapshot) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachinesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		Expect(update.GetSubresource()).To(Equal("status"))

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineSnapshot)

		calls++

		Expect(updateObj.Status).To(Equal(vmSnapshot.Status))
		return true, update.GetObject(), nil
	})
	return &calls
}

func expectVMSnapshotContentCreate(client *kubevirtfake.Clientset, content *snapshotv1.VirtualMachineSnapshotContent) *int {
	calls := 0
	client.Fake.PrependReactor("create", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*snapshotv1.VirtualMachineSnapshotContent)
		Expect(createObj.ObjectMeta).To(Equal(content.ObjectMeta))
		Expect(createObj.Spec.Source).To(Equal(content.Spec.Source))
		Expect(*createObj.Spec.VirtualMachineSnapshotName).To(Equal(*content.Spec.VirtualMachineSnapshotName))
		Expect(createObj.Spec.VolumeBackups).To(ConsistOf(content.Spec.VolumeBackups))
		createObj.Spec.VolumeBackups = []snapshotv1.VolumeBackup{}
		content.Spec.VolumeBackups = []snapshotv1.VolumeBackup{}
		Expect(createObj.Status).To(Equal(content.Status))
		Expect(createObj).To(Equal(content))

		calls++

		return true, create.GetObject(), nil
	})
	return &calls
}

func expectVMSnapshotContentUpdateStatus(client *kubevirtfake.Clientset, content *snapshotv1.VirtualMachineSnapshotContent) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		Expect(update.GetSubresource()).To(Equal("status"))

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineSnapshotContent)

		calls++

		return reflect.DeepEqual(updateObj.Status, content.Status), update.GetObject(), nil
	})
	return &calls
}

func expectVMSnapshotContentDelete(client *kubevirtfake.Clientset, name string) *int {
	calls := 0
	client.Fake.PrependReactor("delete", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())

		Expect(delete.GetName()).To(Equal(name))

		calls++

		return true, nil, nil
	})
	return &calls
}

func expectVolumeSnapshotCreates(
	client *k8ssnapshotfake.Clientset,
	voluemSnapshotClass string,
	content *snapshotv1.VirtualMachineSnapshotContent,
) *int {
	calls := 0
	volumeSnapshots := map[string]string{}
	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName != nil {
			volumeSnapshots[*volumeBackup.VolumeSnapshotName] = volumeBackup.PersistentVolumeClaim.Name
		}
	}
	client.Fake.PrependReactor("create", "volumesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*vsv1.VolumeSnapshot)
		pvc, ok := volumeSnapshots[createObj.Name]
		Expect(ok).To(BeTrue())
		Expect(pvc).Should(Equal(*createObj.Spec.Source.PersistentVolumeClaimName))
		delete(volumeSnapshots, createObj.Name)

		Expect(*createObj.Spec.VolumeSnapshotClassName).Should(Equal(voluemSnapshotClass))
		Expect(createObj.OwnerReferences[0].Name).Should(Equal(content.Name))
		Expect(createObj.OwnerReferences[0].UID).Should(Equal(content.UID))

		calls++

		return true, createObj, nil
	})
	return &calls
}

func createVirtualMachine(namespace, name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       vmUID,
			Labels: map[string]string{
				"kubevirt.io/vm": "vm-alpine-datavolume",
			},
		},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: pointer.P(v1.RunStrategyHalted),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/vm": "vm-alpine-datavolume",
					},
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: diskName,
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.DiskBusVirtio,
										},
									},
								},
							},
							Interfaces: []v1.Interface{{Name: "fake-interface", MacAddress: ""}},
						},
						Resources: v1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceRequestsMemory): resource.MustParse("64M"),
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: diskName,
							VolumeSource: v1.VolumeSource{
								DataVolume: &v1.DataVolumeSource{
									Name: "alpine-dv",
								},
							},
						},
					},
				},
			},
			DataVolumeTemplates: []v1.DataVolumeTemplateSpec{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alpine-dv",
					},
					Spec: cdiv1.DataVolumeSpec{
						Source: &cdiv1.DataVolumeSource{
							HTTP: &cdiv1.DataVolumeSourceHTTP{
								URL: "http://random-url/images/alpine.iso",
							},
						},
						PVC: &corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("2Gi"),
								},
							},
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							StorageClassName: &storageClassName,
						},
					},
				},
			},
		},
	}
}

func createVirtualMachineSnapshot(namespace, name, vmName string) *snapshotv1.VirtualMachineSnapshot {
	return &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       vmSnapshotUID,
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: &vmAPIGroup,
				Kind:     "VirtualMachine",
				Name:     vmName,
			},
			FailureDeadline: noFailureDeadline,
		},
	}
}

func createVirtualMachineSnapshotContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot, vm *v1.VirtualMachine, pvcs []corev1.PersistentVolumeClaim) *snapshotv1.VirtualMachineSnapshotContent {
	var volumeBackups []snapshotv1.VolumeBackup
	vmCpy := &snapshotv1.VirtualMachine{}
	vmCpy.ObjectMeta = *vm.ObjectMeta.DeepCopy()
	vmCpy.Spec = *vm.Spec.DeepCopy()
	vmCpy.ResourceVersion = "1"
	vmCpy.Status = v1.VirtualMachineStatus{}

	for i, pvc := range pvcs {
		diskName := fmt.Sprintf("disk%d", i+1)
		if pvc.Name == "memorydump" {
			diskName = pvc.Name
		}
		volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-volume-%s", vmSnapshot.UID, diskName)
		vb := snapshotv1.VolumeBackup{
			VolumeName: diskName,
			PersistentVolumeClaim: snapshotv1.PersistentVolumeClaim{
				ObjectMeta: pvc.ObjectMeta,
				Spec:       pvc.Spec,
			},
			VolumeSnapshotName: &volumeSnapshotName,
		}
		volumeBackups = append(volumeBackups, vb)
	}

	return &snapshotv1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "vmsnapshot-content-" + vmSnapshotUID,
			Namespace:  testNamespace,
			Finalizers: []string{"snapshot.kubevirt.io/vmsnapshotcontent-protection"},
		},
		Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source: snapshotv1.SourceSpec{
				VirtualMachine: vmCpy,
			},
			VolumeBackups: volumeBackups,
		},
	}
}

func createPVCsForVM(vm *v1.VirtualMachine) []corev1.PersistentVolumeClaim {
	var pvcs []corev1.PersistentVolumeClaim
	for i, dv := range vm.Spec.DataVolumeTemplates {
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vm.Namespace,
				Name:      dv.Name,
			},
			Spec: *dv.Spec.PVC,
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimBound,
			},
		}
		pvc.Spec.VolumeName = fmt.Sprintf("volume%d", i+1)
		pvc.ResourceVersion = "1"
		pvcs = append(pvcs, pvc)
	}
	return pvcs
}

func createPodsUsingPVCs(vm *v1.VirtualMachine) []corev1.Pod {
	var pods []corev1.Pod

	for _, dv := range vm.Spec.DataVolumeTemplates {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vm.Namespace,
				Name:      "pod-" + dv.Name,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "vol-" + dv.Name,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: dv.Name,
							},
						},
					},
				},
			},
		}
		pods = append(pods, pod)
	}

	return pods
}

func memoryDumpPVC() corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       testNamespace,
			Name:            "memorydump",
			ResourceVersion: "2",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "memorydump",
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("500Mi"),
				},
			},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassName,
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
		},
	}
	return pvc
}

func updateVMWithMemoryDump(vm *v1.VirtualMachine) *v1.VirtualMachine {
	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name: "memorydump",
		VolumeSource: v1.VolumeSource{
			MemoryDump: &v1.MemoryDumpVolumeSource{
				PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "memorydump",
					},
				},
			},
		},
	})
	vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
		ClaimName: "memorydump",
		Phase:     v1.MemoryDumpCompleted,
	}
	return vm
}

func createInstancetype() *instancetypev1beta1.VirtualMachineInstancetype {
	return &instancetypev1beta1.VirtualMachineInstancetype{
		TypeMeta: metav1.TypeMeta{
			Kind:       instancetypeapi.SingularResourceName,
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-instancetype",
			Namespace:  testNamespace,
			UID:        instancetypeUID,
			Generation: 1,
		},
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: uint32(2),
			},
		},
	}
}

func createPreference() *instancetypev1beta1.VirtualMachinePreference {
	preferredCPUTopology := instancetypev1beta1.Threads
	return &instancetypev1beta1.VirtualMachinePreference{
		TypeMeta: metav1.TypeMeta{
			Kind:       instancetypeapi.SingularPreferenceResourceName,
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-preference",
			Namespace:  testNamespace,
			UID:        preferenceUID,
			Generation: 1,
		},
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: &preferredCPUTopology,
			},
		},
	}
}

func createInstancetypeVirtualMachineSnapshotCR(vm *v1.VirtualMachine, vmSnapshot *snapshotv1.VirtualMachineSnapshot, obj runtime.Object) *appsv1.ControllerRevision {
	cr, err := revision.CreateControllerRevision(vm, obj)
	Expect(err).ToNot(HaveOccurred())

	// Replace the VM name with the vmSnapshot name and clear the namespace as we don't expect to see this set during creation, only after.
	cr.Name = strings.Replace(cr.Name, vm.Name, vmSnapshot.Name, 1)

	vmSnapshotObj, err := util.GenerateKubeVirtGroupVersionKind(vmSnapshot)
	Expect(err).ToNot(HaveOccurred())
	vmSnapshotCopy, ok := vmSnapshotObj.(*snapshotv1.VirtualMachineSnapshot)
	Expect(ok).To(BeTrue())

	// Set the vmSnapshot as the owner of the CR
	cr.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(vmSnapshotCopy, vmSnapshotCopy.GroupVersionKind())}
	return cr
}

func expectControllerRevisionToEqualExpected(controllerRevision, expectedControllerRevision *appsv1.ControllerRevision) {
	// This is already covered by the below assertion but be explicit here to ensure coverage
	Expect(controllerRevision.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectGenerationLabel))
	Expect(controllerRevision.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectKindLabel))
	Expect(controllerRevision.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectNameLabel))
	Expect(controllerRevision.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectUIDLabel))
	Expect(controllerRevision.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
	Expect(*controllerRevision).To(Equal(*expectedControllerRevision))
}

func expectControllerRevisionCreate(client *k8sfake.Clientset, expectedCR *appsv1.ControllerRevision) *int {
	calls := 0
	client.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		// We don't expect the namespace or resourceversion to be set during creation
		expectedCR.Namespace = ""
		expectedCR.ResourceVersion = ""

		createObj := create.GetObject().(*appsv1.ControllerRevision)
		expectControllerRevisionToEqualExpected(createObj, expectedCR)

		calls++

		return true, createObj, nil
	})
	return &calls
}

func expectCreateControllerRevisionAlreadyExists(client *k8sfake.Clientset, expectedCR *appsv1.ControllerRevision) *int {
	calls := 0
	client.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		// We don't expect the namespace or resourceversion to be set during creation
		expectedCR.Namespace = ""
		expectedCR.ResourceVersion = ""

		createObj := create.GetObject().(*appsv1.ControllerRevision)
		expectControllerRevisionToEqualExpected(createObj, expectedCR)

		calls++

		return true, create.GetObject(), errors.NewAlreadyExists(schema.GroupResource{}, expectedCR.Name)
	})
	return &calls
}

func expectControllerRevisionUpdate(client *k8sfake.Clientset, expectedCR *appsv1.ControllerRevision) *int {
	calls := 0
	client.Fake.PrependReactor("update", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*appsv1.ControllerRevision)
		expectControllerRevisionToEqualExpected(updateObj, expectedCR)

		calls++

		return true, update.GetObject(), nil
	})
	return &calls
}

func expectControllerRevisionDelete(client *k8sfake.Clientset, expectedCRName string) *int {
	calls := 0
	client.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())

		name := delete.GetName()
		Expect(name).To(Equal(expectedCRName))

		calls++

		return true, nil, nil
	})
	return &calls
}
