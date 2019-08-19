package labeler

import (
	"k8s.io/apimachinery/pkg/labels"
)

// Labeler can provide label sets that describe an object
type Labeler interface {
	// LabelSetsFor returns label sets that describe the given object
	LabelSetsFor(obj interface{}) ([]labels.Set, error)
}

// Func is a function type that implements the Labeler interface
type Func func(obj interface{}) ([]labels.Set, error)

// LabelSetsFor calls LabelSetsFor on itself to satisfy the Labeler interface
func (l Func) LabelSetsFor(obj interface{}) ([]labels.Set, error) {
	return l(obj)
}
