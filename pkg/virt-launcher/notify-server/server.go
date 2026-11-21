package notifyserver

import (
	"encoding/json"
	"io"

	"golang.org/x/net/context"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
	client "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ v2.NotifyServer = &notifyServer{}

type notifyPipe interface {
	DomainEventChan() <-chan client.Invoke[*watch.Event]
	K8SEventChan() <-chan client.Invoke[*k8sv1.Event]
}

func NewNotifyServer(pipe notifyPipe) *notifyServer {
	return &notifyServer{
		pipe:   pipe,
		logger: *log.Log.With("component", "notifyserver"),
	}
}

type notifyServer struct {
	pipe   notifyPipe
	logger log.FilteredLogger
}

func (n *notifyServer) Ping(context.Context, *v2.EmptyRequest) (*v2.EmptyResponse, error) {
	return &v2.EmptyResponse{}, nil
}

func (n *notifyServer) DomainEvents(_ *v2.EmptyRequest, stream v2.Notify_DomainEventsServer) error {
	// TODO: Only one stream should be active
	n.logger.Info("Starting to stream domain events")
	for {
		select {
		case invoke, ok := <-n.pipe.DomainEventChan():
			if !ok {
				n.logger.Error("Domain events channel is closed")
				return nil
			}
			event := invoke.Event
			domain := event.Object.(*api.Domain)
			data, err := json.Marshal(domain)
			if err != nil {
				log.Log.Reason(err).Error("Failed to marshal domain")
				invoke.ErrorCh <- err
				continue
			}
			response := v2.DomainEvent{
				EventType: string(event.Type),
				Domain:    data,
			}

			if err := stream.Send(&response); err != nil {
				n.logger.Reason(err).Error("Failed to send domain event")
				invoke.ErrorCh <- err
				if err == io.EOF {
					return nil
				}
				return err
			}
			invoke.ErrorCh <- nil
			continue
		case <-stream.Context().Done():
			n.logger.Info("Domain Events stream is done")
			return nil
		}
	}

}

func (n *notifyServer) KubernetesEvents(_ *v2.EmptyRequest, stream v2.Notify_KubernetesEventsServer) error {
	// TODO: Only one stream should be active
	n.logger.Info("Starting to stream kubernetes events")
	for {
		select {
		case invoke, ok := <-n.pipe.K8SEventChan():
			if !ok {
				n.logger.Error("Kubernetes events channel is closed")
				return nil
			}
			event := invoke.Event
			response := v2.KubernetesEvent{
				Type:    event.Type,
				Reason:  event.Reason,
				Message: event.Message,
			}

			if err := stream.Send(&response); err != nil {
				n.logger.Reason(err).Error("Failed to send kubernetes event")
				invoke.ErrorCh <- err
				if err == io.EOF {
					return nil
				}
				return err
			}
			invoke.ErrorCh <- nil
			continue
		case <-stream.Context().Done():
			n.logger.Info("Kubernetes Events stream is done")
			return nil
		}
	}
}
