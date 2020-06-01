package shared

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureFinalizer ensures that the object's finalizer is included
// in the ObjectMeta Finalizers slice. If it already exists, no state change occurs.
// If it doesn't, the finalizer is appended to the slice.
func EnsureFinalizer(objectMeta *metav1.ObjectMeta, expectedFinalizer string) {
	// First check if the finalizer is already included in the object.
	for _, finalizer := range objectMeta.Finalizers {
		if finalizer == expectedFinalizer {
			return
		}
	}

	// If it doesn't exist, append the finalizer to the object meta.
	objectMeta.Finalizers = append(objectMeta.Finalizers, expectedFinalizer)

	return
}

// RemoveFinalizer removes the finalizer from the object's ObjectMeta.
func RemoveFinalizer(objectMeta *metav1.ObjectMeta, deletingFinalizer string) {
	outFinalizers := make([]string, 0)
	for _, finalizer := range objectMeta.Finalizers {
		if finalizer == deletingFinalizer {
			continue
		}
		outFinalizers = append(outFinalizers, finalizer)
	}

	objectMeta.Finalizers = outFinalizers

	return
}
