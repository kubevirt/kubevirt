package agentpoller

import (
	"reflect"

	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"
)

// MockAgentPollerInterface is a mock of AgentPollerInterface interface.
type MockAgentPollerInterface struct {
	ctrl     *gomock.Controller
	recorder *MockAgentPollerInterfaceMockRecorder
}

// MockAgentPollerInterfaceMockRecorder is the mock recorder for MockAgentPollerInterface.
type MockAgentPollerInterfaceMockRecorder struct {
	mock *MockAgentPollerInterface
}

// NewMockAgentPollerInterface creates a new mock instance.
func NewMockAgentPollerInterface(ctrl *gomock.Controller) *MockAgentPollerInterface {
	mock := &MockAgentPollerInterface{ctrl: ctrl}
	mock.recorder = &MockAgentPollerInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentPollerInterface) EXPECT() *MockAgentPollerInterfaceMockRecorder {
	return m.recorder
}

// Start mocks base method.
func (m *MockAgentPollerInterface) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start.
func (mr *MockAgentPollerInterfaceMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockAgentPollerInterface)(nil).Start))
}

// Stop mocks base method.
func (m *MockAgentPollerInterface) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockAgentPollerInterfaceMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockAgentPollerInterface)(nil).Stop))
}

// UpdateFromEvent mocks base method.
func (m *MockAgentPollerInterface) UpdateFromEvent(domainEvent *libvirt.DomainEventLifecycle, agentEvent *libvirt.DomainEventAgentLifecycle) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateFromEvent", domainEvent, agentEvent)
}

// UpdateFromEvent indicates an expected call of UpdateFromEvent.
func (mr *MockAgentPollerInterfaceMockRecorder) UpdateFromEvent(domainEvent, agentEvent any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFromEvent", reflect.TypeOf((*MockAgentPollerInterface)(nil).UpdateFromEvent), domainEvent, agentEvent)
}
