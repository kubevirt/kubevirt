package predicate

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var _ predicate.Predicate = GenerationOrAnnotationChangedPredicate{}

var log = logf.Log.WithName("predicate")

// GenerationOrAnnotationChangedPredicate implements a predicate function
// that skips update events in which the generation hasn't been incremented,
// nor the annotations have changed.
//
// The implementation is based on sigs.k8s.io/controller-runtime/pkg/predicate.GenerationChangedPredicate.
type GenerationOrAnnotationChangedPredicate struct {
	predicate.Funcs
}

// Update implements default UpdateEvent filter for validating generation/annotation change
func (GenerationOrAnnotationChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.MetaOld == nil {
		log.Error(nil, "Update event has no old metadata", "event", e)
		return false
	}
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	if e.MetaNew == nil {
		log.Error(nil, "Update event has no new metadata", "event", e)
		return false
	}

	return e.MetaNew.GetGeneration() != e.MetaOld.GetGeneration() ||
		!reflect.DeepEqual(e.MetaNew.GetAnnotations(), e.MetaOld.GetAnnotations())
}
