package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/api/core"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	k8ssnapshotfake "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned/fake"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/status"
)

const (
	testNamespace = "default"
	vmSnapshotUID = "snapshot-uid"
	contentUID    = "content-uid"
)

var (
	vmUID             types.UID        = "vm-uid"
	vmAPIGroup                         = "kubevirt.io"
	storageClassName                   = "rook-ceph-block"
	t                                  = true
	f                                  = false
	noFailureDeadline *metav1.Duration = &metav1.Duration{Duration: 0}
)

var _ = Describe("Snapshot controlleer", func() {
	var (
		timeStamp               = metav1.Now()
		vmName                  = "testvm"
		vmRevisionName          = "testvm-revision"
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
			SourceUID:    &vmUID,
			CreationTime: timeFunc(),
			Conditions: []snapshotv1.Condition{
				newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
				newReadyCondition(corev1.ConditionTrue, "Operation complete"),
			},
			Phase: snapshotv1.Succeeded,
		}

		return vms
	}

	createVMSnapshotInProgress := func() *snapshotv1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse: &f,
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
		var mockVMSnapshotQueue *testutils.MockWorkQueue
		var mockVMSnapshotContentQueue *testutils.MockWorkQueue
		var mockCRDQueue *testutils.MockWorkQueue
		var mockVMQueue *testutils.MockWorkQueue

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
			go dvInformer.Run(stop)
			go crInformer.Run(stop)
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
				dvInformer.HasSynced,
				crInformer.HasSynced,
			)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

			vmSnapshotInformer, vmSnapshotSource = testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineSnapshot{}, cache.Indexers{
				"vm": func(obj interface{}) ([]string, error) {
					vms := obj.(*snapshotv1.VirtualMachineSnapshot)
					if vms.Spec.Source.APIGroup != nil &&
						*vms.Spec.Source.APIGroup == core.GroupName &&
						vms.Spec.Source.Kind == "VirtualMachine" {
						return []string{vms.Namespace + "/" + vms.Spec.Source.Name}, nil
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
							volumeSnapshots = append(volumeSnapshots, vmsc.Namespace+"/"+*v.VolumeSnapshotName)
						}
					}
					return volumeSnapshots, nil
				},
			})
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
			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			podInformer, podSource = testutils.NewFakeInformerFor(&corev1.Pod{})
			volumeSnapshotInformer, volumeSnapshotSource = testutils.NewFakeInformerFor(&vsv1beta1.VolumeSnapshot{})
			volumeSnapshotClassInformer, volumeSnapshotClassSource = testutils.NewFakeInformerFor(&vsv1beta1.VolumeSnapshotClass{})
			storageClassInformer, storageClassSource = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
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
				PVCInformer:               pvcInformer,
				CRDInformer:               crdInformer,
				DVInformer:                dvInformer,
				CRInformer:                crInformer,
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

			mockVMQueue = testutils.NewMockWorkQueue(controller.vmQueue)
			controller.vmQueue = mockVMQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface).AnyTimes()

			vmSnapshotClient = kubevirtfake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineSnapshot(testNamespace).
				Return(vmSnapshotClient.SnapshotV1alpha1().VirtualMachineSnapshots(testNamespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshotContent(testNamespace).
				Return(vmSnapshotClient.SnapshotV1alpha1().VirtualMachineSnapshotContents(testNamespace)).AnyTimes()

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

		addCRD := func(crd *extv1.CustomResourceDefinition) {
			syncCaches(stop)
			mockCRDQueue.ExpectAdds(1)
			crdSource.Add(crd)
			mockCRDQueue.Wait()
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
			})

			It("should initialize VirtualMachineSnapshot status", func() {
				vmSnapshot := createVMSnapshot()
				vm := createVM()
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: &f,
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
					Indications: []snapshotv1.Indication{},
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
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					ReadyToUse: &f,
					Phase:      snapshotv1.InProgress,
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
				updatedSnapshot.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: &f,
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
					Indications: []snapshotv1.Indication{},
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
				vmInterface.EXPECT().Update(updatedVM).Return(updatedVM, nil).Times(1)
				statusUpdate := updatedVM.DeepCopy()
				statusUpdate.Status.SnapshotInProgress = nil
				vmInterface.EXPECT().UpdateStatus(statusUpdate).Return(statusUpdate, nil).Times(1)
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
				vmInterface.EXPECT().UpdateStatus(statusUpdate).Return(statusUpdate, nil).Times(1)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should be status error when VM snapshot deleted while in progress", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.DeletionTimestamp = timeFunc()
				vm := createLockedVM()
				updatedVM := vm.DeepCopy()
				updatedVM.ResourceVersion = "2"
				updatedVM.Finalizers = []string{}
				vmInterface.EXPECT().Update(updatedVM).Return(updatedVM, nil).Times(1)
				statusUpdate := updatedVM.DeepCopy()
				statusUpdate.Status.SnapshotInProgress = nil
				vmInterface.EXPECT().UpdateStatus(statusUpdate).Return(statusUpdate, nil).Times(1)
				vmSource.Add(vm)
				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Finalizers = []string{}
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
				content := createVMSnapshotContent()
				updatedContent := content.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Finalizers = []string{}

				vmSnapshotContentSource.Add(content)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				expectVMSnapshotContentDelete(vmSnapshotClient, updatedContent.Name)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				vmSource.Add(vm)
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

			It("should (partial) lock source", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should (finish) lock source", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSource.Add(vm)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should (partial) lock source when VM updated", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				vmSnapshotSource.Add(vmSnapshot)
				vmInterface.EXPECT().UpdateStatus(vmUpdate).Return(vmUpdate, nil).Times(1)
				addVM(vm)
				controller.processVMSnapshotWorkItem()
			})

			It("should (partial) lock source if running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &t
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should (finish) lock source if running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &t
				vm.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				vmSource.Add(vm)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should (partial) lock source if VMI exists", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Status.SnapshotInProgress = &vmSnapshotName
				vmRevision := createVMRevision(vm)
				crSource.Add(vmRevision)
				vmi := createVMI(vm)
				vmi.Status.VirtualMachineRevisionName = vmRevisionName
				vmiSource.Add(vmi)
				vmSource.Add(vm)
				vmInterface.EXPECT().UpdateStatus(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should (finish) lock source if VMI exists", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vm.DeepCopy()
				vmUpdate.ResourceVersion = "1"
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
				vmRevision := createVMRevision(vm)
				crSource.Add(vmRevision)

				vmi := createVMI(vm)
				vmi.Status.VirtualMachineRevisionName = vmRevisionName
				vmiSource.Add(vmi)
				vmSource.Add(vm)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil).Times(1)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

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

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			It("should lock source if pods using PVCs if VM is running", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createVM()
				vm.Spec.Running = &t
				vmStatusUpdate := vm.DeepCopy()
				vmStatusUpdate.ResourceVersion = "1"
				vmStatusUpdate.Status.SnapshotInProgress = &vmSnapshotName
				vmUpdate := vmStatusUpdate.DeepCopy()
				vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}

				pods := createPodsUsingPVCs(vm)
				podSource.Add(&pods[0])
				vmSnapshotSource.Add(vmSnapshot)
				vmInterface.EXPECT().UpdateStatus(vmStatusUpdate).Return(vmStatusUpdate, nil).Times(1)
				vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil).Times(1)
				vmSource.Add(vm)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "2"
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

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
					newProgressingCondition(corev1.ConditionFalse, "Source not locked"),
					newReadyCondition(corev1.ConditionFalse, "Not ready"),
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{}
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

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: &f,
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
					Indications: []snapshotv1.Indication{},
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

				controller.processVMSnapshotWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVirtualMachineSnapshotContentCreate")
			})

			It("create VirtualMachineSnapshotContent online snapshot", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vm := createLockedVM()
				vm.Spec.Running = &t
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
				expectedContent := createVirtualMachineSnapshotContent(vmSnapshot, vm)
				vm.ObjectMeta.Generation = 2
				vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
				vmSource.Add(vm)
				storageClass := createStorageClass()
				storageClassSource.Add(storageClass)
				volumeSnapshotClass := &createVolumeSnapshotClasses()[0]
				volumeSnapshotClassSource.Add(volumeSnapshotClass)
				pvcs := createPersistentVolumeClaims()
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}
				expectVMSnapshotContentCreate(vmSnapshotClient, expectedContent)
				addVirtualMachineSnapshot(vmSnapshot)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status = &snapshotv1.VirtualMachineSnapshotStatus{
					SourceUID:  &vmUID,
					ReadyToUse: &f,
					Phase:      snapshotv1.InProgress,
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"),
						newReadyCondition(corev1.ConditionFalse, "Not ready"),
					},
				}
				updatedSnapshot.Status.Indications = []snapshotv1.Indication{
					snapshotv1.VMSnapshotOnlineSnapshotIndication,
					snapshotv1.VMSnapshotNoGuestAgentIndication,
				}
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)

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
				updatedSnapshot.Status.Phase = snapshotv1.Succeeded
				updatedSnapshot.Status.Indications = nil
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
				vm := createLockedVM()
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
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

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				volumeSnapshotClassSource.Add(volumeSnapshotClass)
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}

				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshot(vmSnapshot)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should create VolumeSnapshot with multiple VolumeSnapshotClasses", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				volumeSnapshotClasses := createVolumeSnapshotClasses()
				pvcs := createPersistentVolumeClaims()
				vmSnapshot := createVMSnapshotInProgress()
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

				vmSource.Add(vm)
				storageClassSource.Add(storageClass)
				for i := range volumeSnapshotClasses {
					volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
				}
				for i := range pvcs {
					pvcSource.Add(&pvcs[i])
				}

				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClasses[0].Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshot(vmSnapshot)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should create VolumeSnapshot with online snapshot no guest agent", func() {
				vm := createLockedVM()
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
				volumeSnapshotClass := &createVolumeSnapshotClasses()[0]
				pvcs := createPersistentVolumeClaims()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.UID = contentUID
				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)

				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)
				updatedVMSnapshot := vmSnapshot.DeepCopy()
				updatedVMSnapshot.ResourceVersion = "1"
				updatedVMSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotNoGuestAgentIndication)

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

				expectVMSnapshotUpdate(vmSnapshotClient, updatedVMSnapshot)
				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should freeze vm with online snapshot and guest agent", func() {
				storageClass := createStorageClass()
				vmSnapshot := createVMSnapshotInProgress()
				volumeSnapshotClass := &createVolumeSnapshotClasses()[0]
				pvcs := createPersistentVolumeClaims()
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

				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)
				updatedVMSnapshot := vmSnapshot.DeepCopy()
				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotNoGuestAgentIndication)
				updatedVMSnapshot.ResourceVersion = "1"
				updatedVMSnapshot.Status.Indications = append(updatedVMSnapshot.Status.Indications, snapshotv1.VMSnapshotGuestAgentIndication)

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

				vmiInterface.EXPECT().Freeze(vm.Name, 0*time.Second).Return(nil)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedVMSnapshot)
				expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
			})

			It("should update VirtualMachineSnapshotContent", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
				updatedContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &t,
					CreationTime: timeFunc(),
				}

				vmSnapshotSource.Add(vmSnapshot)
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
				vmSnapshot := createVMSnapshotInProgress()
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
				vmSnapshotSource.Add(vmSnapshot)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
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
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)

				volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
				for i := range volumeSnapshots {
					message := "bad error"
					volumeSnapshots[i].Status.ReadyToUse = &rtu
					volumeSnapshots[i].Status.CreationTime = ct
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
			},
				Entry("not ready", false, nil),
				Entry("ready", true, timeFunc()),
			)

			It("should update VirtualMachineSnapshotContent when VolumeSnapshot deleted", func() {
				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   &t,
					CreationTime: timeFunc(),
				}
				updatedContent := vmSnapshotContent.DeepCopy()
				updatedContent.ResourceVersion = "1"
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

				vmSnapshotSource.Add(vmSnapshot)
				expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
				addVirtualMachineSnapshotContent(vmSnapshotContent)
				controller.processVMSnapshotContentWorkItem()
				testutils.ExpectEvent(recorder, "VolumeSnapshotMissing")
			})

			It("should unfreeze VMI if vm was taken with guest agent", func() {
				vmSnapshotContent := createVMSnapshotContent()
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   &t,
				}

				vmSnapshot := createVMSnapshotInProgress()
				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotOnlineSnapshotIndication)
				vmSnapshot.Status.Indications = append(vmSnapshot.Status.Indications, snapshotv1.VMSnapshotGuestAgentIndication)

				updatedSnapshot := vmSnapshot.DeepCopy()
				updatedSnapshot.ResourceVersion = "1"
				updatedSnapshot.Status.SourceUID = &vmUID
				updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				updatedSnapshot.Status.CreationTime = timeFunc()
				updatedSnapshot.Status.ReadyToUse = &t
				updatedSnapshot.Status.Phase = snapshotv1.Succeeded
				updatedSnapshot.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Operation complete"),
				}

				vm := createLockedVM()
				vm.Spec.Running = &t

				vmSource.Add(vm)
				vmSnapshotContentSource.Add(vmSnapshotContent)
				vmiInterface.EXPECT().Unfreeze(vm.Name).Return(nil)
				expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
				addVirtualMachineSnapshot(vmSnapshot)
				controller.processVMSnapshotWorkItem()
			})

			DescribeTable("should delete informer", func(crdName string) {
				crd := &extv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name:              crdName,
						DeletionTimestamp: timeFunc(),
					},
					Spec: extv1.CustomResourceDefinitionSpec{
						Versions: []extv1.CustomResourceDefinitionVersion{
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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(2))
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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(2))
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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(8))

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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(1))
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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(1))
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
					UpdateStatus(gomock.Any()).
					Do(func(objs ...interface{}) {
						vm := objs[0].(*v1.VirtualMachine)

						Expect(len(vm.Status.VolumeSnapshotStatuses)).To(Equal(2))
						Expect(vm.Status.VolumeSnapshotStatuses[0].Enabled).To(BeTrue())
						Expect(vm.Status.VolumeSnapshotStatuses[1].Enabled).To(BeTrue())
						updateCalled = true
					})

				controller.processVMWorkItem()
				controller.processVMSnapshotStatusWorkItem()

				Expect(updateCalled).To(BeTrue())
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
				Entry("for VolumeSnapshot", volumeSnapshotCRD),
				Entry("for VolumeSnapshotClass", volumeSnapshotClassCRD),
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
					Spec: cdiv1.DataVolumeSpec{
						Source: &cdiv1.DataVolumeSource{
							HTTP: &cdiv1.DataVolumeSourceHTTP{
								URL: "http://random-url/images/alpine.iso",
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
			FailureDeadline: noFailureDeadline,
		},
	}
}

func createVirtualMachineSnapshotContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot, vm *v1.VirtualMachine) *snapshotv1.VirtualMachineSnapshotContent {
	var volumeBackups []snapshotv1.VolumeBackup
	vmCpy := vm.DeepCopy()
	vmCpy.ResourceVersion = "1"
	vmCpy.Status = v1.VirtualMachineStatus{}

	for i, pvc := range createPVCsForVM(vmCpy) {
		diskName := fmt.Sprintf("disk%d", i+1)
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
