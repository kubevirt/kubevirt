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
package testing

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
)

func MarkAsReady(vmi *v1.VirtualMachineInstance) {
	vmi.Status.Phase = "Running"
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
}

func MarkAsNonReady(vmi *v1.VirtualMachineInstance) {
	controller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse})
}

func VirtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	runStrategy := v1.RunStrategyAlways
	if !started {
		runStrategy = v1.RunStrategyHalted
	}

	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: "vm-uid"},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
		Status: v1.VirtualMachineStatus{
			Conditions: []v1.VirtualMachineCondition{
				{
					Type:   v1.VirtualMachineReady,
					Status: k8sv1.ConditionFalse,
					Reason: "VMINotExists",
				},
			},
		},
	}
	return vm
}

func NewRunningVirtualMachine(vmiName string, node *k8sv1.Node) *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(vmiName)
	vmi.UID = types.UID(uuid.NewString())
	vmi.Status.Phase = v1.Running
	vmi.Status.NodeName = node.Name
	vmi.Labels = map[string]string{
		v1.NodeNameLabel: node.Name,
	}
	return vmi
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := api.NewMinimalVMI(vmiName)
	vmi.GenerateName = "prettyrandom"
	vmi.Status.Phase = v1.Running
	vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineControllerFinalizer)
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	vm.Finalizers = append(vm.Finalizers, v1.VirtualMachineControllerFinalizer)
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         pointer.P(true),
		BlockOwnerDeletion: pointer.P(true),
	}}
	controller.SetLatestApiVersionAnnotation(vmi)
	controller.SetLatestApiVersionAnnotation(vm)
	return vm, vmi
}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}
