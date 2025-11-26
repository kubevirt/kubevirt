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
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/instancetype/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
)

func GetInstancetypeSpec(revision *appsv1.ControllerRevision) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	if err := Decode(revision); err != nil {
		return nil, err
	}
	switch obj := revision.Data.Object.(type) {
	case *v1.VirtualMachineInstancetype:
		return convertV1InstancetypeSpecToV1beta1(&obj.Spec)
	case *v1.VirtualMachineClusterInstancetype:
		return convertV1InstancetypeSpecToV1beta1(&obj.Spec)
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
	case *v1.VirtualMachinePreference:
		return convertV1PreferenceSpecToV1beta1(&obj.Spec)
	case *v1.VirtualMachineClusterPreference:
		return convertV1PreferenceSpecToV1beta1(&obj.Spec)
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

	// Convert v1beta1 objects to v1
	var convertedObj runtime.Object
	switch obj := decodedObj.(type) {
	case *v1beta1.VirtualMachineInstancetype:
		convertedObj, err = convertV1beta1InstancetypeToV1(obj)
		if err != nil {
			return err
		}
	case *v1beta1.VirtualMachineClusterInstancetype:
		convertedObj, err = convertV1beta1ClusterInstancetypeToV1(obj)
		if err != nil {
			return err
		}
	case *v1beta1.VirtualMachinePreference:
		convertedObj, err = convertV1beta1PreferenceToV1(obj)
		if err != nil {
			return err
		}
	case *v1beta1.VirtualMachineClusterPreference:
		convertedObj, err = convertV1beta1ClusterPreferenceToV1(obj)
		if err != nil {
			return err
		}
	case *v1.VirtualMachineInstancetype, *v1.VirtualMachineClusterInstancetype, *v1.VirtualMachinePreference, *v1.VirtualMachineClusterPreference:
		// Already v1, no conversion needed
		convertedObj = decodedObj
	default:
		return fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}

	revision.Data.Object = convertedObj
	return nil
}

func convertV1InstancetypeSpecToV1beta1(in *v1.VirtualMachineInstancetypeSpec) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	out := &v1beta1.VirtualMachineInstancetypeSpec{}
	if err := Convert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec(in, out, nil); err != nil {
		return nil, err
	}
	return out, nil
}

func convertV1PreferenceSpecToV1beta1(in *v1.VirtualMachinePreferenceSpec) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	out := &v1beta1.VirtualMachinePreferenceSpec{}
	if err := Convert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec(in, out, nil); err != nil {
		return nil, err
	}
	return out, nil
}

// convertV1beta1InstancetypeToV1 converts a v1beta1 VirtualMachineInstancetype to v1
func convertV1beta1InstancetypeToV1(in *v1beta1.VirtualMachineInstancetype) (*v1.VirtualMachineInstancetype, error) {
	out := &v1.VirtualMachineInstancetype{}
	if err := Convert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype(in, out, nil); err != nil {
		return nil, err
	}
	// Set the GVK to v1 after conversion
	out.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("VirtualMachineInstancetype"))
	return out, nil
}

// convertV1beta1ClusterInstancetypeToV1 converts a v1beta1 VirtualMachineClusterInstancetype to v1
func convertV1beta1ClusterInstancetypeToV1(in *v1beta1.VirtualMachineClusterInstancetype) (*v1.VirtualMachineClusterInstancetype, error) {
	out := &v1.VirtualMachineClusterInstancetype{}
	if err := Convert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype(in, out, nil); err != nil {
		return nil, err
	}
	// Set the GVK to v1 after conversion
	out.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("VirtualMachineClusterInstancetype"))
	return out, nil
}

// convertV1beta1PreferenceToV1 converts a v1beta1 VirtualMachinePreference to v1
func convertV1beta1PreferenceToV1(in *v1beta1.VirtualMachinePreference) (*v1.VirtualMachinePreference, error) {
	out := &v1.VirtualMachinePreference{}
	if err := Convert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference(in, out, nil); err != nil {
		return nil, err
	}
	// Set the GVK to v1 after conversion
	out.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("VirtualMachinePreference"))
	return out, nil
}

