//nolint:lll
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
package compatibility

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
)

func GetInstancetypeSpec(revision *appsv1.ControllerRevision) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1beta1.VirtualMachineInstancetype:
		return &obj.Spec, nil
	case *v1beta1.VirtualMachineClusterInstancetype:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func GetPreferenceSpec(revision *appsv1.ControllerRevision) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1beta1.VirtualMachinePreference:
		return &obj.Spec, nil
	case *v1beta1.VirtualMachineClusterPreference:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func Decode(revision *appsv1.ControllerRevision) error {
	if len(revision.Data.Raw) == 0 {
		return nil
	}
	return decodeControllerRevisionObject(revision)
}

func decodeControllerRevisionObject(revision *appsv1.ControllerRevision) error {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), revision.Data.Raw)
	if err != nil {
		return fmt.Errorf("failed to decode object in ControllerRevision: %w", err)
	}
	revision.Data.Object = decodedObj
	switch obj := revision.Data.Object.(type) {
	case *v1beta1.VirtualMachineInstancetype, *v1beta1.VirtualMachineClusterInstancetype, *v1beta1.VirtualMachinePreference, *v1beta1.VirtualMachineClusterPreference:
		return nil
	default:
		return fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}
