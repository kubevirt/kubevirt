/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package snapshot

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	kubevirtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	kvtesting "kubevirt.io/client-go/testing"
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

	createRestoreInProgressVM := func() *kubevirtv1.VirtualMachine {
		vm := createVirtualMachine(testNamespace, vmName)
		vm.Status.RestoreInProgress = &vmRestoreName
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
		var controller *VMRestoreController
		var recorder *record.FakeRecorder
		var fakeVolumeSnapshotProvider *MockVolumeSnapshotProvider

		var kubevirtClient *kubevirtfake.Clientset
		var k8sClient *k8sfake.Clientset
		var cdiClient *cdifake.Clientset

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)

			vmRestoreInformer, _ := testutils.NewFakeInformerWithIndexersFor(&snapshotv1.VirtualMachineRestore{}, virtcontroller.GetVirtualMachineRestoreInformerIndexers())
			vmSnapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
			vmSnapshotContentInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
			vmiInformer, _ := testutils.NewFakeInformerFor(&kubevirtv1.VirtualMachineInstance{})
			vmInformer, _ := testutils.NewFakeInformerFor(&kubevirtv1.VirtualMachine{})
			dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			pvcInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			storageClassInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			crInformer, _ := testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, virtcontroller.GetControllerRevisionInformerIndexers())

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
			controller.vmRestoreQueue = testutils.NewMockWorkQueue(controller.vmRestoreQueue)

			// Set up mock client
			kubevirtClient = kubevirtfake.NewSimpleClientset()

			virtClient.EXPECT().VirtualMachine(testNamespace).
				Return(kubevirtClient.KubevirtV1().VirtualMachines(testNamespace)).AnyTimes()
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
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

			k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			currentTime = timeFunc
		})

		addVirtualMachineRestore := func(r *snapshotv1.VirtualMachineRestore) {
			Expect(controller.VMRestoreInformer.GetStore().Add(r)).To(Succeed())
			key, err := virtcontroller.KeyFunc(r)
			Expect(err).ToNot(HaveOccurred())
			controller.vmRestoreQueue.Add(key)
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
				Expect(controller.VMSnapshotInformer.GetStore().Add(s)).To(Succeed())
				Expect(controller.VMSnapshotContentInformer.GetStore().Add(sc)).To(Succeed())
				Expect(controller.StorageClassInformer.GetStore().Add(storageClass)).To(Succeed())
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
				Expect(controller.VMSnapshotInformer.GetStore().Delete(createSnapshot())).To(Succeed())
				if vmSnapshot != nil {
					Expect(controller.VMSnapshotInformer.GetStore().Add(vmSnapshot)).To(Succeed())
				}
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
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

				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				Expect(controller.VMInformer.GetStore().Add(newVM)).To(Succeed())

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
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
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
				rc.Finalizers = finalizers
				rc.OwnerReferences = ownerRefs
				updateCalls := expectVMRestoreUpdate(kubevirtClient, rc)

				rc2 := rc.DeepCopy()
				rc2.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore, or the operation will fail after 5m0s"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore, or the operation will fail after 5m0s"),
					},
				}
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc2)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())

				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
				Expect(*updateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should update restore with finalizer and owner and update vm that restore is in progress", func() {
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
				rc.Finalizers = finalizers
				rc.OwnerReferences = ownerRefs

				rc2 := rc.DeepCopy()
				rc2.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				addInitialVolumeRestores(rc2)

				vm := createModifiedVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				vmStatusUpdate := vm.DeepCopy()
				vmStatusUpdate.Status.RestoreInProgress = &vmRestoreName
				addVirtualMachineRestore(r)

				updateCalls := expectVMRestoreUpdate(kubevirtClient, rc)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc2)
				updateVMStatusCalls := expectVMUpdateStatus(kubevirtClient, vmStatusUpdate)
				controller.processVMRestoreWorkItem()
				Expect(*updateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*updateVMStatusCalls).To(Equal(1))
			})

			It("should update restore status with condition and VolumeRestores", func() {
				r := createRestoreWithOwner()
				vm := createRestoreInProgressVM()
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
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
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
				vm := createRestoreInProgressVM()
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

				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVirtualMachineRestore(r)

				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should create restore PVCs", func() {
				r := createRestoreWithOwner()
				vm := createRestoreInProgressVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVolumeRestores(r)
				pvcSize := resource.MustParse("2Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
				fakeVolumeSnapshotProvider.Add(vs)
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

				vm := createRestoreInProgressVM()
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
								corev1.ResourceStorage: resource.MustParse("2Gi"),
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
				Expect(controller.VMSnapshotContentInformer.GetStore().Delete(sc)).To(Succeed())
				// create ct vmSnapshotContent with the relevant info
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				Expect(controller.VMSnapshotContentInformer.GetStore().Add(sc)).To(Succeed())
				pvcSize := resource.MustParse("2Gi")
				vs1 := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
				vs2 := createVolumeSnapshot(r.Status.Restores[1].VolumeSnapshotName, pvcSize)
				fakeVolumeSnapshotProvider.Add(vs1)
				fakeVolumeSnapshotProvider.Add(vs2)

				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVirtualMachineRestore(r)

				calls := expectPVCCreates(k8sClient, r, pvcSize)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(2))
			})

			It("should create restore PVC with volume snapshot size if bigger then PVC size", func() {
				r := createRestoreWithOwner()
				vm := createRestoreInProgressVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVolumeRestores(r)
				q := resource.MustParse("3Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, q)
				fakeVolumeSnapshotProvider.Add(vs)
				calls := expectPVCCreates(k8sClient, r, q)
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
			})

			It("should create restore PVC with pvc size if restore size is smaller", func() {
				r := createRestoreWithOwner()
				vm := createRestoreInProgressVM()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVolumeRestores(r)
				q := resource.MustParse("1Gi")
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, q)
				fakeVolumeSnapshotProvider.Add(vs)
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

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVirtualMachineRestore(r)
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimPending
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
				}
				controller.processVMRestoreWorkItem()
			})

			It("should keep existing VM runstrategy as before the restore", func() {
				// Update snapshoted VM to have running instead of runstrategy
				// to show the resulted VM has the expected run stratgey as before
				// the restore and doesnt have both running and runstrategy
				sc.Spec.Source.VirtualMachine.Spec.RunStrategy = nil
				sc.Spec.Source.VirtualMachine.Spec.Running = pointer.P(true)
				Expect(controller.VMSnapshotContentInformer.GetStore().Update(sc)).To(Succeed())

				r := createRestoreWithOwner()
				addVolumeRestores(r)
				r.Status.DeletedDataVolumes = getDeletedDataVolumes(createModifiedVM())
				for i := range r.Status.Restores {
					r.Status.Restores[i].DataVolumeName = &r.Status.Restores[i].PersistentVolumeClaimName
				}
				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
				}

				vm := createSnapshotVM()
				vm.Spec.RunStrategy = pointer.P(kubevirtv1.RunStrategyManual)
				vm.Status.RestoreInProgress = &vmRestoreName
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				uvm := vm.DeepCopy()
				uvm.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
				uvm.Spec.DataVolumeTemplates[0].Name = "restore-uid-disk1"
				uvm.Spec.Template.Spec.Volumes[0].DataVolume.Name = "restore-uid-disk1"
				setLegacyFirmwareUUID(uvm)
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimBound
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
				}
				addVirtualMachineRestore(r)

				// Expect vm runstrategy to not change
				updateVMCalls := expectVMUpdate(kubevirtClient, uvm)
				pvcUpdateCalls := expectPVCUpdates(k8sClient, ur)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)

				controller.processVMRestoreWorkItem()
				Expect(*pvcUpdateCalls).To(Equal(1))
				Expect(*updateStatusCalls).To(Equal(1))
				Expect(*updateVMCalls).To(Equal(1))
			})

			It("restored pvc is owned by target VM when volume ownership policy is set to VM", func() {
				r := createRestoreWithOwner()
				r.ResourceVersion = "1"
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}

				// We could omit this, as it is the default policy
				r.Spec.VolumeOwnershipPolicy = pointer.P(snapshotv1.VolumeOwnershipPolicyVm)

				addInitialVolumeRestores(r)
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, resource.MustParse("2Gi"))
				fakeVolumeSnapshotProvider.Add(vs)

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				ref := []metav1.OwnerReference{
					*metav1.NewControllerRef(vm, schema.GroupVersionKind{Group: "kubevirt.io", Version: "v1", Kind: "VirtualMachine"}),
				}

				createdPVCCalls := expectPVCCreatesWithOwnership(k8sClient, ref)

				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*createdPVCCalls).To(Equal(1)) // Restored PVC is owned by the VM
			})

			It("restored pvc has not owner when volume ownership policy is set to none", func() {
				r := createRestoreWithOwner()
				r.ResourceVersion = "1"
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}
				r.Spec.VolumeOwnershipPolicy = pointer.P(snapshotv1.VolumeOwnershipPolicyNone)
				addInitialVolumeRestores(r)
				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, resource.MustParse("2Gi"))
				fakeVolumeSnapshotProvider.Add(vs)

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				createdPVCCalls := expectPVCCreatesWithOwnership(k8sClient, nil)

				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
				Expect(*createdPVCCalls).To(Equal(1)) // Restored PVC is owned by nothing
			})

			It("volume is set to be overwritten when volume restore policy is InPlace", func() {
				r := createRestoreWithOwner()
				r.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				}
				r.Spec.VolumeRestorePolicy = pointer.P(snapshotv1.VolumeRestorePolicyInPlace)
				r.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: "alpine-dv",
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
					},
				}

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				dv := &cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vm.Spec.DataVolumeTemplates[0].Name,
						Namespace: vm.Namespace,
					},
					Status: cdiv1.DataVolumeStatus{
						Phase: cdiv1.Succeeded,
					},
				}

				pvc := getRestorePVCs(r)[0]
				pvc.OwnerReferences = []metav1.OwnerReference{
					*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"}),
				}

				Expect(controller.DataVolumeInformer.GetStore().Add(dv)).To(Succeed())
				Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				}
				ur.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: "alpine-dv", // Name is identical to the original PVC
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
						DataVolumeName:            pointer.P("alpine-dv"),
					},
				}

				addVirtualMachineRestore(r)

				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)
				patchDataVolumeCalls := expectDataVolumePatch(cdiClient, dv, dv)
				deletePVCCalls := expectPVCDeletion(k8sClient, r)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))    // Restore is updated to mark DV owns the PVC
				Expect(*patchDataVolumeCalls).To(Equal(0)) // DV not set to prepopulated yet
				Expect(*deletePVCCalls).To(Equal(0))       // PVC hasn't been deleted yet
			})

			It("source volume/DV gets deleted when volume restore policy is InPlace", func() {
				r := createRestoreWithOwner()
				r.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				}
				r.Spec.VolumeRestorePolicy = pointer.P(snapshotv1.VolumeRestorePolicyInPlace)
				r.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: "alpine-dv", // Name is identical to the original PVC
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
						DataVolumeName:            pointer.P(diskName),
					},
				}

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				// We expect the one and only PVC on that VM to get deleted, so we create it first and link it to its DV
				dv := &cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vm.Spec.DataVolumeTemplates[0].Name,
						Namespace: vm.Namespace,
						Annotations: map[string]string{
							"test": "test",
						},
					},
					Status: cdiv1.DataVolumeStatus{
						Phase: cdiv1.Succeeded,
					},
				}

				Expect(controller.DataVolumeInformer.GetStore().Add(dv)).To(Succeed())

				pvc := getRestorePVCs(r)[0]
				pvc.OwnerReferences = []metav1.OwnerReference{
					*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"}),
				}

				Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: "alpine-dv", // Name is identical to the original PVC
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
						DataVolumeName:            &vm.Spec.DataVolumeTemplates[0].Name, // This property was set to add the DV as owner of the PVC
					},
				}

				addVirtualMachineRestore(r)

				deletePVCCalls := expectPVCDeletion(k8sClient, r)

				updatedDV := dv.DeepCopy()
				updatedDV.Annotations[RestoreNameAnnotation] = r.Name
				updatedDV.Annotations[cdiv1.AnnPrePopulated] = dv.Name
				patchDVCalls := expectDataVolumePatch(cdiClient, dv, updatedDV)

				controller.processVMRestoreWorkItem()

				Expect(*patchDVCalls).To(Equal(1))   // DV associated with PVC edited to be prePopulated
				Expect(*deletePVCCalls).To(Equal(1)) // Original PVC got deleted
			})

			It("new volume gets created when volume restore policy is InPlace", func() {
				r := createRestoreWithOwner()
				r.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				}
				r.Spec.VolumeRestorePolicy = pointer.P(snapshotv1.VolumeRestorePolicyInPlace)
				r.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: diskName,
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
					},
				}

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Updating target spec"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for target update"),
				}

				vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, resource.MustParse("2Gi"))
				fakeVolumeSnapshotProvider.Add(vs)
				addVirtualMachineRestore(r)

				createPVCCalls := expectPVCCreates(k8sClient, ur, resource.MustParse("2Gi"))

				controller.processVMRestoreWorkItem()
				Expect(*createPVCCalls).To(Equal(1)) // New PVC got created in place of the original
			})

			It("can change the name of a destination volume", func() {
				r := createRestoreWithOwner()
				r.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"),
					newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"),
				}
				r.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName:  diskName,
						RestoreName: "renamed-destination",
					},
				}

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				}
				ur.Status.Restores = []snapshotv1.VolumeRestore{
					{
						VolumeName:                diskName,
						PersistentVolumeClaimName: "renamed-destination",
						VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk1",
					},
				}

				addVirtualMachineRestore(r)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)
				controller.processVMRestoreWorkItem()
				Expect(*updateStatusCalls).To(Equal(1))
			})

			It("should override pvcs and volumes", func() {
				r := createRestoreWithOwner()
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
						newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
					},
				}

				r.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName: diskName,
						Labels: map[string]string{
							"newlabel": "value",
						},
						Annotations: map[string]string{
							"newannotation": "annotation",
						},
					},
					{
						VolumeName: "disk2",
						Labels: map[string]string{
							"newlabel": "value",
						},
						Annotations: map[string]string{
							"newannotation": "annotation",
						},
					},
				}

				addVolumeRestores(r)
				r.Status.Restores = append(r.Status.Restores, snapshotv1.VolumeRestore{
					VolumeName:                "disk2",
					PersistentVolumeClaimName: "restore-uid-disk2",
					VolumeSnapshotName:        "vmsnapshot-snapshot-uid-volume-disk2",
				})

				vm := createRestoreInProgressVM()
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
								corev1.ResourceStorage: resource.MustParse("2Gi"),
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
				Expect(controller.VMSnapshotContentInformer.GetStore().Delete(sc)).To(Succeed())
				// create ct vmSnapshotContent with the relevant info
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				Expect(controller.VMSnapshotContentInformer.GetStore().Add(sc)).To(Succeed())
				pvcSize := resource.MustParse("2Gi")
				vs1 := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
				vs2 := createVolumeSnapshot(r.Status.Restores[1].VolumeSnapshotName, pvcSize)
				fakeVolumeSnapshotProvider.Add(vs1)
				fakeVolumeSnapshotProvider.Add(vs2)

				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVirtualMachineRestore(r)

				expectedLabels := map[string]string{"newlabel": "value"}
				expectedAnnotations := map[string]string{"newannotation": "annotation"}

				calls := expectPVCCreatesWithMetadata(k8sClient, r, expectedLabels, expectedAnnotations)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(2))
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

				vm := createRestoreInProgressVM()
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				addVirtualMachineRestore(r)
				pvcUpdateCalls := expectPVCUpdates(k8sClient, ur)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)
				for _, pvc := range getRestorePVCs(r) {
					pvc.Status.Phase = corev1.ClaimBound
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
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
				Expect(controller.VMSnapshotContentInformer.GetStore().Delete(sc)).To(Succeed())
				// create ct vmSnapshotContent with the relevant info
				sc = createVirtualMachineSnapshotContent(s, vm, pvcs)
				sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}
				Expect(controller.VMSnapshotContentInformer.GetStore().Add(sc)).To(Succeed())

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
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
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
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
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
				addVirtualMachineRestore(r)

				vm := createRestoreInProgressVM()
				vm.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}

				for _, n := range r.Status.DeletedDataVolumes {
					dv := &cdiv1.DataVolume{
						ObjectMeta: metav1.ObjectMeta{
							Name:      n,
							Namespace: testNamespace,
						},
					}
					Expect(controller.DataVolumeInformer.GetStore().Add(dv)).To(Succeed())
				}
				for _, pvc := range getRestorePVCs(r) {
					pvc.Annotations["cdi.kubevirt.io/storage.populatedFor"] = pvc.Name
					pvc.Status.Phase = corev1.ClaimBound
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
				}

				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				updatedVM := vm.DeepCopy()
				updatedVM.Status.RestoreInProgress = nil

				ur := r.DeepCopy()
				ur.ResourceVersion = "1"
				ur.Status.Complete = pointer.P(true)
				ur.Status.RestoreTime = timeFunc()
				ur.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "Operation complete"),
					newReadyCondition(corev1.ConditionTrue, "Operation complete"),
				}

				updateVMStatusCalls := expectVMUpdateStatus(kubevirtClient, updatedVM)
				dvDeleteCalls := expectDataVolumeDeletes(cdiClient, r.Status.DeletedDataVolumes)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, ur)

				controller.processVMRestoreWorkItem()

				l, err := cdiClient.CdiV1beta1().DataVolumes("").List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(l.Items).To(BeEmpty())
				Expect(*updateVMStatusCalls).To(Equal(1))
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
							lastRestoreAnnotation: "restore-uid",
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
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
				}

				addVirtualMachineRestore(r)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
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
				vm.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}

				addVirtualMachineRestore(r)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

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
				vm.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}

				addVirtualMachineRestore(r)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				updatedVMRestore := r.DeepCopy()
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

				vm := createRestoreInProgressVM()

				addVirtualMachineRestore(r)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				vmUpdated := vm.DeepCopy()
				vmUpdated.Status.RestoreInProgress = nil

				updatedVMRestore := r.DeepCopy()
				updatedVMRestore.Status.Conditions = []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"),
					newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"),
				}
				updatedVMRestore.ResourceVersion = "1"

				updateVMStatusCalls := expectVMUpdateStatus(kubevirtClient, vmUpdated)
				updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, updatedVMRestore)
				controller.processVMRestoreWorkItem()
				Expect(*updateVMStatusCalls).To(Equal(1))
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

				addVirtualMachineRestore(r)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

				updatedVMRestore := r.DeepCopy()
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
					vm = createRestoreInProgressVM()
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
						Expect(controller.DataVolumeInformer.GetStore().Add(dv)).To(Succeed())
						pvc.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"})}
						r.Status.DeletedDataVolumes = getDeletedDataVolumes(vm)
					} else {
						calls = expectDataVolumeCreate(cdiClient, "restore-uid-disk1")
					}
					Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
					return calls
				}

				DescribeTable("should", func(dvExists bool, phase cdiv1.DataVolumePhase, expecteUpdateVM bool) {
					dvCreateCalls := addRestoreVolumes(dvExists, phase)

					addVirtualMachineRestore(r)
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					updateVMCalls := pointer.P(0)
					if expecteUpdateVM {
						updatedVM := vm.DeepCopy()
						updatedVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
						updatedVM.Spec.DataVolumeTemplates[0].Name = "restore-uid-disk1"
						updatedVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = "restore-uid-disk1"
						setLegacyFirmwareUUID(updatedVM)
						updateVMCalls = expectVMUpdate(kubevirtClient, updatedVM)
					}
					res, err := targetVM.Reconcile()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res).To(BeTrue())
					if dvCreateCalls != nil {
						Expect(*dvCreateCalls).To(Equal(1))
					}
					if expecteUpdateVM {
						Expect(*updateVMCalls).To(Equal(1))
					}
				},
					Entry("update VM spec when dv phase succeeded", true, cdiv1.Succeeded, true),
					Entry("update VM spec when dv phase WFFC", true, cdiv1.WaitForFirstConsumer, true),
					Entry("wait for dvs when dv phase pending", true, cdiv1.Pending, false),
					Entry("create dvs when dv doesnt exists", false, cdiv1.PhaseUnset, false),
				)

				It("should set firmware UUID when it lacks from snapshot", func() {
					addRestoreVolumes(true, cdiv1.Succeeded)
					addVirtualMachineRestore(r)

					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					updatedVM := createSnapshotVM()
					updatedVM.Status.RestoreInProgress = &vmRestoreName
					updatedVM.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
					updatedVM.Spec.DataVolumeTemplates[0].Name = "restore-uid-disk1"
					updatedVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = "restore-uid-disk1"
					setLegacyFirmwareUUID(updatedVM)
					expectVMUpdate(kubevirtClient, updatedVM)
					res, err := targetVM.Reconcile()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res).To(BeTrue())
				})

				It("should not override existing firmware UUID", func() {
					addRestoreVolumes(true, cdiv1.Succeeded)
					addVirtualMachineRestore(r)

					uid := types.UID("existing-uid")
					existingFirmware := &kubevirtv1.Firmware{UUID: uid}

					vm.Spec.Template.Spec.Domain.Firmware = existingFirmware
					sc.Spec.Source.VirtualMachine.Spec.Template.Spec.Domain.Firmware = existingFirmware

					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					updatedVM := createSnapshotVM()
					updatedVM.Status.RestoreInProgress = &vmRestoreName
					updatedVM.Annotations = map[string]string{"restore.kubevirt.io/lastRestoreUID": "restore-uid"}
					updatedVM.Spec.DataVolumeTemplates[0].Name = "restore-uid-disk1"
					updatedVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = "restore-uid-disk1"
					updatedVM.Spec.Template.Spec.Domain.Firmware = &kubevirtv1.Firmware{UUID: uid}
					expectVMUpdate(kubevirtClient, updatedVM)
					res, err := targetVM.Reconcile()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res).To(BeTrue())
				})
			})

			Context("target VM is different than source VM", func() {

				It("should be able to restore to a new VM", func() {
					// Update snapshoted VM to have runstrategy Always
					// and see the resulted new VM has Halted
					sc.Spec.Source.VirtualMachine.Spec.RunStrategy = pointer.P(kubevirtv1.RunStrategyAlways)
					Expect(controller.VMSnapshotContentInformer.GetStore().Update(sc)).To(Succeed())
					By("Creating new VM")
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.UID = newVMUID

					By("Creating VM restore")
					vmRestore := createRestoreWithOwner()
					vmRestore.Spec.Target.Name = newVM.Name
					addVolumeRestores(vmRestore)
					addVirtualMachineRestore(vmRestore)

					By("Creating PVC")
					for _, pvc := range getRestorePVCs(vmRestore) {
						pvc.Status.Phase = corev1.ClaimBound
						Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
					}

					Expect(vmRestore.Status.Restores).To(HaveLen(1))
					vmRestore.Status.Restores[0].DataVolumeName = pointer.P(restoreDVName(vmRestore, vmRestore.Status.Restores[0].VolumeName, ""))
					pvcUpdateCalls := expectPVCUpdates(k8sClient, vmRestore)

					By("Making sure right VM update occurs")
					newVM.Spec.RunStrategy = pointer.P(kubevirtv1.RunStrategyHalted)
					newVM.Spec.DataVolumeTemplates[0].Name = *vmRestore.Status.Restores[0].DataVolumeName
					newVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = *vmRestore.Status.Restores[0].DataVolumeName
					newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
					createVMCalls := expectVMCreate(kubevirtClient, newVM, newVMUID)

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
					Expect(*createVMCalls).To(Equal(1))
					Expect(*pvcUpdateCalls).To(Equal(1))
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("should own the vmrestore after creation of new target", func() {
					By("Creating new VM")
					newVM := createVirtualMachine(testNamespace, newVMName)
					newVM.Status.RestoreInProgress = &vmRestoreName
					newVM.UID = newVMUID
					newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
					Expect(controller.VMInformer.GetStore().Add(newVM)).To(Succeed())

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

				Context("target VM does not exist, should create new VM", func() {

					const (
						newMacAddress = "00:00:5e:00:53:01"
					)

					var (
						r *snapshotv1.VirtualMachineRestore

						changeMacAddressPatch string
					)

					BeforeEach(func() {
						r = createRestore()
						r.Spec.Target.Name = "nonexistent-vm"
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
							Expect(controller.DataVolumeInformer.GetStore().Add(dv)).To(Succeed())
							pvc.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(dv, schema.GroupVersionKind{Group: "cdi.kubevirt.io", Version: "v1beta1", Kind: "DataVolume"})}
							pvc.Annotations["cdi.kubevirt.io/storage.populatedFor"] = pvc.Name
							pvc.Status.Phase = corev1.ClaimBound
							Expect(controller.PVCInformer.GetStore().Add(&pvc)).To(Succeed())
						}

						changeMacAddressPatch = fmt.Sprintf(`{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": "%s"}`, newMacAddress)
					})

					It("with changed MAC address", func() {
						r.Spec.Patches = []string{changeMacAddressPatch}

						newVM := createVirtualMachine(testNamespace, r.Spec.Target.Name)
						newVM.UID = newVMUID
						newVM.Spec.DataVolumeTemplates[0].Name = restoreDVName(r, r.Status.Restores[0].VolumeName, "")
						newVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = restoreDVName(r, r.Status.Restores[0].VolumeName, "")
						newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
						newVM.Spec.Template.Spec.Domain.Devices.Interfaces[0].MacAddress = newMacAddress
						createVMCalls := expectVMCreate(kubevirtClient, newVM, newVMUID)

						targetVM, err := controller.getTarget(r)
						Expect(err).ShouldNot(HaveOccurred())
						success, err := targetVM.Reconcile()
						Expect(success).To(BeTrue())
						Expect(err).ShouldNot(HaveOccurred())
						Expect(*createVMCalls).To(Equal(1))
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
							newProgressingCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore"),
						},
					}
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())
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
							newProgressingCondition(corev1.ConditionFalse, "Automatically stopping restore target for restore operation"),
							newReadyCondition(corev1.ConditionFalse, "Automatically stopping restore target for restore operation"),
						},
					}
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())
					stopCalled := expectVMStop(kubevirtClient)
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)

					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "RestoreTargetNotReady")
					Expect(*updateStatusCalls).To(Equal(1))
					Expect(*stopCalled).To(Equal(1))
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
							newProgressingCondition(corev1.ConditionFalse, "Restore target failed to be ready within 5m0s. Please power off the target VM before attempting restore"),
							newReadyCondition(corev1.ConditionFalse, "Restore target failed to be ready within 5m0s. Please power off the target VM before attempting restore"),
							newFailureCondition(corev1.ConditionTrue, "Restore target failed to be ready within 5m0s. Please power off the target VM before attempting restore"),
						},
					}
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "Operation failed")
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("default - GracePeriodAndFail - should not fail if grace period passed but target became ready", func() {
					r := createRestoreWithOwner()
					r.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore, or the operation will fail after 5m0s"),
							newReadyCondition(corev1.ConditionFalse, "Waiting for target VM to be powered off. Please stop the restore target to proceed with restore, or the operation will fail after 5m0s"),
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

					vm := createRestoreInProgressVM()
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

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
							newProgressingCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
							newReadyCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
							newFailureCondition(corev1.ConditionTrue, "Restore target VMI must be powered off before restore operation"),
						},
					}
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
					Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())
					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, rc)
					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					testutils.ExpectEvent(recorder, "Operation failed")
					Expect(*updateStatusCalls).To(Equal(1))
				})

				It("should not change from failure status even if VM becomes ready", func() {
					r := createRestoreWithOwner()
					r.Spec.TargetReadinessPolicy = pointer.P(snapshotv1.VirtualMachineRestoreFailImmediate)
					// Set up restore with failure condition already present
					r.Status = &snapshotv1.VirtualMachineRestoreStatus{
						Complete: pointer.P(false),
						Conditions: []snapshotv1.Condition{
							newProgressingCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
							newReadyCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
							newFailureCondition(corev1.ConditionTrue, "Restore target VMI must be powered off before restore operation"),
						},
					}
					vm := createModifiedVM()
					// Add VM without VMI (VM is ready), but restore already has failure condition
					Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())

					updateStatusCalls := expectVMRestoreUpdateStatus(kubevirtClient, r)

					addVirtualMachineRestore(r)
					controller.processVMRestoreWorkItem()
					// Verify no status update occurred since restore is already in terminal failure state
					Expect(*updateStatusCalls).To(Equal(0))
				})
			})

			It("Failed restore should be terminating state", func() {
				r := createRestoreWithOwner()
				r.OwnerReferences = nil
				r.Status = &snapshotv1.VirtualMachineRestoreStatus{
					Complete: pointer.P(false),
					Conditions: []snapshotv1.Condition{
						newProgressingCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
						newReadyCondition(corev1.ConditionFalse, "Restore target VMI must be powered off before restore operation"),
						newFailureCondition(corev1.ConditionTrue, "Restore target VMI must be powered off before restore operation"),
					},
				}
				vm := createModifiedVM()
				vmi := createVMI(vm)
				Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
				Expect(controller.VMIInformer.GetStore().Add(vmi)).To(Succeed())
				addVirtualMachineRestore(r)
				controller.processVMRestoreWorkItem()
			})
		})

		It("should create restore PVCs with populated dataSourceRef and dataSource", func() {
			// Mock the restore environment from scratch, so we use source PVCs with dataSourceRef
			vm := createRestoreInProgressVM()
			pvcs := createPVCsForVMWithDataSourceRef(vm)
			s := createSnapshot()
			sc := createVirtualMachineSnapshotContent(s, vm, pvcs)
			storageClass := createStorageClass()
			s.Status.VirtualMachineSnapshotContentName = &sc.Name
			sc.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
				CreationTime: timeFunc(),
				ReadyToUse:   pointer.P(true),
			}
			Expect(controller.VMInformer.GetStore().Add(vm)).To(Succeed())
			Expect(controller.VMSnapshotInformer.GetStore().Add(s)).To(Succeed())
			Expect(controller.VMSnapshotContentInformer.GetStore().Add(sc)).To(Succeed())
			Expect(controller.StorageClassInformer.GetStore().Add(storageClass)).To(Succeed())

			// Actual test
			r := createRestoreWithOwner()
			r.Status = &snapshotv1.VirtualMachineRestoreStatus{
				Complete: pointer.P(false),
				Conditions: []snapshotv1.Condition{
					newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"),
					newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"),
				},
			}
			addVolumeRestores(r)
			pvcSize := resource.MustParse("2Gi")
			vs := createVolumeSnapshot(r.Status.Restores[0].VolumeSnapshotName, pvcSize)
			fakeVolumeSnapshotProvider.Add(vs)
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

			getInstancetypeOriginalCR := func() *appsv1.ControllerRevision { return instancetypeOriginalCR }
			getPreferenceOriginalCR := func() *appsv1.ControllerRevision { return preferenceOriginalCR }
			nilInstancetypeMatcher := func() *kubevirtv1.InstancetypeMatcher { return nil }
			nilPrefrenceMatcher := func() *kubevirtv1.PreferenceMatcher { return nil }

			BeforeEach(func() {
				originalVM = createRestoreInProgressVM()
				originalVM.Spec.DataVolumeTemplates = []kubevirtv1.DataVolumeTemplateSpec{}
				restore = createRestoreWithOwner()

				vmSnapshot = createSnapshot()
				vmSnapshotContent = createVirtualMachineSnapshotContent(vmSnapshot, originalVM, nil)
				vmSnapshotContent.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{
					CreationTime: timeFunc(),
					ReadyToUse:   pointer.P(true),
				}

				vmSnapshot.Status.VirtualMachineSnapshotContentName = &vmSnapshotContent.Name
				Expect(controller.VMSnapshotInformer.GetStore().Add(vmSnapshot)).To(Succeed())

				instancetypeObj = createInstancetype()
				var err error
				instancetypeOriginalCR, err = revision.CreateControllerRevision(originalVM, instancetypeObj)
				Expect(err).ToNot(HaveOccurred())
				Expect(controller.CRInformer.GetStore().Add(instancetypeOriginalCR)).To(Succeed())

				instancetypeSnapshotCR = createInstancetypeVirtualMachineSnapshotCR(originalVM, vmSnapshot, instancetypeObj)
				Expect(controller.CRInformer.GetStore().Add(instancetypeSnapshotCR)).To(Succeed())

				preferenceObj = createPreference()
				preferenceOriginalCR, err = revision.CreateControllerRevision(originalVM, preferenceObj)
				Expect(err).ToNot(HaveOccurred())
				Expect(controller.CRInformer.GetStore().Add(preferenceOriginalCR)).To(Succeed())
				preferenceSnapshotCR = createInstancetypeVirtualMachineSnapshotCR(originalVM, vmSnapshot, preferenceObj)
				Expect(controller.CRInformer.GetStore().Add(preferenceSnapshotCR)).To(Succeed())
			})

			DescribeTable("with an existing VirtualMachine",
				func(getVMInstancetypeMatcher, getSnapshotInstancetypeMatcher func() *kubevirtv1.InstancetypeMatcher, getVMPreferenceMatcher, getSnapshotPreferenceMatcher func() *kubevirtv1.PreferenceMatcher, getExpectedCR func() *appsv1.ControllerRevision) {
					originalVM.Spec.Instancetype = getVMInstancetypeMatcher()
					originalVM.Spec.Preference = getVMPreferenceMatcher()
					Expect(controller.VMInformer.GetStore().Add(originalVM)).To(Succeed())

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					Expect(controller.VMSnapshotContentInformer.GetStore().Add(vmSnapshotContent)).To(Succeed())

					updatedVM := originalVM.DeepCopy()
					updatedVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
					setLegacyFirmwareUUID(updatedVM)
					updateVMCalls := expectVMUpdate(kubevirtClient, updatedVM)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
					Expect(*updateVMCalls).To(Equal(1))
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
					Expect(controller.VMInformer.GetStore().Add(originalVM)).To(Succeed())

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					Expect(controller.VMSnapshotContentInformer.GetStore().Add(vmSnapshotContent)).To(Succeed())

					// Ensure we restore into a new VM
					newVM := originalVM.DeepCopy()
					newVM.Name = "newvm"
					newVM.UID = ""
					newVM.ResourceVersion = ""
					newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
					restore.Spec.Target.Name = newVM.Name

					originalCR := getExpectedCR()
					expectedCreatedCR := originalCR.DeepCopy()
					expectedCreatedCR.Name = strings.Replace(expectedCreatedCR.Name, originalVM.Name, newVM.Name, 1)
					expectedCreatedCR.OwnerReferences = nil
					crCreates := expectControllerRevisionCreate(k8sClient, expectedCreatedCR)

					// We need to be able to find the created CR from the controller so add it to the source
					expectedCreatedCR.Namespace = testNamespace
					Expect(controller.CRInformer.GetStore().Add(expectedCreatedCR)).To(Succeed())

					expectedUpdatedCR := expectedCreatedCR.DeepCopy()
					newVM.UID = newVMUID
					expectedUpdatedCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(newVM, kubevirtv1.VirtualMachineGroupVersionKind)}
					crUpdates := expectControllerRevisionUpdate(k8sClient, expectedUpdatedCR)

					if newVM.Spec.Instancetype != nil {
						newVM.Spec.Instancetype.RevisionName = expectedCreatedCR.Name
					}
					if newVM.Spec.Preference != nil {
						newVM.Spec.Preference.RevisionName = expectedCreatedCR.Name
					}
					createVMCalls := expectVMCreate(kubevirtClient, newVM, newVMUID)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
					Expect(*crCreates).To(Equal(1))
					Expect(*crUpdates).To(Equal(1))
					Expect(*createVMCalls).To(Equal(1))
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
					Expect(controller.VMInformer.GetStore().Add(originalVM)).To(Succeed())

					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = getSnapshotInstancetypeMatcher()
					vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Preference = getSnapshotPreferenceMatcher()
					Expect(controller.VMSnapshotContentInformer.GetStore().Add(vmSnapshotContent)).To(Succeed())

					// Ensure we restore into a new VM
					newVM := originalVM.DeepCopy()
					newVM.UID = newVMUID
					newVM.ResourceVersion = ""
					newVM.Name = "newvm"
					newVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
					restore.Spec.Target.Name = newVM.Name

					originalCR := getExpectedCR()
					expectedCreatedCR := originalCR.DeepCopy()
					expectedCreatedCR.Name = strings.Replace(expectedCreatedCR.Name, originalVM.Name, newVM.Name, 1)
					expectedCreatedCR.OwnerReferences = nil
					crCreates := expectControllerRevisionCreate(k8sClient, expectedCreatedCR)

					// We need to be able to find the created CR from the controller so add it to the source
					expectedCreatedCR.Namespace = testNamespace
					Expect(controller.CRInformer.GetStore().Add(expectedCreatedCR)).To(Succeed())

					expectedUpdatedCR := expectedCreatedCR.DeepCopy()
					expectedUpdatedCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(newVM, kubevirtv1.VirtualMachineGroupVersionKind)}
					crUpdates := expectControllerRevisionUpdate(k8sClient, expectedUpdatedCR)

					if newVM.Spec.Instancetype != nil {
						newVM.Spec.Instancetype.RevisionName = expectedCreatedCR.Name
					}
					if newVM.Spec.Preference != nil {
						newVM.Spec.Preference.RevisionName = expectedCreatedCR.Name
					}
					createVMFailureCalls := expectVMCreateFailure(kubevirtClient, vmCreationFailureMessage)
					failCalls := expectUpdateVMRestoreFailure(restore, "1", vmCreationFailureMessage)

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*createVMFailureCalls).To(Equal(1))

					// We have already created the ControllerRevision but that shouldn't stop the reconcile from progressing
					alreadyExistsCalls := expectCreateControllerRevisionAlreadyExists(k8sClient, expectedCreatedCR)
					createVMCalls := expectVMCreate(kubevirtClient, newVM, newVMUID)
					calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "2")

					addVirtualMachineRestore(restore)
					controller.processVMRestoreWorkItem()
					Expect(*calls).To(Equal(1))
					Expect(*failCalls).To(Equal(1))
					Expect(*crCreates).To(Equal(1))
					Expect(*alreadyExistsCalls).To(Equal(1))
					Expect(*crUpdates).To(Equal(1))
					Expect(*createVMCalls).To(Equal(1))
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
				Expect(controller.CRInformer.GetStore().Update(instancetypeOriginalCR)).To(Succeed())

				originalVM.Spec.Instancetype = &kubevirtv1.InstancetypeMatcher{
					Name:         instancetypeObj.Name,
					Kind:         instancetypeapi.SingularResourceName,
					RevisionName: instancetypeOriginalCR.Name,
				}
				Expect(controller.VMInformer.GetStore().Add(originalVM)).To(Succeed())

				vmSnapshotContent.Spec.Source.VirtualMachine.Spec.Instancetype = &kubevirtv1.InstancetypeMatcher{
					Name:         instancetypeObj.Name,
					Kind:         instancetypeapi.SingularResourceName,
					RevisionName: instancetypeSnapshotCR.Name,
				}
				Expect(controller.VMSnapshotContentInformer.GetStore().Add(vmSnapshotContent)).To(Succeed())

				// We expect the original CR to be deleted and recreated with the correct data
				crDeletes := expectControllerRevisionDelete(k8sClient, instancetypeOriginalCR.Name)
				crCreates := expectControllerRevisionCreate(k8sClient, instancetypeOriginalCRCopy)

				updatedVM := originalVM.DeepCopy()
				updatedVM.Annotations = map[string]string{lastRestoreAnnotation: "restore-uid"}
				setLegacyFirmwareUUID(updatedVM)
				updateVMCalls := expectVMUpdate(kubevirtClient, updatedVM)
				calls := expectUpdateVMRestoreUpdatingTargetSpec(restore, "1")

				addVirtualMachineRestore(restore)
				controller.processVMRestoreWorkItem()
				Expect(*calls).To(Equal(1))
				Expect(*crCreates).To(Equal(1))
				Expect(*crDeletes).To(Equal(1))
				Expect(*updateVMCalls).To(Equal(1))
			})
		})
	})
})

