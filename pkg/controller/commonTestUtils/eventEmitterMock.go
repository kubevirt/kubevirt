package commonTestUtils

import (
	"context"
	"github.com/go-logr/logr"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sync"
)

type MockEvent struct {
	EventType string
	Reason    string
	Msg       string
}

type EventEmitterMock struct {
	storedEvents []MockEvent
	lock         *sync.Mutex
}

func NewEventEmitterMock() *EventEmitterMock {
	return &EventEmitterMock{
		storedEvents: make([]MockEvent, 0),
		lock:         &sync.Mutex{},
	}
}

func (eem *EventEmitterMock) Reset() {
	eem.lock.Lock()
	defer eem.lock.Unlock()

	eem.storedEvents = make([]MockEvent, 0)
}

func (EventEmitterMock) Init(_ context.Context, _ manager.Manager, _ hcoutil.ClusterInfo, _ logr.Logger) {
	/* not implemented; mock only */
}

func (eem *EventEmitterMock) EmitEvent(_ runtime.Object, eventType, reason, msg string) {
	event := MockEvent{
		EventType: eventType,
		Reason:    reason,
		Msg:       msg,
	}

	eem.lock.Lock()
	defer eem.lock.Unlock()

	eem.storedEvents = append(eem.storedEvents, event)
}

func (EventEmitterMock) UpdateClient(_ context.Context, _ client.Reader, _ logr.Logger) {
	/* not implemented; mock only */
}

func (eem EventEmitterMock) CheckEvents(expectedEvents []MockEvent) bool {
	eem.lock.Lock()
	defer eem.lock.Unlock()

	for _, expectedEvent := range expectedEvents {
		if !eventInArray(eem.storedEvents, expectedEvent) {
			return false
		}
	}

	return true
}

func eventInArray(eventList []MockEvent, event MockEvent) bool {
	for _, expectedEvent := range eventList {
		if event == expectedEvent {
			return true
		}
	}
	return false
}
