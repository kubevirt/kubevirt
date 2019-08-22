package kubestate

import (
	"context"
)

type State interface {
	isState()

	Terminal() bool
	// Add() AddedState
	// Update() UpdatedState
	// Delete() DeletedState
}

type ExistsState interface {
	State

	isExistsState()
}

type AddedState interface {
	ExistsState

	isAddedState()
}

type UpdatedState interface {
	ExistsState

	isUpdatedState()
}

type DoesNotExistState interface {
	State

	isDoesNotExistState()
}

type DeletedState interface {
	DoesNotExistState

	isDeletedState()
}

type state struct{}

func (s state) isState() {}

func (s state) Terminal() bool {
	// Not terminal by default
	return false
}

func (s state) Add() AddedState {
	return &addedState{
		ExistsState: &existsState{
			State: s,
		},
	}
}

func (s state) Update() UpdatedState {
	return &updatedState{
		ExistsState: &existsState{
			State: s,
		},
	}
}

func (s state) Delete() DeletedState {
	return &deletedState{
		DoesNotExistState: &doesNotExistState{
			State: s,
		},
	}
}

func NewState() State {
	return &state{}
}

type existsState struct {
	State
}

func (e existsState) isExistsState() {}

type addedState struct {
	ExistsState
}

func (a addedState) isAddedState() {}

type updatedState struct {
	ExistsState
}

func (u updatedState) isUpdatedState() {}

type doesNotExistState struct {
	State
}

func (d doesNotExistState) isDoesNotExistState() {}

type deletedState struct {
	DoesNotExistState
}

func (d deletedState) isDeletedState() {}

type Reconciler interface {
	Reconcile(ctx context.Context, in State) (out State, err error)
}

type ReconcilerFunc func(ctx context.Context, in State) (out State, err error)

func (r ReconcilerFunc) Reconcile(ctx context.Context, in State) (out State, err error) {
	return r(ctx, in)
}

type ReconcilerChain []Reconciler

func (r ReconcilerChain) Reconcile(ctx context.Context, in State) (out State, err error) {
	out = in
	for _, rec := range r {
		if out, err = rec.Reconcile(ctx, out); err != nil || out == nil || out.Terminal() {
			break
		}
	}

	return
}

// ResourceEventType tells an operator what kind of event has occurred on a given resource.
type ResourceEventType string

const (
	// ResourceAdded tells the operator that a given resources has been added.
	ResourceAdded ResourceEventType = "add"
	// ResourceUpdated tells the operator that a given resources has been updated.
	ResourceUpdated ResourceEventType = "update"
	// ResourceDeleted tells the operator that a given resources has been deleted.
	ResourceDeleted ResourceEventType = "delete"
)

type ResourceEvent interface {
	Type() ResourceEventType
	Resource() interface{}
}

type resourceEvent struct {
	eventType ResourceEventType
	resource  interface{}
}

func (r resourceEvent) Type() ResourceEventType {
	return r.eventType
}

func (r resourceEvent) Resource() interface{} {
	return r.resource
}

func NewResourceEvent(eventType ResourceEventType, resource interface{}) ResourceEvent {
	return resourceEvent{
		eventType: eventType,
		resource:  resource,
	}
}

type Notifier interface {
	Notify(event ResourceEvent)
}

type NotifyFunc func(event ResourceEvent)

// SyncFunc syncs resource events.
type SyncFunc func(ctx context.Context, event ResourceEvent) error

// Sync lets a sync func implement Syncer.
func (s SyncFunc) Sync(ctx context.Context, event ResourceEvent) error {
	return s(ctx, event)
}

// Syncer describes something that syncs resource events.
type Syncer interface {
	Sync(ctx context.Context, event ResourceEvent) error
}
