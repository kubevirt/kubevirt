package util

import (
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MergeLabels merges src labels into tgt ones.
func MergeLabels(src, tgt *metav1.ObjectMeta) {
	if src.Labels == nil {
		return
	}

	if tgt.Labels == nil {
		tgt.Labels = make(map[string]string, len(src.Labels))
	}

	maps.Copy(tgt.Labels, src.Labels)
}

// CompareLabels reports whether src labels are contained into tgt ones; extra labels on tgt are ignored.
// It returns true if the src labels map is a subset of the tgt one.
func CompareLabels(src, tgt metav1.Object) bool {
	targetLabels := tgt.GetLabels()
	for key, val := range src.GetLabels() {
		tgt_v, ok := targetLabels[key]
		if !ok || tgt_v != val {
			return false
		}
	}
	return true
}
