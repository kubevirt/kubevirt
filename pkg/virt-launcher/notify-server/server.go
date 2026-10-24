package notifyserver

import (
	"encoding/json"
	"io"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ v2.NotifyServer = &notifyServer{}

func NewNotifyServer(watchNotification <-chan *watch.Event, eventNotificatio <-chan *k8sv1.Event) *notifyServer {
	return &notifyServer{
		watchNotification: watchNotification,
		eventNotificatio:  eventNotificatio,
	}
}

type notifyServer struct {
	watchNotification <-chan *watch.Event
	eventNotificatio  <-chan *k8sv1.Event
}

func (n *notifyServer) DomainEvents(_ *v2.EmptyRequest, stream v2.Notify_DomainEventsServer) error {
	for {
		select {
		case event := <-n.watchNotification:
			domain := event.Object.(*api.Domain)
			data, err := json.Marshal(domain)
			if err != nil {
				log.Log.Reason(err).Error("Failed to marshal domain")
				continue
			}
			response := v2.DomainEvent{
				EventType: string(event.Type),
				Domain:    data,
			}

			if err := stream.Send(&response); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			continue
		case <-stream.Context().Done():
			log.Log.Info("Domain Events stream is done")
			return nil
		}
	}

}

func (n *notifyServer) KubernetesEvents(_ *v2.EmptyRequest, stream v2.Notify_KubernetesEventsServer) error {
	for {
		select {
		case event := <-n.eventNotificatio:
			response := v2.KubernetesEvent{
				Type:    event.Type,
				Reason:  event.Reason,
				Message: event.Message,
			}

			if err := stream.Send(&response); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			continue
		case <-stream.Context().Done():
			log.Log.Info("Kubernetes Events stream is done")
			return nil
		}
	}
}
