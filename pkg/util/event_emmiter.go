package util

import (
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

	if object != nil {
		ee.recorder.Event(object, eventType, reason, msg)
	}

	if ee.csv != nil {
		ee.recorder.Event(ee.csv, eventType, reason, msg)
	}
}