// convertV1beta1ClusterPreferenceToV1 converts a v1beta1 VirtualMachineClusterPreference to v1
func convertV1beta1ClusterPreferenceToV1(in *v1beta1.VirtualMachineClusterPreference) (*v1.VirtualMachineClusterPreference, error) {
	out := &v1.VirtualMachineClusterPreference{}
	if err := Convert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference(in, out, nil); err != nil {
		return nil, err
	}
	// Set the GVK to v1 after conversion
	out.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("VirtualMachineClusterPreference"))
	return out, nil
}

// Conversion functions following conversion-gen naming conventions
// These functions use the same signatures and naming patterns that conversion-gen would generate,
// making future migration to generated code seamless.
//
// Note: These are located in the compatibility package rather than the API packages to avoid
// import cycles between v1 and v1beta1. When conversion-gen is configured for these types,
// it may generate similar functions in the API packages.

// autoConvert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype performs the conversion from v1beta1 to v1.
// Since the types are structurally identical, we use JSON marshaling/unmarshaling.
func autoConvert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype(in *v1beta1.VirtualMachineInstancetype, out *v1.VirtualMachineInstancetype, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype is the public conversion function.
// This matches what conversion-gen would generate.
func Convert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype(in *v1beta1.VirtualMachineInstancetype, out *v1.VirtualMachineInstancetype, s conversion.Scope) error {
	return autoConvert_v1beta1_VirtualMachineInstancetype_To_v1_VirtualMachineInstancetype(in, out, s)
}

// autoConvert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype performs the conversion from v1beta1 to v1.
func autoConvert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype(in *v1beta1.VirtualMachineClusterInstancetype, out *v1.VirtualMachineClusterInstancetype, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype is the public conversion function.
func Convert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype(in *v1beta1.VirtualMachineClusterInstancetype, out *v1.VirtualMachineClusterInstancetype, s conversion.Scope) error {
	return autoConvert_v1beta1_VirtualMachineClusterInstancetype_To_v1_VirtualMachineClusterInstancetype(in, out, s)
}

// autoConvert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference performs the conversion from v1beta1 to v1.
func autoConvert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference(in *v1beta1.VirtualMachinePreference, out *v1.VirtualMachinePreference, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference is the public conversion function.
func Convert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference(in *v1beta1.VirtualMachinePreference, out *v1.VirtualMachinePreference, s conversion.Scope) error {
	return autoConvert_v1beta1_VirtualMachinePreference_To_v1_VirtualMachinePreference(in, out, s)
}

// autoConvert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference performs the conversion from v1beta1 to v1.
func autoConvert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference(in *v1beta1.VirtualMachineClusterPreference, out *v1.VirtualMachineClusterPreference, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference is the public conversion function.
func Convert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference(in *v1beta1.VirtualMachineClusterPreference, out *v1.VirtualMachineClusterPreference, s conversion.Scope) error {
	return autoConvert_v1beta1_VirtualMachineClusterPreference_To_v1_VirtualMachineClusterPreference(in, out, s)
}

// autoConvert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec performs the conversion from v1 to v1beta1.
func autoConvert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec(in *v1.VirtualMachineInstancetypeSpec, out *v1beta1.VirtualMachineInstancetypeSpec, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec is the public conversion function.
func Convert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec(in *v1.VirtualMachineInstancetypeSpec, out *v1beta1.VirtualMachineInstancetypeSpec, s conversion.Scope) error {
	return autoConvert_v1_VirtualMachineInstancetypeSpec_To_v1beta1_VirtualMachineInstancetypeSpec(in, out, s)
}

// autoConvert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec performs the conversion from v1 to v1beta1.
func autoConvert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec(in *v1.VirtualMachinePreferenceSpec, out *v1beta1.VirtualMachinePreferenceSpec, s conversion.Scope) error {
	return convertViaJSON(in, out)
}

// Convert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec is the public conversion function.
func Convert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec(in *v1.VirtualMachinePreferenceSpec, out *v1beta1.VirtualMachinePreferenceSpec, s conversion.Scope) error {
	return autoConvert_v1_VirtualMachinePreferenceSpec_To_v1beta1_VirtualMachinePreferenceSpec(in, out, s)
}

// convertViaJSON converts between structurally identical types using JSON marshaling/unmarshaling.
// This is a temporary implementation used until conversion-gen generates the actual field-by-field conversion.
func convertViaJSON(in, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal object during conversion: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to unmarshal object during conversion: %w", err)
	}
	return nil
}
