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
package vm

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	v1 "kubevirt.io/api/core/v1"
)

// SubresourceHandle should be use for status Update/Patch
func SubresourceHandle(subresources string) bool {
	return subresources == "status"
}

// Handle should be used for Update/Patch without status
func Handle(subresources string) bool {
	return subresources != "status"
}

// ModifyStatusOnlyVM ignores any updates other than to status
func ModifyStatusOnlyVM(new, old runtime.Object) runtime.Object {
	vm := new.(*v1.VirtualMachine)
	oldVM := old.(*v1.VirtualMachine)

	oldVM = oldVM.DeepCopy()
	oldVM.Status = *vm.Status.DeepCopy()
	return oldVM
}

// ModifyVM ignores updates to status
func ModifyVM(new, old runtime.Object) runtime.Object {
	vm := new.(*v1.VirtualMachine)
	oldVM := old.(*v1.VirtualMachine)

	oldVM = oldVM.DeepCopy()
	oldVM.Spec = *vm.Spec.DeepCopy()
	oldVM.ObjectMeta = *vm.ObjectMeta.DeepCopy()
	return oldVM
}

// UpdateReactor should be used to replace default reactor
// handle - takes subresources and should return if the request should be handled, e.g /status
// modify - takes new and old object and should return object that should be stored
func UpdateReactor(handle func(string) bool, tracker testing.ObjectTracker,
	modify func(new, old runtime.Object) runtime.Object,
) func(action testing.Action) (handled bool, ret runtime.Object, err error) {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		if !handle(action.GetSubresource()) {
			return false, nil, nil
		}
		switch action := action.(type) {
		case testing.UpdateActionImpl:
			objMeta, err := meta.Accessor(action.GetObject())
			if err != nil {
				return true, nil, err
			}

			oldObj, err := tracker.Get(action.GetResource(), action.GetNamespace(), objMeta.GetName())
			if err != nil {
				return true, nil, err
			}
			modifiedObj := modify(action.GetObject(), oldObj)

			err = tracker.Update(action.GetResource(), modifiedObj, action.GetNamespace())
			if err != nil {
				return true, nil, err
			}
			oldObj, err = tracker.Get(action.GetResource(), action.GetNamespace(), objMeta.GetName())
			return true, oldObj, err
		default:
			panic("Unexpected action implementation")
		}
	}

}
