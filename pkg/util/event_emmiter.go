package util

import (
	"context"
	"github.com/go-logr/logr"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	evntEmtr EventEmitter = &eventEmitter{}
)

func GetEventEmitter() EventEmitter {
	return evntEmtr
}

type EventEmitter interface {
	Init(ctx context.Context, mgr manager.Manager, ci ClusterInfo, logger logr.Logger)
	EmitEvent(object runtime.Object, eventType, reason, msg string)
	UpdateClient(ctx context.Context, clnt client.Reader, logger logr.Logger)
}

type eventEmitter struct {
	recorder    record.EventRecorder
	clusterInfo ClusterInfo
	pod         *corev1.Pod
	csv         *csvv1alpha1.ClusterServiceVersion
}

func (ee *eventEmitter) Init(ctx context.Context, mgr manager.Manager, ci ClusterInfo, logger logr.Logger) {
	ee.recorder = mgr.GetEventRecorderFor(HyperConvergedName)
	ee.clusterInfo = ci
	clnt := mgr.GetAPIReader()
	ee.UpdateClient(ctx, clnt, logger)
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

func (ee *eventEmitter) UpdateClient(ctx context.Context, clnt client.Reader, logger logr.Logger) {
	if (ee.pod == nil) && !ee.clusterInfo.IsRunningLocally() {
		var err error

		ee.pod, err = GetPod(ctx, clnt, logger, clusterInfo)
		if err != nil {
			ee.pod = nil
			logger.Error(err, "Can't get self pod")
		}
	}

	if (ee.csv == nil) && clusterInfo.IsOpenshift() {
		var err error
		ee.csv, err = GetCSVfromPod(ee.pod, clnt, logger)
		if err != nil {
			logger.Error(err, "Can't get CSV")
			ee.csv = nil
		}
	}
}
