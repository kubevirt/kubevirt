//nolint:dupl,lll
package instancetype

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
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
		newObject := &instancetypev1beta1.VirtualMachinePreference{}
		if err := instancetypev1alpha1.Convert_v1alpha1_VirtualMachinePreference_To_v1beta1_VirtualMachinePreference(oldObject, newObject, nil); err != nil {
			return nil, err
		}
		return newObject, nil
	}

	oldObject, err := decodeOldInstancetypeRevisionObject(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode old revision object: %w", err)
	}
	if oldObject == nil {
		return nil, nil
	}
	newObject := &instancetypev1beta1.VirtualMachineInstancetype{}
	if err := instancetypev1alpha1.Convert_v1alpha1_VirtualMachineInstancetype_To_v1beta1_VirtualMachineInstancetype(oldObject, newObject, nil); err != nil {
		return nil, err
	}
	return newObject, nil
}

func decodeOldInstancetypeRevisionObject(data []byte) (*instancetypev1alpha1.VirtualMachineInstancetype, error) {
	revision := &instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{}
	err := json.Unmarshal(data, revision)
	if err != nil {
		// Failed to unmarshal, so the object is not the expected type
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
