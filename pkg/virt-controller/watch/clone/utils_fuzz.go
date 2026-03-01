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

package clone

import (
	"k8s.io/client-go/util/workqueue"
	clonev1beta1 "kubevirt.io/api/clone/v1beta1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

// These utilities are exposed here for the fuzzer in ./fuzz to use.
func ShutdownCtrlQueue(ctrl *VMCloneController) {
	ctrl.vmCloneQueue.ShutDown()
}

func SetQueue(ctrl *VMCloneController, newQueue *testutils.MockWorkQueue[string]) {
	ctrl.vmCloneQueue = newQueue
}

func AddToVmStore(ctrl *VMCloneController, vm *virtv1.VirtualMachine) {
	ctrl.vmStore.Add(vm)
}

func AddTovmCloneIndexer(ctrl *VMCloneController, vmc *clonev1beta1.VirtualMachineClone) {
	ctrl.vmCloneIndexer.Add(vmc)
}

func GetVmCloneQueue(ctrl *VMCloneController) workqueue.TypedRateLimitingInterface[string] {
	return ctrl.vmCloneQueue
}
