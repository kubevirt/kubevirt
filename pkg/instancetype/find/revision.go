/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package find

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
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
		cr, err := f.controllerRevisionFinder.Find(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      ref.ControllerRevisionRef.Name,
		})
		if err != nil {
			return nil, err
		}
		// Only return the found CR if it is for the referenced instance type
		if label, ok := cr.Labels[instancetypeapi.ControllerRevisionObjectNameLabel]; ok && label == vm.Spec.Instancetype.Name {
			return cr, nil
		}
	}
	return nil, nil
}
