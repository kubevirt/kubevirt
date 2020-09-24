package snapshot

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	k8ssnapshotfake "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned/fake"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/status"
)

const (
	testNamespace = "default"
	vmSnapshotUID = "snapshot-uid"
	contentUID    = "content-uid"
)

var (
	vmUID            types.UID = "vm-uid"
	vmAPIGroup                 = "kubevirt.io"
	storageClassName           = "rook-ceph-block"
	t                          = true
	f                          = false
)

var _ = Describe("Snapshot controlleer", func() {
	var (
		timeStamp               = metav1.Now()
		vmName                  = "testvm"
		vmSnapshotName          = "test-snapshot"
		retain                  = snapshotv1.VirtualMachineSnapshotContentRetain
		volumeSnapshotClassName = "csi-rbdplugin-snapclass"
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
			ReadyToUse:   &t,
			CreationTime: timeFunc(),
		}

		return vms
	}

	createVMSnapshotInProgress := func() *snapshotv1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse: &f,
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
		}
	}

	createPersistentVolumeClaims := func() []corev1.PersistentVolumeClaim {
		return createPVCsForVM(createLockedVM())
	}

	createVMSnapshotContent := func() *snapshotv1.VirtualMachineSnapshotContent {
		vmSnapshot := createVMSnapshotInProgress()
		vm := createLockedVM()
		return createVirtualMachineSnapshotContent(vmSnapshot, vm)
	}

	createStorageClass := func() *storagev1.StorageClass {
		return &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: storageClassName,
			},
			Provisioner: "rook-ceph.rbd.csi.ceph.com",
		}
	}

	createVolumeSnapshots := func(content *snapshotv1.VirtualMachineSnapshotContent) []vsv1beta1.VolumeSnapshot {
		var volumeSnapshots []vsv1beta1.VolumeSnapshot
		for _, vb := range content.Spec.VolumeBackups {
			if vb.VolumeSnapshotName == nil {
				continue
			}
			vs := vsv1beta1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *vb.VolumeSnapshotName,
					Namespace: content.Namespace,
				},
				Spec: vsv1beta1.VolumeSnapshotSpec{
					Source: vsv1beta1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &vb.PersistentVolumeClaim.Name,
					},
				},
				Status: &vsv1beta1.VolumeSnapshotStatus{
					ReadyToUse: &f,
				},
			}
			volumeSnapshots = append(volumeSnapshots, vs)
		}
		return volumeSnapshots
	}

	createVolumeSnapshotClasses := func() []vsv1beta1.VolumeSnapshotClass {
		return []vsv1beta1.VolumeSnapshotClass{
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
					Name: volumeSnapshotClassName + "alt",
				},
				Driver: "rook-ceph.rbd.csi.ceph.com",
			},
		}
	}

	Context("One valid Snapshot controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVirtualMachineInterface
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
		var pvcInformer cache.SharedIndexInformer
		var pvcSource *framework.FakeControllerSource
		var crdInformer cache.SharedIndexInformer
		var crdSource *framework.FakeControllerSource
		var stop chan struct{}
		var controller *VMSnapshotController
		var recorder *record.FakeRecorder
		var mockVMSnapshotQueue *testutils.MockWorkQueue
		var mockVMSnapshotContentQueue *testutils.MockWorkQueue
		var mockCRDQueue *testutils.MockWorkQueue

		var vmSnapshotClient *kubevirtfake.Clientset
		var k8sSnapshotClient *k8ssnapshotfake.Clientset
		var k8sClient *k8sfake.Clientset

		syncCaches := func(stop chan struct{}) {
			go vmSnapshotInformer.Run(stop)
			go vmSnapshotContentInformer.Run(stop)
			go vmInformer.Run(stop)
			go storageClassInformer.Run(stop)
			go pvcInformer.Run(stop)
			go crdInformer.Run(stop)
			go vmiInformer.Run(stop)
			go podInformer.Run(stop)
			Expect(cache.WaitForCacheSync(
				stop,
				vmSnapshotInformer.HasSynced,
				vmSnapshotContentInformer.HasSynced,
				vmInformer.HasSynced,
				storageClassInformer.HasSynced,
				pvcInformer.HasSynced,
				crdInformer.HasSynced,
				vmiInformer.HasSynced,
				podInformer.HasSynced,
			)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

			vmSnapshotInformer, vmSnapshotSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineSnapshot{}, cache.Indexers{
				"vm": func(obj interface{}) ([]string, error) {
					vms := obj.(*snapshotv1.VirtualMachineSnapshot)
					if vms.Spec.Source.APIGroup != nil &&
						*vms.Spec.Source.APIGroup == v1.GroupName &&
						vms.Spec.Source.Kind == "VirtualMachine" {
						return []string{vms.Spec.Source.Name}, nil
					}
					return nil, nil
				},
			})
			vmSnapshotContentInformer, vmSnapshotContentSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineSnapshotContent{}, cache.Indexers{
				"volumeSnapshot": func(obj interface{}) ([]string, error) {
					vmsc := obj.(*snapshotv1.VirtualMachineSnapshotContent)
					var volumeSnapshots []string
					for _, v := range vmsc.Spec.VolumeBackups {
						if v.VolumeSnapshotName != nil {
							volumeSnapshots = append(volumeSnapshots, *v.VolumeSnapshotName)
						}
					}
					return volumeSnapshots, nil
				},
			})
			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			podInformer, podSource = testutils.NewFakeInformerFor(&corev1.Pod{})
			volumeSnapshotInformer, volumeSnapshotSource = testutils.NewFakeInformerFor(&vsv1beta1.VolumeSnapshot{})
			volumeSnapshotClassInformer, volumeSnapshotClassSource = testutils.NewFakeInformerFor(&vsv1beta1.VolumeSnapshotClass{})
			storageClassInformer, storageClassSource = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			pvcInformer, pvcSource = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			crdInformer, crdSource = testutils.NewFakeInformerFor(&extv1beta1.CustomResourceDefinition{})

			recorder = record.NewFakeRecorder(100)

			controller = &VMSnapshotController{
				Client:                    virtClient,
				VMSnapshotInformer:        vmSnapshotInformer,
				VMSnapshotContentInformer: vmSnapshotContentInformer,
				VMInformer:                vmInformer,
				VMIInformer:               vmiInformer,
				PodInformer:               podInformer,
				StorageClassInformer:      storageClassInformer,
				PVCInformer:               pvcInformer,
				CRDInformer:               crdInformer,
				Recorder:                  recorder,
				ResyncPeriod:              60 * time.Second,
				vmStatusUpdater:           status.NewVMStatusUpdater(virtClient),
			}
			controller.Init()

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockVMSnapshotQueue = testutils.NewMockWorkQueue(controller.vmSnapshotQueue)
			controller.vmSnapshotQueue = mockVMSnapshotQueue

			mockVMSnapshotContentQueue = testutils.NewMockWorkQueue(controller.vmSnapshotContentQueue)
			controller.vmSnapshotContentQueue = mockVMSnapshotContentQueue

			mockCRDQueue = testutils.NewMockWorkQueue(controller.crdQueue)
			controller.crdQueue = mockCRDQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()

			vmSnapshotClient = kubevirtfake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineSnapshot(testNamespace).
				Return(vmSnapshotClient.SnapshotV1alpha1().VirtualMachineSnapshots(testNamespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshotContent(testNamespace).
				Return(vmSnapshotClient.SnapshotV1alpha1().VirtualMachineSnapshotContents(testNamespace)).AnyTimes()

			k8sSnapshotClient = k8ssnapshotfake.NewSimpleClientset()
			virtClient.EXPECT().KubernetesSnapshotClient().Return(k8sSnapshotClient).AnyTimes()

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().StorageV1().Return(k8sClient.StorageV1()).AnyTimes()

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

		addVM := func(vm *v1.VirtualMachine) {
			syncCaches(stop)
			mockVMSnapshotQueue.ExpectAdds(1)
			vmSource.Add(vm)
			mockVMSnapshotQueue.Wait()
		}

		addVolumeSnapshot := func(s *vsv1beta1.VolumeSnapshot) {
			syncCaches(stop)
			mockVMSnapshotContentQueue.ExpectAdds(1)
			volumeSnapshotSource.Add(s)
			mockVMSnapshotContentQueue.Wait()
		}

		addCRD := func(crd *extv1beta1.CustomResourceDefinition) {
			syncCaches(stop)
			mockCRDQueue.ExpectAdds(1)
			crdSource.Add(crd)
			mockCRDQueue.Wait()
		}

		Context("with VolumeSnapshot ad VolumeSnapshotContent informers", func() {

			BeforeEach(func() {
				stopCh := make(chan struct{})
				volumeSnapshotInformer.AddEventHandler(controller.eventHandlerMap[volumeSnapshotCRD])
				controller.dynamicInformerMap[volumeSnapshotCRD].stopCh = stopCh
				controller.dynamicInformerMap[volumeSnapshotCRD].informer = volumeSnapshotInformer
				controller.dynamicInformerMap[volumeSnapshotClassCRD].stopCh = stopCh
				controller.dynamicInformerMap[volumeSnapshotClassCRD].informer = volumeSnapshotClassInformer
				go volumeSnapshotInformer.Run(stopCh)
				go volumeSnapshotClassInformer.Run(stopCh)
			})

			It("should initialize VirtualMachineSnapshot status", func() {
				vmSnapshot := createVMSnapshot()
				vm := createVM()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					ReadyToUse: &f,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				vmSource.Add(vm)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should initialize VirtualMachineSnapshot status (no VM)", func() {
				vmSnapshot := createVMSnapshot()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					ReadyToUse: &f,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Source does not exist"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should initialize VirtualMachineSnapshot status (Progressing)", func() {
				vmSnapshot := createVMSnapshot()
				vm := createLockedVM()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					ReadyToUse: &f,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				vmSource.Add(vm)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should unlock source VirtualMachine", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vm := createLockedVM()
				updatedVM := vm.DeepCopy()
				updatedVM.Finalizers = []string{}
				updatedVM.ResourceVersion = "1"
				vmSource.Add(vm)
				vmInterface.EXPECT().Update(updatedVM).Return(updatedVM, nil)
				statusUpdate := updatedVM.DeepCopy()
				statusUpdate.Status.SnapshotInProgress = nil
				vmInterface.EXPECT().UpdateStatus(statusUpdate).Return(statusUpdate, nil)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should finish unlock source VirtualMachine", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vm := createLockedVM()
				vm.Finalizers = []string{}
				statusUpdate := vm.DeepCopy()
				statusUpdate.ResourceVersion = "1"
				statusUpdate.Status.SnapshotInProgress = nil
				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(statusUpdate).Return(statusUpdate, nil)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should be status error when VM snapshot deleted while in progress", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.DeletionTimestamp = timeFunc()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Snapshot cancelled"),
					newReadyCondition(corev1.ConditionFalse, "Snapshot cancelled"),
				}
				updatedSnapshot.Status.Error = &snapshotv1.Error{
					Time:    timeFunc(),
					Message: &[]string{"Snapshot cancelled"}[0],
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("cleanup when VirtualMachineSnapshot is deleted", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vmSnapshot.DeletionTimestamp = timeFunc()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{}

				content := createVMSnapshotContent()
				updatedContent := content.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Finalizers = []string{}

				vmSnapshotContentSource.Add(content)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				expectVMSnapshotContentDelete(vmSnapshotClient, updatedContent.Name)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("cleanup when VirtualMachineSnapshot is deleted and retain if necessary", func() {
				vmSnapshot := createVMSnapshotSuccess()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vmSnapshot.Spec.DeletionPolicy = &retain
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{}

				content := createVMSnapshotContent()
				updatedContent := content.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Finalizers = []string{}

				vmSnapshotContentSource.Add(content)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should lock source", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmStatusUpdate := vm.DeepCopy()
				vmStatusUpdate.ResourceVersion = "1"
				vmStatusUpdate.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vmStatusUpdate.DeepCopy()
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(vmStatusUpdate).Return(vmStatusUpdate, nil)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should finish lock source", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSource.Add(vm)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should lock source when VM updated", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmStatusUpdate := vm.DeepCopy()
				vmStatusUpdate.ResourceVersion = "1"
				vmStatusUpdate.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vmStatusUpdate.DeepCopy()
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSnapshotSource.Add(vmSnapshot)
				vmInterface.EXPECT().UpdateStatus(vmStatusUpdate).Return(vmStatusUpdate, nil)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil)
				addVM(vm)
				controller.processVMSnapshotWorkItem()
			})

			It("should not lock source if running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &t

				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should not lock source if VMI exists", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &f

				vmiSource.Add(createVMI(vm))
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should not lock source if pods using PVCs", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &f

				pods := createPodsUsingPVCs(vm)
				podSource.Add(&pods[0])
				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should not lock source if another snapshot in progress", func() {
				n := "otherSnapshot"
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Status.SnapshotInProgress = &n

				vmSource.Add(vm)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should init VirtualMachineSnapshot", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.Finalizers = nil
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				updatedSnapshot.ResourceVersion = "1"
				vm := createLockedVM()

				vmSource.Add(vm)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should create VirtualMachineSnapshotContent", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				storageClass := createStorageClass()
				volumeSnapshotClass := &createVolumeSnapshotClasses()[0]
				pvcs := createPersistentVolumeClaims()
				vmSnapshotContent := createVMSnapshotContent()

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				volumeSnapshotClassSource.Add(volumeSnapshotClass)
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}
				expectVMSnapshotContentCreate(vmSnapshotClient, vmSnapshotContent)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
			})

			It("should update VirtualMachineSnapshotStatus", func() {
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   &t,
				}

				vmSnapshot := createVMSnapshotInProgress()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SourceUID = &vmUID
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.CreationTime = timeFunc()
				updatedSnapshot.Status.ReadyToUse = &t
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Operation complete"),
				}

				vm := createLockedVM()

				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should create VolumeSnapshot", func() {
				storageClass := createStorageClass()
				volumeSnapshotClass := &createVolumeSnapshotClasses()[0]
				pvcs := createPersistentVolumeClaims()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: &f,
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				storageClassSource.Add(storageClass)
				volumeSnapshotClassSource.Add(volumeSnapshotClass)
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}

				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should create VolumeSnapshot with multiple VolumeSnapshotClasses", func() {
				storageClass := createStorageClass()
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				pvcs := createPersistentVolumeClaims()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: &f,
				}

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					vss := snapshotv1.VolumeSnapshotStatus{
						VolumeSnapshotName: volumeSnapshots[i].Name,
					}
					updatedContent.Status.VolumeSnapshotStatus = append(updatedContent.Status.VolumeSnapshotStatus, vss)
				}

				storageClassSource.Add(storageClass)
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}

				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClasses[0].Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should update VirtualMachineSnapshotContent", func() {
				vmSnapshotContent := createVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &t,
					CreationTime: timeFunc(),
				}

				vmSnapshotContentSource.Add(vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					volumeSnapshots[i].Status.ReadyToUse = &t
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
			})

			It("should update VirtualMachineSnapshotContent no snapshots", func() {
				vmSnapshotContent := createVMSnapshotContent()
				for i := range vmSnapshotContent.Spec.VolumeBackups {
					vmSnapshotContent.Spec.VolumeBackups[i].VolumeSnapshotName = nil
				}

				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &t,
					CreationTime: timeFunc(),
				}

				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
			})

			It("should update VirtualMachineSnapshotContent on error", func() {
				message := "VolumeSnapshot in error state"
				vmSnapshotContent := createVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse: &f,
					Error: &snapshotv1.Error{
						Message: &message,
						Time:    timeFunc(),
					},
				}

				vmSnapshotContentSource.Add(vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					message := "bad error"
					volumeSnapshots[i].Status.ReadyToUse = &f
					volumeSnapshots[i].Status.CreationTime = nil
					volumeSnapshots[i].Status.Error = &vsv1beta1.VolumeSnapshotError{
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
				}

				controller.processVMSnapshotContentWorkItem()
			})

			It("should update VirtualMachineSnapshotContent when VolumeSnapshot deleted", func() {
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &t,
					CreationTime: timeFunc(),
				}
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "2"
				updatedContent.Status.ReadyToUse = &f
				updatedContent.Status.CreationTime = nil

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				var volumeSnapshotNames []string
				for i := range volumeSnapshots {
					volumeSnapshotNames = append(volumeSnapshotNames, volumeSnapshots[i].Name)
				}
				errorMessage := fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(volumeSnapshotNames, ","))
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &f,
					CreationTime: timeFunc(),
					Error: &snapshotv1.Error{
						Message: &errorMessage,
						Time:    timeFunc(),
					},
				}

				vmSnapshotContentSource.Add(vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "VolumeSnapshotMissing")
			})

			table.DescribeTable("should delete informer", func(crdName string) {
				crd := &extv1beta1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name:              crdName,
						DeletionTimestamp: timeFunc(),
					},
					Spec: extv1beta1.CustomResourceDefinitionSpec{
						Versions: []extv1beta1.CustomResourceDefinitionVersion{
							{
								Name:   "v1beta1",
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
				table.Entry("for VolumeSnapshot", volumeSnapshotCRD),
				table.Entry("for VolumeSnapshotClass", volumeSnapshotClassCRD),
			)
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

			table.DescribeTable("should create informer", func(crdName string) {
				crd := &extv1beta1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: crdName,
					},
					Spec: extv1beta1.CustomResourceDefinitionSpec{
						Versions: []extv1beta1.CustomResourceDefinitionVersion{
							{
								Name:   "v1beta1",
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
				table.Entry("for VolumeSnapshot", volumeSnapshotCRD),
				table.Entry("for VolumeSnapshotClass", volumeSnapshotClassCRD),
			)
		})
	})
})

