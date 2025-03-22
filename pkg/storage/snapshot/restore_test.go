package snapshot

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	kubevirtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Restore controller", func() {
	const (
		testNamespace  = "default"
		uid            = "uid"
		vmName         = "testvm"
		vmSnapshotName = "snapshot"
		newVMUID       = "new-vm-UID"
		newVMName      = "new-vm-name"
	)

	var (
		vmAPIGroup       = "kubevirt.io"
		timeStamp        = metav1.Time{Time: time.Now().Truncate(time.Second)}
		storageClassName = "sc"
		vmRestoreName    = "restore"
	)

	timeFunc := func() *metav1.Time {
		return &timeStamp
	}

	createRestore := func() *snapshotv1.VirtualMachineRestore {
		return &snapshotv1.VirtualMachineRestore{
			ObjectMeta: metav1.ObjectMeta{
				Name:              vmRestoreName,
				Namespace:         testNamespace,
				UID:               uid,
				CreationTimestamp: timeStamp,
			},
			Spec: snapshotv1.VirtualMachineRestoreSpec{
				Target: corev1.TypedLocalObjectReference{
					APIGroup: &vmAPIGroup,
					Kind:     "VirtualMachine",
					Name:     vmName,
				},
				VirtualMachineSnapshotName: vmSnapshotName,
			},
		}
	}

	createRestoreWithOwner := func() *snapshotv1.VirtualMachineRestore {
		r := createRestore()
		r.Finalizers = []string{"snapshot.kubevirt.io/vmrestore-protection"}
		r.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion:         kubevirtv1.GroupVersion.String(),
				Kind:               "VirtualMachine",
				Name:               vmName,
				UID:                vmUID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			},
		}
		r.Status = &snapshotv1.VirtualMachineRestoreStatus{
			Complete: pointer.P(false),
		}
		return r
	}

	addInitialVolumeRestores := func(r *snapshotv1.VirtualMachineRestore) {
		r.Status.Restores = []snapshotv1.VolumeRestore{
			{
				VolumeName:                diskName,
				PersistentVolumeClaimName: "restore-uid-disk1",
				VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
			},
		}
	}
	addVolumeRestores := func(r *snapshotv1.VirtualMachineRestore) {
		r.Status.Restores = []snapshotv1.VolumeRestore{
			{
				VolumeName:                diskName,
				PersistentVolumeClaimName: "restore-uid-disk1",
				VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
				DataVolumeName:            pointer.P("restore-uid-disk1"),
			},
		}
	}

	getRestorePVCs := func(vmr *snapshotv1.VirtualMachineRestore) []corev1.PersistentVolumeClaim {
		var pvcs []corev1.PersistentVolumeClaim
		for _, r := range vmr.Status.Restores {
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   testNamespace,
					Name:        r.PersistentVolumeClaimName,
					Annotations: map[string]string{"restore.kubevirt.io/name": vmr.Name},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClassName,
				},
			}
			pvcs = append(pvcs, pvc)
		}
		return pvcs
	}

	createSnapshotWith := func(phase snapshotv1.VirtualMachineSnapshotPhase, ready bool) *snapshotv1.VirtualMachineSnapshot {
		s := createVirtualMachineSnapshot(testNamespace, vmSnapshotName, vmName)
		s.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		s.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse:   pointer.P(ready),
			CreationTime: timeFunc(),
			SourceUID:    &vmUID,
			Phase:        phase,
		}
		return s
	}

	createSnapshot := func() *snapshotv1.VirtualMachineSnapshot {
		return createSnapshotWith(snapshotv1.Succeeded, true)
	}

	createSnapshotVM := func() *kubevirtv1.VirtualMachine {
		return createVirtualMachine(testNamespace, vmName)
	}

	createModifiedVM := func() *kubevirtv1.VirtualMachine {
		vm := createVirtualMachine(testNamespace, vmName)
		vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceName(corev1.ResourceRequestsMemory)] = resource.MustParse("128M")
		return vm
	}

	createVMI := func(vm *kubevirtv1.VirtualMachine) *kubevirtv1.VirtualMachineInstance {
		return &kubevirtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vm.Name,
				Namespace: vm.Namespace,
			},
		}
	}

	getDeletedDataVolumes := func(vm *kubevirtv1.VirtualMachine) []string {
		var names []string
		for _, dvt := range vm.Spec.DataVolumeTemplates {
			names = append(names, dvt.Name)
		}
		return names
	}

	createStorageClass := func() *storagev1.StorageClass {
		bm := storagev1.VolumeBindingImmediate
		return &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: storageClassName,
			},
			VolumeBindingMode: &bm,
		}
	}

	createVolumeSnapshot := func(name string, restoreSize resource.Quantity) *vsv1.VolumeSnapshot {
		return &vsv1.VolumeSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Status: &vsv1.VolumeSnapshotStatus{
				RestoreSize: &restoreSize,
			},
		}
	}

	createPVCsForVMWithDataSourceRef := func(vm *kubevirtv1.VirtualMachine) []corev1.PersistentVolumeClaim {
		var pvcs []corev1.PersistentVolumeClaim
		for i, dv := range vm.Spec.DataVolumeTemplates {
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vm.Namespace,
					Name:      dv.Name,
				},
				Spec: *dv.Spec.PVC,
			}
			if dv.Spec.PVC.DataSource == nil {
				dv.Spec.PVC.DataSource = &corev1.TypedLocalObjectReference{}
			}
			dataSourceRef := &corev1.TypedObjectReference{
				APIGroup: dv.Spec.PVC.DataSource.APIGroup,
				Kind:     dv.Spec.PVC.DataSource.Kind,
				Name:     dv.Spec.PVC.DataSource.Name,
			}

			pvc.Spec.DataSourceRef = dataSourceRef
			pvc.Spec.VolumeName = fmt.Sprintf("volume%d", i+1)
			pvc.ResourceVersion = "1"
			pvcs = append(pvcs, pvc)
		}
		return pvcs
	}

	Context("One valid Restore controller given", func() {

		var ctrl *gomock.Controller

		var vmInterface *kubecli.MockVirtualMachineInterface

		var vmRestoreSource *framework.FakeControllerSource
		var vmRestoreInformer cache.SharedIndexInformer

		var vmSnapshotSource *framework.FakeControllerSource
		var vmSnapshotInformer cache.SharedIndexInformer

		var vmSnapshotContentSource *framework.FakeControllerSource
		var vmSnapshotContentInformer cache.SharedIndexInformer

		var vmInformer cache.SharedIndexInformer
		var vmSource *framework.FakeControllerSource

		var vmiInformer cache.SharedIndexInformer
		var vmiSource *framework.FakeControllerSource

		var dataVolumeInformer cache.SharedIndexInformer
		var dataVolumeSource *framework.FakeControllerSource

		var pvcInformer cache.SharedIndexInformer
		var pvcSource *framework.FakeControllerSource

		var storageClassInformer cache.SharedIndexInformer
		var storageClassSource *framework.FakeControllerSource

		var crInformer cache.SharedIndexInformer
		var crSource *framework.FakeControllerSource

		var stop chan struct{}
		var controller *VMRestoreController
		var recorder *record.FakeRecorder
		var mockVMRestoreQueue *testutils.MockWorkQueue[string]
		var fakeVolumeSnapshotProvider *MockVolumeSnapshotProvider

		var kubevirtClient *kubevirtfake.Clientset
		var k8sClient *k8sfake.Clientset
		var cdiClient *cdifake.Clientset
		var virtClient *kubecli.MockKubevirtClient

		syncCaches := func(stop chan struct{}) {
			go vmRestoreInformer.Run(stop)
			go vmSnapshotInformer.Run(stop)
			go vmSnapshotContentInformer.Run(stop)
			go vmInformer.Run(stop)
			go pvcInformer.Run(stop)
			go vmiInformer.Run(stop)
			go dataVolumeInformer.Run(stop)
			go storageClassInformer.Run(stop)
			go crInformer.Run(stop)
			Expect(cache.WaitForCacheSync(
				stop,
				vmRestoreInformer.HasSynced,
				vmSnapshotInformer.HasSynced,
				vmSnapshotContentInformer.HasSynced,
				vmInformer.HasSynced,
				pvcInformer.HasSynced,
				vmiInformer.HasSynced,
				dataVolumeInformer.HasSynced,
				storageClassInformer.HasSynced,
				crInformer.HasSynced,
			)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

			vmRestoreInformer, vmRestoreSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineRestore{}, virtcontroller.GetVirtualMachineRestoreInformerIndexers())
			vmSnapshotInformer, vmSnapshotSource = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
			vmSnapshotContentInformer, vmSnapshotContentSource = testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&kubevirtv1.VirtualMachineInstance{})
			vmInformer, vmSource = testutils.NewFakeInformerFor(&kubevirtv1.VirtualMachine{})
			dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			pvcInformer, pvcSource = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			storageClassInformer, storageClassSource = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			crInformer, crSource = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, virtcontroller.GetControllerRevisionInformerIndexers())

			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			fakeVolumeSnapshotProvider = &MockVolumeSnapshotProvider{
				volumeSnapshots: []*vsv1.VolumeSnapshot{},
			}

			controller = &VMRestoreController{
				Client:                    virtClient,
				VMRestoreInformer:         vmRestoreInformer,
				VMSnapshotInformer:        vmSnapshotInformer,
				VMSnapshotContentInformer: vmSnapshotContentInformer,
				VMInformer:                vmInformer,
				VMIInformer:               vmiInformer,
				PVCInformer:               pvcInformer,
				StorageClassInformer:      storageClassInformer,
				DataVolumeInformer:        dataVolumeInformer,
				Recorder:                  recorder,
				VolumeSnapshotProvider:    fakeVolumeSnapshotProvider,
				CRInformer:                crInformer,
			}
			controller.Init()

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockVMRestoreQueue = testutils.NewMockWorkQueue(controller.vmRestoreQueue)
			controller.vmRestoreQueue = mockVMRestoreQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()

			kubevirtClient = kubevirtfake.NewSimpleClientset()

			virtClient.EXPECT().VirtualMachineRestore(testNamespace).
				Return(kubevirtClient.SnapshotV1beta1().VirtualMachineRestores(testNamespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshot(testNamespace).
				Return(kubevirtClient.SnapshotV1beta1().VirtualMachineSnapshots(testNamespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshotContent(testNamespace).
				Return(kubevirtClient.SnapshotV1beta1().VirtualMachineSnapshotContents(testNamespace)).AnyTimes()

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

			cdiClient = cdifake.NewSimpleClientset()
			virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

			k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			kubevirtClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			currentTime = timeFunc
		})

		addVirtualMachineRestore := func(r *snapshotv1.VirtualMachineRestore) {
			syncCaches(stop)
			mockVMRestoreQueue.ExpectAdds(1)
			vmRestoreSource.Add(r)
			mockVMRestoreQueue.Wait()
		}

		addPVC := func(pvc *corev1.PersistentVolumeClaim) {
			syncCaches(stop)
			mockVMRestoreQueue.ExpectAdds(1)
			pvcSource.Add(pvc)
			mockVMRestoreQueue.Wait()
		}

		addVM := func(vm *kubevirtv1.VirtualMachine) {
			syncCaches(stop)
			mockVMRestoreQueue.ExpectAdds(1)
			vmSource.Add(vm)
			mockVMRestoreQueue.Wait()
		}

		expectUpdateVMRestoreInProgress := func(vm *kubevirtv1.VirtualMachine) *kubevirtv1.VirtualMachine {
			vmStatusUpdate := vm.DeepCopy()
			vmStatusUpdate.ResourceVersion = "1"
			vmStatusUpdate.Status.RestoreInProgress = &vmRestoreName
			vmInterface.EXPECT().UpdateStatus(context.Background(), vmStatusUpdate, metav1.UpdateOptions{}).Return(vmStatusUpdate, nil).Times(1)
			return vmStatusUpdate
		}

		Context("with initialized snapshot and content", func() {

			var (
				s            *snapshotv1.VirtualMachineSnapshot
				vm           *kubevirtv1.VirtualMachine
				sc           *snapshotv1.VirtualMachineSnapshotContent
				storageClass *storagev1.StorageClass
			)

			BeforeEach(func() {
				s = createSnapshot()
				vm = createSnapshotVM()
				pvcs := createPVCsForVM(vm)
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				storageClass = createStorageClass()
				s.Status.VirtualMachineSnapshotContentName = &sc.Name
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				vmSnapshotSource.Add(s)
				vmSnapshotContentSource.Add(sc)
				storageClassSource.Add(storageClass)
			})

			DescribeTable("should error if snapshot", func(vmSnapshot *snapshotv1.VirtualMachineSnapshot, expectedError string) {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, expectedError),
						newReadyCondition(corev1.ConditionFalse, expectedError),
					},
				}
				vmSnapshotSource.Delete(createSnapshot())
				if vmSnapshot != nil {
					vmSnapshotSource.Add(vmSnapshot)
				}
				vmSource.Add(vm)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				testutils.ExpectEvent(recorder, "VirtualMachineRestoreError")
				Expect(*updateStatusCalls).To(Equal(1))
			},
				Entry("does not exist", nil, "VMSnapshot default/snapshot does not exist"),
				Entry("in failed state", createSnapshotWith(snapshotv1.Failed, false), "VMSnapshot default/snapshot failed and is invalid to use"),
				Entry("not ready", createSnapshotWith(snapshotv1.InProgress, false), "VMSnapshot default/snapshot not ready"),
			)

			It("should error if target exists before the restore and it is not the same as the source", func() {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				newVM := createVirtualMachine(testNamespace, newVMName)
				newVM.UID = newVMUID
				r.Spec.Target.Name = newVM.Name

				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "restore source and restore target are different but restore target already exists"),
						newReadyCondition(corev1.ConditionFalse, "restore source and restore target are different but restore target already exists"),
					},
				}

				vmSource.Add(vm)
				vmSource.Add(newVM)

				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				testutils.ExpectEvent(recorder, "VirtualMachineRestoreError")
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update restore status, initializing conditions", func() {
				r := createRestoreWithOwner()
				r.Finalizers = nil
				r.OwnerReferences = nil
				r.Status = nil
				vm := createModifiedVM()
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"),
						newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"),
					},
				}
				vmSource.Add(vm)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should wait for target to be ready before updating vm in progress", func() {
				r := createRestoreWithOwner()
				finalizers := r.Finalizers
				r.Finalizers = nil
				ownerRefs := r.OwnerReferences
				r.OwnerReferences = nil
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"),
						newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"),
					},
				}
				vm := createModifiedVM()
				vmi := createVMI(vm)
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Finalizers = finalizers
				rc.OwnerReferences = ownerRefs
				updateCalls := expectVMRestoreUpdate(kubevirtClient, rc)

				rc2 := rc.DeepCopy()
				rc2.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
					},
				}
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc2)
				vmSource.Add(vm)
				vmiSource.Add(vmi)

				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
				Expect(*updateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update restore, add finalizer and owner", func() {
				r := createRestoreWithOwner()
				finalizers := r.Finalizers
				r.Finalizers = nil
				ownerRefs := r.OwnerReferences
				r.OwnerReferences = nil
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"),
						newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"),
					},
				}
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Finalizers = finalizers
				rc.OwnerReferences = ownerRefs
				updateCalls := expectVMRestoreUpdate(kubevirtClient, rc)

				rc2 := rc.DeepCopy()
				rc2.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addInitialVolumeRestores(rc2)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc2)

				vm := createModifiedVM()
				vmSource.Add(vm)
				expectUpdateVMRestoreInProgress(vm)

				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*updateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update restore status with condition and VolumeRestores", func() {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addInitialVolumeRestores(rc)
				vmSource.Add(vm)
				expectUpdateVMRestoreInProgress(vm)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should return error if volumesnapshot doesnt exist", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addVolumeRestores(r)
				vm := createModifiedVM()
				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "missing volumeSnapshot vmsnapshot-snapshot-uid-volume-disk1"),
						newReadyCondition(corev1.ConditionFalse, "missing volumeSnapshot vmsnapshot-snapshot-uid-volume-disk1"),
					},
				}
				addVolumeRestores(rc)

				vmSource.Add(vm)
				addVirtualMachineRestore(r)

				expectUpdateVMRestoreInProgress(vm)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should create restore PVCs", func() {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				vmSource.Add(vm)
				addVolumeRestores(r)
				pvcSize := resource.MustParse("2Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
				fakeVolumeSnapshotProvider.Add(vs)
				expectUpdateVMRestoreInProgress(vm)
				calls := expectPVCCreates(k8sClient, r, pvcSize)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
			})

			It("should create pvcs for both datavolume and pvc restore volumes", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addVolumeRestores(r)
				r.Status.Restores = append(r.Status.Restores, snapshotv1.VolumeRestore{
					VolumeName:                "disk2",
					PersistentVolumeClaimName: "restore-uid-disk2",
					VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk2",
				})

				vm := createModifiedVM()
				// create extra pvc
				pvcs := createPVCsForVM(vm)
				pvcs = append(pvcs, corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: vm.Namespace,
						Name:      "extra-pvc",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("2Gi"),
							},
						},
						VolumeName:       "volume2",
						StorageClassName: &storageClass.Name,
					},
				})
				//add pvc volume and disk to vm
				disk := kubevirtv1.Disk{
					Name: "disk2",
					DiskDevice: kubevirtv1.DiskDevice{
						Disk: &kubevirtv1.DiskTarget{
							Bus: kubevirtv1.DiskBusVirtio,
						},
					},
				}
				volume := kubevirtv1.Volume{
					Name: "disk2",
					VolumeSource: kubevirtv1.VolumeSource{
						PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{ClaimName: "extra-pvc"}},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, disk)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, volume)
				// delete previous vmsnapshotcontent
				vmSnapshotContentSource.Delete(sc)
				// create ct vmSnapshotContent with the relevant info
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				vmSnapshotContentSource.Add(sc)
				pvcSize := resource.MustParse("2Gi")
				vs1 := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
				vs2 := createVolumeSnapshot(r.Status.Restores[1].VolumeSnapshotName, pvcSize)
				fakeVolumeSnapshotProvider.Add(vs1)
				fakeVolumeSnapshotProvider.Add(vs2)

				vmSource.Add(vm)
				addVirtualMachineRestore(r)

				expectUpdateVMRestoreInProgress(vm)
				calls := expectPVCCreates(k8sClient, r, pvcSize)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(2))
			})

			It("should create restore PVC with volume snapshot size if bigger then PVC size", func() {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				vmSource.Add(vm)
				addVolumeRestores(r)
				q := resource.MustParse("3Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, q)
				fakeVolumeSnapshotProvider.Add(vs)
				expectUpdateVMRestoreInProgress(vm)
				calls := expectPVCCreates(k8sClient, r, q)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
			})

			It("should create restore PVC with pvc size if restore size is smaller", func() {
				r := createRestoreWithOwner()
				vm := createModifiedVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				vmSource.Add(vm)
				addVolumeRestores(r)
				q := resource.MustParse("1Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, q)
				fakeVolumeSnapshotProvider.Add(vs)
				expectUpdateVMRestoreInProgress(vm)
				pvcSize := resource.MustParse("2Gi")
				calls := expectPVCCreates(k8sClient, r, pvcSize)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
			})

			It("should wait for bound", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addVolumeRestores(r)

				vm := createModifiedVM()
				vmSource.Add(vm)
				vmRestoreSource.Add(r)
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimPending
					addPVC(&pvc)
				}
				expectUpdateVMRestoreInProgress(vm)
				controller.processVMRestoreWorkItem()
			})

			It("should update PVCs and restores to have datavolumename", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
					},
				}
				addVolumeRestores(r)
				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
				}
				ur.Status.DeletedDataVolumes = getDeletedDataVolumes(createModifiedVM())
				for i := range ur.Status.Restores {
					ur.Status.Restores[i].DataVolumeName = &ur.Status.Restores[i].PersistentVolumeClaimName
				}

				vm := createModifiedVM()
				vmSource.Add(vm)
				expectUpdateVMRestoreInProgress(vm)
				vmRestoreSource.Add(r)
				pvcUpdateCalls := expectPVCUpdates(k8sClient, ur)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimBound
					addPVC(&pvc)
				}
				controller.processVMRestoreWorkItem()
				Expect(*pvcUpdateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update correctly restore VolumeRestores with multiple volumes and update relevant PVCs", func() {
				vm := createModifiedVM()
				vm.Status.RestoreInProgress = &vmRestoreName
				// create extra pvc
				pvcs := createPVCsForVM(vm)
				pvcs = append(pvcs, corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: vm.Namespace,
						Name:      "extra-pvc",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName:       "volume2",
						StorageClassName: &storageClass.Name,
					},
				})
				//add pvc volume and disk to vm
				disk := kubevirtv1.Disk{
					Name: "disk2",
					DiskDevice: kubevirtv1.DiskDevice{
						Disk: &kubevirtv1.DiskTarget{
							Bus: kubevirtv1.DiskBusVirtio,
						},
					},
				}
				volume := kubevirtv1.Volume{
					Name: "disk2",
					VolumeSource: kubevirtv1.VolumeSource{
						PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{ClaimName: "extra-pvc"}},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, disk)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, volume)
				// delete previous vmsnapshotcontent
				vmSnapshotContentSource.Delete(sc)
				// create ct vmSnapshotContent with the relevant info
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				vmSnapshotContentSource.Add(sc)

				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				// note here we dont add the datavolumename
				addInitialVolumeRestores(r)
				r.Status.Restores = append(r.Status.Restores, snapshotv1.VolumeRestore{
					VolumeName:                "disk2",
					PersistentVolumeClaimName: "restore-uid-disk2",
					VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk2",
				})
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimBound
					pvcSource.Add(&pvc)
				}

				rc := r.DeepCopy()
				rc.ResourceVersion = "1"
				rc.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
				}
				addVolumeRestores(rc)
				// note here we dont expect the datavolumename
				rc.Status.Restores = append(rc.Status.Restores, snapshotv1.VolumeRestore{
					VolumeName:                "disk2",
					PersistentVolumeClaimName: "restore-uid-disk2",
					VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk2",
				})
				vmSource.Add(vm)
				addVirtualMachineRestore(r)

				pvcUpdateCalls := expectPVCUpdates(k8sClient, rc)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				controller.processVMRestoreWorkItem()
				Expect(*pvcUpdateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should cleanup, unlock vm and mark restore completed", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete:           pointer.P(false),
					DeletedDataVolumes: getDeletedDataVolumes(createModifiedVM()),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
					},
				}
				addVolumeRestores(r)
				for i := range r.Status.Restores {
					r.Status.Restores[i].DataVolumeName = &r.Status.Restores[i].PersistentVolumeClaimName
				}
				vmRestoreSource.Add(r)

				vm := createModifiedVM()
				vm.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
				vm.Status.RestoreInProgress = &vmRestoreName

				for _, n := range r.Status.DeletedDataVolumes {
					dv := &cdiv1.DataVolume{
						ObjectMeta: metav1.ObjectMeta{
							Name:      n,
							Namespace: testNamespace,
						},
					}
					dataVolumeSource.Add(dv)
				}
				for _, pvc := range getRestorePVCs(r) {
					pvc.Annotations["cdi.kubevirt.io/storage.populatedFor"] = pvc.Name
					pvc.Status.Phase = corev1.ClaimBound
					pvcSource.Add(&pvc)
				}

				addVM(vm)

				updatedVM := vm.DeepCopy()
				updatedVM.ResourceVersion = "1"
				updatedVM.Status.RestoreInProgress = nil
				vmInterface.EXPECT().UpdateStatus(context.Background(), updatedVM, metav1.UpdateOptions{}).Return(updatedVM, nil).Times(1)

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Complete = pointer.P(true)
				ur.Status.RestoreTime = timeFunc()
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Operation complete"),
				}

				dvDeleteCalls := expectDataVolumeDeletes(cdiClient, r.Status.DeletedDataVolumes)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)

				controller.processVMRestoreWorkItem()

				l, err := cdiClient.CdiV1beta1().DataVolumes("").List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(l.Items).To(BeEmpty())
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*dvDeleteCalls).To(Equal(len(r.Status.DeletedDataVolumes)))
			})

			It("should complete restore", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete:           pointer.P(false),
					DeletedDataVolumes: getDeletedDataVolumes(createModifiedVM()),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Updating target status"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
					},
				}
				addVolumeRestores(r)
				for i := range r.Status.Restores {
					r.Status.Restores[i].DataVolumeName = &r.Status.Restores[i].PersistentVolumeClaimName
				}

				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vmName,
						Namespace: testNamespace,
						UID:       vmUID,
						Annotations: map[string]string{
							"restore.kubevirt.io/lastRestoreUID": "restore-uid",
						},
					},
				}

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Complete = pointer.P(true)
				ur.Status.RestoreTime = timeFunc()
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Operation complete"),
				}
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)

				for _, pvc := range getRestorePVCs(r) {
					pvc.Annotations["cdi.kubevirt.io/storage.populatedFor"] = pvc.Name
					pvc.Status.Phase = corev1.ClaimBound
					pvcSource.Add(&pvc)
				}

				vmRestoreSource.Add(r)
				addVM(vm)
				controller.processVMRestoreWorkItem()
				testutils.ExpectEvent(recorder, "VirtualMachineRestoreComplete")
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update status if restore deleted after completion", func() {
				r := createRestoreWithOwner()
				r.DeletionTimestamp = timeFunc()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete:           pointer.P(true),
					DeletedDataVolumes: getDeletedDataVolumes(createModifiedVM()),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
						newReadyCondition(corev1.ConditionTrue, "Operation complete"),
					},
					RestoreTime: timeFunc(),
				}

				vm := createModifiedVM()
				vm.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}

				vmRestoreSource.Add(r)
				addVM(vm)

				updatedVMRestore := r.DeepCopy()
				updatedVMRestore.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
					newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
				}
				updatedVMRestore.ResourceVersion = "1"

				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should remove finalizer if restore deleted after completion", func() {
				r := createRestoreWithOwner()
				r.DeletionTimestamp = timeFunc()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete:           pointer.P(true),
					DeletedDataVolumes: getDeletedDataVolumes(createModifiedVM()),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
						newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
					},
					RestoreTime: timeFunc(),
				}

				vm := createModifiedVM()
				vm.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}

				vmRestoreSource.Add(r)
				addVM(vm)

				updatedVMRestore := r.DeepCopy()
				updatedVMRestore.ResourceVersion = "1"
				updatedVMRestore.Finalizers = []string{}

				patchCount := expectVMRestorePatch(kubevirtClient, r, updatedVMRestore)
				controller.processVMRestoreWorkItem()
				Expect(*patchCount).To(Equal(1))
			})

			It("should clean existing vm and then remove restore status if restore deleted before completion", func() {
				r := createRestoreWithOwner()
				r.DeletionTimestamp = timeFunc()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Updating target status"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
					},
				}

				vm := createModifiedVM()
				vm.Status.RestoreInProgress = &vmRestoreName

				vmRestoreSource.Add(r)
				addVM(vm)

				vmUpdated := vm.DeepCopy()
				vmUpdated.Status.RestoreInProgress = nil
				vmInterface.EXPECT().UpdateStatus(context.Background(), vmUpdated, metav1.UpdateOptions{}).Return(vmUpdated, nil).Times(1)

				updatedVMRestore := r.DeepCopy()
				updatedVMRestore.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
					newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
				}
				updatedVMRestore.ResourceVersion = "1"

				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should clean existing vm and then remove restore finalizer if restore deleted before completion", func() {
				r := createRestoreWithOwner()
				r.DeletionTimestamp = timeFunc()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
						newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
					},
				}

				vm := createModifiedVM()

				vmRestoreSource.Add(r)
				addVM(vm)

				updatedVMRestore := r.DeepCopy()
				updatedVMRestore.ResourceVersion = "1"
				updatedVMRestore.Finalizers = []string{}

				patchCount := expectVMRestorePatch(kubevirtClient, r, updatedVMRestore)
				controller.processVMRestoreWorkItem()
				Expect(*patchCount).To(Equal(1))
			})

			Context("target Reconcile", func() {
				var (
					r        *snapshotv1.VirtualMachineRestore
					targetVM restoreTarget
				)
				BeforeEach(func() {
					r = createRestoreWithOwner()
					addVolumeRestores(r)
					vm = createModifiedVM()
					vm.Status.RestoreInProgress = &vmRestoreName
					targetVM, _ = controller.getTarget(r)
					targetVM.UpdateTarget(vm)
				})

				addRestoreVolumes := func(dvExists bool, phase cdiv1.DataVolumePhase) *int {
					var calls *int
					pvc := corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   testNamespace,
							Name:        "restore-uid-disk1",
							Annotations: map[string]string{populatedForPVCAnnotation: "restore-uid-disk1"},
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							StorageClassName: &storageClassName,
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase: corev1.ClaimBound,
						},
					}
					if dvExists {
						dv := &cdiv1.DataVolume{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "restore-uid-disk1",
								Namespace: testNamespace,
							},
							Status: cdiv1.DataVolumeStatus{
								Phase: phase,
							},
						}
						dataVolumeSource.Add(dv)
						pvc.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"})}
						r.Status.DeletedDataVolumes = getDeletedDataVolumes(vm)
					} else {
						calls = expectDataVolumeCreate(cdiClient, "restore-uid-disk1")
					}
					pvcSource.Add(&pvc)
					return calls
				}
				expectUpdateRestoredVM := func() {
					updatedVM := createSnapshotVM()
					updatedVM.Status.RestoreInProgress = &vmRestoreName
					updatedVM.ResourceVersion = "1"
					updatedVM.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
					updatedVM.Spec.DataVolumeTemplates[0].Name = "restore-uid-disk1"
					updatedVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = "restore-uid-disk1"
					vmInterface.EXPECT().Update(context.Background(), updatedVM, metav1.UpdateOptions{}).Return(updatedVM, nil).Times(1)
				}

				DescribeTable("should", func(dvExists bool, phase cdiv1.DataVolumePhase, expecteUpdateVM bool) {
					dvCreateCalls := addRestoreVolumes(dvExists, phase)

					vmRestoreSource.Add(r)
					addVM(vm)
					if expecteUpdateVM == false {
						expectUpdateRestoredVM()
					}
					res, err := targetVM.Reconcile()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res).To(BeTrue())
					if dvCreateCalls != nil {
						Expect(*dvCreateCalls).To(Equal(1))
					}
				},
					Entry("update VM spec when dv phase succeeded", true, cdiv1.Succeeded, false),
					Entry("update VM spec when dv phase WFFC", true, cdiv1.WaitForFirstConsumer, false),
					Entry("wait for dvs when dv phase pending", true, cdiv1.Pending, true),
					Entry("create dvs when dv doesnt exists", false, cdiv1.PhaseUnset, true),
				)
			})

			Context("target VM is different than source VM", func() {

				It("should be able to restore to a new VM", func() {
					By("Creating new VM")
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.UID = ""

					By("Creating VM restore")
					vmRestore := createRestoreWithOwner()
					vmRestore.Spec.Target.Name = newVM.Name
					addVolumeRestores(vmRestore)
					addVirtualMachineRestore(vmRestore)

					By("Creating PVC")
					for _, pvc := range getRestorePVCs(vmRestore) {
						pvc.Status.Phase = corev1.ClaimBound
						addPVC(&pvc)
					}

					Expect(vmRestore.Status.Restores).To(HaveLen(1))
					vmRestore.Status.Restores[0].DataVolumeName = pointer.P(restoreDVName(vmRestore, vmRestore.Status.Restores[0].VolumeName))
					pvcUpdateCalls := expectPVCUpdates(k8sClient, vmRestore)

					By("Making sure right VM update occurs")
					newVM.Spec.DataVolumeTemplates[0].Name = *vmRestore.Status.Restores[0].DataVolumeName
					newVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = *vmRestore.Status.Restores[0].DataVolumeName
					newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}

					vmInterface.EXPECT().Create(context.Background(), newVM, metav1.CreateOptions{}).Return(newVM, nil).Times(1)

					By("Making sure right VMRestore update occurs")
					updatedVMRestore := vmRestore.DeepCopy()
					updatedVMRestore.Status.Conditions = []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
					}
					updatedVMRestore.ResourceVersion = "1"

					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore)

					By("Running the controller")
					controller.processVMRestoreWorkItem()
					Expect(*pvcUpdateCalls).To(Equal(1))
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("should own the vmrestore after creation of new target", func() {
					By("Creating new VM")
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.Status.RestoreInProgress = &vmRestoreName
					newVM.UID = newVMUID
					newVM.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
					vmSource.Add(newVM)

					By("Creating VM restore")
					vmRestore := createRestore()
					vmRestore.Finalizers = []string{"snapshot.kubevirt.io/vmrestore-protection"}
					vmRestore.Spec.Target.Name = newVM.Name
					vmRestore.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"),
							newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"),
						},
					}
					addVirtualMachineRestore(vmRestore)

					By("Making sure right VMRestore update occurs")
					updatedVMRestore := vmRestore.DeepCopy()
					updatedVMRestore.OwnerReferences = []metav1.OwnerReference{
						{
							APIVersion:         kubevirtv1.GroupVersion.String(),
							Kind:               "VirtualMachine",
							Name:               newVM.Name,
							UID:                newVMUID,
							Controller:         pointer.P(true),
							BlockOwnerDeletion: pointer.P(true),
						},
					}
					updatedVMRestore.ResourceVersion = "1"

					updatedVMRestore2 := updatedVMRestore.DeepCopy()
					updatedVMRestore2.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
						},
					}
					addInitialVolumeRestores(updatedVMRestore2)

					updateCount := expectVMRestoreUpdate(kubevirtClient, updatedVMRestore)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore2)

					By("Running the controller")
					controller.processVMRestoreWorkItem()
					Expect(*updateCount).To(Equal(1))
					Expect(*updateStatusCalls).To(Equal(1))
				})

				Context("new VM does not exist, should create new VM", func() {

					const (
						newVmName     = "new-vm"
						newMacAddress = "00:00:5e:00:53:01"
					)

					var (
						r *snapshotv1.VirtualMachineRestore

						changeNamePatch       string
						changeMacAddressPatch string
					)

					BeforeEach(func() {
						r = createRestore()
						r.Spec.Target.Name = "new-vm-name"
						r.Status = &snapshotv1.VirtualMachineRestoreStatus{
							Complete: pointer.P(false),
						}
						addVolumeRestores(r)

						for _, pvc := range getRestorePVCs(r) {
							dv := &cdiv1.DataVolume{
								ObjectMeta: metav1.ObjectMeta{
									Name:      pvc.Name,
									Namespace: pvc.Namespace,
								},
								Status: cdiv1.DataVolumeStatus{
									Phase: cdiv1.Succeeded,
								},
							}
							dataVolumeSource.Add(dv)
							pvc.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"})}
							pvc.Annotations["cdi.kubevirt.io/storage.populatedFor"] = pvc.Name
							pvc.Status.Phase = corev1.ClaimBound
							addPVC(&pvc)
						}

						changeNamePatch = fmt.Sprintf(`{"op": "replace", "path": "/metadata/name", "value": "%s"}`, newVmName)
						changeMacAddressPatch = fmt.Sprintf(`{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": "%s"}`, newMacAddress)
					})

					It("with changed name", func() {
						r.Spec.Patches = []string{changeNamePatch}

						newVM := createVirtualMachine(testNamespace, newVmName)
						newVM.UID = ""
						newVM.Spec.DataVolumeTemplates[0].Name = restoreDVName(r, r.Status.Restores[0].VolumeName)
						newVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = restoreDVName(r, r.Status.Restores[0].VolumeName)
						newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}

						vmInterface.EXPECT().Create(context.Background(), newVM, metav1.CreateOptions{}).Return(newVM, nil).Times(1)

						targetVM, err := controller.getTarget(r)
						Expect(err).ShouldNot(HaveOccurred())
						success, err := targetVM.Reconcile()
						Expect(success).To(BeTrue())
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("with changed name and MAC address", func() {
						r.Spec.Patches = []string{changeNamePatch, changeMacAddressPatch}

						newVM := createVirtualMachine(testNamespace, newVmName)
						newVM.UID = ""
						newVM.Spec.DataVolumeTemplates[0].Name = restoreDVName(r, r.Status.Restores[0].VolumeName)
						newVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = restoreDVName(r, r.Status.Restores[0].VolumeName)
						newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
						newVM.Spec.Template.Spec.Domain.Devices.Interfaces[0].MacAddress = newMacAddress

						vmInterface.EXPECT().Create(context.Background(), newVM, metav1.CreateOptions{}).Return(newVM, nil).Times(1)

						targetVM, err := controller.getTarget(r)
						Expect(err).ShouldNot(HaveOccurred())
						success, err := targetVM.Reconcile()
						Expect(success).To(BeTrue())
						Expect(err).ShouldNot(HaveOccurred())
					})

				})

				It("should update condition if deleted and failed to restore", func() {
					// This VM will never be created
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.UID = ""

					By("Creating VM restore")
					vmRestore := createRestoreWithOwner()
					vmRestore.Spec.Target.Name = newVM.Name
					vmRestore.DeletionTimestamp = timeFunc()
					addVolumeRestores(vmRestore)

					addVirtualMachineRestore(vmRestore)

					updatedVMRestore := vmRestore.DeepCopy()
					updatedVMRestore.Status.Conditions = []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
						newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
					}
					updatedVMRestore.ResourceVersion = "1"

					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore)
					controller.processVMRestoreWorkItem()
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("should remove restore finalizer if deleted and failed to restore", func() {
					// This VM will never be created
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.UID = ""

					By("Creating VM restore")
					vmRestore := createRestoreWithOwner()
					vmRestore.Spec.Target.Name = newVM.Name
					vmRestore.DeletionTimestamp = timeFunc()
					vmRestore.Status.Conditions = []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
						newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
					}
					addVolumeRestores(vmRestore)

					addVirtualMachineRestore(vmRestore)

					updatedVMRestore := vmRestore.DeepCopy()
					updatedVMRestore.ResourceVersion = "1"
					updatedVMRestore.Finalizers = []string{}

					patchCount := expectVMRestorePatch(kubevirtClient, vmRestore, updatedVMRestore)
					controller.processVMRestoreWorkItem()
					Expect(*patchCount).To(Equal(1))
				})
			})

			Describe("restore vm with TargetReadinessPolicy", func() {
				It("WaitEventually - should not fail even if grace period passed", func() {
					r := createRestoreWithOwner()
					r.Spec.TargetReadinessPolicy = pointer.P(snapshotv1.VirtualMachineRestoreWaitEventually)
					//change creation time such that it will make the default grace period pass
					// with TargetReadinessPolicy waitEventually restore should not fail
					r.CreationTimestamp = metav1.Time{Time: r.CreationTimestamp.Time.Add(-snapshotv1.DefaultGracePeriod)}
					vm := createModifiedVM()
					vmi := createVMI(vm)
					rc := r.DeepCopy()
					rc.ResourceVersion = "1"
					rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
						},
					}
					vmSource.Add(vm)
					vmiSource.Add(vmi)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("StopTarget - should call stop on VM target", func() {
					r := createRestoreWithOwner()
					r.Spec.TargetReadinessPolicy = pointer.P(snapshotv1.VirtualMachineRestoreStopTarget)
					vm := createModifiedVM()
					vmi := createVMI(vm)
					rc := r.DeepCopy()
					rc.ResourceVersion = "1"
					rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
						},
					}
					vmSource.Add(vm)
					vmiSource.Add(vmi)
					vmInterface.EXPECT().Stop(context.Background(), vm.Name, &kubevirtv1.StopOptions{}).Return(nil).Times(1)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("default - GracePeriodAndFail - should fail when grace period passed", func() {
					r := createRestoreWithOwner()
					//change creation time such that it will make the default grace period pass
					// with default TargetReadinessPolicy GracePeriodAndFail restore should fail once
					// the grace period passed
					r.CreationTimestamp = metav1.Time{Time: r.CreationTimestamp.Time.Add(-snapshotv1.DefaultGracePeriod)}
					vm := createModifiedVM()
					vmi := createVMI(vm)
					rc := r.DeepCopy()
					rc.ResourceVersion = "1"
					rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Operation failed"),
							newReadyCondition(corev1.ConditionFalse, "Operation failed"),
							newFailureCondition(corev1.ConditionTrue, "Restore target failed to be ready within 5m0s"),
						},
					}
					vmSource.Add(vm)
					vmiSource.Add(vmi)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("default - GracePeriodAndFail - should not fail if grace period passed but target became ready", func() {
					r := createRestoreWithOwner()
					r.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for target to be ready"),
						},
					}
					//change creation time such that it will make the default grace period pass
					// with default TargetReadinessPolicy GracePeriodAndFail if target became ready
					// in that time restore should not fail
					r.CreationTimestamp = metav1.Time{Time: r.CreationTimestamp.Time.Add(-snapshotv1.DefaultGracePeriod)}

					rc := r.DeepCopy()
					rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
						},
					}
					addInitialVolumeRestores(rc)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)

					vm := createModifiedVM()
					vmSource.Add(vm)
					expectUpdateVMRestoreInProgress(vm)

					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("FailImmediate - should fail immediately", func() {
					r := createRestoreWithOwner()
					r.Spec.TargetReadinessPolicy = pointer.P(snapshotv1.VirtualMachineRestoreFailImmediate)
					vm := createModifiedVM()
					vmi := createVMI(vm)
					rc := r.DeepCopy()
					rc.ResourceVersion = "1"
					rc.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Operation failed"),
							newReadyCondition(corev1.ConditionFalse, "Operation failed"),
							newFailureCondition(corev1.ConditionTrue, "Restore target not ready"),
						},
					}
					vmSource.Add(vm)
					vmiSource.Add(vmi)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
					Expect(*updateStatusCalls).To(Equal(1))
				})
			})

			It("Failed restore should be terminating state", func() {
				r := createRestoreWithOwner()
				r.OwnerReferences = nil
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Operation failed"),
						newReadyCondition(corev1.ConditionFalse, "Operation failed"),
						newFailureCondition(corev1.ConditionTrue, "Restore target not ready"),
					},
				}
				vm := createModifiedVM()
				vmi := createVMI(vm)
				vmSource.Add(vm)
				vmiSource.Add(vmi)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
			})
		})

		It("should create restore PVCs with populated dataSourceRef and dataSource", func() {
			// Mock the restore environment from scratch, so we use source PVCs with dataSourceRef
			vm := createSnapshotVM()
			pvcs := createPVCsForVMWithDataSourceRef(vm)
			s := createSnapshot()
			sc := createVirtualMachineSnapshotContent(s, vm, pvcs)
			storageClass := createStorageClass()
			s.Status.VirtualMachineSnapshotContentName = &sc.Name
			sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
				CreationTime: timeFunc(),
				ReadyToUse:   pointer.P(true),
			}
			vmSnapshotSource.Add(s)
			vmSnapshotContentSource.Add(sc)
			storageClassSource.Add(storageClass)

			// Actual test
			r := createRestoreWithOwner()
			vm = createModifiedVM()
			r.Status = &snapshotv1.VirtualMachineRestoreStatus{
				Complete: pointer.P(false),
				Conditions: []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				},
			}
			vmSource.Add(vm)
			addVolumeRestores(r)
			pvcSize := resource.MustParse("2Gi")
			vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
			fakeVolumeSnapshotProvider.Add(vs)
			expectUpdateVMRestoreInProgress(vm)
			calls := expectPVCCreateWithDataSourceRef(k8sClient, r, pvcSize)
			addVirtualMachineRestore(r)
			controller.processVMRestoreWorkItem()
			Expect(*calls).To(Equal(1))
		})

		Describe("restore vm with instancetypes and preferences", func() {
			var (
				vmSnapshot             *snapshotv1.VirtualMachineSnapshot
				originalVM             *kubevirtv1.VirtualMachine
				vmSnapshotContent      *snapshotv1.VirtualMachineSnapshotContent
				restore                *snapshotv1.VirtualMachineRestore
				instancetypeObj        *instancetypev1beta1.VirtualMachineInstancetype
				instancetypeSnapshotCR *appsv1.ControllerRevision
				instancetypeOriginalCR *appsv1.ControllerRevision
				preferenceObj          *instancetypev1beta1.VirtualMachinePreference
				preferenceSnapshotCR   *appsv1.ControllerRevision
				preferenceOriginalCR   *appsv1.ControllerRevision
			)

			const vmCreationFailureMessage = "something failed during VirtualMachine creation"

			expectCreateVM := func(vm *kubevirtv1.VirtualMachine) {
				newVMUID := vm.UID
				vm.UID = ""
				vm.ResourceVersion = ""
				vm.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
				vmInterface.EXPECT().
					Create(context.Background(), vm, metav1.CreateOptions{}).
					Do(func(ctx context.Context, newVM *kubevirtv1.VirtualMachine, options metav1.CreateOptions) {
						vm.UID = newVMUID
					}).Return(vm, nil).Times(1)
			}

			expectUpdateVMRestored := func(vm *kubevirtv1.VirtualMachine) {
				expectedUpdatedVM := vm.DeepCopy()
				expectedUpdatedVM.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
				vmInterface.EXPECT().
					Update(context.Background(), expectedUpdatedVM, metav1.UpdateOptions{}).
					Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
						updatedVM := obj.(*kubevirtv1.VirtualMachine)
						Expect(*updatedVM).To(Equal(*expectedUpdatedVM))
					}).Return(expectedUpdatedVM, nil).Times(1)
			}

			expectUpdateVMRestoreUpdatingTargetSpec := func(vmRestore *snapshotv1.VirtualMachineRestore, resourceVersion string) *int {
				expectedUpdatedRestore := vmRestore.DeepCopy()
				expectedUpdatedRestore.ResourceVersion = resourceVersion
				expectedUpdatedRestore.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
				}
				return expectVMRestoreUpdateStatus(kubevirtClient, expectedUpdatedRestore)
			}

			expectUpdateVMRestoreFailure := func(vmRestore *snapshotv1.VirtualMachineRestore, resourceVersion, failureReason string) *int {
				expectedUpdatedRestore := vmRestore.DeepCopy()
				expectedUpdatedRestore.ResourceVersion = resourceVersion
				expectedUpdatedRestore.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, failureReason),
					newReadyCondition(corev1.ConditionFalse, failureReason),
				}
				return expectVMRestoreUpdateStatus(kubevirtClient, expectedUpdatedRestore)
			}

			expectCreateVMFailure := func(vm *kubevirtv1.VirtualMachine) {
				newVMUID := vm.UID
				vm.UID = ""
				vm.ResourceVersion = ""
				vm.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
				vmInterface.EXPECT().
					Create(context.Background(), vm, metav1.CreateOptions{}).
					Do(func(ctx context.Context, newVM *kubevirtv1.VirtualMachine, options metav1.CreateOptions) {
						vm.UID = newVMUID
					}).Return(vm, fmt.Errorf(vmCreationFailureMessage))
			}

			getInstancetypeOriginalCR := func() *appsv1.ControllerRevision { return instancetypeOriginalCR }
			getPreferenceOriginalCR := func() *appsv1.ControllerRevision { return preferenceOriginalCR }
			nilInstancetypeMatcher := func() *kubevirtv1.InstancetypeMatcher { return nil }
			nilPrefrenceMatcher := func() *kubevirtv1.PreferenceMatcher { return nil }

			BeforeEach(func() {
				virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

				originalVM = createSnapshotVM()
				originalVM.Spec.DataVolumeTemplates = []kubevirtv1.DataVolumeTemplateSpec{}
				restore = createRestoreWithOwner()

				vmSnapshot = createSnapshot()
				vmSnapshotContent = createVirtualMachineSnapshotContent(vmSnapshot, originalVM, nil)
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}

				vmSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				vmSnapshotSource.Add(vmSnapshot)

				instancetypeObj = createInstancetype()
				var err error
				instancetypeOriginalCR, err = revision.CreateControllerRevision(originalVM, instancetypeObj)
				Expect(err).ToNot(HaveOccurred())
				crSource.Add(instancetypeOriginalCR)

				instancetypeSnapshotCR = createInstancetypeVirtualMachineSnapshotCR(originalVM, vmSnapshot, instancetypeObj)
				crSource.Add(instancetypeSnapshotCR)

				preferenceObj = createPreference()
				preferenceOriginalCR, err = revision.CreateControllerRevision(originalVM, preferenceObj)
				Expect(err).ToNot(HaveOccurred())
				crSource.Add(preferenceOriginalCR)

				preferenceSnapshotCR = createInstancetypeVirtualMachineSnapshotCR(originalVM, vmSnapshot, preferenceObj)
				crSource.Add(preferenceSnapshotCR)
			})

			DescribeTable("with an existing VirtualMachine",
				func(getVMInstancetypeMatcher, getSnapshotInstancetypeMatcher func() *kubevirtv1.InstancetypeMatcher, getVMPreferenceMatcher, getSnapshotPreferenceMatcher func() *kubevirtv1.PreferenceMatcher, getExpectedCR func() *appsv1.ControllerRevision) {
					originalVM.Spec.Instancetype = getVMInstancetypeMatcher()
					originalVM.Spec.Preference = getVMPreferenceMatcher()
					vmSource.Add(originalVM)

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					vmSnapshotContentSource.Add(vmSnapshotContent)

					updatedVM := expectUpdateVMRestoreInProgress(originalVM)
					expectUpdateVMRestored(updatedVM)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
				},
				Entry("and referenced instancetype",
					func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeOriginalCR.Name,
						}
					}, func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeSnapshotCR.Name,
						}
					}, nilPrefrenceMatcher, nilPrefrenceMatcher, getInstancetypeOriginalCR,
				),
				Entry("and referenced preference", nilInstancetypeMatcher, nilInstancetypeMatcher,
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceOriginalCR.Name,
						}
					},
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceSnapshotCR.Name,
						}
					}, getPreferenceOriginalCR,
				),
			)

			DescribeTable("with a new VirtualMachine",
				func(getVMInstancetypeMatcher, getSnapshotInstancetypeMatcher func() *kubevirtv1.InstancetypeMatcher, getVMPreferenceMatcher, getSnapshotPreferenceMatcher func() *kubevirtv1.PreferenceMatcher, getExpectedCR func() *appsv1.ControllerRevision) {
					originalVM.Spec.Instancetype = getVMInstancetypeMatcher()
					originalVM.Spec.Preference = getVMPreferenceMatcher()
					vmSource.Add(originalVM)

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					vmSnapshotContentSource.Add(vmSnapshotContent)

					// Ensure we restore into a new VM
					newVM := originalVM.DeepCopy()
					newVM.Name = "newvm"
					newVM.UID = "newvm-uid"
					restore.Spec.Target.Name = newVM.Name

					originalCR := getExpectedCR()
					expectedCreatedCR := originalCR.DeepCopy()
					expectedCreatedCR.Name = strings.Replace(expectedCreatedCR.Name, originalVM.Name, newVM.Name, 1)
					expectedCreatedCR.OwnerReferences = nil
					crCreates := expectControllerRevisionCreate(k8sClient, expectedCreatedCR)

					// We need to be able to find the created CR from the controller so add it to the source
					expectedCreatedCR.Namespace = testNamespace
					crSource.Add(expectedCreatedCR)

					expectedUpdatedCR := expectedCreatedCR.DeepCopy()
					expectedUpdatedCR.ResourceVersion = "5"
					expectedUpdatedCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(newVM, kubevirtv1.VirtualMachineGroupVersionKind)}
					crUpdates := expectControllerRevisionUpdate(k8sClient, expectedUpdatedCR)

					if newVM.Spec.Instancetype != nil {
						newVM.Spec.Instancetype.RevisionName = expectedCreatedCR.Name
					}
					if newVM.Spec.Preference != nil {
						newVM.Spec.Preference.RevisionName = expectedCreatedCR.Name
					}
					expectCreateVM(newVM)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
					Expect(*crCreates).To(Equal(1))
					Expect(*crUpdates).To(Equal(1))
				},
				Entry("and referenced instancetype",
					func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeOriginalCR.Name,
						}
					}, func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeSnapshotCR.Name,
						}
					}, nilPrefrenceMatcher, nilPrefrenceMatcher, getInstancetypeOriginalCR,
				),
				Entry("and referenced preference", nilInstancetypeMatcher, nilInstancetypeMatcher,
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceOriginalCR.Name,
						}
					},
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceSnapshotCR.Name,
						}
					}, getPreferenceOriginalCR,
				),
			)
			DescribeTable("with a failure during VirtualMachine creation",
				func(getVMInstancetypeMatcher, getSnapshotInstancetypeMatcher func() *kubevirtv1.InstancetypeMatcher, getVMPreferenceMatcher, getSnapshotPreferenceMatcher func() *kubevirtv1.PreferenceMatcher, getExpectedCR func() *appsv1.ControllerRevision) {
					originalVM.Spec.Instancetype = getVMInstancetypeMatcher()
					originalVM.Spec.Preference = getVMPreferenceMatcher()
					vmSource.Add(originalVM)

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					vmSnapshotContentSource.Add(vmSnapshotContent)

					// Ensure we restore into a new VM
					newVM := originalVM.DeepCopy()
					newVM.Name = "newvm"
					newVM.UID = "newvm-uid"
					restore.Spec.Target.Name = newVM.Name

					originalCR := getExpectedCR()
					expectedCreatedCR := originalCR.DeepCopy()
					expectedCreatedCR.Name = strings.Replace(expectedCreatedCR.Name, originalVM.Name, newVM.Name, 1)
					expectedCreatedCR.OwnerReferences = nil
					crCreates := expectControllerRevisionCreate(k8sClient, expectedCreatedCR)

					// We need to be able to find the created CR from the controller so add it to the source
					expectedCreatedCR.Namespace = testNamespace
					crSource.Add(expectedCreatedCR)

					expectedUpdatedCR := expectedCreatedCR.DeepCopy()
					expectedUpdatedCR.ResourceVersion = "5"
					expectedUpdatedCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(newVM, kubevirtv1.VirtualMachineGroupVersionKind)}
					crUpdates := expectControllerRevisionUpdate(k8sClient, expectedUpdatedCR)

					if newVM.Spec.Instancetype != nil {
						newVM.Spec.Instancetype.RevisionName = expectedCreatedCR.Name
					}
					if newVM.Spec.Preference != nil {
						newVM.Spec.Preference.RevisionName = expectedCreatedCR.Name
					}

					expectCreateVMFailure(newVM)
					failCalls := expectUpdateVMRestoreFailure(restore, "1", vmCreationFailureMessage)

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()

					// We have already created the ControllerRevision but that shouldn't stop the reconcile from progressing
					alreadyExistsCalls := expectCreateControllerRevisionAlreadyExists(k8sClient, expectedCreatedCR)
					expectCreateVM(newVM)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "2")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
					Expect(*failCalls).To(Equal(1))
					Expect(*crCreates).To(Equal(1))
					Expect(*alreadyExistsCalls).To(Equal(1))
					Expect(*crUpdates).To(Equal(1))
				},
				Entry("and referenced instancetype",
					func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeOriginalCR.Name,
						}
					}, func() *kubevirtv1.InstancetypeMatcher {
						return &kubevirtv1.InstancetypeMatcher{
							Name:         instancetypeObj.Name,
							Kind:         instancetypeapi.SingularResourceName,
							RevisionName: instancetypeSnapshotCR.Name,
						}
					}, nilPrefrenceMatcher, nilPrefrenceMatcher, getInstancetypeOriginalCR,
				),
				Entry("and referenced preference", nilInstancetypeMatcher, nilInstancetypeMatcher,
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceOriginalCR.Name,
						}
					},
					func() *kubevirtv1.PreferenceMatcher {
						return &kubevirtv1.PreferenceMatcher{
							Name:         preferenceObj.Name,
							Kind:         instancetypeapi.SingularPreferenceResourceName,
							RevisionName: preferenceSnapshotCR.Name,
						}
					}, getPreferenceOriginalCR,
				),
			)

			It("Should recreate existing CR if it contains different data to the Snapshot CR", func() {
				// Take a copy of the original CR used by the tests, we expect to see an attempt to recreate later
				instancetypeOriginalCRCopy := instancetypeOriginalCR.DeepCopy()

				// Modify the original CR so it differs from the already generated instancetypeSnapshotCR
				instancetypeObj.Spec.CPU.Guest = uint32(5)
				instancetypeOriginalCR, err := revision.CreateControllerRevision(originalVM, instancetypeObj)
				Expect(err).ToNot(HaveOccurred())
				crSource.Modify(instancetypeOriginalCR)

				originalVM.Spec.Instancetype = &kubevirtv1.InstancetypeMatcher{
					Name:         instancetypeObj.Name,
					Kind:         instancetypeapi.SingularResourceName,
					RevisionName: instancetypeOriginalCR.Name,
				}
				vmSource.Add(originalVM)

				vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = &kubevirtv1.InstancetypeMatcher{
					Name:         instancetypeObj.Name,
					Kind:         instancetypeapi.SingularResourceName,
					RevisionName: instancetypeSnapshotCR.Name,
				}
				vmSnapshotContentSource.Add(vmSnapshotContent)

				// We expect the original CR to be deleted and recreated with the correct data
				crDeletes := expectControllerRevisionDelete(k8sClient, instancetypeOriginalCR.Name)
				crCreates := expectControllerRevisionCreate(k8sClient, instancetypeOriginalCRCopy)

				updatedVM := expectUpdateVMRestoreInProgress(originalVM)
				expectUpdateVMRestored(updatedVM)
				calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

				addVirtualMachineRestore(restore)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
				Expect(*crCreates).To(Equal(1))
				Expect(*crDeletes).To(Equal(1))
			})
		})
	})
})

