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

	instancetype "kubevirt.io/api/instancetype"
	v1 "kubevirt.io/api/instancetype/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
)

func GetInstancetypeSpec(revision *appsv1.ControllerRevision) (*v1.VirtualMachineInstancetypeSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1.VirtualMachineInstancetype:
		return &obj.Spec, nil
	case *v1.VirtualMachineClusterInstancetype:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func GetPreferenceSpec(revision *appsv1.ControllerRevision) (*v1.VirtualMachinePreferenceSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1.VirtualMachinePreference:
		return &obj.Spec, nil
	case *v1.VirtualMachineClusterPreference:
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

	// Convert to v1 if needed using scheme-based conversion
	var convertedObj runtime.Object
	switch decodedObj.(type) {
	case *v1beta1.VirtualMachineInstancetype, *v1beta1.VirtualMachineClusterInstancetype,
		*v1beta1.VirtualMachinePreference, *v1beta1.VirtualMachineClusterPreference:
		convertedObj, err = convertToV1(decodedObj)
		if err != nil {
			return err
		}
	case *v1.VirtualMachineInstancetype, *v1.VirtualMachineClusterInstancetype,
		*v1.VirtualMachinePreference, *v1.VirtualMachineClusterPreference:
		// Already v1, no conversion needed
		convertedObj = decodedObj
	default:
		return fmt.Errorf("unexpected type in ControllerRevision: %T", decodedObj)
	}

	revision.Data.Object = convertedObj
	return nil
}

// convertToV1 converts v1beta1 objects to v1 using scheme-based conversion via internal types
func convertToV1(in runtime.Object) (runtime.Object, error) {
	// Convert v1beta1 -> internal
	internal, err := generatedscheme.Scheme.ConvertToVersion(in, instancetype.SchemeGroupVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to internal: %w", err)
	}

	// Convert internal -> v1
	out, err := generatedscheme.Scheme.ConvertToVersion(internal, v1.SchemeGroupVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to v1: %w", err)
	}

	return out, nil
}

// ConvertToV1 converts v1beta1 objects to v1. This is exported for use by other packages.
func ConvertToV1(in runtime.Object) (runtime.Object, error) {
	switch in.(type) {
	case *v1beta1.VirtualMachineInstancetype, *v1beta1.VirtualMachineClusterInstancetype,
		*v1beta1.VirtualMachinePreference, *v1beta1.VirtualMachineClusterPreference:
		return convertToV1(in)
	case *v1.VirtualMachineInstancetype, *v1.VirtualMachineClusterInstancetype,
		*v1.VirtualMachinePreference, *v1.VirtualMachineClusterPreference:
		// Already v1, return as is
		return in, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", in)
	}
}
