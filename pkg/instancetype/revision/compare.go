/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package revision

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
)

func Compare(revisionA, revisionB *appsv1.ControllerRevision) (bool, error) {
	if err := compatibility.Decode(revisionA); err != nil {
		return false, err
	}

	if err := compatibility.Decode(revisionB); err != nil {
		return false, err
	}

	revisionASpec, err := getSpec(revisionA.Data.Object)
	if err != nil {
		return false, err
	}

	revisionBSpec, err := getSpec(revisionB.Data.Object)
	if err != nil {
		return false, err
	}

	return equality.Semantic.DeepEqual(revisionASpec, revisionBSpec), nil
}

func getSpec(obj runtime.Object) (interface{}, error) {
	switch o := obj.(type) {
	case *v1beta1.VirtualMachineInstancetype:
		return &o.Spec, nil
	case *v1beta1.VirtualMachineClusterInstancetype:
		return &o.Spec, nil
	case *v1beta1.VirtualMachinePreference:
		return &o.Spec, nil
	case *v1beta1.VirtualMachineClusterPreference:
		return &o.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}
}