func expectPVCCreates(client *k8sfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore, expectedSize resource.Quantity) *int {
	calls := 0
	client.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*corev1.PersistentVolumeClaim)
		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vr.PersistentVolumeClaimName == createObj.Name {
				Expect(createObj.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(expectedSize))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())

		calls++
		return true, create.GetObject(), nil
	})
	return &calls
}

func expectPVCCreateWithDataSourceRef(client *k8sfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore, expectedSize resource.Quantity) *int {
	calls := 0
	client.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*corev1.PersistentVolumeClaim)
		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vr.PersistentVolumeClaimName == createObj.Name {
				Expect(createObj.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(expectedSize))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())

		Expect(createObj.Spec.DataSource).ToNot(BeNil())
		Expect(createObj.Spec.DataSourceRef).ToNot(BeNil())

		dataSourceRef := &corev1.TypedLocalObjectReference{
			APIGroup: createObj.Spec.DataSourceRef.APIGroup,
			Kind:     createObj.Spec.DataSourceRef.Kind,
			Name:     createObj.Spec.DataSourceRef.Name,
		}
		Expect(createObj.Spec.DataSource).To(Equal(dataSourceRef))

		calls++

		return true, create.GetObject(), nil
	})
	return &calls
}

func expectPVCUpdates(client *k8sfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore) *int {
	calls := 0
	client.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*corev1.PersistentVolumeClaim)
		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vr.DataVolumeName != nil && *vr.DataVolumeName == updateObj.Annotations["cdi.kubevirt.io/storage.populatedFor"] {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())

		calls++

		return true, update.GetObject(), nil
	})
	return &calls
}

