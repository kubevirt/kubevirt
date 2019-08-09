package v1

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

var errMinObjectRef = errors.New("object reference must have, at a minimum: apiVersion, kind, and name")

// SetObjectReference - updates list of object references based on newObject
func SetObjectReference(objects *[]corev1.ObjectReference, newObject corev1.ObjectReference) error {
	if !minObjectReference(newObject) {
		return errMinObjectRef
	}

	if objects == nil {
		objects = &[]corev1.ObjectReference{}
	}
	existingObject, err := FindObjectReference(*objects, newObject)
	if err != nil {
		return err
	}
	if existingObject == nil { // add it to the slice
		*objects = append(*objects, newObject)
	} else { // update found reference
		*existingObject = newObject
	}
	return nil
}

// RemoveObjectReference - updates list of object references to remove rmObject
func RemoveObjectReference(objects *[]corev1.ObjectReference, rmObject corev1.ObjectReference) error {
	if !minObjectReference(rmObject) {
		return errMinObjectRef
	}

	if objects == nil {
		return nil
	}
	newObjectReferences := []corev1.ObjectReference{}
	// TODO: this is incredibly inefficient. If the performance hit becomes a
	// problem this should be improved.
	for _, object := range *objects {
		if !ObjectReferenceEqual(object, rmObject) {
			newObjectReferences = append(newObjectReferences, object)
		}
	}

	*objects = newObjectReferences
	return nil
}

// FindObjectReference - finds the first ObjectReference in a slice of objects
// matching find.
func FindObjectReference(objects []corev1.ObjectReference, find corev1.ObjectReference) (*corev1.ObjectReference, error) {
	if !minObjectReference(find) {
		return nil, errMinObjectRef
	}

	for i := range objects {
		if ObjectReferenceEqual(find, objects[i]) {
			return &objects[i], nil
		}
	}

	return nil, nil
}

// ObjectReferenceEqual - compares gotRef to expectedRef
// preference order: APIVersion, Kind, Name, and Namespace
// if either gotRef or expectedRef fail minObjectReference test, this function
// will simply return false
func ObjectReferenceEqual(gotRef, expectedRef corev1.ObjectReference) bool {
	if !minObjectReference(gotRef) || !minObjectReference(expectedRef) {
		return false
	}
	if gotRef.APIVersion != expectedRef.APIVersion {
		return false
	}
	if gotRef.Kind != expectedRef.Kind {
		return false
	}
	if gotRef.Name != expectedRef.Name {
		return false
	}
	if expectedRef.Namespace != "" && (gotRef.Namespace != expectedRef.Namespace) {
		return false
	}
	return true
}

// in order to have any meaningful semantics on this we need to
// ensuer that some minimal amount of information is provided in
// the object reference
func minObjectReference(objRef corev1.ObjectReference) bool {
	if objRef.APIVersion == "" {
		return false
	}
	if objRef.Kind == "" {
		return false
	}
	if objRef.Name == "" {
		return false
	}

	return true
}
