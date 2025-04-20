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
package memorydump

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	testPVCName    = "testPVC"
	targetFileName = "memory.dump"
	vmName         = "testVM"
)

var now = metav1.Now()

var _ = Describe("MemoryDump", func() {
	var k8sClient *k8sfake.Clientset
	var virtClient *kubecli.MockKubevirtClient
	var virtFakeClient *fake.Clientset
	var pvcStore cache.Store

	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtFakeClient = fake.NewSimpleClientset()

		pvcInformer, _ := testutils.NewFakeInformerFor(&k8score.PersistentVolumeClaim{})
		pvcStore = pvcInformer.GetStore()

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
		).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
	})

	Context("UpdateRequest", func() {
		It("should update memory dump phase to InProgress when memory dump in vm volumes", func() {
			vm, vmi := createVirtualMachineWithMemoryDump(v1.MemoryDumpAssociating)

			// when the memory dump volume is in the vm volume list
			// we should change status to in progress
			updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
				Phase:     v1.MemoryDumpInProgress,
			}

			UpdateRequest(vm, vmi)
			Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
		})

		It("should update status to unmounting when memory dump timestamp updated", func() {
			vm, vmi := createVirtualMachineWithMemoryDump(v1.MemoryDumpInProgress)
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  testPVCName,
					Phase: v1.MemoryDumpVolumeCompleted,
					MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
						StartTimestamp: pointer.P(now),
						EndTimestamp:   pointer.P(now),
						ClaimName:      testPVCName,
						TargetFileName: targetFileName,
					},
				},
			}

			updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
				ClaimName:      testPVCName,
				Phase:          v1.MemoryDumpUnmounting,
				EndTimestamp:   pointer.P(now),
				StartTimestamp: pointer.P(now),
				FileName:       &vmi.Status.VolumeStatus[0].MemoryDumpVolume.TargetFileName,
			}

			UpdateRequest(vm, vmi)

			Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
		})

		It("should update status to failed when memory dump failed", func() {
			vm, vmi := createVirtualMachineWithMemoryDump(v1.MemoryDumpInProgress)
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:    testPVCName,
					Phase:   v1.MemoryDumpVolumeFailed,
					Message: "Memory dump failed",
					MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
						ClaimName:    testPVCName,
						EndTimestamp: pointer.P(now),
					},
				},
			}

			updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
				ClaimName:    testPVCName,
				Phase:        v1.MemoryDumpFailed,
				Message:      vmi.Status.VolumeStatus[0].Message,
				EndTimestamp: pointer.P(now),
			}

			UpdateRequest(vm, vmi)

			Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
		})

		It("should update memory dump to completed once memory dump volume unmounted", func() {
			vm, vmi := createVirtualMachineWithMemoryDump(v1.MemoryDumpUnmounting)

			// In case the volume is not in vmi volume status
			// we should update status to completed
			updatedMemoryDump := &v1.VirtualMachineMemoryDumpRequest{
				ClaimName:    testPVCName,
				Phase:        v1.MemoryDumpCompleted,
				EndTimestamp: pointer.P(now),
				FileName:     pointer.P(targetFileName),
			}

			UpdateRequest(vm, vmi)

			Expect(vm.Status.MemoryDumpRequest).To(Equal(updatedMemoryDump))
		})

		It("should dissociate memory dump request when status is Dissociating and not in vm volumes", func() {
			// No need to add vmi - can do this action even if vm not running
			vm, _ := createVirtualMachineWithMemoryDump(v1.MemoryDumpDissociating)

			UpdateRequest(vm, nil)

			Expect(vm.Status.MemoryDumpRequest).To(BeNil())
		})
	})

	DescribeTable("should remove memory dump volume from vmi volumes and update pvc annotation", func(phase v1.MemoryDumpPhase, expectedAnnotation string) {
		vm, vmi := createVirtualMachineWithMemoryDump(phase)

		vmi.Status.VolumeStatus = []v1.VolumeStatus{
			{
				Name: testPVCName,
				MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
					ClaimName: testPVCName,
				},
			},
		}

		vmi, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		pvc := &k8score.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testPVCName,
				Namespace: vm.Namespace,
			},
		}
		pvc, err = k8sClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvcStore.Add(pvc)).To(Succeed())

		HandleRequest(virtClient, vm, vmi, pvcStore)

		vmi, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.Spec.Volumes).To(BeEmpty())

		pvc, err = k8sClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), pvc.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvc.Annotations[v1.PVCMemoryDumpAnnotation]).To(Equal(expectedAnnotation))
	},
		Entry("when phase is Unmounting", v1.MemoryDumpUnmounting, targetFileName),
		Entry("when phase is Failed", v1.MemoryDumpFailed, "Memory dump failed"),
	)
})

func ApplyVMIMemoryDumpVol(spec *v1.VirtualMachineInstanceSpec) {
	newVolume := v1.Volume{
		Name: testPVCName,
		VolumeSource: v1.VolumeSource{
			MemoryDump: &v1.MemoryDumpVolumeSource{
				PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
						ClaimName: testPVCName,
					},
					Hotpluggable: true,
				},
			},
		},
	}

	spec.Volumes = append(spec.Volumes, newVolume)
}

func createVirtualMachineWithMemoryDump(memoryDumpPhase v1.MemoryDumpPhase) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := api.NewMinimalVMI(vmName)
	vmi.Status.Phase = v1.Running
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: "vm-uid"},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: pointer.P(v1.RunStrategyAlways),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
		ClaimName: testPVCName,
		Phase:     memoryDumpPhase,
	}
	switch memoryDumpPhase {
	case v1.MemoryDumpAssociating:
		ApplyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
	case v1.MemoryDumpInProgress, v1.MemoryDumpFailed:
		ApplyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
		vmi.Spec = vm.Spec.Template.Spec
	case v1.MemoryDumpUnmounting:
		ApplyVMIMemoryDumpVol(&vm.Spec.Template.Spec)
		vmi.Spec = vm.Spec.Template.Spec
		vm.Status.MemoryDumpRequest.EndTimestamp = pointer.P(now)
		vm.Status.MemoryDumpRequest.FileName = pointer.P(targetFileName)
	}
	return vm, vmi
}
