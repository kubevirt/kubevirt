/*
Copyright 2017 The Kubernetes Authors.
Copyright 2017 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"reflect"
	"sync"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var (
	productionLabel         = map[string]string{"type": "production"}
	testLabel               = map[string]string{"type": "testing"}
	productionLabelSelector = labels.Set{"type": "production"}.AsSelector()
	controllerUID           = "123"
)

func newControllerRef(controller metav1.Object) *metav1.OwnerReference {
	var controllerKind = v1beta1.SchemeGroupVersion.WithKind("Fake")
	blockOwnerDeletion := true
	isController := true
	return &metav1.OwnerReference{
		APIVersion:         controllerKind.GroupVersion().String(),
		Kind:               controllerKind.Kind,
		Name:               "Fake",
		UID:                controller.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

func newVirtualMachine(virtualmachineName string, label map[string]string, owner metav1.Object) *virtv1.VirtualMachineInstance {
	vmi := &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtualmachineName,
			Labels:    label,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: virtv1.VirtualMachineInstanceSpec{},
	}
	if owner != nil {
		vmi.OwnerReferences = []metav1.OwnerReference{*newControllerRef(owner)}
	}
	return vmi
}

func TestClaimVirtualMachineInstance(t *testing.T) {
	controllerKind := schema.GroupVersionKind{}
	type test struct {
		name            string
		manager         *VirtualMachineControllerRefManager
		virtualmachines []*virtv1.VirtualMachineInstance
		filters         []func(*virtv1.VirtualMachineInstance) bool
		claimed         []*virtv1.VirtualMachineInstance
		released        []*virtv1.VirtualMachineInstance
		expectError     bool
	}
	var tests = []test{
		{
			name: "Claim virtualmachines with correct label",
			manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
				&v1.ReplicationController{},
				productionLabelSelector,
				controllerKind,
				func() error { return nil }),
			virtualmachines: []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, nil), newVirtualMachine("virtualmachine2", testLabel, nil)},
			claimed:         []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, nil)},
		},
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			return test{
				name: "Controller marked for deletion can not claim virtualmachines",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				virtualmachines: []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, nil), newVirtualMachine("virtualmachine2", productionLabel, nil)},
				claimed:         nil,
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			return test{
				name: "Controller marked for deletion can not claim new virtualmachines",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				virtualmachines: []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller), newVirtualMachine("virtualmachine2", productionLabel, nil)},
				claimed:         []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller)},
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller2 := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			controller2.UID = types.UID("AAAAA")
			return test{
				name: "Controller can not claim virtualmachines owned by another controller",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				virtualmachines: []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller), newVirtualMachine("virtualmachine2", productionLabel, &controller2)},
				claimed:         []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller)},
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			return test{
				name: "Controller releases claimed virtualmachines when selector doesn't match",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				virtualmachines: []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller), newVirtualMachine("virtualmachine2", testLabel, &controller)},
				claimed:         []*virtv1.VirtualMachineInstance{newVirtualMachine("virtualmachine1", productionLabel, &controller)},
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			virtualmachineToDelete1 := newVirtualMachine("virtualmachine1", productionLabel, &controller)
			virtualmachineToDelete2 := newVirtualMachine("virtualmachine2", productionLabel, nil)
			now := metav1.Now()
			virtualmachineToDelete1.DeletionTimestamp = &now
			virtualmachineToDelete2.DeletionTimestamp = &now

			return test{
				name: "Controller does not claim orphaned virtualmachines marked for deletion",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				virtualmachines: []*virtv1.VirtualMachineInstance{virtualmachineToDelete1, virtualmachineToDelete2},
				claimed:         []*virtv1.VirtualMachineInstance{virtualmachineToDelete1},
			}
		}(),
	}
	for _, test := range tests {
		claimed, err := test.manager.ClaimVirtualMachineInstances(test.virtualmachines)
		if test.expectError && err == nil {
			t.Errorf("Test case `%s`, expected error but got nil", test.name)
		} else if !reflect.DeepEqual(test.claimed, claimed) {
			t.Errorf("Test case `%s`, claimed wrong virtualmachines. Expected %v, got %v", test.name, virtualmachineToStringSlice(test.claimed), virtualmachineToStringSlice(claimed))
		}

	}
}

func newDataVolume(name string, owner metav1.Object) *cdiv1.DataVolume {
	dataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
	}
	if owner != nil {
		dataVolume.OwnerReferences = []metav1.OwnerReference{*newControllerRef(owner)}
	}

	return dataVolume
}

func TestClaimDataVolume(t *testing.T) {
	controllerKind := schema.GroupVersionKind{}
	type test struct {
		name        string
		manager     *VirtualMachineControllerRefManager
		datavolumes []*cdiv1.DataVolume
		filters     []func(*cdiv1.DataVolume) bool
		claimed     []*cdiv1.DataVolume
		released    []*cdiv1.DataVolume
		expectError bool
	}
	var tests = []test{
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			return test{
				name: "Controller marked for deletion can not claim datavolumes",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				datavolumes: []*cdiv1.DataVolume{newDataVolume("datavolume1", nil), newDataVolume("datavolume2", nil)},
				claimed:     nil,
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			now := metav1.Now()
			controller.DeletionTimestamp = &now
			return test{
				name: "Controller marked for deletion can not claim new datavolumes",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				datavolumes: []*cdiv1.DataVolume{newDataVolume("datavolume1", &controller), newDataVolume("datavolume2", nil)},
				claimed:     []*cdiv1.DataVolume{newDataVolume("datavolume1", &controller)},
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller2 := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			controller2.UID = types.UID("AAAAA")
			return test{
				name: "Controller can not claim datavolumes owned by another controller",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				datavolumes: []*cdiv1.DataVolume{newDataVolume("datavolume1", &controller), newDataVolume("datavolume2", &controller2)},
				claimed:     []*cdiv1.DataVolume{newDataVolume("datavolume1", &controller)},
			}
		}(),
		func() test {
			controller := v1.ReplicationController{}
			controller.UID = types.UID(controllerUID)
			datavolumeToDelete1 := newDataVolume("datavolume1", &controller)
			datavolumeToDelete2 := newDataVolume("datavolume2", nil)
			now := metav1.Now()
			datavolumeToDelete1.DeletionTimestamp = &now
			datavolumeToDelete2.DeletionTimestamp = &now

			return test{
				name: "Controller does not claim orphaned datavolumes marked for deletion",
				manager: NewVirtualMachineControllerRefManager(&FakeVirtualMachineControl{},
					&controller,
					productionLabelSelector,
					controllerKind,
					func() error { return nil }),
				datavolumes: []*cdiv1.DataVolume{datavolumeToDelete1, datavolumeToDelete2},
				claimed:     []*cdiv1.DataVolume{datavolumeToDelete1},
			}
		}(),
	}
	for _, test := range tests {
		claimed, err := test.manager.ClaimMatchedDataVolumes(test.datavolumes)
		if test.expectError && err == nil {
			t.Errorf("Test case `%s`, expected error but got nil", test.name)
		} else if !reflect.DeepEqual(test.claimed, claimed) {
			t.Errorf("Test case `%s`, claimed wrong datavolumes. Expected %v, got %v", test.name, datavolumeToStringSlice(test.claimed), datavolumeToStringSlice(claimed))
		}

	}
}

func datavolumeToStringSlice(dataVolumes []*cdiv1.DataVolume) []string {
	var names []string
	for _, dv := range dataVolumes {
		names = append(names, dv.Name)
	}
	return names
}

func virtualmachineToStringSlice(virtualmachines []*virtv1.VirtualMachineInstance) []string {
	var names []string
	for _, virtualmachine := range virtualmachines {
		names = append(names, virtualmachine.Name)
	}
	return names
}

type FakeVirtualMachineControl struct {
	sync.Mutex
	ControllerRefs []metav1.OwnerReference
	Patches        [][]byte
	Err            error
}

var _ VirtualMachineControlInterface = &FakeVirtualMachineControl{}

func (f *FakeVirtualMachineControl) PatchVirtualMachine(_, _ string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}
func (f *FakeVirtualMachineControl) PatchDataVolume(_, _ string, data []byte) error {
	f.Lock()
	defer f.Unlock()
	f.Patches = append(f.Patches, data)
	if f.Err != nil {
		return f.Err
	}
	return nil
}