func expectVMSnapshotUpdate(client *kubevirtfake.Clientset, vmSnapshot *snapshotv1.VirtualMachineSnapshot) {
	client.Fake.PrependReactor("update", "virtualmachinesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineSnapshot)
		Expect(updateObj).To(Equal(vmSnapshot))

		return true, update.GetObject(), nil
	})
}

func expectVMSnapshotContentCreate(client *kubevirtfake.Clientset, content *snapshotv1.VirtualMachineSnapshotContent) {
	client.Fake.PrependReactor("create", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*snapshotv1.VirtualMachineSnapshotContent)
		Expect(createObj).To(Equal(content))

		return true, create.GetObject(), nil
	})
}

func expectVMSnapshotContentUpdate(client *kubevirtfake.Clientset, content *snapshotv1.VirtualMachineSnapshotContent) {
	client.Fake.PrependReactor("update", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*snapshotv1.VirtualMachineSnapshotContent)
		Expect(updateObj).To(Equal(content))

		return true, update.GetObject(), nil
	})
}

func expectVMSnapshotContentDelete(client *kubevirtfake.Clientset, name string) {
	client.Fake.PrependReactor("delete", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())

		Expect(delete.GetName()).To(Equal(name))

		return true, nil, nil
	})
}

func expectVolumeSnapshotCreates(
	client *k8ssnapshotfake.Clientset,
	voluemSnapshotClass string,
	content *snapshotv1.VirtualMachineSnapshotContent,
) {
	volumeSnapshots := map[string]string{}
	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName != nil {
			volumeSnapshots[*volumeBackup.VolumeSnapshotName] = volumeBackup.PersistentVolumeClaim.Name
		}
	}
	client.Fake.PrependReactor("create", "volumesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*vsv1beta1.VolumeSnapshot)
		pvc, ok := volumeSnapshots[createObj.Name]
		Expect(ok).To(BeTrue())
		Expect(pvc).Should(Equal(*createObj.Spec.Source.PersistentVolumeClaimName))
		delete(volumeSnapshots, createObj.Name)

		Expect(*createObj.Spec.VolumeSnapshotClassName).Should(Equal(voluemSnapshotClass))
		Expect(createObj.OwnerReferences[0].Name).Should(Equal(content.Name))
		Expect(createObj.OwnerReferences[0].UID).Should(Equal(content.UID))

		return true, createObj, nil
	})
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
			Running: &f,
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
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
							},
						},
						Resources: v1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceRequestsMemory): resource.MustParse("64M"),
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "disk1",
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
					Spec: cdiv1alpha1.DataVolumeSpec{
						Source: cdiv1alpha1.DataVolumeSource{
							HTTP: &cdiv1alpha1.DataVolumeSourceHTTP{
								URL: "http://cdi-http-import-server.kubevirt/images/alpine.iso",
							},
						},
						PVC: &corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
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
		},
	}
}

func createVirtualMachineSnapshotContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot, vm *v1.VirtualMachine) *snapshotv1.VirtualMachineSnapshotContent {
	var volumeBackups []snapshotv1.VolumeBackup
	vm.ResourceVersion = "1"
	vm.Status = v1.VirtualMachineStatus{}

	for i, pvc := range createPVCsForVM(vm) {
		diskName := fmt.Sprintf("disk%d", i+1)
		volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-volume-%s", vmSnapshot.UID, diskName)
		vb := snapshotv1.VolumeBackup{
			VolumeName:            diskName,
			PersistentVolumeClaim: pvc,
			VolumeSnapshotName:    &volumeSnapshotName,
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
				VirtualMachine: vm,
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
