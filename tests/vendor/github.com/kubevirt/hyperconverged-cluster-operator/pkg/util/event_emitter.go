package util

import (
	"reflect"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

var (
	evntEmtr EventEmitter = &eventEmitter{}
)

func GetEventEmitter() EventEmitter {
	return evntEmtr
}

type EventEmitter interface {
	Init(pod *corev1.Pod, csv *csvv1alpha1.ClusterServiceVersion, recorder record.EventRecorder)
	EmitEvent(object runtime.Object, eventType, reason, msg string)
}

type eventEmitter struct {
	recorder record.EventRecorder
	pod      *corev1.Pod
	csv      *csvv1alpha1.ClusterServiceVersion
}

func (ee *eventEmitter) Init(pod *corev1.Pod, csv *csvv1alpha1.ClusterServiceVersion, recorder record.EventRecorder) {
	ee.recorder = recorder //mgr.GetEventRecorderFor(HyperConvergedName)
	ee.pod = pod
	ee.csv = csv
}

func (ee eventEmitter) EmitEvent(object runtime.Object, eventType, reason, msg string) {
	if ee.pod != nil {
		ee.recorder.Event(ee.pod, eventType, reason, msg)
	}

	if !IsActuallyNil(object) {
		ee.recorder.Event(object, eventType, reason, msg)
	}

	if ee.csv != nil {
		ee.recorder.Event(ee.csv, eventType, reason, msg)
	}
}

// IsActuallyNil checks if an interface object is actually nil. Just checking for == nil won't work, if the parameter is
// a pointer variable that holds nil.
func IsActuallyNil(object interface{}) bool {
	if object == nil {
		return true
	}

	t := reflect.ValueOf(object)

	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return t.IsNil()
	default:
		return false
	}
}
