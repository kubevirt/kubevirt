package resourcemerge

import (
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureObjectMeta writes namespace, name, labels, and annotations.  Don't set other things here.
// TODO finalizer support maybe?
func EnsureObjectMeta(modified *bool, existing *metav1.ObjectMeta, required metav1.ObjectMeta) {
	SetStringIfSet(modified, &existing.Namespace, required.Namespace)
	SetStringIfSet(modified, &existing.Name, required.Name)
	MergeMap(modified, &existing.Labels, required.Labels)
	MergeMap(modified, &existing.Annotations, required.Annotations)
}

// WithCleanLabelsAndAnnotations cleans the metadata off the removal annotations/labels
// (those that end with trailing "-")
func WithCleanLabelsAndAnnotations(obj metav1.Object) metav1.Object {
	obj.SetAnnotations(cleanRemovalKeys(obj.GetAnnotations()))
	obj.SetLabels(cleanRemovalKeys(obj.GetLabels()))
	return obj
}

func cleanRemovalKeys(required map[string]string) map[string]string {
	for k := range required {
		if strings.HasSuffix(k, "-") {
			delete(required, k)
		}
	}
	return required
}

func stringPtr(val string) *string {
	return &val
}

func SetString(modified *bool, existing *string, required string) {
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func SetStringIfSet(modified *bool, existing *string, required string) {
	if len(required) == 0 {
		return
	}
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func setStringPtr(modified *bool, existing **string, required *string) {
	if *existing == nil || (required == nil && *existing != nil) {
		*modified = true
		*existing = required
		return
	}
	SetString(modified, *existing, *required)
}

func SetStringSlice(modified *bool, existing *[]string, required []string) {
	if !reflect.DeepEqual(required, *existing) {
		*existing = required
		*modified = true
	}
}

func SetStringSliceIfSet(modified *bool, existing *[]string, required []string) {
	if required == nil {
		return
	}
	if !reflect.DeepEqual(required, *existing) {
		*existing = required
		*modified = true
	}
}

func BoolPtr(val bool) *bool {
	return &val
}

func SetBool(modified *bool, existing *bool, required bool) {
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func setBoolPtr(modified *bool, existing **bool, required *bool) {
	if *existing == nil || (required == nil && *existing != nil) {
		*modified = true
		*existing = required
		return
	}
	SetBool(modified, *existing, *required)
}

func int64Ptr(val int64) *int64 {
	return &val
}

func SetInt32(modified *bool, existing *int32, required int32) {
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func SetInt32IfSet(modified *bool, existing *int32, required int32) {
	if required == 0 {
		return
	}

	SetInt32(modified, existing, required)
}

func SetInt64(modified *bool, existing *int64, required int64) {
	if required != *existing {
		*existing = required
		*modified = true
	}
}

func setInt64Ptr(modified *bool, existing **int64, required *int64) {
	if *existing == nil || (required == nil && *existing != nil) {
		*modified = true
		*existing = required
		return
	}
	SetInt64(modified, *existing, *required)
}

func MergeMap(modified *bool, existing *map[string]string, required map[string]string) {
	if *existing == nil {
		*existing = map[string]string{}
	}
	for k, v := range required {
		if existingV, ok := (*existing)[k]; !ok || v != existingV {
			*modified = true
			// if "required" map contains a key with "-" as suffix, remove that
			// key from the existing map instead of replacing the value
			if strings.HasSuffix(k, "-") {
				delete(*existing, strings.TrimRight(k, "-"))
			} else {
				(*existing)[k] = v
			}
		}
	}
}

func SetMapStringString(modified *bool, existing *map[string]string, required map[string]string) {
	if *existing == nil {
		*existing = map[string]string{}
	}

	if !reflect.DeepEqual(*existing, required) {
		*existing = required
	}
}

func SetMapStringStringIfSet(modified *bool, existing *map[string]string, required map[string]string) {
	if required == nil {
		return
	}
	if *existing == nil {
		*existing = map[string]string{}
	}

	if !reflect.DeepEqual(*existing, required) {
		*existing = required
	}
}
