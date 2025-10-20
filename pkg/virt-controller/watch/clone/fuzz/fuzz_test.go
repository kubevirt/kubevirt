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

package fuzz

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	clone "kubevirt.io/api/clone/v1beta1"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/testutils"

	fuzz "github.com/google/gofuzz"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	clonecontroller "kubevirt.io/kubevirt/pkg/virt-controller/watch/clone"
)

// FuzzVMCloneController adds up to 5 VMs and VM Clones
// to the VM store and queue respectively and then
// invokes the VM Clone Controller.
func FuzzVMCloneController(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMs, numberOfVMClones uint8) {
		// Create 2 slices with random vms and vm clones.
		// The fuzzer will add these to the store and
		// queue after creating the controller
		if int(numberOfVMs) == 0 {
			return
		}
		if int(numberOfVMClones) == 0 {
			return
		}
		fdp := fuzz.NewFromGoFuzz(data)
		vms := make([]*virtv1.VirtualMachine, 0)
		vmClones := make([]*clone.VirtualMachineClone, 0)

		maxVMs := int(numberOfVMs) % 5
		maxVMClones := int(numberOfVMClones) % 5

		for _ = range maxVMs {
			vm := &virtv1.VirtualMachine{}
			fdp.Fuzz(vm)
			vms = append(vms, vm)
		}
		if len(vms) == 0 {
			return
		}

		for _ = range maxVMClones {
			vmClone := &clone.VirtualMachineClone{}
			fdp.Fuzz(vmClone)
			vmClones = append(vmClones, vmClone)
		}
		if len(vmClones) == 0 {
			return
		}
		// Done creating the VMs and VM Clones

		// Set up the controller
		ctrl := gomock.NewController(t)
		vmInformer, vmCs := testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		snapshotInformer, snapshotCs := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		restoreInformer, restoreCs := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineRestore{})
		cloneInformer, cloneCs := testutils.NewFakeInformerFor(&clone.VirtualMachineClone{})
		snapshotContentInformer, snapthotContentCs := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		pvcInformer, pvcCs := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		defer vmCs.Shutdown()
		defer snapshotCs.Shutdown()
		defer restoreCs.Shutdown()
		defer cloneCs.Shutdown()
		defer snapthotContentCs.Shutdown()
		defer pvcCs.Shutdown()

		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		controller, err := clonecontroller.NewVmCloneController(
			virtClient,
			cloneInformer,
			snapshotInformer,
			restoreInformer,
			vmInformer,
			snapshotContentInformer,
			pvcInformer,
			recorder)
		if err != nil {
			panic(err)
		}
		mockQueue := testutils.NewMockWorkQueue(clonecontroller.GetVmCloneQueue(controller))
		clonecontroller.ShutdownCtrlQueue(controller)
		clonecontroller.SetQueue(controller, mockQueue)

		client := kubevirtfake.NewSimpleClientset()
		// Done setting up the controller

		// Add vms to vm store
		for _, randomVM := range vms {
			clonecontroller.AddToVmStore(controller, randomVM)
		}

		// Add vm clones to the queue
		for _, vmClone := range vmClones {
			var addToQueue bool
			var create bool
			fdp.Fuzz(&addToQueue)
			fdp.Fuzz(&create)
			if addToQueue {
				clonecontroller.AddTovmCloneIndexer(controller, vmClone)
				key, err := kvcontroller.KeyFunc(vmClone)
				if err != nil {
					continue
				}
				mockQueue.Add(key)
			}
			if create {
				client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault).Create(context.Background(), vmClone, metav1.CreateOptions{})
			}
		}
		if mockQueue.Len() == 0 {
			return
		}
		controller.Execute()
	})
}
