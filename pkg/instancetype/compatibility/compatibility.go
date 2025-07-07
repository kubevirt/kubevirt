//nolint:dupl,lll,gocyclo
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
package compatibility

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/json"

	"kubevirt.io/api/instancetype/v1alpha1"
	"kubevirt.io/api/instancetype/v1alpha2"
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

	// Backward compatibility check. Try to decode ControllerRevision from v1alpha1 version.
	oldObject, err := decodeSpecRevision(revision.Data.Raw)
	if err != nil {
		return fmt.Errorf("failed to decode old ControllerRevision: %w", err)
	}
	if oldObject != nil {
		revision.Data.Object = oldObject
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
	case *v1alpha2.VirtualMachineInstancetype:
		dest := &v1beta1.VirtualMachineInstancetype{}
		if err := v1alpha2.Convert_v1alpha2_VirtualMachineInstancetype_To_v1beta1_VirtualMachineInstancetype(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha2.VirtualMachineClusterInstancetype:
		dest := &v1beta1.VirtualMachineClusterInstancetype{}
		if err := v1alpha2.Convert_v1alpha2_VirtualMachineClusterInstancetype_To_v1beta1_VirtualMachineClusterInstancetype(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha2.VirtualMachinePreference:
		dest := &v1beta1.VirtualMachinePreference{}
		if err := v1alpha2.Convert_v1alpha2_VirtualMachinePreference_To_v1beta1_VirtualMachinePreference(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha2.VirtualMachineClusterPreference:
		dest := &v1beta1.VirtualMachineClusterPreference{}
		if err := v1alpha2.Convert_v1alpha2_VirtualMachineClusterPreference_To_v1beta1_VirtualMachineClusterPreference(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha1.VirtualMachineInstancetype:
		dest := &v1beta1.VirtualMachineInstancetype{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachineInstancetype_To_v1beta1_VirtualMachineInstancetype(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha1.VirtualMachineClusterInstancetype:
		dest := &v1beta1.VirtualMachineClusterInstancetype{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachineClusterInstancetype_To_v1beta1_VirtualMachineClusterInstancetype(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha1.VirtualMachinePreference:
		dest := &v1beta1.VirtualMachinePreference{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachinePreference_To_v1beta1_VirtualMachinePreference(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	case *v1alpha1.VirtualMachineClusterPreference:
		dest := &v1beta1.VirtualMachineClusterPreference{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachineClusterPreference_To_v1beta1_VirtualMachineClusterPreference(obj, dest, nil); err != nil {
			return err
		}
		revision.Data.Object = dest
	default:
		return fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
	return nil
}

func decodeSpecRevision(data []byte) (runtime.Object, error) {
	if oldPreferenceObject := decodeVirtualMachinePreferenceSpecRevision(data); oldPreferenceObject != nil {
		newPreferenceObject := &v1beta1.VirtualMachinePreference{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachinePreference_To_v1beta1_VirtualMachinePreference(oldPreferenceObject, newPreferenceObject, nil); err != nil {
			return nil, err
		}
		return newPreferenceObject, nil
	}

	if oldInstancetypeObject := decodeVirtualMachineInstancetypeSpecRevision(data); oldInstancetypeObject != nil {
		newInstancetypeObject := &v1beta1.VirtualMachineInstancetype{}
		if err := v1alpha1.Convert_v1alpha1_VirtualMachineInstancetype_To_v1beta1_VirtualMachineInstancetype(oldInstancetypeObject, newInstancetypeObject, nil); err != nil {
			return nil, err
		}
		return newInstancetypeObject, nil
	}

	return nil, nil
}

func decodeVirtualMachineInstancetypeSpecRevision(data []byte) *v1alpha1.VirtualMachineInstancetype {
	revision := &v1alpha1.VirtualMachineInstancetypeSpecRevision{}
	strictErr, err := json.UnmarshalStrict(data, revision)
	if err != nil || strictErr != nil {
		// Failed to unmarshal, so the object is not the expected type
		return nil
	}

	instancetypeSpec := &v1alpha1.VirtualMachineInstancetypeSpec{}
	strictErr, err = json.UnmarshalStrict(revision.Spec, instancetypeSpec)
	if err != nil || strictErr != nil {
		return nil
	}

	return &v1alpha1.VirtualMachineInstancetype{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachineInstancetype",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-version-of-instancetype-object",
		},
		Spec: *instancetypeSpec,
	}
}

func decodeVirtualMachinePreferenceSpecRevision(data []byte) *v1alpha1.VirtualMachinePreference {
	revision := &v1alpha1.VirtualMachinePreferenceSpecRevision{}
	strictErr, err := json.UnmarshalStrict(data, revision)
	if err != nil || strictErr != nil {
		// Failed to unmarshall, so the object is not the expected type
		return nil
	}

	preferenceSpec := &v1alpha1.VirtualMachinePreferenceSpec{}
	strictErr, err = json.UnmarshalStrict(revision.Spec, preferenceSpec)
	if err != nil || strictErr != nil {
		return nil
	}

	return &v1alpha1.VirtualMachinePreference{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachinePreference",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-version-of-preference-object",
		},
		Spec: *preferenceSpec,
	}
}
