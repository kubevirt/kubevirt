package eventsclient

import (
	"context"
	"fmt"
	"time"

	grpcstatus "google.golang.org/grpc/status"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/wait"
)

var _ notifierClient = &client{}

type Invoke[T any] struct {
	Event   T
	ErrorCh chan error
}

func NewBufferClient() *client {
	return newBufferClientWithCustomTimeouts(defaultIntervalTimeout, defaultSendTimeout, defaultTotalTimeout)
}

func newBufferClientWithCustomTimeouts(interval, send, total time.Duration) *client {
	return &client{
		domainNotification: make(chan Invoke[*watch.Event]),
		k8sNotification:    make(chan Invoke[*k8sv1.Event]),
		intervalTimeout:    interval,
		sendTimeout:        send,
		totalTimeout:       total,
	}
}

type client struct {
	domainNotification chan Invoke[*watch.Event]
	k8sNotification    chan Invoke[*k8sv1.Event]

	intervalTimeout time.Duration
	sendTimeout     time.Duration
	totalTimeout    time.Duration
}

func (client *client) Close() {
}

func (client *client) HandleDomainEvent(event *watch.Event) error {
	if err := deliver(client, client.domainNotification, event); err != nil {
		if st, ok := grpcstatus.FromError(err); ok {
			return fmt.Errorf("failed to send domain notify event with %s: %w", st.Code(), err)
		}
		return err
	}
	return nil
}

func (client *client) HandleK8SEvent(event *k8sv1.Event) error {
	if err := deliver(client, client.k8sNotification, event); err != nil {
		if st, ok := grpcstatus.FromError(err); ok {
			return fmt.Errorf("failed to notify k8s event with %s: %w", st.Code(), err)
		}
		return err
	}
	return nil
}

// deliver mirrors the semantics of the gRPC notify client.
func deliver[T any](client *client, events chan<- Invoke[T], event T) error {
	return wait.PollImmediately(client.intervalTimeout, client.totalTimeout, func(ctx context.Context) (bool, error) {
		sendCtx, cancel := context.WithTimeout(ctx, client.sendTimeout)
		defer cancel()

		errorCh := make(chan error, 1)
		select {
		case events <- Invoke[T]{Event: event, ErrorCh: errorCh}:
		case <-ctx.Done():
			log.Log.Error("No consumer took the notify event")
			return false, ctx.Err()
		case <-sendCtx.Done():
			log.Log.Reason(sendCtx.Err()).Error("No consumer took the notify event")
			return false, nil
		}

		select {
		case err := <-errorCh:
			if err == nil {
				return true, nil
			}
			if isTransientError(err) {
				log.Log.Reason(err).Error("Failed to deliver notify event, retrying")
				return false, nil
			}
			log.Log.Reason(err).Error("Failed to deliver notify event")
			return false, err
		case <-sendCtx.Done():
			log.Log.Error("Timed out waiting for notify event delivery, retrying")
			return false, nil
		case <-ctx.Done():
			log.Log.Error("Timed out waiting for notify event delivery")
			return false, ctx.Err()
		}
	})
}

func (client *client) DomainEventChan() <-chan Invoke[*watch.Event] {
	return client.domainNotification
}

func (client *client) K8SEventChan() <-chan Invoke[*k8sv1.Event] {
	return client.k8sNotification
}
