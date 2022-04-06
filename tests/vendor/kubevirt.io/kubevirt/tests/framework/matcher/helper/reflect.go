package helper

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func IsNil(actual interface{}) bool {
	return actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())
}

func IsSlice(actual interface{}) bool {
	val := reflect.ValueOf(actual)
	return val.Kind() == reflect.Array || val.Kind() == reflect.Slice
}

// ToPointer returns a new pointer to the provided interface{}, if the
// provided value is not already a pointer. if the original value is already a pointer it gets
// returned directly.
func ToPointer(actual interface{}) interface{} {
	if reflect.ValueOf(actual).Kind() != reflect.Ptr {
		p := reflect.New(reflect.TypeOf(actual))
		p.Elem().Set(reflect.ValueOf(actual))
		actual = p.Interface()
	}
	return actual
}

// IterateOverSlice iterates over the provides interface{} until all elements were visited
// or until the visior returns false
func IterateOverSlice(actual interface{}, visitor func(value interface{}) bool) {
	val := reflect.ValueOf(actual)
	for x := 0; x < val.Len(); x++ {
		if !visitor(val.Index(x).Interface()) {
			break
		}
	}
}

// MatchElementsInSlice applies a matcher individually to each element in the slice and returns as
// soon as the matcher fails on an element.
func MatchElementsInSlice(actual interface{}, matcher func(actual interface{}) (success bool, err error)) (bool, error) {
	var success bool
	var err error
	IterateOverSlice(actual, func(value interface{}) bool {
		success, err = matcher(value)
		return success
	})
	return success, err
}

func ToUnstructured(actual interface{}) (*unstructured.Unstructured, error) {
	if IsNil(actual) {
		return nil, fmt.Errorf("object does not exist")
	}
	actual = ToPointer(actual)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(actual)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