func expectVMCreateFailure(client *kubevirtfake.Clientset, failureMsg string) *int {
	calls := 0
	client.Fake.PrependReactor("create", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		_, ok = create.GetObject().(*kubevirtv1.VirtualMachine)
		Expect(ok).To(BeTrue())

		calls++

		return true, nil, fmt.Errorf(failureMsg)
	})
	return &calls
}

func expectVMCreate(client *kubevirtfake.Clientset, vm *kubevirtv1.VirtualMachine, newVMUID types.UID) *int {
	calls := 0
	client.Fake.PrependReactor("create", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj, ok := create.GetObject().(*kubevirtv1.VirtualMachine)
		Expect(ok).To(BeTrue())

		calls++
		createObj.UID = newVMUID
		Expect(createObj.ObjectMeta).To(Equal(vm.ObjectMeta))
		Expect(createObj.Spec).To(Equal(vm.Spec))

		return true, createObj, nil
	})
	return &calls
}

func expectVMUpdate(client *kubevirtfake.Clientset, vm *kubevirtv1.VirtualMachine) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())

		updateObj := update.GetObject().(*kubevirtv1.VirtualMachine)

		calls++
		Expect(vm.ObjectMeta).To(Equal(updateObj.ObjectMeta))
		Expect(vm.Spec).To(Equal(updateObj.Spec))

		return true, update.GetObject(), nil
	})
	return &calls
}

