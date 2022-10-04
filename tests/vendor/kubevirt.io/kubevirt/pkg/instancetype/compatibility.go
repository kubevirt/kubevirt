package instancetype

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
)

func decodeOldObject(data []byte, isPreference bool) (runtime.Object, error) {
	if isPreference {
		oldObject, err := decodeOldPreferenceRevisionObject(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode old revision object: %w", err)
		}
		if oldObject == nil {
			return nil, nil
		}
		return convertPreference(oldObject)
	}

	oldObject, err := decodeOldInstancetypeRevisionObject(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode old revision object: %w", err)
	}
	if oldObject == nil {
		return nil, nil
	}
	return convertInstancetype(oldObject)
}

func decodeOldInstancetypeRevisionObject(data []byte) (*instancetypev1alpha1.VirtualMachineInstancetype, error) {
	revision := &instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{}
	err := json.Unmarshal(data, revision)
	if err != nil {
		// Failed to unmarshal, so the object is not the expected type
		return nil, nil
	}

	if revision.APIVersion != instancetypev1alpha1.SchemeGroupVersion.String() {
		return nil, nil
	}

	instancetypeSpec := &instancetypev1alpha1.VirtualMachineInstancetypeSpec{}
	err = json.Unmarshal(revision.Spec, instancetypeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json into v1alpha1.VirtualMachineInstancetypeSpec: %w", err)
	}

	return &instancetypev1alpha1.VirtualMachineInstancetype{
		TypeMeta: metav1.TypeMeta{
			APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachineInstancetype",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-version-of-instancetype-object",
		},
		Spec: *instancetypeSpec,
	}, nil
}

func decodeOldPreferenceRevisionObject(data []byte) (*instancetypev1alpha1.VirtualMachinePreference, error) {
	revision := &instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{}
	err := json.Unmarshal(data, revision)
	if err != nil {
		// Failed to unmarshall, so the object is not the expected type
		return nil, nil
	}

	if revision.APIVersion != instancetypev1alpha1.SchemeGroupVersion.String() {
		return nil, nil
	}

	preferenceSpec := &instancetypev1alpha1.VirtualMachinePreferenceSpec{}
	err = json.Unmarshal(revision.Spec, preferenceSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall json into v1alpha1.VirtualMachinePreferenceSpec: %w", err)
	}

	return &instancetypev1alpha1.VirtualMachinePreference{
		TypeMeta: metav1.TypeMeta{
			APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachinePreference",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "old-version-of-preference-object",
		},
		Spec: *preferenceSpec,
	}, nil
}

// Manually written conversion functions.
// TODO: Use conversion-gen to generate conversion functions

func convertInstancetype(source *instancetypev1alpha1.VirtualMachineInstancetype) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
	// This is a slow conversion based on json serialization. The v1alpha2 is compatible with v1alpha1.
	jsonBytes, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall source object to JSON: %w", err)
	}

	destination := &instancetypev1alpha2.VirtualMachineInstancetype{}
	err = json.Unmarshal(jsonBytes, destination)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall JSON to VirtualMachineInstancetype: %w", err)
	}

	destination.TypeMeta = metav1.TypeMeta{
		APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
		Kind:       "VirtualMachineInstancetype",
	}

	return destination, nil
}

func convertPreference(source *instancetypev1alpha1.VirtualMachinePreference) (*instancetypev1alpha2.VirtualMachinePreference, error) {
	// This is a slow conversion based on json serialization. The v1alpha2 is compatible with v1alpha1.
	jsonBytes, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall source object to JSON: %w", err)
	}

	destination := &instancetypev1alpha2.VirtualMachinePreference{}
	err = json.Unmarshal(jsonBytes, destination)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall JSON to VirtualMachinePreference: %w", err)
	}

	destination.TypeMeta = metav1.TypeMeta{
		APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
		Kind:       "VirtualMachinePreference",
	}

	return nil, nil
}