func expectVMRestoreUpdate(client *kubevirtfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachinerestores", func(action testing.Action) (bool, runtime.Object, error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		if update.GetSubresource() != "" {
			return false, nil, nil
		}

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineRestore)

		calls++

		return reflect.DeepEqual(vmRestore.ObjectMeta, updateObj.ObjectMeta) && reflect.DeepEqual(vmRestore.Spec, updateObj.Spec),
			update.GetObject(),
			nil
	})
	return &calls
}

func expectVMRestoreUpdateStatus(client *kubevirtfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachinerestores", func(action testing.Action) (bool, runtime.Object, error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		if update.GetSubresource() != "status" {
			return false, nil, nil
		}

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineRestore)

		calls++

		return reflect.DeepEqual(vmRestore.Status, updateObj.Status), update.GetObject(), nil
	})
	return &calls
}

func expectVMRestorePatch(client *kubevirtfake.Clientset, orig, newObj *snapshotv1.VirtualMachineRestore) *int {
	calls := 0
	client.Fake.PrependReactor("patch", "virtualmachinerestores", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		patch, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())

		patched := &snapshotv1.VirtualMachineRestore{}
		if err := applyPatch(patch.GetPatch(), orig, patched); err != nil {
			return false, nil, err
		}

		Expect(patched).To(Equal(newObj))

		calls++

		return true, patched, nil
	})
	return &calls
}

