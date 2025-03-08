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

// EnsureObjectMeta writes namespace, name, labels, and annotations.  Don't set other things here.
func EnsureObjectMeta(modified *bool, existing *metav1.ObjectMeta, required metav1.ObjectMeta) {
	setStringIfSet(modified, &existing.Namespace, required.Namespace)
	setStringIfSet(modified, &existing.Name, required.Name)
	mergeMap(modified, &existing.Labels, required.Labels)
	mergeMap(modified, &existing.Annotations, required.Annotations)
	mergeOwnerRefs(modified, &existing.OwnerReferences, required.OwnerReferences)
}

func setStringIfSet(modified *bool, existing *string, required string) {
	if required == "" {
		return
	}
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func mergeMap(modified *bool, existing *map[string]string, required map[string]string) { //nolint:gocritic
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
			*modified = true
		} else if !ok || v != existingV {
			*modified = true
			(*existing)[actualKey] = v
		}
	}
}

func mergeOwnerRefs(modified *bool, existing *[]metav1.OwnerReference, required []metav1.OwnerReference) {
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
				*modified = true
			}
			continue
		}

		if removeOwner {
			*existing = append((*existing)[:existedIndex], (*existing)[existedIndex+1:]...)
			*modified = true
			continue
		}

		if !reflect.DeepEqual(o, (*existing)[existedIndex]) {
			(*existing)[existedIndex] = o
			*modified = true
		}
	}
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
