package watch

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8ssnapshotv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	k8ssnapshotfake "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned/fake"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Snapshot controlleer", func() {
	const (
		testNamespace = "default"
		vnSnapshotUID = "uid"
	)

	var (
		timeStamp               = metav1.Now()
		vmName                  = "testvm"
		vmSnapshotName          = "test-snapshot"
		retain                  = vmsnapshotv1alpha1.VirtualMachineSnapshotContentRetain
		storageClassName        = "rook-ceph-block"
		volumeSnapshotClassName = "csi-rbdplugin-snapclass"

		t = true
		f = false
	)

	timeFunc := func() *metav1.Time {
		return &timeStamp
	}

	createVMSnapshot := func() *vmsnapshotv1alpha1.VirtualMachineSnapshot {
		return &vmsnapshotv1alpha1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmSnapshotName,
				Namespace: testNamespace,
				UID:       vnSnapshotUID,
			},
			Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotSpec{
				Source: vmsnapshotv1alpha1.VirtualMachineSnapshotSource{
					VirtualMachineName: &vmName,
				},
			},
		}
	}

	createVMSnapshotSuccess := func() *vmsnapshotv1alpha1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &vmsnapshotv1alpha1.VirtualMachineSnapshotStatus{
			ReadyToUse:   &t,
			CreationTime: timeFunc(),
		}

		return vms
	}

	createVMSnapshotInProgress := func() *vmsnapshotv1alpha1.VirtualMachineSnapshot {
		vms := createVMSnapshot()
		vms.Finalizers = []string{"snapshot.kubevirt.io/vmsnapshot-protection"}
		vms.Status = &vmsnapshotv1alpha1.VirtualMachineSnapshotStatus{
			ReadyToUse: &f,
		}

		return vms
	}

	createVM := func() *v1.VirtualMachine {
		return &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmName,
				Namespace: testNamespace,
				UID:       "uid",
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
				DataVolumeTemplates: []cdiv1alpha1.DataVolume{
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

	createLockedVM := func() *v1.VirtualMachine {
		vm := createVM()
		vm.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
		vm.Status.SnapshotInProgress = &[]string{vmSnapshotName}[0]
		return vm
	}

	createPersistentVolumeClaims := func() []corev1.PersistentVolumeClaim {
		vm := createLockedVM()
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

	createVMSnapshotContent := func() *vmsnapshotv1alpha1.VirtualMachineSnapshotContent {
		vmSnapshot := createVMSnapshotInProgress()
		var volumeBackups []vmsnapshotv1alpha1.VolumeBackup
		vm := createLockedVM()
		vm.ResourceVersion = "1"
		vm.Status = v1.VirtualMachineStatus{}

		for i, pvc := range createPersistentVolumeClaims() {
			diskName := fmt.Sprintf("disk%d", i+1)
			volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-disk-%s", vmSnapshot.UID, diskName)
			vb := vmsnapshotv1alpha1.VolumeBackup{
				DiskName:              diskName,
				PersistentVolumeClaim: pvc,
				VolumeSnapshotName:    &volumeSnapshotName,
			}
			volumeBackups = append(volumeBackups, vb)
		}

		return &vmsnapshotv1alpha1.VirtualMachineSnapshotContent{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "vmsnapshot-content-" + vnSnapshotUID,
				Namespace:  testNamespace,
				Finalizers: []string{"snapshot.kubevirt.io/vmsnapshotcontent-protection"},
			},
			Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotContentSpec{
				VirtualMachineSnapshotName: &vmSnapshotName,
				Source: vmsnapshotv1alpha1.SourceSpec{
					VirtualMachine: vm,
				},
				VolumeBackups: volumeBackups,
			},
			Status: &vmsnapshotv1alpha1.VirtualMachineSnapshotContentStatus{
				ReadyToUse: &f,
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

	createVolumeSnapshots := func(content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) []k8ssnapshotv1beta1.VolumeSnapshot {
		var volumeSnapshots []k8ssnapshotv1beta1.VolumeSnapshot
		for _, vb := range content.Spec.VolumeBackups {
			if vb.VolumeSnapshotName == nil {
				continue
			}
			vs := k8ssnapshotv1beta1.VolumeSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *vb.VolumeSnapshotName,
					Namespace: content.Namespace,
				},
				Spec: k8ssnapshotv1beta1.VolumeSnapshotSpec{
					Source: k8ssnapshotv1beta1.VolumeSnapshotSource{
						PersistentVolumeClaimName: &vb.PersistentVolumeClaim.Name,
					},
				},
				Status: &k8ssnapshotv1beta1.VolumeSnapshotStatus{
					ReadyToUse: &f,
				},
			}
			volumeSnapshots = append(volumeSnapshots, vs)
		}
		return volumeSnapshots
	}

	createVolumeSnapshotClasses := func() []k8ssnapshotv1beta1.VolumeSnapshotClass {
		return []k8ssnapshotv1beta1.VolumeSnapshotClass{
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
		var volumeSnapshotInformer cache.SharedIndexInformer
		var volumeSnapshotSource *framework.FakeControllerSource
		var volumeSnapshotClassInformer cache.SharedIndexInformer
		var volumeSnapshotClassSource *framework.FakeControllerSource
		var storageClassInformer cache.SharedIndexInformer
		var storageClassSource *framework.FakeControllerSource
		var pvcInformer cache.SharedIndexInformer
		var pvcSource *framework.FakeControllerSource
		var stop chan struct{}
		var controller *SnapshotController
		var recorder *record.FakeRecorder
		var mockVMSnapshotQueue *testutils.MockWorkQueue
		var mockVMSnapshotContentQueue *testutils.MockWorkQueue

		var vmSnapshotClient *kubevirtfake.Clientset
		var k8sSnapshotClient *k8ssnapshotfake.Clientset
		var k8sClient *k8sfake.Clientset

		syncCaches := func(stop chan struct{}) {
			go vmSnapshotInformer.Run(stop)
			go vmSnapshotContentInformer.Run(stop)
			go vmInformer.Run(stop)
			go volumeSnapshotInformer.Run(stop)
			go volumeSnapshotClassInformer.Run(stop)
			go storageClassInformer.Run(stop)
			go pvcInformer.Run(stop)
			Expect(cache.WaitForCacheSync(
				stop,
				vmSnapshotInformer.HasSynced,
				vmSnapshotContentInformer.HasSynced,
				vmInformer.HasSynced,
				volumeSnapshotInformer.HasSynced,
				volumeSnapshotClassInformer.HasSynced,
				storageClassInformer.HasSynced,
			)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

			vmSnapshotInformer, vmSnapshotSource = testutils.NewFakeInformerWithIndexersFor(&vmsnapshotv1alpha1.VirtualMachineSnapshot{}, cache.Indexers{
				"vm": func(obj interface{}) ([]string, error) {
					vms := obj.(*vmsnapshotv1alpha1.VirtualMachineSnapshot)
					if vms.Spec.Source.VirtualMachineName != nil {
						return []string{*vms.Spec.Source.VirtualMachineName}, nil
					}
					return nil, nil
				},
			})
			vmSnapshotContentInformer, vmSnapshotContentSource = testutils.NewFakeInformerWithIndexersFor(&vmsnapshotv1alpha1.VirtualMachineSnapshotContent{}, cache.Indexers{
				"volumeSnapshot": func(obj interface{}) ([]string, error) {
					vmsc := obj.(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent)
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
			volumeSnapshotInformer, volumeSnapshotSource = testutils.NewFakeInformerFor(&k8ssnapshotv1beta1.VolumeSnapshot{})
			volumeSnapshotClassInformer, volumeSnapshotClassSource = testutils.NewFakeInformerFor(&k8ssnapshotv1beta1.VolumeSnapshotClass{})
			storageClassInformer, storageClassSource = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			pvcInformer, pvcSource = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})

			recorder = record.NewFakeRecorder(100)

			controller = NewSnapshotController(
				virtClient,
				vmSnapshotInformer,
				vmSnapshotContentInformer,
				vmInformer,
				volumeSnapshotInformer,
				volumeSnapshotClassInformer,
				storageClassInformer,
				pvcInformer,
				recorder,
				60*time.Second,
			)

			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockVMSnapshotQueue = testutils.NewMockWorkQueue(controller.vmSnapshotQueue)
			controller.vmSnapshotQueue = mockVMSnapshotQueue

			mockVMSnapshotContentQueue = testutils.NewMockWorkQueue(controller.vmSnapshotContentQueue)
			controller.vmSnapshotContentQueue = mockVMSnapshotContentQueue

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

		AfterEach(func() {
			close(stop)
		})

		addVirtualMachineSnapshot := func(s *vmsnapshotv1alpha1.VirtualMachineSnapshot) {
			syncCaches(stop)
			mockVMSnapshotQueue.ExpectAdds(1)
			vmSnapshotSource.Add(s)
			mockVMSnapshotQueue.Wait()
		}

		addVirtualMachineSnapshotContent := func(s *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) {
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

		addVolumeSnapshot := func(s *k8ssnapshotv1beta1.VolumeSnapshot) {
			syncCaches(stop)
			mockVMSnapshotContentQueue.ExpectAdds(1)
			volumeSnapshotSource.Add(s)
			mockVMSnapshotContentQueue.Wait()
		}

		It("should initialize VirtualMachineSnapshot status", func() {
			vmSnapshot := createVMSnapshot()
			updatedSnapshot := vmSnapshot.DeepCopy()
			updatedSnapshot.ResourceVersion = "1"
			updatedSnapshot.Status = &vmsnapshotv1alpha1.VirtualMachineSnapshotStatus{
				ReadyToUse: &f,
			}
			expectVMSnapshotUpdate(vmSnapshotClient, updatedSnapshot)
			addVirtualMachineSnapshot(vmSnapshot)
			controller.processVMSnapshotWorkItem()
		})

		It("should unlock source VirtualMachine", func() {
			vmSnapshot := createVMSnapshotSuccess()
			vm := createLockedVM()
			updatedVM := createVM()
			updatedVM.Finalizers = []string{}
			updatedVM.ResourceVersion = "1"
			vmSource.Add(vm)
			vmInterface.EXPECT().Update(updatedVM).Return(updatedVM, nil)
			addVirtualMachineSnapshot(vmSnapshot)
			controller.processVMSnapshotWorkItem()
		})

		It("should be status error when VM snapshot deleted while in progress", func() {
			vmSnapshot := createVMSnapshotInProgress()
			vmSnapshot.DeletionTimestamp = timeFunc()
			updatedSnapshot := vmSnapshot.DeepCopy()
			updatedSnapshot.ResourceVersion = "1"
			updatedSnapshot.Status.Error = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
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
			vmUpdate := vm.DeepCopy()
			vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
			vmUpdate.ResourceVersion = "1"
			vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

			vmSource.Add(vm)
			vmInterface.EXPECT().Update(vmUpdate).Return(vmUpdate, nil)
			addVirtualMachineSnapshot(vmSnapshot)
			controller.processVMSnapshotWorkItem()
		})

		It("should lock source when VM updated", func() {
			vmSnapshot := createVMSnapshotInProgress()
			vm := createVM()
			vmUpdate := vm.DeepCopy()
			vmUpdate.Finalizers = []string{"snapshot.kubevirt.io/snapshot-source-protection"}
			vmUpdate.ResourceVersion = "1"
			vmUpdate.Status.SnapshotInProgress = &vmSnapshotName

			vmSnapshotSource.Add(vmSnapshot)
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
			vmSnapshotContent.Status.CreationTime = timeFunc()
			vmSnapshotContent.Status.ReadyToUse = &t

			vmSnapshot := createVMSnapshotInProgress()
			updatedSnapshot := vmSnapshot.DeepCopy()
			updatedSnapshot.ResourceVersion = "1"
			updatedSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
			updatedSnapshot.Status.CreationTime = timeFunc()
			updatedSnapshot.Status.ReadyToUse = &t

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
			vmSnapshotContent.UID = "uid"

			storageClassSource.Add(storageClass)
			volumeSnapshotClassSource.Add(volumeSnapshotClass)
			for i := range pvcs {
				pvcSource.Add(&pvcs[i])
			}

			expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClass.Name, vmSnapshotContent)
			addVirtualMachineSnapshotContent(vmSnapshotContent)
			controller.processVMSnapshotContentWorkItem()
			testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
		})

		It("should create VolumeSnapshot with multiple VolumeSnapshotClasses", func() {
			storageClass := createStorageClass()
			volumeSnapshotClasses := createVolumeSnapshotClasses()
			pvcs := createPersistentVolumeClaims()
			vmSnapshotContent := createVMSnapshotContent()
			vmSnapshotContent.UID = "uid"

			storageClassSource.Add(storageClass)
			for i := range volumeSnapshotClasses {
				volumeSnapshotClassSource.Add(&volumeSnapshotClasses[i])
			}
			for i := range pvcs {
				pvcSource.Add(&pvcs[i])
			}

			expectVolumeSnapshotCreates(k8sSnapshotClient, volumeSnapshotClasses[0].Name, vmSnapshotContent)
			addVirtualMachineSnapshotContent(vmSnapshotContent)
			controller.processVMSnapshotContentWorkItem()
			testutils.ExpectEvent(recorder, "SuccessfulVolumeSnapshotCreate")
		})

		It("should update VirtualMachineSnapshotContent", func() {
			vmSnapshotContent := createVMSnapshotContent()
			updatedContent := vmSnapshotContent.DeepCopy()
			updatedContent.ResourceVersion = "1"
			updatedContent.Status.ReadyToUse = &t
			updatedContent.Status.CreationTime = timeFunc()

			vmSnapshotContentSource.Add(vmSnapshotContent)
			expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)

			volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
			for i := range volumeSnapshots {
				volumeSnapshots[i].Status.ReadyToUse = &t
				volumeSnapshots[i].Status.CreationTime = timeFunc()
				addVolumeSnapshot(&volumeSnapshots[i])
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
			updatedContent.Status.ReadyToUse = &t
			updatedContent.Status.CreationTime = timeFunc()

			expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
			addVirtualMachineSnapshotContent(vmSnapshotContent)
			controller.processVMSnapshotContentWorkItem()
		})

		It("should update VirtualMachineSnapshotContent on error", func() {
			message := "bad error"
			vmSnapshotContent := createVMSnapshotContent()
			updatedContent := vmSnapshotContent.DeepCopy()
			updatedContent.ResourceVersion = "1"
			updatedContent.Status.ReadyToUse = &f
			updatedContent.Status.CreationTime = nil
			updatedContent.Status.Error = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
				Message: &message,
				Time:    timeFunc(),
			}

			vmSnapshotContentSource.Add(vmSnapshotContent)
			expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)

			volumeSnapshots := createVolumeSnapshots(vmSnapshotContent)
			for i := range volumeSnapshots {
				volumeSnapshots[i].Status.ReadyToUse = &f
				volumeSnapshots[i].Status.CreationTime = nil
				volumeSnapshots[i].Status.Error = &k8ssnapshotv1beta1.VolumeSnapshotError{
					Message: &message,
					Time:    timeFunc(),
				}
				addVolumeSnapshot(&volumeSnapshots[i])
			}

			controller.processVMSnapshotContentWorkItem()
		})

		It("should update VirtualMachineSnapshotContent when VolumeSnapshot deleted", func() {
			vmSnapshotContent := createVMSnapshotContent()
			vmSnapshotContent.Status.ReadyToUse = &t
			vmSnapshotContent.Status.CreationTime = timeFunc()
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
			updatedContent.Status.Error = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
				Message: &errorMessage,
				Time:    timeFunc(),
			}

			vmSnapshotContentSource.Add(vmSnapshotContent)
			expectVMSnapshotContentUpdate(vmSnapshotClient, updatedContent)
			addVirtualMachineSnapshotContent(vmSnapshotContent)
			controller.processVMSnapshotContentWorkItem()
			testutils.ExpectEvent(recorder, "VolumeSnapshotMissing")
		})
	})
})

func expectVMSnapshotUpdate(client *kubevirtfake.Clientset, vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) {
	client.Fake.PrependReactor("update", "virtualmachinesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*vmsnapshotv1alpha1.VirtualMachineSnapshot)
		Expect(updateObj).To(Equal(vmSnapshot))

		return true, update.GetObject(), nil
	})
}

func expectVMSnapshotContentCreate(client *kubevirtfake.Clientset, content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) {
	client.Fake.PrependReactor("create", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent)
		Expect(createObj).To(Equal(content))

		return true, create.GetObject(), nil
	})
}

func expectVMSnapshotContentUpdate(client *kubevirtfake.Clientset, content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) {
	client.Fake.PrependReactor("update", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent)
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
	content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent,
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

		createObj := create.GetObject().(*k8ssnapshotv1beta1.VolumeSnapshot)
		pvc, ok := volumeSnapshots[createObj.Name]
		Expect(ok).To(BeTrue())
		Expect(pvc).Should(Equal(*createObj.Spec.Source.PersistentVolumeClaimName))
		delete(volumeSnapshots, createObj.Name)

		Expect(*createObj.Spec.VolumeSnapshotClassName).Should(Equal(voluemSnapshotClass))
		Expect(createObj.OwnerReferences[0].Name).Should(Equal(content.Name))
		Expect(createObj.OwnerReferences[0].UID).Should(Equal(content.UID))

		return true, nil, nil
	})
}
