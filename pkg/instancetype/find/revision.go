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
 * Copyright The KubeVirt Authors
 *
 */
package find

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type revisionFinder struct {
	controllerRevisionFinder *controllerRevisionFinder
}

func NewRevisionFinder(store cache.Store, virtClient kubecli.KubevirtClient) *revisionFinder {
	return &revisionFinder{
		controllerRevisionFinder: NewControllerRevisionFinder(store, virtClient),
	}
}

func (f *revisionFinder) Find(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	// Avoid a race with Store() here and use RevisionName if already provided over Whatever is in ControllerRevisionRef
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.RevisionName != "" {
		return f.controllerRevisionFinder.Find(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      vm.Spec.Instancetype.RevisionName,
		})
	}
	ref := vm.Status.InstancetypeRef
	if ref != nil && ref.ControllerRevisionRef != nil && ref.ControllerRevisionRef.Name != "" {
		return f.controllerRevisionFinder.Find(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      ref.ControllerRevisionRef.Name,
		})
	}
	return nil, nil
}
