package util

import (
	"context"

	"github.com/go-logr/logr"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	evntEmtr EventEmitter = &eventEmitter{}
)

func GetEventEmitter() EventEmitter {
	return evntEmtr
}

type EventEmitter interface {
	Init(ctx context.Context, cl client.Client, recorder record.EventRecorder, logger logr.Logger)
	EmitEvent(object runtime.Object, eventType, reason, msg string)
}

type eventEmitter struct {
	recorder record.EventRecorder
	pod      *corev1.Pod
	csv      *csvv1alpha1.ClusterServiceVersion
}

func (ee *eventEmitter) Init(ctx context.Context, cl client.Client, recorder record.EventRecorder, logger logr.Logger) {
	ee.recorder = recorder //mgr.GetEventRecorderFor(HyperConvergedName)
	ee.getResource(ctx, cl, logger)
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

func (ee *eventEmitter) getResource(ctx context.Context, cl client.Reader, logger logr.Logger) {
	if !GetClusterInfo().IsRunningLocally() {
		var err error

		ee.pod, err = GetPod(ctx, cl, logger)
		if err != nil {
			ee.pod = nil
			logger.Error(err, "Can't get self pod")
		}
	}

	if GetClusterInfo().IsOpenshift() {
		var err error
		ee.csv, err = GetCSVfromPod(ee.pod, cl, logger)
		if err != nil {
			logger.Error(err, "Can't get CSV")
			ee.csv = nil
		}
	}
}
