// This file was copied from: github.com/openshift/library-go/pkg/operator/resource/resourcemerge
//
// Here is the link to specific file and commit from which it was copied:
// https://github.com/openshift/library-go/blob/eca2c467c492/pkg/operator/resource/resourcemerge/object_merger.go

package resourcemerge

import (
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EnsureObjectMeta writes namespace, name, labels, and annotations.
func EnsureObjectMeta(existing *metav1.ObjectMeta, required metav1.ObjectMeta) bool {
	var modified bool

	modified = modified || setStringIfSet(&existing.Namespace, required.Namespace)
	modified = modified || setStringIfSet(&existing.Name, required.Name)
	modified = modified || mergeMap(&existing.Labels, required.Labels)
	modified = modified || mergeMap(&existing.Annotations, required.Annotations)
	modified = modified || mergeOwnerRefs(&existing.OwnerReferences, required.OwnerReferences)

	return modified
}

func setStringIfSet(existing *string, required string) bool {
	if required == "" {
		return false
	}
	if required != *existing {
		*existing = required
		return true
	}
	return false
}

func mergeMap(existing *map[string]string, required map[string]string) bool { //nolint:gocritic
	var modified bool
	if *existing == nil {
		*existing = map[string]string{}
	}
	for k, v := range required {
		actualKey := k
		removeKey := false

		// if "required" map contains a key with "-" as suffix, remove that
		// key from the existing map instead of replacing the value
		if strings.HasSuffix(k, "-") {
			removeKey = true
			actualKey = strings.TrimRight(k, "-")
		}

		if existingV, ok := (*existing)[actualKey]; removeKey {
			if !ok {
				continue
			}
			// value found -> it should be removed
			delete(*existing, actualKey)
			modified = true
		} else if !ok || v != existingV {
			modified = true
			(*existing)[actualKey] = v
		}
	}
	return modified
}

func mergeOwnerRefs(existing *[]metav1.OwnerReference, required []metav1.OwnerReference) bool {
	var modified bool

	if *existing == nil {
		*existing = []metav1.OwnerReference{}
	}

	for _, o := range required {
		removeOwner := false

		// if "required" ownerRefs contain an owner.UID with "-" as suffix, remove that
		// ownerRef from the existing ownerRefs instead of replacing the value
		// NOTE: this is the same format as kubectl annotate and kubectl label
		if strings.HasSuffix(string(o.UID), "-") {
			removeOwner = true
		}

		existedIndex := 0

		for existedIndex < len(*existing) {
			if ownerRefMatched(o, (*existing)[existedIndex]) {
				break
			}
			existedIndex++
		}

		if existedIndex == len(*existing) {
			// There is no matched ownerref found, append the ownerref
			// if it is not to be removed.
			if !removeOwner {
				*existing = append(*existing, o)
				modified = true
			}
			continue
		}

		if removeOwner {
			*existing = append((*existing)[:existedIndex], (*existing)[existedIndex+1:]...)
			modified = true
			continue
		}

		if !reflect.DeepEqual(o, (*existing)[existedIndex]) {
			(*existing)[existedIndex] = o
			modified = true
		}
	}
	return modified
}

func ownerRefMatched(existing, required metav1.OwnerReference) bool {
	if existing.Name != required.Name {
		return false
	}

	if existing.Kind != required.Kind {
		return false
	}

	existingGV, err := schema.ParseGroupVersion(existing.APIVersion)
	if err != nil {
		return false
	}

	requiredGV, err := schema.ParseGroupVersion(required.APIVersion)
	if err != nil {
		return false
	}

	if existingGV.Group != requiredGV.Group {
		return false
	}

	return true
}
