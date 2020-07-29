package util

import (
	"context"
	"github.com/go-logr/logr"
	csvv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	evntEmtr EventEmitter = &eventEmitter{}
)

func GetEventEmitter() EventEmitter {
	return evntEmtr
}

type EventEmitter interface {
	Init(ctx context.Context, mgr manager.Manager, ci ClusterInfo, logger logr.Logger) error
	EmitEvent(object runtime.Object, eventType, reason, msg string)
}

type eventEmitter struct {
	recorder   record.EventRecorder
	clientInfo ClusterInfo
	pod        *corev1.Pod
	csv        *csvv1alpha1.ClusterServiceVersion
}

func (ee *eventEmitter) Init(ctx context.Context, mgr manager.Manager, ci ClusterInfo, logger logr.Logger) error {
	ee.recorder = mgr.GetEventRecorderFor(HyperConvergedName)
	ee.clientInfo = ci

	if !ci.IsRunningLocally() {
		var (
			err    error
			client = mgr.GetAPIReader()
		)

		ee.pod, err = GetPod(ctx, client, logger, ci)
		if err != nil {
			ee.pod = nil
			return err
		}

		ee.csv, err = GetCSVfromPod(ee.pod, client, logger)
		if err != nil {
			ee.csv = nil
		}
	}

	return nil
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