func expectDataVolumeCreate(client *cdifake.Clientset, name string) *int {
	calls := 0
	client.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		a, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		dv, ok := a.GetObject().(*cdiv1.DataVolume)
		Expect(ok).To(BeTrue())
		Expect(dv.Name).To(Equal(name))
		Expect(dv.Annotations[cdiv1.AnnPrePopulated]).To(Equal("true"))

		calls++

		return true, nil, nil
	})
	return &calls
}

func expectDataVolumeDeletes(client *cdifake.Clientset, names []string) *int {
	calls := 0
	client.Fake.PrependReactor("delete", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		a, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())

		dvName := a.GetName()
		found := false
		for _, name := range names {
			if name == dvName {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())

		calls++

		return true, nil, nil
	})
	return &calls
}

// A mock to implement volumeSnapshotProvider interface
type MockVolumeSnapshotProvider struct {
	volumeSnapshots []*vsv1.VolumeSnapshot
}

func (v *MockVolumeSnapshotProvider) GetVolumeSnapshot(namespace, name string) (*vsv1.VolumeSnapshot, error) {
	if len(v.volumeSnapshots) == 0 {
		return nil, nil
	}
	vs := v.volumeSnapshots[0]
	v.volumeSnapshots = v.volumeSnapshots[1:]
	return vs, nil
}

func (v *MockVolumeSnapshotProvider) Add(s *vsv1.VolumeSnapshot) {
	v.volumeSnapshots = append(v.volumeSnapshots, s)
}