func expectVMUpdateStatus(client *kubevirtfake.Clientset, vm *kubevirtv1.VirtualMachine) *int {
	calls := 0
	client.Fake.PrependReactor("update", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		if update.GetSubresource() != "status" {
			return false, nil, nil
		}

		updateObj := update.GetObject().(*kubevirtv1.VirtualMachine)
		Expect(vm.Status).To(Equal(updateObj.Status))

		calls++

		return true, update.GetObject(), nil
	})
	return &calls
}

func expectVMStop(client *kubevirtfake.Clientset) *int {
	stopCalled := 0
	client.Fake.PrependReactor("put", "virtualmachines/stop", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		_, ok := action.(kvtesting.PutAction[*kubevirtv1.StopOptions])
		Expect(ok).To(BeTrue())

		stopCalled++
		return true, nil, nil
	})
	return &stopCalled
}

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
				Expect(createObj.Annotations[lastRestoreAnnotation]).To(Equal(getRestoreAnnotationValue(vmRestore)))
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

func expectPVCCreatesWithMetadata(client *k8sfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore, labels, annotations map[string]string) *int {
	calls := 0
	client.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*corev1.PersistentVolumeClaim)
		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vr.PersistentVolumeClaimName == createObj.Name {
				for key, expectedVal := range labels {
					actualVal, ok := createObj.Labels[key]
					Expect(ok).To(BeTrue(), fmt.Sprintf("couldn't find key %s with value %s, got %v", key, expectedVal, createObj.Labels))
					Expect(actualVal).To(Equal(expectedVal))
				}

				for key, expectedVal := range annotations {
					actualVal, ok := createObj.Annotations[key]
					Expect(ok).To(BeTrue(), fmt.Sprintf("couldn't find key %s with value %s", key, expectedVal))
					Expect(actualVal).To(Equal(expectedVal))
				}
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

func expectPVCCreatesWithOwnership(client *k8sfake.Clientset, ref []metav1.OwnerReference) *int {
	calls := 0
	client.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		createObj := create.GetObject().(*corev1.PersistentVolumeClaim)
		found := reflect.DeepEqual(createObj.OwnerReferences, ref)
		Expect(found).To(BeTrue())

		calls++
		return true, create.GetObject(), nil
	})
	return &calls
}

func expectPVCDeletion(client *k8sfake.Clientset, vmRestore *snapshotv1.VirtualMachineRestore) *int {
	calls := 0
	client.Fake.PrependReactor("delete", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		del, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())

		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vr.PersistentVolumeClaimName == del.GetName() {
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

		Expect(updateObj.Status).To(Equal(vmRestore.Status))
		return true, update.GetObject(), nil
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

func expectDataVolumePatch(client *cdifake.Clientset, orig, newObj *cdiv1.DataVolume) *int {
	calls := 0
	client.Fake.PrependReactor("patch", "datavolumes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		patch, ok := action.(testing.PatchAction)
		Expect(ok).To(BeTrue())

		patched := &cdiv1.DataVolume{}
		if err := applyPatch(patch.GetPatch(), orig, patched); err != nil {
			Expect(err).NotTo(HaveOccurred())
			return false, nil, err
		}

		Expect(patched).To(Equal(newObj))

		calls++

		return true, patched, nil
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
